package inputsource

import "testing"

func TestResolveTarget(t *testing.T) {
	config := Config{
		Enabled:         true,
		DefaultSourceID: "default",
		AppRules: []AppRule{
			{AppID: "ignored", Mode: RuleIgnore},
			{AppID: "locked", Mode: RuleLock, SourceID: "locked-source"},
			{AppID: "empty-lock", Mode: RuleLock},
			{AppID: "use-default", Mode: RuleUseDefault},
		},
	}

	tests := []struct {
		name string
		app  App
		want string
	}{
		{name: "global default", app: App{ID: "other"}, want: "default"},
		{name: "ignored", app: App{ID: "ignored"}, want: ""},
		{name: "locked", app: App{ID: "locked"}, want: "locked-source"},
		{name: "empty locked falls back", app: App{ID: "empty-lock"}, want: "default"},
		{name: "use default", app: App{ID: "use-default"}, want: "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveTarget(config, tt.app); got != tt.want {
				t.Fatalf("resolveTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeConfig(t *testing.T) {
	config := normalizeConfig(Config{})
	if config.PollInterval != DefaultPollInterval {
		t.Fatalf("PollInterval = %s, want %s", config.PollInterval, DefaultPollInterval)
	}
	if config.SuppressionWindow != DefaultSuppressionWindow {
		t.Fatalf("SuppressionWindow = %s, want %s", config.SuppressionWindow, DefaultSuppressionWindow)
	}
}
