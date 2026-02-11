package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	// "github.com/ltaoo/velo/updater/checker"
	"github.com/ltaoo/velo/updater/types"
	"github.com/ltaoo/velo/updater/util"
)

// DownloadOptions configures download behavior
type DownloadOptions struct {
	MaxRetries    int               // Maximum number of retry attempts
	RetryDelay    time.Duration     // Initial delay between retries
	Timeout       time.Duration     // Timeout for each download attempt
	ResumeSupport bool              // Enable resume support
	Headers       map[string]string // Optional HTTP headers (e.g., for authentication)
}

// DefaultDownloadOptions returns sensible defaults
func DefaultDownloadOptions() DownloadOptions {
	return DownloadOptions{
		MaxRetries:    3,
		RetryDelay:    2 * time.Second,
		Timeout:       5 * time.Minute,
		ResumeSupport: true,
	}
}

// UpdateDownloadManager wraps the existing download functionality for the update system
type UpdateDownloadManager struct {
	logger  zerolog.Logger
	options DownloadOptions
}

// NewUpdateDownloadManager creates a new download manager for updates
func NewUpdateDownloadManager(logger *zerolog.Logger) *UpdateDownloadManager {
	return &UpdateDownloadManager{
		logger:  logger.With().Str("component", "download-manager").Logger(),
		options: DefaultDownloadOptions(),
	}
}

// NewUpdateDownloadManagerWithOptions creates a download manager with custom options
func NewUpdateDownloadManagerWithOptions(logger zerolog.Logger, options DownloadOptions) *UpdateDownloadManager {
	return &UpdateDownloadManager{
		logger:  logger.With().Str("component", "download-manager").Logger(),
		options: options,
	}
}

// Download downloads a file from the given URL to the destination path
// It verifies HTTPS, reports progress, validates the checksum (if skipChecksum is false), and supports resume
func (dm *UpdateDownloadManager) Download(
	ctx context.Context,
	downloadURL string,
	headers map[string]string,
	destPath string,
	expectedChecksum string,
	skipChecksum bool,
	callback types.DownloadCallback,
) error {
	dm.logger.Info().
		Str("url", downloadURL).
		Str("dest", destPath).
		Msg("Starting download")

	// Validate HTTPS
	if err := dm.validateHTTPS(downloadURL); err != nil {
		dm.logger.Error().Err(err).Msg("HTTPS validation failed")
		return &types.UpdateError{
			Category: types.ErrCategorySecurity,
			Message:  "download URL must use HTTPS",
			Cause:    err,
			Context: map[string]interface{}{
				"url": downloadURL,
			},
		}
	}

	// Create temporary file for download
	tmpPath := destPath + ".tmp"
	defer func() {
		// Clean up temp file if it exists and we're not resuming
		if _, err := os.Stat(tmpPath); err == nil {
			// Only remove if download is complete (destPath exists)
			if _, err := os.Stat(destPath); err == nil {
				os.Remove(tmpPath)
			}
		}
	}()

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		dm.logger.Error().Err(err).Str("dir", destDir).Msg("Failed to create destination directory")
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create destination directory",
			Cause:    err,
			Context: map[string]interface{}{
				"directory": destDir,
			},
		}
	}

	// Download with retry logic
	var lastErr error
	for attempt := 0; attempt <= dm.options.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := dm.options.RetryDelay * time.Duration(attempt)
			dm.logger.Info().
				Int("attempt", attempt+1).
				Int("max_retries", dm.options.MaxRetries+1).
				Dur("delay", delay).
				Msg("Retrying download after delay")
			time.Sleep(delay)
		}

		// Check if we can resume from a partial download
		var startByte int64 = 0
		if dm.options.ResumeSupport {
			if stat, err := os.Stat(tmpPath); err == nil {
				startByte = stat.Size()
				if startByte > 0 {
					dm.logger.Info().
						Int64("resume_from", startByte).
						Int("attempt", attempt+1).
						Msg("Resuming partial download")
				}
			}
		}

		// Create context with timeout
		downloadCtx := ctx
		if dm.options.Timeout > 0 {
			var cancel context.CancelFunc
			downloadCtx, cancel = context.WithTimeout(ctx, dm.options.Timeout)
			defer cancel()
		}

		// Attempt download
		err := dm.downloadWithResume(downloadCtx, downloadURL, headers, tmpPath, startByte, callback)
		if err == nil {
			// Download successful, break retry loop
			break
		}

		lastErr = err
		dm.logger.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Int("max_retries", dm.options.MaxRetries+1).
			Msg("Download attempt failed")

		// Check if context was cancelled
		if ctx.Err() != nil {
			return &types.UpdateError{
				Category: types.ErrCategoryNetwork,
				Message:  "download cancelled",
				Cause:    ctx.Err(),
			}
		}
	}

	if lastErr != nil {
		dm.logger.Error().Err(lastErr).Msg("Download failed after all retries")
		return &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  fmt.Sprintf("failed to download after %d attempts", dm.options.MaxRetries+1),
			Cause:    lastErr,
			Context: map[string]interface{}{
				"url":      downloadURL,
				"attempts": dm.options.MaxRetries + 1,
			},
		}
	}

	var actualChecksum string
	if skipChecksum {
		dm.logger.Info().Msg("Skipping checksum verification as configured")
	} else {
		// Verify checksum
		dm.logger.Info().Msg("Verifying checksum")
		checksum, err := dm.calculateSHA256(tmpPath)
		if err != nil {
			dm.logger.Error().Err(err).Msg("Failed to calculate checksum")
			return &types.UpdateError{
				Category: types.ErrCategoryValidation,
				Message:  "failed to calculate file checksum",
				Cause:    err,
				Context: map[string]interface{}{
					"file": tmpPath,
				},
			}
		}
		actualChecksum = checksum

		if !dm.verifyChecksum(actualChecksum, expectedChecksum) {
			dm.logger.Error().
				Str("expected", expectedChecksum).
				Str("actual", actualChecksum).
				Msg("Checksum verification failed")

			// Clean up the invalid file
			os.Remove(tmpPath)

			return &types.UpdateError{
				Category: types.ErrCategoryValidation,
				Message:  "checksum verification failed",
				Context: map[string]interface{}{
					"expected_checksum": expectedChecksum,
					"actual_checksum":   actualChecksum,
					"file":              tmpPath,
				},
			}
		}

		dm.logger.Info().Msg("Checksum verified successfully")
	}

	// Move temp file to final destination
	if err := os.Rename(tmpPath, destPath); err != nil {
		dm.logger.Error().Err(err).Msg("Failed to move file to destination")
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to move downloaded file to destination",
			Cause:    err,
			Context: map[string]interface{}{
				"source": tmpPath,
				"dest":   destPath,
			},
		}
	}

	dm.logger.Info().
		Str("dest", destPath).
		Str("checksum", actualChecksum).
		Msg("Download completed successfully")

	return nil
}

