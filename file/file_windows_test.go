//go:build windows
// +build windows

package file

import (
	"reflect"
	"testing"
	"unicode/utf16"
)

func TestWindowsFileTypePatterns(t *testing.T) {
	got := windows_file_type_patterns([]string{"png", ".jpg", "*.gif", " PNG ", "", "bad/type", "*"})
	want := []string{"*.png", "*.jpg", "*.gif"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windows_file_type_patterns() = %#v, want %#v", got, want)
	}
}

func TestWindowsFileDialogFilter(t *testing.T) {
	got := decode_windows_file_dialog_filter(windows_file_dialog_filter([]string{"png", "jpg"}))
	want := []string{"Supported files (*.png;*.jpg)", "*.png;*.jpg"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windows_file_dialog_filter() = %#v, want %#v", got, want)
	}
}

func TestWindowsFileDialogFilterDefaultsToAllFiles(t *testing.T) {
	got := decode_windows_file_dialog_filter(windows_file_dialog_filter(nil))
	want := []string{"All files (*.*)", "*.*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windows_file_dialog_filter(nil) = %#v, want %#v", got, want)
	}
}

func decode_windows_file_dialog_filter(encoded []uint16) []string {
	parts := make([]string, 0, 2)
	start := 0
	for i, value := range encoded {
		if value != 0 {
			continue
		}
		if i == start {
			break
		}
		parts = append(parts, string(utf16.Decode(encoded[start:i])))
		start = i + 1
	}
	return parts
}
