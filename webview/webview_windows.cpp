#if 0
#include <windows.h>
#include <objbase.h>
#include <string>
#include <vector>
#include <memory>
#include "webview_windows.h"
#include "WebView2.h"

extern "C" {
void GoHandleMessage(void* webview, const char* msg);
void GoHandleSchemeTask(void* webview, void* task, const char* url);
}
 

static HWND g_hwnd = nullptr;
static ICoreWebView2Environment* g_env = nullptr;
static ICoreWebView2Controller* g_controller = nullptr;
static ICoreWebView2* g_webview = nullptr;

struct SchemeTask {
    ICoreWebView2WebResourceRequestedEventArgs* args = nullptr;
    ICoreWebView2Deferral* deferral = nullptr;
    std::wstring contentType;
    std::wstring headers;
    int status = 200;
    std::vector<unsigned char> body;
};

static std::wstring ToWide(const char* s) {
    if (!s) return L"";
    int len = MultiByteToWideChar(CP_UTF8, 0, s, -1, nullptr, 0);
    std::wstring ws;
    ws.resize(len ? len - 1 : 0);
    if (len) MultiByteToWideChar(CP_UTF8, 0, s, -1, &ws[0], len);
    return ws;
}

static std::string ToUtf8(const std::wstring& ws) {
    int len = WideCharToMultiByte(CP_UTF8, 0, ws.c_str(), -1, nullptr, 0, nullptr, nullptr);
    std::string s;
    s.resize(len ? len - 1 : 0);
    if (len) WideCharToMultiByte(CP_UTF8, 0, ws.c_str(), -1, &s[0], len, nullptr, nullptr);
    return s;
}

static IStream* MakeStream(const unsigned char* data, size_t len) {
    HGLOBAL hMem = GlobalAlloc(GMEM_MOVEABLE, len);
    if (!hMem) return nullptr;
    void* p = GlobalLock(hMem);
    if (!p) {
        GlobalFree(hMem);
        return nullptr;
    }
    memcpy(p, data, len);
    GlobalUnlock(hMem);
    IStream* stream = nullptr;
    if (FAILED(CreateStreamOnHGlobal(hMem, TRUE, &stream))) {
        GlobalFree(hMem);
        return nullptr;
    }
    return stream;
}

static LRESULT CALLBACK WndProc(HWND hWnd, UINT message, WPARAM wParam, LPARAM lParam) {
    switch (message) {
    case WM_SIZE:
        if (g_controller) {
            RECT bounds;
            GetClientRect(hWnd, &bounds);
            g_controller->put_Bounds(bounds);
        }
        break;
    case WM_DESTROY:
        PostQuitMessage(0);
        break;
    default:
        return DefWindowProc(hWnd, message, wParam, lParam);
    }
    return 0;
}

static HRESULT InitWindow(HINSTANCE hInstance) {
    WNDCLASS wc = {};
    wc.lpfnWndProc = WndProc;
    wc.hInstance = hInstance;
    wc.lpszClassName = L"WebView2WindowClass";
    RegisterClass(&wc);
    g_hwnd = CreateWindowEx(0, wc.lpszClassName, L"My App", WS_OVERLAPPEDWINDOW | WS_VISIBLE, CW_USEDEFAULT, CW_USEDEFAULT, 1024, 768, nullptr, nullptr, hInstance, nullptr);
    return g_hwnd ? S_OK : E_FAIL;
}

void webviewEval(void* webview, const char* js) {
    ICoreWebView2* wv = reinterpret_cast<ICoreWebView2*>(webview);
    if (!wv || !js) return;
    std::wstring wjs = ToWide(js);
    wv->ExecuteScript(wjs.c_str(), nullptr);
}

void webviewSchemeTaskDidReceiveResponse(void* taskPtr, int status, const char* contentType, const char* headers) {
    SchemeTask* task = reinterpret_cast<SchemeTask*>(taskPtr);
    if (!task) return;
    task->status = status;
    task->contentType = ToWide(contentType);
    task->headers = ToWide(headers);
}

void webviewSchemeTaskDidReceiveData(void* taskPtr, const void* data, int length) {
    SchemeTask* task = reinterpret_cast<SchemeTask*>(taskPtr);
    if (!task || !data || length <= 0) return;
    const unsigned char* p = reinterpret_cast<const unsigned char*>(data);
    task->body.insert(task->body.end(), p, p + length);
}

