package webview

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"sync"
)

type Engine string

const (
	EngineNative   Engine = "native"
	EngineElectron Engine = "electron"
)

func NormalizeEngine(engine Engine) Engine {
	switch engine {
	case EngineElectron:
		return EngineElectron
	default:
		return EngineNative
	}
}

type Handler func(message string) (id string, result string)
type DragDropHandler func(event string, payload string)
type ReopenHandler func()
type CloseHandler func(name string)

type BoxWebviewOptions struct {
	ID                     string
	Name                   string
	URL                    string
	Pathname               string
	IconData               []byte
	InjectedJS             string
	AppName                string
	Title                  string
	Width                  int
	Height                 int
	X                      int
	Y                      int
	HasPosition            bool
	Mux                    http.Handler
	FrontendFS             fs.FS
	HandleMessage          Handler
	HandleDragDrop         DragDropHandler
	HandleReopen           ReopenHandler
	HandleClose            CloseHandler
	QuitOnLastWindowClosed bool
	Engine                 Engine
	ElectronCommand        string
	RuntimeJSON            string
	Frameless              bool
	Hidden                 bool
	HideTrafficLights      bool
	NonActivating          bool
	PreserveStateOnFocus   bool
}

type backend interface {
	OpenWebview(opts *BoxWebviewOptions) *Webview
	OpenWindow(opts *BoxWebviewOptions) *Webview
	FocusWindow(opts *BoxWebviewOptions) bool
	SendCallback(id, result string)
	SendMessage(payload string) bool
	SetTitle(name, title string)
	SetSize(name string, width, height int)
	SetMinSize(name string, width, height int)
	SetMaxSize(name string, width, height int)
	SetPosition(name string, x, y int)
	GetPosition(name string) (int, int)
	GetSize(name string) (int, int)
	Show(name string)
	Hide(name string)
	Minimize(name string)
	Maximize(name string)
	Fullscreen(name string)
	UnFullscreen(name string)
	Restore(name string)
	SetAlwaysOnTop(name string, onTop bool)
	SetURL(name, url string)
	Close(name string)
}

type Webview struct {
	name    string
	engine  Engine
	backend backend
}

func NewHandle(name string, engine Engine) *Webview {
	return &Webview{
		name:    name,
		engine:  NormalizeEngine(engine),
		backend: backendForEngine(engine),
	}
}

type nativeBackend struct{}

func (nativeBackend) OpenWebview(opts *BoxWebviewOptions) *Webview {
	open_webview(opts)
	return nil // open_webview blocks; this is reached after window closes
}

func (nativeBackend) OpenWindow(opts *BoxWebviewOptions) *Webview {
	open_window(opts)
	return nil
}

func (nativeBackend) FocusWindow(opts *BoxWebviewOptions) bool  { return focus_window(opts) }
func (nativeBackend) SendCallback(id, result string)            { sendCallback(id, result) }
func (nativeBackend) SendMessage(payload string) bool           { return sendMessage(payload) }
func (nativeBackend) SetTitle(name, title string)               { setTitle(title) }
func (nativeBackend) SetSize(name string, width, height int)    { setSize(width, height) }
func (nativeBackend) SetMinSize(name string, width, height int) { setMinSize(width, height) }
func (nativeBackend) SetMaxSize(name string, width, height int) { setMaxSize(width, height) }
func (nativeBackend) SetPosition(name string, x, y int)         { setPosition(x, y) }
func (nativeBackend) GetPosition(name string) (int, int)        { return getPosition() }
func (nativeBackend) GetSize(name string) (int, int)            { return getSize() }
func (nativeBackend) Show(name string)                          { show() }
func (nativeBackend) Hide(name string)                          { hide() }
func (nativeBackend) Minimize(name string)                      { minimize() }
func (nativeBackend) Maximize(name string)                      { maximize() }
func (nativeBackend) Fullscreen(name string)                    { fullscreen() }
func (nativeBackend) UnFullscreen(name string)                  { unFullscreen() }
func (nativeBackend) Restore(name string)                       { restore() }
func (nativeBackend) SetAlwaysOnTop(name string, onTop bool)    { setAlwaysOnTop(onTop) }
func (nativeBackend) SetURL(name, url string)                   { setURL(url) }
func (nativeBackend) Close(name string)                         { close_webview() }

