#if 1
#include <windows.h>
#include <objbase.h>
#include <dwmapi.h>
#include <string>
#include <vector>
#include <memory>
#include <unordered_map>
#include "webview_windows.h"
#include "WebView2.h"

// Windows 11 rounded-corner API (SDK 10.0.22000+). Define fallbacks so we
// compile against older SDKs — the API itself is probed at runtime and is
// a no-op on pre-Windows-11 systems.
#ifndef DWMWA_WINDOW_CORNER_PREFERENCE
#define DWMWA_WINDOW_CORNER_PREFERENCE 33
typedef enum {
    DWMWCP_DEFAULT_CUSTOM    = 0,
    DWMWCP_DONOTROUND_CUSTOM = 1,
    DWMWCP_ROUND_CUSTOM      = 2,
    DWMWCP_ROUNDSMALL_CUSTOM = 3,
} DWM_WINDOW_CORNER_PREFERENCE_CUSTOM;
#define DWMWCP_ROUND       DWMWCP_ROUND_CUSTOM
#define DWMWCP_ROUNDSMALL  DWMWCP_ROUNDSMALL_CUSTOM
#endif

extern "C" {
void GoHandleMessage(void* webview, const char* msg);
void GoHandleSchemeTask(void* webview, void* task, const char* url);
void GoTrace(const char* msg);
}

static void Trace(const char* fmt, ...) {
    char buf[1024];
    va_list ap;
    va_start(ap, fmt);
    vsnprintf(buf, sizeof(buf), fmt, ap);
    va_end(ap);
    GoTrace(buf);
}

static HWND g_hwnd = nullptr;
static ICoreWebView2Environment* g_env = nullptr;
static ICoreWebView2Controller* g_controller = nullptr;
static ICoreWebView2* g_webview = nullptr;
static bool g_frameless = false;
static bool g_hidden = false;

// Dynamic loading of WebView2Loader.dll
typedef HRESULT (__stdcall *CreateEnvWithOptionsFunc)(
    PCWSTR browserExecutableFolder,
    PCWSTR userDataFolder,
    ICoreWebView2EnvironmentOptions* environmentOptions,
    ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler* environmentCreatedHandler);

static CreateEnvWithOptionsFunc pCreateCoreWebView2EnvironmentWithOptions = nullptr;

static bool LoadWebView2DLL() {
    if (pCreateCoreWebView2EnvironmentWithOptions) return true;
    HMODULE hDll = LoadLibraryW(L"WebView2Loader.dll");
    if (!hDll) return false;
    pCreateCoreWebView2EnvironmentWithOptions = (CreateEnvWithOptionsFunc)GetProcAddress(hDll, "CreateCoreWebView2EnvironmentWithOptions");
    return pCreateCoreWebView2EnvironmentWithOptions != nullptr;
}

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

// Raw COM callback implementations (no WRL dependency)
template<typename Interface>
struct ComCallback : Interface {
    ULONG refCount = 1;

    ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&refCount); }
};

static LRESULT CALLBACK WndProc(HWND hWnd, UINT message, WPARAM wParam, LPARAM lParam);
static void DoSchemeFinishOnUIThread(SchemeTask* task);

// Custom message used to marshal scheme task Finish back to the UI thread.
// WebView2 COM objects (g_env, args, deferral) are apartment-threaded — they
// MUST be called from the thread that owns them (the UI thread running the
// message loop). Finish arrives from Go on a goroutine OS thread, so we
// PostMessage(WM_APP+1, task, 0) and do the actual COM work in WndProc.
static const UINT WM_VELO_SCHEME_FINISH = WM_APP + 1;

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
        if (message == WM_VELO_SCHEME_FINISH) {
            DoSchemeFinishOnUIThread(reinterpret_cast<SchemeTask*>(wParam));
            return 0;
        }
        return DefWindowProcW(hWnd, message, wParam, lParam);
    }
    return 0;
}

