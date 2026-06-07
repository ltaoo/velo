//go:build !darwin

package main

import (
	"errors"
	"runtime"
)

type keyboardState struct {
	Supported         bool   `json:"supported"`
	OS                string `json:"os"`
	Disabled          bool   `json:"disabled"`
	PermissionGranted bool   `json:"permission_granted"`
	EventTapReady     bool   `json:"event_tap_ready"`
}

func readKeyboardState() keyboardState {
	return keyboardState{
		Supported: false,
		OS:        runtime.GOOS,
	}
}

func disableKeyboard() (keyboardState, error) {
	return readKeyboardState(), errors.New("键盘禁用功能仅支持 macOS")
}

func enableKeyboard() (keyboardState, error) {
	return readKeyboardState(), errors.New("键盘启用功能仅支持 macOS")
}
