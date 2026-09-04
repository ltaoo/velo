package velo

import "testing"

func TestNewWebviewPassesWindowRestrictions(t *testing.T) {
	app := NewApp(&VeloAppOpt{Mode: ModeHttp, HideDockIcon: true})
	background_color := NewRGB(27, 38, 54)
	app.NewWebview(&VeloWebviewOpt{
		DisableResize:          true,
		DisableMinimize:        true,
		DisableMaximize:        true,
		DisableZoom:            true,
		ReloadContextMenu:      true,
		BackgroundColor:        background_color,
		MacBackdropTranslucent: true,
		MacTitleBarHeight:      50,
		MacTitleBarInset:       true,
		HiddenOnTaskbar:        true,
	})

	if len(app.webviews) != 1 {
		t.Fatalf("webviews = %d, want 1", len(app.webviews))
	}
	options := app.webviews[0]
	if !options.DisableResize || !options.DisableMinimize || !options.DisableMaximize {
		t.Fatalf("window restrictions were not preserved: %+v", options)
	}
	if !options.HideDockIcon {
		t.Fatal("HideDockIcon was not preserved")
	}
	if !options.ReloadContextMenu {
		t.Fatal("ReloadContextMenu was not preserved")
	}
	if options.BackgroundColor != background_color || !options.DisableZoom || !options.MacBackdropTranslucent || options.MacTitleBarHeight != 50 || !options.MacTitleBarInset || !options.HiddenOnTaskbar {
		t.Fatal("window appearance options were not preserved")
	}
}
