//go:build darwin

package autostart

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework ServiceManagement
#include "autostart_darwin.h"
*/
import "C"
import "fmt"

type darwinAutoStart struct {
	appName string
}

func newPlatformAutoStart(appName string) AutoStart {
	return &darwinAutoStart{appName: appName}
}

func (a *darwinAutoStart) Enable() error {
	if C.enableLoginItem() == 0 {
		return fmt.Errorf("failed to enable login item")
	}
	return nil
}

func (a *darwinAutoStart) Disable() error {
	if C.disableLoginItem() == 0 {
		return fmt.Errorf("failed to disable login item")
	}
	return nil
}

func (a *darwinAutoStart) IsEnabled() bool {
	return C.isLoginItemEnabled() == 1
}
