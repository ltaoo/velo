package main

import (
	"embed"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/database"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/ltaoo/velo/file"
	"github.com/ltaoo/velo/shortcut"
	"github.com/ltaoo/velo/tray"
	"github.com/rs/zerolog"
)

//go:embed frontend
var frontend_folder embed.FS

//go:embed app-config.json
var appConfigData []byte

//go:embed assets/appicon.png
var appIcon []byte

//go:embed migrations
var migrations embed.FS

var Version = "1.0.0"

type Novel struct {
	ID             uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string `json:"name" gorm:"not null"`
	Path           string `json:"path" gorm:"not null;uniqueIndex"`
	WordCount      int    `json:"word_count" gorm:"default:0"`
	FileSize       int64  `json:"file_size" gorm:"default:0"`
	CurrentChapter int    `json:"current_chapter" gorm:"default:0"`
	CurrentOffset  int    `json:"current_offset" gorm:"default:0"`
	IsCurrent      int    `json:"is_current" gorm:"default:0"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func (Novel) TableName() string {
	return "novels"
}

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

	logger.Info().Msgf("Store path: %s", b.Store.Path())

	if err := b.UseDatabase(database.DefaultSQLiteConfig(), &migrations); err != nil {
		fatal(logger, "failed to initialize database: "+err.Error())
	}

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

	b.Get("/api/file/open", func(c *velo.BoxContext) interface{} {
		path, err := file.ShowFileSelectDialogWithTypes("default", []string{"txt"})
		if err != nil {
			return c.Error(err.Error())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return c.Error(err.Error())
		}
		info, _ := os.Stat(path)
		var fileSize int64
		if info != nil {
			fileSize = info.Size()
		}
		name := filepath.Base(path)
		name = name[:len(name)-len(filepath.Ext(name))]
		return c.Ok(velo.H{"content": string(content), "title": name, "path": path, "file_size": fileSize})
	})

	b.Get("/api/file/read", func(c *velo.BoxContext) interface{} {
		path := c.Query("path")
		if path == "" {
			return c.Error("path is required")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return c.Error(err.Error())
		}
		name := filepath.Base(path)
		name = name[:len(name)-len(filepath.Ext(name))]
		return c.Ok(velo.H{"content": string(content), "title": name})
	})

	b.Post("/api/book/load", func(c *velo.BoxContext) interface{} {
		var req struct {
			Name      string `json:"name"`
			Path      string `json:"path"`
			WordCount int    `json:"word_count"`
			FileSize  int64  `json:"file_size"`
		}
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}

		// Clear is_current on all novels
		b.DB.Model(&Novel{}).Where("is_current = 1").Update("is_current", 0)

		// Upsert by path
		var novel Novel
		result := b.DB.Where("path = ?", req.Path).First(&novel)
		if result.Error != nil {
			// Create new
			novel = Novel{
				Name:      req.Name,
				Path:      req.Path,
				WordCount: req.WordCount,
				FileSize:  req.FileSize,
				IsCurrent: 1,
			}
			b.DB.Create(&novel)
		} else {
			// Update existing
			b.DB.Model(&novel).Updates(map[string]interface{}{
				"name":       req.Name,
				"word_count": req.WordCount,
				"file_size":  req.FileSize,
				"is_current": 1,
			})
		}

		return c.Ok(velo.H{"novel": novel})
	})

	b.Post("/api/book/progress", func(c *velo.BoxContext) interface{} {
		var req struct {
			Path           string `json:"path"`
			CurrentChapter int    `json:"current_chapter"`
			CurrentOffset  int    `json:"current_offset"`
		}
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}

		b.DB.Model(&Novel{}).Where("path = ?", req.Path).Updates(map[string]interface{}{
			"current_chapter": req.CurrentChapter,
			"current_offset":  req.CurrentOffset,
		})

		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/book/current", func(c *velo.BoxContext) interface{} {
		var novel Novel
		result := b.DB.Where("is_current = 1").First(&novel)
		if result.Error != nil {
			return c.Ok(velo.H{"found": false})
		}
		return c.Ok(velo.H{"found": true, "novel": novel})
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

	b.NewWebview(&velo.VeloWebviewOpt{
		Name:       "reader",
		Title:      "Reader",
		FrontendFS: frontend_folder,
		Pathname:   "/reader",
		Width:      400,
		Height:     600,
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
				info, _ := os.Stat(path)
				var fileSize int64
				if info != nil {
					fileSize = info.Size()
				}
				name := filepath.Base(path)
				name = name[:len(name)-len(filepath.Ext(name))]
				b.SendMessage(velo.H{
					"type":      "fileDrop",
					"content":   string(content),
					"title":     name,
					"path":      path,
					"file_size": fileSize,
				})
				break
			}
		},
	})

	b.Run()
}