void webviewSchemeTaskDidFinish(void* taskPtr) {
    SchemeTask* task = reinterpret_cast<SchemeTask*>(taskPtr);
    if (!task) return;
    if (g_env && task->args) {
        IStream* stream = MakeStream(task->body.data(), task->body.size());
        ICoreWebView2WebResourceResponse* resp = nullptr;
        std::wstring statusText = L"OK";
        g_env->CreateWebResourceResponse(stream, task->status, statusText.c_str(), task->headers.c_str(), &resp);
        if (resp) {
            task->args->put_Response(resp);
            resp->Release();
        }
        if (stream) stream->Release();
    }
    if (task->deferral) {
        task->deferral->Complete();
        task->deferral->Release();
        task->deferral = nullptr;
    }
    if (task->args) {
        task->args->Release();
        task->args = nullptr;
    }
    delete task;
}

static void SetupHandlers(const char* injectedJS) {
    if (!g_webview) return;
    if (injectedJS && injectedJS[0]) {
        std::wstring wjs = ToWide(injectedJS);
        g_webview->AddScriptToExecuteOnDocumentCreated(wjs.c_str(), nullptr);
    }
    EventRegistrationToken tokenMsg;
    g_webview->add_WebMessageReceived(Microsoft::WRL::Callback<ICoreWebView2WebMessageReceivedEventHandler>(
        [](ICoreWebView2* sender, ICoreWebView2WebMessageReceivedEventArgs* args) -> HRESULT {
            LPWSTR msg = nullptr;
            args->get_WebMessageAsString(&msg);
            std::string s = ToUtf8(msg ? std::wstring(msg) : L"");
            if (msg) CoTaskMemFree(msg);
            GoHandleMessage(sender, s.c_str());
            return S_OK;
        }).Get(), &tokenMsg);
    g_webview->AddWebResourceRequestedFilter(L"*", COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL);
    EventRegistrationToken tokenReq;
    g_webview->add_WebResourceRequested(Microsoft::WRL::Callback<ICoreWebView2WebResourceRequestedEventHandler>(
        [](ICoreWebView2* sender, ICoreWebView2WebResourceRequestedEventArgs* args) -> HRESULT {
            ICoreWebView2WebResourceRequest* req = nullptr;
            args->get_Request(&req);
            LPWSTR uri = nullptr;
            if (req) req->get_Uri(&uri);
            std::wstring wuri = uri ? std::wstring(uri) : L"";
            if (uri) CoTaskMemFree(uri);
            if (req) req->Release();
            std::string suri = ToUtf8(wuri);
            if (suri.rfind("funzm://", 0) == 0) {
                ICoreWebView2Deferral* def = nullptr;
                args->GetDeferral(&def);
                SchemeTask* task = new SchemeTask();
                task->args = args;
                task->args->AddRef();
                task->deferral = def;
                GoHandleSchemeTask(sender, task, suri.c_str());
                return S_OK;
            }
            return S_OK;
        }).Get(), &tokenReq);
}

void webviewTerminate() {
    PostQuitMessage(0);
}

static void NavigateTo(const char* url) {
    if (!g_webview || !url) return;
    std::wstring wurl = ToWide(url);
    g_webview->Navigate(wurl.c_str());
}

