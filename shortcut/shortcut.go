package shortcut

import (
	"fmt"
	"strings"
	"sync"

	"golang.design/x/hotkey"
)

var keyCodeMap = map[string]hotkey.Key{
	"Escape":     hotkey.KeyEscape,
	"Digit1":     hotkey.Key1,
	"Digit2":     hotkey.Key2,
	"Digit3":     hotkey.Key3,
	"Digit4":     hotkey.Key4,
	"Digit5":     hotkey.Key5,
	"Digit6":     hotkey.Key6,
	"Digit7":     hotkey.Key7,
	"Digit8":     hotkey.Key8,
	"Digit9":     hotkey.Key9,
	"Digit0":     hotkey.Key0,
	"KeyQ":       hotkey.KeyQ,
	"KeyW":       hotkey.KeyW,
	"KeyE":       hotkey.KeyE,
	"KeyR":       hotkey.KeyR,
	"KeyT":       hotkey.KeyT,
	"KeyY":       hotkey.KeyY,
	"KeyU":       hotkey.KeyU,
	"KeyI":       hotkey.KeyI,
	"KeyO":       hotkey.KeyO,
	"KeyP":       hotkey.KeyP,
	"KeyA":       hotkey.KeyA,
	"KeyS":       hotkey.KeyS,
	"KeyD":       hotkey.KeyD,
	"KeyF":       hotkey.KeyF,
	"KeyG":       hotkey.KeyG,
	"KeyH":       hotkey.KeyH,
	"KeyJ":       hotkey.KeyJ,
	"KeyK":       hotkey.KeyK,
	"KeyL":       hotkey.KeyL,
	"KeyZ":       hotkey.KeyZ,
	"KeyX":       hotkey.KeyX,
	"KeyC":       hotkey.KeyC,
	"KeyV":       hotkey.KeyV,
	"KeyB":       hotkey.KeyB,
	"KeyN":       hotkey.KeyN,
	"KeyM":       hotkey.KeyM,
	"Space":      hotkey.KeySpace,
	"Tab":        hotkey.KeyTab,
	"ArrowUp":    hotkey.KeyUp,
	"ArrowDown":  hotkey.KeyDown,
	"ArrowLeft":  hotkey.KeyLeft,
	"ArrowRight": hotkey.KeyRight,
}

// Manager manages global shortcut registrations.
type Manager struct {
	mu      sync.Mutex
	hotkeys map[string]*hotkey.Hotkey
}

// NewManager creates a new shortcut manager.
func NewManager() *Manager {
	return &Manager{hotkeys: make(map[string]*hotkey.Hotkey)}
}

// Register registers a global shortcut and calls handler on each trigger.
// The shortcut format is "Modifier+Key", e.g. "MetaLeft+KeyM", "ControlLeft+ShiftLeft+KeyP".
func (m *Manager) Register(shortcut string, handler func()) error {
	hk, err := parseHotkey(shortcut)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if old, ok := m.hotkeys[shortcut]; ok {
		old.Unregister()
	}
	m.hotkeys[shortcut] = hk
	m.mu.Unlock()

	var listen func()
	listen = func() {
		if err := hk.Register(); err != nil {
			return
		}
		<-hk.Keydown()
		<-hk.Keyup()
		hk.Unregister()
		handler()
		listen()
	}
	go listen()
	return nil
}

// Unregister removes a registered shortcut.
func (m *Manager) Unregister(shortcut string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	hk, ok := m.hotkeys[shortcut]
	if !ok {
		return fmt.Errorf("shortcut %q not registered", shortcut)
	}
	delete(m.hotkeys, shortcut)
	return hk.Unregister()
}

func parseHotkey(s string) (*hotkey.Hotkey, error) {
	keys := strings.Split(s, "+")
	var modifiers []hotkey.Modifier
	var key hotkey.Key
	for _, code := range keys {
		if mod, ok := modifierCodeMap[code]; ok {
			modifiers = append(modifiers, mod)
			continue
		}
		if k, ok := keyCodeMap[code]; ok && k != 0 {
			key = k
			continue
		}
	}
	if len(modifiers) == 0 {
		return nil, fmt.Errorf("shortcut must have a modifier")
	}
	if key == 0 {
		return nil, fmt.Errorf("shortcut must have a key")
	}
	return hotkey.New(modifiers, key), nil
}