static HRESULT InitWindow(HINSTANCE hInstance, bool frameless, bool hidden) {
    WNDCLASSW wc = {};
    wc.lpfnWndProc = WndProc;
    wc.hInstance = hInstance;
    wc.lpszClassName = L"WebView2WindowClass";
    RegisterClassW(&wc);

    // Always create with WS_VISIBLE so Windows performs the implicit first-show
    // and WebView2's DirectComposition surface initializes correctly. For hidden
    // startup, position the window off-screen to avoid any visible flash, then
    // call ShowWindow(SW_HIDE). webviewShow() will move it back on-screen on
    // first show.
    DWORD style = (frameless ? WS_POPUP : WS_OVERLAPPEDWINDOW) | WS_VISIBLE;
    int x = CW_USEDEFAULT, y = CW_USEDEFAULT;
    if (hidden) {
        x = -32000;
        y = -32000;
    }
    g_hwnd = CreateWindowExW(0, wc.lpszClassName, L"My App", style,
        x, y, 1024, 768,
        nullptr, nullptr, hInstance, nullptr);
    if (!g_hwnd) return E_FAIL;

    // Windows 11 rounded corners for frameless windows — DWM handles
    // anti-aliasing, shadows, DPI scaling. No-op on Windows 10/earlier
    // (E_INVALIDARG returned, safe to ignore). For framed windows, Windows
    // already rounds the system chrome so we skip this.
    if (frameless) {
        int pref = DWMWCP_ROUND;
        DwmSetWindowAttribute(g_hwnd, DWMWA_WINDOW_CORNER_PREFERENCE, &pref, sizeof(pref));
    }

    if (hidden) {
        ShowWindow(g_hwnd, SW_HIDE);
    }
    return S_OK;
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

// Executes the COM work for a scheme task finish. MUST run on the UI thread
// (the thread that owns g_env / args / deferral).
static void DoSchemeFinishOnUIThread(SchemeTask* task) {
    if (!task) return;
    if (g_env && task->args) {
        IStream* stream = MakeStream(task->body.data(), task->body.size());
        ICoreWebView2WebResourceResponse* resp = nullptr;
        std::wstring statusText = L"OK";
        std::wstring hdr = L"Content-Type: " + task->contentType;
        if (!task->headers.empty()) {
            hdr += L"\r\n" + task->headers;
        }
        // Diagnostic: log exactly what we send to WebView2
        {
            std::string hdrUtf8 = ToUtf8(hdr);
            Trace("finish: status=%d bytes=%zu headers=[%s]",
                task->status, task->body.size(), hdrUtf8.c_str());
            if (task->body.size() > 0) {
                size_t n = task->body.size() < 80 ? task->body.size() : 80;
                std::string head(reinterpret_cast<const char*>(task->body.data()), n);
                for (auto& c : head) if (c == '\n' || c == '\r') c = ' ';
                Trace("body head: %s", head.c_str());
            }
        }
        HRESULT hrCreate = g_env->CreateWebResourceResponse(stream, task->status, statusText.c_str(), hdr.c_str(), &resp);
        Trace("CreateWebResourceResponse HRESULT=0x%08X resp=%p", (unsigned int)hrCreate, (void*)resp);
        if (resp) {
            HRESULT hrPut = task->args->put_Response(resp);
            Trace("put_Response HRESULT=0x%08X", (unsigned int)hrPut);
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

void webviewSchemeTaskDidFinish(void* taskPtr) {
    SchemeTask* task = reinterpret_cast<SchemeTask*>(taskPtr);
    if (!task) return;
    // Marshal to UI thread — WebView2 COM objects are apartment-threaded and
    // must be called on the thread that created them.
    if (g_hwnd) {
        PostMessageW(g_hwnd, WM_VELO_SCHEME_FINISH, (WPARAM)task, 0);
    } else {
        // Fallback: no window yet; drop the task to avoid leaks/crash.
        if (task->deferral) { task->deferral->Complete(); task->deferral->Release(); }
        if (task->args) task->args->Release();
        delete task;
    }
}

// Raw COM implementation of ICoreWebView2WebMessageReceivedEventHandler
struct WebMessageReceivedHandler : ICoreWebView2WebMessageReceivedEventHandler {
    ULONG m_ref = 1;

    ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&m_ref); }
    ULONG STDMETHODCALLTYPE Release() override {
        ULONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return r;
    }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppv) override {
        if (!ppv) return E_POINTER;
        if (riid == IID_IUnknown || riid == IID_ICoreWebView2WebMessageReceivedEventHandler) {
            *ppv = static_cast<ICoreWebView2WebMessageReceivedEventHandler*>(this);
            AddRef();
            return S_OK;
        }
        *ppv = nullptr;
        return E_NOINTERFACE;
    }
    HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2* sender, ICoreWebView2WebMessageReceivedEventArgs* args) override {
        LPWSTR msg = nullptr;
        args->TryGetWebMessageAsString(&msg);
        std::string s = ToUtf8(msg ? std::wstring(msg) : L"");
        if (msg) CoTaskMemFree(msg);
        GoHandleMessage(sender, s.c_str());
        return S_OK;
    }
};

