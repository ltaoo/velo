//go:build darwin && !ios

package webview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ltaoo/velo/webview/cocoa"
)

var (
	webview_opts  *BoxWebviewOptions
	globalWindow  cocoa.ID
	globalWebView cocoa.ID

	// Map webview pointer to options for multi-window support
	webviewMap = make(map[uintptr]*BoxWebviewOptions)
	// Map webview pointer to its parent NSWindow for window operations (e.g. drag)
	nsWindowMap = make(map[uintptr]cocoa.ID)
	// Maps for named-window reuse. Keyed by BoxWebviewOptions.Name.
	namedWebViewMap   = make(map[string]cocoa.ID)
	webviewNameMap    = make(map[uintptr]string)
	windowWebViewMap  = make(map[uintptr]cocoa.ID)
	windowDelegateMap = make(map[uintptr]cocoa.ID)
	mapLock           sync.RWMutex

	quitOnLastWindowClosed = true
)

func init() {
	// Pin the main goroutine to the OS main thread for the entire app lifetime.
	// Cocoa requires that NSWindow and other UI operations happen on the main
	// thread. Without LockOSThread here, Go's runtime may reschedule the main
	// goroutine to a different thread after init (especially when packages like
	// GORM/SQLite create goroutines during their init), causing NSWindow creation
	// to crash with "NSWindow should only be instantiated on the main thread!".
	runtime.LockOSThread()

	fmt.Fprintln(os.Stderr, "DEBUG: webview_darwin_pure.go init()")

	// Register VeloSchemeHandler class
	handlerClass := cocoa.AllocateClassPair(cocoa.GetClass("NSObject"), "VeloSchemeHandler", 0)
	cocoa.AddMethod(handlerClass, cocoa.RegisterName("webView:startURLSchemeTask:"), startURLSchemeTask, "v@:@@")
	cocoa.AddMethod(handlerClass, cocoa.RegisterName("webView:stopURLSchemeTask:"), stopURLSchemeTask, "v@:@@")
	cocoa.RegisterClassPair(handlerClass)
	fmt.Fprintln(os.Stderr, "DEBUG: VeloSchemeHandler registered")

	// Register VeloScriptMessageHandler class
	scriptHandlerClass := cocoa.AllocateClassPair(cocoa.GetClass("NSObject"), "VeloScriptMessageHandler", 0)
	cocoa.AddMethod(scriptHandlerClass, cocoa.RegisterName("userContentController:didReceiveScriptMessage:"), didReceiveScriptMessage, "v@:@@")
	cocoa.RegisterClassPair(scriptHandlerClass)
	fmt.Fprintln(os.Stderr, "DEBUG: VeloScriptMessageHandler registered")

	// Register VeloAppDelegate class
	appDelegateClass := cocoa.AllocateClassPair(cocoa.GetClass("NSObject"), "VeloAppDelegate", 0)
	cocoa.AddMethod(appDelegateClass, cocoa.RegisterName("applicationShouldTerminateAfterLastWindowClosed:"), applicationShouldTerminateAfterLastWindowClosed, "B@:@")
	cocoa.AddMethod(appDelegateClass, cocoa.RegisterName("applicationShouldHandleReopen:hasVisibleWindows:"), applicationShouldHandleReopen, "B@:@B")
	cocoa.RegisterClassPair(appDelegateClass)
	fmt.Fprintln(os.Stderr, "DEBUG: VeloAppDelegate registered")

	// Register VeloWindowDelegate class for named-window cleanup on close and focus/blur events.
	windowDelegateClass := cocoa.AllocateClassPair(cocoa.GetClass("NSObject"), "VeloWindowDelegate", 0)
	cocoa.AddMethod(windowDelegateClass, cocoa.RegisterName("windowWillClose:"), windowWillClose, "v@:@")
	cocoa.AddMethod(windowDelegateClass, cocoa.RegisterName("windowDidBecomeKey:"), windowDidBecomeKey, "v@:@")
	cocoa.AddMethod(windowDelegateClass, cocoa.RegisterName("windowDidResignKey:"), windowDidResignKey, "v@:@")
	cocoa.RegisterClassPair(windowDelegateClass)
	fmt.Fprintln(os.Stderr, "DEBUG: VeloWindowDelegate registered")

	// Register VeloPanel class for transient launchers that need keyboard focus
	// without activating the whole application.
	panelClass := cocoa.AllocateClassPair(cocoa.GetClass("NSPanel"), "VeloPanel", 0)
	cocoa.AddMethod(panelClass, cocoa.RegisterName("canBecomeKeyWindow"), veloPanelCanBecomeKeyWindow, "B@:")
	cocoa.AddMethod(panelClass, cocoa.RegisterName("canBecomeMainWindow"), veloPanelCanBecomeMainWindow, "B@:")
	cocoa.RegisterClassPair(panelClass)
	fmt.Fprintln(os.Stderr, "DEBUG: VeloPanel registered")

	// Register VeloWebView class (subclass of WKWebView with drag-drop support)
	webViewClass := cocoa.AllocateClassPair(cocoa.GetClass("WKWebView"), "VeloWebView", 0)
	cocoa.AddMethod(webViewClass, cocoa.RegisterName("acceptsFirstMouse:"), veloWebViewAcceptsFirstMouse, "B@:@")
	cocoa.AddMethod(webViewClass, cocoa.RegisterName("draggingEntered:"), veloWebViewDraggingEntered, "Q@:@")
	cocoa.AddMethod(webViewClass, cocoa.RegisterName("draggingUpdated:"), veloWebViewDraggingUpdated, "Q@:@")
	cocoa.AddMethod(webViewClass, cocoa.RegisterName("draggingExited:"), veloWebViewDraggingExited, "v@:@")
	cocoa.AddMethod(webViewClass, cocoa.RegisterName("performDragOperation:"), veloWebViewPerformDragOperation, "B@:@")
	cocoa.RegisterClassPair(webViewClass)
	fmt.Fprintln(os.Stderr, "DEBUG: VeloWebView registered")
}

// Callback for applicationShouldTerminateAfterLastWindowClosed:
func applicationShouldTerminateAfterLastWindowClosed(self, _cmd, app uintptr) uintptr {
	if quitOnLastWindowClosed {
		return 1
	}
	return 0
}

