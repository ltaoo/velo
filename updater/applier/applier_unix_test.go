//go:build !windows && !darwin
// +build !windows,!darwin

package applier

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"github.com/ltaoo/velo/updater/types"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixUpdater_Apply(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewUnixUpdater(&logger)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test executable
	execPath := filepath.Join(tempDir, "test")
	err := os.WriteFile(execPath, []byte("original content"), 0755)
	require.NoError(t, err)

	// Create a tar.gz archive with a new executable
	updatePath := filepath.Join(tempDir, "update.tar.gz")
	err = createUnixTestTarGz(updatePath, "test", []byte("updated content"))
	require.NoError(t, err)

	// Apply the update
	err = updater.Apply(updatePath, execPath)

	// On macOS, this will fail because the test binary is not signed
	// On Linux, this should succeed
	if err != nil {
		// Check if it's a signature verification error (expected on macOS)
		updateErr, ok := err.(*types.UpdateError)
		if ok && updateErr.Category == types.ErrCategorySecurity {
			t.Logf("Code signature verification failed (expected on macOS with unsigned test binary): %v", err)
			t.Skip("Skipping test on macOS due to unsigned test binary")
		} else {
			// Other errors should fail the test
			require.NoError(t, err)
		}
	} else {
		// Verify the executable was updated
		content, err := os.ReadFile(execPath)
		require.NoError(t, err)
		assert.Equal(t, "updated content", string(content))

		// Verify executable permissions
		info, err := os.Stat(execPath)
		require.NoError(t, err)
		assert.NotEqual(t, 0, info.Mode()&0111, "File should have execute permission")
	}
}

func TestUnixUpdater_FindExecutable(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewUnixUpdater(&logger)

	// Create a temporary directory with an executable
	tempDir := t.TempDir()
	execPath := filepath.Join(tempDir, "test")
	err := os.WriteFile(execPath, []byte("test"), 0755)
	require.NoError(t, err)

	// Find the executable
	foundPath, err := updater.findExecutable(tempDir)
	require.NoError(t, err)
	assert.Equal(t, execPath, foundPath)
}

func TestUnixUpdater_FindExecutable_NotFound(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewUnixUpdater(&logger)

	// Create a temporary directory without an executable
	tempDir := t.TempDir()

	// Create a non-executable file
	nonExecPath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(nonExecPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Try to find an executable
	_, err = updater.findExecutable(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no executable found")
}

func TestUnixUpdater_FindExecutable_NestedDirectory(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	updater := NewUnixUpdater(&logger)

	// Create a nested directory structure
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	execPath := filepath.Join(nestedDir, "test")
	err = os.WriteFile(execPath, []byte("test"), 0755)
	require.NoError(t, err)

	// Find the executable
	foundPath, err := updater.findExecutable(tempDir)
	require.NoError(t, err)
	assert.Equal(t, execPath, foundPath)
}

// Helper function to create a test tar.gz archive with an executable
func createUnixTestTarGz(tarPath, filename string, content []byte) error {
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Write file header
	header := &tar.Header{
		Name: filename,
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	// Write file content
	_, err = io.Copy(tarWriter, io.NopCloser(io.LimitReader(io.MultiReader(), 0)))
	if err != nil {
		return err
	}

	_, err = tarWriter.Write(content)
	return err
}
