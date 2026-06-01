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
	} `json:"app"`
	Update buildcfg.UpdateSection `json:"update"`
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

type Box struct {
	get_handlers           map[string]Handler
	post_handlers          map[string]Handler
	webviews               []*webview.BoxWebviewOptions
	Webview                *webview.Webview
	Store                  *store.Store
	DB                     *gorm.DB
	mux                    *http.ServeMux
	mode                   Mode
	frontendDir            string
	appName                string
	title                  string
	iconData               []byte
	quitOnLastWindowClosed bool
}

type VeloAppOpt struct {
	Mode                   Mode
	AppName                string
	Title                  string
	IconData               []byte
	QuitOnLastWindowClosed *bool
}

func NewApp(o *VeloAppOpt) *Box {
	b := &Box{
		get_handlers:           make(map[string]Handler),
		post_handlers:          make(map[string]Handler),
		frontendDir:            "frontend",
		appName:                LoadAppConfig().displayName(),
		quitOnLastWindowClosed: true,
	}
	b.mode = o.Mode
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
	return webview.SendMessage(message)
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

	opts := &webview.BoxWebviewOptions{
		ID:                     id,
		Pathname:               pathname,
		IconData:               b.iconData,
		InjectedJS:             string(asset.JSRuntime),
		AppName:                b.appName,
		Title:                  title,
		Width:                  opt.Width,
		Height:                 opt.Height,
		Mux:                    mux,
		FrontendFS:             opt.FrontendFS,
		HandleMessage:          b.handleMessage,
		HandleDragDrop:         opt.OnDragDrop,
		QuitOnLastWindowClosed: b.quitOnLastWindowClosed,
	}
	if opt.URL != "" {
		opts.URL = opt.URL
	} else if b.mode == ModeBridgeHttp {
		opts.URL = "http://127.0.0.1:8080" + pathname
	} else {
		opts.URL = "velo://localhost" + pathname
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
		return c.Ok(H{
			"version": Version,
			"mode":    "development",
		})
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
	Name        string // window name used as storage key for position/size persistence
	Pathname    string
	Title       string
	Width       int
	Height      int
	Frameless   bool
	Hidden      bool
	FrontendDir string
	FrontendFS  fs.FS
	EntryPage   string
	OnDragDrop  func(event string, payload string)
	URL         string
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
	savedState := b.Store.GetWindow(windowName)
	if savedState != nil && savedState.Width > 0 && savedState.Height > 0 {
		width = savedState.Width
		height = savedState.Height
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
	opts := &webview.BoxWebviewOptions{
		ID:                     id,
		Pathname:               pathname,
		IconData:               b.iconData,
		InjectedJS:             string(asset.JSRuntime),
		AppName:                b.appName,
		Title:                  title,
		Width:                  width,
		Height:                 height,
		Mux:                    mux,
		FrontendFS:             opt.FrontendFS,
		HandleMessage:          b.handleMessage,
		HandleDragDrop:         opt.OnDragDrop,
		QuitOnLastWindowClosed: b.quitOnLastWindowClosed,
		Frameless:              opt.Frameless,
		Hidden:                 opt.Hidden,
	}
	b.webviews = append(b.webviews, opts)
	wv := &webview.Webview{}
	b.Webview = wv
	// Restore saved window position
	if savedState != nil && (savedState.X != 0 || savedState.Y != 0) {
		wv.SetPosition(savedState.X, savedState.Y)
	}
	return wv
}
