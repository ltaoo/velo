#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <libproc.h>
#import <crt_externs.h>
#import "webview_darwin.h"

// Global variable to track temp bundle path for cleanup
static NSString* gTempBundlePath = nil;

// Forward declaration for creating temp app bundle
static NSString* createTempAppBundle(NSString* appName, NSData* iconData);
static void cleanupTempAppBundle(NSString* bundlePath);
static BOOL isRunningInAppBundle(void);

extern void GoHandleMessage(void* webview, const char* msg);
extern void GoHandleSchemeTask(void* webview, void* task, const char* url);
extern void GoHandleDragDrop(void* webview, const char* event, const char* payload);

@interface DraggableWebView : WKWebView
@end

@implementation DraggableWebView
- (NSDragOperation)draggingEntered:(id<NSDraggingInfo>)sender {
    GoHandleDragDrop((__bridge void*)self, "enter", "");
    return NSDragOperationCopy;
}

- (NSDragOperation)draggingUpdated:(id<NSDraggingInfo>)sender {
    return NSDragOperationCopy;
}

- (void)draggingExited:(id<NSDraggingInfo>)sender {
    GoHandleDragDrop((__bridge void*)self, "leave", "");
}

- (BOOL)prepareForDragOperation:(id<NSDraggingInfo>)sender {
    return YES;
}

- (BOOL)performDragOperation:(id<NSDraggingInfo>)sender {
    NSPasteboard *pboard = [sender draggingPasteboard];
    if ([[pboard types] containsObject:NSPasteboardTypeFileURL]) {
        NSURL *fileURL = [NSURL URLFromPasteboard:pboard];
        if (fileURL) {
            GoHandleDragDrop((__bridge void*)self, "drop", [[fileURL path] UTF8String]);
            return YES;
        }
    }
    return NO;
}
@end

@interface AppSchemeHandler : NSObject <WKURLSchemeHandler>
@end

@implementation AppSchemeHandler
- (void)webView:(WKWebView *)webView startURLSchemeTask:(id<WKURLSchemeTask>)urlSchemeTask {
    const char* url = [[urlSchemeTask.request.URL absoluteString] UTF8String];
    GoHandleSchemeTask((__bridge void*)webView, (__bridge void*)urlSchemeTask, url);
}

- (void)webView:(WKWebView *)webView stopURLSchemeTask:(id<WKURLSchemeTask>)urlSchemeTask {
}
@end

@interface MiniDelegate : NSObject <NSApplicationDelegate, NSWindowDelegate, WKScriptMessageHandler>
@property(strong) NSWindow* window;
@property(strong) WKWebView* webview;
@property(strong) AppSchemeHandler* schemeHandler;
@property(copy) NSString* startURL;
@property(copy) NSString* injectedJS;
@property(copy) NSString* appName;
@property(assign) int windowWidth;
@property(assign) int windowHeight;
@end

@implementation MiniDelegate
- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    NSLog(@"Webview: applicationDidFinishLaunching");
    int winWidth = self.windowWidth > 0 ? self.windowWidth : 1024;
    int winHeight = self.windowHeight > 0 ? self.windowHeight : 768;
    NSRect frame = NSMakeRect(0, 0, winWidth, winHeight);
    self.window = [[NSWindow alloc] initWithContentRect:frame
                                              styleMask:(NSWindowStyleMaskTitled |
                                                         NSWindowStyleMaskClosable |
                                                         NSWindowStyleMaskMiniaturizable |
                                                         NSWindowStyleMaskResizable)
                                                backing:NSBackingStoreBuffered
                                                  defer:NO];
    [self.window center];
    [self.window setDelegate:self];  // Set window delegate to receive close notifications
    
    // Set window title from app name
    NSString* windowTitle = self.appName ? self.appName : @"App";
    [self.window setTitle:windowTitle];

    WKWebViewConfiguration* config = [[WKWebViewConfiguration alloc] init];
    WKUserContentController* controller = [[WKUserContentController alloc] init];
    [controller addScriptMessageHandler:self name:@"go"];
    if (self.injectedJS != nil && [self.injectedJS length] > 0) {
        WKUserScript* userScript = [[WKUserScript alloc] initWithSource:self.injectedJS
                                                          injectionTime:WKUserScriptInjectionTimeAtDocumentStart
                                                       forMainFrameOnly:NO];
        [controller addUserScript:userScript];
    }
    config.userContentController = controller;
    
    self.schemeHandler = [AppSchemeHandler new];
    [config setURLSchemeHandler:self.schemeHandler forURLScheme:@"velo"];

    [config.preferences setValue:@YES forKey:@"developerExtrasEnabled"];

    self.webview = [[DraggableWebView alloc] initWithFrame:frame configuration:config];
    [self.webview registerForDraggedTypes:@[NSPasteboardTypeFileURL]];
    if (@available(macOS 13.3, *)) {
        self.webview.inspectable = YES;
    }
    [self.window setContentView:self.webview];
    
    [self setupMenu];
    
    [self.window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    
    NSURL* url = [NSURL URLWithString:self.startURL];
    [self.webview loadRequest:[NSURLRequest requestWithURL:url]];
    NSLog(@"Webview: Loading Start URL: %@", self.startURL);
}

