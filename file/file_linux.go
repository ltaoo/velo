//go:build linux

package file

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func show_open_dialog(options OpenDialogOptions) ([]string, error) {
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection"}
		if options.Title != "" {
			args = append(args, "--title="+options.Title)
		}
		if options.Directory != "" {
			args = append(args, "--filename="+filepath.Clean(options.Directory)+string(filepath.Separator))
		}
		if options.CanChooseDirectories && !options.CanChooseFiles {
			args = append(args, "--directory")
		}
		if options.AllowsMultipleSelection {
			args = append(args, "--multiple", "--separator=\n")
		}
		if len(options.AllowedTypes) > 0 {
			patterns := make([]string, len(options.AllowedTypes))
			for index, allowed_type := range options.AllowedTypes {
				patterns[index] = "*." + allowed_type
			}
			args = append(args, "--file-filter="+strings.Join(patterns, " "))
		}
		return run_open_command("zenity", args)
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		command := "--getopenfilename"
		if options.CanChooseDirectories && !options.CanChooseFiles {
			command = "--getexistingdirectory"
		}
		args := []string{command, options.Directory}
		if len(options.AllowedTypes) > 0 && command == "--getopenfilename" {
			patterns := make([]string, len(options.AllowedTypes))
			for index, allowed_type := range options.AllowedTypes {
				patterns[index] = "*." + allowed_type
			}
			args = append(args, strings.Join(patterns, " "))
		}
		if options.AllowsMultipleSelection {
			args = append(args, "--multiple", "--separate-output")
		}
		if options.Title != "" {
			args = append(args, "--title", options.Title)
		}
		return run_open_command("kdialog", args)
	}
	return nil, fmt.Errorf("native file dialog requires zenity or kdialog")
}

func show_save_dialog(options SaveDialogOptions) (string, error) {
	initial_path := filepath.Join(options.Directory, options.DefaultFilename)
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--save", "--confirm-overwrite", "--filename=" + initial_path}
		if options.Title != "" {
			args = append(args, "--title="+options.Title)
		}
		if len(options.AllowedTypes) > 0 {
			patterns := make([]string, len(options.AllowedTypes))
			for index, allowed_type := range options.AllowedTypes {
				patterns[index] = "*." + allowed_type
			}
			args = append(args, "--file-filter="+strings.Join(patterns, " "))
		}
		paths, err := run_open_command("zenity", args)
		if err != nil {
			return "", err
		}
		return paths[0], nil
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		args := []string{"--getsavefilename", initial_path}
		if len(options.AllowedTypes) > 0 {
			patterns := make([]string, len(options.AllowedTypes))
			for index, allowed_type := range options.AllowedTypes {
				patterns[index] = "*." + allowed_type
			}
			args = append(args, strings.Join(patterns, " "))
		}
		if options.Title != "" {
			args = append(args, "--title", options.Title)
		}
		paths, err := run_open_command("kdialog", args)
		if err != nil {
			return "", err
		}
		return paths[0], nil
	}
	return "", fmt.Errorf("native file dialog requires zenity or kdialog")
}

func run_open_command(name string, args []string) ([]string, error) {
	output, err := exec.Command(name, args...).Output()
	if err != nil {
		var exit_error *exec.ExitError
		if errors.As(err, &exit_error) && exit_error.ExitCode() == 1 {
			return nil, ErrCancelled
		}
		return nil, err
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, ErrCancelled
	}
	return strings.Split(trimmed, "\n"), nil
}
