package inputsource

import (
	"errors"
	"testing"
	"time"
)

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
	if config.FailureRetryInterval != 0 {
		t.Fatalf("FailureRetryInterval = %s, want 0", config.FailureRetryInterval)
	}
}

func TestReportErrorSuppressesRepeatedError(t *testing.T) {
	manager := NewManager(Config{})
	var reported []string
	manager.OnError = func(err error) {
		reported = append(reported, err.Error())
	}

	manager.reportError(errors.New("same"))
	manager.reportError(errors.New("same"))
	if len(reported) != 1 {
		t.Fatalf("reported %d errors, want 1", len(reported))
	}

	manager.reportError(errors.New("different"))
	if len(reported) != 2 {
		t.Fatalf("reported %d errors after different error, want 2", len(reported))
	}

	manager.reportError(errors.New("same"))
	if len(reported) != 2 {
		t.Fatalf("reported %d errors after repeating previous error, want 2", len(reported))
	}

	manager.mu.Lock()
	manager.reportedErrors["same"] = time.Now().Add(-time.Second)
	manager.mu.Unlock()

	manager.reportError(errors.New("same"))
	if len(reported) != 3 {
		t.Fatalf("reported %d errors after interval expired, want 3", len(reported))
	}
}

func TestSelectFailureActive(t *testing.T) {
	manager := NewManager(Config{})
	now := time.Now()

	manager.rememberSelectFailure("app", "source", 0, now)
	if !manager.selectFailureActive("app", "source", now.Add(time.Second)) {
		t.Fatalf("select failure should be active")
	}
	if !manager.selectFailureActive("app", "source", now.Add(time.Hour)) {
		t.Fatalf("select failure should remain active without a retry interval")
	}
	if manager.selectFailureActive("app", "other-source", now.Add(time.Second)) {
		t.Fatalf("select failure should not apply to another source")
	}
	if manager.selectFailureActive("other-app", "source", now.Add(time.Second)) {
		t.Fatalf("select failure should not apply to another app")
	}

	manager.rememberSelectFailure("app", "source", time.Hour, now)
	if !manager.selectFailureActive("app", "source", now.Add(time.Second)) {
		t.Fatalf("select failure should be active before retry interval")
	}
	if manager.selectFailureActive("app", "source", now.Add(time.Hour+time.Second)) {
		t.Fatalf("select failure should expire")
	}

	manager.SetConfig(Config{})
	if manager.selectFailureActive("app", "source", now.Add(time.Second)) {
		t.Fatalf("SetConfig should clear select failure")
	}
}
