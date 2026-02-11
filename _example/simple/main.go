package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/file"
	updater "github.com/ltaoo/velo/updater/api"
	uconfig "github.com/ltaoo/velo/updater/config"
	utypes "github.com/ltaoo/velo/updater/types"
	uversion "github.com/ltaoo/velo/updater/version"

	"github.com/rs/zerolog"
)

//go:embed frontend
var frontend_folder embed.FS

var Version = "(dev)"

func initUpdater(logger *zerolog.Logger) (*updater.AppUpdater, error) {
	updateConfig := uconfig.DefaultUpdaterConfig()
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
	if versionInfo.UpdateMode.ShouldCheckAtStartup() {
		go func() {
			time.Sleep(2 * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			u.CheckForUpdatesWithCallback(ctx, func(event utypes.UpdateEvent) {
				switch event.Type {
				case utypes.EventUpdateAvailable:
					fmt.Printf("New version available: %s\n", event.ReleaseInfo.Version)
				case utypes.EventNoUpdateAvailable:
					fmt.Println("You are running the latest version")
				case utypes.EventError:
					fmt.Printf("Update check failed: %v\n", event.Error)
				}
			})
		}()
	}
	return u, nil
}

func main() {
	writer := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(writer).With().Timestamp().Logger()
	logger.Info().Msgf("Version: %s, OS: %s/%s", Version, runtime.GOOS, runtime.GOARCH)

	app_updater, err := initUpdater(&logger)
	if err != nil {
		logger.Warn().Msgf("Updater init: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to get current working directory")
	}
	frontendDir := filepath.Join(cwd, "frontend")
	if _, err := os.Stat(filepath.Join(frontendDir, "index.html")); os.IsNotExist(err) {
		logger.Warn().Msgf("frontend/index.html not found in %s", frontendDir)
	}

	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, FrontendDir: frontendDir, FrontendFS: frontend_folder}
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
			return c.Ok(velo.H{"hasUpdate": false, "currentVersion": Version, "error": "Updater not initialized"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()
		releaseInfo, err := app_updater.CheckForUpdatesForce(ctx)
		if err != nil {
			return c.Ok(velo.H{"hasUpdate": false, "currentVersion": Version, "error": err.Error()})
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
		updatePath, err := app_updater.DownloadUpdate(ctx, releaseInfo, nil)
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

	fmt.Println("starting server on http://127.0.0.1:8080")

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
