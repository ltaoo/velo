//go:build windows

package webview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ltaoo/velo/internal/webview2loader"
)

/*
#cgo CXXFLAGS: -std=c++17
#cgo LDFLAGS: -lole32 -lshell32 -luser32 -lgdi32 -luuid -ldwmapi
#include <stdlib.h>
#include "webview_windows.h"
*/
import "C"

var webview_opts *BoxWebviewOptions
var globalWebview unsafe.Pointer

var windowsMu sync.RWMutex
var windowsByName = make(map[string]*BoxWebviewOptions)
var windowsByWebview = make(map[unsafe.Pointer]*BoxWebviewOptions)

var traceLogMu sync.Mutex
var traceLogFile *os.File

func traceLog(format string, args ...interface{}) {
	traceLogMu.Lock()
	defer traceLogMu.Unlock()
	if traceLogFile == nil {
		path := filepath.Join(os.TempDir(), "velo-webview-trace.log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		traceLogFile = f
		fmt.Fprintf(traceLogFile, "\n=== %s === trace started (pid=%d)\n",
			time.Now().Format(time.RFC3339), os.Getpid())
	}
	fmt.Fprintf(traceLogFile, "[%s] %s\n",
		time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
	traceLogFile.Sync()
}

type schemeResponseWriter struct {
	task           unsafe.Pointer
	header         http.Header
	wroteHeader    bool
	finished       bool
	statusLog      int
	bytesWritten   int
	contentTypeLog string
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
	C.webviewSchemeTaskDidReceiveData(s.task, unsafe.Pointer(&b[0]), C.int(len(b)))
	s.bytesWritten += len(b)
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
	s.statusLog = statusCode
	s.contentTypeLog = contentType
	// Serialize all headers except Content-Type (C++ side prepends it).
	// Format: "Name: Value\r\nName2: Value2" — CRLF separators required by
	// ICoreWebView2Environment::CreateWebResourceResponse.
	var headerLines []string
	if s.header != nil {
		for name, values := range s.header {
			if strings.EqualFold(name, "Content-Type") {
				continue
			}
			for _, v := range values {
				headerLines = append(headerLines, name+": "+v)
			}
		}
	}
	headersStr := strings.Join(headerLines, "\r\n")
	cContentType := C.CString(contentType)
	defer C.free(unsafe.Pointer(cContentType))
	cHeaders := C.CString(headersStr)
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

//export GoTrace
func GoTrace(msg *C.char) {
	if msg == nil {
		return
	}
	traceLog("[cpp] %s", C.GoString(msg))
}

func registerWindowsOptions(opts *BoxWebviewOptions) {
	if opts == nil {
		return
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "default"
	}
	windowsMu.Lock()
	windowsByName[name] = opts
	for webview, current := range windowsByWebview {
		currentName := "default"
		if current != nil && strings.TrimSpace(current.Name) != "" {
			currentName = strings.TrimSpace(current.Name)
		}
		if currentName == name {
			windowsByWebview[webview] = opts
		}
	}
	windowsMu.Unlock()
}

func windowsOptionsForWebview(webview unsafe.Pointer) *BoxWebviewOptions {
	windowsMu.RLock()
	opts := windowsByWebview[webview]
	windowsMu.RUnlock()
	if opts != nil {
		return opts
	}
	return webview_opts
}

//export GoRegisterWebview
func GoRegisterWebview(webview unsafe.Pointer, name *C.char) {
	windowName := strings.TrimSpace(C.GoString(name))
	windowsMu.Lock()
	opts := windowsByName[windowName]
	if opts != nil {
		windowsByWebview[webview] = opts
	}
	windowsMu.Unlock()
	if globalWebview == nil {
		globalWebview = webview
	}
}

//export GoHandleWindowClosed
func GoHandleWindowClosed(webview unsafe.Pointer, name *C.char) {
	windowName := strings.TrimSpace(C.GoString(name))
	windowsMu.Lock()
	opts := windowsByWebview[webview]
	if opts == nil {
		opts = windowsByName[windowName]
	}
	delete(windowsByWebview, webview)
	if windowsByName[windowName] == opts {
		delete(windowsByName, windowName)
	}
	windowsMu.Unlock()
	if opts != nil && opts.HandleClose != nil {
		go opts.HandleClose(windowName)
	}
}

//export GoHandleMessage
func GoHandleMessage(webview unsafe.Pointer, msg *C.char) {
	notifyReady()
	goMsg := C.GoString(msg)

	// Intercept built-in window drag message before routing to user handlers
	// (mirrors the Darwin implementation). JS from runtime.js posts this when
	// the user mousedowns on a .velo-drag / [data-velo-drag] element.
	var parsed struct {
		ID     string      `json:"id"`
		Method string      `json:"method"`
		Args   interface{} `json:"args"`
	}
	if json.Unmarshal([]byte(goMsg), &parsed) == nil && handleWindowControlMessage(webview, parsed.ID, parsed.Method, parsed.Args) {
		return
	}

	opts := windowsOptionsForWebview(webview)
	if opts == nil || opts.HandleMessage == nil {
		return
	}
	id, result := opts.HandleMessage(goMsg)
	if id == "" {
		return
	}
	sendCallbackTo(webview, id, result)
}

func sendCallback(id, result string) {
	if globalWebview == nil {
		return
	}
	sendCallbackTo(globalWebview, id, result)
}

func sendCallbackTo(webview unsafe.Pointer, id, result string) {
	if webview == nil {
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

func handleWindowControlMessage(webview unsafe.Pointer, id string, method string, args interface{}) bool {
	if !strings.HasPrefix(method, "__velo/window/") {
		return false
	}

	switch method {
	case "__velo/window/start_drag":
		C.webviewStartWindowDragFor(webview)
	case "__velo/window/close":
		if id != "" {
			sendCallbackTo(webview, id, `{"success":true}`)
		}
		C.webviewCloseWindow(webview)
		return true
	case "__velo/window/minimize":
		C.webviewMinimizeWindow(webview)
	case "__velo/window/hide":
		C.webviewHideWindow(webview)
	case "__velo/window/set_size":
		width := intArg(args, "width")
		height := intArg(args, "height")
		if width > 0 && height > 0 {
			C.webviewSetWindowSize(webview, C.int(width), C.int(height))
		}
	case "__velo/window/state":
		if id != "" {
			sendCallbackTo(webview, id, windowsWindowState(webview))
		}
		return true
	case "__velo/window/toggle_maximize":
		C.webviewMaximizeWindow(webview)
	case "__velo/window/maximize":
		C.webviewMaximizeWindow(webview)
	case "__velo/window/restore":
		C.webviewRestoreWindow(webview)
	case "__velo/window/set_always_on_top":
		onTop := C.int(0)
		if boolArg(args, "onTop") {
			onTop = 1
		}
		C.webviewSetWindowAlwaysOnTop(webview, onTop)
	default:
		return false
	}

	if id != "" {
		sendCallbackTo(webview, id, `{"success":true}`)
	}
	return true
}

func windowsWindowState(webview unsafe.Pointer) string {
	var x, y, width, height C.int
	C.webviewGetWindowState(webview, &x, &y, &width, &height)
	if width <= 0 || height <= 0 {
		return `{"success":false}`
	}
	return fmt.Sprintf(
		`{"success":true,"x":%d,"y":%d,"width":%d,"height":%d}`,
		int(x), int(y), int(width), int(height),
	)
}

func intArg(args interface{}, key string) int {
	values, ok := args.(map[string]interface{})
	if !ok || values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func boolArg(args interface{}, key string) bool {
	values, ok := args.(map[string]interface{})
	if !ok || values == nil {
		return false
	}
	v, ok := values[key]
	if !ok {
		return false
	}
	switch value := v.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true") || value == "1"
	case float64:
		return value != 0
	default:
		return false
	}
}

func sendMessage(payload string) bool {
	windowsMu.RLock()
	targets := make([]unsafe.Pointer, 0, len(windowsByWebview))
	for webview := range windowsByWebview {
		targets = append(targets, webview)
	}
	windowsMu.RUnlock()
	if len(targets) == 0 {
		return false
	}
	js := fmt.Sprintf("window.__receiveGoMessage && window.__receiveGoMessage(%s);", payload)
	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	for _, webview := range targets {
		C.webviewEval(webview, cjs)
	}
	return true
}

//export GoHandleSchemeTask
func GoHandleSchemeTask(webview unsafe.Pointer, task unsafe.Pointer, urlPtr *C.char) {
	opts := windowsOptionsForWebview(webview)
	if opts == nil || opts.Mux == nil {
		traceLog("GoHandleSchemeTask: no mux configured")
		return
	}
	goUrl := C.GoString(urlPtr)
	traceLog("GoHandleSchemeTask: url=%s", goUrl)
	go func() {
		u, err := url.Parse(goUrl)
		if err != nil {
			traceLog("url.Parse failed: %v", err)
			return
		}
		u.Scheme = "http"
		req, _ := http.NewRequest("GET", u.String(), nil)
		rw := &schemeResponseWriter{task: task}
		opts.Mux.ServeHTTP(rw, req)
		rw.Finish()
		traceLog("served %s -> status=%d, body=%d bytes, contentType=%q",
			goUrl, rw.statusLog, rw.bytesWritten, rw.contentTypeLog)
	}()
}

func open_webview(opts *BoxWebviewOptions) {
	webview_opts = opts
	registerWindowsOptions(opts)
	runtime.LockOSThread()
	loaderPath, loaderErr := webview2loader.Find(runtime.GOARCH)
	if loaderErr != nil {
		traceLog("WebView2 loader lookup failed: %v", loaderErr)
	}
	cLoaderPath := C.CString(loaderPath)
	defer C.free(unsafe.Pointer(cLoaderPath))
	cUrl := C.CString(opts.URL)
	defer C.free(unsafe.Pointer(cUrl))
	cInjectedJS := C.CString(opts.InjectedJS)
	defer C.free(unsafe.Pointer(cInjectedJS))
	var cIcon unsafe.Pointer
	var cIconLen C.int
	if len(opts.IconData) > 0 {
		cIcon = C.CBytes(opts.IconData)
		defer C.free(cIcon)
		cIconLen = C.int(len(opts.IconData))
	}
	cTitle := C.CString(opts.Title)
	defer C.free(unsafe.Pointer(cTitle))
	cName := C.CString(opts.Name)
	defer C.free(unsafe.Pointer(cName))
	frameless := C.int(0)
	if opts.Frameless {
		frameless = 1
	}
	hidden := C.int(0)
	if opts.Hidden {
		hidden = 1
	}
	C.webviewRunApp(cName, cUrl, cInjectedJS, cIcon, cIconLen, cTitle, C.int(opts.Width), C.int(opts.Height), frameless, hidden, cLoaderPath)
}

func open_window(opts *BoxWebviewOptions) {
	if opts == nil {
		return
	}
	registerWindowsOptions(opts)
	cName := C.CString(opts.Name)
	defer C.free(unsafe.Pointer(cName))
	cURL := C.CString(opts.URL)
	defer C.free(unsafe.Pointer(cURL))
	cInjectedJS := C.CString(opts.InjectedJS)
	defer C.free(unsafe.Pointer(cInjectedJS))
	cTitle := C.CString(opts.Title)
	defer C.free(unsafe.Pointer(cTitle))
	C.webviewOpenWindow(
		cName,
		cURL,
		cInjectedJS,
		cTitle,
		C.int(opts.Width),
		C.int(opts.Height),
		C.int(opts.X),
		C.int(opts.Y),
		boolCInt(opts.HasPosition),
		boolCInt(opts.Frameless),
		boolCInt(opts.Hidden),
		boolCInt(opts.NonActivating),
		boolCInt(opts.PreserveStateOnFocus),
	)
}

func boolCInt(value bool) C.int {
	if value {
		return 1
	}
	return 0
}

func focus_window(opts *BoxWebviewOptions) bool { return false }

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
