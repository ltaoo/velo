#include "tray_windows.h"
#include <windows.h>
#include <shellapi.h>
#include <map>
#include <string>
#include <vector>
#include "_cgo_export.h"

#define WM_TRAY_CALLBACK_MESSAGE (WM_USER + 1)
#define ID_TRAY_ICON 1001

std::wstring Utf8ToWide(const char* str) {
    if (!str) return L"";
    int len = MultiByteToWideChar(CP_UTF8, 0, str, -1, NULL, 0);
    if (len <= 0) return L"";
    std::vector<wchar_t> buf(len);
    MultiByteToWideChar(CP_UTF8, 0, str, -1, buf.data(), len);
    return std::wstring(buf.data());
}

class Tray {
public:
    Tray();
    ~Tray();
    void Init();
    void RunLoop();
    void Quit();
    void SetIcon(const char* data, int length);
    void SetTooltip(const char* tooltip);
    void AddMenuItem(int id, const char* title, const char* shortcut, int disabled, int checked, int parentId, int isSubmenu);
    void AddSeparator(int parentId);
    
    // Updates
    void SetItemLabel(int id, const char* label);
    void SetItemChecked(int id, int checked);
    void SetItemDisabled(int id, int disabled);

    static LRESULT CALLBACK WndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

private:
    HWND hwnd;
    NOTIFYICONDATAW nid;
    HMENU hMenu;
    std::map<int, HMENU> subMenus; // Map ID (of submenu item) -> HMENU
};

static Tray* g_tray = nullptr;

Tray::Tray() : hwnd(NULL), hMenu(NULL) {
    memset(&nid, 0, sizeof(nid));
}

Tray::~Tray() {
    if (nid.hIcon) DestroyIcon(nid.hIcon);
    if (hMenu) DestroyMenu(hMenu);
    Shell_NotifyIconW(NIM_DELETE, &nid);
}

void Tray::Init() {
    WNDCLASSEXW wc;
    memset(&wc, 0, sizeof(wc));
    wc.cbSize = sizeof(wc);
    wc.lpfnWndProc = Tray::WndProc;
    wc.hInstance = GetModuleHandle(NULL);
    wc.lpszClassName = L"GoBoxTrayClass";
    RegisterClassExW(&wc);

    hwnd = CreateWindowExW(0, L"GoBoxTrayClass", L"GoBox Tray", 0, 0, 0, 0, 0, HWND_MESSAGE, NULL, GetModuleHandle(NULL), NULL);

    nid.cbSize = sizeof(nid);
    nid.hWnd = hwnd;
    nid.uID = ID_TRAY_ICON;
    nid.uFlags = NIF_MESSAGE;
    nid.uCallbackMessage = WM_TRAY_CALLBACK_MESSAGE;
    Shell_NotifyIconW(NIM_ADD, &nid);

    hMenu = CreatePopupMenu();
    subMenus[0] = hMenu; // Root
}

LRESULT CALLBACK Tray::WndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
    case WM_TRAY_CALLBACK_MESSAGE:
        if (lParam == WM_RBUTTONUP || lParam == WM_LBUTTONUP) {
            POINT p;
            GetCursorPos(&p);
            SetForegroundWindow(hwnd);
            if (g_tray) {
                TrackPopupMenu(g_tray->hMenu, TPM_BOTTOMALIGN | TPM_LEFTALIGN, p.x, p.y, 0, hwnd, NULL);
            }
        }
        break;
    case WM_COMMAND:
        {
            int id = LOWORD(wParam);
            trayCallback(id);
        }
        break;
    case WM_DESTROY:
        PostQuitMessage(0);
        break;
    default:
        return DefWindowProcW(hwnd, msg, wParam, lParam);
    }
    return 0;
}

