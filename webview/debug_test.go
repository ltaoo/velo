package webview

import (
	"bytes"
	"testing"
)

func TestDebugLoggingRequiresExplicitOptIn(t *testing.T) {
	var output bytes.Buffer

	debugOutputMu.Lock()
	previousOutput := debugOutput
	debugOutput = &output
	debugOutputMu.Unlock()
	t.Cleanup(func() {
		SetDebug(false)
		debugOutputMu.Lock()
		debugOutput = previousOutput
		debugOutputMu.Unlock()
	})

	if debugEnabled() {
		t.Fatal("debug logging should be disabled by default")
	}
	debugln("DEBUG: hidden")
	if output.Len() != 0 {
		t.Fatalf("debug output should be silent by default, got %q", output.String())
	}

	SetDebug(true)
	debugf("DEBUG: visible %d\n", 42)
	if got, want := output.String(), "DEBUG: visible 42\n"; got != want {
		t.Fatalf("debug output = %q, want %q", got, want)
	}

	output.Reset()
	SetDebug(false)
	debugf("DEBUG: hidden again\n")
	if output.Len() != 0 {
		t.Fatalf("debug output should stop after disabling, got %q", output.String())
	}
}
