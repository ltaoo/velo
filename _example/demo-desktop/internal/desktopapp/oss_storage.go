package desktopapp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ltaoo/velo"
)

const cloudStorageSettingsKey = "demo-desktop:settings:cloud-storage:v1"
const defaultLocalStorageRelativeRoot = "storage"
const localStorageRootModeAbsolute = "absolute"
const localStorageRootModeVault = "vault"

type LocalOSSSettings struct {
	Root     string `json:"root,omitempty"`
	RootMode string `json:"rootMode,omitempty"`
}

type OSSConfig struct {
	AccessKeyID     string            `json:"accessKeyId"`
	Bucket          string            `json:"bucket"`
	Enabled         bool              `json:"enabled"`
	Endpoint        string            `json:"endpoint"`
	ForcePathStyle  bool              `json:"forcePathStyle"`
	ID              string            `json:"id"`
	Local           *LocalOSSSettings `json:"local,omitempty"`
	Name            string            `json:"name"`
	PathPrefix      string            `json:"pathPrefix"`
	Provider        string            `json:"provider"`
	PublicBaseURL   string            `json:"publicBaseUrl"`
	Region          string            `json:"region"`
	SecretAccessKey string            `json:"secretAccessKey"`
	SessionToken    string            `json:"sessionToken"`
	UseSSL          bool              `json:"useSSL"`
}

type CloudStorageSettings struct {
	ActiveStorageID     string      `json:"activeStorageId"`
	DefaultsInitialized bool        `json:"defaultsInitialized,omitempty"`
	Storages            []OSSConfig `json:"storages"`
}

type OSSUploadRequest struct {
	Config        OSSConfig `json:"config"`
	ContentBase64 string    `json:"content_base64"`
	Name          string    `json:"name"`
	StorageID     string    `json:"storageId"`
	Type          string    `json:"type"`
}

