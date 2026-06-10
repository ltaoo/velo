// Package inputsource exposes a small cross-platform API for reading and
// selecting keyboard input sources.
//
// On macOS this wraps Text Input Services (TIS) and NSWorkspace. On Windows it
// wraps User32 keyboard-layout APIs for the current foreground window. Other
// platforms return ErrUnsupported.
package inputsource

import "errors"

// ErrUnsupported is returned when the current platform has no implementation.
var ErrUnsupported = errors.New("inputsource: unsupported platform")

// Source describes a selectable keyboard input source.
//
// ID is the stable platform identifier to pass to Select. On macOS this is the
// TIS input source ID. On Windows this is the HKL/KLID rendered as eight hex
// digits, for example "00000409".
type Source struct {
	ID         string
	Name       string
	Language   string
	Enabled    bool
	Selectable bool
}

// App describes the current foreground application/window owner.
//
// ID is the value rules should normally match. On macOS it is the bundle
// identifier. On Windows it is the executable path when available.
type App struct {
	ID   string
	Name string
	PID  int
}

// List returns selectable input sources known to the operating system.
func List() ([]Source, error) {
	return list()
}

// Current returns the input source active for the current keyboard focus.
func Current() (Source, error) {
	return current()
}

// Select switches the current keyboard focus to sourceID.
func Select(sourceID string) error {
	return selectSource(sourceID)
}

// FrontmostApp returns the current foreground application/window owner.
func FrontmostApp() (App, error) {
	return frontmostApp()
}
