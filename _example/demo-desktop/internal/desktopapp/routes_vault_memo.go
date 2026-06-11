package desktopapp

import (
	"os"

	"example/simple/internal/desktopapp/platform"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/store"
)

func registerVaultProjectMemoRoutes(b *velo.Box) {
	b.Get("/api/ping", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"message": "pong"})
	})

	b.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"version": appVersion(), "velo": velo.GetVersion(), "mode": appMode()})
	})

	b.Get("/api/vault/status", func(c *velo.BoxContext) interface{} {
		registry, err := loadVaultRegistry()
		if err != nil {
			return c.Error(err.Error())
		}
		dataPath, err := globalVaultDataPath()
		if err != nil {
			return c.Error(err.Error())
		}
		_, statErr := os.Stat(dataPath)
		return c.Ok(velo.H{
			"active":         activeVaultSnapshot(),
			"activeVaultId":  registry.ActiveVaultID,
			"dataFileExists": statErr == nil,
			"dataPath":       dataPath,
			"vaults":         registry.Vaults,
		})
	})

	b.Get("/api/vault/select-directory", func(c *velo.BoxContext) interface{} {
		path, err := platform.SelectVaultDirectory()
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"path": path})
	})

	b.Post("/api/vault/open", func(c *velo.BoxContext) interface{} {
		var req VaultOpenRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		ctx, existing, err := openVaultDirectory(req.Path, true)
		if err != nil {
			return c.Error(err.Error())
		}
		registry, err := registerActiveVault(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		setActiveVault(ctx)
		setMainWindowPathname("/desktop")
		b.Store = store.NewWithDir(ctx.VeloDir)
		return c.Ok(velo.H{
			"active":   ctx,
			"created":  !existing,
			"existing": existing,
			"registry": registry,
		})
	})

	b.Get("/api/projects", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		file, err := listVaultProjects(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"activeProjectId": file.ActiveProjectID,
			"projects":        file.Projects,
		})
	})

	b.Post("/api/projects/create", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req ProjectCreateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		project, err := createVaultProject(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"project": project})
	})

	b.Post("/api/projects/update", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req ProjectUpdateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		project, err := updateVaultProject(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"project": project})
	})

	b.Post("/api/projects/activate", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req ProjectActivateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		file, err := activateVaultProject(ctx, req.ProjectID)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"activeProjectId": file.ActiveProjectID,
			"projects":        file.Projects,
		})
	})

	b.Get("/api/memos", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		memos, err := listVaultMemos(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"memos": memos})
	})

	b.Post("/api/memos/create", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoCreateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		memo, err := createVaultMemo(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"memo": memo})
	})

	b.Post("/api/memos/update", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoUpdateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		memo, err := updateVaultMemo(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"memo": memo})
	})

	b.Post("/api/memos/delete", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoDeleteRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cleanupAssets := true
		if req.CleanupAssets != nil {
			cleanupAssets = *req.CleanupAssets
		}
		deleteTasks := false
		if req.DeleteTasks != nil {
			deleteTasks = *req.DeleteTasks
		}
		result, err := deleteVaultMemoWithOptions(ctx, req.ID, MemoDeleteOptions{
			CleanupAssets:   cleanupAssets,
			DeleteTasks:     deleteTasks,
			Parent:          c.Context(),
			StorageSettings: b.Store.Get(cloudStorageSettingsKey),
			StorePath:       b.Store.Path(),
		})
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"assetErrors":   result.AssetErrors,
			"assetsDeleted": result.AssetsDeleted,
			"assetsSkipped": result.AssetsSkipped,
			"success":       true,
			"tasksDeleted":  result.TasksDeleted,
		})
	})

	b.Get("/api/memo-comments", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		comments, err := listVaultMemoComments(ctx, c.Query("memoId"))
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"comments": comments})
	})

	b.Post("/api/memo-comments/create", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoCommentCreateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		comment, err := createVaultMemoComment(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"comment": comment})
	})

	b.Post("/api/memo-comments/update", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoCommentUpdateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		comment, err := updateVaultMemoComment(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"comment": comment})
	})

	b.Post("/api/memo-comments/delete", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoCommentDeleteRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cleanupAssets := true
		if req.CleanupAssets != nil {
			cleanupAssets = *req.CleanupAssets
		}
		result, err := deleteVaultMemoCommentWithOptions(ctx, req.ID, MemoDeleteOptions{
			CleanupAssets:   cleanupAssets,
			Parent:          c.Context(),
			StorageSettings: b.Store.Get(cloudStorageSettingsKey),
			StorePath:       b.Store.Path(),
		})
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"assetErrors":   result.AssetErrors,
			"assetsDeleted": result.AssetsDeleted,
			"assetsSkipped": result.AssetsSkipped,
			"success":       true,
		})
	})

	b.Get("/api/memo-drafts", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		drafts, err := listVaultMemoDrafts(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"drafts": drafts})
	})

	b.Post("/api/memo-drafts/upsert", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoDraftUpsertRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		draft, err := upsertVaultMemoDraft(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"draft": draft})
	})

	b.Post("/api/memo-drafts/delete", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoDraftDeleteRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		if err := deleteVaultMemoDraft(ctx, req.ID); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})
}
