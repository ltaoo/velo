//go:build !windows && !darwin
// +build !windows,!darwin

package applier

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/ltaoo/velo/updater/master"
	"github.com/ltaoo/velo/updater/types"
)

// UnixUpdater implements PlatformUpdater for Unix-like systems (Linux, macOS)
type UnixUpdater struct {
	*BaseApplier
}

// NewUnixUpdater creates a new Unix-specific updater
func NewUnixUpdater(logger *zerolog.Logger) *UnixUpdater {
	return &UnixUpdater{
		BaseApplier: NewBaseApplier(logger.With().Str("platform", "unix").Logger()),
	}
}

// Apply applies the update by extracting the archive and replacing the executable
// If any step fails, it automatically triggers rollback from backup
func (uu *UnixUpdater) Apply(updatePath, execPath string) error {
	uu.logger.Info().
		Str("update", updatePath).
		Str("target", execPath).
		Msg("Applying Unix update")

	// Create backup path
	backupPath := execPath + ".backup"

	// Create backup before applying update
	uu.logger.Info().Msg("Creating backup before applying update")
	if err := uu.Backup(execPath, backupPath); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create backup before update",
			Cause:    err,
			Context: map[string]interface{}{
				"exec_path":   execPath,
				"backup_path": backupPath,
			},
		}
	}

	// Ensure backup is cleaned up on success, or used for rollback on failure
	defer func() {
		if _, err := os.Stat(backupPath); err == nil {
			// Backup still exists, clean it up
			uu.Cleanup(backupPath)
		}
	}()

	// Create temporary directory for extraction
	tempDir := filepath.Join(os.TempDir(), "WXChannelsDownload")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		uu.triggerRollback(backupPath, execPath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create temporary extraction directory",
			Cause:    err,
			Context: map[string]interface{}{
				"temp_dir": tempDir,
			},
		}
	}
	// defer os.RemoveAll(tempDir) // Commented out to keep files for inspection

	// Extract the update archive
	if err := uu.ExtractArchive(updatePath, tempDir); err != nil {
		uu.triggerRollback(backupPath, execPath)
		return err
	}

	// Find the executable in the extracted files
	newExecPath, err := uu.findExecutable(tempDir)
	if err != nil {
		uu.triggerRollback(backupPath, execPath)
		return err
	}

	uu.logger.Info().
		Str("new_exec", newExecPath).
		Msg("Found new executable")

	// Get the original file permissions
	origInfo, err := os.Stat(execPath)
	if err != nil && !os.IsNotExist(err) {
		uu.triggerRollback(backupPath, execPath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to get original file info",
			Cause:    err,
			Context: map[string]interface{}{
				"target": execPath,
			},
		}
	}

	// Remove the old executable
	if err := os.Remove(execPath); err != nil {
		if !os.IsNotExist(err) {
			uu.triggerRollback(backupPath, execPath)
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "failed to remove old executable",
				Cause:    err,
				Context: map[string]interface{}{
					"target": execPath,
				},
			}
		}
	}

	// Copy the new executable to the target location
	if err := uu.copyFile(newExecPath, execPath); err != nil {
		uu.triggerRollback(backupPath, execPath)
		return err
	}

	// Ensure executable permissions
	mode := os.FileMode(0755)
	if origInfo != nil {
		mode = origInfo.Mode()
	}

	if err := os.Chmod(execPath, mode); err != nil {
		uu.triggerRollback(backupPath, execPath)
		return &types.UpdateError{
			Category: types.ErrCategoryPermission,
			Message:  "failed to set executable permissions",
			Cause:    err,
			Context: map[string]interface{}{
				"target": execPath,
				"mode":   mode,
			},
		}
	}

	// Verify code signature on macOS
	if err := uu.VerifyCodeSignature(execPath); err != nil {
		// If signature verification fails, trigger rollback
		uu.triggerRollback(backupPath, execPath)
		return err
	}

	// Verify the updated file integrity
	if err := uu.verifyFileIntegrity(execPath); err != nil {
		uu.triggerRollback(backupPath, execPath)
		return err
	}

	uu.logger.Info().Msg("Update applied successfully")
	return nil
}

