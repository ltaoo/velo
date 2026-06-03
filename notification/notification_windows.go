//go:build windows
// +build windows

package notification

import (
	"encoding/xml"
	"fmt"
	"os/exec"
)

func showNative(opts Options) error {
	title := xmlEscape(opts.Title)
	body := xmlEscape(opts.Body)
	appID := opts.AppName
	if appID == "" {
		appID = "Velo"
	}

	script := `& {
param([string]$AppId, [string]$Title, [string]$Body)
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml("<toast><visual><binding template=""ToastGeneric""><text>$Title</text><text>$Body</text></binding></visual></toast>")
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($AppId)
$notifier.Show($toast)
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script, appID, title, body)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("notification: powershell toast failed: %w: %s", err, string(out))
	}
	return nil
}

func cleanupNative(opts CleanupOptions) error {
	appID := opts.AppName
	if appID == "" {
		appID = "Velo"
	}

	script := `& {
param([string]$AppId)
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.UI.Notifications.ToastNotificationManager]::History.Clear($AppId)
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script, appID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("notification: powershell cleanup failed: %w: %s", err, string(out))
	}
	return nil
}

func permissionStatusNative() Status {
	return Status{
		Supported: true,
		Status:    "unknown",
	}
}

func xmlEscape(s string) string {
	var out []byte
	if err := xml.EscapeText((*sliceWriter)(&out), []byte(s)); err != nil {
		return ""
	}
	return string(out)
}

type sliceWriter []byte

func (w *sliceWriter) Write(p []byte) (int, error) {
	*w = append(*w, p...)
	return len(p), nil
}