func applicationShouldHandleReopen(self, _cmd, app, hasVisibleWindows uintptr) uintptr {
	if webview_opts != nil && webview_opts.HandleReopen != nil {
		go webview_opts.HandleReopen()
		return 1
	}
	if globalWindow != 0 {
		cocoa.DispatchMain(func() {
			nsApp := cocoa.GetClass("NSApplication").Send(cocoa.RegisterName("sharedApplication"))
			nsApp.Send(cocoa.RegisterName("activateIgnoringOtherApps:"), true)
			globalWindow.Send(cocoa.RegisterName("makeKeyAndOrderFront:"), 0)
		})
	}
	return 1
}

func windowWillClose(self, _cmd, notification uintptr) {
	nsWindow := cocoa.ID(notification).Send(cocoa.RegisterName("object"))
	cleanupWindow(nsWindow)
}

func windowDidBecomeKey(self, _cmd, notification uintptr) {
	nsWindow := cocoa.ID(notification).Send(cocoa.RegisterName("object"))
	emitWindowFocusEvent(nsWindow, true)
}

func windowDidResignKey(self, _cmd, notification uintptr) {
	nsWindow := cocoa.ID(notification).Send(cocoa.RegisterName("object"))
	emitWindowFocusEvent(nsWindow, false)
}

func emitWindowFocusEvent(nsWindow cocoa.ID, focused bool) {
	mapLock.RLock()
	wkWebView := windowWebViewMap[uintptr(nsWindow)]
	mapLock.RUnlock()

	if wkWebView == 0 {
		return
	}

	eventType := "__velo_window_blur"
	if focused {
		eventType = "__velo_window_focus"
	}

	sendWebViewMessage(wkWebView, map[string]interface{}{
		"type": eventType,
	})
}

func veloWebViewAcceptsFirstMouse(self, _cmd, event uintptr) uintptr {
	return 1
}

func veloPanelCanBecomeKeyWindow(self, _cmd uintptr) uintptr {
	return 1
}

func veloPanelCanBecomeMainWindow(self, _cmd uintptr) uintptr {
	return 1
}

// Callback for webView:startURLSchemeTask:
func startURLSchemeTask(self, _cmd, webView, task uintptr) {
	fmt.Fprintf(os.Stderr, "DEBUG: startURLSchemeTask called\n")
	// Get options for this webview
	mapLock.RLock()
	opts := webviewMap[webView]
	mapLock.RUnlock()

	if opts == nil {
		opts = webview_opts
	}

	if opts == nil || opts.Mux == nil {
		fmt.Fprintf(os.Stderr, "DEBUG: Mux is nil\n")
		return
	}

	// Create a task wrapper to use in goroutine
	taskID := cocoa.ID(task)

	// Get URL from task
	// task.request.URL.absoluteString
	req := taskID.Send(cocoa.RegisterName("request"))
	nsURL := req.Send(cocoa.RegisterName("URL"))
	nsString := nsURL.Send(cocoa.RegisterName("absoluteString"))
	urlStr := cocoa.NSStringToString(nsString)
	fmt.Fprintf(os.Stderr, "DEBUG: URL scheme request: %s\n", urlStr)

	go func() {
		u, err := url.Parse(urlStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG: Failed to parse URL: %v\n", err)
			return
		}

		// Map custom scheme to http for Mux
		u.Scheme = "http"

		fmt.Fprintf(os.Stderr, "DEBUG: Rewritten URL: %s\n", u.String())

		goReq, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG: Failed to create request: %v\n", err)
			return
		}

		rw := &schemeResponseWriter{
			task:   taskID,
			header: make(http.Header),
		}

		// Original Logic
		opts.Mux.ServeHTTP(rw, goReq)
		rw.Finish()
	}()
}

// Callback for webView:stopURLSchemeTask:
func stopURLSchemeTask(self, _cmd, webView, task uintptr) {
	// No-op
}

// Callback for userContentController:didReceiveScriptMessage:
func didReceiveScriptMessage(self, _cmd, userContentController, message uintptr) {
	msgID := cocoa.ID(message)
	body := msgID.Send(cocoa.RegisterName("body"))
	str := cocoa.NSStringToString(body)

	webView := msgID.Send(cocoa.RegisterName("webView"))

	// Check for built-in window drag message before routing to Go handler
	// This is done early to minimize latency for drag operations
	var parsed struct {
		ID     string      `json:"id"`
		Method string      `json:"method"`
		Args   interface{} `json:"args"`
	}
	if json.Unmarshal([]byte(str), &parsed) == nil && handleWindowControlMessage(webView, parsed.ID, parsed.Method, parsed.Args) {
		return
	}

	mapLock.RLock()
	opts := webviewMap[uintptr(webView)]
	mapLock.RUnlock()

	if opts == nil {
		opts = webview_opts
	}

	if opts == nil || opts.HandleMessage == nil {
		return
	}

	// Handle message in a goroutine to avoid blocking the main thread.
	// This prevents deadlocks when handlers need to run UI code on the main thread
	// (e.g. showing a native file dialog via performSelectorOnMainThread).
	wv := cocoa.ID(webView)
	go func() {
		id, result := opts.HandleMessage(str)
		if id != "" {
			cocoa.DispatchMain(func() {
				sendCallbackTo(wv, id, result)
			})
		}
	}()
}

func handleWindowControlMessage(webView cocoa.ID, id string, method string, args interface{}) bool {
	if !strings.HasPrefix(method, "__velo/window/") {
		return false
	}

	mapLock.RLock()
	nsWindow := nsWindowMap[uintptr(webView)]
	mapLock.RUnlock()

	if nsWindow == 0 {
		if id != "" {
			sendCallbackTo(webView, id, `{"success":false}`)
		}
		return true
	}

	switch method {
	case "__velo/window/start_drag":
		nsApp := cocoa.GetClass("NSApplication").Send(cocoa.RegisterName("sharedApplication"))
		currentEvent := nsApp.Send(cocoa.RegisterName("currentEvent"))
		if currentEvent != 0 {
			nsWindow.Send(cocoa.RegisterName("performWindowDragWithEvent:"), currentEvent)
		}
	case "__velo/window/close":
		if id != "" {
			sendCallbackTo(webView, id, `{"success":true}`)
		}
		nsWindow.Send(cocoa.RegisterName("performClose:"), 0)
		return true
	case "__velo/window/minimize":
		nsWindow.Send(cocoa.RegisterName("miniaturize:"), 0)
	case "__velo/window/hide":
		nsWindow.Send(cocoa.RegisterName("orderOut:"), 0)
	case "__velo/window/set_size":
		width := intArg(args, "width")
		height := intArg(args, "height")
		if width > 0 && height > 0 {
			setWindowFrameSizeKeepingTop(nsWindow, width, height)
		}
	case "__velo/window/state":
		if id != "" {
			sendCallbackTo(webView, id, windowStateResult(nsWindow))
		}
		return true
	case "__velo/window/toggle_maximize":
		nsWindow.Send(cocoa.RegisterName("zoom:"), 0)
	case "__velo/window/maximize":
		if nsWindow.Send(cocoa.RegisterName("isZoomed")) == 0 {
			nsWindow.Send(cocoa.RegisterName("zoom:"), 0)
		}
	case "__velo/window/restore":
		if nsWindow.Send(cocoa.RegisterName("isMiniaturized")) != 0 {
			nsWindow.Send(cocoa.RegisterName("deminiaturize:"), 0)
		}
		if nsWindow.Send(cocoa.RegisterName("isZoomed")) != 0 {
			nsWindow.Send(cocoa.RegisterName("zoom:"), 0)
		}
	case "__velo/window/set_always_on_top":
		level := cocoa.NSNormalWindowLevel
		if boolArg(args, "onTop") {
			level = cocoa.NSFloatingWindowLevel
		}
		nsWindow.Send(cocoa.RegisterName("setLevel:"), level)
	default:
		return false
	}

	if id != "" {
		sendCallbackTo(webView, id, `{"success":true}`)
	}
	return true
}

