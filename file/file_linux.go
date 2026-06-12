//go:build linux
// +build linux

package file

import "fmt"

func showFileSelectDialog(options FileSelectOptions) (string, error) {
	return "", fmt.Errorf("not support")
}