void Tray::RunLoop() {
    MSG msg;
    while (GetMessage(&msg, NULL, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
    }
}

void Tray::Quit() {
    PostMessage(hwnd, WM_CLOSE, 0, 0);
}

void Tray::SetIcon(const char* data, int length) {
    HICON hIcon = CreateIconFromResourceEx((PBYTE)data, length, TRUE, 0x00030000, 0, 0, LR_DEFAULTCOLOR);
    if (hIcon) {
        if (nid.hIcon) DestroyIcon(nid.hIcon);
        nid.hIcon = hIcon;
        nid.uFlags |= NIF_ICON;
        Shell_NotifyIconW(NIM_MODIFY, &nid);
    }
}

void Tray::SetTooltip(const char* tooltip) {
    std::wstring wTooltip = Utf8ToWide(tooltip);
    wcsncpy(nid.szTip, wTooltip.c_str(), 127);
    nid.szTip[127] = 0;
    nid.uFlags |= NIF_TIP;
    Shell_NotifyIconW(NIM_MODIFY, &nid);
}

void Tray::AddMenuItem(int id, const char* title, const char* shortcut, int disabled, int checked, int parentId, int isSubmenu) {
    HMENU parent = subMenus[parentId];
    if (!parent && parentId == 0) parent = hMenu;
    if (!parent) return;

    std::wstring wTitle = Utf8ToWide(title);
    if (shortcut && *shortcut) {
        wTitle += L"\t";
        wTitle += Utf8ToWide(shortcut);
    }
    UINT flags = MF_STRING;
    if (disabled) flags |= MF_GRAYED;
    if (checked) flags |= MF_CHECKED;

    if (isSubmenu) {
        HMENU hSub = CreatePopupMenu();
        subMenus[id] = hSub;
        flags |= MF_POPUP;
        AppendMenuW(parent, flags, (UINT_PTR)hSub, wTitle.c_str());
    } else {
        AppendMenuW(parent, flags, id, wTitle.c_str());
    }
}

void Tray::AddSeparator(int parentId) {
    HMENU parent = subMenus[parentId];
    if (!parent && parentId == 0) parent = hMenu;
    if (!parent) return;
    AppendMenuW(parent, MF_SEPARATOR, 0, NULL);
}

void Tray::SetItemLabel(int id, const char* label) {
    std::wstring wLabel = Utf8ToWide(label);
    for (auto const& item : subMenus) {
        HMENU h = item.second;
        if (ModifyMenuW(h, id, MF_BYCOMMAND | MF_STRING, id, wLabel.c_str())) return;
    }
}

void Tray::SetItemChecked(int id, int checked) {
    for (auto const& item : subMenus) {
         HMENU h = item.second;
         DWORD state = checked ? MF_CHECKED : MF_UNCHECKED;
         if (CheckMenuItem(h, id, MF_BYCOMMAND | state) != -1) return;
    }
}

void Tray::SetItemDisabled(int id, int disabled) {
    for (auto const& item : subMenus) {
        HMENU h = item.second;
        DWORD state = disabled ? MF_GRAYED : MF_ENABLED;
        if (EnableMenuItem(h, id, MF_BYCOMMAND | state) != -1) return;
    }
}

// C exports
void init_tray_win() {
    if (!g_tray) {
        g_tray = new Tray();
        g_tray->Init();
    }
}

void set_icon_win(const char* data, int length) {
    if (g_tray) g_tray->SetIcon(data, length);
}

void set_tooltip_win(const char* tooltip) {
    if (g_tray) g_tray->SetTooltip(tooltip);
}

void add_menu_item_win(int id, const char* title, const char* shortcut, int disabled, int checked, int parentId, int isSubmenu) {
    if (g_tray) g_tray->AddMenuItem(id, title, shortcut, disabled, checked, parentId, isSubmenu);
}

void add_separator_win(int parentId) {
    if (g_tray) g_tray->AddSeparator(parentId);
}

void run_loop_win() {
    if (g_tray) g_tray->RunLoop();
}

void quit_app_win() {
    if (g_tray) g_tray->Quit();
}

void set_item_label_win(int id, const char* label) {
    if (g_tray) g_tray->SetItemLabel(id, label);
}
void set_item_tooltip_win(int id, const char* tooltip) {
    // Not implemented
}
void set_item_checked_win(int id, int checked) {
    if (g_tray) g_tray->SetItemChecked(id, checked);
}
void set_item_disabled_win(int id, int disabled) {
    if (g_tray) g_tray->SetItemDisabled(id, disabled);
}
