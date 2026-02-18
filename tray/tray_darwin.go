package tray

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>

void init_tray(void);
void set_icon(const char* data, int length, int isTemplate);
void set_title(const char* title);
void set_tooltip(const char* tooltip);
void add_menu_item(int id, const char* title, const char* shortcut, int disabled, int checked, int parentId, const char* imgData, int imgLen);
void add_separator(int parentId);
void run_loop(void);
void quit_app(void);

void set_item_label(int id, const char* label);
void set_item_tooltip(int id, const char* tooltip);
void set_item_checked(int id, int checked);
void set_item_disabled(int id, int disabled);
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
	C.init_tray()

	if t.Icon != nil {
		cData := C.CBytes(t.Icon)
		isTemplate := 0
		if t.IsTemplate {
			isTemplate = 1
		}
		C.set_icon((*C.char)(cData), C.int(len(t.Icon)), C.int(isTemplate))
		C.free(cData)
	}

	if t.Title != "" {
		cTitle := C.CString(t.Title)
		C.set_title(cTitle)
		C.free(unsafe.Pointer(cTitle))
	}

	if t.Tooltip != "" {
		cTooltip := C.CString(t.Tooltip)
		C.set_tooltip(cTooltip)
		C.free(unsafe.Pointer(cTooltip))
	}

	if t.Menu != nil {
		buildMenuNative(t.Menu, 0)
	}

	if onReady != nil {
		go onReady()
	}

	// This blocks
	C.run_loop()

	if onExit != nil {
		onExit()
	}
}

func buildMenuNative(menu *Menu, parentId int) {
	for _, item := range menu.Items {
		if item.IsSeparator {
			C.add_separator(C.int(parentId))
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

		var cImgData unsafe.Pointer
		var imgLen int
		if len(item.Image) > 0 {
			cImgData = C.CBytes(item.Image)
			imgLen = len(item.Image)
		}

		C.add_menu_item(C.int(item.ID), cLabel, cShortcut, C.int(disabled), C.int(checked), C.int(parentId), (*C.char)(cImgData), C.int(imgLen))
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
	C.init_tray()

	if t.Icon != nil {
		cData := C.CBytes(t.Icon)
		isTemplate := 0
		if t.IsTemplate {
			isTemplate = 1
		}
		C.set_icon((*C.char)(cData), C.int(len(t.Icon)), C.int(isTemplate))
		C.free(cData)
	}

	if t.Title != "" {
		cTitle := C.CString(t.Title)
		C.set_title(cTitle)
		C.free(unsafe.Pointer(cTitle))
	}

	if t.Tooltip != "" {
		cTooltip := C.CString(t.Tooltip)
		C.set_tooltip(cTooltip)
		C.free(unsafe.Pointer(cTooltip))
	}

	if t.Menu != nil {
		buildMenuNative(t.Menu, 0)
	}
}

func quitNative() {
	C.quit_app()
}

func setIconNative(icon []byte) {
	cData := C.CBytes(icon)
	defer C.free(cData)
	C.set_icon((*C.char)(cData), C.int(len(icon)), 0)
}

func setTemplateIconNative(icon []byte) {
	cData := C.CBytes(icon)
	defer C.free(cData)
	C.set_icon((*C.char)(cData), C.int(len(icon)), 1)
}

func setTitleNative(title string) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.set_title(cTitle)
}

func setTooltipNative(tooltip string) {
	cTooltip := C.CString(tooltip)
	defer C.free(unsafe.Pointer(cTooltip))
	C.set_tooltip(cTooltip)
}

func setMenuItemLabelNative(id uint32, label string) {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	C.set_item_label(C.int(id), cLabel)
}

func setMenuItemTooltipNative(id uint32, tooltip string) {
	cTooltip := C.CString(tooltip)
	defer C.free(unsafe.Pointer(cTooltip))
	C.set_item_tooltip(C.int(id), cTooltip)
}

func setMenuItemCheckedNative(id uint32, checked bool) {
	cChecked := 0
	if checked {
		cChecked = 1
	}
	C.set_item_checked(C.int(id), C.int(cChecked))
}

func setMenuItemDisabledNative(id uint32, disabled bool) {
	cDisabled := 0
	if disabled {
		cDisabled = 1
	}
	C.set_item_disabled(C.int(id), C.int(cDisabled))
}