// Raw COM implementation of ICoreWebView2WebResourceRequestedEventHandler
struct WebResourceRequestedHandler : ICoreWebView2WebResourceRequestedEventHandler {
    ULONG m_ref = 1;

    ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&m_ref); }
    ULONG STDMETHODCALLTYPE Release() override {
        ULONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return r;
    }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppv) override {
        if (!ppv) return E_POINTER;
        if (riid == IID_IUnknown || riid == IID_ICoreWebView2WebResourceRequestedEventHandler) {
            *ppv = static_cast<ICoreWebView2WebResourceRequestedEventHandler*>(this);
            AddRef();
            return S_OK;
        }
        *ppv = nullptr;
        return E_NOINTERFACE;
    }
    HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2* sender, ICoreWebView2WebResourceRequestedEventArgs* args) override {
        ICoreWebView2WebResourceRequest* req = nullptr;
        args->get_Request(&req);
        LPWSTR uri = nullptr;
        if (req) req->get_Uri(&uri);
        std::wstring wuri = uri ? std::wstring(uri) : L"";
        if (uri) CoTaskMemFree(uri);
        if (req) req->Release();
        std::string suri = ToUtf8(wuri);
        Trace("WebResourceRequested: %s", suri.c_str());
        if (suri.rfind("velo://", 0) == 0) {
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
    }
};

// Raw COM implementation of ICoreWebView2NavigationCompletedEventHandler.
// Used only for diagnostic logging of navigation success/error code.
struct NavigationCompletedHandler : ICoreWebView2NavigationCompletedEventHandler {
    ULONG m_ref = 1;

    ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&m_ref); }
    ULONG STDMETHODCALLTYPE Release() override {
        ULONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return r;
    }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppv) override {
        if (!ppv) return E_POINTER;
        if (riid == IID_IUnknown || riid == IID_ICoreWebView2NavigationCompletedEventHandler) {
            *ppv = static_cast<ICoreWebView2NavigationCompletedEventHandler*>(this);
            AddRef();
            return S_OK;
        }
        *ppv = nullptr;
        return E_NOINTERFACE;
    }
    HRESULT STDMETHODCALLTYPE Invoke(ICoreWebView2* sender, ICoreWebView2NavigationCompletedEventArgs* args) override {
        BOOL success = FALSE;
        COREWEBVIEW2_WEB_ERROR_STATUS err = COREWEBVIEW2_WEB_ERROR_STATUS_UNKNOWN;
        if (args) {
            args->get_IsSuccess(&success);
            args->get_WebErrorStatus(&err);
        }
        Trace("NavigationCompleted: success=%d errorStatus=%d", success ? 1 : 0, (int)err);
        return S_OK;
    }
};

// Raw COM implementation of ICoreWebView2CustomSchemeRegistration.
// Needed so WebView2 intercepts velo:// URIs (custom schemes are NOT matched
// by AddWebResourceRequestedFilter unless registered here).
struct VeloSchemeRegistration : ICoreWebView2CustomSchemeRegistration {
    ULONG m_ref = 1;
    std::wstring m_schemeName;
    BOOL m_treatAsSecure = TRUE;
    BOOL m_hasAuthorityComponent = TRUE;

