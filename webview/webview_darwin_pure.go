//go:build darwin

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
	mapLock    sync.RWMutex

	quitOnLastWindowClosed = true
)

func init() {
	// Ensure we run on main thread for init
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
	cocoa.RegisterClassPair(appDelegateClass)
	fmt.Fprintln(os.Stderr, "DEBUG: VeloAppDelegate registered")
}

// Callback for applicationShouldTerminateAfterLastWindowClosed:
func applicationShouldTerminateAfterLastWindowClosed(self, _cmd, app uintptr) uintptr {
	if quitOnLastWindowClosed {
		return 1
	}
	return 0
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

	mapLock.RLock()
	opts := webviewMap[uintptr(webView)]
	mapLock.RUnlock()

	if opts == nil {
		opts = webview_opts
	}

	if opts == nil || opts.HandleMessage == nil {
		return
	}

	// Handle message
	id, result := opts.HandleMessage(str)
	if id != "" {
		sendCallbackTo(webView, id, result)
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

func open_window(opts *BoxWebviewOptions) {
	cocoa.DispatchMain(func() {
		createWindow(opts, false)
	})
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

	nsWindow := cocoa.GetClass("NSWindow").Send(cocoa.RegisterName("alloc")).SendRectStyle(
		cocoa.RegisterName("initWithContentRect:styleMask:backing:defer:"),
		rect,
		uintptr(styleMask),
		cocoa.NSBackingStoreBuffered,
		false, // defer
	)

	// Set Title
	nsWindow.Send(cocoa.RegisterName("setTitle:"), cocoa.StringToNSString(opts.Title))

	// Center window
	nsWindow.Send(cocoa.RegisterName("center"))

	// Make Key and Order Front
	nsWindow.Send(cocoa.RegisterName("makeKeyAndOrderFront:"), 0)

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

	// Create WKWebView
	wkWebView := cocoa.GetClass("WKWebView").Send(cocoa.RegisterName("alloc")).SendRect(
		cocoa.RegisterName("initWithFrame:configuration:"),
		rect,
		uintptr(config),
	)

	if isMain {
		globalWebView = wkWebView
	}

	// Register in map
	mapLock.Lock()
	webviewMap[uintptr(wkWebView)] = opts
	mapLock.Unlock()

	// Set as content view
	nsWindow.Send(cocoa.RegisterName("setContentView:"), wkWebView)

	// Load URL
	fmt.Fprintf(os.Stderr, "DEBUG: Loading URL: %s\n", opts.URL)
	nsURL := cocoa.GetClass("NSURL").Send(cocoa.RegisterName("URLWithString:"), cocoa.StringToNSString(opts.URL))
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
	fmt.Fprintln(os.Stderr, "DEBUG: URL request loaded (sent)")
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
			// Convert webview top-left coordinates to Cocoa top-left
			// Wait, setFrameTopLeftPoint takes Cocoa screen coordinates.
			// Cocoa screen coordinates have (0,0) at bottom-left.
			// To position at (x, y) where y is from top-left:
			// y_cocoa = ScreenHeight - y_webview

			screenHeight := getPrimaryScreenHeight()
			cy := screenHeight - y

			point := cocoa.CGPoint{
				X: cocoa.CGFloat(x),
				Y: cocoa.CGFloat(cy),
			}
			globalWindow.SendPoint(cocoa.RegisterName("setFrameTopLeftPoint:"), point)
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
			nsURL := cocoa.GetClass("NSURL").Send(cocoa.RegisterName("URLWithString:"), cocoa.StringToNSString(u))
			req := cocoa.GetClass("NSURLRequest").Send(cocoa.RegisterName("requestWithURL:"), nsURL)
			globalWebView.Send(cocoa.RegisterName("loadRequest:"), req)
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
	if globalWebView == 0 {
		return false
	}

	cocoa.DispatchMain(func() {
		script := fmt.Sprintf("window.__receiveGoMessage && window.__receiveGoMessage(%s);", payload)
		globalWebView.Send(cocoa.RegisterName("evaluateJavaScript:completionHandler:"), cocoa.StringToNSString(script), 0)
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
