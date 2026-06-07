package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/autostart"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/ltaoo/velo/file"
	"github.com/ltaoo/velo/shortcut"
	"github.com/ltaoo/velo/store"
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

const cloudStorageSettingsKey = "demo-desktop:settings:cloud-storage:v1"

type OSSConfig struct {
	AccessKeyID     string `json:"accessKeyId"`
	Bucket          string `json:"bucket"`
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	ForcePathStyle  bool   `json:"forcePathStyle"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	PathPrefix      string `json:"pathPrefix"`
	Provider        string `json:"provider"`
	PublicBaseURL   string `json:"publicBaseUrl"`
	Region          string `json:"region"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	UseSSL          bool   `json:"useSSL"`
}

type CloudStorageSettings struct {
	ActiveStorageID     string      `json:"activeStorageId"`
	DefaultsInitialized bool        `json:"defaultsInitialized,omitempty"`
	Storages            []OSSConfig `json:"storages"`
}

type OSSUploadRequest struct {
	Config        OSSConfig `json:"config"`
	ContentBase64 string    `json:"content_base64"`
	Name          string    `json:"name"`
	StorageID     string    `json:"storageId"`
	Type          string    `json:"type"`
}

type OSSFileListRequest struct {
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFilePreviewRequest struct {
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileMkdirRequest struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileDeleteRequest struct {
	IsDir     bool   `json:"isDir"`
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileUploadRequest struct {
	ContentBase64 string `json:"content_base64"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	StorageID     string `json:"storageId"`
	Type          string `json:"type"`
}

type OSSFileView struct {
	ID      string `json:"id"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Ref     string `json:"ref"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	URL     string `json:"url"`
}

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

func projectDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filename)
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
	logger.Info().Msgf("Version: %s, Velo: %s, Mode: %s, OS: %s/%s", Version, velo.GetVersion(), Mode, runtime.GOOS, runtime.GOARCH)

	app_updater, err := initUpdater(logger)
	if err != nil {
		logger.Warn().Msgf("Updater init: %v", err)
	}

	quitOnLastWindowClosed := false
	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, IconData: appIcon, QuitOnLastWindowClosed: &quitOnLastWindowClosed}
	b := velo.NewApp(&opt)
	if Mode == "dev" {
		b.Store = store.NewWithDir(projectDir())
	}
	logger.Info().Msgf("Store path: %s", b.Store.Path())

	b.Get("/api/ping", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"message": "pong"})
	})

	b.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"version": Version, "velo": velo.GetVersion(), "mode": Mode})
	})

	b.Get("/api/window/show", func(c *velo.BoxContext) interface{} {
		b.Webview.Show()
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/hide", func(c *velo.BoxContext) interface{} {
		b.Webview.Hide()
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

		var path string
		var err error
		if len(allowedTypes) > 0 {
			path, err = file.ShowFileSelectDialogWithTypes("default", allowedTypes)
		} else {
			path, err = file.ShowFileSelectDialog("default")
		}
		if err != nil {
			return c.Error(err.Error())
		}

		selectedFile, err := droppedFileForPath(path)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"file": selectedFile})
	})

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
			stored, err := json.Marshal(settings)
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
		raw, err := json.Marshal(settings)
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

	b.Get("/api/open_window", func(c *velo.BoxContext) interface{} {
		pathname := c.Query("pathname")
		if pathname == "" {
			pathname = "/settings"
		}
		storageID := sanitizeStorageID(c.Query("storageId"))
		objectPath := cleanOSSObjectPath(c.Query("objectPath"))
		provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
		if pathname == "/oss-manager" && storageID != "" {
			pathname += "?storageId=" + url.QueryEscape(storageID)
		}
		if pathname == "/oss-storage-editor" {
			params := url.Values{}
			if storageID != "" {
				params.Set("storageId", storageID)
			}
			if provider != "" {
				params.Set("provider", provider)
			}
			if encoded := params.Encode(); encoded != "" {
				pathname += "?" + encoded
			}
		}
		if pathname == "/oss-preview" {
			params := url.Values{}
			if storageID != "" {
				params.Set("storageId", storageID)
			}
			if objectPath != "" {
				params.Set("objectPath", objectPath)
			}
			if encoded := params.Encode(); encoded != "" {
				pathname += "?" + encoded
			}
		}
		pathBase := pathname
		if index := strings.Index(pathBase, "?"); index >= 0 {
			pathBase = pathBase[:index]
		}
		entryPage := "index.html"
		name := "app-window"
		title := "App"
		width := 760
		height := 640
		if pathBase == "/settings" {
			entryPage = "settings.html"
			name = "settings"
			title = "App-Settings"
		}
		if pathBase == "/oss-manager" {
			entryPage = "oss-manager.html"
			name = "oss-manager"
			title = "OSS 文件管理"
			width = 1040
			height = 720
			if storageID != "" {
				name += "-" + storageID
			}
		}
		if pathBase == "/oss-storage-editor" {
			entryPage = "oss-storage-editor.html"
			name = "oss-storage-editor"
			title = "OSS 存储编辑"
			width = 760
			height = 720
		}
		if pathBase == "/oss-preview" {
			entryPage = "oss-preview.html"
			name = "oss-preview"
			title = "OSS 文件预览"
			width = 860
			height = 680
			if storageID != "" {
				name += "-" + storageID
			}
			if objectPath != "" {
				name += "-" + sanitizeStorageID(objectPath)
			}
		}
		b.OpenWindow(&velo.VeloWebviewOpt{
			Name:       name,
			Title:      title,
			Pathname:   pathname,
			Width:      width,
			Height:     height,
			EntryPage:  entryPage,
			FrontendFS: frontend_folder,
		})
		return c.Ok(velo.H{"success": true})
	})

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
		Name:       "desktop",
		Title:      "App-Main",
		FrontendFS: frontend_folder,
		Pathname:   "/desktop",
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

func droppedFilesFromPayload(payload string, logger *zerolog.Logger) []velo.H {
	var paths []string
	if err := json.Unmarshal([]byte(payload), &paths); err != nil {
		logger.Error().Err(err).Msg("failed to parse dropped file payload")
		return nil
	}

	files := make([]velo.H, 0, len(paths))
	for _, path := range paths {
		file, err := droppedFileForPath(path)
		if err != nil {
			logger.Error().Err(err).Str("path", path).Msg("failed to read dropped file")
			continue
		}
		files = append(files, file)
	}
	return files
}

func droppedFileForPath(path string) (velo.H, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("dropped path is a directory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return velo.H{
		"name":    filepath.Base(path),
		"path":    path,
		"size":    info.Size(),
		"type":    contentType,
		"dataURL": "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

func imageFileExtensions() []string {
	return []string{"avif", "bmp", "gif", "jpg", "jpeg", "png", "svg", "webp"}
}

func uploadOSSObject(parent context.Context, req OSSUploadRequest) (velo.H, error) {
	cfg := req.Config
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return uploadLocalOSSObject(parent, req)
	}

	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	key := objectKey(cfg.PathPrefix, req.Name)
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"bucket":    cfg.Bucket,
		"key":       key,
		"name":      req.Name,
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, endpoint, key),
	}, nil
}

func storedOSSConfig(raw json.RawMessage, storageID string, storePath string) (OSSConfig, error) {
	settings, err := loadStoredCloudStorageSettings(raw)
	if err != nil {
		return OSSConfig{}, err
	}
	settings, _, err = prepareCloudStorageSettings(settings, storePath, len(settings.Storages) == 0)
	if err != nil {
		return OSSConfig{}, err
	}
	cfg, err := activeOSSConfig(settings, storageID)
	if err != nil {
		return OSSConfig{}, err
	}
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, storageID, "default"))
	return cfg, nil
}

func listOSSFiles(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return listLocalOSSFiles(parent, cfg, objectPath)
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	cleanPath := cleanOSSObjectPath(objectPath)
	prefix := ossFolderPrefix(cleanPath)
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(cfg.Bucket),
		Delimiter: aws.String("/"),
		MaxKeys:   1000,
		Prefix:    aws.String(prefix),
	}
	seen := map[string]bool{}
	items := make([]OSSFileView, 0)
	for {
		out, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, commonPrefix := range out.CommonPrefixes {
			key := stringValue(commonPrefix.Prefix)
			view := ossFileView(cfg, endpoint, cleanPath, key, true, 0, nil, "")
			if view.Path == "" || seen[view.Path] {
				continue
			}
			seen[view.Path] = true
			items = append(items, view)
		}

		for _, object := range out.Contents {
			key := stringValue(object.Key)
			if key == "" || key == prefix {
				continue
			}
			isDir := strings.HasSuffix(key, "/")
			view := ossFileView(cfg, endpoint, cleanPath, key, isDir, object.Size, object.LastModified, "")
			if view.Path == "" || seen[view.Path] {
				continue
			}
			seen[view.Path] = true
			items = append(items, view)
		}

		if !out.IsTruncated || out.NextContinuationToken == nil {
			break
		}
		input.ContinuationToken = out.NextContinuationToken
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return velo.H{
		"bucket":    cfg.Bucket,
		"list":      items,
		"path":      cleanPath,
		"prefix":    prefix,
		"storageId": cfg.ID,
	}, nil
}

func previewOSSFile(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return previewLocalOSSFile(parent, cfg, objectPath)
	}
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}

	client, _, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	if head.ContentLength > 8*1024*1024 {
		return nil, fmt.Errorf("file is too large to preview, max size is 8 MB")
	}

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	content, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	name := pathpkg.Base(key)
	ext := strings.ToLower(filepath.Ext(name))
	contentType := firstNonEmpty(stringValue(out.ContentType), stringValue(head.ContentType), mime.TypeByExtension(ext), "application/octet-stream")
	if isTextPreview(ext, contentType) {
		return velo.H{
			"content":  string(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "text",
		}, nil
	}
	if strings.HasPrefix(contentType, "image/") {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "image",
		}, nil
	}
	if contentType == "application/pdf" {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "pdf",
		}, nil
	}
	return velo.H{
		"mimeType": contentType,
		"name":     name,
		"path":     key,
		"size":     head.ContentLength,
		"type":     "unknown",
	}, nil
}

func makeOSSFolder(parent context.Context, cfg OSSConfig, req OSSFileMkdirRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return makeLocalOSSFolder(parent, cfg, req)
	}

	folderPath := cleanOSSObjectPath(req.Path)
	if strings.TrimSpace(req.Name) != "" {
		folderPath = objectPathJoin(folderPath, req.Name)
	}
	if folderPath == "" {
		return nil, fmt.Errorf("folder path is required")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	key := ossFolderPrefix(folderPath)
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(nil),
		ContentType: aws.String("application/x-directory"),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"file":      ossFileView(cfg, endpoint, pathpkg.Dir(folderPath), key, true, 0, nil, "application/x-directory"),
		"path":      folderPath,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func deleteOSSFile(parent context.Context, cfg OSSConfig, req OSSFileDeleteRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return deleteLocalOSSFile(parent, cfg, req)
	}

	key := cleanOSSObjectPath(req.Path)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}

	client, _, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	deleted := 0
	if req.IsDir {
		prefix := ossFolderPrefix(key)
		input := &s3.ListObjectsV2Input{
			Bucket:  aws.String(cfg.Bucket),
			MaxKeys: 1000,
			Prefix:  aws.String(prefix),
		}
		for {
			out, err := client.ListObjectsV2(ctx, input)
			if err != nil {
				return nil, err
			}
			for _, object := range out.Contents {
				objectKey := stringValue(object.Key)
				if objectKey == "" {
					continue
				}
				if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: aws.String(cfg.Bucket),
					Key:    aws.String(objectKey),
				}); err != nil {
					return nil, err
				}
				deleted++
			}
			if !out.IsTruncated || out.NextContinuationToken == nil {
				break
			}
			input.ContinuationToken = out.NextContinuationToken
		}
	} else {
		if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
		}); err != nil {
			return nil, err
		}
		deleted = 1
	}

	return velo.H{
		"deleted":   deleted,
		"path":      key,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func uploadOSSManagedFile(parent context.Context, cfg OSSConfig, req OSSFileUploadRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return uploadLocalOSSManagedFile(parent, cfg, req)
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("file name is required")
	}

	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	key := objectPathJoin(req.Path, req.Name)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"bucket":    cfg.Bucket,
		"file":      ossFileView(cfg, endpoint, cleanOSSObjectPath(req.Path), key, false, int64(len(data)), nil, contentType),
		"key":       key,
		"name":      sanitizeObjectName(req.Name),
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"success":   true,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, endpoint, key),
	}, nil
}

func uploadLocalOSSObject(parent context.Context, req OSSUploadRequest) (velo.H, error) {
	cfg := req.Config
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSConfig(cfg); err != nil {
		return nil, err
	}
	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	key := objectKey(cfg.PathPrefix, req.Name)
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := writeLocalOSSObject(parent, cfg, key, data); err != nil {
		return nil, err
	}
	return velo.H{
		"bucket":    cfg.Bucket,
		"key":       key,
		"name":      req.Name,
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, "", key),
	}, nil
}

func listLocalOSSFiles(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cleanPath := cleanOSSObjectPath(objectPath)
	target, err := localOSSObjectDiskPath(cfg, cleanPath)
	if err != nil {
		return nil, err
	}
	if _, err := ensureLocalOSSBucket(cfg); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			return velo.H{
				"bucket":    cfg.Bucket,
				"list":      []OSSFileView{},
				"path":      cleanPath,
				"prefix":    ossFolderPrefix(cleanPath),
				"storageId": cfg.ID,
			}, nil
		}
		return nil, err
	}
	items := make([]OSSFileView, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-parent.Done():
			return nil, parent.Err()
		default:
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		key := objectPathJoin(cleanPath, entry.Name())
		modTime := info.ModTime()
		size := info.Size()
		if entry.IsDir() {
			size = 0
		}
		items = append(items, ossFileView(cfg, "", cleanPath, key, entry.IsDir(), size, &modTime, ""))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return velo.H{
		"bucket":    cfg.Bucket,
		"list":      items,
		"path":      cleanPath,
		"prefix":    ossFolderPrefix(cleanPath),
		"storageId": cfg.ID,
	}, nil
}

func previewLocalOSSFile(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("folder cannot be previewed")
	}
	if info.Size() > 8*1024*1024 {
		return nil, fmt.Errorf("file is too large to preview, max size is 8 MB")
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	name := pathpkg.Base(key)
	ext := strings.ToLower(filepath.Ext(name))
	contentType := firstNonEmpty(mime.TypeByExtension(ext), http.DetectContentType(content), "application/octet-stream")
	if isTextPreview(ext, contentType) {
		return velo.H{
			"content":  string(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "text",
		}, nil
	}
	if strings.HasPrefix(contentType, "image/") {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "image",
		}, nil
	}
	if contentType == "application/pdf" {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "pdf",
		}, nil
	}
	return velo.H{
		"mimeType": contentType,
		"name":     name,
		"path":     key,
		"size":     info.Size(),
		"type":     "unknown",
	}, nil
}

func makeLocalOSSFolder(parent context.Context, cfg OSSConfig, req OSSFileMkdirRequest) (velo.H, error) {
	folderPath := cleanOSSObjectPath(req.Path)
	if strings.TrimSpace(req.Name) != "" {
		folderPath = objectPathJoin(folderPath, req.Name)
	}
	if folderPath == "" {
		return nil, fmt.Errorf("folder path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, folderPath)
	if err != nil {
		return nil, err
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, err
	}
	return velo.H{
		"file":      ossFileView(cfg, "", pathpkg.Dir(folderPath), folderPath, true, 0, nil, "application/x-directory"),
		"path":      folderPath,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func deleteLocalOSSFile(parent context.Context, cfg OSSConfig, req OSSFileDeleteRequest) (velo.H, error) {
	key := cleanOSSObjectPath(req.Path)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return nil, err
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	if err := os.RemoveAll(target); err != nil {
		return nil, err
	}
	return velo.H{
		"deleted":   1,
		"path":      key,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func uploadLocalOSSManagedFile(parent context.Context, cfg OSSConfig, req OSSFileUploadRequest) (velo.H, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("file name is required")
	}
	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	key := objectPathJoin(req.Path, req.Name)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := writeLocalOSSObject(parent, cfg, key, data); err != nil {
		return nil, err
	}
	return velo.H{
		"bucket":    cfg.Bucket,
		"file":      ossFileView(cfg, "", cleanOSSObjectPath(req.Path), key, false, int64(len(data)), nil, contentType),
		"key":       key,
		"name":      sanitizeObjectName(req.Name),
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"success":   true,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, "", key),
	}, nil
}

func writeLocalOSSObject(parent context.Context, cfg OSSConfig, key string, data []byte) error {
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return err
	}
	select {
	case <-parent.Done():
		return parent.Err()
	default:
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0644)
}

func serveLocalOSSAsset(w http.ResponseWriter, cfg OSSConfig, objectPath string) error {
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("folder cannot be served")
	}
	file, err := os.Open(target)
	if err != nil {
		return err
	}
	defer file.Close()
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(target)))
	if contentType == "" {
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, err = io.Copy(w, file)
	return err
}

func writePlainError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}

func isLocalOSSConfig(cfg OSSConfig) bool {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	return provider == "local" || provider == "local-oss"
}

func ensureLocalOSSBucket(cfg OSSConfig) (string, error) {
	root, err := localOSSBucketRoot(cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	return root, nil
}

func localOSSBucketRoot(cfg OSSConfig) (string, error) {
	if err := validateOSSAccessConfig(cfg); err != nil {
		return "", err
	}
	root := expandLocalPath(strings.TrimSpace(cfg.Endpoint))
	if root == "" {
		return "", fmt.Errorf("local root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if err := validateLocalOSSBucket(bucket); err != nil {
		return "", err
	}
	return filepath.Join(absRoot, bucket), nil
}

func localOSSObjectDiskPath(cfg OSSConfig, objectPath string) (string, error) {
	bucketRoot, err := ensureLocalOSSBucket(cfg)
	if err != nil {
		return "", err
	}
	cleanKey := cleanOSSObjectPath(objectPath)
	target := bucketRoot
	if cleanKey != "" {
		target = filepath.Join(bucketRoot, filepath.FromSlash(cleanKey))
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != bucketRoot && !strings.HasPrefix(absTarget, bucketRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("object path escapes bucket root: %s", objectPath)
	}
	return absTarget, nil
}

func expandLocalPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(value, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	return value
}

func validateLocalOSSBucket(bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if bucket == "." || bucket == ".." {
		return fmt.Errorf("invalid bucket: %s", bucket)
	}
	if strings.ContainsAny(bucket, `/\`) {
		return fmt.Errorf("bucket must not contain path separators: %s", bucket)
	}
	return nil
}

func loadStoredCloudStorageSettings(raw json.RawMessage) (CloudStorageSettings, error) {
	if raw == nil {
		return normalizeCloudStorageSettings(CloudStorageSettings{}), nil
	}

	var settings CloudStorageSettings
	if err := json.Unmarshal(raw, &settings); err == nil && (settings.Storages != nil || strings.TrimSpace(settings.ActiveStorageID) != "") {
		return normalizeCloudStorageSettings(settings), nil
	}

	var cfg OSSConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return CloudStorageSettings{}, fmt.Errorf("read cloud storage config: %w", err)
	}
	if !hasOSSConfig(cfg) {
		return CloudStorageSettings{}, fmt.Errorf("cloud storage config is empty")
	}
	cfg.ID = firstNonEmpty(cfg.ID, "default")
	cfg.Name = firstNonEmpty(cfg.Name, "默认存储")
	return normalizeCloudStorageSettings(CloudStorageSettings{
		ActiveStorageID: cfg.ID,
		Storages:        []OSSConfig{cfg},
	}), nil
}

func normalizeCloudStorageSettings(settings CloudStorageSettings) CloudStorageSettings {
	seen := map[string]int{}
	next := make([]OSSConfig, 0, len(settings.Storages))
	for i, cfg := range settings.Storages {
		cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
		if cfg.Provider == "" {
			cfg.Provider = "s3"
		}
		cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
		cfg.Bucket = strings.TrimSpace(cfg.Bucket)
		cfg.PathPrefix = strings.TrimSpace(cfg.PathPrefix)
		cfg.PublicBaseURL = strings.TrimSpace(cfg.PublicBaseURL)
		cfg.Region = strings.TrimSpace(cfg.Region)
		baseID := sanitizeStorageID(cfg.ID)
		if baseID == "" {
			baseID = sanitizeStorageID(firstNonEmpty(cfg.Name, cfg.Provider, cfg.Bucket))
		}
		if baseID == "" {
			baseID = fmt.Sprintf("storage-%d", i+1)
		}
		seen[baseID]++
		if seen[baseID] > 1 {
			baseID = fmt.Sprintf("%s-%d", baseID, seen[baseID])
		}
		cfg.ID = baseID
		if strings.TrimSpace(cfg.Name) == "" {
			cfg.Name = storageDisplayName(cfg, i)
		} else {
			cfg.Name = strings.TrimSpace(cfg.Name)
		}
		next = append(next, cfg)
	}

	activeID := sanitizeStorageID(settings.ActiveStorageID)
	if !storageIDExists(next, activeID) {
		activeID = ""
		for _, cfg := range next {
			if cfg.Enabled {
				activeID = cfg.ID
				break
			}
		}
		if activeID == "" && len(next) > 0 {
			activeID = next[0].ID
		}
	}

	return CloudStorageSettings{
		ActiveStorageID:     activeID,
		DefaultsInitialized: settings.DefaultsInitialized,
		Storages:            next,
	}
}

func prepareCloudStorageSettings(settings CloudStorageSettings, storePath string, initializeDefault bool) (CloudStorageSettings, bool, error) {
	settings = normalizeCloudStorageSettings(settings)
	changed := false
	if initializeDefault || len(settings.Storages) == 0 {
		defaultCfg := defaultLocalMemoOSSConfig(storePath)
		if !storageIDExists(settings.Storages, defaultCfg.ID) {
			settings.Storages = append(settings.Storages, defaultCfg)
			changed = true
		}
		if settings.ActiveStorageID == "" {
			settings.ActiveStorageID = defaultCfg.ID
			changed = true
		}
	}
	if !settings.DefaultsInitialized {
		settings.DefaultsInitialized = true
		changed = true
	}
	settings = normalizeCloudStorageSettings(settings)
	for _, cfg := range settings.Storages {
		if isLocalOSSConfig(cfg) && strings.TrimSpace(cfg.Endpoint) != "" && strings.TrimSpace(cfg.Bucket) != "" {
			if _, err := ensureLocalOSSBucket(cfg); err != nil {
				return CloudStorageSettings{}, changed, err
			}
		}
	}
	return settings, changed, nil
}

func cloudStorageDefaults(storePath string) velo.H {
	return velo.H{
		"localRoot":  defaultLocalStorageRoot(storePath),
		"memoBucket": "memos",
	}
}

func defaultLocalMemoOSSConfig(storePath string) OSSConfig {
	return OSSConfig{
		Bucket:         "memos",
		Enabled:        true,
		Endpoint:       defaultLocalStorageRoot(storePath),
		ForcePathStyle: true,
		ID:             "memo-local",
		Name:           "本地 Memo 存储",
		Provider:       "local",
		UseSSL:         false,
	}
}

func defaultLocalStorageRoot(storePath string) string {
	base := filepath.Dir(strings.TrimSpace(storePath))
	if base == "" || base == "." {
		base = projectDir()
	}
	return filepath.Join(base, "workdir", "storage")
}

func activeOSSConfig(settings CloudStorageSettings, storageID string) (OSSConfig, error) {
	settings = normalizeCloudStorageSettings(settings)
	targetID := sanitizeStorageID(storageID)
	if targetID == "" {
		targetID = settings.ActiveStorageID
	}
	if targetID == "" {
		return OSSConfig{}, fmt.Errorf("cloud storage config is not saved")
	}
	for _, cfg := range settings.Storages {
		if cfg.ID == targetID {
			return cfg, nil
		}
	}
	return OSSConfig{}, fmt.Errorf("cloud storage profile not found: %s", targetID)
}

func storageIDExists(storages []OSSConfig, id string) bool {
	if id == "" {
		return false
	}
	for _, cfg := range storages {
		if cfg.ID == id {
			return true
		}
	}
	return false
}

func storageDisplayName(cfg OSSConfig, index int) string {
	for _, value := range []string{cfg.Bucket, cfg.Provider, cfg.ID} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return fmt.Sprintf("存储 %d", index+1)
}

func sanitizeStorageID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func hasOSSConfig(cfg OSSConfig) bool {
	return cfg.Enabled ||
		strings.TrimSpace(cfg.Endpoint) != "" ||
		strings.TrimSpace(cfg.Bucket) != "" ||
		strings.TrimSpace(cfg.AccessKeyID) != "" ||
		strings.TrimSpace(cfg.SecretAccessKey) != "" ||
		strings.TrimSpace(cfg.SessionToken) != "" ||
		strings.TrimSpace(cfg.PublicBaseURL) != "" ||
		strings.TrimSpace(cfg.PathPrefix) != "" ||
		strings.TrimSpace(cfg.Region) != ""
}

func validateOSSConfig(cfg OSSConfig) error {
	if !cfg.Enabled {
		return fmt.Errorf("cloud storage is not enabled")
	}
	return validateOSSAccessConfig(cfg)
}

func validateOSSAccessConfig(cfg OSSConfig) error {
	missing := make([]string, 0, 5)
	if strings.TrimSpace(cfg.Endpoint) == "" {
		if isLocalOSSConfig(cfg) {
			missing = append(missing, "local root")
		} else {
			missing = append(missing, "endpoint")
		}
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		missing = append(missing, "bucket")
	}
	if !isLocalOSSConfig(cfg) {
		if strings.TrimSpace(cfg.AccessKeyID) == "" {
			missing = append(missing, "access key id")
		}
		if strings.TrimSpace(cfg.SecretAccessKey) == "" {
			missing = append(missing, "secret access key")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("cloud storage config missing: %s", strings.Join(missing, ", "))
	}
	if isLocalOSSConfig(cfg) {
		return validateLocalOSSBucket(cfg.Bucket)
	}
	return nil
}

func newOSSClient(cfg OSSConfig) (*s3.Client, string, error) {
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, "", err
	}

	endpoint := normalizeOSSEndpoint(cfg.Endpoint, cfg.UseSSL)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	awsCfg := aws.Config{
		Region: region,
		Credentials: aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
				SessionToken:    cfg.SessionToken,
				Source:          "oss-config",
			}, nil
		})),
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
		o.UsePathStyle = cfg.ForcePathStyle
	})
	return client, endpoint, nil
}

func decodeUploadContent(contentBase64 string) ([]byte, error) {
	value := strings.TrimSpace(contentBase64)
	if value == "" {
		return nil, fmt.Errorf("content_base64 is required")
	}
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		value = value[comma+1:]
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode content_base64: %w", err)
	}
	return data, nil
}

func cleanOSSObjectPath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	value = strings.Trim(value, "/")
	if value == "" || value == "." {
		return ""
	}

	parts := strings.Split(value, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(clean) > 0 {
				clean = clean[:len(clean)-1]
			}
			continue
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/")
}

func ossFolderPrefix(objectPath string) string {
	cleanPath := cleanOSSObjectPath(objectPath)
	if cleanPath == "" {
		return ""
	}
	return cleanPath + "/"
}

func objectPathJoin(parent string, name string) string {
	cleanParent := cleanOSSObjectPath(parent)
	cleanName := sanitizeObjectName(name)
	if cleanName == "" {
		return cleanParent
	}
	if cleanParent == "" {
		return cleanName
	}
	return pathpkg.Join(cleanParent, cleanName)
}

func ossFileView(cfg OSSConfig, endpoint string, parent string, key string, isDir bool, size int64, modTime *time.Time, contentType string) OSSFileView {
	cleanKey := cleanOSSObjectPath(key)
	name := ossFileName(parent, cleanKey)
	if name == "" {
		name = pathpkg.Base(cleanKey)
	}
	if contentType == "" && !isDir {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	}
	if contentType == "" {
		if isDir {
			contentType = "folder"
		} else {
			contentType = "application/octet-stream"
		}
	}

	ref := ""
	publicURL := ""
	if !isDir {
		ref = assetRef(cfg.ID, cleanKey)
		publicURL = publicOSSObjectURL(cfg, endpoint, cleanKey)
	}

	modTimeText := ""
	if modTime != nil && !modTime.IsZero() {
		modTimeText = modTime.Format(time.RFC3339)
	}
	return OSSFileView{
		ID:      cleanKey,
		IsDir:   isDir,
		ModTime: modTimeText,
		Name:    name,
		Path:    cleanKey,
		Ref:     ref,
		Size:    size,
		Type:    contentType,
		URL:     publicURL,
	}
}

func ossFileName(parent string, key string) string {
	cleanKey := cleanOSSObjectPath(key)
	cleanParent := cleanOSSObjectPath(parent)
	rel := cleanKey
	if cleanParent != "" && strings.HasPrefix(rel, cleanParent+"/") {
		rel = strings.TrimPrefix(rel, cleanParent+"/")
	}
	if index := strings.Index(rel, "/"); index >= 0 {
		rel = rel[:index]
	}
	return rel
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isTextPreview(ext string, contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(contentType, "text/") ||
		strings.Contains(contentType, "json") ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "xml") ||
		strings.Contains(contentType, "yaml") {
		return true
	}
	switch strings.ToLower(ext) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".css", ".scss", ".html", ".htm", ".json", ".md", ".markdown", ".txt", ".csv", ".xml", ".yaml", ".yml", ".toml", ".ini", ".log", ".sql", ".sh", ".zsh", ".bash":
		return true
	default:
		return false
	}
}

func normalizeOSSEndpoint(endpoint string, useSSL bool) string {
	value := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if useSSL {
		return "https://" + value
	}
	return "http://" + value
}

func assetRef(storageID string, key string) string {
	id := sanitizeStorageID(storageID)
	if id == "" {
		id = "default"
	}
	return "@assets/" + id + "/" + strings.TrimLeft(key, "/")
}

func objectKey(prefix string, name string) string {
	cleanPrefix := strings.Trim(pathpkg.Clean("/"+strings.TrimSpace(prefix)), "/")
	cleanName := sanitizeObjectName(name)
	if cleanName == "" {
		cleanName = "upload.bin"
	}
	fileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), cleanName)
	if cleanPrefix == "" || cleanPrefix == "." {
		return fileName
	}
	return pathpkg.Join(cleanPrefix, fileName)
}

func sanitizeObjectName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	base = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\:?*<>|"`, r) {
			return '-'
		}
		return r
	}, base)
	return strings.Trim(base, ". ")
}