// downloadWithResume downloads a file with support for resuming from a specific byte
func (dm *UpdateDownloadManager) downloadWithResume(ctx context.Context, downloadURL string, headers map[string]string, destPath string, startByte int64, callback types.DownloadCallback) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: dm.options.Timeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers (e.g., for authentication)
	for key, value := range dm.options.Headers {
		req.Header.Set(key, value)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// For GitHub API URLs, add Accept header for asset download
	if util.IsGitHubAPIURL(downloadURL) {
		req.Header.Set("Accept", "application/octet-stream")
		dm.logger.Debug().Msg("Using GitHub API asset download with Accept: application/octet-stream")
	}

	// Add Range header if resuming
	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
		dm.logger.Debug().
			Int64("start_byte", startByte).
			Msg("Requesting resume from byte position")
	}

	// Log request details for debugging
	dm.logger.Debug().
		Str("url", downloadURL).
		Str("method", req.Method).
		Str("user_agent", req.Header.Get("User-Agent")).
		Str("authorization", maskAuthorization(req.Header.Get("Authorization"))).
		Str("accept", req.Header.Get("Accept")).
		Str("range", req.Header.Get("Range")).
		Int64("start_byte", startByte).
		Msg("Executing download request")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if startByte > 0 {
		// When resuming, we expect 206 Partial Content
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			// If server doesn't support resume, start from beginning
			if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
				dm.logger.Warn().Msg("Server doesn't support resume, starting from beginning")
				return dm.downloadWithResume(ctx, downloadURL, headers, destPath, 0, callback)
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
			totalSize = size + startByte // Add start byte for total progress
		}
	}

	// Open file for writing (append if resuming)
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
	buf := make([]byte, 32*1024) // 32KB buffer
	downloaded := startByte
	lastReportTime := time.Now()
	lastReportBytes := downloaded

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := file.Write(buf[:n]); werr != nil {
				return fmt.Errorf("failed to write to file: %w", werr)
			}
			downloaded += int64(n)

			// Report progress (throttle to avoid too many callbacks)
			now := time.Now()
			if callback != nil && now.Sub(lastReportTime) >= 100*time.Millisecond {
				percentage := float64(0)
				if totalSize > 0 {
					percentage = float64(downloaded) / float64(totalSize) * 100
				}

				// Calculate speed (bytes per second)
				elapsed := now.Sub(lastReportTime).Seconds()
				speed := int64(0)
				if elapsed > 0 {
					speed = int64(float64(downloaded-lastReportBytes) / elapsed)
				}

				callback(types.DownloadProgress{
					BytesDownloaded: downloaded,
					TotalBytes:      totalSize,
					Percentage:      percentage,
					Speed:           speed,
				})

				lastReportTime = now
				lastReportBytes = downloaded
			}
		}

		if err == io.EOF {
			// Final progress report
			if callback != nil {
				percentage := float64(100)
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
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
	}

	return nil
}