var (
	backendMu       sync.Mutex
	nativeWebview   backend = nativeBackend{}
	electronWebview backend = newElectronBackend()
	activeWebview   backend = nativeWebview
)

func backendForEngine(engine Engine) backend {
	switch NormalizeEngine(engine) {
	case EngineElectron:
		return electronWebview
	default:
		return nativeWebview
	}
}

func setActiveBackend(b backend) {
	if b == nil {
		return
	}
	backendMu.Lock()
	activeWebview = b
	backendMu.Unlock()
}

func currentBackend() backend {
	backendMu.Lock()
	defer backendMu.Unlock()
	return activeWebview
}

func (w *Webview) webviewBackend() backend {
	if w != nil && w.backend != nil {
		return w.backend
	}
	return currentBackend()
}

func (w *Webview) windowName() string {
	if w == nil || w.name == "" {
		return "default"
	}
	return w.name
}

func OpenWebview(opts *BoxWebviewOptions) *Webview {
	b := backendForEngine(opts.Engine)
	setActiveBackend(b)
	return b.OpenWebview(opts)
}

func OpenWindow(opts *BoxWebviewOptions) *Webview {
	b := backendForEngine(opts.Engine)
	setActiveBackend(b)
	if b.FocusWindow(opts) {
		return &Webview{name: opts.Name, engine: NormalizeEngine(opts.Engine), backend: b}
	}
	return b.OpenWindow(opts)
}

func (w *Webview) SetTitle(title string) {
	w.webviewBackend().SetTitle(w.windowName(), title)
}
func (w *Webview) SetSize(width, height int) {
	w.webviewBackend().SetSize(w.windowName(), width, height)
}
func (w *Webview) SetMinSize(width, height int) {
	w.webviewBackend().SetMinSize(w.windowName(), width, height)
}
func (w *Webview) SetMaxSize(width, height int) {
	w.webviewBackend().SetMaxSize(w.windowName(), width, height)
}
func (w *Webview) SetPosition(x, y int) {
	w.webviewBackend().SetPosition(w.windowName(), x, y)
}
func (w *Webview) GetPosition() (int, int) {
	return w.webviewBackend().GetPosition(w.windowName())
}
func (w *Webview) GetSize() (int, int) {
	return w.webviewBackend().GetSize(w.windowName())
}
func (w *Webview) Show()         { w.webviewBackend().Show(w.windowName()) }
func (w *Webview) Hide()         { w.webviewBackend().Hide(w.windowName()) }
func (w *Webview) Minimize()     { w.webviewBackend().Minimize(w.windowName()) }
func (w *Webview) Maximize()     { w.webviewBackend().Maximize(w.windowName()) }
func (w *Webview) Fullscreen()   { w.webviewBackend().Fullscreen(w.windowName()) }
func (w *Webview) UnFullscreen() { w.webviewBackend().UnFullscreen(w.windowName()) }
func (w *Webview) Restore()      { w.webviewBackend().Restore(w.windowName()) }
func (w *Webview) SetAlwaysOnTop(onTop bool) {
	w.webviewBackend().SetAlwaysOnTop(w.windowName(), onTop)
}
func (w *Webview) SetURL(url string) { w.webviewBackend().SetURL(w.windowName(), url) }
func (w *Webview) Close()            { w.webviewBackend().Close(w.windowName()) }

func SendCallback(id, result string) {
	currentBackend().SendCallback(id, result)
}

var pendingMessages []string
var pendingMu sync.Mutex

func SendMessage(message interface{}) bool {
	payload, err := json.Marshal(message)
	if err != nil {
		return false
	}
	raw := string(payload)
	if currentBackend().SendMessage(raw) {
		return true
	}
	pendingMu.Lock()
	pendingMessages = append(pendingMessages, raw)
	pendingMu.Unlock()
	return false
}

func notifyReady() {
	pendingMu.Lock()
	msgs := append([]string(nil), pendingMessages...)
	pendingMessages = nil
	pendingMu.Unlock()
	if len(msgs) == 0 {
		return
	}
	for _, msg := range msgs {
		if !sendMessage(msg) {
			pendingMu.Lock()
			pendingMessages = append([]string{msg}, pendingMessages...)
			pendingMu.Unlock()
			return
		}
	}
}
