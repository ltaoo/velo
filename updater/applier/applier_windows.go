//go:build windows
// +build windows

package applier

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

	// Create temporary directory for extraction
	tempDir := filepath.Join(os.TempDir(), "WXChannelsDownload")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		wu.triggerRollback(backupPath, execPath)
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
	if err := wu.ExtractArchive(updatePath, tempDir); err != nil {
		wu.triggerRollback(backupPath, execPath)
		return err
	}

	// Find the executable in the extracted files
	newExecPath, err := wu.findExecutable(tempDir)
	if err != nil {
		wu.triggerRollback(backupPath, execPath)
		return err
	}

	wu.logger.Info().
		Str("new_exec", newExecPath).
		Msg("Found new executable")

	// Try direct replacement first
	if err := wu.tryDirectReplace(newExecPath, execPath); err != nil {
		wu.logger.Warn().
			Err(err).
			Msg("Direct replacement failed, scheduling delayed replacement")

		// If direct replacement fails (file locked), schedule delayed replacement
		if err := wu.scheduleDelayedReplace(newExecPath, execPath); err != nil {
			wu.triggerRollback(backupPath, execPath)
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "failed to schedule delayed replacement",
				Cause:    err,
				Context: map[string]interface{}{
					"new_exec": newExecPath,
					"target":   execPath,
				},
			}
		}

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

// findExecutable finds the executable file in the extracted directory
func (wu *WindowsUpdater) findExecutable(dir string) (string, error) {
	var execPath string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for .exe files
		if !info.IsDir() && filepath.Ext(path) == ".exe" {
			execPath = path
			return filepath.SkipDir // Stop after finding first .exe
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
func (wu *WindowsUpdater) Restart(execPath string, args []string) error {
	wu.logger.Info().
		Str("exec", execPath).
		Strs("args", args).
		Msg("Restarting application")

	// Create command
	cmd := exec.Command(execPath, args...)

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
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

	wu.logger.Info().
		Int("pid", cmd.Process.Pid).
		Msg("New process started")

	// Exit current process
	os.Exit(0)

	return nil
}

// newPlatformUpdaterImpl creates a Windows-specific updater
func newPlatformUpdaterImpl(logger *zerolog.Logger) master.UpdateApplier {
	return NewWindowsUpdater(logger)
}
