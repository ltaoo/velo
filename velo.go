// Package velo is a lightweight framework for building desktop applications
// with web frontends. It provides native webview, system tray, file dialogs,
// and error dialogs across macOS, Windows, and Linux.
package velo

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/ltaoo/velo/asset"
	"github.com/ltaoo/velo/webview"
)

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
	if c.query == nil {
		return ""
	}
	return c.query[key]
}

func (c *BoxContext) SetQuery(query map[string]string) {
	c.query = query
}

func (c *BoxContext) GetHeader(key string) string {
	if c.headers == nil {
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
}

func loadAppConfig() string {
	data, err := os.ReadFile("app-config.json")
	if err != nil {
		return "App"
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "App"
	}
	if cfg.App.DisplayName != "" {
		return cfg.App.DisplayName
	}
	if cfg.App.Name != "" {
		return cfg.App.Name
	}
	return "App"
}

type Mode int

const (
	ModeBridge     Mode = iota // webview only, velo:// scheme, no HTTP server
	ModeBridgeHttp             // HTTP server + webview pointing to HTTP
	ModeHttp                   // HTTP server only, no webview
)

type Box struct {
	get_handlers  map[string]Handler
	post_handlers map[string]Handler
	webviews      []*webview.BoxWebviewOptions
	Webview       *webview.Webview
	mux           *http.ServeMux
	mode          Mode
	frontendDir   string
	frontendFS    fs.FS
	appName       string
	iconData      []byte
}

type VeloAppOpt struct {
	Mode        Mode
	FrontendDir string
	FrontendFS  fs.FS
	AppName     string
	IconData    []byte
}

func NewApp(o *VeloAppOpt) *Box {
	b := &Box{
		get_handlers:  make(map[string]Handler),
		post_handlers: make(map[string]Handler),
		frontendDir:   "frontend",
		frontendFS:    nil,
		appName:       loadAppConfig(),
	}
	b.mode = o.Mode
	if o.FrontendDir != "" {
		b.frontendDir = o.FrontendDir
	}
	if o.FrontendFS != nil {
		b.frontendFS = o.FrontendFS
	}
	if o.AppName != "" {
		b.appName = o.AppName
	}
	if o.IconData != nil {
		b.iconData = o.IconData
	}
	return b
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
	fmt.Println("match methods in get handlers or post handlers", msg.Method)
	handler, exists := b.get_handlers[msg.Method]
	if !exists {
		if postHandler, ok := b.post_handlers[msg.Method]; ok {
			handler = postHandler
			exists = true
		}
	}
	ctx := &BoxContext{
		ctx:     context.Background(),
		id:      msg.ID,
		method:  msg.Method,
		headers: msg.Headers,
		args:    msg.Args,
	}
	if !exists {
		return msg.ID, fmt.Sprintf("%v", ctx.Error("unknown method"))
	}
	result := handler(ctx)
	return msg.ID, fmt.Sprintf("%v", result)
}

func (box *Box) Run() {
	box.mux = http.NewServeMux()

	var fileServer http.Handler
	var indexBytes func() ([]byte, error)
	if box.mode == ModeBridgeHttp {
		fileServer = http.FileServer(http.Dir(box.frontendDir))
		indexBytes = func() ([]byte, error) {
			return os.ReadFile(filepath.Join(box.frontendDir, "index.html"))
		}
	} else {
		fs_frontend, _ := fs.Sub(box.frontendFS, "frontend")
		fileServer = http.FileServer(http.FS(fs_frontend))
		indexBytes = func() ([]byte, error) {
			return fs.ReadFile(fs_frontend, "index.html")
		}
	}
	box.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		fileServer.ServeHTTP(rec, r)
		if rec.Code == http.StatusNotFound {
			data, err := indexBytes()
			if err != nil {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		for k, v := range rec.Result().Header {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
	})

	for path, handler := range box.get_handlers {
		path, handler := path, handler
		fmt.Printf("[velo] registering GET %s\n", path)
		box.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
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
		box.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
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

	if len(box.webviews) > 0 && box.mode != ModeHttp {
		first := box.webviews[0]
		first.Mux = box.mux
		pathname := first.Pathname
		if box.mode == ModeBridgeHttp {
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
		server := &http.Server{Addr: "127.0.0.1:8080", Handler: box.mux}
		server.ListenAndServe()
	}
}

type VeloWebviewOpt struct {
	Pathname   string
	Width      int
	Height     int
	OnDragDrop func(event string, payload string)
}

func (b *Box) NewWebview(opt *VeloWebviewOpt) *webview.Webview {
	pathname := opt.Pathname
	if pathname == "" {
		pathname = "/"
	}
	opts := &webview.BoxWebviewOptions{
		Pathname:       pathname,
		IconData:       b.iconData,
		InjectedJS:     string(asset.JSRuntime),
		AppName:        b.appName,
		Width:          opt.Width,
		Height:         opt.Height,
		HandleMessage:  b.handleMessage,
		HandleDragDrop: opt.OnDragDrop,
	}
	b.webviews = append(b.webviews, opts)
	wv := &webview.Webview{}
	b.Webview = wv
	return wv
}
