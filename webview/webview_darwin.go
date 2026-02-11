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
	"unsafe"
)

var (
	webview_opts *BoxWebviewOptions
)

//export GoHandleSchemeTask
func GoHandleSchemeTask(webview unsafe.Pointer, task unsafe.Pointer, urlPtr *C.char) {
	if webview_opts == nil || webview_opts.Mux == nil {
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
		webview_opts.Mux.ServeHTTP(rw, req)
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
	if webview_opts == nil || webview_opts.HandleMessage == nil {
		return
	}
	goMsg := C.GoString(msg)
	id, result := webview_opts.HandleMessage(goMsg)
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
	if webview_opts == nil || webview_opts.HandleDragDrop == nil {
		return
	}
	goEvent := C.GoString(event)
	goPayload := C.GoString(payload)
	webview_opts.HandleDragDrop(goEvent, goPayload)
}

func open_webview(opts *BoxWebviewOptions) {
	webview_opts = opts
	runtime.LockOSThread()
	cUrl := C.CString(opts.URL)
	defer C.free(unsafe.Pointer(cUrl))
	cInjectedJS := C.CString(opts.InjectedJS)
	defer C.free(unsafe.Pointer(cInjectedJS))
	cAppName := C.CString(opts.AppName)
	defer C.free(unsafe.Pointer(cAppName))

	var cIcon unsafe.Pointer
	var cIconLen C.int
	if len(opts.IconData) > 0 {
		cIcon = C.CBytes(opts.IconData)
		defer C.free(cIcon)
		cIconLen = C.int(len(opts.IconData))
	}

	C.webviewRunApp(cUrl, cInjectedJS, cIcon, cIconLen, cAppName, C.int(opts.Width), C.int(opts.Height))
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
