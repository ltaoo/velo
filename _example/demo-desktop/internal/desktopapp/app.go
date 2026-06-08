package desktopapp

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ltaoo/velo"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/ltaoo/velo/shortcut"
	"github.com/ltaoo/velo/store"
	updater "github.com/ltaoo/velo/updater/api"
	utypes "github.com/ltaoo/velo/updater/types"
	uversion "github.com/ltaoo/velo/updater/version"

	"github.com/rs/zerolog"
)

type Assets struct {
	AppConfigData []byte
	AppIcon       []byte
	FrontendFS    fs.FS
	Mode          string
	ProjectDir    string
	Version       string
}

var appAssets Assets

func appVersion() string {
	if appAssets.Version == "" {
		return "1.0.0"
	}
	return appAssets.Version
}

func appMode() string {
	if appAssets.Mode == "" {
		return "dev"
	}
	return appAssets.Mode
}

func setupLogger() *zerolog.Logger {
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".myapp")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	var writer io.Writer
	if err != nil {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else if appMode() == "release" {
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

func projectDir() string {
	if appAssets.ProjectDir != "" {
		return appAssets.ProjectDir
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filename)
}

func initUpdater(logger *zerolog.Logger) (*updater.AppUpdater, error) {
	appCfg := velo.LoadAppConfig(appAssets.AppConfigData)
	updateConfig := appCfg.Update.ToUpdaterConfig()
	versionInfo := uversion.ParseVersionInfo(appVersion(), updateConfig)
	if !versionInfo.UpdateMode.IsEnabled() {
		return nil, fmt.Errorf("auto-update is disabled (mode: %s)", versionInfo.UpdateMode)
	}
	effectiveVersion := appVersion()
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

func Run(assets Assets) {
	appAssets = assets

	logger := setupLogger()
	logger.Info().Msgf("Version: %s, Velo: %s, Mode: %s, OS: %s/%s", appVersion(), velo.GetVersion(), appMode(), runtime.GOOS, runtime.GOARCH)

	app_updater, err := initUpdater(logger)
	if err != nil {
		logger.Warn().Msgf("Updater init: %v", err)
	}

	quitOnLastWindowClosed := false
	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, IconData: appAssets.AppIcon, QuitOnLastWindowClosed: &quitOnLastWindowClosed}
	b := velo.NewApp(&opt)
	initialPathname := "/vault-picker"
	if startupVault, err := loadStartupVault(); err != nil {
		logger.Warn().Msgf("Active vault unavailable: %v", err)
	} else if startupVault != nil {
		setActiveVault(startupVault)
		if _, err := registerActiveVault(startupVault); err != nil {
			logger.Warn().Msgf("Failed to update active vault registry: %v", err)
		}
		b.Store = store.NewWithDir(startupVault.VeloDir)
		initialPathname = "/desktop"
		logger.Info().Msgf("Active vault: %s", startupVault.RootDir)
	} else if dir, err := globalVeloDir(); err == nil {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Warn().Msgf("Failed to create global velo dir: %v", err)
		} else {
			b.Store = store.NewWithDir(dir)
		}
	}
	logger.Info().Msgf("Store path: %s", b.Store.Path())

	registerRoutes(b, logger, app_updater)

	fmt.Println("starting server...")

	// 注册全局快捷键: Cmd+Shift+M (macOS) / Win+Shift+M (Windows) 显示/隐藏主窗口
	sm := shortcut.NewManager()
	sm.Register("MetaLeft+ShiftLeft+KeyM", func() {
		b.Webview.Show()
	})
	sm.Register("MetaLeft+ShiftLeft+KeyH", func() {
		b.Webview.Hide()
	})
	_ = sm

	b.NewWebview(&velo.VeloWebviewOpt{
		Name:       "desktop",
		Title:      "App-Main",
		FrontendFS: appAssets.FrontendFS,
		Pathname:   initialPathname,
		Width:      1024,
		Height:     768,
		OnDragDrop: func(event string, payload string) {
			if event != "drop" {
				return
			}
			files := droppedFilesFromPayload(payload, logger)
			if len(files) == 0 {
				return
			}
			b.SendMessage(velo.H{
				"type":  "memo_file_drop",
				"files": files,
			})
		},
	})
	b.Run()
}