- (void)userContentController:(WKUserContentController *)userContentController
      didReceiveScriptMessage:(WKScriptMessage *)message {
    if (![message.name isEqualToString:@"go"]) {
        return;
    }
    NSString* m = (NSString*)message.body;
    const char* utf8 = [m UTF8String];
    GoHandleMessage((__bridge void*)self.webview, utf8);
}

- (void)setupMenu {
    NSMenu* mainMenu = [NSMenu new];
    [NSApp setMainMenu:mainMenu];

    // Application Menu
    NSMenuItem* appMenuItem = [NSMenuItem new];
    [mainMenu addItem:appMenuItem];
    NSMenu* appMenu = [NSMenu new];
    [appMenuItem setSubmenu:appMenu];
    [appMenu addItemWithTitle:@"Quit" action:@selector(terminate:) keyEquivalent:@"q"];

    // Edit Menu
    NSMenuItem* editMenuItem = [NSMenuItem new];
    [mainMenu addItem:editMenuItem];
    NSMenu* editMenu = [[NSMenu alloc] initWithTitle:@"Edit"];
    [editMenuItem setSubmenu:editMenu];
    [editMenu addItemWithTitle:@"Undo" action:@selector(undo:) keyEquivalent:@"z"];
    [editMenu addItemWithTitle:@"Redo" action:@selector(redo:) keyEquivalent:@"Z"];
    [editMenu addItem:[NSMenuItem separatorItem]];
    [editMenu addItemWithTitle:@"Cut" action:@selector(cut:) keyEquivalent:@"x"];
    [editMenu addItemWithTitle:@"Copy" action:@selector(copy:) keyEquivalent:@"c"];
    [editMenu addItemWithTitle:@"Paste" action:@selector(paste:) keyEquivalent:@"v"];
    [editMenu addItemWithTitle:@"Select All" action:@selector(selectAll:) keyEquivalent:@"a"];

    // View Menu
    NSMenuItem* viewMenuItem = [NSMenuItem new];
    [mainMenu addItem:viewMenuItem];
    NSMenu* viewMenu = [[NSMenu alloc] initWithTitle:@"View"];
    [viewMenuItem setSubmenu:viewMenu];
    [viewMenu addItemWithTitle:@"Reload" action:@selector(reload:) keyEquivalent:@"r"];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return YES;
}

- (void)windowWillClose:(NSNotification *)notification {
    NSLog(@"Webview: Window will close, terminating...");
    // Force exit the entire process including Go runtime
    // Use exit() on main thread to ensure it works
    dispatch_async(dispatch_get_main_queue(), ^{
        exit(0);
    });
}

- (void)applicationWillTerminate:(NSNotification *)notification {
    NSLog(@"Webview: Application will terminate");
    exit(0);
}

@end

// Global delegate for window control
static MiniDelegate* gDelegate = nil;

