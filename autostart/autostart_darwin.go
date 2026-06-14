//go:build darwin

package autostart

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type darwinAutoStart struct {
	appName string
}

func newPlatformAutoStart(appName string) AutoStart {
	return &darwinAutoStart{appName: appName}
}

func (a *darwinAutoStart) Enable() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}
	path := a.launchAgentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create launch agent directory: %w", err)
	}
	return os.WriteFile(path, []byte(a.launchAgentPlist(execPath)), 0644)
}

func (a *darwinAutoStart) Disable() error {
	if err := os.Remove(a.launchAgentPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove launch agent: %w", err)
	}
	return nil
}

func (a *darwinAutoStart) IsEnabled() bool {
	_, err := os.Stat(a.launchAgentPath())
	return err == nil
}

func (a *darwinAutoStart) launchAgentPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, "Library", "LaunchAgents", a.launchAgentLabel()+".plist")
}

func (a *darwinAutoStart) launchAgentLabel() string {
	return "com.ltaoo.velo.autostart." + safeIdentifier(a.appName)
}

func (a *darwinAutoStart) launchAgentPlist(execPath string) string {
	args := []string{execPath}
	if appPath := macOSAppBundlePath(execPath); appPath != "" {
		args = []string{"/usr/bin/open", "-n", appPath}
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
%s
  </array>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
`, xmlEscape(a.launchAgentLabel()), plistStringArray(args))
}

func macOSAppBundlePath(execPath string) string {
	marker := ".app/Contents/MacOS/"
	idx := strings.Index(execPath, marker)
	if idx < 0 {
		return ""
	}
	appPath := execPath[:idx+len(".app")]
	if info, err := os.Stat(appPath); err == nil && info.IsDir() {
		return appPath
	}
	return ""
}

func plistStringArray(values []string) string {
	var b strings.Builder
	for _, value := range values {
		b.WriteString("    <string>")
		b.WriteString(xmlEscape(value))
		b.WriteString("</string>\n")
	}
	return b.String()
}

func xmlEscape(value string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(value))
	return b.String()
}
