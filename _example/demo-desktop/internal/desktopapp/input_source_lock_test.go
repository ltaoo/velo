package desktopapp

import (
	"encoding/json"
	"testing"
)

func TestLoadStoredInputSourceLockSettingsDefaults(t *testing.T) {
	settings, err := loadStoredInputSourceLockSettings(nil)
	if err != nil {
		t.Fatalf("loadStoredInputSourceLockSettings: %v", err)
	}
	if settings.Enabled {
		t.Fatalf("Enabled = true, want false")
	}
	if settings.AppRules == nil {
		t.Fatalf("AppRules should be initialized")
	}
}

func TestNormalizeInputSourceLockSettings(t *testing.T) {
	settings := normalizeInputSourceLockSettings(InputSourceLockSettings{
		Enabled:         true,
		DefaultSourceID: " abc ",
		AppRules: []InputSourceAppRule{
			{AppID: " com.apple.Terminal ", AppName: " Terminal ", SourceID: " source ", Enabled: true},
			{AppID: "com.apple.Terminal", AppName: "Duplicate", SourceID: "other", Enabled: true},
			{AppID: " ", SourceID: "ignored", Enabled: true},
		},
	})
	if !settings.Enabled || settings.DefaultSourceID != "abc" {
		t.Fatalf("settings = %#v", settings)
	}
	if len(settings.AppRules) != 1 {
		t.Fatalf("len(AppRules) = %d, want 1", len(settings.AppRules))
	}
	rule := settings.AppRules[0]
	if rule.AppID != "com.apple.Terminal" || rule.AppName != "Terminal" || rule.SourceID != "source" || !rule.Enabled {
		t.Fatalf("rule = %#v", rule)
	}
}

func TestInputSourceManagerConfigSkipsDisabledRules(t *testing.T) {
	config := inputSourceManagerConfig(InputSourceLockSettings{
		Enabled:         true,
		DefaultSourceID: "default",
		AppRules: []InputSourceAppRule{
			{AppID: "a", SourceID: "source-a", Enabled: true},
			{AppID: "b", SourceID: "source-b", Enabled: false},
			{AppID: "c", Enabled: true},
		},
	})
	if !config.Enabled || config.DefaultSourceID != "default" {
		t.Fatalf("config = %#v", config)
	}
	if len(config.AppRules) != 1 || config.AppRules[0].AppID != "a" || config.AppRules[0].SourceID != "source-a" {
		t.Fatalf("AppRules = %#v", config.AppRules)
	}
}

func TestMarshalInputSourceLockSettingsForStore(t *testing.T) {
	raw, err := marshalInputSourceLockSettingsForStore(InputSourceLockSettings{
		Enabled:         true,
		DefaultSourceID: "default",
		AppRules:        []InputSourceAppRule{{AppID: "app", SourceID: "source", Enabled: true}},
	})
	if err != nil {
		t.Fatalf("marshalInputSourceLockSettingsForStore: %v", err)
	}
	var settings InputSourceLockSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !settings.Enabled || settings.DefaultSourceID != "default" || len(settings.AppRules) != 1 {
		t.Fatalf("settings = %#v", settings)
	}
}