func windowStateResult(nsWindow cocoa.ID) string {
	if nsWindow == 0 {
		return `{"success":false}`
	}
	value := nsWindow.Send(cocoa.RegisterName("valueForKey:"), cocoa.StringToNSString("frame"))
	if value == 0 {
		return `{"success":false}`
	}
	var rect cocoa.CGRect
	value.Send(cocoa.RegisterName("getValue:"), unsafe.Pointer(&rect))
	screenHeight := getPrimaryScreenHeight()
	x := int(rect.X)
	y := screenHeight - int(rect.Y+rect.Height)
	width := int(rect.Width)
	height := int(rect.Height)
	return fmt.Sprintf(`{"success":true,"x":%d,"y":%d,"width":%d,"height":%d}`, x, y, width, height)
}

func intArg(args interface{}, key string) int {
	values, ok := args.(map[string]interface{})
	if !ok || values == nil {
		return 0
	}
	v, ok := values[key]
	if !ok {
		return 0
	}
	switch value := v.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return n
		}
	}
	return 0
}

func setWindowFrameSizeKeepingTop(nsWindow cocoa.ID, width int, height int) {
	if nsWindow == 0 || width <= 0 || height <= 0 {
		return
	}

	frame := cocoa.CGRect{
		X:      0,
		Y:      0,
		Width:  cocoa.CGFloat(width),
		Height: cocoa.CGFloat(height),
	}
	value := nsWindow.Send(cocoa.RegisterName("valueForKey:"), cocoa.StringToNSString("frame"))
	if value != 0 {
		var current cocoa.CGRect
		value.Send(cocoa.RegisterName("getValue:"), unsafe.Pointer(&current))
		frame.X = current.X
		frame.Y = current.Y + current.Height - cocoa.CGFloat(height)
	}

	nsWindow.SendRect(cocoa.RegisterName("setFrame:display:"), frame, 1)
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

type schemeResponseWriter struct {
	task   cocoa.ID
	header http.Header
	code   int
}

func (w *schemeResponseWriter) Header() http.Header {
	return w.header
}

func (w *schemeResponseWriter) Write(data []byte) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if len(data) > 0 {
		// Make a copy of data because DispatchMain is async and original data buffer might be reused
		// by the caller (http.ResponseWriter contract) after Write returns.
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		cocoa.DispatchMain(func() {
			fmt.Fprintf(os.Stderr, "DEBUG: schemeResponseWriter.Write: %d bytes\n", len(dataCopy))
			nsData := cocoa.BytesToNSData(dataCopy)
			w.task.Send(cocoa.RegisterName("didReceiveData:"), nsData)
		})
	}
	return len(data), nil
}

func (w *schemeResponseWriter) WriteHeader(statusCode int) {
	w.code = statusCode

	// Capture headers to avoid race conditions
	headers := make(map[string]string)
	for k, v := range w.header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	cocoa.DispatchMain(func() {
		req := w.task.Send(cocoa.RegisterName("request"))
		urlObj := req.Send(cocoa.RegisterName("URL"))

		// Build header dictionary
		headerDict := cocoa.GetClass("NSMutableDictionary").Send(cocoa.RegisterName("dictionary"))
		for k, v := range headers {
			kStr := cocoa.StringToNSString(k)
			vStr := cocoa.StringToNSString(v)
			headerDict.Send(cocoa.RegisterName("setValue:forKey:"), vStr, kStr)
		}

		// Create NSHTTPURLResponse
		response := cocoa.GetClass("NSHTTPURLResponse").Send(cocoa.RegisterName("alloc")).Send(
			cocoa.RegisterName("initWithURL:statusCode:HTTPVersion:headerFields:"),
			urlObj,
			statusCode,
			cocoa.StringToNSString("HTTP/1.1"),
			headerDict,
		)

		w.task.Send(cocoa.RegisterName("didReceiveResponse:"), response)
	})
}

func (w *schemeResponseWriter) Finish() {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	cocoa.DispatchMain(func() {
		fmt.Fprintln(os.Stderr, "DEBUG: schemeResponseWriter.Finish calling didFinish")
		w.task.Send(cocoa.RegisterName("didFinish"))
	})
}

