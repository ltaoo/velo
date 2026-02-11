#ifndef WEBVIEW_DARWIN_H
#define WEBVIEW_DARWIN_H

void webviewRunApp(const char* url, const char* injectedJS, const void* iconData, int iconLen, const char* appName, int width, int height);
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
void webviewMinimize(void);
void webviewMaximize(void);
void webviewFullscreen(void);
void webviewUnFullscreen(void);
void webviewRestore(void);
void webviewSetAlwaysOnTop(int onTop);
void webviewSetURL(const char* url);
void webviewClose(void);

#endif
