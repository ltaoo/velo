package desktopapp

import (
	"encoding/json"
	"testing"

	"github.com/ltaoo/velo/inputsource"
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

func TestInputSourceManagerConfigForAvailableSkipsMissingSources(t *testing.T) {
	config := inputSourceManagerConfigForAvailable(InputSourceLockSettings{
		Enabled:         true,
		DefaultSourceID: "default",
		AppRules: []InputSourceAppRule{
			{AppID: "a", SourceID: "source-a", Enabled: true},
			{AppID: "b", SourceID: "missing-source", Enabled: true},
			{AppID: "c", SourceID: "disabled-missing-source", Enabled: false},
		},
	}, map[string]bool{
		"default":  true,
		"source-a": true,
	})

	if !config.Enabled || config.DefaultSourceID != "default" {
		t.Fatalf("config = %#v", config)
	}
	if len(config.AppRules) != 2 {
		t.Fatalf("len(AppRules) = %d, want 2", len(config.AppRules))
	}
	if config.AppRules[0].AppID != "a" || config.AppRules[0].Mode != inputsource.RuleLock || config.AppRules[0].SourceID != "source-a" {
		t.Fatalf("first rule = %#v", config.AppRules[0])
	}
	if config.AppRules[1].AppID != "b" || config.AppRules[1].Mode != inputsource.RuleIgnore || config.AppRules[1].SourceID != "" {
		t.Fatalf("missing source rule = %#v", config.AppRules[1])
	}
}

func TestInputSourceManagerConfigForAvailableDisablesWhenNoSourcesMatch(t *testing.T) {
	config := inputSourceManagerConfigForAvailable(InputSourceLockSettings{
		Enabled:         true,
		DefaultSourceID: "missing-default",
		AppRules:        []InputSourceAppRule{{AppID: "app", SourceID: "missing-source", Enabled: true}},
	}, map[string]bool{})

	if config.Enabled {
		t.Fatalf("Enabled = true, want false")
	}
	if config.DefaultSourceID != "" {
		t.Fatalf("DefaultSourceID = %q, want empty", config.DefaultSourceID)
	}
	if len(config.AppRules) != 1 || config.AppRules[0].Mode != inputsource.RuleIgnore {
		t.Fatalf("AppRules = %#v", config.AppRules)
	}
}

func TestInputSourceLockAvailabilityReportsMissingSources(t *testing.T) {
	availability := inputSourceLockAvailability(InputSourceLockSettings{
		Enabled:         true,
		DefaultSourceID: "missing-default",
		AppRules: []InputSourceAppRule{
			{AppID: "a", AppName: "A", SourceID: "missing-rule", Enabled: true},
			{AppID: "b", SourceID: "source-b", Enabled: true},
			{AppID: "c", SourceID: "disabled-missing-source", Enabled: false},
		},
	}, map[string]bool{"source-b": true})

	if !availability.HasMissingSources {
		t.Fatalf("HasMissingSources = false, want true")
	}
	if availability.MissingDefaultSourceID != "missing-default" {
		t.Fatalf("MissingDefaultSourceID = %q", availability.MissingDefaultSourceID)
	}
	if got, want := availability.MissingSourceIDs, []string{"missing-default", "missing-rule"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("MissingSourceIDs = %#v, want %#v", got, want)
	}
	if len(availability.MissingAppRules) != 1 || availability.MissingAppRules[0].AppID != "a" || availability.MissingAppRules[0].SourceID != "missing-rule" {
		t.Fatalf("MissingAppRules = %#v", availability.MissingAppRules)
	}
	if !availability.RuntimeEnabled {
		t.Fatalf("RuntimeEnabled = false, want true for remaining valid rule")
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
