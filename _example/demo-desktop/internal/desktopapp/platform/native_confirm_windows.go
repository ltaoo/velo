//go:build windows
// +build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	externalConfirmUser32      = syscall.NewLazyDLL("user32.dll")
	externalConfirmMessageBoxW = externalConfirmUser32.NewProc("MessageBoxW")
)

const (
	externalConfirmIDYES         = 6
	externalConfirmMBYesNo       = 0x00000004
	externalConfirmMBIconWarning = 0x00000030
	externalConfirmMBTaskModal   = 0x00002000
	externalConfirmMBSetFg       = 0x00010000
)

func ConfirmExternalBrowserOpen(message string) (bool, error) {
	title, err := syscall.UTF16PtrFromString("打开外部链接")
	if err != nil {
		return false, err
	}
	text, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return false, err
	}
	result, _, _ := externalConfirmMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		uintptr(externalConfirmMBYesNo|externalConfirmMBIconWarning|externalConfirmMBTaskModal|externalConfirmMBSetFg),
	)
	return result == externalConfirmIDYES, nil
}
