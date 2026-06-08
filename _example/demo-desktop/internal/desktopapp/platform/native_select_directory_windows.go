//go:build windows
// +build windows

package platform

import "fmt"

func SelectVaultDirectory() (string, error) {
	return "", fmt.Errorf("directory picker is not implemented on Windows")
}