// Main implementation for Darwin using purego
func open_webview(opts *BoxWebviewOptions) {
	fmt.Println("DEBUG: open_webview started")
	// Ensure we run on main thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Relaunch from temp .app bundle for proper Dock name
	if opts.AppName != "" && !isRunningInAppBundle() {
		if bundlePath := createTempAppBundle(opts.AppName, opts.IconData); bundlePath != "" {
			execPath := filepath.Join(bundlePath, "Contents", "MacOS", opts.AppName)
			cwd, _ := os.Getwd()
			os.Setenv("__WEBVIEW_ORIGINAL_CWD", cwd)
			argv := append([]string{execPath}, os.Args[1:]...)
			syscall.Exec(execPath, argv, os.Environ())
			// only reached if exec fails
		}
	}
	// Restore working directory if relaunched from bundle
	if cwd := os.Getenv("__WEBVIEW_ORIGINAL_CWD"); cwd != "" {
		os.Chdir(cwd)
		os.Unsetenv("__WEBVIEW_ORIGINAL_CWD")
	}

	webview_opts = opts
	quitOnLastWindowClosed = opts.QuitOnLastWindowClosed

	// Initialize NSApplication
	nsApp := cocoa.GetClass("NSApplication").Send(cocoa.RegisterName("sharedApplication"))
	fmt.Println("DEBUG: NSApplication initialized")

	// Set Application Delegate
	appDelegate := cocoa.GetClass("VeloAppDelegate").Send(cocoa.RegisterName("alloc")).Send(cocoa.RegisterName("init"))
	nsApp.Send(cocoa.RegisterName("setDelegate:"), appDelegate)

	// Set Process Name
	if opts.AppName != "" {
		pInfo := cocoa.GetClass("NSProcessInfo").Send(cocoa.RegisterName("processInfo"))
		pInfo.Send(cocoa.RegisterName("setProcessName:"), cocoa.StringToNSString(opts.AppName))
	}

	// Set activation policy to Regular
	nsApp.Send(cocoa.RegisterName("setActivationPolicy:"), cocoa.NSApplicationActivationPolicyRegular)
	installStandardApplicationMenu(nsApp, opts.AppName)
	nsApp.Send(cocoa.RegisterName("activateIgnoringOtherApps:"), true)

	// Set Application Icon if provided
	fmt.Printf("DEBUG: open_webview IconData length: %d\n", len(opts.IconData))
	if len(opts.IconData) > 0 {
		fmt.Println("DEBUG: Setting application icon")
		nsData := cocoa.BytesToNSData(opts.IconData)
		nsImage := cocoa.GetClass("NSImage").Send(cocoa.RegisterName("alloc")).Send(cocoa.RegisterName("initWithData:"), nsData)
		if nsImage == 0 {
			fmt.Println("ERROR: Failed to create NSImage from IconData")
		} else {
			nsApp.Send(cocoa.RegisterName("setApplicationIconImage:"), nsImage)
		}
	}

	fmt.Println("DEBUG: Creating window...")
	createWindow(opts, true)
	fmt.Println("DEBUG: Window created")

	// Run Application
	fmt.Println("DEBUG: Starting run loop...")
	nsApp.Send(cocoa.RegisterName("run"))
	fmt.Println("DEBUG: Run loop ended")
}

func installStandardApplicationMenu(nsApp cocoa.ID, appName string) {
	if appName == "" {
		appName = "App"
	}

	mainMenu := cocoa.GetClass("NSMenu").Send(cocoa.RegisterName("alloc")).Send(
		cocoa.RegisterName("initWithTitle:"),
		cocoa.StringToNSString(""),
	)

	appMenuItem := newMenuItem("", "", "")
	mainMenu.Send(cocoa.RegisterName("addItem:"), appMenuItem)
	appMenu := cocoa.GetClass("NSMenu").Send(cocoa.RegisterName("alloc")).Send(
		cocoa.RegisterName("initWithTitle:"),
		cocoa.StringToNSString(appName),
	)
	appMenu.Send(cocoa.RegisterName("addItem:"), newMenuItem("Quit "+appName, "terminate:", "q"))
	appMenuItem.Send(cocoa.RegisterName("setSubmenu:"), appMenu)

	editMenuItem := newMenuItem("", "", "")
	mainMenu.Send(cocoa.RegisterName("addItem:"), editMenuItem)
	editMenu := cocoa.GetClass("NSMenu").Send(cocoa.RegisterName("alloc")).Send(
		cocoa.RegisterName("initWithTitle:"),
		cocoa.StringToNSString("Edit"),
	)
	editMenu.Send(cocoa.RegisterName("addItem:"), newMenuItem("Undo", "undo:", "z"))
	editMenu.Send(cocoa.RegisterName("addItem:"), newMenuItem("Redo", "redo:", "Z"))
	editMenu.Send(cocoa.RegisterName("addItem:"), cocoa.GetClass("NSMenuItem").Send(cocoa.RegisterName("separatorItem")))
	editMenu.Send(cocoa.RegisterName("addItem:"), newMenuItem("Cut", "cut:", "x"))
	editMenu.Send(cocoa.RegisterName("addItem:"), newMenuItem("Copy", "copy:", "c"))
	editMenu.Send(cocoa.RegisterName("addItem:"), newMenuItem("Paste", "paste:", "v"))
	editMenu.Send(cocoa.RegisterName("addItem:"), newMenuItem("Select All", "selectAll:", "a"))
	editMenuItem.Send(cocoa.RegisterName("setSubmenu:"), editMenu)

	nsApp.Send(cocoa.RegisterName("setMainMenu:"), mainMenu)
}

func newMenuItem(title, action, keyEquivalent string) cocoa.ID {
	var selector cocoa.Selector
	if action != "" {
		selector = cocoa.RegisterName(action)
	}
	return cocoa.GetClass("NSMenuItem").Send(cocoa.RegisterName("alloc")).Send(
		cocoa.RegisterName("initWithTitle:action:keyEquivalent:"),
		cocoa.StringToNSString(title),
		selector,
		cocoa.StringToNSString(keyEquivalent),
	)
}

func open_window(opts *BoxWebviewOptions) {
	// If NSApp is not running (globalWindow is 0 as a proxy, though imperfect),
	// and we are calling OpenWindow, we should probably start the loop?
	// But usually OpenWindow implies "add a window".
	// The velo framework seems to imply OpenWindow starts the app.

	// If we are on macOS, and this is the first window, we should use open_webview
	// to ensure the run loop starts.
	mapLock.RLock()
	count := len(webviewMap)
	mapLock.RUnlock()

	if count == 0 {
		open_webview(opts)
	} else {
		isMain := false
		if webview_opts != nil && strings.TrimSpace(opts.Name) != "" && opts.Name == webview_opts.Name {
			isMain = true
		}
		cocoa.DispatchMain(func() {
			createWindow(opts, isMain)
		})
	}
}

