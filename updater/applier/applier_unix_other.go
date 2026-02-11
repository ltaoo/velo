//go:build !windows && !darwin
// +build !windows,!darwin

package applier

// VerifyCodeSignature is a no-op on non-macOS Unix systems
func (uu *UnixUpdater) VerifyCodeSignature(execPath string) error {
	// No code signature verification on Linux and other Unix systems
	return nil
}
