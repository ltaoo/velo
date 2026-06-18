package desktopapp

import (
	"fmt"
	"os"
	"path/filepath"
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
	Fixed  *bool  `json:"fixed,omitempty"`
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
		if err := updatePersistedOpenWindowFrame(b.Store, req.Name, req.X, req.Y, req.Width, req.Height, req.Fixed); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Post("/api/window/session/save", func(c *velo.BoxContext) interface{} {
		var req PersistedOpenWindow
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			return c.Error("name is required")
		}
		if req.Width > 0 && req.Height > 0 {
			if err := b.Store.SaveWindow(req.Name, &store.WindowState{X: req.X, Y: req.Y, Width: req.Width, Height: req.Height}); err != nil {
				return c.Error(err.Error())
			}
		}
		if err := savePersistedWindowSession(b.Store, req); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/session/list", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"windows": loadPersistedOpenWindows(b.Store).Windows})
	})

	b.Get("/api/window/session/get", func(c *velo.BoxContext) interface{} {
		name := strings.TrimSpace(c.Query("name"))
		if name == "" {
			return c.Error("name is required")
		}
		session, ok := persistedWindowSession(b.Store, name)
		if !ok {
			return c.Ok(velo.H{"found": false})
		}
		return c.Ok(velo.H{"found": true, "session": session})
	})

	b.Get("/api/window/session/forget", func(c *velo.BoxContext) interface{} {
		name := strings.TrimSpace(c.Query("name"))
		if name == "" {
			return c.Error("name is required")
		}
		if err := forgetPersistedOpenWindow(b.Store, name); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/opened/forget", func(c *velo.BoxContext) interface{} {
		name := strings.TrimSpace(c.Query("name"))
		if name == "" {
			return c.Error("name is required")
		}
		if err := forgetPersistedOpenWindow(b.Store, name); err != nil {
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

		path, err := file.ShowFileSelectDialogWithOptions(file.FileSelectOptions{
			AnimationType: "default",
			AllowedTypes:  allowedTypes,
			Directory:     userDocumentsDirectory(),
		})
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

		editorSelection := editorSelectionForOpen(resolvedFile, c.Query("app"), c.Query("appName"), c.Query("appPath"), b.Store.Get(editorSettingsKey))
		if err := openFileInEditor(resolvedFile, line, col, editorSelection); err != nil {
			logger.Error().Err(err).Str("file", resolvedFile).Msg("failed to open file in editor")
			return c.Error(fmt.Sprintf("Failed to open editor: %v", err))
		}

		return c.Ok(velo.H{
			"success": true,
			"file":    resolvedFile,
			"line":    line,
			"col":     col,
			"editor":  normalizeEditorAppSelection(editorSelection),
		})
	})

	b.Get("/api/editor/apps", func(c *velo.BoxContext) interface{} {
		settings, _ := loadStoredEditorSettings(b.Store.Get(editorSettingsKey))
		return c.Ok(velo.H{
			"apps":     listEditorApplications(c.Query("q")),
			"selected": normalizeEditorAppSelection(settings.FileEditor),
		})
	})

	b.Get("/api/external/open", func(c *velo.BoxContext) interface{} {
		target, err := external.NormalizeBrowserURL(c.Query("url"))
		if err != nil {
			return c.Error(err.Error())
		}

		if !strings.EqualFold(strings.TrimSpace(c.Query("confirm")), "false") {
			confirmed, err := platform.ConfirmExternalBrowserOpen(external.BrowserConfirmMessage(target))
			if err != nil {
				logger.Error().Err(err).Str("url", target).Msg("failed to confirm external URL")
				return c.Error(fmt.Sprintf("Failed to show confirm dialog: %v", err))
			}
			if !confirmed {
				return c.Ok(velo.H{"success": false, "cancelled": true, "url": target})
			}
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

		windowName := memoWindowName(memoID)
		if err := rememberMemoWindow(b.Store, memoID, req); err != nil {
			return c.Error(err.Error())
		}
		b.OpenWindow(&velo.VeloWebviewOpt{
			Name:       windowName,
			Title:      "Memo",
			Pathname:   memoWindowPathname(memoID, req.Fixed),
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
			var found bool
			payload, found = loadPersistedMemoWindowPayload(b.Store, memoID)
			if !found {
				return c.Ok(velo.H{"found": false})
			}
			memoWindowCache.Lock()
			memoWindowCache.items[memoID] = payload
			memoWindowCache.Unlock()
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

func userDocumentsDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return ""
	}
	documentsDir := filepath.Join(homeDir, "Documents")
	if info, err := os.Stat(documentsDir); err == nil && info.IsDir() {
		return documentsDir
	}
	return homeDir
}
