//go:build windows

package shortcut

import "golang.design/x/hotkey"

var modifierCodeMap = map[string]hotkey.Modifier{
	"ShiftLeft":    hotkey.ModShift,
	"ControlLeft":  hotkey.ModCtrl,
	"MetaLeft":     hotkey.ModWin,
	"AltLeft":      hotkey.ModAlt,
	"ShiftRight":   hotkey.ModShift,
	"ControlRight": hotkey.ModCtrl,
	"MetaRight":    hotkey.ModWin,
	"AltRight":     hotkey.ModAlt,
}
