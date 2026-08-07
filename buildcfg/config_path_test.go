package buildcfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathPrefersVeloJSON(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{ConfigFileName, LegacyConfigFileName} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	path, legacy, err := ResolveConfigPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, ConfigFileName); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	if legacy {
		t.Fatal("velo.json must not be marked as legacy")
	}
}

func TestResolveConfigPathFallsBackToLegacyName(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, LegacyConfigFileName)
	if err := os.WriteFile(legacyPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	path, legacy, err := ResolveConfigPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if path != legacyPath {
		t.Fatalf("path = %q, want %q", path, legacyPath)
	}
	if !legacy {
		t.Fatal("app-config.json must be marked as legacy")
	}
}

func TestResolveConfigPathReportsMissingConfig(t *testing.T) {
	if _, _, err := ResolveConfigPath(t.TempDir()); err == nil {
		t.Fatal("expected an error when no Velo configuration exists")
	}
}
