package webview

import (
	"encoding/json"
	"net/http"
	"sync"
)

type Handler func(message string) (id string, result string)
type DragDropHandler func(event string, payload string)

type BoxWebviewOptions struct {
	URL            string
	Pathname       string
	IconData       []byte
	InjectedJS     string
	AppName        string
	Width          int
	Height         int
	Mux            *http.ServeMux
	HandleMessage  Handler
	HandleDragDrop DragDropHandler
}

type Webview struct{}

func OpenWebview(opts *BoxWebviewOptions) *Webview {
	open_webview(opts)
	return nil // open_webview blocks; this is reached after window closes
}

func (w *Webview) SetTitle(title string)        { setTitle(title) }
func (w *Webview) SetSize(width, height int)    { setSize(width, height) }
func (w *Webview) SetMinSize(width, height int) { setMinSize(width, height) }
func (w *Webview) SetMaxSize(width, height int) { setMaxSize(width, height) }
func (w *Webview) SetPosition(x, y int)         { setPosition(x, y) }
func (w *Webview) GetPosition() (int, int)      { return getPosition() }
func (w *Webview) GetSize() (int, int)          { return getSize() }
func (w *Webview) Show()                        { show() }
func (w *Webview) Hide()                        { hide() }
func (w *Webview) Minimize()                    { minimize() }
func (w *Webview) Maximize()                    { maximize() }
func (w *Webview) Fullscreen()                  { fullscreen() }
func (w *Webview) UnFullscreen()                { unFullscreen() }
func (w *Webview) Restore()                     { restore() }
func (w *Webview) SetAlwaysOnTop(onTop bool)    { setAlwaysOnTop(onTop) }
func (w *Webview) SetURL(url string)            { setURL(url) }
func (w *Webview) Close()                       { close_webview() }

func SendCallback(id, result string) {
	sendCallback(id, result)
}

var pendingMessages []string
var pendingMu sync.Mutex

func SendMessage(message interface{}) bool {
	payload, err := json.Marshal(message)
	if err != nil {
		return false
	}
	raw := string(payload)
	if sendMessage(raw) {
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