func focus_window(opts *BoxWebviewOptions) bool {
	if opts == nil || strings.TrimSpace(opts.Name) == "" {
		return false
	}

	name := strings.TrimSpace(opts.Name)
	mapLock.Lock()
	wkWebView := namedWebViewMap[name]
	nsWindow := cocoa.ID(0)
	if wkWebView != 0 {
		nsWindow = nsWindowMap[uintptr(wkWebView)]
	}
	if wkWebView == 0 || nsWindow == 0 {
		if wkWebView != 0 {
			delete(namedWebViewMap, name)
			delete(webviewNameMap, uintptr(wkWebView))
			delete(webviewMap, uintptr(wkWebView))
			delete(nsWindowMap, uintptr(wkWebView))
		}
		mapLock.Unlock()
		return false
	}
	webviewMap[uintptr(wkWebView)] = opts
	mapLock.Unlock()

	cocoa.DispatchMain(func() {
		if opts.Title != "" {
			nsWindow.Send(cocoa.RegisterName("setTitle:"), cocoa.StringToNSString(opts.Title))
		}
		if opts.URL != "" && !opts.PreserveStateOnFocus {
			loadURLInWebView(wkWebView, opts.URL)
		}
		if nsWindow.Send(cocoa.RegisterName("isMiniaturized")) != 0 {
			nsWindow.Send(cocoa.RegisterName("deminiaturize:"), 0)
		}
		if opts.NonActivating {
			nsWindow.Send(cocoa.RegisterName("orderFrontRegardless"))
			nsWindow.Send(cocoa.RegisterName("makeKeyAndOrderFront:"), 0)
			return
		}
		nsApp := cocoa.GetClass("NSApplication").Send(cocoa.RegisterName("sharedApplication"))
		nsApp.Send(cocoa.RegisterName("activateIgnoringOtherApps:"), true)
		nsWindow.Send(cocoa.RegisterName("makeKeyAndOrderFront:"), 0)
	})
	return true
}

func cleanupWindow(nsWindow cocoa.ID) {
	if nsWindow == 0 {
		return
	}
	mapLock.Lock()
	wkWebView := windowWebViewMap[uintptr(nsWindow)]
	delete(windowWebViewMap, uintptr(nsWindow))
	delete(windowDelegateMap, uintptr(nsWindow))
	if wkWebView == 0 {
		mapLock.Unlock()
		return
	}

	opts := webviewMap[uintptr(wkWebView)]
	name := webviewNameMap[uintptr(wkWebView)]
	if name == "" && opts != nil {
		name = strings.TrimSpace(opts.Name)
	}
	delete(webviewMap, uintptr(wkWebView))
	delete(nsWindowMap, uintptr(wkWebView))
	if name := webviewNameMap[uintptr(wkWebView)]; name != "" {
		if namedWebViewMap[name] == wkWebView {
			delete(namedWebViewMap, name)
		}
		delete(webviewNameMap, uintptr(wkWebView))
	}
	if globalWindow == nsWindow {
		globalWindow = 0
	}
	if globalWebView == wkWebView {
		globalWebView = 0
	}
	mapLock.Unlock()

	if opts != nil && opts.HandleClose != nil {
		go opts.HandleClose(name)
	}
}

// NSDraggingDestination callbacks for VeloWebView

func hasDraggedFiles(sender uintptr) bool {
	senderID := cocoa.ID(sender)
	pasteboard := senderID.Send(cocoa.RegisterName("draggingPasteboard"))
	fileURLType := cocoa.StringToNSString("public.file-url")
	types := cocoa.GetClass("NSArray").Send(cocoa.RegisterName("arrayWithObject:"), fileURLType)
	available := pasteboard.Send(cocoa.RegisterName("availableTypeFromArray:"), types)
	return available != 0
}

func veloWebViewDraggingEntered(self, _cmd, sender uintptr) uintptr {
	if !hasDraggedFiles(sender) {
		return 0 // NSDragOperationNone
	}
	sendFileDragPointMessage(cocoa.ID(self), cocoa.ID(sender))
	return 1 // NSDragOperationCopy
}

func veloWebViewDraggingUpdated(self, _cmd, sender uintptr) uintptr {
	if !hasDraggedFiles(sender) {
		return 0
	}
	sendFileDragPointMessage(cocoa.ID(self), cocoa.ID(sender))
	return 1
}

func veloWebViewDraggingExited(self, _cmd, sender uintptr) {
	sendFileDragLeaveMessage(cocoa.ID(self))
}

func veloWebViewPerformDragOperation(self, _cmd, sender uintptr) uintptr {
	// Hide overlay immediately
	webView := cocoa.ID(self)
	sendFileDragLeaveMessage(webView)

	senderID := cocoa.ID(sender)
	point := dropPointInWebView(webView, senderID)
	pasteboard := senderID.Send(cocoa.RegisterName("draggingPasteboard"))

	// Read file URLs from pasteboard using readObjectsForClasses:options:
	nsURLClass := cocoa.GetClass("NSURL")
	classArray := cocoa.GetClass("NSArray").Send(cocoa.RegisterName("arrayWithObject:"), nsURLClass)
	options := cocoa.GetClass("NSDictionary").Send(cocoa.RegisterName("dictionary"))

	urls := pasteboard.Send(cocoa.RegisterName("readObjectsForClasses:options:"), classArray, options)
	if urls == 0 {
		return 0
	}

	count := uintptr(urls.Send(cocoa.RegisterName("count")))
	if count == 0 {
		return 0
	}

	var paths []string
	for i := uintptr(0); i < count; i++ {
		urlObj := urls.Send(cocoa.RegisterName("objectAtIndex:"), i)
		path := urlObj.Send(cocoa.RegisterName("path"))
		pathStr := cocoa.NSStringToString(path)
		if pathStr != "" {
			paths = append(paths, pathStr)
		}
	}

	if len(paths) == 0 {
		return 0
	}

	// Look up options for this webview
	mapLock.RLock()
	opts := webviewMap[self]
	mapLock.RUnlock()

	if opts == nil {
		opts = webview_opts
	}

	if opts != nil && opts.HandleDragDrop != nil {
		payload := map[string]interface{}{
			"paths": paths,
			"x":     point.X,
			"y":     point.Y,
		}
		payloadJSON, _ := json.Marshal(payload)
		go opts.HandleDragDrop("drop", string(payloadJSON))
	}

	return 1
}

func sendFileDragPointMessage(webView cocoa.ID, sender cocoa.ID) {
	point := dropPointInWebView(webView, sender)
	payload := map[string]interface{}{
		"type": "__velo_file_drag_over",
		"point": map[string]float64{
			"x": float64(point.X),
			"y": float64(point.Y),
		},
	}
	sendWebViewMessage(webView, payload)
}

func sendFileDragLeaveMessage(webView cocoa.ID) {
	sendWebViewMessage(webView, map[string]interface{}{"type": "__velo_drag_leave"})
}

func sendWebViewMessage(webView cocoa.ID, payload map[string]interface{}) {
	payloadJSON, _ := json.Marshal(payload)
	script := cocoa.StringToNSString(fmt.Sprintf(
		`window.__receiveGoMessage && window.__receiveGoMessage(%s);`,
		string(payloadJSON),
	))
	webView.Send(cocoa.RegisterName("evaluateJavaScript:completionHandler:"), script, 0)
}

