package velo

import "testing"

func TestNewWebviewPassesWindowRestrictions(t *testing.T) {
	app := NewApp(&VeloAppOpt{Mode: ModeHttp})
	app.NewWebview(&VeloWebviewOpt{
		DisableResize:   true,
		DisableMinimize: true,
		DisableMaximize: true,
	})

	if len(app.webviews) != 1 {
		t.Fatalf("webviews = %d, want 1", len(app.webviews))
	}
	options := app.webviews[0]
	if !options.DisableResize || !options.DisableMinimize || !options.DisableMaximize {
		t.Fatalf("window restrictions were not preserved: %+v", options)
	}
}
