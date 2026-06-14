package desktopapp

import (
	"testing"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/autostart"
)

type fakeAutoStart struct {
	enabled bool
}

func (f fakeAutoStart) Enable() error {
	return nil
}

func (f fakeAutoStart) Disable() error {
	return nil
}

func (f fakeAutoStart) IsEnabled() bool {
	return f.enabled
}

func TestAutoStartAppNameFromConfig(t *testing.T) {
	previousAssets := appAssets
	defer func() { appAssets = previousAssets }()

	appAssets = Assets{AppConfigData: []byte(`{"app":{"name":"DemoApp","display_name":"Demo App"}}`)}
	if got := autoStartAppName(); got != "DemoApp" {
		t.Fatalf("autoStartAppName() = %q, want DemoApp", got)
	}
}

func TestAutoStartConfig(t *testing.T) {
	previousAssets := appAssets
	previousNewAutoStart := newAutoStart
	defer func() {
		appAssets = previousAssets
		newAutoStart = previousNewAutoStart
	}()

	var gotName string
	appAssets = Assets{AppConfigData: []byte(`{"app":{"name":"DemoApp"}}`)}
	newAutoStart = func(appName string) autostart.AutoStart {
		gotName = appName
		return fakeAutoStart{enabled: true}
	}

	payload := autoStartConfig()
	config, ok := payload["config"].(velo.H)
	if !ok {
		t.Fatalf("config payload type = %T, want velo.H", payload["config"])
	}
	if gotName != "DemoApp" {
		t.Fatalf("newAutoStart appName = %q, want DemoApp", gotName)
	}
	if config["appName"] != "DemoApp" {
		t.Fatalf("config appName = %v, want DemoApp", config["appName"])
	}
	if config["enabled"] != true {
		t.Fatalf("config enabled = %v, want true", config["enabled"])
	}
}
