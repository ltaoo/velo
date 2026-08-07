package velo

import "github.com/ltaoo/velo/webview"

// SetDebug enables or disables Velo's webview diagnostic logging.
// Debug logging is disabled by default. Call this before opening a window to
// also control diagnostic messages injected into the browser console.
func SetDebug(enabled bool) {
	webview.SetDebug(enabled)
}
