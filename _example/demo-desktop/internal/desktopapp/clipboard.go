package desktopapp

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ltaoo/clipboard-go"
	"github.com/ltaoo/velo"
	"github.com/rs/zerolog"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

type ClipboardSnapshot struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	RawType       string `json:"rawType"`
	Content       string `json:"content"`
	ContentBase64 string `json:"contentBase64,omitempty"`
	DataURL       string `json:"dataURL,omitempty"`
	MimeType      string `json:"mimeType,omitempty"`
	Name          string `json:"name,omitempty"`
	Size          int    `json:"size,omitempty"`
	CapturedAt    string `json:"capturedAt"`
	ChangedAt     string `json:"changedAt,omitempty"`
}

type ClipboardImageWriteRequest struct {
	ContentBase64 string `json:"contentBase64"`
	MimeType      string `json:"mimeType"`
	Source        string `json:"source"`
	URL           string `json:"url"`
}

const clipboardImageMaxBytes = 32 * 1024 * 1024

var clipboardReadMu sync.Mutex

var observedClipboard = struct {
	sync.RWMutex
	id        string
	changedAt time.Time
}{}

func registerClipboardRoutes(b *velo.Box, logger *zerolog.Logger) {
	b.Get("/api/clipboard/latest", func(c *velo.BoxContext) interface{} {
		item, ok := readCurrentClipboardSnapshot(logger)
		if !ok {
			return c.Ok(velo.H{"found": false})
		}

		maxAge := clipboardMaxAge(c.Query("maxAgeSeconds"))
		changedAt, ageMs, fresh := clipboardFreshness(item.ID, maxAge)
		if !changedAt.IsZero() {
			item.ChangedAt = changedAt.Format(time.RFC3339Nano)
		}

		return c.Ok(velo.H{
			"ageMs":     ageMs,
			"changedAt": item.ChangedAt,
			"found":     true,
			"fresh":     fresh,
			"item":      item,
		})
	})

	b.Post("/api/clipboard/image/write", func(c *velo.BoxContext) interface{} {
		var req ClipboardImageWriteRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		pngData, err := clipboardPNGFromImageRequest(c.Context(), b.Store.Get(cloudStorageSettingsKey), b.Store.Path(), req)
		if err != nil {
			return c.Error(err.Error())
		}
		if err := clipboard.Init(); err != nil {
			return c.Error(err.Error())
		}
		if err := clipboard.WriteImage(pngData); err != nil {
			return c.Error(err.Error())
		}

		encoded := base64.StdEncoding.EncodeToString(pngData)
		rememberClipboardChange(clipboardSnapshotID("image", encoded), time.Now())
		return c.Ok(velo.H{
			"mimeType": "image/png",
			"size":     len(pngData),
			"success":  true,
		})
	})
}

func initClipboardReader(logger *zerolog.Logger) {
	if err := clipboard.Init(); err != nil {
		logger.Warn().Err(err).Msg("clipboard unavailable")
		return
	}
	go watchClipboardChanges(logger)
}

func watchClipboardChanges(logger *zerolog.Logger) {
	ch := clipboard.Watch(context.Background())
	for data := range ch {
		if data.Error != nil {
			logger.Debug().Err(data.Error).Str("type", data.Type).Msg("clipboard read failed")
			continue
		}
		changedAt := time.Now()
		item, ok := clipboardSnapshotFromContent(data, changedAt)
		if !ok {
			continue
		}
		rememberClipboardChange(item.ID, changedAt)
	}
}

func readCurrentClipboardSnapshot(logger *zerolog.Logger) (ClipboardSnapshot, bool) {
	clipboardReadMu.Lock()
	defer clipboardReadMu.Unlock()

	if err := clipboard.Init(); err != nil {
		logger.Debug().Err(err).Msg("clipboard unavailable")
		return ClipboardSnapshot{}, false
	}

	capturedAt := time.Now().Format(time.RFC3339Nano)
	for _, rawType := range clipboard.GetContentTypes(clipboard.ContentTypeParams{IsEnabled: false}) {
		switch rawType {
		case "public.html":
			if item, ok := readClipboardText(rawType, capturedAt, true); ok {
				return item, true
			}
		case "public.utf8-plain-text":
			if item, ok := readClipboardText(rawType, capturedAt, false); ok {
				return item, true
			}
		case "public.png":
			if item, ok := readClipboardImage(rawType, capturedAt); ok {
				return item, true
			}
		}
	}

	return ClipboardSnapshot{}, false
}

