package main

import (
	"embed"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ltaoo/velo"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/ltaoo/velo/tray"
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

	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, IconData: appIcon}
	b := velo.NewApp(&opt)

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

	b.NewWebview(&velo.VeloWebviewOpt{
		Title:              "Reader",
		FrontendFS:         frontend_folder,
		Pathname:           "/reader",
		Width:              400,
		Height:             600,
		Frameless:          true,
	})

	b.Run()
}
