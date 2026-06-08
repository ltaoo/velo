package desktopapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/url"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func newOSSClient(cfg OSSConfig) (*s3.Client, string, error) {
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, "", err
	}

	endpoint := normalizeOSSEndpoint(cfg.Endpoint, cfg.UseSSL)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	awsCfg := aws.Config{
		Region: region,
		Credentials: aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
				SessionToken:    cfg.SessionToken,
				Source:          "oss-config",
			}, nil
		})),
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
		o.UsePathStyle = cfg.ForcePathStyle
	})
	return client, endpoint, nil
}

func decodeUploadContent(contentBase64 string) ([]byte, error) {
	value := strings.TrimSpace(contentBase64)
	if value == "" {
		return nil, fmt.Errorf("content_base64 is required")
	}
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		value = value[comma+1:]
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode content_base64: %w", err)
	}
	return data, nil
}

func cleanOSSObjectPath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	value = strings.Trim(value, "/")
	if value == "" || value == "." {
		return ""
	}

	parts := strings.Split(value, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(clean) > 0 {
				clean = clean[:len(clean)-1]
			}
			continue
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/")
}

func ossFolderPrefix(objectPath string) string {
	cleanPath := cleanOSSObjectPath(objectPath)
	if cleanPath == "" {
		return ""
	}
	return cleanPath + "/"
}

func objectPathJoin(parent string, name string) string {
	cleanParent := cleanOSSObjectPath(parent)
	cleanName := sanitizeObjectName(name)
	if cleanName == "" {
		return cleanParent
	}
	if cleanParent == "" {
		return cleanName
	}
	return pathpkg.Join(cleanParent, cleanName)
}

func ossFileView(cfg OSSConfig, endpoint string, parent string, key string, isDir bool, size int64, modTime *time.Time, contentType string) OSSFileView {
	cleanKey := cleanOSSObjectPath(key)
	name := ossFileName(parent, cleanKey)
	if name == "" {
		name = pathpkg.Base(cleanKey)
	}
	if contentType == "" && !isDir {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	}
	if contentType == "" {
		if isDir {
			contentType = "folder"
		} else {
			contentType = "application/octet-stream"
		}
	}

	ref := ""
	publicURL := ""
	if !isDir {
		ref = assetRef(cfg.ID, cleanKey)
		publicURL = publicOSSObjectURL(cfg, endpoint, cleanKey)
	}

	modTimeText := ""
	if modTime != nil && !modTime.IsZero() {
		modTimeText = modTime.Format(time.RFC3339)
	}
	return OSSFileView{
		ID:      cleanKey,
		IsDir:   isDir,
		ModTime: modTimeText,
		Name:    name,
		Path:    cleanKey,
		Ref:     ref,
		Size:    size,
		Type:    contentType,
		URL:     publicURL,
	}
}

func ossFileName(parent string, key string) string {
	cleanKey := cleanOSSObjectPath(key)
	cleanParent := cleanOSSObjectPath(parent)
	rel := cleanKey
	if cleanParent != "" && strings.HasPrefix(rel, cleanParent+"/") {
		rel = strings.TrimPrefix(rel, cleanParent+"/")
	}
	if index := strings.Index(rel, "/"); index >= 0 {
		rel = rel[:index]
	}
	return rel
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isTextPreview(ext string, contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(contentType, "text/") ||
		strings.Contains(contentType, "json") ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "xml") ||
		strings.Contains(contentType, "yaml") {
		return true
	}
	switch strings.ToLower(ext) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".css", ".scss", ".html", ".htm", ".json", ".md", ".markdown", ".txt", ".csv", ".xml", ".yaml", ".yml", ".toml", ".ini", ".log", ".sql", ".sh", ".zsh", ".bash":
		return true
	default:
		return false
	}
}

func normalizeOSSEndpoint(endpoint string, useSSL bool) string {
	value := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if useSSL {
		return "https://" + value
	}
	return "http://" + value
}

func assetRef(storageID string, key string) string {
	id := sanitizeStorageID(storageID)
	if id == "" {
		id = "default"
	}
	return "@assets/" + id + "/" + strings.TrimLeft(key, "/")
}

func objectKey(prefix string, name string) string {
	cleanPrefix := strings.Trim(pathpkg.Clean("/"+strings.TrimSpace(prefix)), "/")
	cleanName := sanitizeObjectName(name)
	if cleanName == "" {
		cleanName = "upload.bin"
	}
	fileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), cleanName)
	if cleanPrefix == "" || cleanPrefix == "." {
		return fileName
	}
	return pathpkg.Join(cleanPrefix, fileName)
}

func sanitizeObjectName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	base = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\:?*<>|"`, r) {
			return '-'
		}
		return r
	}, base)
	return strings.Trim(base, ". ")
}

func publicOSSObjectURL(cfg OSSConfig, endpoint string, key string) string {
	escapedKey := escapedObjectKey(key)
	if isLocalOSSConfig(cfg) {
		return localOSSAssetURL(cfg.ID, key)
	}
	if base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"); base != "" {
		return base + "/" + escapedKey
	}
	if cfg.ForcePathStyle {
		return strings.TrimRight(endpoint, "/") + "/" + url.PathEscape(strings.Trim(cfg.Bucket, "/")) + "/" + escapedKey
	}
	parsed, err := url.Parse(endpoint)
	if err == nil && parsed.Host != "" {
		parsed.Host = strings.Trim(cfg.Bucket, ".") + "." + parsed.Host
		parsed.Path = "/" + escapedKey
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String()
	}
	return strings.TrimRight(endpoint, "/") + "/" + escapedKey
}

func localOSSAssetURL(storageID string, key string) string {
	id := sanitizeStorageID(storageID)
	if id == "" {
		id = "default"
	}
	cleanKey := cleanOSSObjectPath(key)
	if cleanKey == "" {
		return ""
	}
	return "/api/oss/assets?storageId=" + url.QueryEscape(id) + "&path=" + url.QueryEscape(cleanKey)
}

func escapedObjectKey(key string) string {
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
