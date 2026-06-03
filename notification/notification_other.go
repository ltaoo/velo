//go:build !darwin && !linux && !windows
// +build !darwin,!linux,!windows

package notification

import (
	"fmt"
	"runtime"
)

func showNative(opts Options) error {
	return fmt.Errorf("notification: unsupported platform %s", runtime.GOOS)
}
