//go:build linux
// +build linux

package notification

import (
	"fmt"
	"os/exec"
	"strconv"
)

func showNative(opts Options) error {
	args := []string{"--app-name", opts.AppName}
	if urgency := linuxUrgency(opts.Type); urgency != "" {
		args = append(args, "--urgency", urgency)
	}
	if opts.Icon != "" {
		args = append(args, "--icon", opts.Icon)
	}
	args = append(args, opts.Title)
	if opts.Body != "" {
		args = append(args, opts.Body)
	}

	notifyErr := exec.Command("notify-send", args...).Run()
	if notifyErr == nil {
		return nil
	}

	dbusArgs := []string{
		"--session",
		"--dest=org.freedesktop.Notifications",
		"--type=method_call",
		"--print-reply",
		"/org/freedesktop/Notifications",
		"org.freedesktop.Notifications.Notify",
		"string:" + opts.AppName,
		"uint32:0",
		"string:" + opts.Icon,
		"string:" + opts.Title,
		"string:" + opts.Body,
		"array:string:",
		"dict:string:variant:",
		"int32:" + strconv.Itoa(-1),
	}
	if err := exec.Command("dbus-send", dbusArgs...).Run(); err != nil {
		return fmt.Errorf("notification: notify-send failed: %w; dbus-send failed: %v", notifyErr, err)
	}
	return nil
}

func linuxUrgency(notificationType string) string {
	switch notificationType {
	case TypeError:
		return "critical"
	case TypeWarning:
		return "normal"
	case TypeSuccess, TypeInfo:
		return "low"
	default:
		return "normal"
	}
}

func cleanupNative(opts CleanupOptions) error {
	return nil
}

func permissionStatusNative() Status {
	return Status{
		Supported: true,
		Status:    "unknown",
	}
}
