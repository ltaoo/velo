package desktopapp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ltaoo/velo"
)

func registerStorageRoutes(b *velo.Box) {
	b.Get("/api/settings/cloud-storage", func(c *velo.BoxContext) interface{} {
		raw := b.Store.Get(cloudStorageSettingsKey)
		settings, err := loadStoredCloudStorageSettings(raw)
		if err != nil {
			return c.Error(err.Error())
		}
		settings, changed, err := prepareCloudStorageSettings(settings, b.Store.Path(), raw == nil || !settings.DefaultsInitialized)
		if err != nil {
			return c.Error(err.Error())
		}
		if raw == nil || changed {
			stored, err := marshalCloudStorageSettingsForStore(settings)
			if err != nil {
				return c.Error(err.Error())
			}
			if err := b.Store.Set(cloudStorageSettingsKey, json.RawMessage(stored)); err != nil {
				return c.Error(err.Error())
			}
		}
		return c.Ok(velo.H{"found": true, "config": settings, "defaults": cloudStorageDefaults(b.Store.Path())})
	})

	b.Post("/api/settings/cloud-storage/save", func(c *velo.BoxContext) interface{} {
		var settings CloudStorageSettings
		if err := c.BindJSON(&settings); err != nil {
			return c.Error(err.Error())
		}

		settings = normalizeCloudStorageSettings(settings)
		settings, _, err := prepareCloudStorageSettings(settings, b.Store.Path(), len(settings.Storages) == 0)
		if err != nil {
			return c.Error(err.Error())
		}
		raw, err := marshalCloudStorageSettingsForStore(settings)
		if err != nil {
			return c.Error(err.Error())
		}
		if err := b.Store.Set(cloudStorageSettingsKey, json.RawMessage(raw)); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true, "config": settings})
	})

	b.Get("/api/settings/cloud-storage/delete", func(c *velo.BoxContext) interface{} {
		if err := b.Store.Delete(cloudStorageSettingsKey); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Post("/api/oss/upload", func(c *velo.BoxContext) interface{} {
		var req OSSUploadRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		if !hasOSSConfig(req.Config) {
			settings, err := loadStoredCloudStorageSettings(b.Store.Get(cloudStorageSettingsKey))
			if err != nil {
				return c.Error(err.Error())
			}
			settings, _, err = prepareCloudStorageSettings(settings, b.Store.Path(), len(settings.Storages) == 0)
			if err != nil {
				return c.Error(err.Error())
			}
			cfg, err := activeOSSConfig(settings, req.StorageID)
			if err != nil {
				return c.Error(err.Error())
			}
			req.Config = cfg
		} else if strings.TrimSpace(req.Config.ID) == "" && strings.TrimSpace(req.StorageID) != "" {
			req.Config.ID = req.StorageID
		}
		if isLocalOSSConfig(req.Config) {
			req.Config, _ = prepareLocalOSSConfig(req.Config, b.Store.Path())
		}

		result, err := uploadOSSObject(c.Context(), req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/list", func(c *velo.BoxContext) interface{} {
		var req OSSFileListRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := listOSSFiles(c.Context(), cfg, req.Path)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/preview", func(c *velo.BoxContext) interface{} {
		var req OSSFilePreviewRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := previewOSSFile(c.Context(), cfg, req.Path)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/mkdir", func(c *velo.BoxContext) interface{} {
		var req OSSFileMkdirRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := makeOSSFolder(c.Context(), cfg, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/delete", func(c *velo.BoxContext) interface{} {
		var req OSSFileDeleteRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := deleteOSSFile(c.Context(), cfg, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/upload", func(c *velo.BoxContext) interface{} {
		var req OSSFileUploadRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := uploadOSSManagedFile(c.Context(), cfg, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Get("/api/oss/assets", func(c *velo.BoxContext) interface{} {
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), c.Query("storageId"), b.Store.Path())
		if err != nil {
			writePlainError(c.Writer, http.StatusBadRequest, err.Error())
			return nil
		}
		objectPath := cleanOSSObjectPath(firstNonEmpty(c.Query("path"), c.Query("key")))
		if objectPath == "" {
			writePlainError(c.Writer, http.StatusBadRequest, "file path is required")
			return nil
		}
		if !isLocalOSSConfig(cfg) {
			endpoint := normalizeOSSEndpoint(cfg.Endpoint, cfg.UseSSL)
			c.Writer.Header().Set("Location", publicOSSObjectURL(cfg, endpoint, objectPath))
			c.Writer.WriteHeader(http.StatusFound)
			return nil
		}
		if err := serveLocalOSSAsset(c.Writer, cfg, objectPath); err != nil {
			writePlainError(c.Writer, http.StatusNotFound, err.Error())
		}
		return nil
	})
}