void webviewRunApp(const char* startURL, const char* injectedJS, const void* iconData, int iconLen, const char* appName, int width, int height) {
    NSLog(@"Webview: Starting webviewRunApp");
    @autoreleasepool {
        NSString* name = appName ? [NSString stringWithUTF8String:appName] : @"App";
        NSData* iconNSData = (iconData && iconLen > 0) ? [NSData dataWithBytes:iconData length:iconLen] : nil;
        
        int winWidth = width > 0 ? width : 1024;
        int winHeight = height > 0 ? height : 768;
        
        // Check if we were relaunched and need to restore working directory
        const char* originalCwd = getenv("__WEBVIEW_ORIGINAL_CWD");
        if (originalCwd) {
            chdir(originalCwd);
            NSLog(@"Webview: Restored working directory to: %s", originalCwd);
        }
        
        // Check if we need to relaunch inside a temp app bundle
        if (!isRunningInAppBundle()) {
            NSLog(@"Webview: Not running in app bundle, creating temp bundle...");
            NSString* bundlePath = createTempAppBundle(name, iconNSData);
            if (bundlePath) {
                gTempBundlePath = bundlePath;
                
                // Get the executable path inside the bundle
                NSString* execPath = [bundlePath stringByAppendingPathComponent:
                    [NSString stringWithFormat:@"Contents/MacOS/%@", name]];
                
                // Save current working directory
                NSString* cwd = [[NSFileManager defaultManager] currentDirectoryPath];
                setenv("__WEBVIEW_ORIGINAL_CWD", [cwd UTF8String], 1);
                
                NSLog(@"Webview: Relaunching from bundle: %@", execPath);
                NSLog(@"Webview: Original working directory: %@", cwd);
                
                // Use execv to replace current process (no fork)
                // This keeps the same terminal session
                int argc = *_NSGetArgc();
                char** oldArgv = *_NSGetArgv();
                char** newArgv = malloc((argc + 1) * sizeof(char*));
                newArgv[0] = strdup([execPath UTF8String]);
                for (int i = 1; i < argc; i++) {
                    newArgv[i] = oldArgv[i];
                }
                newArgv[argc] = NULL;
                
                execv([execPath UTF8String], newArgv);
                // Only reached if exec fails
                NSLog(@"Webview: execv failed, falling back to normal mode");
                free(newArgv[0]);
                free(newArgv);
            }
        }
        NSApplication* app = [NSApplication sharedApplication];
        [app setActivationPolicy:NSApplicationActivationPolicyRegular];
        
        if (iconNSData) {
             NSImage* icon = [[NSImage alloc] initWithData:iconNSData];
             if (icon) {
                 [app setApplicationIconImage:icon];
             }
        }

        MiniDelegate* delegate = [MiniDelegate new];
        gDelegate = delegate;
        delegate.startURL = [NSString stringWithUTF8String:startURL];
        if (injectedJS != NULL) {
            delegate.injectedJS = [NSString stringWithUTF8String:injectedJS];
        } else {
            delegate.injectedJS = @"";
        }
        delegate.appName = name;
        delegate.windowWidth = winWidth;
        delegate.windowHeight = winHeight;
        
        [app setDelegate:delegate];
        [app run];
        
        // Cleanup temp bundle on exit
        if (gTempBundlePath) {
            cleanupTempAppBundle(gTempBundlePath);
        }
    }
}

void webviewTerminate() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}

void webviewEval(void* webview, const char* js) {
    WKWebView* wv = (__bridge WKWebView*)webview;
    NSString* code = [NSString stringWithUTF8String:js];
    dispatch_async(dispatch_get_main_queue(), ^{
        [wv evaluateJavaScript:code completionHandler:nil];
    });
}

void webviewSchemeTaskDidReceiveResponse(void* task, int status, const char* contentType, const char* headersJson) {
    id<WKURLSchemeTask> t = (__bridge id<WKURLSchemeTask>)task;
    NSString* ct = [NSString stringWithUTF8String:contentType];
    NSString* hj = headersJson ? [NSString stringWithUTF8String:headersJson] : nil;
    
    dispatch_sync(dispatch_get_main_queue(), ^{
        NSMutableDictionary* headers = [NSMutableDictionary dictionary];
        headers[@"Content-Type"] = ct;
        
        if (hj) {
            NSData* jsonData = [hj dataUsingEncoding:NSUTF8StringEncoding];
            NSError* error = nil;
            NSDictionary* dict = [NSJSONSerialization JSONObjectWithData:jsonData options:0 error:&error];
            if (!error && [dict isKindOfClass:[NSDictionary class]]) {
                for (NSString* key in dict) {
                    id value = dict[key];
                    if ([value isKindOfClass:[NSArray class]]) {
                        headers[key] = [value componentsJoinedByString:@", "];
                    } else if ([value isKindOfClass:[NSString class]]) {
                        headers[key] = value;
                    }
                }
            }
        }

        NSHTTPURLResponse* response = [[NSHTTPURLResponse alloc] initWithURL:t.request.URL
                                                                   statusCode:status
                                                                  HTTPVersion:@"HTTP/1.1"
                                                                headerFields:headers];
        [t didReceiveResponse:response];
    });
}

