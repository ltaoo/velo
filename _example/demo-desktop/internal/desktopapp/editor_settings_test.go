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
	if settings.FileEditor == nil || settings.FileEditor.ID != "code" || settings.FileEditor.Name != "VS Code" {
		t.Fatalf("FileEditor = %#v, want VS Code", settings.FileEditor)
	}
	if len(settings.FileEditorRules) != 6 {
		t.Fatalf("FileEditorRules = %#v, want default common rules", settings.FileEditorRules)
	}
	if settings.FileEditorRules[4].Extension != ".mp4" || settings.FileEditorRules[4].Editor.ID != "none" {
		t.Fatalf("mp4 default rule = %#v, want no-op", settings.FileEditorRules[4])
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

func TestLoadStoredEditorSettingsReadsFileEditor(t *testing.T) {
	settings, err := loadStoredEditorSettings(json.RawMessage(`{"fileEditor":{"id":"cursor","name":"Cursor"}}`))
	if err != nil {
		t.Fatalf("loadStoredEditorSettings: %v", err)
	}
	if settings.FileEditor == nil || settings.FileEditor.ID != "cursor" || settings.FileEditor.Name != "Cursor" {
		t.Fatalf("FileEditor = %#v, want Cursor", settings.FileEditor)
	}
}

func TestLoadStoredEditorSettingsReadsFileEditorRules(t *testing.T) {
	settings, err := loadStoredEditorSettings(json.RawMessage(`{"fileEditorRules":[{"extension":"mp4","editor":{"id":"system","name":"系统默认应用"}},{"extension":".TS","editor":{"id":"cursor","name":"Cursor"}}]}`))
	if err != nil {
		t.Fatalf("loadStoredEditorSettings: %v", err)
	}
	if len(settings.FileEditorRules) != 2 {
		t.Fatalf("FileEditorRules = %#v, want 2 rules", settings.FileEditorRules)
	}
	if settings.FileEditorRules[0].Extension != ".mp4" || settings.FileEditorRules[0].Editor.ID != "system" {
		t.Fatalf("first rule = %#v, want .mp4 system", settings.FileEditorRules[0])
	}
	if settings.FileEditorRules[1].Extension != ".ts" || settings.FileEditorRules[1].Editor.ID != "cursor" {
		t.Fatalf("second rule = %#v, want .ts cursor", settings.FileEditorRules[1])
	}
}

func TestEditorSelectionForOpenUsesFileExtensionRule(t *testing.T) {
	raw := json.RawMessage(`{"fileEditor":{"id":"code","name":"VS Code"},"fileEditorRules":[{"extension":".mp4","editor":{"id":"system","name":"系统默认应用"}}]}`)
	selection := editorSelectionForOpen("/Users/me/movie.mp4", "", "", "", raw)
	if selection == nil || selection.ID != "system" {
		t.Fatalf("selection = %#v, want system default app", selection)
	}
}

func TestEditorSelectionForOpenDefaultsMediaToNoop(t *testing.T) {
	selection := editorSelectionForOpen("/Users/me/song.mp3", "", "", "", nil)
	if selection == nil || selection.ID != "none" {
		t.Fatalf("selection = %#v, want no-op", selection)
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
	raw, err := marshalEditorSettingsForStore(EditorSettings{
		VimMode:           true,
		CalendarWeekStart: "sunday",
		FileEditor:        &EditorAppSelection{ID: "cursor", Name: "Cursor"},
		FileEditorRules: []EditorFileRule{
			{Extension: ".mp4", Editor: &EditorAppSelection{ID: "none", Name: "不打开"}},
		},
	})
	if err != nil {
		t.Fatalf("marshalEditorSettingsForStore: %v", err)
	}
	if string(raw) != `{"vimMode":true,"calendarWeekStart":"sunday","fileEditor":{"id":"cursor","name":"Cursor"},"fileEditorRules":[{"extension":".mp4","editor":{"id":"none","name":"不打开"}}]}` {
		t.Fatalf("stored settings = %s, want vimMode true, sunday week start, cursor editor, and file rule", raw)
	}
}
