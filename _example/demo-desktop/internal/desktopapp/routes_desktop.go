package desktopapp

import (
	"fmt"
	"net/url"
	"strings"

	"example/simple/internal/desktopapp/external"
	"example/simple/internal/desktopapp/platform"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/file"
	"github.com/ltaoo/velo/store"
	"github.com/rs/zerolog"
)

type WindowStateSaveRequest struct {
	Height int    `json:"height"`
	Name   string `json:"name"`
	Width  int    `json:"width"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

func memoWindowName(memoID string) string {
	nameSuffix := sanitizeStorageID(memoID)
	if nameSuffix == "" {
		nameSuffix = "memo"
	}
	return "memo-window-" + nameSuffix
}

func registerDesktopRoutes(b *velo.Box, logger *zerolog.Logger) {
	b.Get("/api/window/show", func(c *velo.BoxContext) interface{} {
		showMainWindow(b, logger)
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/hide", func(c *velo.BoxContext) interface{} {
		b.Webview.Hide()
		return c.Ok(velo.H{"success": true})
	})

	b.Post("/api/window/state/save", func(c *velo.BoxContext) interface{} {
		var req WindowStateSaveRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			return c.Error("name is required")
		}
		if req.Width <= 0 || req.Height <= 0 {
			return c.Error("width and height are required")
		}
		if err := b.Store.SaveWindow(req.Name, &store.WindowState{X: req.X, Y: req.Y, Width: req.Width, Height: req.Height}); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/state/restore", func(c *velo.BoxContext) interface{} {
		name := c.Query("name")
		if name == "" {
			name = "default"
		}
		ws := b.Store.GetWindow(name)
		if ws == nil {
			return c.Ok(velo.H{"found": false})
		}
		if ws.Width > 0 && ws.Height > 0 {
			b.Webview.SetSize(ws.Width, ws.Height)
		}
		if ws.X != 0 || ws.Y != 0 {
			b.Webview.SetPosition(ws.X, ws.Y)
		}
		return c.Ok(velo.H{"found": true, "x": ws.X, "y": ws.Y, "width": ws.Width, "height": ws.Height})
	})

	b.Get("/api/file/select", func(c *velo.BoxContext) interface{} {
		path, err := file.ShowFileSelectDialog("default")
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"path": path})
	})

	b.Get("/api/file/select-data-url", func(c *velo.BoxContext) interface{} {
		var allowedTypes []string
		if c.Query("accept") == "image" {
			allowedTypes = imageFileExtensions()
		}

		var path string
		var err error
		if len(allowedTypes) > 0 {
			path, err = file.ShowFileSelectDialogWithTypes("default", allowedTypes)
		} else {
			path, err = file.ShowFileSelectDialog("default")
		}
		if err != nil {
			return c.Error(err.Error())
		}

		selectedFile, err := droppedFileForPath(path)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"file": selectedFile})
	})

	b.Get("/api/editor/open", func(c *velo.BoxContext) interface{} {
		fileParam := strings.TrimSpace(c.Query("file"))
		if fileParam == "" {
			return c.Error("file is required")
		}

		fileTarget, embeddedLine, embeddedCol := splitEditorLocation(fileParam)
		line := editorPositionValue(c.Query("line"), embeddedLine)
		col := editorPositionValue(c.Query("col"), embeddedCol)
		resolvedFile, err := resolveEditorFileTarget(fileTarget, b.Store.Get(cloudStorageSettingsKey), b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}

		if err := openFileInEditor(resolvedFile, line, col, c.Query("app")); err != nil {
			logger.Error().Err(err).Str("file", resolvedFile).Msg("failed to open file in editor")
			return c.Error(fmt.Sprintf("Failed to open editor: %v", err))
		}

		return c.Ok(velo.H{
			"success": true,
			"file":    resolvedFile,
			"line":    line,
			"col":     col,
		})
	})

	b.Get("/api/external/open", func(c *velo.BoxContext) interface{} {
		target, err := external.NormalizeBrowserURL(c.Query("url"))
		if err != nil {
			return c.Error(err.Error())
		}

		confirmed, err := platform.ConfirmExternalBrowserOpen(external.BrowserConfirmMessage(target))
		if err != nil {
			logger.Error().Err(err).Str("url", target).Msg("failed to confirm external URL")
			return c.Error(fmt.Sprintf("Failed to show confirm dialog: %v", err))
		}
		if !confirmed {
			return c.Ok(velo.H{"success": false, "cancelled": true, "url": target})
		}

		if err := external.OpenBrowser(target); err != nil {
			logger.Error().Err(err).Str("url", target).Msg("failed to open external URL")
			return c.Error(fmt.Sprintf("Failed to open default browser: %v", err))
		}

		return c.Ok(velo.H{"success": true, "url": target})
	})

	b.Post("/api/memo-window/open", func(c *velo.BoxContext) interface{} {
		var req MemoWindowPayload
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}

		memoID, err := memoWindowMemoID(req.Memo)
		if err != nil {
			return c.Error(err.Error())
		}
		req.Memos = memoWindowMemosPayload(req.Memo, req.Memos)

		memoWindowCache.Lock()
		memoWindowCache.items[memoID] = req
		memoWindowCache.Unlock()

		params := url.Values{}
		params.Set("id", memoID)
		if req.Fixed {
			params.Set("fixed", "1")
		}

		windowName := memoWindowName(memoID)
		b.OpenWindow(&velo.VeloWebviewOpt{
			Name:       windowName,
			Title:      "Memo",
			Pathname:   "/memo-window?" + params.Encode(),
			Width:      460,
			Height:     560,
			Frameless:  true,
			EntryPage:  "memo-window.html",
			FrontendFS: appAssets.FrontendFS,
		})
		return c.Ok(velo.H{"success": true, "id": memoID, "windowName": windowName})
	})

	b.Get("/api/memo-window/get", func(c *velo.BoxContext) interface{} {
		memoID := strings.TrimSpace(c.Query("id"))
		if memoID == "" {
			return c.Error("id is required")
		}

		memoWindowCache.RLock()
		payload, ok := memoWindowCache.items[memoID]
		memoWindowCache.RUnlock()
		if !ok {
			return c.Ok(velo.H{"found": false})
		}
		return c.Ok(velo.H{
			"found":      true,
			"fixed":      payload.Fixed,
			"memo":       payload.Memo,
			"memos":      payload.Memos,
			"windowName": memoWindowName(memoID),
		})
	})
}
