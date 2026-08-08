package velo

import (
	"os"
	"testing"
)

func TestNewAppDoesNotEnableLocalStorageByDefault(t *testing.T) {
	app := NewApp(&VeloAppOpt{Mode: ModeHttp})

	if app.Store != nil {
		t.Fatal(`expected local storage to be disabled by default`)
	}
	assertStoreRoutes(t, app, false)
}

func TestNewAppEnablesLocalStorageExplicitly(t *testing.T) {
	app := NewApp(&VeloAppOpt{Mode: ModeHttp, EnableLocalStorage: true})

	if app.Store == nil {
		t.Fatal(`expected local storage to be enabled`)
	}
	storagePath := app.Store.Path()
	t.Cleanup(func() { _ = os.Remove(storagePath) })
	if _, err := os.Stat(storagePath); err != nil {
		t.Fatalf(`expected storage file to be created: %v`, err)
	}
	assertStoreRoutes(t, app, true)
}

func assertStoreRoutes(t *testing.T, app *Box, want bool) {
	t.Helper()
	for _, route := range []string{
		`/api/storage/get`,
		`/api/storage/set`,
		`/api/storage/delete`,
		`/api/window/state/snapshot`,
		`/api/window/state/load`,
	} {
		_, got := app.get_handlers[route]
		if got != want {
			t.Errorf(`route %q registered = %t, want %t`, route, got, want)
		}
	}
}