func dropPointInWebView(webView cocoa.ID, sender cocoa.ID) cocoa.CGPoint {
	windowPoint := sender.SendPointReturn(cocoa.RegisterName("draggingLocation"))
	viewPoint := webView.SendPointIDReturn(cocoa.RegisterName("convertPoint:fromView:"), windowPoint, 0)

	boundsValue := webView.Send(cocoa.RegisterName("valueForKey:"), cocoa.StringToNSString("bounds"))
	if boundsValue == 0 {
		return viewPoint
	}

	var bounds cocoa.CGRect
	boundsValue.Send(cocoa.RegisterName("getValue:"), unsafe.Pointer(&bounds))
	return cocoa.CGPoint{
		X: viewPoint.X,
		Y: bounds.Height - viewPoint.Y,
	}
}

func createWindow(opts *BoxWebviewOptions, isMain bool) {
	// Create Window
	rect := cocoa.CGRect{
		X:      0,
		Y:      0,
		Width:  cocoa.CGFloat(opts.Width),
		Height: cocoa.CGFloat(opts.Height),
	}

	styleMask := cocoa.NSWindowStyleMaskTitled |
		cocoa.NSWindowStyleMaskClosable |
		cocoa.NSWindowStyleMaskMiniaturizable |
		cocoa.NSWindowStyleMaskResizable

	if opts.Frameless {
		styleMask |= cocoa.NSWindowStyleMaskFullSizeContentView
	}
	if opts.NonActivating {
		styleMask |= cocoa.NSWindowStyleMaskNonactivatingPanel
	}

	windowClass := cocoa.GetClass("NSWindow")
	if opts.NonActivating {
		windowClass = cocoa.GetClass("VeloPanel")
	}
	nsWindow := windowClass.Send(cocoa.RegisterName("alloc")).SendRectStyle(
		cocoa.RegisterName("initWithContentRect:styleMask:backing:defer:"),
		rect,
		uintptr(styleMask),
		cocoa.NSBackingStoreBuffered,
		false, // defer
	)
	if opts.NonActivating {
		nsWindow.Send(cocoa.RegisterName("setFloatingPanel:"), true)
		nsWindow.Send(cocoa.RegisterName("setHidesOnDeactivate:"), false)
		nsWindow.Send(cocoa.RegisterName("setBecomesKeyOnlyIfNeeded:"), false)
		nsWindow.Send(cocoa.RegisterName("setLevel:"), cocoa.NSFloatingWindowLevel)
	}

	if opts.Frameless {
		nsWindow.Send(cocoa.RegisterName("setTitlebarAppearsTransparent:"), true)
		nsWindow.Send(cocoa.RegisterName("setTitleVisibility:"), 1) // NSWindowTitleHidden

		// Use windowBackgroundColor for a native-looking light border instead of
		// clearColor, so the dark window shadow doesn't show through as a black edge.
		nsWindow.Send(cocoa.RegisterName("setBackgroundColor:"),
			cocoa.GetClass("NSColor").Send(cocoa.RegisterName("windowBackgroundColor")))
		nsWindow.Send(cocoa.RegisterName("setOpaque:"), true)
	}
	if opts.HideTrafficLights {
		hideTrafficLights(nsWindow)
	}

	// Set Title
	nsWindow.Send(cocoa.RegisterName("setTitle:"), cocoa.StringToNSString(opts.Title))
	windowDelegate := cocoa.GetClass("VeloWindowDelegate").Send(cocoa.RegisterName("alloc")).Send(cocoa.RegisterName("init"))
	nsWindow.Send(cocoa.RegisterName("setDelegate:"), windowDelegate)

	if opts.HasPosition {
		setWindowTopLeft(nsWindow, opts.X, opts.Y)
	} else {
		nsWindow.Send(cocoa.RegisterName("center"))
	}

	// Make Key and Order Front (unless hidden)
	if !opts.Hidden {
		if opts.NonActivating {
			nsWindow.Send(cocoa.RegisterName("orderFrontRegardless"))
		}
		nsWindow.Send(cocoa.RegisterName("makeKeyAndOrderFront:"), 0)
	}

	if isMain {
		globalWindow = nsWindow
	}

	// Create WKWebViewConfiguration
	config := cocoa.GetClass("WKWebViewConfiguration").Send(cocoa.RegisterName("alloc")).Send(cocoa.RegisterName("init"))

	// Fix permission errors by using non-persistent data store (or specific directory)
	// This resolves "Operation not permitted" errors in sandboxed environments
	dataStore := cocoa.GetClass("WKWebsiteDataStore").Send(cocoa.RegisterName("nonPersistentDataStore"))
	config.Send(cocoa.RegisterName("setWebsiteDataStore:"), dataStore)

	// Enable Developer Extras (Inspector)
	preferences := config.Send(cocoa.RegisterName("preferences"))
	preferences.Send(cocoa.RegisterName("setValue:forKey:"), cocoa.GetClass("NSNumber").Send(cocoa.RegisterName("numberWithBool:"), true), cocoa.StringToNSString("developerExtrasEnabled"))

	// Set URL Scheme Handler for "velo"
	handler := cocoa.GetClass("VeloSchemeHandler").Send(cocoa.RegisterName("alloc")).Send(cocoa.RegisterName("init"))
	config.Send(cocoa.RegisterName("setURLSchemeHandler:forURLScheme:"), handler, cocoa.StringToNSString("velo"))

	// Set Script Message Handler
	userContentController := config.Send(cocoa.RegisterName("userContentController"))
	scriptHandler := cocoa.GetClass("VeloScriptMessageHandler").Send(cocoa.RegisterName("alloc")).Send(cocoa.RegisterName("init"))
	if scriptHandler == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: Failed to allocate VeloScriptMessageHandler")
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG: VeloScriptMessageHandler allocated: %d\n", scriptHandler)
	}
	userContentController.Send(cocoa.RegisterName("addScriptMessageHandler:name:"), scriptHandler, cocoa.StringToNSString("go"))
	fmt.Fprintln(os.Stderr, "DEBUG: Script message handler 'go' added")

	// Inject setup script
	setupScript := `
		console.log("DEBUG: setupScript running");
		try {
			if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.go) {
				console.log("DEBUG: window.webkit.messageHandlers.go available");
			} else {
				console.log("DEBUG: window.webkit.messageHandlers.go MISSING");
				if (window.webkit) console.log("DEBUG: window.webkit exists");
				if (window.webkit && window.webkit.messageHandlers) console.log("DEBUG: messageHandlers exists: " + Object.keys(window.webkit.messageHandlers));
			}
		} catch (e) {
			console.log("DEBUG: Error checking messageHandlers: " + e);
		}
		window.external = {
			invoke: function(msg) {
				window.webkit.messageHandlers.go.postMessage(msg);
			}
		};
	`
	if opts.InjectedJS != "" {
		setupScript += "\n" + opts.InjectedJS
	}

	wkUserScript := cocoa.GetClass("WKUserScript").Send(cocoa.RegisterName("alloc")).Send(
		cocoa.RegisterName("initWithSource:injectionTime:forMainFrameOnly:"),
		cocoa.StringToNSString(setupScript),
		0, // WKUserScriptInjectionTimeAtDocumentStart
		false,
	)
	userContentController.Send(cocoa.RegisterName("addUserScript:"), wkUserScript)

	// Create VeloWebView (WKWebView subclass with drag-drop support)
	wkWebView := cocoa.GetClass("VeloWebView").Send(cocoa.RegisterName("alloc")).SendRect(
		cocoa.RegisterName("initWithFrame:configuration:"),
		rect,
		uintptr(config),
	)

	// Register for file drag-and-drop
	if opts.HandleDragDrop != nil {
		fileURLType := cocoa.StringToNSString("public.file-url")
		dragTypes := cocoa.GetClass("NSArray").Send(cocoa.RegisterName("arrayWithObject:"), fileURLType)
		wkWebView.Send(cocoa.RegisterName("registerForDraggedTypes:"), dragTypes)
	}

	if isMain {
		globalWebView = wkWebView
	}

	// Register in map
	mapLock.Lock()
	webviewMap[uintptr(wkWebView)] = opts
	nsWindowMap[uintptr(wkWebView)] = nsWindow
	windowWebViewMap[uintptr(nsWindow)] = wkWebView
	windowDelegateMap[uintptr(nsWindow)] = windowDelegate
	if name := strings.TrimSpace(opts.Name); name != "" {
		namedWebViewMap[name] = wkWebView
		webviewNameMap[uintptr(wkWebView)] = name
	}
	mapLock.Unlock()

	// Set as content view
	nsWindow.Send(cocoa.RegisterName("setContentView:"), wkWebView)

	// Transparent WKWebView background for frameless mode
	if opts.Frameless {
		wkWebView.Send(cocoa.RegisterName("setUnderPageBackgroundColor:"),
			cocoa.GetClass("NSColor").Send(cocoa.RegisterName("clearColor")))
	}

	// Load URL
	fmt.Fprintf(os.Stderr, "DEBUG: Loading URL: %s\n", opts.URL)
	loadURLInWebView(wkWebView, opts.URL)
	fmt.Fprintln(os.Stderr, "DEBUG: URL request loaded (sent)")
}

