package applier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUpdateExtractionDirCreatesUniqueEmptyDirectories(t *testing.T) {
	first_dir, err := create_update_extraction_dir()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(first_dir) })

	marker_path := filepath.Join(first_dir, "stale-executable")
	require.NoError(t, os.WriteFile(marker_path, []byte("stale"), 0755))

	second_dir, err := create_update_extraction_dir()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(second_dir) })

	assert.NotEqual(t, first_dir, second_dir)
	entries, err := os.ReadDir(second_dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestFindUpdateExecutablePrefersTargetName(t *testing.T) {
	temp_dir := t.TempDir()
	stale_path := filepath.Join(temp_dir, "test")
	target_path := filepath.Join(temp_dir, "wx_video_download_macos")
	require.NoError(t, os.WriteFile(stale_path, []byte("stale"), 0755))
	require.NoError(t, os.WriteFile(target_path, []byte("mach-o"), 0755))

	found_path, err := find_update_executable(
		temp_dir,
		"wx_video_download_macos",
		false,
		func(_ string, info os.FileInfo) bool { return info.Mode()&0111 != 0 },
	)
	require.NoError(t, err)
	assert.Equal(t, target_path, found_path)
}

func TestFindUpdateExecutableAllowsSingleFallback(t *testing.T) {
	temp_dir := t.TempDir()
	fallback_path := filepath.Join(temp_dir, "renamed-release-binary")
	require.NoError(t, os.WriteFile(fallback_path, []byte("binary"), 0755))

	found_path, err := find_update_executable(
		temp_dir,
		"current-name",
		false,
		func(_ string, info os.FileInfo) bool { return info.Mode()&0111 != 0 },
	)
	require.NoError(t, err)
	assert.Equal(t, fallback_path, found_path)
}

func TestFindUpdateExecutableRejectsAmbiguousFallback(t *testing.T) {
	temp_dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(temp_dir, "first"), []byte("first"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(temp_dir, "second"), []byte("second"), 0755))

	_, err := find_update_executable(
		temp_dir,
		"current-name",
		false,
		func(_ string, info os.FileInfo) bool { return info.Mode()&0111 != 0 },
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple executable files found")
}
