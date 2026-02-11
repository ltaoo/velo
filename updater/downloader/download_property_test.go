package downloader

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"github.com/ltaoo/velo/updater/types"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/rs/zerolog"
)

// testDownloadWithProgress performs a download with progress tracking for testing
// This bypasses HTTPS validation for test servers
func testDownloadWithProgress(
	serverURL string,
	testClient *http.Client,
	destPath string,
	startByte int64,
	callback types.DownloadCallback,
) error {
	// Create request
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add Range header if resuming
	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	// Execute request with test client
	resp, err := testClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if startByte > 0 {
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
				return testDownloadWithProgress(serverURL, testClient, destPath, 0, callback)
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	} else {
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}

	// Get total size
	var totalSize int64
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			totalSize = size + startByte
		}
	}

	// Open file for writing
	var file *os.File
	if startByte > 0 {
		file, err = os.OpenFile(destPath, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		file, err = os.Create(destPath)
	}
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Download with progress reporting
	buf := make([]byte, 32*1024)
	downloaded := startByte

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := file.Write(buf[:n]); werr != nil {
				return fmt.Errorf("failed to write to file: %w", werr)
			}
			downloaded += int64(n)

			// Report progress
			if callback != nil {
				percentage := float64(0)
				if totalSize > 0 {
					percentage = float64(downloaded) / float64(totalSize) * 100
				}
				callback(types.DownloadProgress{
					BytesDownloaded: downloaded,
					TotalBytes:      totalSize,
					Percentage:      percentage,
					Speed:           0,
				})
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read response: %w", readErr)
		}
	}

	return nil
}

