// Package velo is a lightweight framework for building desktop applications
// with web frontends. It provides native webview, system tray, file dialogs,
// and error dialogs across macOS, Windows, and Linux.
package velo

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"time"

	"github.com/ltaoo/velo/asset"
	"github.com/ltaoo/velo/buildcfg"
	"github.com/ltaoo/velo/database"
	"github.com/ltaoo/velo/frontendserver"
	"github.com/ltaoo/velo/store"
	"github.com/ltaoo/velo/webview"
	"gorm.io/gorm"
)

// Version is the velo framework version, injected at build time via -ldflags.
// In development the default value is "(dev)".
// When velo is used as a dependency, runtime/debug.ReadBuildInfo() provides the
// actual module version.
var Version = "(dev)"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/ltaoo/velo" {
				Version = dep.Version
				return
			}
		}
	}
}

// GetVersion returns the velo framework version string.
func GetVersion() string {
	return Version
}

type BoxContext struct {
	ctx     context.Context
	id      string
	method  string
	args    interface{}
	query   map[string]string
	headers interface{}
	Writer  http.ResponseWriter
}

type H map[string]interface{}

type BoxResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (c *BoxContext) Ok(data interface{}) string {
	r, _ := json.Marshal(BoxResult{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
	// if err != nil {
	// 	return fmt.Sprintf(`{"error": %q}`, err.Error())
	// }
	return string(r)
}

func (c *BoxContext) Error(message string) string {
	r, _ := json.Marshal(BoxResult{
		Code: 100,
		Msg:  message,
		Data: nil,
	})
	return string(r)
}

func (c *BoxContext) Query(key string) string {
	if c.query != nil {
		if v, ok := c.query[key]; ok && v != "" {
			return v
		}
	}
	if c.args != nil {
		if args, ok := c.args.(map[string]interface{}); ok {
			if v, ok := args[key]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return ""
}

func (c *BoxContext) SetQuery(query map[string]string) {
	c.query = query
}

func (c *BoxContext) GetHeader(key string) string {
	if c.headers == nil {
		return ""
	}
	// Try http.Header type assertion first (Go's http.Header is named type alias of map[string][]string)
	if headers, ok := c.headers.(http.Header); ok {
		if values, ok := headers[key]; ok && len(values) > 0 {
			return values[0]
		}
		return ""
	}
	if headers, ok := c.headers.(map[string][]string); ok {
		if values, ok := headers[key]; ok && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func (c *BoxContext) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *BoxContext) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *BoxContext) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *BoxContext) Err() error {
	return c.ctx.Err()
}

func (c *BoxContext) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

func (c *BoxContext) ID() string {
	return c.id
}

func (c *BoxContext) Method() string {
	return c.method
}

func (c *BoxContext) Args() interface{} {
	return c.args
}

func (c *BoxContext) BindJSON(obj interface{}) error {
	if c.args == nil {
		return fmt.Errorf("no data to bind")
	}
	bytes, err := json.Marshal(c.args)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, obj)
}

func (c *BoxContext) Context() context.Context {
	return c.ctx
}

type Handler func(c *BoxContext) interface{}

type AppConfig struct {
	App struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Author      string `json:"author"`
		Icon        string `json:"icon"`
		TrayIcon    string `json:"tray_icon"`
	} `json:"app"`
	Desktop buildcfg.DesktopSection `json:"desktop"`
	Update  buildcfg.UpdateSection  `json:"update"`
}

func LoadAppConfig(embedded ...[]byte) *AppConfig {
	var data []byte
	if len(embedded) > 0 && len(embedded[0]) > 0 {
		data = embedded[0]
	} else {
		var err error
		data, err = os.ReadFile("app-config.json")
		if err != nil {
			return &AppConfig{}
		}
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &AppConfig{}
	}
	return &cfg
}

func (c *AppConfig) displayName() string {
	var name string
	if c.App.DisplayName != "" {
		name = c.App.DisplayName
	} else if c.App.Name != "" {
		name = c.App.Name
	} else {
		name = "App"
	}

	return name
}

type Mode int

const (
	ModeBridge     Mode = iota // webview only, velo:// scheme, no HTTP server
	ModeBridgeHttp             // HTTP server + webview pointing to HTTP
	ModeHttp                   // HTTP server only, no webview
)

func (m Mode) String() string {
	switch m {
	case ModeBridge:
		return "ModeBridge"
	case ModeBridgeHttp:
		return "ModeBridgeHttp"
	case ModeHttp:
		return "ModeHttp"
	default:
		return fmt.Sprintf("Mode(%d)", m)
	}
}

type veloRuntimeAppConfig struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Icon        string `json:"icon"`
	TrayIcon    string `json:"tray_icon"`
}

