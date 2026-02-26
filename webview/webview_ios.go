//go:build ios

package webview

import (
	"fmt"

	"github.com/ltaoo/velo/webview/uikit"
)

var (
	webview_opts *BoxWebviewOptions
	wkWebView    uikit.ID
	navController uikit.ID
)

func open_webview(opts *BoxWebviewOptions) {
	fmt.Println("DEBUG: Starting iOS Webview (open_webview)")
	// Store options for AppDelegate
	webview_opts = opts

	app := uikit.SharedApplication()
	fmt.Printf("DEBUG: SharedApplication returned: %v\n", app)

	if app != 0 {
		fmt.Println("DEBUG: UIApplication already running, navigating existing webview")
		uikit.DispatchAsyncMain(func() {
			pushWindow(opts)
		})
		// Do not block, return immediately to allow handler to complete
		return
	} else {
		fmt.Println("DEBUG: UIApplication not running, starting new one")
		// Create AppDelegate class
		createAppDelegate()
		fmt.Println("DEBUG: VeloAppDelegate class registered")

		// Start UIApplication
		// This blocks until app exit
		fmt.Println("DEBUG: Calling UIApplicationMain...")
		ret := uikit.UIApplicationMain(0, nil, "", "VeloAppDelegate")
		fmt.Printf("DEBUG: UIApplicationMain returned: %d\n", ret)
	}
}

func open_window(opts *BoxWebviewOptions) {
	// On iOS, the first window is the main app window.
	// Since Velo calls OpenWindow to start the UI, we must initialize the app loop here.
	// This call will block until the app exits.
	open_webview(opts)
}

func createAppDelegate() {
	super := uikit.GetClass("UIResponder")
	cls := uikit.AllocateClassPair(super, "VeloAppDelegate", 0)

	// Add application:didFinishLaunchingWithOptions:
	uikit.AddMethod(cls, uikit.RegisterName("application:didFinishLaunchingWithOptions:"), applicationDidFinishLaunching, "B@:@@")

	uikit.RegisterClassPair(cls)
}

func createScriptMessageHandler() {
	if uikit.GetClass("VeloScriptMessageHandler") != 0 {
		return
	}
	super := uikit.GetClass("NSObject")
	cls := uikit.AllocateClassPair(super, "VeloScriptMessageHandler", 0)

	// Add userContentController:didReceiveScriptMessage:
	uikit.AddMethod(cls, uikit.RegisterName("userContentController:didReceiveScriptMessage:"), scriptMessageReceived, "v@:@@")

	uikit.RegisterClassPair(cls)
}

func scriptMessageReceived(self, _cmd, controller, message uintptr) uintptr {
	// message is WKScriptMessage
	// body property contains the message body
	msgID := uikit.ID(message)
	body := msgID.Send(uikit.RegisterName("body"))

	// The body should be a string (JSON) because we send JSON.stringify(payload) from JS
	msgStr := uikit.NSStringToString(body)

	fmt.Printf("DEBUG: Received message from JS: %s\n", msgStr)

	if webview_opts != nil && webview_opts.HandleMessage != nil {
		go func() {
			fmt.Println("DEBUG: Handling message in goroutine")
			id, result := webview_opts.HandleMessage(msgStr)
			fmt.Printf("DEBUG: Message handled, id: %s, result: %s\n", id, result)
			if id != "" {
				uikit.DispatchAsyncMain(func() {
					fmt.Println("DEBUG: Sending callback to JS")
					sendCallback(id, result)
				})
			}
		}()
	} else {
		fmt.Println("DEBUG: webview_opts or HandleMessage is nil")
	}
	return 0
}

func applicationDidFinishLaunching(self, _cmd, app, options uintptr) uintptr {
	fmt.Println("Velo: applicationDidFinishLaunching")
	initWindow()
	return 1 // true
}