// validateHTTPS ensures the URL uses HTTPS protocol
func (dm *UpdateDownloadManager) validateHTTPS(downloadURL string) error {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use HTTPS protocol, got: %s", parsedURL.Scheme)
	}

	return nil
}

// calculateSHA256 calculates the SHA256 checksum of a file
func (dm *UpdateDownloadManager) calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	return checksum, nil
}

// verifyChecksum compares two checksums (case-insensitive)
func (dm *UpdateDownloadManager) verifyChecksum(actual, expected string) bool {
	// Normalize both checksums to lowercase for comparison
	actualLower := strings.ToLower(strings.TrimSpace(actual))
	expectedLower := strings.ToLower(strings.TrimSpace(expected))

	return actualLower == expectedLower
}

// maskAuthorization masks the authorization header for logging security
func maskAuthorization(auth string) string {
	if auth == "" {
		return "<not set>"
	}
	if len(auth) > 20 {
		return auth[:7] + "..." + auth[len(auth)-7:]
	}
	return "***"
}

// DownloadUpdate downloads the update package for the given release
func (uo *UpdateDownloadManager) DownloadUpdate(ctx context.Context, release *types.ReleaseInfo, progressCallback types.DownloadCallback) (string, error) {
	if release == nil {
		return "", &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "release info cannot be nil",
		}
	}
	uo.logger.Info().
		Str("version", release.Version).
		Str("url", release.AssetURL).
		Str("token", release.Headers["Authorization"]).
		Int64("size", release.AssetSize).
		Msg("Starting update download")

	// logpkg.LogDownloadStart(uo.logger, release.AssetURL, "", release.AssetSize)

	// Determine download destination path
	// Use a fixed temporary directory for downloads
	tmpDir := "/tmp/WXChannelsDownload"
	destPath := fmt.Sprintf("%s/%s", tmpDir, release.AssetName)

	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}

	// Configure download options with GitHub token if available
	// downloadOptions := DefaultDownloadOptions()
	// if uo.githubToken != "" && isGitHubURL(release.AssetURL) {
	// 	downloadOptions.Headers = map[string]string{
	// 		"Authorization": fmt.Sprintf("token %s", uo.githubToken),
	// 	}
	// 	uo.logger.Debug().Msg("Using GitHub token for download authentication")
	// }

	// Create download manager with custom options
	// downloadManager := NewUpdateDownloadManagerWithOptions(uo.logger, downloadOptions)

	// Download the update package
	err := uo.Download(
		ctx,
		release.AssetURL,
		release.Headers,
		destPath,
		release.Checksum,
		!release.NeedCheckChecksum,
		progressCallback,
	)
	if err != nil {
		// logpkg.LogDownloadError(uo.logger, err, release.AssetURL)
		return "", &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to download update package",
			Cause:    err,
			Context: map[string]interface{}{
				"version": release.Version,
				"url":     release.AssetURL,
			},
		}
	}

	uo.logger.Info().
		Str("path", destPath).
		Str("version", release.Version).
		Msg("Update download completed successfully")

	return destPath, nil
}

// func (a *UpdateDownloadManager) DownloadUpdate(ctx context.Context, release *types.UpdateCheckResult, callback func(progress types.DownloadProgress)) (string, error) {
// 	releaseInfo, err := a.orchestrator.CheckForUpdatesForce(ctx)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to check for updates: %w", err)
// 	}
// 	if releaseInfo.Version != release.Version {
// 		return "", fmt.Errorf("version mismatch: requested %s, but latest is %s", release.Version, releaseInfo.Version)
// 	}
// 	updatePath, err := a.orchestrator.DownloadUpdate(ctx, releaseInfo, func(progress types.DownloadProgress) {
// 		fmt.Printf("\rDownload progress: %.1f%% (%d/%d bytes)",
// 			progress.Percentage, progress.BytesDownloaded, progress.TotalBytes)
// 	})
// 	if err != nil {
// 		return "", fmt.Errorf("failed to download update: %w", err)
// 	}
// 	return updatePath, nil
// }
