package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
	"github.com/ltaoo/velo/updater/types"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateDownloadManager_ResumeDownload(t *testing.T) {
	// Create test content
	testContent := []byte("This is a test file for resume functionality. It needs to be long enough to test partial downloads.")
	hash := sha256.Sum256(testContent)
	expectedChecksum := hex.EncodeToString(hash[:])

	// Track how many times the server is called
	var callCount int32
	var lastRangeHeader string

	// Create test server that supports Range requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		rangeHeader := r.Header.Get("Range")
		lastRangeHeader = rangeHeader

		if rangeHeader != "" {
			// Parse range header (simplified for test)
			var start int
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)

			// Send partial content
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(testContent)-1, len(testContent)))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)-start))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(testContent[start:])
		} else {
			// Send full content
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(testContent)
		}
	}))
	defer server.Close()

	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test_file")
	tmpPath := destPath + ".tmp"

	// First, create a partial file to simulate interrupted download
	partialContent := testContent[:50] // First 50 bytes
	err := os.WriteFile(tmpPath, partialContent, 0644)
	require.NoError(t, err)

	// Now download with resume
	// Note: We need to use HTTP URL for testing since httptest doesn't support HTTPS
	// In production, HTTPS validation will work correctly
	// For this test, we'll temporarily bypass HTTPS validation by using the internal method
	ctx := context.Background()
	err = dm.downloadWithResume(ctx, server.URL, map[string]string{}, tmpPath, int64(len(partialContent)), nil)
	require.NoError(t, err)

	// Verify the server was called with Range header
	assert.Contains(t, lastRangeHeader, "bytes=50-", "Should request resume from byte 50")

	// Verify the complete file
	downloadedContent, err := os.ReadFile(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloadedContent, "Downloaded content should match original")

	// Verify checksum
	actualChecksum, err := dm.calculateSHA256(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, expectedChecksum, actualChecksum, "Checksum should match")
}

func TestUpdateDownloadManager_ResumeFromZero(t *testing.T) {
	// Test that download works correctly when starting from byte 0
	testContent := []byte("Test content for zero-byte start")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")

		if rangeHeader != "" {
			t.Errorf("Should not send Range header when starting from 0, got: %s", rangeHeader)
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test_file.tmp")

	ctx := context.Background()
	err := dm.downloadWithResume(ctx, server.URL, map[string]string{}, tmpPath, 0, nil)
	require.NoError(t, err)

	// Verify content
	downloadedContent, err := os.ReadFile(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloadedContent)
}

func TestUpdateDownloadManager_ResumeProgressCallback(t *testing.T) {
	testContent := make([]byte, 1000) // 1KB of data
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")

		if rangeHeader != "" {
			var start int
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(testContent)-1, len(testContent)))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)-start))
			w.WriteHeader(http.StatusPartialContent)

			// Write in chunks to trigger progress callbacks
			chunkSize := 100
			for i := start; i < len(testContent); i += chunkSize {
				end := i + chunkSize
				if end > len(testContent) {
					end = len(testContent)
				}
				w.Write(testContent[i:end])
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				time.Sleep(10 * time.Millisecond) // Small delay to allow progress tracking
			}
		} else {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(testContent)
		}
	}))
	defer server.Close()

	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test_file.tmp")

	// Create partial file
	partialSize := 300
	err := os.WriteFile(tmpPath, testContent[:partialSize], 0644)
	require.NoError(t, err)

	// Track progress
	var progressUpdates []types.DownloadProgress
	callback := func(progress types.DownloadProgress) {
		progressUpdates = append(progressUpdates, progress)
	}

	ctx := context.Background()
	err = dm.downloadWithResume(ctx, server.URL, map[string]string{}, tmpPath, int64(partialSize), callback)
	require.NoError(t, err)

	// Verify we got progress updates
	assert.Greater(t, len(progressUpdates), 0, "Should receive progress updates")

	// Verify progress is monotonically increasing
	for i := 1; i < len(progressUpdates); i++ {
		assert.GreaterOrEqual(t, progressUpdates[i].BytesDownloaded, progressUpdates[i-1].BytesDownloaded,
			"Downloaded bytes should be monotonically increasing")
	}

	// Verify final progress
	if len(progressUpdates) > 0 {
		lastProgress := progressUpdates[len(progressUpdates)-1]
		assert.Equal(t, int64(len(testContent)), lastProgress.BytesDownloaded,
			"Final progress should show all bytes downloaded")
	}
}

func TestUpdateDownloadManager_ServerDoesNotSupportResume(t *testing.T) {
	testContent := []byte("Test content")

	// Server that doesn't support Range requests
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		rangeHeader := r.Header.Get("Range")

		if rangeHeader != "" && count == 1 {
			// First call with Range header - return 416 Range Not Satisfiable
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		// Subsequent call or no Range header - return full content
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	dm := NewUpdateDownloadManager(&logger)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test_file.tmp")

	// Create partial file
	partialContent := []byte("partial")
	err := os.WriteFile(tmpPath, partialContent, 0644)
	require.NoError(t, err)

	// Try to resume - should fall back to full download
	ctx := context.Background()
	err = dm.downloadWithResume(ctx, server.URL, map[string]string{}, tmpPath, int64(len(partialContent)), nil)
	require.NoError(t, err)

	// Verify server was called twice (once with Range, once without)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount), "Server should be called twice")

	// Verify content (should be full content, not partial + remaining)
	downloadedContent, err := os.ReadFile(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloadedContent)
}
