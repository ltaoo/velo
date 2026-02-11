package applier

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/ulikunitz/xz"

	"github.com/ltaoo/velo/updater/types"
)

// BaseApplier provides common functionality for platform-specific updaters
type BaseApplier struct {
	logger zerolog.Logger
}

// NewBaseApplier creates a new base updater with common functionality
func NewBaseApplier(logger zerolog.Logger) *BaseApplier {
	return &BaseApplier{
		logger: logger.With().Str("component", "base-updater").Logger(),
	}
}

// Backup creates a backup of the current executable
func (bu *BaseApplier) Backup(execPath, backupPath string) error {
	bu.logger.Info().
		Str("source", execPath).
		Str("backup", backupPath).
		Msg("Creating backup")

	// Check if source file exists and get its info
	srcInfo, err := os.Stat(execPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "source executable not found",
			Cause:    err,
			Context: map[string]interface{}{
				"exec_path": execPath,
			},
		}
	}

	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create backup directory",
			Cause:    err,
			Context: map[string]interface{}{
				"backup_dir": backupDir,
			},
		}
	}

	// Open source file
	srcFile, err := os.Open(execPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open source file",
			Cause:    err,
			Context: map[string]interface{}{
				"exec_path": execPath,
			},
		}
	}
	defer srcFile.Close()

	// Create backup file with the same permissions as the source
	dstFile, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create backup file",
			Cause:    err,
			Context: map[string]interface{}{
				"backup_path": backupPath,
			},
		}
	}
	defer dstFile.Close()

	// Copy file contents
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		// Clean up partial backup
		os.Remove(backupPath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to copy file to backup",
			Cause:    err,
			Context: map[string]interface{}{
				"exec_path":   execPath,
				"backup_path": backupPath,
			},
		}
	}

	// Verify size
	if written != srcInfo.Size() {
		os.Remove(backupPath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "backup file size mismatch",
			Context: map[string]interface{}{
				"expected": srcInfo.Size(),
				"actual":   written,
			},
		}
	}

	bu.logger.Info().
		Str("backup", backupPath).
		Int64("size", written).
		Msg("Backup created successfully")

	return nil
}

// Restore restores the executable from backup
func (bu *BaseApplier) Restore(backupPath, execPath string) error {
	bu.logger.Info().
		Str("backup", backupPath).
		Str("target", execPath).
		Msg("Restoring from backup")

	// Check if backup exists and get its info
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "backup file not found",
			Cause:    err,
			Context: map[string]interface{}{
				"backup_path": backupPath,
			},
		}
	}

	// Remove current file if it exists
	if _, err := os.Stat(execPath); err == nil {
		if err := os.Remove(execPath); err != nil {
			bu.logger.Warn().Err(err).Msg("Failed to remove current file before restore")
		}
	}

	// Copy backup to target
	srcFile, err := os.Open(backupPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open backup file",
			Cause:    err,
			Context: map[string]interface{}{
				"backup_path": backupPath,
			},
		}
	}
	defer srcFile.Close()

	// Create destination file with the same permissions as the backup
	dstFile, err := os.OpenFile(execPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, backupInfo.Mode())
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create target file",
			Cause:    err,
			Context: map[string]interface{}{
				"exec_path": execPath,
			},
		}
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to restore from backup",
			Cause:    err,
			Context: map[string]interface{}{
				"backup_path": backupPath,
				"exec_path":   execPath,
			},
		}
	}

	bu.logger.Info().Msg("Restore completed successfully")

	return nil
}

// Cleanup removes backup and temporary files
func (bu *BaseApplier) Cleanup(paths ...string) error {
	bu.logger.Info().
		Int("count", len(paths)).
		Msg("Cleaning up files")

	var errors []string
	for _, path := range paths {
		if path == "" {
			continue
		}

		if err := os.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				bu.logger.Warn().
					Err(err).
					Str("path", path).
					Msg("Failed to remove file")
				errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			}
		} else {
			bu.logger.Debug().
				Str("path", path).
				Msg("Removed file")
		}
	}

	if len(errors) > 0 {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to cleanup some files",
			Context: map[string]interface{}{
				"errors": errors,
			},
		}
	}

	bu.logger.Info().Msg("Cleanup completed successfully")
	return nil
}

// ExtractArchive extracts an archive file to a destination directory
// Supports .zip, .tar.gz, and .tar.xz formats
func (bu *BaseApplier) ExtractArchive(archivePath, destDir string) error {
	bu.logger.Info().
		Str("archive", archivePath).
		Str("dest", destDir).
		Msg("Extracting archive")

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create destination directory",
			Cause:    err,
			Context: map[string]interface{}{
				"dest_dir": destDir,
			},
		}
	}

	// Determine archive type by extension
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch {
	case ext == ".zip":
		return bu.extractZip(archivePath, destDir)
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		return bu.extractTarGz(archivePath, destDir)
	case strings.HasSuffix(archivePath, ".tar.xz"):
		return bu.extractTarXz(archivePath, destDir)
	default:
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "unsupported archive format",
			Context: map[string]interface{}{
				"archive": archivePath,
				"ext":     ext,
			},
		}
	}
}

