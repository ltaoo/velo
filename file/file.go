package file

import (
	"errors"
	"strings"
)

var ErrCancelled = errors.New("file dialog cancelled")

type OpenDialogOptions struct {
	Title                   string
	Directory               string
	AllowedTypes            []string
	CanChooseFiles          bool
	CanChooseDirectories    bool
	AllowsMultipleSelection bool
}

type SaveDialogOptions struct {
	Title           string
	Directory       string
	DefaultFilename string
	AllowedTypes    []string
}

// FileSelectOptions is kept for backward compatibility.
type FileSelectOptions struct {
	AnimationType string
	AllowedTypes  []string
	Directory     string
}

func ShowOpenDialog(options OpenDialogOptions) ([]string, error) {
	if !options.CanChooseFiles && !options.CanChooseDirectories {
		options.CanChooseFiles = true
	}
	options.AllowedTypes = normalize_allowed_types(options.AllowedTypes)
	return show_open_dialog(options)
}

func ShowSaveDialog(options SaveDialogOptions) (string, error) {
	options.AllowedTypes = normalize_allowed_types(options.AllowedTypes)
	return show_save_dialog(options)
}

// ShowFileSelectDialog shows a single-file selection dialog.
func ShowFileSelectDialog(animation_type string) (string, error) {
	return ShowFileSelectDialogWithOptions(FileSelectOptions{AnimationType: animation_type})
}

// ShowFileSelectDialogWithTypes shows a single-file selection dialog with extension filtering.
func ShowFileSelectDialogWithTypes(animation_type string, allowed_types []string) (string, error) {
	return ShowFileSelectDialogWithOptions(FileSelectOptions{AnimationType: animation_type, AllowedTypes: allowed_types})
}

func ShowFileSelectDialogWithOptions(options FileSelectOptions) (string, error) {
	paths, err := ShowOpenDialog(OpenDialogOptions{
		Directory:      options.Directory,
		AllowedTypes:   options.AllowedTypes,
		CanChooseFiles: true,
	})
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", ErrCancelled
	}
	return paths[0], nil
}

func normalize_allowed_types(types []string) []string {
	result := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	for _, item := range types {
		item = strings.ToLower(strings.TrimSpace(item))
		item = strings.TrimPrefix(item, "*.")
		item = strings.TrimPrefix(item, ".")
		if item == "" || item == "*" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
