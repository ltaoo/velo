//go:build windows
// +build windows

package applier

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/ltaoo/velo/updater/master"
	"github.com/ltaoo/velo/updater/types"
	"github.com/rs/zerolog"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW  = kernel32.NewProc("MoveFileExW")
	procGetLastError = kernel32.NewProc("GetLastError")
)

const (
	MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
	MOVEFILE_REPLACE_EXISTING   = 0x1
)

// WindowsUpdater implements PlatformUpdater for Windows
type WindowsUpdater struct {
	logger *zerolog.Logger
	*BaseApplier
}

// NewWindowsUpdater creates a new Windows-specific updater
func NewWindowsUpdater(parent_logger *zerolog.Logger) *WindowsUpdater {
	logger := parent_logger.With().Str("component", "windows-updater").Logger()
	return &WindowsUpdater{
		BaseApplier: NewBaseApplier(logger.With().Str("platform", "windows").Logger()),
		logger:      &logger,
	}
}

// Apply applies the update by extracting the archive and replacing the executable
// If any step fails, it automatically triggers rollback from backup
func (wu *WindowsUpdater) Apply(updatePath, execPath string) error {
	wu.logger.Info().
		Str("update", updatePath).
		Str("target", execPath).
		Msg("Applying Windows update")

	// Create backup path
	backupPath := execPath + ".backup"

	// Create backup before applying update
	wu.logger.Info().Msg("Creating backup before applying update")
	if err := wu.Backup(execPath, backupPath); err != nil {
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
			wu.Cleanup(backupPath)
		}
	}()

	// Create an isolated temporary directory so stale files cannot be selected.
	temp_dir, err := create_update_extraction_dir()
	if err != nil {
		wu.triggerRollback(backupPath, execPath)
		return err
	}
	cleanup_temp_dir := true
	defer func() {
		if cleanup_temp_dir {
			_ = os.RemoveAll(temp_dir)
		}
	}()

	// Extract the update archive
	if err := wu.ExtractArchive(updatePath, temp_dir); err != nil {
		wu.triggerRollback(backupPath, execPath)
		return err
	}

	// Prefer the current target name; only a single candidate may be used as fallback.
	new_exec_path, err := find_update_executable(temp_dir, filepath.Base(execPath), false, is_platform_executable)
	if err != nil {
		wu.triggerRollback(backupPath, execPath)
		return err
	}

	wu.logger.Info().
		Str("new_exec", new_exec_path).
		Msg("Found new executable")

	// Try direct replacement first
	if err := wu.tryDirectReplace(new_exec_path, execPath); err != nil {
		wu.logger.Warn().
			Err(err).
			Msg("Direct replacement failed, scheduling delayed replacement")

		// If direct replacement fails (file locked), schedule delayed replacement
		if err := wu.scheduleDelayedReplace(new_exec_path, execPath); err != nil {
			wu.triggerRollback(backupPath, execPath)
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "failed to schedule delayed replacement",
				Cause:    err,
				Context: map[string]interface{}{
					"new_exec": new_exec_path,
					"target":   execPath,
				},
			}
		}

		cleanup_temp_dir = false
		wu.logger.Info().Msg("Update scheduled for next restart")
		return nil
	}

	// Verify the updated file integrity
	if err := wu.verifyFileIntegrity(execPath); err != nil {
		wu.triggerRollback(backupPath, execPath)
		return err
	}

	wu.logger.Info().Msg("Update applied successfully")
	return nil
}

// triggerRollback attempts to restore from backup when update fails
func (wu *WindowsUpdater) triggerRollback(backupPath, execPath string) {
	wu.logger.Warn().
		Str("backup", backupPath).
		Str("target", execPath).
		Msg("Update failed, triggering rollback")

	if err := wu.Restore(backupPath, execPath); err != nil {
		wu.logger.Error().
			Err(err).
			Msg("Rollback failed - system may be in inconsistent state")
		return
	}

	// Verify rollback integrity
	if err := wu.verifyFileIntegrity(execPath); err != nil {
		wu.logger.Error().
			Err(err).
			Msg("Rollback completed but file integrity verification failed")
		return
	}

	wu.logger.Info().Msg("Rollback completed successfully, original executable restored")
}

// verifyFileIntegrity verifies that the executable file is valid
func (wu *WindowsUpdater) verifyFileIntegrity(execPath string) error {
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

	// Check if file has .exe extension
	if filepath.Ext(execPath) != ".exe" {
		wu.logger.Warn().
			Str("path", execPath).
			Msg("Executable does not have .exe extension")
	}

	wu.logger.Debug().
		Str("path", execPath).
		Int64("size", info.Size()).
		Str("mode", info.Mode().String()).
		Msg("File integrity verified")

	return nil
}

// tryDirectReplace attempts to directly replace the executable
func (wu *WindowsUpdater) tryDirectReplace(newExecPath, execPath string) error {
	// Remove the old executable
	if err := os.Remove(execPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	// Copy the new executable to the target location
	srcFile, err := os.Open(newExecPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open new executable",
			Cause:    err,
			Context: map[string]interface{}{
				"new_exec": newExecPath,
			},
		}
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(execPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create target executable",
			Cause:    err,
			Context: map[string]interface{}{
				"target": execPath,
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
					Message:  "failed to write to target executable",
					Cause:    writeErr,
					Context: map[string]interface{}{
						"target": execPath,
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
				Message:  "failed to read from new executable",
				Cause:    err,
				Context: map[string]interface{}{
					"new_exec": newExecPath,
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

// scheduleDelayedReplace schedules a file replacement on next reboot using MoveFileEx
func (wu *WindowsUpdater) scheduleDelayedReplace(newExecPath, execPath string) error {
	// Convert paths to UTF-16
	newExecPathUTF16, err := syscall.UTF16PtrFromString(newExecPath)
	if err != nil {
		return err
	}

	execPathUTF16, err := syscall.UTF16PtrFromString(execPath)
	if err != nil {
		return err
	}

	// Call MoveFileExW with MOVEFILE_DELAY_UNTIL_REBOOT flag
	ret, _, err := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(newExecPathUTF16)),
		uintptr(unsafe.Pointer(execPathUTF16)),
		uintptr(MOVEFILE_DELAY_UNTIL_REBOOT|MOVEFILE_REPLACE_EXISTING),
	)

	if ret == 0 {
		return fmt.Errorf("MoveFileExW failed: %v", err)
	}

	return nil
}

func is_platform_executable(path string, _ os.FileInfo) bool {
	return strings.EqualFold(filepath.Ext(path), ".exe")
}

// newPlatformUpdaterImpl creates a Windows-specific updater
func newPlatformUpdaterImpl(logger *zerolog.Logger) master.UpdateApplier {
	return NewWindowsUpdater(logger)
}