void webviewRunApp(const char* url, const char* injectedJS, const void* iconData, int iconLen) {
    HINSTANCE hInstance = GetModuleHandle(nullptr);
    CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
    if (FAILED(InitWindow(hInstance))) return;
    CreateCoreWebView2EnvironmentWithOptions(nullptr, nullptr, nullptr,
        Microsoft::WRL::Callback<ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler>(
            [](HRESULT result, ICoreWebView2Environment* env) -> HRESULT {
                if (FAILED(result) || !env) return E_FAIL;
                g_env = env;
                g_env->AddRef();
                g_env->CreateCoreWebView2Controller(g_hwnd,
                    Microsoft::WRL::Callback<ICoreWebView2CreateCoreWebView2ControllerCompletedHandler>(
                        [](HRESULT res, ICoreWebView2Controller* controller) -> HRESULT {
                            if (FAILED(res) || !controller) return E_FAIL;
                            g_controller = controller;
                            g_controller->AddRef();
                            g_controller->get_CoreWebView2(&g_webview);
                            if (!g_webview) return E_FAIL;
                            RECT bounds;
                            GetClientRect(g_hwnd, &bounds);
                            g_controller->put_Bounds(bounds);
                            return S_OK;
                        }).Get());
                return S_OK;
            }).Get());
    MSG msg;
    bool initialized = false;
    while (GetMessage(&msg, nullptr, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
        if (!initialized && g_webview) {
            initialized = true;
            SetupHandlers(injectedJS);
            NavigateTo(url);
        }
    }
    if (g_webview) { g_webview->Release(); g_webview = nullptr; }
    if (g_controller) { g_controller->Release(); g_controller = nullptr; }
    if (g_env) { g_env->Release(); g_env = nullptr; }
}

void webviewSetTitle(const char* title) {
    if (!g_hwnd || !title) return;
    std::wstring wt = ToWide(title);
    SetWindowTextW(g_hwnd, wt.c_str());
}

void webviewSetSize(int width, int height) {
    if (!g_hwnd) return;
    RECT rc;
    GetWindowRect(g_hwnd, &rc);
    MoveWindow(g_hwnd, rc.left, rc.top, width, height, TRUE);
}

void webviewSetMinSize(int width, int height) {
    // Win32 min/max size requires WM_GETMINMAXINFO handling; stub for now
}

void webviewSetMaxSize(int width, int height) {
    // Win32 min/max size requires WM_GETMINMAXINFO handling; stub for now
}

void webviewSetPosition(int x, int y) {
    if (!g_hwnd) return;
    RECT rc;
    GetWindowRect(g_hwnd, &rc);
    int w = rc.right - rc.left;
    int h = rc.bottom - rc.top;
    MoveWindow(g_hwnd, x, y, w, h, TRUE);
}

void webviewGetPosition(int* x, int* y) {
    if (!g_hwnd) { *x = 0; *y = 0; return; }
    RECT rc;
    GetWindowRect(g_hwnd, &rc);
    *x = rc.left;
    *y = rc.top;
}

void webviewGetSize(int* width, int* height) {
    if (!g_hwnd) { *width = 0; *height = 0; return; }
    RECT rc;
    GetWindowRect(g_hwnd, &rc);
    *width = rc.right - rc.left;
    *height = rc.bottom - rc.top;
}

void webviewShow() {
    if (g_hwnd) ShowWindow(g_hwnd, SW_SHOW);
}

void webviewHide() {
    if (g_hwnd) ShowWindow(g_hwnd, SW_HIDE);
}

void webviewMinimize() {
    if (g_hwnd) ShowWindow(g_hwnd, SW_MINIMIZE);
}

void webviewMaximize() {
    if (g_hwnd) ShowWindow(g_hwnd, SW_MAXIMIZE);
}

void webviewFullscreen() {
    if (!g_hwnd) return;
    DWORD style = GetWindowLong(g_hwnd, GWL_STYLE);
    MONITORINFO mi = { sizeof(mi) };
    if (GetMonitorInfo(MonitorFromWindow(g_hwnd, MONITOR_DEFAULTTOPRIMARY), &mi)) {
        SetWindowLong(g_hwnd, GWL_STYLE, style & ~WS_OVERLAPPEDWINDOW);
        SetWindowPos(g_hwnd, HWND_TOP, mi.rcMonitor.left, mi.rcMonitor.top,
            mi.rcMonitor.right - mi.rcMonitor.left, mi.rcMonitor.bottom - mi.rcMonitor.top,
            SWP_NOOWNERZORDER | SWP_FRAMECHANGED);
    }
}

void webviewUnFullscreen() {
    if (!g_hwnd) return;
    SetWindowLong(g_hwnd, GWL_STYLE, WS_OVERLAPPEDWINDOW | WS_VISIBLE);
    SetWindowPos(g_hwnd, NULL, 0, 0, 0, 0,
        SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_FRAMECHANGED);
}

void webviewRestore() {
    if (g_hwnd) ShowWindow(g_hwnd, SW_RESTORE);
}

void webviewSetAlwaysOnTop(int onTop) {
    if (!g_hwnd) return;
    SetWindowPos(g_hwnd, onTop ? HWND_TOPMOST : HWND_NOTOPMOST, 0, 0, 0, 0,
        SWP_NOMOVE | SWP_NOSIZE);
}

void webviewSetURL(const char* url) {
    if (!g_webview || !url) return;
    std::wstring wurl = ToWide(url);
    g_webview->Navigate(wurl.c_str());
}

void webviewClose() {
    if (g_hwnd) DestroyWindow(g_hwnd);
}

#endif
