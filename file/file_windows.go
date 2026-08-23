//go:build windows
// +build windows

package file

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ofnHideReadOnly  = 0x00000004
	ofnNoChangeDir   = 0x00000008
	ofnPathMustExist = 0x00000800
	ofnFileMustExist = 0x00001000
	ofnExplorer      = 0x00080000
	ofnEnableSizing  = 0x00800000
)

var (
	comdlg32                 = windows.NewLazySystemDLL("comdlg32.dll")
	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")
)

// openFileName mirrors OPENFILENAMEW. Pointer-sized fields keep the structure
// layout correct on both 32-bit and 64-bit Windows.
type openFileName struct {
	structSize       uint32
	owner            uintptr
	instance         uintptr
	filter           *uint16
	customFilter     *uint16
	maxCustomFilter  uint32
	filterIndex      uint32
	file             *uint16
	maxFile          uint32
	fileTitle        *uint16
	maxFileTitle     uint32
	initialDirectory *uint16
	title            *uint16
	flags            uint32
	fileOffset       uint16
	fileExtension    uint16
	defaultExtension *uint16
	customData       uintptr
	hook             uintptr
	templateName     *uint16
	reserved         unsafe.Pointer
	reservedFlags    uint32
	flagsEx          uint32
}

func showFileSelectDialog(options FileSelectOptions) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fileBuffer := make([]uint16, 32768)
	filterBuffer := windowsFileDialogFilter(options.AllowedTypes)

	var initialDirectory *uint16
	if options.Directory != "" {
		var err error
		initialDirectory, err = windows.UTF16PtrFromString(options.Directory)
		if err != nil {
			return "", fmt.Errorf("file picker: invalid initial directory: %w", err)
		}
	}

	dialog := openFileName{
		filter:           &filterBuffer[0],
		filterIndex:      1,
		file:             &fileBuffer[0],
		maxFile:          uint32(len(fileBuffer)),
		initialDirectory: initialDirectory,
		flags: ofnExplorer |
			ofnEnableSizing |
			ofnFileMustExist |
			ofnPathMustExist |
			ofnHideReadOnly |
			ofnNoChangeDir,
	}
	dialog.structSize = uint32(unsafe.Sizeof(dialog))

	ok, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&dialog)))
	runtime.KeepAlive(filterBuffer)
	runtime.KeepAlive(fileBuffer)
	if ok == 0 {
		code, _, _ := procCommDlgExtendedError.Call()
		if code == 0 {
			return "", errors.New("cancelled")
		}
		return "", fmt.Errorf("file picker: GetOpenFileNameW failed with code 0x%04X", code)
	}

	path := windows.UTF16ToString(fileBuffer)
	if path == "" {
		return "", errors.New("file picker: selected path is empty")
	}
	return path, nil
}

func windowsFileDialogFilter(allowedTypes []string) []uint16 {
	patterns := windowsFileTypePatterns(allowedTypes)
	parts := []string{"All files (*.*)", "*.*"}
	if len(patterns) > 0 {
		specification := strings.Join(patterns, ";")
		parts = []string{"Supported files (" + specification + ")", specification}
	}

	var encoded []uint16
	for _, part := range parts {
		encoded = append(encoded, utf16.Encode([]rune(part))...)
		encoded = append(encoded, 0)
	}
	// Windows filter lists end with two NUL characters.
	return append(encoded, 0)
}

func windowsFileTypePatterns(allowedTypes []string) []string {
	patterns := make([]string, 0, len(allowedTypes))
	seen := make(map[string]struct{}, len(allowedTypes))
	for _, allowedType := range allowedTypes {
		extension := strings.TrimSpace(allowedType)
		extension = strings.TrimPrefix(extension, "*.")
		extension = strings.TrimPrefix(extension, ".")
		if extension == "" || strings.ContainsAny(extension, "\\/:;*?\x00") {
			continue
		}

		key := strings.ToLower(extension)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		patterns = append(patterns, "*."+extension)
	}
	return patterns
}
