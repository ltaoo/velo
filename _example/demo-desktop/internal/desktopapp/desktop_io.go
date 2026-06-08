package desktopapp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ltaoo/velo"
	"github.com/rs/zerolog"
)

var memoWindowCache = struct {
	sync.RWMutex
	items map[string]MemoWindowPayload
}{
	items: make(map[string]MemoWindowPayload),
}

type MemoWindowPayload struct {
	Fixed bool            `json:"fixed"`
	Memo  json.RawMessage `json:"memo"`
	Memos json.RawMessage `json:"memos"`
}

func droppedFilesFromPayload(payload string, logger *zerolog.Logger) []velo.H {
	var paths []string
	if err := json.Unmarshal([]byte(payload), &paths); err != nil {
		logger.Error().Err(err).Msg("failed to parse dropped file payload")
		return nil
	}

	files := make([]velo.H, 0, len(paths))
	for _, path := range paths {
		file, err := droppedFileForPath(path)
		if err != nil {
			logger.Error().Err(err).Str("path", path).Msg("failed to read dropped file")
			continue
		}
		files = append(files, file)
	}
	return files
}

func droppedFileForPath(path string) (velo.H, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("dropped path is a directory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return velo.H{
		"name":    filepath.Base(path),
		"path":    path,
		"size":    info.Size(),
		"type":    contentType,
		"dataURL": "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

func memoWindowMemoID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || !json.Valid(raw) {
		return "", fmt.Errorf("memo is required")
	}
	var memo struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &memo); err != nil {
		return "", err
	}
	id := strings.TrimSpace(memo.ID)
	if id == "" {
		return "", fmt.Errorf("memo id is required")
	}
	return id, nil
}

func memoWindowMemosPayload(memo json.RawMessage, memos json.RawMessage) json.RawMessage {
	if len(memos) > 0 && json.Valid(memos) && strings.TrimSpace(string(memos)) != "null" {
		return memos
	}
	return json.RawMessage("[" + string(memo) + "]")
}
