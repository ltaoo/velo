//go:build windows
// +build windows

package file

import "fmt"

func showFileSelectDialog(animationType string) (string, error) {
	return "", fmt.Errorf("not support")
}
