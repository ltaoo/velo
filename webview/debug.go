package webview

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

var (
	debugMode     atomic.Bool
	debugOutput   io.Writer = os.Stderr
	debugOutputMu sync.Mutex
)

// SetDebug enables or disables diagnostic logging from the webview package.
// Debug logging is disabled by default.
func SetDebug(enabled bool) {
	debugMode.Store(enabled)
}

func debugEnabled() bool {
	return debugMode.Load()
}

func debugln(args ...interface{}) {
	if !debugEnabled() {
		return
	}
	debugOutputMu.Lock()
	defer debugOutputMu.Unlock()
	fmt.Fprintln(debugOutput, args...)
}

func debugf(format string, args ...interface{}) {
	if !debugEnabled() {
		return
	}
	debugOutputMu.Lock()
	defer debugOutputMu.Unlock()
	fmt.Fprintf(debugOutput, format, args...)
}
