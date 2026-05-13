package dir

import (
	"os"
	"path/filepath"
)

// Dir provides common file paths for a desktop application.
type Dir struct {
	appName string
}

// New creates a Dir for the given application name.
func New(appName string) *Dir {
	return &Dir{appName: appName}
}

// Data returns the app data directory (~/.appName) and ensures it exists.
func (d *Dir) Data() string {
	p := filepath.Join(homeDir(), "."+d.appName)
	os.MkdirAll(p, 0755)
	return p
}

// LogFile returns the path to the app log file.
func (d *Dir) LogFile() string {
	return filepath.Join(d.Data(), "app.log")
}

// UpdateStateFile returns the path to the update state file.
func (d *Dir) UpdateStateFile() string {
	return filepath.Join(d.Data(), "update_state.json")
}

// WorkDir returns the current working directory (where the command was executed).
func WorkDir() string {
	d, _ := os.Getwd()
	return d
}

// ExeDir returns the directory where the application binary is located.
// It resolves symlinks so that it works correctly even when the binary
// is launched from a macOS .app bundle (which symlinks back to the original).
func ExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return filepath.Dir(resolved)
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