    VeloSchemeRegistration(const wchar_t* name) : m_schemeName(name) {}

    ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&m_ref); }
    ULONG STDMETHODCALLTYPE Release() override {
        ULONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return r;
    }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppv) override {
        if (!ppv) return E_POINTER;
        if (riid == IID_IUnknown || riid == IID_ICoreWebView2CustomSchemeRegistration) {
            *ppv = static_cast<ICoreWebView2CustomSchemeRegistration*>(this);
            AddRef();
            return S_OK;
        }
        *ppv = nullptr;
        return E_NOINTERFACE;
    }
    HRESULT STDMETHODCALLTYPE get_SchemeName(LPWSTR* value) override {
        if (!value) return E_POINTER;
        size_t bytes = (m_schemeName.size() + 1) * sizeof(wchar_t);
        *value = (LPWSTR)CoTaskMemAlloc(bytes);
        if (!*value) return E_OUTOFMEMORY;
        memcpy(*value, m_schemeName.c_str(), bytes);
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE get_TreatAsSecure(BOOL* value) override {
        if (!value) return E_POINTER;
        *value = m_treatAsSecure;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE put_TreatAsSecure(BOOL value) override {
        m_treatAsSecure = value;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE GetAllowedOrigins(UINT32* count, LPWSTR** origins) override {
        if (!count || !origins) return E_POINTER;
        *count = 0;
        *origins = nullptr;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE SetAllowedOrigins(UINT32 count, LPCWSTR* origins) override {
        (void)count; (void)origins;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE get_HasAuthorityComponent(BOOL* value) override {
        if (!value) return E_POINTER;
        *value = m_hasAuthorityComponent;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE put_HasAuthorityComponent(BOOL value) override {
        m_hasAuthorityComponent = value;
        return S_OK;
    }
};

// Raw COM implementation of ICoreWebView2EnvironmentOptions (+ Options4 for
// custom scheme support). Minimal surface — we only care about the scheme
// registrations; the string properties return empty values.
struct VeloEnvOptions : ICoreWebView2EnvironmentOptions, ICoreWebView2EnvironmentOptions4 {
    ULONG m_ref = 1;
    std::vector<ICoreWebView2CustomSchemeRegistration*> m_schemes;

    ~VeloEnvOptions() {
        for (auto* s : m_schemes) s->Release();
    }

    ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&m_ref); }
    ULONG STDMETHODCALLTYPE Release() override {
        ULONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return r;
    }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppv) override {
        if (!ppv) return E_POINTER;
        if (riid == IID_IUnknown || riid == IID_ICoreWebView2EnvironmentOptions) {
            *ppv = static_cast<ICoreWebView2EnvironmentOptions*>(this);
            AddRef();
            return S_OK;
        }
        if (riid == IID_ICoreWebView2EnvironmentOptions4) {
            *ppv = static_cast<ICoreWebView2EnvironmentOptions4*>(this);
            AddRef();
            return S_OK;
        }
        *ppv = nullptr;
        return E_NOINTERFACE;
    }

    // ICoreWebView2EnvironmentOptions — mirror WRL default: unset string props
    // return nullptr (COM convention), except TargetCompatibleBrowserVersion
    // which defaults to the SDK's hardcoded version (WebView2 validates it).
    HRESULT STDMETHODCALLTYPE get_AdditionalBrowserArguments(LPWSTR* value) override {
        if (!value) return E_POINTER;
        *value = nullptr;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE put_AdditionalBrowserArguments(LPCWSTR) override { return S_OK; }
    HRESULT STDMETHODCALLTYPE get_Language(LPWSTR* value) override {
        if (!value) return E_POINTER;
        *value = nullptr;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE put_Language(LPCWSTR) override { return S_OK; }
    HRESULT STDMETHODCALLTYPE get_TargetCompatibleBrowserVersion(LPWSTR* value) override {
        if (!value) return E_POINTER;
        // Must match SDK's CORE_WEBVIEW_TARGET_PRODUCT_VERSION (in
        // WebView2EnvironmentOptions.h: L"129.0.2792.45" for SDK 1.0.2792.45).
        static const wchar_t kVersion[] = L"129.0.2792.45";
        size_t bytes = sizeof(kVersion);
        *value = (LPWSTR)CoTaskMemAlloc(bytes);
        if (!*value) return E_OUTOFMEMORY;
        memcpy(*value, kVersion, bytes);
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE put_TargetCompatibleBrowserVersion(LPCWSTR) override { return S_OK; }
    HRESULT STDMETHODCALLTYPE get_AllowSingleSignOnUsingOSPrimaryAccount(BOOL* allow) override {
        if (!allow) return E_POINTER;
        *allow = FALSE;
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE put_AllowSingleSignOnUsingOSPrimaryAccount(BOOL) override { return S_OK; }

    // ICoreWebView2EnvironmentOptions4
    HRESULT STDMETHODCALLTYPE GetCustomSchemeRegistrations(UINT32* count, ICoreWebView2CustomSchemeRegistration*** schemes) override {
        if (!count || !schemes) return E_POINTER;
        *count = (UINT32)m_schemes.size();
        if (*count == 0) {
            *schemes = nullptr;
            return S_OK;
        }
        *schemes = (ICoreWebView2CustomSchemeRegistration**)CoTaskMemAlloc(*count * sizeof(ICoreWebView2CustomSchemeRegistration*));
        if (!*schemes) return E_OUTOFMEMORY;
        for (UINT32 i = 0; i < *count; i++) {
            (*schemes)[i] = m_schemes[i];
            m_schemes[i]->AddRef();
        }
        return S_OK;
    }
    HRESULT STDMETHODCALLTYPE SetCustomSchemeRegistrations(UINT32 count, ICoreWebView2CustomSchemeRegistration** schemes) override {
        for (auto* s : m_schemes) s->Release();
        m_schemes.clear();
        for (UINT32 i = 0; i < count; i++) {
            m_schemes.push_back(schemes[i]);
            schemes[i]->AddRef();
        }
        return S_OK;
    }
};

// Raw COM implementation of ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler
struct EnvCompletedHandler : ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler {
    ULONG m_ref = 1;
    const char* injectedJS;
    const char* url;

    EnvCompletedHandler(const char* js, const char* u) : injectedJS(js), url(u) {}

    ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&m_ref); }
    ULONG STDMETHODCALLTYPE Release() override {
        ULONG r = InterlockedDecrement(&m_ref);
        if (r == 0) delete this;
        return r;
    }
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppv) override {
        if (!ppv) return E_POINTER;
        if (riid == IID_IUnknown || riid == IID_ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler) {
            *ppv = static_cast<ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler*>(this);
            AddRef();
            return S_OK;
        }
        *ppv = nullptr;
        return E_NOINTERFACE;
    }
    HRESULT STDMETHODCALLTYPE Invoke(HRESULT result, ICoreWebView2Environment* env) override {
        if (FAILED(result) || !env) {
            wchar_t buf[128];
            swprintf_s(buf, L"EnvCompletedHandler HRESULT: 0x%08X", (unsigned int)result);
            MessageBoxW(nullptr, buf, L"WebView2 Env Failed", MB_ICONERROR);
            return E_FAIL;
        }
        g_env = env;
        g_env->AddRef();

        // Controller completed handler
        struct ControllerHandler : ICoreWebView2CreateCoreWebView2ControllerCompletedHandler {
            ULONG m_ref = 1;
            const char* injectedJS;
            const char* url;

            ControllerHandler(const char* js, const char* u) : injectedJS(js), url(u) {}

            ULONG STDMETHODCALLTYPE AddRef() override { return InterlockedIncrement(&m_ref); }
            ULONG STDMETHODCALLTYPE Release() override {
                ULONG r = InterlockedDecrement(&m_ref);
                if (r == 0) delete this;
                return r;
            }
            HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid, void** ppv) override {
                if (!ppv) return E_POINTER;
                if (riid == IID_IUnknown || riid == IID_ICoreWebView2CreateCoreWebView2ControllerCompletedHandler) {
                    *ppv = static_cast<ICoreWebView2CreateCoreWebView2ControllerCompletedHandler*>(this);
                    AddRef();
                    return S_OK;
                }
                *ppv = nullptr;
                return E_NOINTERFACE;
            }
            HRESULT STDMETHODCALLTYPE Invoke(HRESULT res, ICoreWebView2Controller* controller) override {
                if (FAILED(res) || !controller) {
                    wchar_t buf[128];
                    swprintf_s(buf, L"ControllerHandler HRESULT: 0x%08X", (unsigned int)res);
                    MessageBoxW(nullptr, buf, L"WebView2 Controller Failed", MB_ICONERROR);
                    return E_FAIL;
                }
                g_controller = controller;
                g_controller->AddRef();
                g_controller->get_CoreWebView2(&g_webview);
                if (!g_webview) return E_FAIL;

                // Setup message handler
                EventRegistrationToken tokenMsg;
                g_webview->add_WebMessageReceived(new WebMessageReceivedHandler(), &tokenMsg);

                // Setup resource request handler
                g_webview->AddWebResourceRequestedFilter(L"*", COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL);
                EventRegistrationToken tokenReq;
                g_webview->add_WebResourceRequested(new WebResourceRequestedHandler(), &tokenReq);

                // Setup navigation completed handler (diagnostic)
                EventRegistrationToken tokenNav;
                g_webview->add_NavigationCompleted(new NavigationCompletedHandler(), &tokenNav);

                // Inject JS
                if (injectedJS && injectedJS[0]) {
                    std::wstring wjs = ToWide(injectedJS);
                    g_webview->AddScriptToExecuteOnDocumentCreated(wjs.c_str(), nullptr);
                }

                // Navigate
                if (url && url[0]) {
                    Trace("Navigate -> %s", url);
                    std::wstring wurl = ToWide(url);
                    HRESULT navHr = g_webview->Navigate(wurl.c_str());
                    Trace("Navigate HRESULT=0x%08X", (unsigned int)navHr);
                }

                RECT bounds;
                GetClientRect(g_hwnd, &bounds);
                g_controller->put_Bounds(bounds);
                return S_OK;
            }
        };

        g_env->CreateCoreWebView2Controller(g_hwnd, new ControllerHandler(injectedJS, url));
        return S_OK;
    }
};

void webviewTerminate() {
    PostQuitMessage(0);
}

void webviewRunApp(const char* url, const char* injectedJS, const void* iconData, int iconLen, const char* title, int width, int height, int frameless, int hidden) {
    g_frameless = (frameless != 0);
    g_hidden = (hidden != 0);
    HINSTANCE hInstance = GetModuleHandle(nullptr);
    CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
    if (FAILED(InitWindow(hInstance, g_frameless, g_hidden))) return;

    if (title) {
        webviewSetTitle(title);
    }

    // Set initial window size
    if (width > 0 && height > 0) {
        RECT rc;
        GetWindowRect(g_hwnd, &rc);
        MoveWindow(g_hwnd, rc.left, rc.top, width, height, TRUE);
    }

    // Set window icon if provided
    if (iconData && iconLen > 0) {
        // Icon handling could be added here
    }

    if (!LoadWebView2DLL()) {
        MessageBoxW(nullptr, L"Failed to load WebView2Loader.dll", L"Error", MB_ICONERROR);
        return;
    }

    // Register velo:// as a custom scheme so WebResourceRequested fires for it.
    // The env options object is read asynchronously by WebView2, so we must
    // keep it alive beyond this call — we leak it intentionally (one small
    // alloc for the lifetime of the process).
    VeloEnvOptions* envOptions = new VeloEnvOptions();
    VeloSchemeRegistration* veloScheme = new VeloSchemeRegistration(L"velo");
    ICoreWebView2CustomSchemeRegistration* schemeArr[] = { veloScheme };
    envOptions->SetCustomSchemeRegistrations(1, schemeArr);
    veloScheme->Release();

    HRESULT hr = pCreateCoreWebView2EnvironmentWithOptions(nullptr, nullptr,
        static_cast<ICoreWebView2EnvironmentOptions*>(envOptions),
        new EnvCompletedHandler(injectedJS, url));
    if (FAILED(hr)) {
        wchar_t buf[128];
        swprintf_s(buf, L"CreateCoreWebView2EnvironmentWithOptions sync HRESULT: 0x%08X", (unsigned int)hr);
        MessageBoxW(nullptr, buf, L"WebView2 Create Failed", MB_ICONERROR);
    }

    // Diagnostic: display controller/env results via MessageBox in the async handlers.
    // (Sync return of CreateCoreWebView2EnvironmentWithOptions is rarely meaningful.)

    MSG msg;
    while (GetMessage(&msg, nullptr, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
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
    if (!g_hwnd) return;
    // If the window was created hidden off-screen (see InitWindow), move it
    // back onto the primary monitor on first show. Subsequent shows preserve
    // the user's last position.
    RECT rc;
    GetWindowRect(g_hwnd, &rc);
    if (rc.left < -10000 || rc.top < -10000) {
        int w = rc.right - rc.left;
        int h = rc.bottom - rc.top;
        int screenW = GetSystemMetrics(SM_CXSCREEN);
        int screenH = GetSystemMetrics(SM_CYSCREEN);
        int newX = (screenW - w) / 2;
        int newY = (screenH - h) / 2;
        if (newX < 0) newX = 0;
        if (newY < 0) newY = 0;
        MoveWindow(g_hwnd, newX, newY, w, h, FALSE);
    }
    ShowWindow(g_hwnd, SW_SHOW);
    SetForegroundWindow(g_hwnd);
}

void webviewHide() {
    if (!g_hwnd) return;
    ShowWindow(g_hwnd, SW_HIDE);
}

void webviewMinimize() {
    if (g_hwnd) ShowWindow(g_hwnd, SW_MINIMIZE);
}

void webviewMaximize() {
    if (g_hwnd) ShowWindow(g_hwnd, SW_MAXIMIZE);
}

static DWORD g_savedStyle = 0;

void webviewFullscreen() {
    if (!g_hwnd) return;
    g_savedStyle = GetWindowLong(g_hwnd, GWL_STYLE);
    MONITORINFO mi = { sizeof(mi) };
    if (GetMonitorInfo(MonitorFromWindow(g_hwnd, MONITOR_DEFAULTTOPRIMARY), &mi)) {
        SetWindowLong(g_hwnd, GWL_STYLE, g_savedStyle & ~(WS_OVERLAPPEDWINDOW | WS_POPUP));
        SetWindowPos(g_hwnd, HWND_TOP, mi.rcMonitor.left, mi.rcMonitor.top,
            mi.rcMonitor.right - mi.rcMonitor.left, mi.rcMonitor.bottom - mi.rcMonitor.top,
            SWP_NOOWNERZORDER | SWP_FRAMECHANGED);
    }
}

void webviewUnFullscreen() {
    if (!g_hwnd) return;
    SetWindowLong(g_hwnd, GWL_STYLE, g_savedStyle);
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

// Initiate a native window drag. Called from Go when JS posts
// __velo/window/start_drag (from a .velo-drag / [data-velo-drag] mousedown).
// Standard Win32 trick: release any existing mouse capture, then post a
// non-client left-button-down message claiming the cursor is on the title
// bar (HTCAPTION). Windows then takes over and drives the drag until
// mouse-up, including live move and Aero snap.
void webviewStartWindowDrag() {
    if (!g_hwnd) return;
    ReleaseCapture();
    PostMessageW(g_hwnd, WM_NCLBUTTONDOWN, HTCAPTION, 0);
}

#endif
