package desktopapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ltaoo/velo"
)

func uploadLocalOSSObject(parent context.Context, req OSSUploadRequest) (velo.H, error) {
	cfg := req.Config
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSConfig(cfg); err != nil {
		return nil, err
	}
	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	key := objectKey(cfg.PathPrefix, req.Name)
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := writeLocalOSSObject(parent, cfg, key, data); err != nil {
		return nil, err
	}
	return velo.H{
		"bucket":    cfg.Bucket,
		"key":       key,
		"name":      req.Name,
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, "", key),
	}, nil
}

func listLocalOSSFiles(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cleanPath := cleanOSSObjectPath(objectPath)
	target, err := localOSSObjectDiskPath(cfg, cleanPath)
	if err != nil {
		return nil, err
	}
	if _, err := ensureLocalOSSBucket(cfg); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			return velo.H{
				"bucket":    cfg.Bucket,
				"list":      []OSSFileView{},
				"path":      cleanPath,
				"prefix":    ossFolderPrefix(cleanPath),
				"storageId": cfg.ID,
			}, nil
		}
		return nil, err
	}
	items := make([]OSSFileView, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-parent.Done():
			return nil, parent.Err()
		default:
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		key := objectPathJoin(cleanPath, entry.Name())
		modTime := info.ModTime()
		size := info.Size()
		if entry.IsDir() {
			size = 0
		}
		items = append(items, ossFileView(cfg, "", cleanPath, key, entry.IsDir(), size, &modTime, ""))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return velo.H{
		"bucket":    cfg.Bucket,
		"list":      items,
		"path":      cleanPath,
		"prefix":    ossFolderPrefix(cleanPath),
		"storageId": cfg.ID,
	}, nil
}

func previewLocalOSSFile(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("folder cannot be previewed")
	}
	if info.Size() > 8*1024*1024 {
		return nil, fmt.Errorf("file is too large to preview, max size is 8 MB")
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	name := pathpkg.Base(key)
	ext := strings.ToLower(filepath.Ext(name))
	contentType := firstNonEmpty(mime.TypeByExtension(ext), http.DetectContentType(content), "application/octet-stream")
	if isTextPreview(ext, contentType) {
		return velo.H{
			"content":  string(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "text",
		}, nil
	}
	if strings.HasPrefix(contentType, "image/") {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "image",
		}, nil
	}
	if contentType == "application/pdf" {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "pdf",
		}, nil
	}
	return velo.H{
		"mimeType": contentType,
		"name":     name,
		"path":     key,
		"size":     info.Size(),
		"type":     "unknown",
	}, nil
}

func makeLocalOSSFolder(parent context.Context, cfg OSSConfig, req OSSFileMkdirRequest) (velo.H, error) {
	folderPath := cleanOSSObjectPath(req.Path)
	if strings.TrimSpace(req.Name) != "" {
		folderPath = objectPathJoin(folderPath, req.Name)
	}
	if folderPath == "" {
		return nil, fmt.Errorf("folder path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, folderPath)
	if err != nil {
		return nil, err
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, err
	}
	return velo.H{
		"file":      ossFileView(cfg, "", pathpkg.Dir(folderPath), folderPath, true, 0, nil, "application/x-directory"),
		"path":      folderPath,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func deleteLocalOSSFile(parent context.Context, cfg OSSConfig, req OSSFileDeleteRequest) (velo.H, error) {
	key := cleanOSSObjectPath(req.Path)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return nil, err
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	if err := os.RemoveAll(target); err != nil {
		return nil, err
	}
	return velo.H{
		"deleted":   1,
		"path":      key,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func uploadLocalOSSManagedFile(parent context.Context, cfg OSSConfig, req OSSFileUploadRequest) (velo.H, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("file name is required")
	}
	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	key := objectPathJoin(req.Path, req.Name)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := writeLocalOSSObject(parent, cfg, key, data); err != nil {
		return nil, err
	}
	return velo.H{
		"bucket":    cfg.Bucket,
		"file":      ossFileView(cfg, "", cleanOSSObjectPath(req.Path), key, false, int64(len(data)), nil, contentType),
		"key":       key,
		"name":      sanitizeObjectName(req.Name),
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"success":   true,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, "", key),
	}, nil
}

func writeLocalOSSObject(parent context.Context, cfg OSSConfig, key string, data []byte) error {
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return err
	}
	select {
	case <-parent.Done():
		return parent.Err()
	default:
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0644)
}

func serveLocalOSSAsset(w http.ResponseWriter, cfg OSSConfig, objectPath string) error {
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("folder cannot be served")
	}
	file, err := os.Open(target)
	if err != nil {
		return err
	}
	defer file.Close()
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(target)))
	if contentType == "" {
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, err = io.Copy(w, file)
	return err
}

func writePlainError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}

func isLocalOSSConfig(cfg OSSConfig) bool {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	return provider == "local" || provider == "local-oss"
}

func ensureLocalOSSBucket(cfg OSSConfig) (string, error) {
	root, err := localOSSBucketRoot(cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	return root, nil
}

func localOSSBucketRoot(cfg OSSConfig) (string, error) {
	if err := validateOSSAccessConfig(cfg); err != nil {
		return "", err
	}
	root := strings.TrimSpace(cfg.Endpoint)
	if root == "" {
		root = localOSSAbsoluteRoot(cfg)
	}
	root = expandLocalPath(root)
	if root == "" {
		return "", fmt.Errorf("local root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if err := validateLocalOSSBucket(bucket); err != nil {
		return "", err
	}
	return filepath.Join(absRoot, bucket), nil
}

func localOSSAbsoluteRoot(cfg OSSConfig) string {
	local := normalizeLocalOSSSettings(cfg.Local)
	if local == nil || local.RootMode != localStorageRootModeAbsolute {
		return ""
	}
	return strings.TrimSpace(local.Root)
}

func localOSSObjectDiskPath(cfg OSSConfig, objectPath string) (string, error) {
	bucketRoot, err := ensureLocalOSSBucket(cfg)
	if err != nil {
		return "", err
	}
	cleanKey := cleanOSSObjectPath(objectPath)
	target := bucketRoot
	if cleanKey != "" {
		target = filepath.Join(bucketRoot, filepath.FromSlash(cleanKey))
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != bucketRoot && !strings.HasPrefix(absTarget, bucketRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("object path escapes bucket root: %s", objectPath)
	}
	return absTarget, nil
}

func expandLocalPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(value, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	return value
}

func validateLocalOSSBucket(bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if bucket == "." || bucket == ".." {
		return fmt.Errorf("invalid bucket: %s", bucket)
	}
	if strings.ContainsAny(bucket, `/\`) {
		return fmt.Errorf("bucket must not contain path separators: %s", bucket)
	}
	return nil
}
