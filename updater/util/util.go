package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/ltaoo/velo/updater/types"
	"github.com/rs/zerolog"
)

// semver_regex matches valid semver strings (without v prefix)
var semver_regex = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

var date_version_regex = regexp.MustCompile(`^\d{6}(\d{2})?$`)

// isValidSemver checks if a version string (without v prefix) is valid semver
func IsValidSemver(version string) bool {
	return semver_regex.MatchString(version)
}

// IsValidVersion accepts semantic versions and the date-based YYMMDD or
// YYMMDDNN versions commonly used by command-line applications.
func IsValidVersion(version string) bool {
	normalized_version := normalize_version(version)
	if IsValidSemver(normalized_version) {
		return true
	}
	_, _, ok := parse_date_version(normalized_version)
	return ok
}

func CompareVersions(current, latest string) (bool, error) {
	current_date, current_revision, current_is_date := parse_date_version(normalize_version(current))
	latest_date, latest_revision, latest_is_date := parse_date_version(normalize_version(latest))
	if current_is_date || latest_is_date {
		if !current_is_date {
			return false, invalid_version_error("current", current, nil)
		}
		if !latest_is_date {
			return false, invalid_version_error("latest", latest, nil)
		}
		if latest_date.After(current_date) {
			return true, nil
		}
		if latest_date.Equal(current_date) {
			return latest_revision > current_revision, nil
		}
		return false, nil
	}

	current_ver, err := semver.Parse(normalize_version(current))
	if err != nil {
		return false, invalid_version_error("current", current, err)
	}

	latest_ver, err := semver.Parse(normalize_version(latest))
	if err != nil {
		return false, invalid_version_error("latest", latest, err)
	}

	return latest_ver.GT(current_ver), nil
}

func normalize_version(version string) string {
	return strings.TrimPrefix(version, "v")
}

func parse_date_version(version string) (time.Time, int, bool) {
	if !date_version_regex.MatchString(version) {
		return time.Time{}, 0, false
	}
	date, err := time.Parse("060102", version[:6])
	if err != nil {
		return time.Time{}, 0, false
	}
	revision := 0
	if len(version) == 8 {
		revision, err = strconv.Atoi(version[6:])
		if err != nil {
			return time.Time{}, 0, false
		}
	}
	return date, revision, true
}

func invalid_version_error(kind string, version string, cause error) error {
	return &types.UpdateError{
		Category: types.ErrCategoryValidation,
		Message:  fmt.Sprintf("invalid %s version format", kind),
		Cause:    cause,
		Context: map[string]interface{}{
			kind + "_version": version,
		},
	}
}

// ExecutableValidator provides security validation for executable files
type ExecutableValidator struct {
	logger zerolog.Logger
}

// NewExecutableValidator creates a new executable validator
func NewExecutableValidator(logger zerolog.Logger) *ExecutableValidator {
	return &ExecutableValidator{
		logger: logger.With().Str("component", "security-validator").Logger(),
	}
}

// ValidateExecutable validates the integrity and format of an executable file
// It checks for valid PE (Windows), ELF (Linux), or Mach-O (macOS) format
func (ev *ExecutableValidator) ValidateExecutable(path string) error {
	ev.logger.Info().
		Str("path", path).
		Msg("Validating executable file")

	// Check if file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return ev.securityError("executable file not found", err, map[string]interface{}{
			"path": path,
		})
	}

	// Check if file is not empty
	if fileInfo.Size() == 0 {
		return ev.securityError("executable file is empty", nil, map[string]interface{}{
			"path": path,
		})
	}

	// Open file for reading
	file, err := os.Open(path)
	if err != nil {
		return ev.securityError("failed to open executable file", err, map[string]interface{}{
			"path": path,
		})
	}
	defer file.Close()

	// Read file header (first 4 bytes is enough to identify format)
	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		return ev.securityError("failed to read executable header", err, map[string]interface{}{
			"path": path,
		})
	}

	// Validate based on magic bytes
	if err := ev.validateFormat(header, path); err != nil {
		return err
	}

	ev.logger.Info().
		Str("path", path).
		Msg("Executable validation passed")

	return nil
}

// validateFormat validates the executable format based on magic bytes
func (ev *ExecutableValidator) validateFormat(header []byte, path string) error {
	// Check for PE format (Windows) - starts with "MZ" (0x4D 0x5A)
	if len(header) >= 2 && header[0] == 0x4D && header[1] == 0x5A {
		ev.logger.Debug().Msg("Detected PE (Windows) executable format")
		return nil
	}

	// Check for ELF format (Linux) - starts with 0x7F 'E' 'L' 'F'
	if len(header) >= 4 && header[0] == 0x7F && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
		ev.logger.Debug().Msg("Detected ELF (Linux) executable format")
		return nil
	}

	// Check for Mach-O format (macOS) - multiple possible magic numbers
	if len(header) >= 4 {
		magic := binary.LittleEndian.Uint32(header)
		// Mach-O 32-bit
		if magic == 0xFEEDFACE || magic == 0xCEFAEDFE {
			ev.logger.Debug().Msg("Detected Mach-O 32-bit (macOS) executable format")
			return nil
		}
		// Mach-O 64-bit
		if magic == 0xFEEDFACF || magic == 0xCFFAEDFE {
			ev.logger.Debug().Msg("Detected Mach-O 64-bit (macOS) executable format")
			return nil
		}
		// Universal binary (Fat Mach-O)
		if magic == 0xCAFEBABE || magic == 0xBEBAFECA {
			ev.logger.Debug().Msg("Detected Universal (Fat Mach-O) executable format")
			return nil
		}
	}

	// Unknown format
	return ev.securityError("unrecognized executable format", nil, map[string]interface{}{
		"path":   path,
		"header": fmt.Sprintf("%X", header),
	})
}

