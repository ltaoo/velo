package updater

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ltaoo/velo/updater/types"
	"github.com/rs/zerolog"
)

type test_downloader struct {
	path  string
	calls int
}

func (d *test_downloader) DownloadUpdate(
	ctx context.Context,
	release *types.ReleaseInfo,
	on_progress types.DownloadCallback,
) (string, error) {
	d.calls++
	return d.path, nil
}

type test_restart_coordinator struct {
	executable_path string
	arguments       []string
	environment     []string
	calls           int
}

func (c *test_restart_coordinator) Request(
	executable_path string,
	arguments []string,
	environment []string,
	request_shutdown func(),
) error {
	c.calls++
	c.executable_path = executable_path
	c.arguments = append([]string(nil), arguments...)
	c.environment = append([]string(nil), environment...)
	request_shutdown()
	return nil
}

func TestUpdaterUsesInjectedDownloader(t *testing.T) {
	custom_downloader := &test_downloader{path: "/tmp/custom-update.zip"}
	app_updater := new_test_updater(t, &types.UpdaterOptions{
		Config:     test_update_config(),
		Downloader: custom_downloader,
	})

	path, err := app_updater.DownloadUpdate(
		context.Background(),
		&types.ReleaseInfo{Version: "1.1.0"},
		nil,
	)
	if err != nil {
		t.Fatalf("download update: %v", err)
	}
	if path != custom_downloader.path || custom_downloader.calls != 1 {
		t.Fatalf("custom downloader result: path=%q calls=%d", path, custom_downloader.calls)
	}
}

func TestUpdaterCoordinatesGracefulRestart(t *testing.T) {
	restart_coordinator := &test_restart_coordinator{}
	shutdown_calls := 0
	app_updater := new_test_updater(t, &types.UpdaterOptions{
		Config:             test_update_config(),
		RestartCoordinator: restart_coordinator,
		RequestShutdown:    func() { shutdown_calls++ },
	})

	if err := app_updater.RestartApplication([]string{"server"}); err != nil {
		t.Fatalf("restart application: %v", err)
	}
	executable_path, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve executable: %v", err)
	}
	want_arguments := []string{executable_path, "server"}
	if restart_coordinator.calls != 1 || shutdown_calls != 1 {
		t.Fatalf("restart calls=%d shutdown calls=%d", restart_coordinator.calls, shutdown_calls)
	}
	if restart_coordinator.executable_path != executable_path {
		t.Fatalf("restart executable=%q, want %q", restart_coordinator.executable_path, executable_path)
	}
	if !reflect.DeepEqual(restart_coordinator.arguments, want_arguments) {
		t.Fatalf("restart arguments=%#v, want %#v", restart_coordinator.arguments, want_arguments)
	}
	if len(restart_coordinator.environment) == 0 {
		t.Fatal("restart environment is empty")
	}
}

func TestUpdaterRestartRequiresCoordinator(t *testing.T) {
	app_updater := new_test_updater(t, &types.UpdaterOptions{Config: test_update_config()})
	if err := app_updater.RestartApplication(nil); err == nil {
		t.Fatal("restart without coordinator succeeded")
	}
}

func TestPlatformApplierRejectsLegacyDirectRestart(t *testing.T) {
	app_updater := new_test_updater(t, &types.UpdaterOptions{Config: test_update_config()})
	if err := app_updater.applier.Restart("/tmp/example", []string{"server"}); err == nil {
		t.Fatal("legacy direct restart succeeded")
	}
}

func new_test_updater(t *testing.T, options *types.UpdaterOptions) *AppUpdater {
	t.Helper()
	options.CurrentVersion = "1.0.0"
	options.StatePath = filepath.Join(t.TempDir(), "update-state.json")
	logger := zerolog.Nop()
	app_updater, err := NewUpdaterWithOptions(options, &logger)
	if err != nil {
		t.Fatalf("create updater: %v", err)
	}
	return app_updater
}

func test_update_config() *types.UpdateConfig {
	return &types.UpdateConfig{
		Enabled:        true,
		CheckFrequency: "manual",
		Sources: []types.UpdateSource{{
			Type:        "http",
			Priority:    1,
			ManifestURL: "https://example.com/manifest.json",
			Enabled:     true,
		}},
	}
}
