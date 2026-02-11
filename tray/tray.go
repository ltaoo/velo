package tray

import (
	"sync"
	"sync/atomic"
)

// Tray defines the configuration for the system tray.
type Tray struct {
	// Icon is the icon data in bytes.
	Icon []byte
	// Title is the title displayed next to the icon (macOS only).
	Title string
	// Tooltip is the tooltip text (Windows/Linux).
	Tooltip string
	// IsTemplate specifies if the icon is a template icon (macOS only).
	IsTemplate bool
	// Menu is the context menu.
	Menu *Menu
	// OnLeftClick handles the left click event (Windows only).
	OnLeftClick func()
	// OnRightClick handles the right click event (Windows only).
	OnRightClick func()
}

// Menu represents a list of menu items.
type Menu struct {
	Items []*MenuItem
}

// MenuItem represents a single item in the menu.
type MenuItem struct {
	// ID is the unique identifier for the menu item.
	ID uint32
	// Label is the text displayed for the item.
	Label string
	// Tooltip is the hover text.
	Tooltip string
	// Disabled disables the item.
	Disabled bool
	// Checked adds a checkmark.
	Checked bool
	// Shortcut represents the keyboard shortcut (e.g., "Cmd+S", "Ctrl+Shift+P").
	Shortcut string
	// IsSeparator indicates if this item is a separator.
	IsSeparator bool
	// Click handles the click event.
	Click func(*MenuItem)
	// SubMenu defines a submenu for this item.
	SubMenu *Menu
}

// NewTray creates a new Tray configuration.
func NewTray() *Tray {
	return &Tray{}
}

// Run starts the system tray application.
// This function blocks until the application exits.
func Run(t *Tray, onReady func(), onExit func()) {
	// Assign IDs to menu items if not already assigned
	if t.Menu != nil {
		assignIDs(t.Menu)
	}

	runNative(t, onReady, onExit)
}

// Quit stops the system tray application.
func Quit() {
	quitNative()
}

// Global methods to update the tray

func (t *Tray) SetIcon(icon []byte) {
	t.Icon = icon
	setIconNative(icon)
}

func (t *Tray) SetTitle(title string) {
	t.Title = title
	setTitleNative(title)
}

func (t *Tray) SetTooltip(tooltip string) {
	t.Tooltip = tooltip
	setTooltipNative(tooltip)
}

func (t *Tray) SetTemplateIcon(icon []byte) {
	t.Icon = icon
	t.IsTemplate = true
	// For now, map template icon to normal icon setter with a flag if possible,
	// or just use setIconNative.
	// We might need a specific native method for template icons if the platform supports it differently.
	// For macOS, we can handle it in setIconNative based on a global flag or passed param.
	// But since setIconNative signature is fixed, let's assume implementation handles it
	// or we add a specific method.
	// Let's stick to simple SetIcon for now and maybe handle IsTemplate in the implementation if we store the tray instance.
	// Better: pass it to native.
	setTemplateIconNative(icon)
}

// MenuItem methods

func (m *MenuItem) SetLabel(label string) {
	m.Label = label
	setMenuItemLabelNative(m.ID, label)
}

func (m *MenuItem) SetTooltip(tooltip string) {
	m.Tooltip = tooltip
	setMenuItemTooltipNative(m.ID, tooltip)
}

func (m *MenuItem) Check() {
	m.Checked = true
	setMenuItemCheckedNative(m.ID, true)
}

func (m *MenuItem) Uncheck() {
	m.Checked = false
	setMenuItemCheckedNative(m.ID, false)
}

func (m *MenuItem) Enable() {
	m.Disabled = false
	setMenuItemDisabledNative(m.ID, false)
}

func (m *MenuItem) Disable() {
	m.Disabled = true
	setMenuItemDisabledNative(m.ID, true)
}

// Internal helpers

var (
	menuItems     = make(map[uint32]*MenuItem)
	menuItemsLock sync.RWMutex
	currentID     uint32
)

func assignIDs(menu *Menu) {
	for _, item := range menu.Items {
		if item.ID == 0 {
			item.ID = atomic.AddUint32(&currentID, 1)
		}
		menuItemsLock.Lock()
		menuItems[item.ID] = item
		menuItemsLock.Unlock()

		if item.SubMenu != nil {
			assignIDs(item.SubMenu)
		}
	}
}

func getMenuItem(id uint32) *MenuItem {
	menuItemsLock.RLock()
	defer menuItemsLock.RUnlock()
	return menuItems[id]
}