void webviewSchemeTaskDidReceiveData(void* task, const void* data, int length) {
    id<WKURLSchemeTask> t = (__bridge id<WKURLSchemeTask>)task;
    NSData* d = [NSData dataWithBytes:data length:length];
    dispatch_sync(dispatch_get_main_queue(), ^{
        [t didReceiveData:d];
    });
}

void webviewSchemeTaskDidFinish(void* task) {
    id<WKURLSchemeTask> t = (__bridge id<WKURLSchemeTask>)task;
    dispatch_sync(dispatch_get_main_queue(), ^{
        [t didFinish];
    });
}

void webviewSetTitle(const char* title) {
    NSString* t = [NSString stringWithUTF8String:title];
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window setTitle:t];
    });
}

void webviewSetSize(int width, int height) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSRect frame = [gDelegate.window frame];
        NSRect newFrame = NSMakeRect(frame.origin.x, frame.origin.y + frame.size.height - height, width, height);
        [gDelegate.window setFrame:newFrame display:YES animate:NO];
    });
}

void webviewSetMinSize(int width, int height) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window setMinSize:NSMakeSize(width, height)];
    });
}

void webviewSetMaxSize(int width, int height) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window setMaxSize:NSMakeSize(width, height)];
    });
}

void webviewSetPosition(int x, int y) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSScreen* screen = [NSScreen mainScreen];
        CGFloat screenH = screen.frame.size.height;
        CGFloat winH = gDelegate.window.frame.size.height;
        [gDelegate.window setFrameOrigin:NSMakePoint(x, screenH - y - winH)];
    });
}

void webviewGetPosition(int* x, int* y) {
    __block int ox, oy;
    dispatch_sync(dispatch_get_main_queue(), ^{
        NSRect frame = [gDelegate.window frame];
        NSScreen* screen = [NSScreen mainScreen];
        CGFloat screenH = screen.frame.size.height;
        ox = (int)frame.origin.x;
        oy = (int)(screenH - frame.origin.y - frame.size.height);
    });
    *x = ox;
    *y = oy;
}

void webviewGetSize(int* width, int* height) {
    __block int ow, oh;
    dispatch_sync(dispatch_get_main_queue(), ^{
        NSRect frame = [gDelegate.window frame];
        ow = (int)frame.size.width;
        oh = (int)frame.size.height;
    });
    *width = ow;
    *height = oh;
}

void webviewShow(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window makeKeyAndOrderFront:nil];
    });
}

void webviewHide(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window orderOut:nil];
    });
}

void webviewMinimize(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window miniaturize:nil];
    });
}

void webviewMaximize(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window zoom:nil];
    });
}

void webviewFullscreen(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (([gDelegate.window styleMask] & NSWindowStyleMaskFullScreen) == 0) {
            [gDelegate.window toggleFullScreen:nil];
        }
    });
}

void webviewUnFullscreen(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (([gDelegate.window styleMask] & NSWindowStyleMaskFullScreen) != 0) {
            [gDelegate.window toggleFullScreen:nil];
        }
    });
}

void webviewRestore(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if ([gDelegate.window isMiniaturized]) {
            [gDelegate.window deminiaturize:nil];
        }
        if (([gDelegate.window styleMask] & NSWindowStyleMaskFullScreen) != 0) {
            [gDelegate.window toggleFullScreen:nil];
        }
    });
}

void webviewSetAlwaysOnTop(int onTop) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window setLevel:onTop ? NSFloatingWindowLevel : NSNormalWindowLevel];
    });
}

