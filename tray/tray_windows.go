package tray

/*
#cgo CXXFLAGS: -std=c++11
#cgo LDFLAGS: -static -lgdiplus -lshlwapi

#include <stdlib.h>
#include "tray_windows.h"
*/
import "C"
import "unsafe"

//export trayCallback
func trayCallback(id C.int) {
	go func() {
		item := getMenuItem(uint32(id))
		if item != nil && item.Click != nil {
			item.Click(item)
		}
	}()
}

func runNative(t *Tray, onReady func(), onExit func()) {
	C.init_tray_win()

	if t.Icon != nil {
		cData := C.CBytes(t.Icon)
		C.set_icon_win((*C.char)(cData), C.int(len(t.Icon)))
		C.free(cData)
	}

	if t.Tooltip != "" {
		cTooltip := C.CString(t.Tooltip)
		C.set_tooltip_win(cTooltip)
		C.free(unsafe.Pointer(cTooltip))
	}

	if t.Menu != nil {
		buildMenuNative(t.Menu, 0)
	}

	if onReady != nil {
		go onReady()
	}

	C.run_loop_win()

	if onExit != nil {
		onExit()
	}
}

func buildMenuNative(menu *Menu, parentId int) {
	for _, item := range menu.Items {
		if item.IsSeparator {
			C.add_separator_win(C.int(parentId))
			continue
		}

		cLabel := C.CString(item.Label)
		cShortcut := C.CString(item.Shortcut)
		disabled := 0
		if item.Disabled {
			disabled = 1
		}
		checked := 0
		if item.Checked {
			checked = 1
		}
		isSubmenu := 0
		if item.SubMenu != nil {
			isSubmenu = 1
		}

		var cImgData unsafe.Pointer
		var imgLen int
		if len(item.Image) > 0 {
			cImgData = C.CBytes(item.Image)
			imgLen = len(item.Image)
		}

		C.add_menu_item_win(C.int(item.ID), cLabel, cShortcut, C.int(disabled), C.int(checked), C.int(parentId), C.int(isSubmenu), (*C.char)(cImgData), C.int(imgLen))
		C.free(unsafe.Pointer(cLabel))
		C.free(unsafe.Pointer(cShortcut))
		if cImgData != nil {
			C.free(cImgData)
		}

		if item.SubMenu != nil {
			buildMenuNative(item.SubMenu, int(item.ID))
		}
	}
}

func setupNative(t *Tray) {
	go runNative(t, nil, nil)
}

func quitNative() {
	C.quit_app_win()
}

func setIconNative(icon []byte) {
	cData := C.CBytes(icon)
	defer C.free(cData)
	C.set_icon_win((*C.char)(cData), C.int(len(icon)))
}

func setTemplateIconNative(icon []byte) {
	// Windows doesn't strictly support template icons like macOS.
	// Treat as normal icon.
	setIconNative(icon)
}

func setTitleNative(title string) {
	// Windows tray icon doesn't support title next to icon (only tooltip).
	// Ignore.
}

func setTooltipNative(tooltip string) {
	cTooltip := C.CString(tooltip)
	defer C.free(unsafe.Pointer(cTooltip))
	C.set_tooltip_win(cTooltip)
}

func setMenuItemLabelNative(id uint32, label string) {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	C.set_item_label_win(C.int(id), cLabel)
}

func setMenuItemTooltipNative(id uint32, tooltip string) {
	cTooltip := C.CString(tooltip)
	defer C.free(unsafe.Pointer(cTooltip))
	C.set_item_tooltip_win(C.int(id), cTooltip)
}

func setMenuItemCheckedNative(id uint32, checked bool) {
	cChecked := 0
	if checked {
		cChecked = 1
	}
	C.set_item_checked_win(C.int(id), C.int(cChecked))
}

func setMenuItemDisabledNative(id uint32, disabled bool) {
	cDisabled := 0
	if disabled {
		cDisabled = 1
	}
	C.set_item_disabled_win(C.int(id), C.int(cDisabled))
}
