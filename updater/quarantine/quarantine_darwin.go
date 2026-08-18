//go:build darwin

package quarantine

import (
	"fmt"
	"os/exec"
	"strings"
)

// Remove clears the macOS quarantine attribute from a file or bundle. Missing
// attributes are treated as success so callers can invoke it unconditionally.
func Remove(path string) error {
	output, err := exec.Command(
		"/usr/bin/xattr",
		"-dr",
		"com.apple.quarantine",
		path,
	).CombinedOutput()
	if err == nil {
		return nil
	}

	detail := strings.TrimSpace(string(output))
	if attribute_missing(detail) {
		return nil
	}
	if detail == "" {
		return fmt.Errorf("remove macOS quarantine attribute from %s: %w", path, err)
	}
	return fmt.Errorf("remove macOS quarantine attribute from %s: %w: %s", path, err, detail)
}

func attribute_missing(message string) bool {
	lower_message := strings.ToLower(message)
	return strings.Contains(lower_message, "no such xattr") ||
		strings.Contains(lower_message, "attribute not found") ||
		strings.Contains(lower_message, "errno 93")
}
