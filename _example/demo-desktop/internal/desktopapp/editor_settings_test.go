package desktopapp

import (
	"encoding/json"
	"testing"
)

func TestLoadStoredEditorSettingsDefaultsToVimDisabled(t *testing.T) {
	settings, err := loadStoredEditorSettings(nil)
	if err != nil {
		t.Fatalf("loadStoredEditorSettings: %v", err)
	}
	if settings.VimMode {
		t.Fatalf("VimMode = true, want false")
	}
}

func TestLoadStoredEditorSettingsReadsVimMode(t *testing.T) {
	settings, err := loadStoredEditorSettings(json.RawMessage(`{"vimMode":true}`))
	if err != nil {
		t.Fatalf("loadStoredEditorSettings: %v", err)
	}
	if !settings.VimMode {
		t.Fatalf("VimMode = false, want true")
	}
}

func TestLoadStoredEditorSettingsRejectsInvalidJSON(t *testing.T) {
	if _, err := loadStoredEditorSettings(json.RawMessage(`{`)); err == nil {
		t.Fatalf("loadStoredEditorSettings should reject invalid JSON")
	}
}

func TestMarshalEditorSettingsForStore(t *testing.T) {
	raw, err := marshalEditorSettingsForStore(EditorSettings{VimMode: true})
	if err != nil {
		t.Fatalf("marshalEditorSettingsForStore: %v", err)
	}
	if string(raw) != `{"vimMode":true}` {
		t.Fatalf("stored settings = %s, want vimMode true", raw)
	}
}
