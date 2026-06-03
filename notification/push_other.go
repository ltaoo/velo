//go:build !darwin
// +build !darwin

package notification

import (
	"fmt"
	"runtime"
)

func registerRemotePushNative(callbacks RemotePushCallbacks) error {
	return fmt.Errorf("notification: remote push is unsupported on %s", runtime.GOOS)
}
