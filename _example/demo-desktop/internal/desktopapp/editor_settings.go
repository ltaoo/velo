package desktopapp

import (
	"encoding/json"
	"fmt"
)

const editorSettingsKey = "demo-desktop:settings:editor:v1"

type EditorSettings struct {
	VimMode bool `json:"vimMode"`
}

func defaultEditorSettings() EditorSettings {
	return EditorSettings{
		VimMode: false,
	}
}

func normalizeEditorSettings(settings EditorSettings) EditorSettings {
	return EditorSettings{
		VimMode: settings.VimMode,
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