void webviewSetURL(const char* url) {
    NSString* u = [NSString stringWithUTF8String:url];
    dispatch_async(dispatch_get_main_queue(), ^{
        NSURL* nsurl = [NSURL URLWithString:u];
        [gDelegate.webview loadRequest:[NSURLRequest requestWithURL:nsurl]];
    });
}

void webviewClose(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [gDelegate.window close];
    });
}

// Create a temporary .app bundle with proper Info.plist for Dock name
static NSString* createTempAppBundle(NSString* appName, NSData* iconData) {
    NSFileManager* fm = [NSFileManager defaultManager];
    NSString* tempDir = NSTemporaryDirectory();
    NSString* bundleName = [NSString stringWithFormat:@"%@.dev.app", appName];
    NSString* bundlePath = [tempDir stringByAppendingPathComponent:bundleName];
    NSString* contentsPath = [bundlePath stringByAppendingPathComponent:@"Contents"];
    NSString* macosPath = [contentsPath stringByAppendingPathComponent:@"MacOS"];
    NSString* resourcesPath = [contentsPath stringByAppendingPathComponent:@"Resources"];
    
    // Remove existing bundle if any
    [fm removeItemAtPath:bundlePath error:nil];
    // Create directory structure
    NSError* error = nil;
    if (![fm createDirectoryAtPath:macosPath withIntermediateDirectories:YES attributes:nil error:&error]) {
        NSLog(@"Failed to create MacOS directory: %@", error);
        return nil;
    }
    if (![fm createDirectoryAtPath:resourcesPath withIntermediateDirectories:YES attributes:nil error:&error]) {
        NSLog(@"Failed to create Resources directory: %@", error);
        return nil;
    }
    
    // Get current executable path
    NSString* execPath = [[NSBundle mainBundle] executablePath];
    if (!execPath) {
        execPath = [NSString stringWithUTF8String:getenv("_")];
    }
    if (!execPath) {
        // Fallback: use /proc/self/exe equivalent on macOS
        char pathbuf[PROC_PIDPATHINFO_MAXSIZE];
        pid_t pid = getpid();
        if (proc_pidpath(pid, pathbuf, sizeof(pathbuf)) > 0) {
            execPath = [NSString stringWithUTF8String:pathbuf];
        }
    }
    
    // Create symlink to current executable
    NSString* execName = appName;
    NSString* newExecPath = [macosPath stringByAppendingPathComponent:execName];
    if (![fm createSymbolicLinkAtPath:newExecPath withDestinationPath:execPath error:&error]) {
        NSLog(@"Failed to create executable symlink: %@", error);
        return nil;
    }
    
    // Write icon if provided
    if (iconData) {
        NSString* iconPath = [resourcesPath stringByAppendingPathComponent:@"appicon.png"];
        [iconData writeToFile:iconPath atomically:YES];
    }
    
    // Create Info.plist
    NSString* plistPath = [contentsPath stringByAppendingPathComponent:@"Info.plist"];
    NSDictionary* plist = @{
        @"CFBundleName": appName,
        @"CFBundleDisplayName": appName,
        @"CFBundleExecutable": execName,
        @"CFBundleIdentifier": [NSString stringWithFormat:@"dev.%@", appName],
        @"CFBundlePackageType": @"APPL",
        @"CFBundleVersion": @"1.0",
        @"CFBundleShortVersionString": @"1.0",
        @"LSMinimumSystemVersion": @"10.13",
        @"NSHighResolutionCapable": @YES,
        @"CFBundleIconFile": @"appicon"
    };
    
    if (![plist writeToFile:plistPath atomically:YES]) {
        NSLog(@"Failed to write Info.plist");
        return nil;
    }
    
    NSLog(@"Created temp app bundle at: %@", bundlePath);
    return bundlePath;
}

static void cleanupTempAppBundle(NSString* bundlePath) {
    if (bundlePath) {
        [[NSFileManager defaultManager] removeItemAtPath:bundlePath error:nil];
    }
}

// Check if we're running inside an app bundle
static BOOL isRunningInAppBundle(void) {
    NSBundle* mainBundle = [NSBundle mainBundle];
    NSString* bundlePath = [mainBundle bundlePath];
    return [bundlePath hasSuffix:@".app"];
}