// extractZip extracts a ZIP archive
func (bu *BaseApplier) extractZip(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "failed to open zip archive",
			Cause:    err,
			Context: map[string]interface{}{
				"archive": archivePath,
			},
		}
	}
	defer reader.Close()

	for _, file := range reader.File {
		if err := bu.extractZipFile(file, destDir); err != nil {
			return err
		}
	}

	bu.logger.Info().
		Int("files", len(reader.File)).
		Msg("ZIP extraction completed")

	return nil
}

// extractZipFile extracts a single file from a ZIP archive
func (bu *BaseApplier) extractZipFile(file *zip.File, destDir string) error {
	// Construct destination path
	destPath := filepath.Join(destDir, file.Name)

	// Prevent path traversal attacks
	if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return &types.UpdateError{
			Category: types.ErrCategorySecurity,
			Message:  "invalid file path in archive (path traversal attempt)",
			Context: map[string]interface{}{
				"file": file.Name,
			},
		}
	}

	// Handle directories
	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, file.Mode())
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create parent directory",
			Cause:    err,
			Context: map[string]interface{}{
				"path": filepath.Dir(destPath),
			},
		}
	}

	// Open source file
	srcFile, err := file.Open()
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open file in archive",
			Cause:    err,
			Context: map[string]interface{}{
				"file": file.Name,
			},
		}
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create destination file",
			Cause:    err,
			Context: map[string]interface{}{
				"path": destPath,
			},
		}
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to extract file",
			Cause:    err,
			Context: map[string]interface{}{
				"file": file.Name,
			},
		}
	}

	return nil
}

// extractTarGz extracts a .tar.gz archive
func (bu *BaseApplier) extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open tar.gz archive",
			Cause:    err,
			Context: map[string]interface{}{
				"archive": archivePath,
			},
		}
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "failed to create gzip reader",
			Cause:    err,
			Context: map[string]interface{}{
				"archive": archivePath,
			},
		}
	}
	defer gzReader.Close()

	return bu.extractTar(gzReader, destDir)
}

// extractTarXz extracts a .tar.xz archive
func (bu *BaseApplier) extractTarXz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open tar.xz archive",
			Cause:    err,
			Context: map[string]interface{}{
				"archive": archivePath,
			},
		}
	}
	defer file.Close()

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "failed to create xz reader",
			Cause:    err,
			Context: map[string]interface{}{
				"archive": archivePath,
			},
		}
	}

	return bu.extractTar(xzReader, destDir)
}

// extractTar extracts a tar archive from a reader
func (bu *BaseApplier) extractTar(reader io.Reader, destDir string) error {
	tarReader := tar.NewReader(reader)
	fileCount := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return &types.UpdateError{
				Category: types.ErrCategoryValidation,
				Message:  "failed to read tar header",
				Cause:    err,
			}
		}

		if err := bu.extractTarFile(tarReader, header, destDir); err != nil {
			return err
		}
		fileCount++
	}

	bu.logger.Info().
		Int("files", fileCount).
		Msg("TAR extraction completed")

	return nil
}

// extractTarFile extracts a single file from a tar archive
func (bu *BaseApplier) extractTarFile(tarReader *tar.Reader, header *tar.Header, destDir string) error {
	// Construct destination path
	destPath := filepath.Join(destDir, header.Name)

	// Prevent path traversal attacks
	if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return &types.UpdateError{
			Category: types.ErrCategorySecurity,
			Message:  "invalid file path in archive (path traversal attempt)",
			Context: map[string]interface{}{
				"file": header.Name,
			},
		}
	}

	// Handle different file types
	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(destPath, os.FileMode(header.Mode))

	case tar.TypeReg:
		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "failed to create parent directory",
				Cause:    err,
				Context: map[string]interface{}{
					"path": filepath.Dir(destPath),
				},
			}
		}

		// Create destination file
		dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "failed to create destination file",
				Cause:    err,
				Context: map[string]interface{}{
					"path": destPath,
				},
			}
		}
		defer dstFile.Close()

		// Copy contents
		if _, err := io.Copy(dstFile, tarReader); err != nil {
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "failed to extract file",
				Cause:    err,
				Context: map[string]interface{}{
					"file": header.Name,
				},
			}
		}

	default:
		bu.logger.Debug().
			Str("file", header.Name).
			Uint8("type", header.Typeflag).
			Msg("Skipping unsupported file type")
	}

	return nil
}