// triggerRollback attempts to restore from backup when update fails
func (uu *UnixUpdater) triggerRollback(backupPath, execPath string) {
	uu.logger.Warn().
		Str("backup", backupPath).
		Str("target", execPath).
		Msg("Update failed, triggering rollback")

	if err := uu.Restore(backupPath, execPath); err != nil {
		uu.logger.Error().
			Err(err).
			Msg("Rollback failed - system may be in inconsistent state")
		return
	}

	// Verify rollback integrity
	if err := uu.verifyFileIntegrity(execPath); err != nil {
		uu.logger.Error().
			Err(err).
			Msg("Rollback completed but file integrity verification failed")
		return
	}

	uu.logger.Info().Msg("Rollback completed successfully, original executable restored")
}

// verifyFileIntegrity verifies that the executable file is valid
func (uu *UnixUpdater) verifyFileIntegrity(execPath string) error {
	// Check if file exists
	info, err := os.Stat(execPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "executable file not found after operation",
			Cause:    err,
			Context: map[string]interface{}{
				"exec_path": execPath,
			},
		}
	}

	// Check if file is not empty
	if info.Size() == 0 {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "executable file is empty",
			Context: map[string]interface{}{
				"exec_path": execPath,
			},
		}
	}

	// Check if file has execute permissions
	if info.Mode()&0111 == 0 {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "executable file does not have execute permissions",
			Context: map[string]interface{}{
				"exec_path": execPath,
				"mode":      info.Mode(),
			},
		}
	}

	uu.logger.Debug().
		Str("path", execPath).
		Int64("size", info.Size()).
		Str("mode", info.Mode().String()).
		Msg("File integrity verified")

	return nil
}

// copyFile copies a file from src to dst
func (uu *UnixUpdater) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open source file",
			Cause:    err,
			Context: map[string]interface{}{
				"source": src,
			},
		}
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create destination file",
			Cause:    err,
			Context: map[string]interface{}{
				"dest": dst,
			},
		}
	}
	defer dstFile.Close()

	// Copy file contents
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	written := int64(0)
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
				return &types.UpdateError{
					Category: types.ErrCategoryFileSystem,
					Message:  "failed to write to destination file",
					Cause:    writeErr,
					Context: map[string]interface{}{
						"dest": dst,
					},
				}
			}
			written += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "failed to read from source file",
				Cause:    err,
				Context: map[string]interface{}{
					"source": src,
				},
			}
		}
	}

	// Verify size
	if written != srcInfo.Size() {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "file size mismatch after copy",
			Context: map[string]interface{}{
				"expected": srcInfo.Size(),
				"actual":   written,
			},
		}
	}

	return nil
}

// findExecutable finds the executable file in the extracted directory
func (uu *UnixUpdater) findExecutable(dir string) (string, error) {
	var execPath string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for executable files (files with execute permission)
		if !info.IsDir() && info.Mode()&0111 != 0 {
			execPath = path
			return filepath.SkipDir // Stop after finding first executable
		}

		return nil
	})

	if err != nil {
		return "", &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to search for executable",
			Cause:    err,
			Context: map[string]interface{}{
				"dir": dir,
			},
		}
	}

	if execPath == "" {
		return "", &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "no executable found in update archive",
			Context: map[string]interface{}{
				"dir": dir,
			},
		}
	}

	return execPath, nil
}

// Restart restarts the application with the given arguments
func (uu *UnixUpdater) Restart(execPath string, args []string) error {
	uu.logger.Info().
		Str("exec", execPath).
		Strs("args", args).
		Msg("Restarting application")

	// Create command
	cmd := exec.Command(execPath, args...)

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// Start the new process
	if err := cmd.Start(); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to start new process",
			Cause:    err,
			Context: map[string]interface{}{
				"exec": execPath,
				"args": args,
			},
		}
	}

	uu.logger.Info().
		Int("pid", cmd.Process.Pid).
		Msg("New process started")

	// Exit current process
	os.Exit(0)

	return nil
}

// newPlatformUpdaterImpl creates a Unix-specific updater
func newPlatformUpdaterImpl(logger *zerolog.Logger) master.UpdateApplier {
	return NewUnixUpdater(logger)
}