func readClipboardText(rawType string, capturedAt string, allowHTMLFallback bool) (ClipboardSnapshot, bool) {
	text, _ := clipboard.ReadText()
	if strings.TrimSpace(text) == "" && allowHTMLFallback {
		text, _ = clipboard.ReadHTML()
	}
	if strings.TrimSpace(text) == "" {
		return ClipboardSnapshot{}, false
	}
	return textClipboardSnapshot(rawType, text, capturedAt), true
}

func readClipboardImage(rawType string, capturedAt string) (ClipboardSnapshot, bool) {
	imageData, err := clipboard.ReadImage()
	if err != nil || len(imageData) == 0 {
		return ClipboardSnapshot{}, false
	}
	return imageClipboardSnapshot(rawType, imageData, capturedAt), true
}

func clipboardSnapshotFromContent(data clipboard.ClipboardContent, capturedAt time.Time) (ClipboardSnapshot, bool) {
	rawType := strings.TrimSpace(data.Type)
	capturedAtText := capturedAt.Format(time.RFC3339Nano)

	switch rawType {
	case "public.utf8-plain-text":
		text, ok := data.Data.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return ClipboardSnapshot{}, false
		}
		return textClipboardSnapshot(rawType, text, capturedAtText), true
	case "public.html":
		text, _ := data.BackupData.(string)
		if strings.TrimSpace(text) == "" {
			text, _ = clipboard.ReadText()
		}
		if strings.TrimSpace(text) == "" {
			text, _ = data.Data.(string)
		}
		if strings.TrimSpace(text) == "" {
			return ClipboardSnapshot{}, false
		}
		return textClipboardSnapshot(rawType, text, capturedAtText), true
	case "public.png":
		imageData, ok := data.Data.([]byte)
		if !ok || len(imageData) == 0 {
			return ClipboardSnapshot{}, false
		}
		return imageClipboardSnapshot(rawType, imageData, capturedAtText), true
	default:
		return ClipboardSnapshot{}, false
	}
}

func rememberClipboardChange(id string, changedAt time.Time) {
	if strings.TrimSpace(id) == "" || changedAt.IsZero() {
		return
	}
	observedClipboard.Lock()
	defer observedClipboard.Unlock()
	observedClipboard.id = id
	observedClipboard.changedAt = changedAt
}

func clipboardFreshness(id string, maxAge time.Duration) (time.Time, int64, bool) {
	observedClipboard.RLock()
	changedAt := observedClipboard.changedAt
	if observedClipboard.id != id {
		changedAt = time.Time{}
	}
	observedClipboard.RUnlock()

	if changedAt.IsZero() {
		return time.Time{}, -1, maxAge <= 0
	}

	age := time.Since(changedAt)
	fresh := maxAge <= 0 || age <= maxAge
	return changedAt, age.Milliseconds(), fresh
}

func clipboardMaxAge(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func textClipboardSnapshot(rawType string, text string, capturedAt string) ClipboardSnapshot {
	content := strings.TrimSpace(text)
	itemType := "text"
	if isClipboardURL(content) {
		itemType = "link"
	}
	return ClipboardSnapshot{
		ID:         clipboardSnapshotID(itemType, content),
		Type:       itemType,
		RawType:    rawType,
		Content:    content,
		CapturedAt: capturedAt,
	}
}

func imageClipboardSnapshot(rawType string, imageData []byte, capturedAt string) ClipboardSnapshot {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	name := "clipboard-" + time.Now().Format("20060102-150405") + ".png"
	content := fmt.Sprintf("PNG image, %s", humanBytes(len(imageData)))
	return ClipboardSnapshot{
		ID:            clipboardSnapshotID("image", encoded),
		Type:          "image",
		RawType:       rawType,
		Content:       content,
		ContentBase64: encoded,
		DataURL:       "data:image/png;base64," + encoded,
		MimeType:      "image/png",
		Name:          name,
		Size:          len(imageData),
		CapturedAt:    capturedAt,
	}
}

func clipboardPNGFromImageRequest(parent context.Context, rawSettings json.RawMessage, storePath string, req ClipboardImageWriteRequest) ([]byte, error) {
	data, mimeType, err := clipboardImageDataFromRequest(parent, rawSettings, storePath, req)
	if err != nil {
		return nil, err
	}
	return clipboardImageToPNG(data, mimeType)
}

func clipboardImageDataFromRequest(parent context.Context, rawSettings json.RawMessage, storePath string, req ClipboardImageWriteRequest) ([]byte, string, error) {
	if strings.TrimSpace(req.ContentBase64) != "" {
		data, err := decodeUploadContent(req.ContentBase64)
		if err != nil {
			return nil, "", err
		}
		return data, firstNonEmpty(req.MimeType, http.DetectContentType(data)), nil
	}

	for _, value := range []string{req.Source, req.URL} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if data, mimeType, ok, err := decodeClipboardImageDataURL(value); ok || err != nil {
			return data, mimeType, err
		}
		if ref, ok := parseMemoAssetReference(value); ok {
			return clipboardImageDataFromAsset(parent, rawSettings, storePath, ref.StorageID, ref.Key)
		}
		if storageID, key, ok, err := parseEditorOSSAssetURL(value); ok || err != nil {
			if err != nil {
				return nil, "", err
			}
			return clipboardImageDataFromAsset(parent, rawSettings, storePath, storageID, key)
		}
		if data, mimeType, ok, err := clipboardImageDataFromHTTP(parent, value); ok || err != nil {
			return data, mimeType, err
		}
	}

	return nil, "", fmt.Errorf("image source is required")
}

