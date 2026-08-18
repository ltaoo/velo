//go:build darwin

package quarantine

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRemove(t *testing.T) {
	target_path := filepath.Join(t.TempDir(), "quarantined")
	if err := os.WriteFile(target_path, []byte("test"), 0755); err != nil {
		t.Fatalf("create test file: %v", err)
	}
	if output, err := exec.Command(
		"/usr/bin/xattr",
		"-w",
		"com.apple.quarantine",
		"0081;test;velo;",
		target_path,
	).CombinedOutput(); err != nil {
		t.Fatalf("set quarantine attribute: %v: %s", err, output)
	}

	if err := Remove(target_path); err != nil {
		t.Fatalf("remove quarantine attribute: %v", err)
	}
	if err := exec.Command(
		"/usr/bin/xattr",
		"-p",
		"com.apple.quarantine",
		target_path,
	).Run(); err == nil {
		t.Fatal("quarantine attribute still exists")
	}
	if err := Remove(target_path); err != nil {
		t.Fatalf("remove missing quarantine attribute: %v", err)
	}
}
