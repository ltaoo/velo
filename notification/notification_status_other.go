//go:build !darwin && !windows && !linux
// +build !darwin,!windows,!linux

package notification

func permissionStatusNative() Status {
	return Status{
		Supported: true,
		Status:    "unknown",
	}
}

func cleanupNative(opts CleanupOptions) error {
	return nil
}