func hideTrafficLights(nsWindow cocoa.ID) {
	for _, button := range []int{0, 1, 2} {
		standardButton := nsWindow.Send(cocoa.RegisterName("standardWindowButton:"), button)
		if standardButton != 0 {
			standardButton.Send(cocoa.RegisterName("setHidden:"), true)
		}
	}
}

func setWindowTopLeft(nsWindow cocoa.ID, x, y int) {
	if nsWindow == 0 {
		return
	}
	screenHeight := getPrimaryScreenHeight()
	cy := screenHeight - y
	point := cocoa.CGPoint{
		X: cocoa.CGFloat(x),
		Y: cocoa.CGFloat(cy),
	}
	nsWindow.SendPoint(cocoa.RegisterName("setFrameTopLeftPoint:"), point)
}

func loadURLInWebView(wkWebView cocoa.ID, rawURL string) {
	if wkWebView == 0 || strings.TrimSpace(rawURL) == "" {
		return
	}
	nsURL := cocoa.GetClass("NSURL").Send(cocoa.RegisterName("URLWithString:"), cocoa.StringToNSString(rawURL))
	fmt.Fprintf(os.Stderr, "DEBUG: nsURL: %d\n", nsURL)
	if nsURL == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: nsURL is nil")
	}
	req := cocoa.GetClass("NSURLRequest").Send(cocoa.RegisterName("requestWithURL:"), nsURL)
	fmt.Fprintf(os.Stderr, "DEBUG: req: %d\n", req)
	if req == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: req is nil")
	}
	wkWebView.Send(cocoa.RegisterName("loadRequest:"), req)
}

func close_webview() {
	cocoa.DispatchMain(func() {
		nsApp := cocoa.GetClass("NSApplication").Send(cocoa.RegisterName("sharedApplication"))
		nsApp.Send(cocoa.RegisterName("terminate:"), 0)
	})
}

func setTitle(title string) {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			globalWindow.Send(cocoa.RegisterName("setTitle:"), cocoa.StringToNSString(title))
		}
	})
}

func setSize(width, height int) {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			// NSWindow setContentSize: takes NSSize (2 doubles)
			size := cocoa.CGSize{
				Width:  cocoa.CGFloat(width),
				Height: cocoa.CGFloat(height),
			}
			globalWindow.SendSize(cocoa.RegisterName("setContentSize:"), size)
		}
	})
}

func setMinSize(width, height int) {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			size := cocoa.CGSize{
				Width:  cocoa.CGFloat(width),
				Height: cocoa.CGFloat(height),
			}
			globalWindow.SendSize(cocoa.RegisterName("setMinSize:"), size)
		}
	})
}

func setMaxSize(width, height int) {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			size := cocoa.CGSize{
				Width:  cocoa.CGFloat(width),
				Height: cocoa.CGFloat(height),
			}
			globalWindow.SendSize(cocoa.RegisterName("setMaxSize:"), size)
		}
	})
}

func setPosition(x, y int) {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			setWindowTopLeft(globalWindow, x, y)
		}
	})
}

func getPosition() (int, int) {
	var x, y int
	wg := sync.WaitGroup{}
	wg.Add(1)

	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			value := globalWindow.Send(cocoa.RegisterName("valueForKey:"), cocoa.StringToNSString("frame"))
			if value != 0 {
				var rect cocoa.CGRect
				value.Send(cocoa.RegisterName("getValue:"), unsafe.Pointer(&rect))

				// Get screen height for coordinate conversion
				screenHeight := getPrimaryScreenHeight()

				x = int(rect.X)
				// Convert Cocoa bottom-left based coordinates to top-left based
				// rect.Y is bottom-left y
				// Top-left y in Cocoa is rect.Y + rect.Height
				// Top-left y in webview coordinates is ScreenHeight - (rect.Y + rect.Height)
				y = screenHeight - int(rect.Y+rect.Height)
			}
		}
		wg.Done()
	})

	wg.Wait()
	return x, y
}

