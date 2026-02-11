package logger

import (
	"time"
	"github.com/ltaoo/velo/updater/types"

	"github.com/rs/zerolog"
)

// LogVersionCheck logs a version check operation
func LogVersionCheck(logger zerolog.Logger, currentVersion, latestVersion string, isNewer bool, source string) {
	logger.Info().
		Str("current_version", currentVersion).
		Str("latest_version", latestVersion).
		Bool("is_newer", isNewer).
		Str("source", source).
		Time("timestamp", time.Now()).
		Msg("Version check completed")
}

// LogVersionCheckStart logs the start of a version check
func LogVersionCheckStart(logger zerolog.Logger, currentVersion string, sourceCount int) {
	logger.Info().
		Str("current_version", currentVersion).
		Int("source_count", sourceCount).
		Time("timestamp", time.Now()).
		Msg("Starting version check")
}

// LogVersionCheckError logs a version check error
func LogVersionCheckError(logger zerolog.Logger, err error, source string) {
	if updateErr, ok := err.(*types.UpdateError); ok {
		logger.Error().
			Err(err).
			Str("source", source).
			Str("error_category", updateErr.Category.String()).
			Interface("context", updateErr.Context).
			Time("timestamp", time.Now()).
			Msg("Version check failed")
	} else {
		logger.Error().
			Err(err).
			Str("source", source).
			Time("timestamp", time.Now()).
			Msg("Version check failed")
	}
}

// LogDownloadStart logs the start of a download operation
func LogDownloadStart(logger zerolog.Logger, url string, destPath string, expectedSize int64) {
	logger.Info().
		Str("url", url).
		Str("dest_path", destPath).
		Int64("expected_size", expectedSize).
		Time("timestamp", time.Now()).
		Msg("Starting download")
}

// LogDownloadProgress logs download progress
func LogDownloadProgress(logger zerolog.Logger, progress types.DownloadProgress) {
	logger.Debug().
		Int64("bytes_downloaded", progress.BytesDownloaded).
		Int64("total_bytes", progress.TotalBytes).
		Float64("percentage", progress.Percentage).
		Int64("speed", progress.Speed).
		Time("timestamp", time.Now()).
		Msg("Download progress")
}

// LogDownloadComplete logs successful download completion
func LogDownloadComplete(logger zerolog.Logger, destPath string, size int64, checksum string, duration time.Duration) {
	logger.Info().
		Str("dest_path", destPath).
		Int64("size", size).
		Str("checksum", checksum).
		Dur("duration", duration).
		Time("timestamp", time.Now()).
		Msg("Download completed successfully")
}

// LogDownloadError logs a download error
func LogDownloadError(logger zerolog.Logger, err error, url string) {
	if updateErr, ok := err.(*types.UpdateError); ok {
		logger.Error().
			Err(err).
			Str("url", url).
			Str("error_category", updateErr.Category.String()).
			Interface("context", updateErr.Context).
			Time("timestamp", time.Now()).
			Msg("Download failed")
	} else {
		logger.Error().
			Err(err).
			Str("url", url).
			Time("timestamp", time.Now()).
			Msg("Download failed")
	}
}

// LogUpdateStart logs the start of an update operation
func LogUpdateStart(logger zerolog.Logger, currentVersion, newVersion string) {
	logger.Info().
		Str("current_version", currentVersion).
		Str("new_version", newVersion).
		Time("timestamp", time.Now()).
		Msg("Starting update operation")
}

// LogUpdateComplete logs successful update completion
func LogUpdateComplete(logger zerolog.Logger, newVersion string, duration time.Duration) {
	logger.Info().
		Str("new_version", newVersion).
		Dur("duration", duration).
		Time("timestamp", time.Now()).
		Msg("Update completed successfully")
}

// LogUpdateError logs an update error
func LogUpdateError(logger zerolog.Logger, err error, operation string) {
	if updateErr, ok := err.(*types.UpdateError); ok {
		logger.Error().
			Err(err).
			Str("operation", operation).
			Str("error_category", updateErr.Category.String()).
			Interface("context", updateErr.Context).
			Time("timestamp", time.Now()).
			Msg("Update operation failed")
	} else {
		logger.Error().
			Err(err).
			Str("operation", operation).
			Time("timestamp", time.Now()).
			Msg("Update operation failed")
	}
}

// LogSecurityWarning logs a security-related warning
func LogSecurityWarning(logger zerolog.Logger, message string, context map[string]interface{}) {
	logger.Warn().
		Str("security_warning", message).
		Interface("context", context).
		Time("timestamp", time.Now()).
		Msg("Security validation warning")
}

// LogSecurityError logs a security-related error
func LogSecurityError(logger zerolog.Logger, err error, operation string) {
	if updateErr, ok := err.(*types.UpdateError); ok {
		logger.Error().
			Err(err).
			Str("operation", operation).
			Str("error_category", updateErr.Category.String()).
			Interface("context", updateErr.Context).
			Time("timestamp", time.Now()).
			Msg("Security validation failed")
	} else {
		logger.Error().
			Err(err).
			Str("operation", operation).
			Time("timestamp", time.Now()).
			Msg("Security validation failed")
	}
}

