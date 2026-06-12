package desktopapp

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ltaoo/clipboard-go"
	"github.com/ltaoo/velo"
	"github.com/rs/zerolog"
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