func createWebviewController(opts *BoxWebviewOptions, rect uikit.CGRect) (uikit.ID, uikit.ID) {
	// Create View Controller
	vc := uikit.GetClass("UIViewController").Send(uikit.RegisterName("alloc")).Send(uikit.RegisterName("init"))
	if opts.Title != "" {
		vc.Send(uikit.RegisterName("setTitle:"), uikit.NSString(opts.Title))
	}

	// Create WKWebView Configuration
	config := uikit.GetClass("WKWebViewConfiguration").Send(uikit.RegisterName("alloc")).Send(uikit.RegisterName("init"))

	// Create ScriptMessageHandler
	createScriptMessageHandler()
	handler := uikit.GetClass("VeloScriptMessageHandler").Send(uikit.RegisterName("alloc")).Send(uikit.RegisterName("init"))

	// Add ScriptMessageHandler
	ucc := config.Send(uikit.RegisterName("userContentController"))
	ucc.Send(uikit.RegisterName("addScriptMessageHandler:name:"), handler, uikit.NSString("go"))

	// Inject JS Runtime
	if opts != nil && opts.InjectedJS != "" {
		jsStr := uikit.NSString(opts.InjectedJS)
		// WKUserScriptInjectionTimeAtDocumentStart = 0
		// forMainFrameOnly: NO = 0 (or YES = 1)
		script := uikit.GetClass("WKUserScript").Send(uikit.RegisterName("alloc")).Send(uikit.RegisterName("initWithSource:injectionTime:forMainFrameOnly:"), jsStr, 0, 0)
		ucc.Send(uikit.RegisterName("addUserScript:"), script)
	}

	// Create WKWebView
	alloc := uikit.GetClass("WKWebView").Send(uikit.RegisterName("alloc"))

	// Use SendRectAndID for initWithFrame:configuration:
	wv := alloc.SendRectAndID(uikit.RegisterName("initWithFrame:configuration:"), rect, uikit.ID(config))

	// Load URL
	if opts != nil {
		if opts.URL != "" {
			nsURL := uikit.GetClass("NSURL").Send(uikit.RegisterName("URLWithString:"), uikit.NSString(opts.URL))
			req := uikit.GetClass("NSURLRequest").Send(uikit.RegisterName("requestWithURL:"), nsURL)
			wv.Send(uikit.RegisterName("loadRequest:"), req)
		}
	}

	// Add WebView to VC
	view := vc.Send(uikit.RegisterName("view"))
	view.Send(uikit.RegisterName("addSubview:"), wv)

	return vc, wv
}

func pushWindow(opts *BoxWebviewOptions) {
	fmt.Println("DEBUG: pushWindow called")
	if navController == 0 {
		fmt.Println("DEBUG: navController is 0, cannot push")
		return
	}
	
	screen := uikit.GetClass("UIScreen").Send(uikit.RegisterName("mainScreen"))
	rect := screen.SendGetRect(uikit.RegisterName("bounds"))
	
	fmt.Printf("DEBUG: Creating new webview controller for URL: %s\n", opts.URL)
	vc, wv := createWebviewController(opts, rect)
	wkWebView = wv // Update current active webview
	
	// Push VC
	// pushViewController:animated:
	fmt.Println("DEBUG: Pushing view controller to navigation stack")
	navController.Send(uikit.RegisterName("pushViewController:animated:"), vc, true)
}

func initWindow() {
	// Create UIWindow
	// Get screen bounds
	screen := uikit.GetClass("UIScreen").Send(uikit.RegisterName("mainScreen"))
	rect := screen.SendGetRect(uikit.RegisterName("bounds"))

	win := uikit.GetClass("UIWindow").Send(uikit.RegisterName("alloc"))
	win = win.SendRect(uikit.RegisterName("initWithFrame:"), rect)

	// Create Root View Controller (Navigation Controller)
	vc, wv := createWebviewController(webview_opts, rect)
	wkWebView = wv
	
	// Initialize UINavigationController with rootViewController
	navController = uikit.GetClass("UINavigationController").Send(uikit.RegisterName("alloc")).Send(uikit.RegisterName("initWithRootViewController:"), vc)
	
	// Set Root VC
	win.Send(uikit.RegisterName("setRootViewController:"), navController)

	// Make Key and Visible
	win.Send(uikit.RegisterName("makeKeyAndVisible"))
}

func setTitle(title string)        {}
func setSize(width, height int)    {}
func setMinSize(width, height int) {}
func setMaxSize(width, height int) {}
func setPosition(x, y int)         {}
func getPosition() (int, int)      { return 0, 0 }
func getSize() (int, int)          { return 0, 0 }
func show()                        {}
func hide()                        {}
func minimize()                    {}
func maximize()                    {}
func fullscreen()                  {}
func unFullscreen()                {}
func restore()                     {}
func setAlwaysOnTop(onTop bool)    {}
func setURL(url string) {
	if wkWebView != 0 {
		nsURL := uikit.GetClass("NSURL").Send(uikit.RegisterName("URLWithString:"), uikit.NSString(url))
		req := uikit.GetClass("NSURLRequest").Send(uikit.RegisterName("requestWithURL:"), nsURL)
		wkWebView.Send(uikit.RegisterName("loadRequest:"), req)
	}
}
func close_webview() {}
func sendCallback(id, result string) {
	if wkWebView == 0 {
		return
	}
	// window.invoke_cbs["id"](result)
	js := fmt.Sprintf("window.invoke_cbs[\"%s\"](%s)", id, result)
	fmt.Printf("DEBUG: Evaluating JS: %s\n", js)

	wkWebView.Send(uikit.RegisterName("evaluateJavaScript:completionHandler:"), uikit.NSString(js), 0)
}
func sendMessage(message string) bool { return false }