func publicOSSObjectURL(cfg OSSConfig, endpoint string, key string) string {
	escapedKey := escapedObjectKey(key)
	if isLocalOSSConfig(cfg) {
		return localOSSAssetURL(cfg.ID, key)
	}
	if base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"); base != "" {
		return base + "/" + escapedKey
	}
	if cfg.ForcePathStyle {
		return strings.TrimRight(endpoint, "/") + "/" + url.PathEscape(strings.Trim(cfg.Bucket, "/")) + "/" + escapedKey
	}
	parsed, err := url.Parse(endpoint)
	if err == nil && parsed.Host != "" {
		parsed.Host = strings.Trim(cfg.Bucket, ".") + "." + parsed.Host
		parsed.Path = "/" + escapedKey
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String()
	}
	return strings.TrimRight(endpoint, "/") + "/" + escapedKey
}

func localOSSAssetURL(storageID string, key string) string {
	id := sanitizeStorageID(storageID)
	if id == "" {
		id = "default"
	}
	cleanKey := cleanOSSObjectPath(key)
	if cleanKey == "" {
		return ""
	}
	return "/api/oss/assets?storageId=" + url.QueryEscape(id) + "&path=" + url.QueryEscape(cleanKey)
}

func escapedObjectKey(key string) string {
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