func getPrimaryScreenHeight() int {
	screens := cocoa.GetClass("NSScreen").Send(cocoa.RegisterName("screens"))
	if screens != 0 {
		// Get primary screen (index 0)
		primary := screens.Send(cocoa.RegisterName("objectAtIndex:"), 0)
		if primary != 0 {
			value := primary.Send(cocoa.RegisterName("valueForKey:"), cocoa.StringToNSString("frame"))
			if value != 0 {
				var rect cocoa.CGRect
				value.Send(cocoa.RegisterName("getValue:"), unsafe.Pointer(&rect))
				return int(rect.Height)
			}
		}
	}
	return 0
}

func getSize() (int, int) {
	var w, h int
	wg := sync.WaitGroup{}
	wg.Add(1)

	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			value := globalWindow.Send(cocoa.RegisterName("valueForKey:"), cocoa.StringToNSString("frame"))
			if value != 0 {
				var rect cocoa.CGRect
				value.Send(cocoa.RegisterName("getValue:"), unsafe.Pointer(&rect))
				w = int(rect.Width)
				h = int(rect.Height)
			}
		}
		wg.Done()
	})

	wg.Wait()
	return w, h
}

func show() {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			nsApp := cocoa.GetClass("NSApplication").Send(cocoa.RegisterName("sharedApplication"))
			nsApp.Send(cocoa.RegisterName("activateIgnoringOtherApps:"), true)
			globalWindow.Send(cocoa.RegisterName("makeKeyAndOrderFront:"), 0)
		}
	})
}

func hide() {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			globalWindow.Send(cocoa.RegisterName("orderOut:"), 0)
		}
	})
}

func minimize() {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			globalWindow.Send(cocoa.RegisterName("miniaturize:"), 0)
		}
	})
}

func maximize() {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			if globalWindow.Send(cocoa.RegisterName("isZoomed")) == 0 {
				globalWindow.Send(cocoa.RegisterName("zoom:"), 0)
			}
		}
	})
}

func fullscreen() {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			styleMask := globalWindow.Send(cocoa.RegisterName("styleMask"))
			if styleMask&cocoa.NSWindowStyleMaskFullScreen == 0 {
				globalWindow.Send(cocoa.RegisterName("toggleFullScreen:"), 0)
			}
		}
	})
}

func unFullscreen() {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			styleMask := globalWindow.Send(cocoa.RegisterName("styleMask"))
			if styleMask&cocoa.NSWindowStyleMaskFullScreen != 0 {
				globalWindow.Send(cocoa.RegisterName("toggleFullScreen:"), 0)
			}
		}
	})
}

func restore() {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			if globalWindow.Send(cocoa.RegisterName("isMiniaturized")) != 0 {
				globalWindow.Send(cocoa.RegisterName("deminiaturize:"), 0)
			}
			if globalWindow.Send(cocoa.RegisterName("isZoomed")) != 0 {
				globalWindow.Send(cocoa.RegisterName("zoom:"), 0)
			}
		}
	})
}

func setAlwaysOnTop(on bool) {
	cocoa.DispatchMain(func() {
		if globalWindow != 0 {
			level := cocoa.NSNormalWindowLevel
			if on {
				level = cocoa.NSFloatingWindowLevel
			}
			globalWindow.Send(cocoa.RegisterName("setLevel:"), level)
		}
	})
}

func setURL(u string) {
	cocoa.DispatchMain(func() {
		if globalWebView != 0 {
			loadURLInWebView(globalWebView, u)
		}
	})
}

func sendCallback(id, result string) {
	cocoa.DispatchMain(func() {
		if globalWebView != 0 {
			sendCallbackTo(globalWebView, id, result)
		}
	})
}

func sendCallbackTo(webView cocoa.ID, id, result string) {
	jsonResult, _ := json.Marshal(result)

	script := fmt.Sprintf(
		"window._goCallbacks && window._goCallbacks[%q] && window._goCallbacks[%q](%s);",
		id, id, string(jsonResult),
	)
	webView.Send(cocoa.RegisterName("evaluateJavaScript:completionHandler:"), cocoa.StringToNSString(script), 0)
}

func sendMessage(payload string) bool {
	mapLock.RLock()
	targets := make([]cocoa.ID, 0, len(webviewMap))
	for wv := range webviewMap {
		targets = append(targets, cocoa.ID(wv))
	}
	mapLock.RUnlock()

	if len(targets) == 0 {
		return false
	}

	script := fmt.Sprintf("window.__receiveGoMessage && window.__receiveGoMessage(%s);", payload)
	cocoa.DispatchMain(func() {
		for _, wv := range targets {
			wv.Send(cocoa.RegisterName("evaluateJavaScript:completionHandler:"), cocoa.StringToNSString(script), 0)
		}
	})

	return true
}

func isRunningInAppBundle() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return strings.HasSuffix(filepath.Dir(resolved), ".app/Contents/MacOS") ||
		strings.Contains(resolved, ".app/Contents/MacOS/")
}

func createTempAppBundle(appName string, iconData []byte) string {
	bundlePath := filepath.Join(os.TempDir(), appName+".dev.app")
	macosPath := filepath.Join(bundlePath, "Contents", "MacOS")
	resourcesPath := filepath.Join(bundlePath, "Contents", "Resources")

	os.RemoveAll(bundlePath)
	if err := os.MkdirAll(macosPath, 0755); err != nil {
		return ""
	}
	if err := os.MkdirAll(resourcesPath, 0755); err != nil {
		return ""
	}

	// Symlink current executable
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if err := os.Symlink(exe, filepath.Join(macosPath, appName)); err != nil {
		return ""
	}

	// Write icon
	if len(iconData) > 0 {
		os.WriteFile(filepath.Join(resourcesPath, "appicon.png"), iconData, 0644)
	}

	// Write Info.plist
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>%s</string>
	<key>CFBundleDisplayName</key><string>%s</string>
	<key>CFBundleExecutable</key><string>%s</string>
	<key>CFBundleIdentifier</key><string>dev.%s</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleVersion</key><string>1.0</string>
	<key>CFBundleShortVersionString</key><string>1.0</string>
	<key>LSMinimumSystemVersion</key><string>10.13</string>
	<key>NSHighResolutionCapable</key><true/>
	<key>CFBundleIconFile</key><string>appicon</string>
</dict>
</plist>`, appName, appName, appName, appName)
	if err := os.WriteFile(filepath.Join(bundlePath, "Contents", "Info.plist"), []byte(plist), 0644); err != nil {
		return ""
	}

	return bundlePath
}
