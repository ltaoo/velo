//go:build windows

package inputsource

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmInputLangChangeRequest       = 0x0050
	klfActivate                    = 0x00000001
	klfSubstituteOK                = 0x00000002
	processQueryLimitedInformation = 0x1000
)

var (
	user32                         = windows.NewLazySystemDLL("user32.dll")
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procGetKeyboardLayoutList      = user32.NewProc("GetKeyboardLayoutList")
	procGetKeyboardLayout          = user32.NewProc("GetKeyboardLayout")
	procLoadKeyboardLayoutW        = user32.NewProc("LoadKeyboardLayoutW")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procPostMessageW               = user32.NewProc("PostMessageW")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
)

func list() ([]Source, error) {
	count, _, err := procGetKeyboardLayoutList.Call(0, 0)
	if count == 0 {
		if err != syscall.Errno(0) {
			return nil, fmt.Errorf("inputsource: GetKeyboardLayoutList failed: %w", err)
		}
		return nil, nil
	}

	layouts := make([]uintptr, int(count))
	got, _, err := procGetKeyboardLayoutList.Call(
		uintptr(len(layouts)),
		uintptr(unsafe.Pointer(&layouts[0])),
	)
	if got == 0 {
		if err != syscall.Errno(0) {
			return nil, fmt.Errorf("inputsource: GetKeyboardLayoutList failed: %w", err)
		}
		return nil, errors.New("inputsource: GetKeyboardLayoutList returned no layouts")
	}

	sources := make([]Source, 0, int(got))
	for _, layout := range layouts[:int(got)] {
		sources = append(sources, sourceFromHKL(layout))
	}
	return sources, nil
}

func current() (Source, error) {
	hwnd := foregroundWindow()
	var threadID uintptr
	if hwnd != 0 {
		threadID, _, _ = procGetWindowThreadProcessId.Call(hwnd, 0)
	}
	layout, _, err := procGetKeyboardLayout.Call(threadID)
	if layout == 0 {
		if err != syscall.Errno(0) {
			return Source{}, fmt.Errorf("inputsource: GetKeyboardLayout failed: %w", err)
		}
		return Source{}, errors.New("inputsource: GetKeyboardLayout returned no layout")
	}
	return sourceFromHKL(layout), nil
}

func selectSource(sourceID string) error {
	layout, err := parseHKL(sourceID)
	if err != nil {
		return err
	}

	if layout == 0 {
		ptr, err := windows.UTF16PtrFromString(sourceID)
		if err != nil {
			return fmt.Errorf("inputsource: invalid source ID %q: %w", sourceID, err)
		}
		loaded, _, callErr := procLoadKeyboardLayoutW.Call(
			uintptr(unsafe.Pointer(ptr)),
			uintptr(klfActivate|klfSubstituteOK),
		)
		if loaded == 0 {
			if callErr != syscall.Errno(0) {
				return fmt.Errorf("inputsource: LoadKeyboardLayoutW failed: %w", callErr)
			}
			return fmt.Errorf("inputsource: LoadKeyboardLayoutW failed for %q", sourceID)
		}
		layout = loaded
	}

	hwnd := foregroundWindow()
	if hwnd == 0 {
		return errors.New("inputsource: no foreground window")
	}

	ok, _, callErr := procPostMessageW.Call(
		hwnd,
		uintptr(wmInputLangChangeRequest),
		0,
		layout,
	)
	if ok == 0 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("inputsource: WM_INPUTLANGCHANGEREQUEST failed: %w", callErr)
		}
		return errors.New("inputsource: WM_INPUTLANGCHANGEREQUEST failed")
	}
	return nil
}

func frontmostApp() (App, error) {
	hwnd := foregroundWindow()
	if hwnd == 0 {
		return App{}, errors.New("inputsource: no foreground window")
	}

	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return App{}, errors.New("inputsource: foreground window has no process ID")
	}

	path := processImagePath(pid)
	title := windowText(hwnd)
	name := title
	if path != "" {
		name = filepath.Base(path)
	}
	id := path
	if id == "" {
		id = fmt.Sprintf("pid:%d", pid)
	}

	return App{ID: id, Name: name, PID: int(pid)}, nil
}

func foregroundWindow() uintptr {
	hwnd, _, _ := procGetForegroundWindow.Call()
	return hwnd
}

func sourceFromHKL(layout uintptr) Source {
	id := fmt.Sprintf("%08X", uint32(layout))
	langID := uint16(layout & 0xffff)
	return Source{
		ID:         id,
		Name:       id,
		Language:   fmt.Sprintf("0x%04X", langID),
		Enabled:    true,
		Selectable: true,
	}
}

func parseHKL(sourceID string) (uintptr, error) {
	id := strings.TrimSpace(sourceID)
	if id == "" {
		return 0, errors.New("inputsource: source ID is empty")
	}
	id = strings.TrimPrefix(strings.ToUpper(id), "0X")
	if len(id) > 8 {
		return 0, fmt.Errorf("inputsource: invalid HKL/KLID %q", sourceID)
	}
	value, err := strconv.ParseUint(id, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("inputsource: invalid HKL/KLID %q: %w", sourceID, err)
	}
	return uintptr(value), nil
}

func processImagePath(pid uint32) string {
	handle, err := windows.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)

	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	ok, _, _ := procQueryFullProcessImageNameW.Call(
		uintptr(handle),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ok == 0 || size == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

func windowText(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, int(length)+1)
	got, _, _ := procGetWindowTextW.Call(
		hwnd,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if got == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:got])
}
