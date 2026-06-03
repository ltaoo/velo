// Package notification provides system-level desktop notifications.
package notification

import "errors"

const (
	TypeInfo    = "info"
	TypeSuccess = "success"
	TypeWarning = "warning"
	TypeError   = "error"
)

// Options configures a system notification.
type Options struct {
	// Type classifies the message. Supported values are "info", "success",
	// "warning", and "error". Platforms apply the closest native behavior.
	Type string
	// Title is the primary notification text.
	Title string
	// Body is the secondary notification text.
	Body string
	// AppName identifies the sending application where the platform allows it.
	AppName string
	// Icon is an optional icon path on platforms that support it.
	Icon string
	// Sound requests the platform default notification sound where supported.
	Sound bool
}

// Status describes the platform notification permission state where available.
type Status struct {
	Supported     bool   `json:"supported"`
	Status        string `json:"status"`
	BundleID      string `json:"bundle_id,omitempty"`
	BundlePath    string `json:"bundle_path,omitempty"`
	Notifications string `json:"notifications,omitempty"`
}

// CleanupOptions configures notification cleanup for uninstall flows.
type CleanupOptions struct {
	// AppName identifies the application on platforms that need an app id.
	AppName string
}

// Show displays a system-level notification using the native platform backend.
func Show(opts Options) error {
	if opts.Title == "" && opts.Body == "" {
		return errors.New("notification: title or body is required")
	}
	if opts.AppName == "" {
		opts.AppName = "Velo"
	}
	if opts.Type == "" {
		opts.Type = TypeInfo
	}
	return showNative(opts)
}

// Push is an alias for Show.
func Push(opts Options) error {
	return Show(opts)
}

// PermissionStatus returns the platform notification permission state.
func PermissionStatus() Status {
	return permissionStatusNative()
}

// Cleanup removes pending and delivered notifications where supported.
//
// Desktop operating systems generally do not provide a public API for an app to
// revoke its own notification permission. Users or uninstallers must handle that
// through the platform's settings or installer-specific cleanup.
func Cleanup(opts CleanupOptions) error {
	if opts.AppName == "" {
		opts.AppName = "Velo"
	}
	return cleanupNative(opts)
}
