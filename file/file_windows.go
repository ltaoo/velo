//go:build windows
// +build windows

package file

import "fmt"

func showFileSelectDialog(options FileSelectOptions) (string, error) {
	return "", fmt.Errorf("not support")
}
