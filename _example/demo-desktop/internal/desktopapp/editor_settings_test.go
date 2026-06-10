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
	if settings.CalendarWeekStart != "monday" {
		t.Fatalf("CalendarWeekStart = %q, want monday", settings.CalendarWeekStart)
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
	if settings.CalendarWeekStart != "monday" {
		t.Fatalf("CalendarWeekStart = %q, want monday", settings.CalendarWeekStart)
	}
}

func TestLoadStoredEditorSettingsReadsCalendarWeekStart(t *testing.T) {
	settings, err := loadStoredEditorSettings(json.RawMessage(`{"calendarWeekStart":"sunday"}`))
	if err != nil {
		t.Fatalf("loadStoredEditorSettings: %v", err)
	}
	if settings.CalendarWeekStart != "sunday" {
		t.Fatalf("CalendarWeekStart = %q, want sunday", settings.CalendarWeekStart)
	}
}

func TestLoadStoredEditorSettingsNormalizesCalendarWeekStart(t *testing.T) {
	settings, err := loadStoredEditorSettings(json.RawMessage(`{"calendarWeekStart":"friday"}`))
	if err != nil {
		t.Fatalf("loadStoredEditorSettings: %v", err)
	}
	if settings.CalendarWeekStart != "monday" {
		t.Fatalf("CalendarWeekStart = %q, want monday", settings.CalendarWeekStart)
	}
}

func TestLoadStoredEditorSettingsRejectsInvalidJSON(t *testing.T) {
	if _, err := loadStoredEditorSettings(json.RawMessage(`{`)); err == nil {
		t.Fatalf("loadStoredEditorSettings should reject invalid JSON")
	}
}

func TestMarshalEditorSettingsForStore(t *testing.T) {
	raw, err := marshalEditorSettingsForStore(EditorSettings{VimMode: true, CalendarWeekStart: "sunday"})
	if err != nil {
		t.Fatalf("marshalEditorSettingsForStore: %v", err)
	}
	if string(raw) != `{"vimMode":true,"calendarWeekStart":"sunday"}` {
		t.Fatalf("stored settings = %s, want vimMode true and sunday week start", raw)
	}
}