type veloRuntimeUpdateSourceConfig struct {
	Type              string `json:"type"`
	Priority          int    `json:"priority"`
	Enabled           bool   `json:"enabled"`
	NeedCheckChecksum bool   `json:"need_check_checksum"`
}

type veloRuntimeUpdateConfig struct {
	Enabled        bool                            `json:"enabled"`
	CheckFrequency string                          `json:"check_frequency"`
	Channel        string                          `json:"channel"`
	AutoDownload   bool                            `json:"auto_download"`
	Timeout        int                             `json:"timeout"`
	Sources        []veloRuntimeUpdateSourceConfig `json:"sources"`
}

type veloRuntimeConfig struct {
	App    veloRuntimeAppConfig    `json:"app"`
	Update veloRuntimeUpdateConfig `json:"update"`
}

type veloRuntimeWindowInfo struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Pathname          string `json:"pathname"`
	URL               string `json:"url"`
	Title             string `json:"title"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	Frameless         bool   `json:"frameless"`
	Hidden            bool   `json:"hidden"`
	HideTrafficLights bool   `json:"hideTrafficLights"`
}

type veloRuntimeInfo struct {
	Version   string                 `json:"version"`
	Mode      string                 `json:"mode"`
	ModeValue int                    `json:"mode_value"`
	Engine    string                 `json:"engine"`
	AppName   string                 `json:"app_name"`
	Title     string                 `json:"title"`
	Config    veloRuntimeConfig      `json:"config"`
	Window    *veloRuntimeWindowInfo `json:"window"`
}

type Box struct {
	get_handlers           map[string]Handler
	post_handlers          map[string]Handler
	webviews               []*webview.BoxWebviewOptions
	Webview                *webview.Webview
	Store                  *store.Store
	DB                     *gorm.DB
	mux                    *http.ServeMux
	wsHub                  *veloWSHub
	mode                   Mode
	frontendDir            string
	appName                string
	title                  string
	iconData               []byte
	appConfig              *AppConfig
	quitOnLastWindowClosed bool
	webviewEngine          webview.Engine
}

type VeloAppOpt struct {
	Mode                   Mode
	WebviewEngine          webview.Engine
	AppName                string
	Title                  string
	IconData               []byte
	AppConfig              *AppConfig
	QuitOnLastWindowClosed *bool
}

func NewApp(o *VeloAppOpt) *Box {
	appConfig := LoadAppConfig()
	if o.AppConfig != nil {
		appConfig = o.AppConfig
	}
	b := &Box{
		get_handlers:           make(map[string]Handler),
		post_handlers:          make(map[string]Handler),
		wsHub:                  newVeloWSHub(),
		frontendDir:            "frontend",
		appName:                appConfig.displayName(),
		appConfig:              appConfig,
		quitOnLastWindowClosed: true,
		webviewEngine:          resolveWebviewEngine(appConfig, o.WebviewEngine),
	}
	b.mode = o.Mode
	if b.webviewEngine == webview.EngineElectron && b.mode == ModeBridge {
		fmt.Println("[velo] electron webview engine uses HTTP/WebSocket transport; switching ModeBridge to ModeBridgeHttp")
		b.mode = ModeBridgeHttp
	}
	if o.AppName != "" {
		b.appName = o.AppName
	}
	if o.Title != "" {
		b.title = o.Title
	}
	if o.IconData != nil {
		b.iconData = o.IconData
	}
	if o.QuitOnLastWindowClosed != nil {
		b.quitOnLastWindowClosed = *o.QuitOnLastWindowClosed
	}
	b.Store = store.New()
	b.registerStoreRoutes()
	b.registerVeloRoutes()
	return b
}

func resolveWebviewEngine(cfg *AppConfig, override webview.Engine) webview.Engine {
	if override != "" {
		return webview.NormalizeEngine(override)
	}
	if env := os.Getenv("VELO_WEBVIEW_ENGINE"); env != "" {
		return webview.NormalizeEngine(webview.Engine(env))
	}
	if cfg != nil && cfg.Desktop.Engine != "" {
		return webview.NormalizeEngine(webview.Engine(cfg.Desktop.Engine))
	}
	if cfg != nil && cfg.Desktop.Electron.Enabled {
		return webview.EngineElectron
	}
	return webview.EngineNative
}

func (c *AppConfig) runtimeConfig() veloRuntimeConfig {
	if c == nil {
		return veloRuntimeConfig{}
	}
	sources := make([]veloRuntimeUpdateSourceConfig, 0, len(c.Update.Sources))
	for _, source := range c.Update.Sources {
		sources = append(sources, veloRuntimeUpdateSourceConfig{
			Type:              source.Type,
			Priority:          source.Priority,
			Enabled:           source.Enabled,
			NeedCheckChecksum: source.NeedCheckChecksum,
		})
	}
	return veloRuntimeConfig{
		App: veloRuntimeAppConfig{
			Name:        c.App.Name,
			DisplayName: c.App.DisplayName,
			Description: c.App.Description,
			Version:     c.App.Version,
			Author:      c.App.Author,
			Icon:        c.App.Icon,
			TrayIcon:    c.App.TrayIcon,
		},
		Update: veloRuntimeUpdateConfig{
			Enabled:        c.Update.Enabled,
			CheckFrequency: c.Update.CheckFrequency,
			Channel:        c.Update.Channel,
			AutoDownload:   c.Update.AutoDownload,
			Timeout:        c.Update.Timeout,
			Sources:        sources,
		},
	}
}

func (b *Box) runtimeInfo(window *veloRuntimeWindowInfo) veloRuntimeInfo {
	title := b.title
	if title == "" {
		title = b.appName
	}
	return veloRuntimeInfo{
		Version:   Version,
		Mode:      b.mode.String(),
		ModeValue: int(b.mode),
		Engine:    string(b.webviewEngine),
		AppName:   b.appName,
		Title:     title,
		Config:    b.appConfig.runtimeConfig(),
		Window:    window,
	}
}

func (b *Box) injectedRuntimeJS(window *veloRuntimeWindowInfo) string {
	data := b.runtimeJSON(window)
	if data == "" {
		return string(asset.JSRuntime)
	}
	return fmt.Sprintf(`Object.defineProperty(window, "__VELO__", {
  value: %s,
  writable: false,
  configurable: false,
  enumerable: false
});
%s`, data, string(asset.JSRuntime))
}

func (b *Box) runtimeJSON(window *veloRuntimeWindowInfo) string {
	data, err := json.Marshal(b.runtimeInfo(window))
	if err != nil {
		return ""
	}
	return string(data)
}

func (b *Box) webviewURL(optURL, pathname string) string {
	if optURL != "" {
		return optURL
	}
	if b.mode == ModeBridgeHttp {
		return "http://127.0.0.1:8080" + pathname
	}
	return "velo://localhost" + pathname
}

// UseDatabase opens a database connection, runs migrations, and stores the
// resulting *gorm.DB on b.DB. Apps opt-in by calling this method.
func (b *Box) UseDatabase(cfg *database.DBConfig, migrations *embed.FS) error {
	db, err := database.NewDatabase(cfg)
	if err != nil {
		return err
	}
	if migrations != nil {
		m := database.NewMigrator(cfg, migrations)
		if err := m.MigrateUp(db); err != nil {
			return err
		}
	}
	b.DB = db
	return nil
}

func (b *Box) Get(name string, handler Handler) {
	b.get_handlers[name] = handler
}
func (b *Box) Post(name string, handler Handler) {
	b.post_handlers[name] = handler
}

func (b *Box) SendMessage(message interface{}) bool {
	delivered := false
	if b.mode != ModeHttp {
		delivered = webview.SendMessage(message)
	}
	if b.wsHub != nil && b.wsHub.BroadcastMessage(message) {
		delivered = true
	}
	return delivered
}

func (b *Box) OpenWindow(opt *VeloWebviewOpt) *webview.Webview {
	pathname := opt.Pathname
	if pathname == "" {
		pathname = "/"
	}

	mux := b.setupMux(opt.FrontendFS, opt.EntryPage)
	id := generateID()

	title := opt.Title
	if title == "" {
		title = b.title
	}
	if title == "" {
		title = b.appName
	}
	windowName := opt.Name
	if windowName == "" {
		windowName = "default"
	}
	width := opt.Width
	height := opt.Height
	var x, y int
	hasPosition := false
	savedState := b.Store.GetWindow(windowName)
	if savedState != nil {
		if savedState.Width > 0 && savedState.Height > 0 {
			width = savedState.Width
			height = savedState.Height
		}
		if savedState.X != 0 || savedState.Y != 0 {
			x = savedState.X
			y = savedState.Y
			hasPosition = true
		}
	}
	windowURL := b.webviewURL(opt.URL, pathname)
	windowInfo := &veloRuntimeWindowInfo{
		ID:                id,
		Name:              windowName,
		Pathname:          pathname,
		URL:               windowURL,
		Title:             title,
		Width:             width,
		Height:            height,
		Frameless:         opt.Frameless,
		Hidden:            opt.Hidden,
		HideTrafficLights: opt.HideTrafficLights,
	}

	opts := &webview.BoxWebviewOptions{
		ID:                     id,
		Name:                   windowName,
		Pathname:               pathname,
		IconData:               b.iconData,
		InjectedJS:             b.injectedRuntimeJS(windowInfo),
		RuntimeJSON:            b.runtimeJSON(windowInfo),
		AppName:                b.appName,
		Title:                  title,
		Width:                  width,
		Height:                 height,
		X:                      x,
		Y:                      y,
		HasPosition:            hasPosition,
		Mux:                    mux,
		FrontendFS:             opt.FrontendFS,
		HandleMessage:          b.handleMessage,
		HandleDragDrop:         opt.OnDragDrop,
		HandleReopen:           opt.OnReopen,
		QuitOnLastWindowClosed: b.quitOnLastWindowClosed,
		Engine:                 b.webviewEngine,
		ElectronCommand:        b.appConfig.Desktop.Electron.Command,
		Frameless:              opt.Frameless,
		Hidden:                 opt.Hidden,
		HideTrafficLights:      opt.HideTrafficLights,
		NonActivating:          opt.NonActivating,
		PreserveStateOnFocus:   opt.PreserveStateOnFocus,
		URL:                    windowURL,
	}
	return webview.OpenWindow(opts)
}

func (b *Box) handleMessage(message string) (string, string) {
	var msg struct {
		ID      string      `json:"id"`
		Method  string      `json:"method"`
		Headers interface{} `json:"headers"`
		Args    interface{} `json:"args"`
	}
	if err := json.Unmarshal([]byte(message), &msg); err != nil {
		fmt.Println("[box]handleMessage - unmarshal message failed", err)
		return "", ""
	}
	// Separate path and query string so that handlers registered by path
	// can be matched even when the frontend sends query parameters in the URL.
	path := msg.Method
	var queryParams map[string]string
	if u, err := url.Parse(msg.Method); err == nil {
		path = u.Path
		if len(u.Query()) > 0 {
			queryParams = make(map[string]string, len(u.Query()))
			for k, v := range u.Query() {
				if len(v) > 0 {
					queryParams[k] = v[0]
				}
			}
		}
	}
	fmt.Println("match methods in get handlers or post handlers", path)
	handler, exists := b.get_handlers[path]
	if !exists {
		if postHandler, ok := b.post_handlers[path]; ok {
			handler = postHandler
			exists = true
		}
	}
	ctx := &BoxContext{
		ctx:     context.Background(),
		id:      msg.ID,
		method:  path,
		headers: msg.Headers,
		args:    msg.Args,
		query:   queryParams,
	}
	if !exists {
		return msg.ID, fmt.Sprintf("%v", ctx.Error("unknown method"))
	}
	result := handler(ctx)
	return msg.ID, fmt.Sprintf("%v", result)
}

func (b *Box) registerStoreRoutes() {
	b.Get("/api/storage/get", func(c *BoxContext) interface{} {
		key := c.Query("key")
		if key == "" {
			return c.Ok(H{"data": b.Store.GetAll()})
		}
		v := b.Store.Get(key)
		if v == nil {
			return c.Ok(H{"found": false})
		}
		return c.Ok(H{"found": true, "value": json.RawMessage(v)})
	})
	b.Get("/api/storage/set", func(c *BoxContext) interface{} {
		key := c.Query("key")
		val := c.Query("value")
		if key == "" {
			return c.Error("key is required")
		}
		if err := b.Store.Set(key, json.RawMessage(val)); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(H{"success": true})
	})
	b.Get("/api/storage/delete", func(c *BoxContext) interface{} {
		key := c.Query("key")
		if key == "" {
			return c.Error("key is required")
		}
		if err := b.Store.Delete(key); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(H{"success": true})
	})
	b.Get("/api/window/state/snapshot", func(c *BoxContext) interface{} {
		name := c.Query("name")
		if name == "" {
			name = "default"
		}
		x, y := b.Webview.GetPosition()
		w, h := b.Webview.GetSize()
		if err := b.Store.SaveWindow(name, &store.WindowState{X: x, Y: y, Width: w, Height: h}); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(H{"success": true, "x": x, "y": y, "width": w, "height": h})
	})
	b.Get("/api/window/state/load", func(c *BoxContext) interface{} {
		name := c.Query("name")
		if name == "" {
			name = "default"
		}
		ws := b.Store.GetWindow(name)
		if ws == nil {
			return c.Ok(H{"found": false})
		}
		return c.Ok(H{"found": true, "x": ws.X, "y": ws.Y, "width": ws.Width, "height": ws.Height})
	})
}

func (b *Box) registerVeloRoutes() {
	b.Get("/api/velo/info", func(c *BoxContext) interface{} {
		return c.Ok(b.runtimeInfo(nil))
	})
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (box *Box) setupMux(frontendFS fs.FS, entryPage string) *http.ServeMux {
	mux := http.NewServeMux()

	if entryPage == "" {
		entryPage = "index.html"
	}

	if frontendFS != nil {
		mux.Handle("/", frontendserver.New(frontendserver.Options{
			Mode:      frontendserver.ModeProd,
			Root:      "frontend",
			Embedded:  frontendFS,
			EntryPage: entryPage,
		}))
	} else if box.mode == ModeBridgeHttp {
		mux.Handle("/", frontendserver.New(frontendserver.Options{
			Root:      box.frontendDir,
			EntryPage: entryPage,
		}))
	}

	if box.wsHub != nil {
		mux.HandleFunc(VeloWebSocketPath, func(w http.ResponseWriter, r *http.Request) {
			box.wsHub.ServeHTTP(w, r, box.handleMessage)
		})
	}
	mux.HandleFunc(VeloRuntimePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Write([]byte(box.injectedRuntimeJS(nil)))
	})

	for path, handler := range box.get_handlers {
		path, handler := path, handler
		fmt.Printf("[velo] registering GET %s\n", path)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[velo] handling GET %s (registered as %s)\n", r.URL.Path, path)
			query_params := make(map[string]string)
			for key, values := range r.URL.Query() {
				if len(values) > 0 {
					query_params[key] = values[0]
				}
			}

			ctx := &BoxContext{
				ctx:     r.Context(),
				id:      "",
				method:  path,
				args:    nil,
				query:   query_params,
				headers: r.Header,
				Writer:  w,
			}
			result := handler(ctx)
			if result != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(fmt.Sprintf("%v", result)))
			}
		})
	}

	for path, handler := range box.post_handlers {
		path, handler := path, handler
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			var args interface{}
			if r.Method == "POST" {
				json.NewDecoder(r.Body).Decode(&args)
			}
			query_params := make(map[string]string)
			for key, values := range r.URL.Query() {
				if len(values) > 0 {
					query_params[key] = values[0]
				}
			}

			ctx := &BoxContext{
				ctx:     r.Context(),
				id:      "",
				method:  path,
				args:    args,
				query:   query_params,
				headers: r.Header,
				Writer:  w,
			}
			result := handler(ctx)
			if result != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(fmt.Sprintf("%v", result)))
			}
		})
	}

	return mux
}

func (box *Box) Run() {
	fmt.Printf("[velo] version: %s\n", Version)
	if box.mode == ModeHttp {
		if len(box.webviews) > 0 {
			first := box.webviews[0]
			if mux, ok := first.Mux.(*http.ServeMux); ok {
				box.mux = mux
			} else {
				box.mux = box.setupMux(first.FrontendFS, "")
			}
		} else {
			box.mux = box.setupMux(nil, "")
		}
		server := &http.Server{Addr: "127.0.0.1:8080", Handler: box.mux}
		server.ListenAndServe()
		return
	}

	if len(box.webviews) > 0 && box.mode != ModeHttp {
		first := box.webviews[0]
		pathname := first.Pathname
		if box.mode == ModeBridgeHttp {
			if mux, ok := first.Mux.(*http.ServeMux); ok {
				box.mux = mux
			} else {
				// Fallback if Mux is generic handler
				box.mux = box.setupMux(first.FrontendFS, "")
			}
			go func() {
				server := &http.Server{Addr: "127.0.0.1:8080", Handler: box.mux}
				server.ListenAndServe()
			}()
			first.URL = "http://127.0.0.1:8080" + pathname
		} else {
			first.URL = "velo://localhost" + pathname
		}
		webview.OpenWebview(first)
	} else {
		box.mux = box.setupMux(nil, "")
		server := &http.Server{Addr: "127.0.0.1:8080", Handler: box.mux}
		server.ListenAndServe()
	}
}

type VeloWebviewOpt struct {
	Name                 string // window name used as storage key for position/size persistence
	Pathname             string
	Title                string
	Width                int
	Height               int
	Frameless            bool
	Hidden               bool
	HideTrafficLights    bool
	NonActivating        bool
	PreserveStateOnFocus bool
	FrontendDir          string
	FrontendFS           fs.FS
	EntryPage            string
	OnDragDrop           func(event string, payload string)
	OnReopen             func()
	URL                  string
}

func (b *Box) NewWebview(opt *VeloWebviewOpt) *webview.Webview {
	if opt.FrontendDir != "" {
		b.frontendDir = opt.FrontendDir
	}

	// Restore saved window size from storage
	windowName := opt.Name
	if windowName == "" {
		windowName = "default"
	}
	width := opt.Width
	height := opt.Height
	var x, y int
	hasPosition := false
	savedState := b.Store.GetWindow(windowName)
	if savedState != nil {
		if savedState.Width > 0 && savedState.Height > 0 {
			width = savedState.Width
			height = savedState.Height
		}
		if savedState.X != 0 || savedState.Y != 0 {
			x = savedState.X
			y = savedState.Y
			hasPosition = true
		}
	}

	mux := b.setupMux(opt.FrontendFS, opt.EntryPage)
	id := generateID()

	pathname := opt.Pathname
	if pathname == "" {
		pathname = "/"
	}
	title := opt.Title
	if title == "" {
		title = b.title
	}
	if title == "" {
		title = b.appName
	}
	windowURL := b.webviewURL(opt.URL, pathname)
	windowInfo := &veloRuntimeWindowInfo{
		ID:                id,
		Name:              windowName,
		Pathname:          pathname,
		URL:               windowURL,
		Title:             title,
		Width:             width,
		Height:            height,
		Frameless:         opt.Frameless,
		Hidden:            opt.Hidden,
		HideTrafficLights: opt.HideTrafficLights,
	}
	opts := &webview.BoxWebviewOptions{
		ID:                     id,
		Name:                   windowName,
		Pathname:               pathname,
		IconData:               b.iconData,
		InjectedJS:             b.injectedRuntimeJS(windowInfo),
		RuntimeJSON:            b.runtimeJSON(windowInfo),
		AppName:                b.appName,
		Title:                  title,
		Width:                  width,
		Height:                 height,
		X:                      x,
		Y:                      y,
		HasPosition:            hasPosition,
		Mux:                    mux,
		FrontendFS:             opt.FrontendFS,
		HandleMessage:          b.handleMessage,
		HandleDragDrop:         opt.OnDragDrop,
		HandleReopen:           opt.OnReopen,
		QuitOnLastWindowClosed: b.quitOnLastWindowClosed,
		Engine:                 b.webviewEngine,
		ElectronCommand:        b.appConfig.Desktop.Electron.Command,
		Frameless:              opt.Frameless,
		Hidden:                 opt.Hidden,
		HideTrafficLights:      opt.HideTrafficLights,
		NonActivating:          opt.NonActivating,
		PreserveStateOnFocus:   opt.PreserveStateOnFocus,
		URL:                    windowURL,
	}
	b.webviews = append(b.webviews, opts)
	wv := webview.NewHandle(windowName, b.webviewEngine)
	b.Webview = wv
	return wv
}
