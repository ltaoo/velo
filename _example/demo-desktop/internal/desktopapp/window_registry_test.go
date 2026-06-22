package desktopapp

import (
	"encoding/json"
	"strings"
	"testing"

	"example/simple/internal/desktopapp/windowing"

	"github.com/ltaoo/velo/store"
)

func TestPersistedOpenWindowRegistryLifecycle(t *testing.T) {
	st := store.NewWithDir(t.TempDir())

	if err := rememberWindowSpec(st, windowing.WindowSpec{
		EntryPage: "settings.html",
		Height:    640,
		Name:      "settings",
		Pathname:  "/settings",
		Title:     "App-Settings",
		Width:     760,
	}); err != nil {
		t.Fatalf("rememberWindowSpec failed: %v", err)
	}

	file := loadPersistedOpenWindows(st)
	if got := len(file.Windows); got != 1 {
		t.Fatalf("persisted windows = %d, want 1", got)
	}
	if got := file.Windows[0].Name; got != "settings" {
		t.Fatalf("window name = %q, want settings", got)
	}

	if err := updatePersistedOpenWindowFixed(st, "settings", true); err != nil {
		t.Fatalf("updatePersistedOpenWindowFixed failed: %v", err)
	}
	file = loadPersistedOpenWindows(st)
	if !file.Windows[0].Fixed {
		t.Fatal("fixed state was not persisted")
	}
	if !strings.Contains(file.Windows[0].Pathname, "fixed=1") {
		t.Fatalf("pathname = %q, want fixed=1", file.Windows[0].Pathname)
	}

	if err := forgetPersistedOpenWindow(st, "settings"); err != nil {
		t.Fatalf("forgetPersistedOpenWindow failed: %v", err)
	}
	if got := len(loadPersistedOpenWindows(st).Windows); got != 0 {
		t.Fatalf("persisted windows after forget = %d, want 0", got)
	}
}

func TestPersistedWindowCloseHandlerForgetsSession(t *testing.T) {
	st := store.NewWithDir(t.TempDir())
	if err := rememberWindowSpec(st, windowing.WindowSpec{
		EntryPage: "settings.html",
		Height:    640,
		Name:      "settings",
		Pathname:  "/settings",
		Title:     "App-Settings",
		Width:     760,
	}); err != nil {
		t.Fatalf("rememberWindowSpec failed: %v", err)
	}

	onClose := forgetPersistedOpenWindowOnClose(st, nil)
	onClose("settings")
	onClose("desktop")

	if got := len(loadPersistedOpenWindows(st).Windows); got != 0 {
		t.Fatalf("persisted windows after close = %d, want 0", got)
	}
}

func TestPersistedWindowSessionStoresURLFrameAndState(t *testing.T) {
	st := store.NewWithDir(t.TempDir())
	rawState := json.RawMessage(`{"panel":"input-source","activeStorageId":"local"}`)

	if err := savePersistedWindowSession(st, PersistedOpenWindow{
		EntryPage: "settings.html",
		Fixed:     true,
		Height:    700,
		Kind:      persistedWindowKindOpenWindow,
		Name:      "settings",
		Pathname:  "/settings?panel=input-source",
		State:     rawState,
		Title:     "设置",
		Width:     800,
		X:         11,
		Y:         22,
	}); err != nil {
		t.Fatalf("savePersistedWindowSession failed: %v", err)
	}

	if st.Get(windowSessionsStoreKey) == nil {
		t.Fatalf("session was not written to %s", windowSessionsStoreKey)
	}
	session, ok := persistedWindowSession(st, "settings")
	if !ok {
		t.Fatal("session not found")
	}
	if session.Pathname != "/settings?fixed=1&panel=input-source" {
		t.Fatalf("pathname = %q, want fixed URL", session.Pathname)
	}
	if session.X != 11 || session.Y != 22 || session.Width != 800 || session.Height != 700 {
		t.Fatalf("frame = %d,%d %dx%d, want 11,22 800x700", session.X, session.Y, session.Width, session.Height)
	}
	if string(session.State) != string(rawState) {
		t.Fatalf("state = %s, want %s", session.State, rawState)
	}
}

func TestPersistedOpenWindowRegistrySkipsMainWindows(t *testing.T) {
	st := store.NewWithDir(t.TempDir())

	if err := rememberWindowSpec(st, windowing.WindowSpec{Name: "desktop", Pathname: "/desktop"}); err != nil {
		t.Fatalf("rememberWindowSpec failed: %v", err)
	}
	if err := rememberWindowSpec(st, windowing.WindowSpec{Name: "snippet-launcher", Pathname: "/snippet-launcher"}); err != nil {
		t.Fatalf("rememberWindowSpec failed: %v", err)
	}
	if got := len(loadPersistedOpenWindows(st).Windows); got != 0 {
		t.Fatalf("persisted windows = %d, want 0", got)
	}
}

func TestPersistedMemoWindowFixedUpdatesPayload(t *testing.T) {
	st := store.NewWithDir(t.TempDir())
	payload := MemoWindowPayload{
		Fixed: true,
		Memo:  json.RawMessage(`{"id":"memo-1","content":"hello"}`),
	}

	if err := rememberMemoWindow(st, "memo-1", payload); err != nil {
		t.Fatalf("rememberMemoWindow failed: %v", err)
	}
	file := loadPersistedOpenWindows(st)
	if got := len(file.Windows); got != 1 {
		t.Fatalf("persisted windows = %d, want 1", got)
	}
	if file.Windows[0].Kind != persistedWindowKindMemoWindow {
		t.Fatalf("kind = %q, want memo window", file.Windows[0].Kind)
	}
	if file.Windows[0].Payload == nil || !file.Windows[0].Payload.Fixed {
		t.Fatal("memo payload fixed state was not persisted")
	}

	if err := updatePersistedOpenWindowFixed(st, file.Windows[0].Name, false); err != nil {
		t.Fatalf("updatePersistedOpenWindowFixed failed: %v", err)
	}
	file = loadPersistedOpenWindows(st)
	if file.Windows[0].Fixed {
		t.Fatal("fixed state = true, want false")
	}
	if file.Windows[0].Payload == nil || file.Windows[0].Payload.Fixed {
		t.Fatal("memo payload fixed state was not updated")
	}
	if strings.Contains(file.Windows[0].Pathname, "fixed=1") {
		t.Fatalf("pathname = %q, did not expect fixed=1", file.Windows[0].Pathname)
	}
}
