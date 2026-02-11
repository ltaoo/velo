package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"github.com/ltaoo/velo/updater/types"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateDownloadManager_Download_Success(t *testing.T) {
	// Create test content
	testContent := []byte("test file content for update")
	hash := sha256.Sum256(testContent)
	expectedChecksum := hex.EncodeToString(hash[:])

	// Create test HTTPS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", string(rune(len(testContent))))
		w.Write(testContent)
	}))
	defer server.Close()

	// Create download manager
	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	// Create temp directory for test
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded_file")

	// Track progress callbacks
	progressCalled := false
	callback := func(progress types.DownloadProgress) {
		progressCalled = true
		assert.GreaterOrEqual(t, progress.BytesDownloaded, int64(0))
		assert.GreaterOrEqual(t, progress.TotalBytes, int64(0))
	}

	// Download file
	ctx := context.Background()
	err := dm.Download(ctx, server.URL, map[string]string{}, destPath, expectedChecksum, false, callback)

	// Verify
	require.NoError(t, err)
	assert.True(t, progressCalled, "Progress callback should be called")

	// Verify file exists and content matches
	downloadedContent, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloadedContent)
}

func TestUpdateDownloadManager_Download_HTTPSValidation(t *testing.T) {
	// Create HTTP (not HTTPS) server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	}))
	defer server.Close()

	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "file")

	ctx := context.Background()
	err := dm.Download(ctx, server.URL, map[string]string{}, destPath, "dummy", false, nil)

	// Should fail due to non-HTTPS URL
	require.Error(t, err)
	updateErr, ok := err.(*types.UpdateError)
	require.True(t, ok, "Error should be UpdateError")
	assert.Equal(t, types.ErrCategorySecurity, updateErr.Category)
}

func TestUpdateDownloadManager_Download_ChecksumMismatch(t *testing.T) {
	testContent := []byte("test content")

	// Create test HTTPS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(testContent)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "file")

	// Use wrong checksum
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	ctx := context.Background()
	err := dm.Download(ctx, server.URL, map[string]string{}, destPath, wrongChecksum, false, nil)

	// Should fail due to checksum mismatch
	require.Error(t, err)
	updateErr, ok := err.(*types.UpdateError)
	require.True(t, ok, "Error should be UpdateError")
	assert.Equal(t, types.ErrCategoryValidation, updateErr.Category)

	// Verify temp file was cleaned up
	tmpPath := destPath + ".tmp"
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "Temp file should be cleaned up")

	// Verify destination file was not created
	_, err = os.Stat(destPath)
	assert.True(t, os.IsNotExist(err), "Destination file should not exist")
}

func TestUpdateDownloadManager_ValidateHTTPS(t *testing.T) {
	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tests := []struct {
		name      string
		url       string
		shouldErr bool
	}{
		{
			name:      "valid HTTPS URL",
			url:       "https://example.com/file.zip",
			shouldErr: false,
		},
		{
			name:      "HTTP URL should fail",
			url:       "http://example.com/file.zip",
			shouldErr: true,
		},
		{
			name:      "FTP URL should fail",
			url:       "ftp://example.com/file.zip",
			shouldErr: true,
		},
		{
			name:      "invalid URL should fail",
			url:       "not a url",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dm.validateHTTPS(tt.url)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateDownloadManager_CalculateSHA256(t *testing.T) {
	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content for checksum")

	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// Calculate checksum
	checksum, err := dm.calculateSHA256(testFile)
	require.NoError(t, err)

	// Verify checksum is correct
	hash := sha256.Sum256(testContent)
	expectedChecksum := hex.EncodeToString(hash[:])
	assert.Equal(t, expectedChecksum, checksum)
}

func TestUpdateDownloadManager_VerifyChecksum(t *testing.T) {
	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tests := []struct {
		name     string
		actual   string
		expected string
		match    bool
	}{
		{
			name:     "exact match",
			actual:   "abc123",
			expected: "abc123",
			match:    true,
		},
		{
			name:     "case insensitive match",
			actual:   "ABC123",
			expected: "abc123",
			match:    true,
		},
		{
			name:     "with whitespace",
			actual:   " abc123 ",
			expected: "abc123",
			match:    true,
		},
		{
			name:     "mismatch",
			actual:   "abc123",
			expected: "def456",
			match:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dm.verifyChecksum(tt.actual, tt.expected)
			assert.Equal(t, tt.match, result)
		})
	}
}
