package desktopapp

import (
	"encoding/json"
	"fmt"
)

const editorSettingsKey = "demo-desktop:settings:editor:v1"

type EditorSettings struct {
	VimMode           bool   `json:"vimMode"`
	CalendarWeekStart string `json:"calendarWeekStart"`
}

func defaultEditorSettings() EditorSettings {
	return EditorSettings{
		VimMode:           false,
		CalendarWeekStart: "monday",
	}
}

func normalizeEditorSettings(settings EditorSettings) EditorSettings {
	weekStart := settings.CalendarWeekStart
	if weekStart != "sunday" {
		weekStart = "monday"
	}
	return EditorSettings{
		VimMode:           settings.VimMode,
		CalendarWeekStart: weekStart,
	}
}

func loadStoredEditorSettings(raw json.RawMessage) (EditorSettings, error) {
	if raw == nil {
		return defaultEditorSettings(), nil
	}

	var settings EditorSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return EditorSettings{}, fmt.Errorf("read editor settings: %w", err)
	}
	return normalizeEditorSettings(settings), nil
}

func marshalEditorSettingsForStore(settings EditorSettings) ([]byte, error) {
	return json.Marshal(normalizeEditorSettings(settings))
}
