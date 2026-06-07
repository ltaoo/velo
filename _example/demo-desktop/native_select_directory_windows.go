//go:build windows
// +build windows

package main

import "fmt"

func selectVaultDirectory() (string, error) {
	return "", fmt.Errorf("directory picker is not implemented on Windows")
}
