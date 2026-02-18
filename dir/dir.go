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

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
