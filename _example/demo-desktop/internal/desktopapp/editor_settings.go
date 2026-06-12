package desktopapp

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const editorSettingsKey = "demo-desktop:settings:editor:v1"

type EditorSettings struct {
	VimMode           bool                `json:"vimMode"`
	CalendarWeekStart string              `json:"calendarWeekStart"`
	FileEditor        *EditorAppSelection `json:"fileEditor,omitempty"`
	FileEditorRules   []EditorFileRule    `json:"fileEditorRules,omitempty"`
}

type EditorAppSelection struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
}

type EditorFileRule struct {
	Extension string              `json:"extension"`
	Editor    *EditorAppSelection `json:"editor"`
}

func defaultEditorSettings() EditorSettings {
	return EditorSettings{
		VimMode:           false,
		CalendarWeekStart: "monday",
		FileEditor:        defaultEditorAppSelection(),
		FileEditorRules:   defaultEditorFileRules(),
	}
}

func normalizeEditorSettings(settings EditorSettings) EditorSettings {
	weekStart := settings.CalendarWeekStart
	if weekStart != "sunday" {
		weekStart = "monday"
	}
	rules := normalizeEditorFileRules(settings.FileEditorRules)
	if settings.FileEditorRules == nil {
		rules = defaultEditorFileRules()
	}
	return EditorSettings{
		VimMode:           settings.VimMode,
		CalendarWeekStart: weekStart,
		FileEditor:        normalizeEditorAppSelection(settings.FileEditor),
		FileEditorRules:   rules,
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

func defaultEditorAppSelection() *EditorAppSelection {
	return &EditorAppSelection{
		ID:   "code",
		Name: editorDisplayName("code"),
	}
}

func defaultEditorNoneSelection() *EditorAppSelection {
	return &EditorAppSelection{
		ID:   editorNoneAppID,
		Name: editorDisplayName(editorNoneAppID),
	}
}

func defaultEditorBrowserSelection() *EditorAppSelection {
	return &EditorAppSelection{
		ID:   editorBrowserAppID,
		Name: editorDisplayName(editorBrowserAppID),
	}
}

func defaultEditorFileRules() []EditorFileRule {
	codeEditor := defaultEditorAppSelection()
	noneEditor := defaultEditorNoneSelection()
	browserEditor := defaultEditorBrowserSelection()
	extensions := []struct {
		extension string
		editor    *EditorAppSelection
	}{
		{".js", codeEditor},
		{".ts", codeEditor},
		{".tsx", codeEditor},
		{".jsx", codeEditor},
		{".html", browserEditor},
		{".mp4", noneEditor},
		{".mp3", noneEditor},
	}
	rules := make([]EditorFileRule, 0, len(extensions))
	for _, item := range extensions {
		rules = append(rules, EditorFileRule{
			Extension: item.extension,
			Editor:    item.editor,
		})
	}
	return rules
}

func normalizeEditorAppSelection(selection *EditorAppSelection) *EditorAppSelection {
	if selection == nil {
		return defaultEditorAppSelection()
	}
	id := normalizeEditorAppID(selection.ID)
	name := strings.TrimSpace(selection.Name)
	path := strings.TrimSpace(selection.Path)
	if id == "" && path == "" && name == "" {
		return defaultEditorAppSelection()
	}
	if id == "" && path != "" {
		id = editorAppIDForPath(path)
	}
	if path == "" && isCustomEditorAppID(id) {
		path = strings.TrimSpace(strings.TrimPrefix(id, editorAppIDPrefix))
	}
	if name == "" {
		name = editorDisplayName(id)
	}
	if name == "" && path != "" {
		name = editorAppNameFromPath(path)
	}
	if name == "" {
		name = id
	}
	return &EditorAppSelection{
		ID:   id,
		Name: name,
		Path: path,
	}
}

func normalizeEditorFileRules(rules []EditorFileRule) []EditorFileRule {
	seen := make(map[string]bool)
	normalized := make([]EditorFileRule, 0, len(rules))
	for _, rule := range rules {
		extension := normalizeEditorFileExtension(rule.Extension)
		if extension == "" || seen[extension] {
			continue
		}
		seen[extension] = true
		normalized = append(normalized, EditorFileRule{
			Extension: extension,
			Editor:    normalizeEditorAppSelection(rule.Editor),
		})
	}
	return normalized
}

func normalizeEditorFileExtension(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimLeft(value, ".")
	if value == "" {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return ""
	}
	return "." + value
}

func normalizeEditorAppID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || isCustomEditorAppID(value) {
		return value
	}
	return normalizeEditorName(value)
}

func editorAppSelectionFromRequest(id string, name string, path string) *EditorAppSelection {
	return normalizeEditorAppSelection(&EditorAppSelection{
		ID:   id,
		Name: name,
		Path: path,
	})
}

func editorSelectionForOpen(file string, id string, name string, path string, rawSettings json.RawMessage) *EditorAppSelection {
	if strings.TrimSpace(id) != "" || strings.TrimSpace(name) != "" || strings.TrimSpace(path) != "" {
		return editorAppSelectionFromRequest(id, name, path)
	}
	settings, err := loadStoredEditorSettings(rawSettings)
	if err != nil {
		return defaultEditorAppSelection()
	}
	if selection := editorSelectionForFile(file, settings); selection != nil {
		return selection
	}
	return normalizeEditorAppSelection(settings.FileEditor)
}

func editorSelectionForFile(file string, settings EditorSettings) *EditorAppSelection {
	extension := normalizeEditorFileExtension(strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), "."))
	if extension == "" {
		return nil
	}
	for _, rule := range normalizeEditorFileRules(settings.FileEditorRules) {
		if rule.Extension == extension {
			return normalizeEditorAppSelection(rule.Editor)
		}
	}
	return nil
}
