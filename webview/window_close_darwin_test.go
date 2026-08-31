//go:build darwin && !ios

package webview

import (
	"testing"

	"github.com/ltaoo/velo/webview/cocoa"
)

func TestWindowByNameReturnsRegisteredWindow(t *testing.T) {
	const name = "settings-test"
	webview := cocoa.ID(11)
	window := cocoa.ID(22)
	mapLock.Lock()
	namedWebViewMap[name] = webview
	nsWindowMap[uintptr(webview)] = window
	mapLock.Unlock()
	t.Cleanup(func() {
		mapLock.Lock()
		delete(namedWebViewMap, name)
		delete(nsWindowMap, uintptr(webview))
		mapLock.Unlock()
	})

	if actual := window_by_name("  " + name + "  "); actual != window {
		t.Fatalf("window_by_name() = %d, want %d", actual, window)
	}
	if actual := window_by_name("missing"); actual != 0 {
		t.Fatalf("window_by_name(missing) = %d, want 0", actual)
	}
}
