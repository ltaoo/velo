package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/autostart"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/ltaoo/velo/file"
	"github.com/ltaoo/velo/shortcut"
	"github.com/ltaoo/velo/tray"
	updater "github.com/ltaoo/velo/updater/api"
	utypes "github.com/ltaoo/velo/updater/types"
	uversion "github.com/ltaoo/velo/updater/version"

	"github.com/rs/zerolog"
)

//go:embed frontend
var frontend_folder embed.FS

//go:embed app-config.json
var appConfigData []byte

//go:embed assets/appicon.png
var appIcon []byte

var Version = "1.0.0"
var Mode = "dev"

func setupLogger() *zerolog.Logger {
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".myapp")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	var writer io.Writer
	if err != nil {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else if Mode == "release" {
		writer = logFile
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

func initUpdater(logger *zerolog.Logger) (*updater.AppUpdater, error) {
	appCfg := velo.LoadAppConfig(appConfigData)
	updateConfig := appCfg.Update.ToUpdaterConfig()
	versionInfo := uversion.ParseVersionInfo(Version, updateConfig)
	if !versionInfo.UpdateMode.IsEnabled() {
		return nil, fmt.Errorf("auto-update is disabled (mode: %s)", versionInfo.UpdateMode)
	}
	effectiveVersion := Version
	if versionInfo.IsDevelopment() && updateConfig.DevVersion != "" {
		effectiveVersion = updateConfig.DevVersion
	}
	homeDir, _ := os.UserHomeDir()
	statePath := filepath.Join(homeDir, ".myapp", "update_state.json")
	opts := utypes.UpdaterOptions{
		Config:         updateConfig,
		CurrentVersion: effectiveVersion,
		Logger:         logger,
		StatePath:      statePath,
	}
	u, err := updater.NewUpdaterWithOptions(&opts, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}
	return u, nil
}

func main() {
	logger := setupLogger()
	logger.Info().Msgf("Version: %s, Mode: %s, OS: %s/%s", Version, Mode, runtime.GOOS, runtime.GOARCH)

	app_updater, err := initUpdater(logger)
	if err != nil {
		logger.Warn().Msgf("Updater init: %v", err)
	}

	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, FrontendFS: frontend_folder, IconData: appIcon}
	b := velo.NewApp(&opt)
	b.Get("/api/ping", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"message": "pong"})
	})
	b.Get("/api/file/select", func(c *velo.BoxContext) interface{} {
		path, err := file.ShowFileSelectDialog("default")
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"path": path})
	})
	b.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"version": Version})
	})
	b.Get("/api/update/check", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Error("Updater not initialized")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()
		releaseInfo, err := app_updater.CheckForUpdatesForce(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		if releaseInfo != nil && releaseInfo.IsNewer {
			return c.Ok(velo.H{"hasUpdate": true, "version": releaseInfo.Version, "currentVersion": Version, "releaseNotes": releaseInfo.ReleaseNotes})
		}
		return c.Ok(velo.H{"hasUpdate": false, "currentVersion": Version})
	})
	b.Get("/api/update/download", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		ctx := c.Context()
		releaseInfo, err := app_updater.CheckForUpdatesForce(ctx)
		if err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		if releaseInfo == nil || !releaseInfo.IsNewer {
			return c.Ok(velo.H{"success": false, "error": "No update available"})
		}
		updatePath, err := app_updater.DownloadUpdate(ctx, releaseInfo, func(progress utypes.DownloadProgress) {
			b.SendMessage(velo.H{
				"type":            "download_progress",
				"bytesDownloaded": progress.BytesDownloaded,
				"totalBytes":      progress.TotalBytes,
				"percentage":      progress.Percentage,
				"speed":           progress.Speed,
			})
		})
		if err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true, "updatePath": updatePath})
	})
	b.Get("/api/update/restart", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		if err := app_updater.ApplyUpdateThenRestartApplication(c.Context()); err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true})
	})
	b.Get("/api/update/skip", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		args, _ := c.Args().(map[string]interface{})
		v, _ := args["version"].(string)
		if v == "" {
			return c.Ok(velo.H{"success": false, "error": "version required"})
		}
		if err := app_updater.SkipVersion(v); err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true})
	})

	fmt.Println("starting server...")

	// 注册全局快捷键: Cmd+Shift+M (macOS) / Win+Shift+M (Windows) 显示/隐藏主窗口
	sm := shortcut.NewManager()
	sm.Register("MetaLeft+ShiftLeft+KeyM", func() {
		b.Webview.Show()
	})

	as := autostart.New("MyApp")

	proxyEnabled := false
	proxyItem := &tray.MenuItem{Label: "设置系统代理", Click: func(m *tray.MenuItem) {
		proxyEnabled = !proxyEnabled
		if proxyEnabled {
			m.Check()
		} else {
			m.Uncheck()
		}
	}}

	autoStartItem := &tray.MenuItem{Label: "开机自启动", Checked: as.IsEnabled(), Click: func(m *tray.MenuItem) {
		if as.IsEnabled() {
			as.Disable()
			m.Uncheck()
		} else {
			as.Enable()
			m.Check()
		}
	}}

	tray.Setup(&tray.Tray{
		Icon:    appIcon,
		Tooltip: "MyApp",
		Menu: &tray.Menu{
			Items: []*tray.MenuItem{
				{Label: "显示主窗口", Click: func(m *tray.MenuItem) {
					b.Webview.Show()
				}},
				proxyItem,
				autoStartItem,
				{IsSeparator: true},
				{Label: "退出", Click: func(m *tray.MenuItem) {
					tray.Quit()
				}},
			},
		},
	})

	b.NewWebview(&velo.VeloWebviewOpt{
		Pathname: "/home/index",
		Width:    1024,
		Height:   768,
		OnDragDrop: func(event string, payload string) {
			fmt.Printf("OnDragDrop: %s, %s\n", event, payload)
		},
	})
	b.Run()
}