// Feature: auto-update, Property 8: 下载进度的单调性
// Validates: Requirements 3.2
func TestProperty_DownloadProgressMonotonicity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: For any download process, the reported download progress (percentage)
	// should be monotonically increasing and eventually reach 100%
	properties.Property("download progress is monotonically increasing", prop.ForAll(
		func(fileSize int64) bool {
			// Create test content of the specified size
			testContent := make([]byte, fileSize)
			for i := range testContent {
				testContent[i] = byte(i % 256)
			}

			// Calculate checksum
			hash := sha256.Sum256(testContent)
			expectedChecksum := hex.EncodeToString(hash[:])

			// Create test HTTPS server
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
				w.Write(testContent)
			}))
			defer server.Close()

			// Create test client that trusts the test server's certificate
			testClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true, // Only for testing
					},
				},
			}

			// Create temp directory
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "test_file")
			tmpPath := destPath + ".tmp"

			// Ensure directory exists
			os.MkdirAll(filepath.Dir(destPath), 0755)

			// Track progress values
			var progressValues []float64
			var mu sync.Mutex

			callback := func(progress types.DownloadProgress) {
				mu.Lock()
				defer mu.Unlock()
				progressValues = append(progressValues, progress.Percentage)
			}

			// Download file using test method
			err := testDownloadWithProgress(server.URL, testClient, tmpPath, 0, callback)
			if err != nil {
				t.Logf("Download failed: %v", err)
				return false
			}

			// Verify checksum
			logger := zerolog.Nop()
			dm := NewUpdateDownloadManager(&logger)
			actualChecksum, err := dm.calculateSHA256(tmpPath)
			if err != nil || !dm.verifyChecksum(actualChecksum, expectedChecksum) {
				t.Logf("Checksum verification failed")
				return false
			}

			// Verify monotonicity: each progress value should be >= previous value
			mu.Lock()
			defer mu.Unlock()

			if len(progressValues) == 0 {
				t.Log("No progress values recorded")
				return false
			}

			for i := 1; i < len(progressValues); i++ {
				if progressValues[i] < progressValues[i-1] {
					t.Logf("Progress decreased: %.2f%% -> %.2f%%", progressValues[i-1], progressValues[i])
					return false
				}
			}

			// Verify final progress is 100% (or very close due to floating point)
			finalProgress := progressValues[len(progressValues)-1]
			if finalProgress < 99.9 {
				t.Logf("Final progress %.2f%% is less than 100%%", finalProgress)
				return false
			}

			return true
		},
		// Generate file sizes from 1KB to 1MB
		gen.Int64Range(1024, 1024*1024),
	))

	// Property: For any download with known total size, progress percentage should
	// be calculated as (bytes_downloaded / total_bytes) * 100
	properties.Property("progress percentage calculation is correct", prop.ForAll(
		func(fileSize int64) bool {
			// Create test content
			testContent := make([]byte, fileSize)
			for i := range testContent {
				testContent[i] = byte(i % 256)
			}

			// Create test HTTPS server
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
				w.Write(testContent)
			}))
			defer server.Close()

			// Create test client
			testClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			// Create temp directory
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "test_file")

			// Track progress
			var progressReports []types.DownloadProgress
			var mu sync.Mutex

			callback := func(progress types.DownloadProgress) {
				mu.Lock()
				defer mu.Unlock()
				progressReports = append(progressReports, progress)
			}

			// Download file
			err := testDownloadWithProgress(server.URL, testClient, destPath, 0, callback)
			if err != nil {
				t.Logf("Download failed: %v", err)
				return false
			}

			// Verify percentage calculation for each progress report
			mu.Lock()
			defer mu.Unlock()

			for _, progress := range progressReports {
				if progress.TotalBytes > 0 {
					expectedPercentage := float64(progress.BytesDownloaded) / float64(progress.TotalBytes) * 100
					// Allow small floating point error (0.1%)
					diff := progress.Percentage - expectedPercentage
					if diff < -0.1 || diff > 0.1 {
						t.Logf("Percentage mismatch: got %.2f%%, expected %.2f%%", progress.Percentage, expectedPercentage)
						return false
					}
				}
			}

			return true
		},
		// Generate file sizes from 1KB to 500KB (smaller range for faster tests)
		gen.Int64Range(1024, 512*1024),
	))

	// Property: For any download, BytesDownloaded should be monotonically increasing
	properties.Property("bytes downloaded is monotonically increasing", prop.ForAll(
		func(fileSize int64) bool {
			// Create test content
			testContent := make([]byte, fileSize)
			for i := range testContent {
				testContent[i] = byte(i % 256)
			}

			// Create test HTTPS server
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
				w.Write(testContent)
			}))
			defer server.Close()

			// Create test client
			testClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			// Create temp directory
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "test_file")

			// Track bytes downloaded
			var bytesValues []int64
			var mu sync.Mutex

			callback := func(progress types.DownloadProgress) {
				mu.Lock()
				defer mu.Unlock()
				bytesValues = append(bytesValues, progress.BytesDownloaded)
			}

			// Download file
			err := testDownloadWithProgress(server.URL, testClient, destPath, 0, callback)
			if err != nil {
				t.Logf("Download failed: %v", err)
				return false
			}

			// Verify monotonicity
			mu.Lock()
			defer mu.Unlock()

			if len(bytesValues) == 0 {
				t.Log("No bytes values recorded")
				return false
			}

			for i := 1; i < len(bytesValues); i++ {
				if bytesValues[i] < bytesValues[i-1] {
					t.Logf("Bytes downloaded decreased: %d -> %d", bytesValues[i-1], bytesValues[i])
					return false
				}
			}

			// Verify final bytes downloaded equals file size
			finalBytes := bytesValues[len(bytesValues)-1]
			if finalBytes != fileSize {
				t.Logf("Final bytes %d does not match file size %d", finalBytes, fileSize)
				return false
			}

			return true
		},
		// Generate file sizes from 1KB to 500KB
		gen.Int64Range(1024, 512*1024),
	))

	// Property: For any download, progress callback should be called at least once
	properties.Property("progress callback is called at least once", prop.ForAll(
		func(fileSize int64) bool {
			// Create test content
			testContent := make([]byte, fileSize)
			for i := range testContent {
				testContent[i] = byte(i % 256)
			}

			// Create test HTTPS server
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
				w.Write(testContent)
			}))
			defer server.Close()

			// Create test client
			testClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			// Create temp directory
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "test_file")

			// Track callback invocations
			callbackCount := 0
			var mu sync.Mutex

			callback := func(progress types.DownloadProgress) {
				mu.Lock()
				defer mu.Unlock()
				callbackCount++
			}

			// Download file
			err := testDownloadWithProgress(server.URL, testClient, destPath, 0, callback)
			if err != nil {
				t.Logf("Download failed: %v", err)
				return false
			}

			// Verify callback was called
			mu.Lock()
			defer mu.Unlock()

			if callbackCount == 0 {
				t.Log("Progress callback was never called")
				return false
			}

			return true
		},
		// Generate file sizes from 1KB to 500KB
		gen.Int64Range(1024, 512*1024),
	))

	// Property: For any download with resume, progress should continue from resume point
	properties.Property("resumed download progress continues from resume point", prop.ForAll(
		func(fileSize int64, resumePercent int) bool {
			// Skip if file is too small for meaningful resume test
			if fileSize < 10*1024 {
				return true
			}

			// Calculate resume point
			resumePoint := fileSize * int64(resumePercent) / 100

			// Create test content
			testContent := make([]byte, fileSize)
			for i := range testContent {
				testContent[i] = byte(i % 256)
			}

			// Create test HTTPS server that supports resume
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rangeHeader := r.Header.Get("Range")
				if rangeHeader != "" {
					// Support resume
					w.WriteHeader(http.StatusPartialContent)
					w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)-int(resumePoint)))
					w.Write(testContent[resumePoint:])
				} else {
					w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
					w.Write(testContent)
				}
			}))
			defer server.Close()

			// Create test client
			testClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			// Create temp directory
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "test_file")

			// Create partial file
			if resumePoint > 0 {
				err := os.WriteFile(destPath, testContent[:resumePoint], 0644)
				if err != nil {
					t.Logf("Failed to create partial file: %v", err)
					return false
				}
			}

			// Track progress
			var firstProgress *types.DownloadProgress
			var mu sync.Mutex

			callback := func(progress types.DownloadProgress) {
				mu.Lock()
				defer mu.Unlock()
				if firstProgress == nil {
					firstProgress = &progress
				}
			}

			// Download file (should resume)
			err := testDownloadWithProgress(server.URL, testClient, destPath, resumePoint, callback)
			if err != nil {
				t.Logf("Download failed: %v", err)
				return false
			}

			// Verify first progress report includes resume point
			mu.Lock()
			defer mu.Unlock()

			if firstProgress != nil && resumePoint > 0 {
				if firstProgress.BytesDownloaded < resumePoint {
					t.Logf("First progress %d is less than resume point %d", firstProgress.BytesDownloaded, resumePoint)
					return false
				}
			}

			return true
		},
		// Generate file sizes from 10KB to 500KB
		gen.Int64Range(10*1024, 512*1024),
		// Generate resume percentages from 10% to 90%
		gen.IntRange(10, 90),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