type OSSFileListRequest struct {
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFilePreviewRequest struct {
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileMkdirRequest struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileDeleteRequest struct {
	IsDir     bool   `json:"isDir"`
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileUploadRequest struct {
	ContentBase64 string `json:"content_base64"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	StorageID     string `json:"storageId"`
	Type          string `json:"type"`
}

type OSSFileView struct {
	ID      string `json:"id"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Ref     string `json:"ref"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	URL     string `json:"url"`
}

func imageFileExtensions() []string {
	return []string{"avif", "bmp", "gif", "jpg", "jpeg", "png", "svg", "webp"}
}

func uploadOSSObject(parent context.Context, req OSSUploadRequest) (velo.H, error) {
	cfg := req.Config
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return uploadLocalOSSObject(parent, req)
	}

	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	key := objectKey(cfg.PathPrefix, req.Name)
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
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
		"url":       publicOSSObjectURL(cfg, endpoint, key),
	}, nil
}

func storedOSSConfig(raw json.RawMessage, storageID string, storePath string) (OSSConfig, error) {
	settings, err := loadStoredCloudStorageSettings(raw)
	if err != nil {
		return OSSConfig{}, err
	}
	settings, _, err = prepareCloudStorageSettings(settings, storePath, len(settings.Storages) == 0)
	if err != nil {
		return OSSConfig{}, err
	}
	cfg, err := activeOSSConfig(settings, storageID)
	if err != nil {
		return OSSConfig{}, err
	}
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, storageID, "default"))
	return cfg, nil
}

func listOSSFiles(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return listLocalOSSFiles(parent, cfg, objectPath)
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	cleanPath := cleanOSSObjectPath(objectPath)
	prefix := ossFolderPrefix(cleanPath)
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(cfg.Bucket),
		Delimiter: aws.String("/"),
		MaxKeys:   1000,
		Prefix:    aws.String(prefix),
	}
	seen := map[string]bool{}
	items := make([]OSSFileView, 0)
	for {
		out, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, commonPrefix := range out.CommonPrefixes {
			key := stringValue(commonPrefix.Prefix)
			view := ossFileView(cfg, endpoint, cleanPath, key, true, 0, nil, "")
			if view.Path == "" || seen[view.Path] {
				continue
			}
			seen[view.Path] = true
			items = append(items, view)
		}

		for _, object := range out.Contents {
			key := stringValue(object.Key)
			if key == "" || key == prefix {
				continue
			}
			isDir := strings.HasSuffix(key, "/")
			view := ossFileView(cfg, endpoint, cleanPath, key, isDir, object.Size, object.LastModified, "")
			if view.Path == "" || seen[view.Path] {
				continue
			}
			seen[view.Path] = true
			items = append(items, view)
		}

		if !out.IsTruncated || out.NextContinuationToken == nil {
			break
		}
		input.ContinuationToken = out.NextContinuationToken
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
		"prefix":    prefix,
		"storageId": cfg.ID,
	}, nil
}

func previewOSSFile(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return previewLocalOSSFile(parent, cfg, objectPath)
	}
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}

	client, _, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	if head.ContentLength > 8*1024*1024 {
		return nil, fmt.Errorf("file is too large to preview, max size is 8 MB")
	}

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	content, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	name := pathpkg.Base(key)
	ext := strings.ToLower(filepath.Ext(name))
	contentType := firstNonEmpty(stringValue(out.ContentType), stringValue(head.ContentType), mime.TypeByExtension(ext), "application/octet-stream")
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
		"size":     head.ContentLength,
		"type":     "unknown",
	}, nil
}

func makeOSSFolder(parent context.Context, cfg OSSConfig, req OSSFileMkdirRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return makeLocalOSSFolder(parent, cfg, req)
	}

	folderPath := cleanOSSObjectPath(req.Path)
	if strings.TrimSpace(req.Name) != "" {
		folderPath = objectPathJoin(folderPath, req.Name)
	}
	if folderPath == "" {
		return nil, fmt.Errorf("folder path is required")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	key := ossFolderPrefix(folderPath)
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(nil),
		ContentType: aws.String("application/x-directory"),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"file":      ossFileView(cfg, endpoint, pathpkg.Dir(folderPath), key, true, 0, nil, "application/x-directory"),
		"path":      folderPath,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func deleteOSSFile(parent context.Context, cfg OSSConfig, req OSSFileDeleteRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return deleteLocalOSSFile(parent, cfg, req)
	}

	key := cleanOSSObjectPath(req.Path)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}

	client, _, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	deleted := 0
	if req.IsDir {
		prefix := ossFolderPrefix(key)
		input := &s3.ListObjectsV2Input{
			Bucket:  aws.String(cfg.Bucket),
			MaxKeys: 1000,
			Prefix:  aws.String(prefix),
		}
		for {
			out, err := client.ListObjectsV2(ctx, input)
			if err != nil {
				return nil, err
			}
			for _, object := range out.Contents {
				objectKey := stringValue(object.Key)
				if objectKey == "" {
					continue
				}
				if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: aws.String(cfg.Bucket),
					Key:    aws.String(objectKey),
				}); err != nil {
					return nil, err
				}
				deleted++
			}
			if !out.IsTruncated || out.NextContinuationToken == nil {
				break
			}
			input.ContinuationToken = out.NextContinuationToken
		}
	} else {
		if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
		}); err != nil {
			return nil, err
		}
		deleted = 1
	}

	return velo.H{
		"deleted":   deleted,
		"path":      key,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func uploadOSSManagedFile(parent context.Context, cfg OSSConfig, req OSSFileUploadRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return uploadLocalOSSManagedFile(parent, cfg, req)
	}
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

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
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

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"bucket":    cfg.Bucket,
		"file":      ossFileView(cfg, endpoint, cleanOSSObjectPath(req.Path), key, false, int64(len(data)), nil, contentType),
		"key":       key,
		"name":      sanitizeObjectName(req.Name),
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"success":   true,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, endpoint, key),
	}, nil
}