// LogRollbackStart logs the start of a rollback operation
func LogRollbackStart(logger zerolog.Logger, reason string) {
	logger.Warn().
		Str("reason", reason).
		Time("timestamp", time.Now()).
		Msg("Starting rollback operation")
}

// LogRollbackComplete logs successful rollback completion
func LogRollbackComplete(logger zerolog.Logger) {
	logger.Info().
		Time("timestamp", time.Now()).
		Msg("Rollback completed successfully")
}

// LogRollbackError logs a rollback error
func LogRollbackError(logger zerolog.Logger, err error) {
	logger.Error().
		Err(err).
		Time("timestamp", time.Now()).
		Msg("Rollback failed")
}

// LogConfigLoad logs configuration loading
func LogConfigLoad(logger zerolog.Logger, configPath string, sourceCount int) {
	logger.Info().
		Str("config_path", configPath).
		Int("source_count", sourceCount).
		Time("timestamp", time.Now()).
		Msg("Configuration loaded")
}

// LogConfigError logs a configuration error
func LogConfigError(logger zerolog.Logger, err error, configPath string) {
	if updateErr, ok := err.(*types.UpdateError); ok {
		logger.Error().
			Err(err).
			Str("config_path", configPath).
			Str("error_category", updateErr.Category.String()).
			Interface("context", updateErr.Context).
			Time("timestamp", time.Now()).
			Msg("Configuration error")
	} else {
		logger.Error().
			Err(err).
			Str("config_path", configPath).
			Time("timestamp", time.Now()).
			Msg("Configuration error")
	}
}

// LogChecksumVerification logs checksum verification
func LogChecksumVerification(logger zerolog.Logger, filePath string, expected, actual string, match bool) {
	if match {
		logger.Info().
			Str("file_path", filePath).
			Str("checksum", actual).
			Time("timestamp", time.Now()).
			Msg("Checksum verification passed")
	} else {
		logger.Error().
			Str("file_path", filePath).
			Str("expected_checksum", expected).
			Str("actual_checksum", actual).
			Time("timestamp", time.Now()).
			Msg("Checksum verification failed")
	}
}

// LogBackupOperation logs backup operations
func LogBackupOperation(logger zerolog.Logger, operation string, sourcePath, backupPath string, success bool) {
	if success {
		logger.Info().
			Str("operation", operation).
			Str("source_path", sourcePath).
			Str("backup_path", backupPath).
			Time("timestamp", time.Now()).
			Msg("Backup operation completed")
	} else {
		logger.Error().
			Str("operation", operation).
			Str("source_path", sourcePath).
			Str("backup_path", backupPath).
			Time("timestamp", time.Now()).
			Msg("Backup operation failed")
	}
}

// LogArchiveExtraction logs archive extraction operations
func LogArchiveExtraction(logger zerolog.Logger, archivePath, destDir string, fileCount int, success bool) {
	if success {
		logger.Info().
			Str("archive_path", archivePath).
			Str("dest_dir", destDir).
			Int("file_count", fileCount).
			Time("timestamp", time.Now()).
			Msg("Archive extraction completed")
	} else {
		logger.Error().
			Str("archive_path", archivePath).
			Str("dest_dir", destDir).
			Time("timestamp", time.Now()).
			Msg("Archive extraction failed")
	}
}

// LogCleanup logs cleanup operations
func LogCleanup(logger zerolog.Logger, paths []string, errors []string) {
	if len(errors) == 0 {
		logger.Info().
			Strs("cleaned_paths", paths).
			Time("timestamp", time.Now()).
			Msg("Cleanup completed successfully")
	} else {
		logger.Warn().
			Strs("cleaned_paths", paths).
			Strs("errors", errors).
			Time("timestamp", time.Now()).
			Msg("Cleanup completed with errors")
	}
}

// LogManifestParse logs manifest parsing operations
func LogManifestParse(logger zerolog.Logger, version string, assetCount int, success bool) {
	if success {
		logger.Info().
			Str("version", version).
			Int("asset_count", assetCount).
			Time("timestamp", time.Now()).
			Msg("Manifest parsed successfully")
	} else {
		logger.Error().
			Time("timestamp", time.Now()).
			Msg("Manifest parsing failed")
	}
}

// LogPlatformDetection logs platform detection
func LogPlatformDetection(logger zerolog.Logger, platformKey string) {
	logger.Debug().
		Str("platform_key", platformKey).
		Time("timestamp", time.Now()).
		Msg("Platform detected")
}

// LogHTTPRequest logs HTTP request details
func LogHTTPRequest(logger zerolog.Logger, method, url string, statusCode int) {
	logger.Debug().
		Str("method", method).
		Str("url", url).
		Int("status_code", statusCode).
		Time("timestamp", time.Now()).
		Msg("HTTP request completed")
}

// LogRetryAttempt logs retry attempts
func LogRetryAttempt(logger zerolog.Logger, operation string, attempt, maxAttempts int, err error) {
	logger.Warn().
		Str("operation", operation).
		Int("attempt", attempt).
		Int("max_attempts", maxAttempts).
		Err(err).
		Time("timestamp", time.Now()).
		Msg("Retry attempt")
}
