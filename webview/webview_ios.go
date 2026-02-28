//go:build ios

package webview

import (
	"fmt"

	"github.com/ltaoo/velo/webview/uikit"
)

var (
	webview_opts  *BoxWebviewOptions
	wkWebView     uikit.ID
	navController uikit.ID
	appWindow     uikit.ID
)

func open_webview(opts *BoxWebviewOptions) {
	fmt.Println("========================================")
	fmt.Println("DEBUG: Starting iOS Webview (open_webview)")
	fmt.Printf("DEBUG: Options - Title: %s, URL: %s\n", opts.Title, opts.URL)
	fmt.Println("========================================")

	// Store options for AppDelegate
	webview_opts = opts

	app := uikit.SharedApplication()
	fmt.Printf("DEBUG: SharedApplication returned: %v\n", app)

	if app != 0 {
		fmt.Println("========================================")
		fmt.Println("DEBUG: UIApplication already running, navigating existing webview")
		fmt.Printf("DEBUG: Current navController: %v\n", navController)
		fmt.Printf("DEBUG: Current wkWebView: %v\n", wkWebView)
		fmt.Println("========================================")
		uikit.DispatchAsyncMain(func() {
			fmt.Println("DEBUG: Inside DispatchAsyncMain, calling pushWindow")
			pushWindow(opts)
			fmt.Println("DEBUG: pushWindow completed")
		})
		// Do not block, return immediately to allow handler to complete
		fmt.Println("DEBUG: Returning from open_webview (app already running)")
		return
	} else {
		fmt.Println("DEBUG: UIApplication not running, starting new one")
		// Create AppDelegate class
		createAppDelegate()
		fmt.Println("DEBUG: VeloAppDelegate class registered")

		// Start UIApplication
		// This blocks until app exit
		fmt.Println("DEBUG: Calling UIApplicationMain...")
		// The third argument (principalClassName) can be nil, or "UIApplication"
		// The fourth argument is our delegate class name
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

	// Add window property methods (required for correct AppDelegate behavior)
	// @property (strong, nonatomic) UIWindow *window;
	uikit.AddMethod(cls, uikit.RegisterName("window"), getWindow, "@@:")
	uikit.AddMethod(cls, uikit.RegisterName("setWindow:"), setWindow, "v@:@")

	uikit.RegisterClassPair(cls)
}

func getWindow(self, _cmd uintptr) uintptr {
	return uintptr(appWindow)
}

func setWindow(self, _cmd, win uintptr) uintptr {
	appWindow = uikit.ID(win)
	return 0
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
	fmt.Println("========================================")
	fmt.Println("DEBUG: scriptMessageReceived called")

	// message is WKScriptMessage
	// body property contains the message body
	msgID := uikit.ID(message)
	body := msgID.Send(uikit.RegisterName("body"))

	// The body should be a string (JSON) because we send JSON.stringify(payload) from JS
	msgStr := uikit.NSStringToString(body)

	fmt.Printf("DEBUG: Received message from JS: %s\n", msgStr)
	fmt.Printf("DEBUG: webview_opts is nil: %v\n", webview_opts == nil)
	if webview_opts != nil {
		fmt.Printf("DEBUG: HandleMessage is nil: %v\n", webview_opts.HandleMessage == nil)
	}
	fmt.Println("========================================")

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
		fmt.Println("ERROR: webview_opts or HandleMessage is nil, cannot handle message")
	}
	return 0
}

func applicationDidFinishLaunching(self, _cmd, app, options uintptr) uintptr {
	fmt.Println("Velo: applicationDidFinishLaunching")
	
	// Create and setup the window
	initWindow()
	
	// Assign the window to the delegate
	// self.window = appWindow
	uikit.ID(self).Send(uikit.RegisterName("setWindow:"), appWindow)
	
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
	fmt.Println("========================================")
	fmt.Println("DEBUG: pushWindow called (pushing to navigation stack)")
	fmt.Printf("DEBUG: pushWindow - Title: %s, URL: %s\n", opts.Title, opts.URL)
	fmt.Printf("DEBUG: pushWindow - navController: %v\n", navController)
	fmt.Println("========================================")

	// Check if navigation controller exists
	if navController == 0 {
		fmt.Println("ERROR: navController is 0, cannot push")
		return
	}

	screen := uikit.GetClass("UIScreen").Send(uikit.RegisterName("mainScreen"))
	fmt.Printf("DEBUG: Got screen: %v\n", screen)

	rect := screen.SendGetRect(uikit.RegisterName("bounds"))
	fmt.Printf("DEBUG: Screen bounds - X: %.0f, Y: %.0f, W: %.0f, H: %.0f\n",
		rect.Origin.X, rect.Origin.Y, rect.Size.Width, rect.Size.Height)

	fmt.Printf("DEBUG: Creating new webview controller for URL: %s\n", opts.URL)
	vc, wv := createWebviewController(opts, rect)
	fmt.Printf("DEBUG: Created vc: %v, wv: %v\n", vc, wv)

	wkWebView = wv // Update current active webview
	fmt.Printf("DEBUG: Updated global wkWebView to: %v\n", wkWebView)

	// Push the new view controller onto the navigation stack
	// [navController pushViewController:vc animated:YES]
	fmt.Println("DEBUG: About to call pushViewController:animated:")
	navController.Send(uikit.RegisterName("pushViewController:animated:"), vc, true)
	fmt.Println("DEBUG: pushViewController:animated: completed")
	fmt.Println("========================================")
}

func initWindow() {
	// Create UIWindow
	// Get screen bounds
	screen := uikit.GetClass("UIScreen").Send(uikit.RegisterName("mainScreen"))
	rect := screen.SendGetRect(uikit.RegisterName("bounds"))

	win := uikit.GetClass("UIWindow").Send(uikit.RegisterName("alloc"))
	win = win.SendRect(uikit.RegisterName("initWithFrame:"), rect)
	appWindow = win

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
