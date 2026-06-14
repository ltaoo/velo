//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type linuxAutoStart struct {
	appName string
}

func newPlatformAutoStart(appName string) AutoStart {
	return &linuxAutoStart{appName: appName}
}

func (a *linuxAutoStart) Enable() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}
	path := a.desktopEntryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create autostart directory: %w", err)
	}
	return os.WriteFile(path, []byte(a.desktopEntry(execPath)), 0644)
}

func (a *linuxAutoStart) Disable() error {
	if err := os.Remove(a.desktopEntryPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove autostart desktop entry: %w", err)
	}
	return nil
}

func (a *linuxAutoStart) IsEnabled() bool {
	_, err := os.Stat(a.desktopEntryPath())
	return err == nil
}

func (a *linuxAutoStart) desktopEntryPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = "."
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "autostart", safeIdentifier(a.appName)+".desktop")
}

func (a *linuxAutoStart) desktopEntry(execPath string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=%s
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, desktopEscape(displayName(a.appName)), desktopExecEscape(execPath))
}

func desktopExecEscape(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", "\\$")
	return `"` + replacer.Replace(value) + `"`
}

func desktopEscape(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", "")
	return replacer.Replace(value)
}
