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
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
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
	PathPrefix      string `json:"pathPrefix"`
	Provider        string `json:"provider"`
	PublicBaseURL   string `json:"publicBaseUrl"`
	Region          string `json:"region"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	UseSSL          bool   `json:"useSSL"`
}

type OSSUploadRequest struct {
	Config        OSSConfig `json:"config"`
	ContentBase64 string    `json:"content_base64"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
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
		if raw == nil {
			return c.Ok(velo.H{"found": false, "config": nil})
		}

		var cfg OSSConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"found": true, "config": cfg})
	})

	b.Post("/api/settings/cloud-storage/save", func(c *velo.BoxContext) interface{} {
		var cfg OSSConfig
		if err := c.BindJSON(&cfg); err != nil {
			return c.Error(err.Error())
		}

		raw, err := json.Marshal(cfg)
		if err != nil {
			return c.Error(err.Error())
		}
		if err := b.Store.Set(cloudStorageSettingsKey, json.RawMessage(raw)); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true, "config": cfg})
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
			cfg, err := loadStoredOSSConfig(b.Store.Get(cloudStorageSettingsKey))
			if err != nil {
				return c.Error(err.Error())
			}
			req.Config = cfg
		}

		result, err := uploadOSSObject(c.Context(), req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
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
		entryPage := "index.html"
		if pathname == "/settings" {
			entryPage = "settings.html"
		}
		b.OpenWindow(&velo.VeloWebviewOpt{
			Name:       "settings",
			Title:      "App-Settings",
			Pathname:   pathname,
			Width:      760,
			Height:     640,
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

	endpoint := normalizeOSSEndpoint(cfg.Endpoint, cfg.UseSSL)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

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
		"bucket": cfg.Bucket,
		"key":    key,
		"name":   req.Name,
		"size":   len(data),
		"type":   contentType,
		"url":    publicOSSObjectURL(cfg, endpoint, key),
	}, nil
}

func loadStoredOSSConfig(raw json.RawMessage) (OSSConfig, error) {
	if raw == nil {
		return OSSConfig{}, fmt.Errorf("cloud storage config is not saved")
	}

	var cfg OSSConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return OSSConfig{}, fmt.Errorf("read cloud storage config: %w", err)
	}
	return cfg, nil
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
	missing := make([]string, 0, 5)
	if strings.TrimSpace(cfg.Endpoint) == "" {
		missing = append(missing, "endpoint")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		missing = append(missing, "bucket")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		missing = append(missing, "access key id")
	}
	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		missing = append(missing, "secret access key")
	}
	if len(missing) > 0 {
		return fmt.Errorf("cloud storage config missing: %s", strings.Join(missing, ", "))
	}
	return nil
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
	if base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"); base != "" {
		return base + "/" + strings.TrimLeft(key, "/")
	}
	if cfg.ForcePathStyle {
		return strings.TrimRight(endpoint, "/") + "/" + strings.Trim(cfg.Bucket, "/") + "/" + strings.TrimLeft(key, "/")
	}
	return strings.TrimRight(endpoint, "/") + "/" + strings.TrimLeft(key, "/")
}
