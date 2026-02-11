package tray

func runNative(t *Tray, onReady func(), onExit func()) {
	// Not implemented
	if onReady != nil {
		onReady()
	}
	// Block forever or until exit?
	// Since it's a stub, maybe just block to simulate running.
	select {}
}

func quitNative() {}

func setIconNative(icon []byte) {}

func setTemplateIconNative(icon []byte) {}

func setTitleNative(title string) {}

func setTooltipNative(tooltip string) {}

func setMenuItemLabelNative(id uint32, label string) {}

func setMenuItemTooltipNative(id uint32, tooltip string) {}

func setMenuItemCheckedNative(id uint32, checked bool) {}

func setMenuItemDisabledNative(id uint32, disabled bool) {}
