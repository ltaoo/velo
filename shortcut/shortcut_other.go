//go:build !darwin && !windows

package shortcut

import "golang.design/x/hotkey"

var modifierCodeMap = map[string]hotkey.Modifier{}
