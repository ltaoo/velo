//go:build windows

package file

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ofn_explorer         = 0x00080000
	ofn_file_must_exist  = 0x00001000
	ofn_path_must_exist  = 0x00000800
	ofn_allow_multi      = 0x00000200
	ofn_overwrite_prompt = 0x00000002
	ofn_hide_read_only   = 0x00000004
	ofn_no_change_dir    = 0x00000008
	ofn_enable_sizing    = 0x00800000
	bif_return_only_dirs = 0x00000001
	bif_new_dialog_style = 0x00000040
)

var (
	comdlg32_dll              = windows.NewLazySystemDLL("comdlg32.dll")
	shell32_dll               = windows.NewLazySystemDLL("shell32.dll")
	ole32_dll                 = windows.NewLazySystemDLL("ole32.dll")
	get_open_file_name_proc   = comdlg32_dll.NewProc("GetOpenFileNameW")
	get_save_file_name_proc   = comdlg32_dll.NewProc("GetSaveFileNameW")
	comm_dialog_error_proc    = comdlg32_dll.NewProc("CommDlgExtendedError")
	sh_browse_for_folder_proc = shell32_dll.NewProc("SHBrowseForFolderW")
	sh_get_path_proc          = shell32_dll.NewProc("SHGetPathFromIDListW")
	co_task_mem_free_proc     = ole32_dll.NewProc("CoTaskMemFree")
)

type open_file_name struct {
	struct_size       uint32
	owner             uintptr
	instance          uintptr
	filter            *uint16
	custom_filter     *uint16
	max_custom_filter uint32
	filter_index      uint32
	file              *uint16
	max_file          uint32
	file_title        *uint16
	max_file_title    uint32
	initial_directory *uint16
	title             *uint16
	flags             uint32
	file_offset       uint16
	file_extension    uint16
	default_extension *uint16
	custom_data       uintptr
	hook              uintptr
	template_name     *uint16
	reserved          unsafe.Pointer
	reserved_flags    uint32
	flags_ex          uint32
}

type browse_info struct {
	owner        uintptr
	root         uintptr
	display_name *uint16
	title        *uint16
	flags        uint32
	callback     uintptr
	parameter    uintptr
	image        int32
}

func show_open_dialog(options OpenDialogOptions) ([]string, error) {
	if options.CanChooseDirectories && !options.CanChooseFiles {
		path, err := choose_directory(options.Title)
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	return choose_files(options)
}

func show_save_dialog(options SaveDialogOptions) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	buffer := make([]uint16, 32768)
	copy(buffer, utf16.Encode([]rune(options.DefaultFilename)))
	dialog, filter, err := new_open_file_name(buffer, options.Title, options.Directory, options.AllowedTypes)
	if err != nil {
		return "", err
	}
	dialog.flags = common_dialog_flags() | ofn_overwrite_prompt
	result, _, _ := get_save_file_name_proc.Call(uintptr(unsafe.Pointer(&dialog)))
	runtime.KeepAlive(filter)
	if result == 0 {
		return "", dialog_error()
	}
	return windows.UTF16ToString(buffer), nil
}

func choose_files(options OpenDialogOptions) ([]string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	buffer := make([]uint16, 32768)
	dialog, filter, err := new_open_file_name(buffer, options.Title, options.Directory, options.AllowedTypes)
	if err != nil {
		return nil, err
	}
	dialog.flags = common_dialog_flags() | ofn_file_must_exist
	if options.AllowsMultipleSelection {
		dialog.flags |= ofn_allow_multi
	}
	result, _, _ := get_open_file_name_proc.Call(uintptr(unsafe.Pointer(&dialog)))
	runtime.KeepAlive(filter)
	if result == 0 {
		return nil, dialog_error()
	}
	parts := split_utf16(buffer)
	if len(parts) <= 1 {
		return parts, nil
	}
	paths := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		paths = append(paths, filepath.Join(parts[0], name))
	}
	return paths, nil
}

func choose_directory(title string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	display_name := make([]uint16, windows.MAX_PATH)
	title_pointer, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", err
	}
	info := browse_info{
		display_name: &display_name[0],
		title:        title_pointer,
		flags:        bif_return_only_dirs | bif_new_dialog_style,
	}
	item, _, _ := sh_browse_for_folder_proc.Call(uintptr(unsafe.Pointer(&info)))
	if item == 0 {
		return "", ErrCancelled
	}
	defer co_task_mem_free_proc.Call(item)
	path := make([]uint16, windows.MAX_PATH)
	result, _, _ := sh_get_path_proc.Call(item, uintptr(unsafe.Pointer(&path[0])))
	if result == 0 {
		return "", fmt.Errorf("failed to read selected directory")
	}
	return windows.UTF16ToString(path), nil
}

func new_open_file_name(buffer []uint16, title, directory string, allowed_types []string) (open_file_name, []uint16, error) {
	title_pointer, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return open_file_name{}, nil, fmt.Errorf("file dialog: invalid title: %w", err)
	}
	directory_pointer, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return open_file_name{}, nil, fmt.Errorf("file dialog: invalid initial directory: %w", err)
	}
	filter := windows_file_dialog_filter(allowed_types)
	return open_file_name{
		struct_size:       uint32(unsafe.Sizeof(open_file_name{})),
		filter:            &filter[0],
		filter_index:      1,
		file:              &buffer[0],
		max_file:          uint32(len(buffer)),
		initial_directory: directory_pointer,
		title:             title_pointer,
	}, filter, nil
}

func common_dialog_flags() uint32 {
	return ofn_explorer | ofn_path_must_exist | ofn_hide_read_only | ofn_no_change_dir | ofn_enable_sizing
}

func windows_file_type_patterns(allowed_types []string) []string {
	patterns := make([]string, 0, len(allowed_types))
	seen := make(map[string]struct{}, len(allowed_types))
	for _, allowed_type := range allowed_types {
		extension := strings.TrimSpace(allowed_type)
		extension = strings.TrimPrefix(extension, "*.")
		extension = strings.TrimPrefix(extension, ".")
		if extension == "" || extension == "*" || strings.ContainsAny(extension, `\/:;*?`+"\x00") {
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

func windows_file_dialog_filter(allowed_types []string) []uint16 {
	patterns := windows_file_type_patterns(allowed_types)
	pattern := "*.*"
	label := "All files (*.*)"
	if len(patterns) > 0 {
		pattern = strings.Join(patterns, ";")
		label = "Supported files (" + pattern + ")"
	}
	return append(utf16.Encode([]rune(label+"\x00"+pattern+"\x00\x00")), 0)
}

func split_utf16(buffer []uint16) []string {
	var result []string
	start := 0
	for index, value := range buffer {
		if value != 0 {
			continue
		}
		if index == start {
			break
		}
		result = append(result, string(utf16.Decode(buffer[start:index])))
		start = index + 1
	}
	return result
}

func dialog_error() error {
	code, _, _ := comm_dialog_error_proc.Call()
	if code == 0 {
		return ErrCancelled
	}
	return fmt.Errorf("native file dialog failed: 0x%x", code)
}
