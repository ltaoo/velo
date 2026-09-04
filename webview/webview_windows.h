#ifndef WEBVIEW_WINDOWS_H
#define WEBVIEW_WINDOWS_H
#ifdef __cplusplus
extern "C" {
#endif
void webviewRunApp(const char* name, const char* url, const char* injectedJS, const void* iconData, int iconLen, const char* title, int width, int height, int frameless, int hidden, int hideOnClose, int disableResize, int disableMinimize, int disableMaximize, const char* loaderPath);
void webviewOpenWindow(const char* name, const char* url, const char* injectedJS, const char* title, int width, int height, int x, int y, int hasPosition, int frameless, int hidden, int nonActivating, int preserveStateOnFocus, int hideOnClose, int disableResize, int disableMinimize, int disableMaximize);
void webviewEval(void* webview, const char* js);
void webviewTerminate();
void webviewSchemeTaskDidReceiveResponse(void* task, int status, const char* contentType, const char* headers);
void webviewSchemeTaskDidReceiveData(void* task, const void* data, int length);
void webviewSchemeTaskDidFinish(void* task);

void webviewSetTitle(const char* title);
void webviewSetSize(int width, int height);
void webviewSetMinSize(int width, int height);
void webviewSetMaxSize(int width, int height);
void webviewSetPosition(int x, int y);
void webviewGetPosition(int* x, int* y);
void webviewGetSize(int* width, int* height);
void webviewShow(void);
void webviewHide(void);
void webviewFocus(void);
int webviewIsVisible(void);
int webviewIsFocused(void);
void webviewMinimize(void);
void webviewMaximize(void);
void webviewFullscreen(void);
void webviewUnFullscreen(void);
void webviewRestore(void);
void webviewSetAlwaysOnTop(int onTop);
void webviewSetURL(const char* url);
void webviewClose(void);
void webviewStartWindowDrag(void);
void webviewCloseWindow(void* webview);
void webviewMinimizeWindow(void* webview);
void webviewMaximizeWindow(void* webview);
void webviewRestoreWindow(void* webview);
void webviewHideWindow(void* webview);
void webviewSetWindowSize(void* webview, int width, int height);
void webviewGetWindowState(void* webview, int* x, int* y, int* width, int* height);
void webviewSetWindowAlwaysOnTop(void* webview, int onTop);
void webviewStartWindowDragFor(void* webview);
#ifdef __cplusplus
}
#endif
#endif
