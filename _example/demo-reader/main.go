package main

import (
	"embed"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ltaoo/velo"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/ltaoo/velo/file"
	"github.com/ltaoo/velo/shortcut"
	"github.com/ltaoo/velo/tray"
	"example/reader/store"
	"github.com/rs/zerolog"
)

//go:embed frontend
var frontend_folder embed.FS

//go:embed app-config.json
var appConfigData []byte

//go:embed assets/appicon.png
var appIcon []byte

var Version = "1.0.0"

func setupLogger() *zerolog.Logger {
	logDir := filepath.Join(os.TempDir(), "velo-demo-reader")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	var writer io.Writer
	if err != nil {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		writer = io.MultiWriter(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}, logFile)
	}
	logger := zerolog.New(writer).With().Timestamp().Logger()
	return &logger
}

func fatal(logger *zerolog.Logger, msg string) {
	logger.Error().Msg(msg)
	veloerr.ShowErrorDialog(msg)
	os.Exit(1)
}

func main() {
	logger := setupLogger()
	logger.Info().Msgf("Version: %s, OS: %s/%s", Version, runtime.GOOS, runtime.GOARCH)

	quitOnLastWindowClosed := false
	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, IconData: appIcon, QuitOnLastWindowClosed: &quitOnLastWindowClosed}
	b := velo.NewApp(&opt)

	st := store.New()
	logger.Info().Msgf("Store path: %s", st.Path())

	b.Get("/api/ping", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"message": "pong"})
	})
	b.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"version": Version})
	})

	b.Get("/api/window/set_pinned", func(c *velo.BoxContext) interface{} {
		pinned := c.Query("pinned") == "true"
		b.Webview.SetAlwaysOnTop(pinned)
		return c.Ok(velo.H{"success": true, "pinned": pinned})
	})

	b.Get("/api/window/hide", func(c *velo.BoxContext) interface{} {
		b.Webview.Hide()
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/show", func(c *velo.BoxContext) interface{} {
		b.Webview.Show()
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/state/save", func(c *velo.BoxContext) interface{} {
		name := c.Query("name")
		if name == "" {
			name = "default"
		}
		x, _ := strconv.Atoi(c.Query("x"))
		y, _ := strconv.Atoi(c.Query("y"))
		w, _ := strconv.Atoi(c.Query("width"))
		h, _ := strconv.Atoi(c.Query("height"))
		err := st.SaveWindow(name, &store.WindowState{X: x, Y: y, Width: w, Height: h})
		if err != nil {
			logger.Error().Err(err).Msg("failed to save window state")
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/state/load", func(c *velo.BoxContext) interface{} {
		name := c.Query("name")
		if name == "" {
			name = "default"
		}
		ws := st.GetWindow(name)
		if ws == nil {
			return c.Ok(velo.H{"found": false})
		}
		return c.Ok(velo.H{"found": true, "x": ws.X, "y": ws.Y, "width": ws.Width, "height": ws.Height})
	})

	b.Get("/api/window/state/snapshot", func(c *velo.BoxContext) interface{} {
		name := c.Query("name")
		if name == "" {
			name = "default"
		}
		x, y := b.Webview.GetPosition()
		w, h := b.Webview.GetSize()
		err := st.SaveWindow(name, &store.WindowState{X: x, Y: y, Width: w, Height: h})
		if err != nil {
			logger.Error().Err(err).Msg("failed to snapshot window state")
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true, "x": x, "y": y, "width": w, "height": h})
	})

	b.Get("/api/config/get", func(c *velo.BoxContext) interface{} {
		key := c.Query("key")
		if key == "" {
			return c.Ok(velo.H{"data": st.GetAll()})
		}
		v := st.Get(key)
		if v == nil {
			return c.Ok(velo.H{"found": false})
		}
		return c.Ok(velo.H{"found": true, "value": json.RawMessage(v)})
	})

	b.Get("/api/config/set", func(c *velo.BoxContext) interface{} {
		key := c.Query("key")
		val := c.Query("value")
		if key == "" {
			return c.Error("key is required")
		}
		if err := st.Set(key, json.RawMessage(val)); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/config/delete", func(c *velo.BoxContext) interface{} {
		key := c.Query("key")
		if key == "" {
			return c.Error("key is required")
		}
		if err := st.Delete(key); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/file/open", func(c *velo.BoxContext) interface{} {
		path, err := file.ShowFileSelectDialogWithTypes("default", []string{"txt"})
		if err != nil {
			return c.Error(err.Error())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return c.Error(err.Error())
		}
		name := filepath.Base(path)
		name = name[:len(name)-len(filepath.Ext(name))]
		return c.Ok(velo.H{"content": string(content), "title": name})
	})

	tray.Setup(&tray.Tray{
		Icon:    appIcon,
		Tooltip: "Reader",
		Menu: &tray.Menu{
			Items: []*tray.MenuItem{
				{Label: "显示阅读器", Click: func(m *tray.MenuItem) {
					b.Webview.Show()
				}},
				{IsSeparator: true},
				{Label: "退出", Click: func(m *tray.MenuItem) {
					tray.Quit()
				}},
			},
		},
	})

	// Register global shortcuts
	sm := shortcut.NewManager()
	sm.Register("MetaLeft+ShiftLeft+KeyS", func() {
		b.Webview.Show()
	})
	sm.Register("MetaLeft+ShiftLeft+KeyH", func() {
		b.Webview.Hide()
	})
	sm.Register("MetaLeft+ShiftLeft+KeyJ", func() {
		b.SendMessage(velo.H{"type": "startAutoScroll"})
	})
	sm.Register("MetaLeft+ShiftLeft+KeyK", func() {
		b.SendMessage(velo.H{"type": "stopAutoScroll"})
	})
	_ = sm

	windowName := "reader"
	initWidth := 400
	initHeight := 600
	savedState := st.GetWindow(windowName)
	if savedState != nil && savedState.Width > 0 && savedState.Height > 0 {
		initWidth = savedState.Width
		initHeight = savedState.Height
	}

	b.NewWebview(&velo.VeloWebviewOpt{
		Title:      "Reader",
		FrontendFS: frontend_folder,
		Pathname:   "/reader",
		Width:      initWidth,
		Height:     initHeight,
		Frameless:  true,
		Hidden:     true,
		OnDragDrop: func(event string, payload string) {
			if event != "drop" {
				return
			}
			var paths []string
			if err := json.Unmarshal([]byte(payload), &paths); err != nil {
				logger.Error().Err(err).Msg("failed to parse drop payload")
				return
			}
			for _, path := range paths {
				if !strings.HasSuffix(strings.ToLower(path), ".txt") {
					continue
				}
				content, err := os.ReadFile(path)
				if err != nil {
					logger.Error().Err(err).Str("path", path).Msg("failed to read dropped file")
					continue
				}
				name := filepath.Base(path)
				name = name[:len(name)-len(filepath.Ext(name))]
				b.SendMessage(velo.H{
					"type":    "fileDrop",
					"content": string(content),
					"title":   name,
				})
				break
			}
		},
	})

	if savedState != nil && (savedState.X != 0 || savedState.Y != 0) {
		b.Webview.SetPosition(savedState.X, savedState.Y)
	}

	b.Run()
}
