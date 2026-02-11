//go:build windows
// +build windows

package error

import (
	"syscall"
	"unsafe"
)

var (
	user32       = syscall.NewLazyDLL("user32.dll")
	messageBoxW  = user32.NewProc("MessageBoxW")
	MB_OK        = 0x00000000
	MB_ICONERROR = 0x00000010
)

// showErrorDialog shows a native error dialog on Windows
func showErrorDialog(message string) {
	title, _ := syscall.UTF16PtrFromString("Application Error")
	text, _ := syscall.UTF16PtrFromString(message)
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		uintptr(MB_OK|MB_ICONERROR),
	)
}
