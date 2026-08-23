//go:build windows
// +build windows

package file

import (
	"reflect"
	"testing"
	"unicode/utf16"
)

func TestWindowsFileTypePatterns(t *testing.T) {
	got := windowsFileTypePatterns([]string{"png", ".jpg", "*.gif", " PNG ", "", "bad/type", "*"})
	want := []string{"*.png", "*.jpg", "*.gif"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windowsFileTypePatterns() = %#v, want %#v", got, want)
	}
}

func TestWindowsFileDialogFilter(t *testing.T) {
	got := decodeWindowsFileDialogFilter(windowsFileDialogFilter([]string{"png", "jpg"}))
	want := []string{"Supported files (*.png;*.jpg)", "*.png;*.jpg"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windowsFileDialogFilter() = %#v, want %#v", got, want)
	}
}

func TestWindowsFileDialogFilterDefaultsToAllFiles(t *testing.T) {
	got := decodeWindowsFileDialogFilter(windowsFileDialogFilter(nil))
	want := []string{"All files (*.*)", "*.*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windowsFileDialogFilter(nil) = %#v, want %#v", got, want)
	}
}

func decodeWindowsFileDialogFilter(encoded []uint16) []string {
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
