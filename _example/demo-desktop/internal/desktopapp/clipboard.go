package desktopapp

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
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
}

var recentClipboard = struct {
	sync.RWMutex
	item *ClipboardSnapshot
}{}

func registerClipboardRoutes(b *velo.Box, logger *zerolog.Logger) {
	b.Get("/api/clipboard/latest", func(c *velo.BoxContext) interface{} {
		item := latestClipboardSnapshot()
		if item == nil {
			return c.Ok(velo.H{"found": false})
		}
		return c.Ok(velo.H{"found": true, "item": item})
	})
}

func startClipboardWatcher(b *velo.Box, logger *zerolog.Logger) {
	if err := clipboard.Init(); err != nil {
		logger.Warn().Err(err).Msg("clipboard unavailable")
		return
	}

	go func() {
		ch := clipboard.Watch(context.Background())
		for data := range ch {
			if data.Error != nil {
				logger.Debug().Err(data.Error).Str("type", data.Type).Msg("clipboard read failed")
				continue
			}
			item, ok := normalizeClipboardContent(data)
			if !ok {
				continue
			}
			if storeClipboardSnapshot(item) {
				b.SendMessage(velo.H{
					"type": "clipboard_update",
					"item": item,
				})
			}
		}
	}()
}

func latestClipboardSnapshot() *ClipboardSnapshot {
	recentClipboard.RLock()
	defer recentClipboard.RUnlock()
	if recentClipboard.item == nil {
		return nil
	}
	item := *recentClipboard.item
	return &item
}

func storeClipboardSnapshot(item ClipboardSnapshot) bool {
	recentClipboard.Lock()
	defer recentClipboard.Unlock()
	if recentClipboard.item != nil && recentClipboard.item.ID == item.ID {
		return false
	}
	recentClipboard.item = &item
	return true
}

func normalizeClipboardContent(data clipboard.ClipboardContent) (ClipboardSnapshot, bool) {
	rawType := strings.TrimSpace(data.Type)
	capturedAt := time.Now().Format(time.RFC3339Nano)

	switch rawType {
	case "public.utf8-plain-text":
		text, ok := data.Data.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return ClipboardSnapshot{}, false
		}
		return textClipboardSnapshot(rawType, text, capturedAt), true
	case "public.html":
		text, _ := clipboard.ReadText()
		if strings.TrimSpace(text) == "" {
			text, _ = data.Data.(string)
		}
		if strings.TrimSpace(text) == "" {
			return ClipboardSnapshot{}, false
		}
		return textClipboardSnapshot(rawType, text, capturedAt), true
	case "public.png":
		imageData, ok := data.Data.([]byte)
		if !ok || len(imageData) == 0 {
			return ClipboardSnapshot{}, false
		}
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
		}, true
	default:
		return ClipboardSnapshot{}, false
	}
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
