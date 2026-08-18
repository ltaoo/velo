//go:build !darwin

package quarantine

// Remove is a no-op on platforms without the macOS quarantine attribute.
func Remove(path string) error {
	return nil
}
