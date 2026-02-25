//go:build darwin

package webview

/*
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#include <stdlib.h>
#include "webview_darwin.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"unsafe"
)

var (
	webview_opts *BoxWebviewOptions
	webviewMap   = make(map[uintptr]*BoxWebviewOptions)
	pendingOpts  = make(map[string]*BoxWebviewOptions)
	mapLock      sync.RWMutex
)

//export GoRegisterWebview
func GoRegisterWebview(webview unsafe.Pointer, id *C.char) {
	goID := C.GoString(id)
	mapLock.Lock()
	defer mapLock.Unlock()
	if opts, ok := pendingOpts[goID]; ok {
		webviewMap[uintptr(webview)] = opts
		delete(pendingOpts, goID)
	}
}

//export GoHandleSchemeTask
func GoHandleSchemeTask(webview unsafe.Pointer, task unsafe.Pointer, urlPtr *C.char) {
	mapLock.RLock()
	opts := webviewMap[uintptr(webview)]
	mapLock.RUnlock()
	if opts == nil {
		opts = webview_opts
	}
	if opts == nil || opts.Mux == nil {
		fmt.Printf("Webview: Scheme Error: Mux is nil\n")
		return
	}
	goUrl := C.GoString(urlPtr)

	go func() {
		// fmt.Printf("Webview: Scheme Start: %s\n", goUrl)

		u, err := url.Parse(goUrl)
		if err != nil {
			fmt.Printf("Webview: URL Parse Error: %v\n", err)
			return
		}
		u.Scheme = "http"

		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			fmt.Printf("Webview: NewRequest Error: %v\n", err)
			return
		}

		rw := &schemeResponseWriter{task: task}
		opts.Mux.ServeHTTP(rw, req)
		rw.Finish()
		// fmt.Printf("Webview: Scheme Finish: %s\n", goUrl)
	}()
}

type schemeResponseWriter struct {
	task        unsafe.Pointer
	header      http.Header
	wroteHeader bool
	finished    bool
}

func (s *schemeResponseWriter) Header() http.Header {
	if s.header == nil {
		s.header = make(http.Header)
	}
	return s.header
}

func (s *schemeResponseWriter) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if !s.wroteHeader {
		s.WriteHeader(http.StatusOK)
	}
	// fmt.Printf("Webview: Scheme Data: %v bytes\n", len(b))
	C.webviewSchemeTaskDidReceiveData(s.task, unsafe.Pointer(&b[0]), C.int(len(b)))
	return len(b), nil
}

func (s *schemeResponseWriter) WriteHeader(statusCode int) {
	if s.wroteHeader {
		return
	}
	s.wroteHeader = true
	contentType := s.Header().Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	headersJson, _ := json.Marshal(s.Header())
	// fmt.Printf("Webview: Scheme Response: %d, %s, headers: %s\n", statusCode, contentType, string(headersJson))

	cContentType := C.CString(contentType)
	defer C.free(unsafe.Pointer(cContentType))
	cHeaders := C.CString(string(headersJson))
	defer C.free(unsafe.Pointer(cHeaders))
	C.webviewSchemeTaskDidReceiveResponse(s.task, C.int(statusCode), cContentType, cHeaders)
}

func (s *schemeResponseWriter) Finish() {
	if s.finished {
		return
	}
	s.finished = true
	C.webviewSchemeTaskDidFinish(s.task)
}

var globalWebview unsafe.Pointer

func sendCallback(id, result string) {
	if globalWebview == nil {
		return
	}
	js := fmt.Sprintf(
		"window._goCallbacks && window._goCallbacks[%q] && window._goCallbacks[%q](%q);",
		id, id, result,
	)

	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	C.webviewEval(globalWebview, cjs)
}

func sendMessage(payload string) bool {
	if globalWebview == nil {
		return false
	}
	js := fmt.Sprintf("window.__receiveGoMessage && window.__receiveGoMessage(%s);", payload)
	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	C.webviewEval(globalWebview, cjs)
	return true
}

//export GoHandleMessage
func GoHandleMessage(webview unsafe.Pointer, msg *C.char) {
	globalWebview = webview
	notifyReady()
	mapLock.RLock()
	opts := webviewMap[uintptr(webview)]
	mapLock.RUnlock()
	if opts == nil {
		opts = webview_opts
	}
	if opts == nil || opts.HandleMessage == nil {
		return
	}
	goMsg := C.GoString(msg)
	id, result := opts.HandleMessage(goMsg)
	if id == "" {
		return
	}

	js := fmt.Sprintf(
		"window._goCallbacks && window._goCallbacks[%q] && window._goCallbacks[%q](%q);",
		id, id, result,
	)

	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	C.webviewEval(webview, cjs)
}

//export GoHandleDragDrop
func GoHandleDragDrop(webview unsafe.Pointer, event *C.char, payload *C.char) {
	mapLock.RLock()
	opts := webviewMap[uintptr(webview)]
	mapLock.RUnlock()
	if opts == nil {
		opts = webview_opts
	}
	if opts == nil || opts.HandleDragDrop == nil {
		return
	}
	goEvent := C.GoString(event)
	goPayload := C.GoString(payload)
	opts.HandleDragDrop(goEvent, goPayload)
}

func open_webview(opts *BoxWebviewOptions) {
	webview_opts = opts
	mapLock.Lock()
	pendingOpts[opts.ID] = opts
	mapLock.Unlock()

	runtime.LockOSThread()
	cID := C.CString(opts.ID)
	defer C.free(unsafe.Pointer(cID))
	cUrl := C.CString(opts.URL)
	defer C.free(unsafe.Pointer(cUrl))
	cInjectedJS := C.CString(opts.InjectedJS)
	defer C.free(unsafe.Pointer(cInjectedJS))
	cAppName := C.CString(opts.AppName)
	defer C.free(unsafe.Pointer(cAppName))
	cTitle := C.CString(opts.Title)
	defer C.free(unsafe.Pointer(cTitle))

	var cIcon unsafe.Pointer
	var cIconLen C.int
	if len(opts.IconData) > 0 {
		cIcon = C.CBytes(opts.IconData)
		defer C.free(cIcon)
		cIconLen = C.int(len(opts.IconData))
	}

	C.webviewRunApp(cID, cUrl, cInjectedJS, cIcon, cIconLen, cAppName, cTitle, C.int(opts.Width), C.int(opts.Height))
}

func open_window(opts *BoxWebviewOptions) {
	mapLock.Lock()
	pendingOpts[opts.ID] = opts
	mapLock.Unlock()

	cID := C.CString(opts.ID)
	defer C.free(unsafe.Pointer(cID))
	cUrl := C.CString(opts.URL)
	defer C.free(unsafe.Pointer(cUrl))
	cInjectedJS := C.CString(opts.InjectedJS)
	defer C.free(unsafe.Pointer(cInjectedJS))
	cAppName := C.CString(opts.AppName)
	defer C.free(unsafe.Pointer(cAppName))
	cTitle := C.CString(opts.Title)
	defer C.free(unsafe.Pointer(cTitle))

	C.webviewCreateWindow(cID, cUrl, cInjectedJS, cAppName, cTitle, C.int(opts.Width), C.int(opts.Height))
}

func Terminate() {
	C.webviewTerminate()
}

func setTitle(title string) {
	ct := C.CString(title)
	defer C.free(unsafe.Pointer(ct))
	C.webviewSetTitle(ct)
}

func setSize(width, height int) {
	C.webviewSetSize(C.int(width), C.int(height))
}

func setMinSize(width, height int) {
	C.webviewSetMinSize(C.int(width), C.int(height))
}

func setMaxSize(width, height int) {
	C.webviewSetMaxSize(C.int(width), C.int(height))
}

func setPosition(x, y int) {
	C.webviewSetPosition(C.int(x), C.int(y))
}

func getPosition() (int, int) {
	var x, y C.int
	C.webviewGetPosition(&x, &y)
	return int(x), int(y)
}

func getSize() (int, int) {
	var w, h C.int
	C.webviewGetSize(&w, &h)
	return int(w), int(h)
}

func show()         { C.webviewShow() }
func hide()         { C.webviewHide() }
func minimize()     { C.webviewMinimize() }
func maximize()     { C.webviewMaximize() }
func fullscreen()   { C.webviewFullscreen() }
func unFullscreen() { C.webviewUnFullscreen() }
func restore()      { C.webviewRestore() }

func setAlwaysOnTop(onTop bool) {
	v := C.int(0)
	if onTop {
		v = 1
	}
	C.webviewSetAlwaysOnTop(v)
}

func setURL(url string) {
	cu := C.CString(url)
	defer C.free(unsafe.Pointer(cu))
	C.webviewSetURL(cu)
}

func close_webview() {
	C.webviewClose()
}
