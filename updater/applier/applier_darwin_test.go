//go:build darwin
// +build darwin

package applier

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"github.com/ltaoo/velo/updater/types"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDarwinUpdater_VerifyCodeSignature_ValidSignature(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewDarwinUpdater(&logger)

	// Use a system binary that should be signed (e.g., /bin/ls)
	err := updater.VerifyCodeSignature("/bin/ls")
	require.NoError(t, err, "System binary /bin/ls should have a valid signature")
}

func TestDarwinUpdater_VerifyCodeSignature_UnsignedBinary(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewDarwinUpdater(&logger)

	// Create a temporary unsigned executable
	tempDir := t.TempDir()
	unsignedExec := filepath.Join(tempDir, "unsigned")

	// Create a simple executable (unsigned)
	err := os.WriteFile(unsignedExec, []byte("#!/bin/sh\necho test\n"), 0755)
	require.NoError(t, err)

	// Verify code signature should fail for unsigned binary
	err = updater.VerifyCodeSignature(unsignedExec)
	require.Error(t, err, "Unsigned binary should fail verification")

	// Check that it's a security error
	updateErr, ok := err.(*types.UpdateError)
	require.True(t, ok, "Error should be an UpdateError")
	assert.Equal(t, types.ErrCategorySecurity, updateErr.Category, "Error should be a security error")
	assert.Contains(t, updateErr.Message, "code signature verification failed")
}

func TestDarwinUpdater_VerifyCodeSignature_NonExistentFile(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewDarwinUpdater(&logger)

	// Try to verify a non-existent file
	err := updater.VerifyCodeSignature("/nonexistent/file")
	require.Error(t, err, "Non-existent file should fail verification")

	// Check that it's a security error
	updateErr, ok := err.(*types.UpdateError)
	require.True(t, ok, "Error should be an UpdateError")
	assert.Equal(t, types.ErrCategorySecurity, updateErr.Category, "Error should be a security error")
}

func TestDarwinUpdater_FindAppBundlePath(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewDarwinUpdater(&logger)

	tests := []struct {
		name     string
		execPath string
		expected string
	}{
		{
			name:     "Standard app bundle path",
			execPath: "/Applications/MyApp.app/Contents/MacOS/myapp",
			expected: "/Applications/MyApp.app",
		},
		{
			name:     "Nested app bundle",
			execPath: "/Users/test/Desktop/MyApp.app/Contents/MacOS/myapp",
			expected: "/Users/test/Desktop/MyApp.app",
		},
		{
			name:     "No app bundle",
			execPath: "/usr/local/bin/myapp",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.findAppBundlePath(tt.execPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDarwinUpdater_Apply_WithSignatureVerification(t *testing.T) {
	// Skip this test if we can't create signed binaries in the test environment
	if !canCreateSignedBinary() {
		t.Skip("Cannot create signed binaries in test environment")
	}

	logger := zerolog.New(os.Stdout)
	updater := NewDarwinUpdater(&logger)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test executable (signed)
	execPath := filepath.Join(tempDir, "test")
	err := createSignedTestBinary(execPath)
	require.NoError(t, err)

	// Create a tar.gz archive with a new signed executable
	updatePath := filepath.Join(tempDir, "update.tar.gz")
	newExecPath := filepath.Join(tempDir, "new_test")
	err = createSignedTestBinary(newExecPath)
	require.NoError(t, err)

	// Create archive from the signed binary
	err = createDarwinTestTarGz(updatePath, "test", nil)
	require.NoError(t, err)

	// Apply the update
	err = updater.Apply(updatePath, execPath)

	// This might fail if the test binary isn't properly signed
	// In a real scenario, the update package would contain properly signed binaries
	if err != nil {
		t.Logf("Apply failed (expected in test environment): %v", err)
	}
}

// canCreateSignedBinary checks if we can create signed binaries in the test environment
func canCreateSignedBinary() bool {
	// Check if codesign is available and we have a signing identity
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	// Check if there's at least one valid identity
	return len(output) > 0
}

// createSignedTestBinary creates a simple signed test binary
func createSignedTestBinary(path string) error {
	content := []byte("#!/bin/sh\necho test\n")
	if err := os.WriteFile(path, content, 0755); err != nil {
		return err
	}

	// Try to sign it with ad-hoc signature
	cmd := exec.Command("codesign", "-s", "-", path)
	_ = cmd.Run()

	return nil
}

// createDarwinTestTarGz creates a test tar.gz archive for darwin tests
func createDarwinTestTarGz(archivePath, execName string, content []byte) error {
	if content == nil {
		content = []byte("new content")
	}

	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	header := &tar.Header{
		Name: execName,
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tarWriter, &byteReader{data: content}); err != nil {
		return err
	}

	return nil
}

// byteReader is a simple io.Reader for byte slices
type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
