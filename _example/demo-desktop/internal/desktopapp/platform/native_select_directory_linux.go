//go:build linux
// +build linux

package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

func SelectVaultDirectory() (string, error) {
	commands := [][]string{
		{"zenity", "--file-selection", "--directory", "--title=Select Velo vault"},
		{"kdialog", "--getexistingdirectory", ".", "--title", "Select Velo vault"},
	}
	for _, candidate := range commands {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		out, err := exec.Command(candidate[0], candidate[1:]...).Output()
		if err != nil {
			return "", err
		}
		path := strings.TrimSpace(string(out))
		if path == "" {
			return "", fmt.Errorf("cancelled")
		}
		return path, nil
	}
	return "", fmt.Errorf("directory picker is not available on this system")
}
