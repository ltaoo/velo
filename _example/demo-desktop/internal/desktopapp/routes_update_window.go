package desktopapp

import (
	"context"
	"time"

	"example/simple/internal/desktopapp/windowing"

	"github.com/ltaoo/velo"
	updater "github.com/ltaoo/velo/updater/api"
	utypes "github.com/ltaoo/velo/updater/types"
)

func registerUpdateAndWindowRoutes(b *velo.Box, appUpdater *updater.AppUpdater) {
	b.Get("/api/update/check", func(c *velo.BoxContext) interface{} {
		if appUpdater == nil {
			return c.Error("Updater not initialized")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()
		releaseInfo, err := appUpdater.CheckForUpdatesForce(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		if releaseInfo != nil && releaseInfo.IsNewer {
			return c.Ok(velo.H{"hasUpdate": true, "version": releaseInfo.Version, "currentVersion": appVersion(), "releaseNotes": releaseInfo.ReleaseNotes})
		}
		return c.Ok(velo.H{"hasUpdate": false, "currentVersion": appVersion()})
	})
	b.Get("/api/update/download", func(c *velo.BoxContext) interface{} {
		if appUpdater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		ctx := c.Context()
		releaseInfo, err := appUpdater.CheckForUpdatesForce(ctx)
		if err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		if releaseInfo == nil || !releaseInfo.IsNewer {
			return c.Ok(velo.H{"success": false, "error": "No update available"})
		}
		updatePath, err := appUpdater.DownloadUpdate(ctx, releaseInfo, func(progress utypes.DownloadProgress) {
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
		if appUpdater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		if err := appUpdater.ApplyUpdateThenRestartApplication(c.Context()); err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true})
	})
	b.Get("/api/update/skip", func(c *velo.BoxContext) interface{} {
		if appUpdater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		args, _ := c.Args().(map[string]interface{})
		v, _ := args["version"].(string)
		if v == "" {
			return c.Ok(velo.H{"success": false, "error": "version required"})
		}
		if err := appUpdater.SkipVersion(v); err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/open_window", func(c *velo.BoxContext) interface{} {
		storageID := sanitizeStorageID(c.Query("storageId"))
		objectPath := cleanOSSObjectPath(c.Query("objectPath"))
		previewID := sanitizeStorageID(c.Query("previewId"))
		spec := windowing.BuildOpenWindowSpec(windowing.OpenWindowRequest{
			ObjectPath:       objectPath,
			ObjectPathSuffix: sanitizeStorageID(objectPath),
			Pathname:         c.Query("pathname"),
			PreviewID:        previewID,
			PreviewSrc:       c.Query("previewSrc"),
			PreviewTitle:     c.Query("previewTitle"),
			Provider:         c.Query("provider"),
			StorageID:        storageID,
		})
		b.OpenWindow(&velo.VeloWebviewOpt{
			Name:       spec.Name,
			Title:      spec.Title,
			Pathname:   spec.Pathname,
			Width:      spec.Width,
			Height:     spec.Height,
			EntryPage:  spec.EntryPage,
			FrontendFS: appAssets.FrontendFS,
		})
		return c.Ok(velo.H{"success": true})
	})
}