func clipboardImageDataFromAsset(parent context.Context, rawSettings json.RawMessage, storePath string, storageID string, key string) ([]byte, string, error) {
	cfg, err := storedOSSConfig(rawSettings, storageID, storePath)
	if err != nil {
		return nil, "", err
	}
	result, err := previewOSSFile(parent, cfg, key)
	if err != nil {
		return nil, "", err
	}
	if fmt.Sprintf("%v", result["type"]) != "image" {
		return nil, "", fmt.Errorf("file is not an image")
	}
	content := strings.TrimSpace(fmt.Sprintf("%v", result["content"]))
	if content == "" {
		return nil, "", fmt.Errorf("image content is empty")
	}
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, "", fmt.Errorf("decode image content: %w", err)
	}
	return data, fmt.Sprintf("%v", result["mimeType"]), nil
}

func decodeClipboardImageDataURL(value string) ([]byte, string, bool, error) {
	text := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(text), "data:") {
		return nil, "", false, nil
	}
	comma := strings.Index(text, ",")
	if comma < 0 {
		return nil, "", true, fmt.Errorf("invalid image data url")
	}
	meta := text[5:comma]
	payload := text[comma+1:]
	mimeType := strings.TrimSpace(strings.Split(meta, ";")[0])
	if mimeType == "" {
		mimeType = "text/plain"
	}
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return nil, "", true, fmt.Errorf("data url is not an image")
	}
	if strings.Contains(strings.ToLower(meta), ";base64") {
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", true, fmt.Errorf("decode image data url: %w", err)
		}
		return data, mimeType, true, nil
	}
	decoded, err := url.PathUnescape(payload)
	if err != nil {
		return nil, "", true, err
	}
	return []byte(decoded), mimeType, true, nil
}

func clipboardImageDataFromHTTP(parent context.Context, value string) ([]byte, string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return nil, "", true, err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, "", false, nil
	}

	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", true, err
	}
	httpReq.Header.Set("Accept", "image/*,*/*;q=0.8")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, "", true, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", true, fmt.Errorf("download image failed: HTTP %d", resp.StatusCode)
	}

	mimeType := normalizedClipboardImageMIME(resp.Header.Get("Content-Type"))
	data, err := io.ReadAll(io.LimitReader(resp.Body, clipboardImageMaxBytes+1))
	if err != nil {
		return nil, "", true, err
	}
	if len(data) > clipboardImageMaxBytes {
		return nil, "", true, fmt.Errorf("image is too large to copy, max size is %s", humanBytes(clipboardImageMaxBytes))
	}
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return nil, "", true, fmt.Errorf("downloaded file is not an image")
	}
	return data, mimeType, true, nil
}

func clipboardImageToPNG(data []byte, mimeType string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("image content is empty")
	}
	if len(data) > clipboardImageMaxBytes {
		return nil, fmt.Errorf("image is too large to copy, max size is %s", humanBytes(clipboardImageMaxBytes))
	}
	if bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return data, nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		if strings.Contains(strings.ToLower(mimeType), "svg") {
			return nil, fmt.Errorf("svg images cannot be copied as png yet")
		}
		return nil, fmt.Errorf("image format is not supported for clipboard copy")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func normalizedClipboardImageMIME(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return strings.ToLower(value)
	}
	return strings.ToLower(mediaType)
}

func clipboardSnapshotID(itemType string, content string) string {
	sum := sha1.Sum([]byte(itemType + "\x00" + content))
	return itemType + "_" + hex.EncodeToString(sum[:8])
}

func isClipboardURL(value string) bool {
	text := strings.TrimSpace(value)
	if strings.ContainsAny(text, " \n\r\t") {
		return false
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") && parsed.Host != ""
}

func humanBytes(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}