// securityError creates a security-category UpdateError
func (ev *ExecutableValidator) securityError(message string, cause error, context map[string]interface{}) error {
	ev.logger.Warn().
		Err(cause).
		Interface("context", context).
		Msg(message)

	return &types.UpdateError{
		Category: types.ErrCategorySecurity,
		Message:  message,
		Cause:    cause,
		Context:  context,
	}
}

// HandleSecurityFailure handles security validation failures by logging and cleaning up
func (ev *ExecutableValidator) HandleSecurityFailure(err error, tempFiles ...string) error {
	ev.logger.Error().
		Err(err).
		Strs("temp_files", tempFiles).
		Msg("Security validation failed - cleaning up")

	// Log security warning
	if updateErr, ok := err.(*types.UpdateError); ok {
		ev.logger.Warn().
			Str("category", fmt.Sprintf("%d", updateErr.Category)).
			Interface("context", updateErr.Context).
			Msg("Security validation failure details")
	}

	// Clean up temporary files
	var cleanupErrors []string
	for _, path := range tempFiles {
		if path == "" {
			continue
		}

		if removeErr := os.Remove(path); removeErr != nil {
			if !os.IsNotExist(removeErr) {
				ev.logger.Warn().
					Err(removeErr).
					Str("path", path).
					Msg("Failed to remove temporary file during security cleanup")
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("%s: %v", path, removeErr))
			}
		} else {
			ev.logger.Debug().
				Str("path", path).
				Msg("Removed temporary file during security cleanup")
		}
	}

	// If cleanup had errors, wrap them with the original error
	if len(cleanupErrors) > 0 {
		return &types.UpdateError{
			Category: types.ErrCategorySecurity,
			Message:  "security validation failed and cleanup had errors",
			Cause:    err,
			Context: map[string]interface{}{
				"cleanup_errors": cleanupErrors,
			},
		}
	}

	return err
}

// ValidateChecksumMatch validates that a file's checksum matches the expected value
// This is a helper for security validation during downloads
func (ev *ExecutableValidator) ValidateChecksumMatch(actualChecksum, expectedChecksum string) error {
	if actualChecksum != expectedChecksum {
		return ev.securityError("checksum mismatch", nil, map[string]interface{}{
			"expected": expectedChecksum,
			"actual":   actualChecksum,
		})
	}
	return nil
}

// ValidateHTTPS validates that a URL uses HTTPS protocol
func (ev *ExecutableValidator) ValidateHTTPS(url string) error {
	if len(url) < 8 || url[:8] != "https://" {
		return ev.securityError("non-HTTPS URL rejected", nil, map[string]interface{}{
			"url": url,
		})
	}
	return nil
}

// ValidateFileSize validates that a file size matches the expected size
func (ev *ExecutableValidator) ValidateFileSize(path string, expectedSize int64) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return ev.securityError("failed to stat file for size validation", err, map[string]interface{}{
			"path": path,
		})
	}

	if fileInfo.Size() != expectedSize {
		return ev.securityError("file size mismatch", nil, map[string]interface{}{
			"path":     path,
			"expected": expectedSize,
			"actual":   fileInfo.Size(),
		})
	}

	return nil
}

// ValidateNoPathTraversal validates that a path doesn't contain path traversal attempts
func (ev *ExecutableValidator) ValidateNoPathTraversal(path, baseDir string) error {
	// This is already handled in the archive extraction code, but provided as a utility
	cleanPath := bytes.TrimSpace([]byte(path))
	if bytes.Contains(cleanPath, []byte("..")) {
		return ev.securityError("path traversal attempt detected", nil, map[string]interface{}{
			"path":     path,
			"base_dir": baseDir,
		})
	}
	return nil
}

// ValidateExecutable is a package-level convenience function for validating executables
// It creates a temporary validator and validates the executable
func ValidateExecutable(path string) error {
	// Create a no-op logger for this convenience function
	logger := zerolog.Nop()
	validator := NewExecutableValidator(logger)
	return validator.ValidateExecutable(path)
}

// GetExecutablePath returns the path to the current executable
func GetExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return execPath, nil
}

// isGitHubURL checks if a URL is from GitHub
func IsGitHubURL(url string) bool {
	return strings.Contains(url, "github.com") || strings.Contains(url, "githubusercontent.com")
}

// isGitHubAPIURL checks if a URL is a GitHub API URL for asset download
func IsGitHubAPIURL(urlStr string) bool {
	return strings.Contains(urlStr, "api.github.com/repos/") && strings.Contains(urlStr, "/releases/assets/")
}
