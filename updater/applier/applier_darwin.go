//go:build darwin
// +build darwin

package applier

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"github.com/ltaoo/velo/updater/master"
	"github.com/ltaoo/velo/updater/types"

	"github.com/rs/zerolog"
)

// DarwinUpdater implements PlatformUpdater for macOS
type DarwinUpdater struct {
	logs []string
	*BaseApplier
}

// NewDarwinUpdater creates a new macOS-specific updater
func NewDarwinUpdater(logger *zerolog.Logger) *DarwinUpdater {
	return &DarwinUpdater{
		BaseApplier: NewBaseApplier(logger.With().Str("platform", "darwin").Logger()),
	}
}

// Apply applies the update by extracting .app from DMG and replacing the .app bundle
// If any step fails, it automatically triggers rollback from backup
func (du *DarwinUpdater) Apply(updatePath, execPath string) error {
	du.logger.Info().
		Str("update", updatePath).
		Str("target", execPath).
		Msg("Applying macOS update")

	if !strings.HasSuffix(updatePath, ".dmg") {
		du.logger.Warn().
			Str("update", updatePath).
			Msg("Update is not a DMG file, falling back to archive extraction")
		return du.applyFromArchive(updatePath, execPath)
	}

	appBundlePath := du.findAppBundlePath(execPath)
	if appBundlePath == "" {
		du.logger.Warn().
			Str("exec_path", execPath).
			Msg("Could not find .app bundle, falling back to executable replacement")
		return du.applyExecutableOnly(updatePath, execPath)
	}

	du.logger.Info().
		Str("app_bundle", appBundlePath).
		Msg("Found .app bundle, will replace entire bundle")

	backupPath := appBundlePath + ".backup"

	du.logger.Info().Msg("Creating backup of .app bundle")
	if err := du.backupAppBundle(appBundlePath, backupPath); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create backup before update",
			Cause:    err,
			Context: map[string]interface{}{
				"app_bundle":  appBundlePath,
				"backup_path": backupPath,
			},
		}
	}

	defer func() {
		if _, err := os.Stat(backupPath); err == nil {
			os.RemoveAll(backupPath)
		}
	}()

	tempDir := filepath.Join(os.TempDir(), "WXChannelsDownload")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create temporary directory",
			Cause:    err,
			Context: map[string]interface{}{
				"temp_dir": tempDir,
			},
		}
	}

	du.logger.Info().
		Str("dmg", updatePath).
		Str("extract_to", tempDir).
		Msg("Extracting .app from DMG")

	newAppPath, err := du.extractAppFromDmg(updatePath, tempDir)
	if err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	du.logger.Info().
		Str("new_app", newAppPath).
		Msg("Found new .app bundle in DMG")

	fmt.Println("DarwinUpdater.Apply before validateAppBundle", newAppPath)
	if err := du.validateAppBundle(newAppPath); err != nil {
		du.logger.Warn().Err(err).Str("path", newAppPath).Msg("Invalid .app bundle in DMG, falling back to executable replacement")
		du.triggerRollback(backupPath, appBundlePath)
		return du.applyExecutableOnly(updatePath, execPath)
	}

	fmt.Println("DarwinUpdater.Apply before os.removeAll", appBundlePath)
	if err := os.RemoveAll(appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to remove old .app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"app_bundle": appBundlePath,
			},
		}
	}

	fmt.Println("DarwinUpdater.Apply before du.moveAppBundle", newAppPath)
	if err := du.moveAppBundle(newAppPath, appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	if err := du.ensureExecutablePermission(appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	if err := du.VerifyCodeSignature(appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	du.logger.Info().Msg("macOS update applied successfully")
	return nil
}

// applyFromArchive handles non-DMG archives (zip, tar.gz, tar.xz)
func (du *DarwinUpdater) applyFromArchive(updatePath, execPath string) error {
	du.logger.Info().
		Str("update", updatePath).
		Str("target", execPath).
		Msg("Applying macOS update from archive")

	appBundlePath := du.findAppBundlePath(execPath)
	if appBundlePath == "" {
		du.logger.Warn().
			Str("exec_path", execPath).
			Msg("Could not find .app bundle, falling back to executable replacement")
		return du.applyExecutableOnly(updatePath, execPath)
	}

	du.logger.Info().
		Str("app_bundle", appBundlePath).
		Msg("Found .app bundle, will replace entire bundle")

	backupPath := appBundlePath + ".backup"

	du.logger.Info().Msg("Creating backup of .app bundle")
	if err := du.backupAppBundle(appBundlePath, backupPath); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create backup before update",
			Cause:    err,
			Context: map[string]interface{}{
				"app_bundle":  appBundlePath,
				"backup_path": backupPath,
			},
		}
	}

	defer func() {
		if _, err := os.Stat(backupPath); err == nil {
			os.RemoveAll(backupPath)
		}
	}()

	tempDir := filepath.Join(os.TempDir(), "WXChannelsDownload")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create temporary extraction directory",
			Cause:    err,
			Context: map[string]interface{}{
				"temp_dir": tempDir,
			},
		}
	}

	du.logger.Info().
		Str("archive", updatePath).
		Str("extract_to", tempDir).
		Msg("Starting archive extraction")

	fmt.Println("DarwinUpdater.Apply before ExtractArchive", tempDir)
	if err := du.ExtractArchive(updatePath, tempDir); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	du.logger.Info().
		Str("extract_dir", tempDir).
		Msg("Archive extraction completed successfully")

	newAppPath, err := du.findExtractedAppBundle(tempDir)
	fmt.Println("DarwinUpdater.Apply after findExtractedAppBundle", newAppPath)
	if err != nil {
		du.logger.Warn().Err(err).Msg("No .app bundle found in archive, trying executable replacement")
		du.triggerRollback(backupPath, appBundlePath)
		return du.applyExecutableOnly(updatePath, execPath)
	}

	du.logger.Info().
		Str("new_app", newAppPath).
		Msg("Found new .app bundle in archive")

	fmt.Println("DarwinUpdater.Apply before validateAppBundle", newAppPath)
	if err := du.validateAppBundle(newAppPath); err != nil {
		du.logger.Warn().Err(err).Str("path", newAppPath).Msg("Invalid .app bundle in archive, falling back to executable replacement")
		du.triggerRollback(backupPath, appBundlePath)
		return du.applyExecutableOnly(updatePath, execPath)
	}

	fmt.Println("DarwinUpdater.Apply before os.removeAll", appBundlePath)
	if err := os.RemoveAll(appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to remove old .app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"app_bundle": appBundlePath,
			},
		}
	}

	fmt.Println("DarwinUpdater.Apply before du.moveAppBundle", newAppPath)
	if err := du.moveAppBundle(newAppPath, appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	if err := du.ensureExecutablePermission(appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	if err := du.VerifyCodeSignature(appBundlePath); err != nil {
		du.triggerRollback(backupPath, appBundlePath)
		return err
	}

	du.logger.Info().Msg("macOS update applied successfully")
	return nil
}

// findAppBundlePath finds the .app bundle path from an executable path
func (du *DarwinUpdater) findAppBundlePath(execPath string) string {
	// Walk up the path looking for .app directory
	path := execPath
	for {
		if strings.HasSuffix(path, ".app") {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			// Reached root
			return ""
		}
		path = parent
	}
}

// findExtractedAppBundle finds a .app bundle in the extracted directory
func (du *DarwinUpdater) findExtractedAppBundle(dir string) (string, error) {
	var appPath string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for .app directories
		if info.IsDir() && strings.HasSuffix(info.Name(), ".app") {
			appPath = path
			return filepath.SkipAll // Stop after finding first .app
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to search for .app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"dir": dir,
			},
		}
	}

	if appPath == "" {
		return "", &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "no .app bundle found in update archive",
			Context: map[string]interface{}{
				"dir": dir,
			},
		}
	}

	return appPath, nil
}

// backupAppBundle creates a backup of the entire .app bundle
func (du *DarwinUpdater) backupAppBundle(appPath, backupPath string) error {
	// Remove existing backup if present
	os.RemoveAll(backupPath)

	// Use cp -R to preserve all attributes and symlinks
	cmd := exec.Command("cp", "-R", appPath, backupPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to backup .app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"output": string(output),
			},
		}
	}

	return nil
}

// moveAppBundle moves a .app bundle to a new location
func (du *DarwinUpdater) moveAppBundle(src, dst string) error {
	// Try rename first (fastest if on same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fall back to cp -R and remove
	cmd := exec.Command("cp", "-R", src, dst)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to copy new .app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"output": string(output),
			},
		}
	}

	// Remove source
	os.RemoveAll(src)

	return nil
}

// triggerRollback attempts to restore from backup when update fails
func (du *DarwinUpdater) triggerRollback(backupPath, targetPath string) {
	du.logger.Warn().
		Str("backup", backupPath).
		Str("target", targetPath).
		Msg("Update failed, triggering rollback")

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		du.logger.Error().Msg("Backup not found, cannot rollback")
		return
	}

	// Remove failed update
	os.RemoveAll(targetPath)

	// Restore from backup
	if err := os.Rename(backupPath, targetPath); err != nil {
		// Try cp -R as fallback
		cmd := exec.Command("cp", "-R", backupPath, targetPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			du.logger.Error().
				Err(err).
				Str("output", string(output)).
				Msg("Rollback failed - system may be in inconsistent state")
			return
		}
		os.RemoveAll(backupPath)
	}

	du.logger.Info().Msg("Rollback completed successfully")
}

// applyExecutableOnly falls back to replacing just the executable (for non-.app bundles)
func (du *DarwinUpdater) applyExecutableOnly(updatePath, execPath string) error {
	du.logger.Info().Msg("Applying executable-only update")

	// Create backup path
	backupPath := execPath + ".backup"

	// Create backup
	if err := du.Backup(execPath, backupPath); err != nil {
		return err
	}

	defer func() {
		if _, err := os.Stat(backupPath); err == nil {
			du.Cleanup(backupPath)
		}
	}()

	// Create temporary directory for extraction
	tempDir := filepath.Join(os.TempDir(), "WXChannelsDownload")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		du.restoreExecutable(backupPath, execPath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create temporary extraction directory",
			Cause:    err,
		}
	}
	// defer os.RemoveAll(tempDir) // Commented out to keep files for inspection

	// Extract the update archive (executable-only mode)
	du.logger.Info().
		Str("archive", updatePath).
		Str("extract_to", tempDir).
		Msg("Starting executable-only archive extraction")

	if err := du.ExtractArchive(updatePath, tempDir); err != nil {
		du.restoreExecutable(backupPath, execPath)
		return err
	}

	du.logger.Info().
		Str("extract_dir", tempDir).
		Msg("Executable-only archive extraction completed successfully")

	// Find the executable in the extracted files
	newExecPath, err := du.findExecutable(tempDir)
	if err != nil {
		du.restoreExecutable(backupPath, execPath)
		return err
	}

	// Get the original file permissions
	origInfo, err := os.Stat(execPath)
	if err != nil && !os.IsNotExist(err) {
		du.restoreExecutable(backupPath, execPath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to get original file info",
			Cause:    err,
		}
	}

	// Remove the old executable
	if err := os.Remove(execPath); err != nil && !os.IsNotExist(err) {
		du.restoreExecutable(backupPath, execPath)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to remove old executable",
			Cause:    err,
		}
	}

	// Copy the new executable
	if err := du.copyFile(newExecPath, execPath); err != nil {
		du.restoreExecutable(backupPath, execPath)
		return err
	}

	// Set permissions
	mode := os.FileMode(0755)
	if origInfo != nil {
		mode = origInfo.Mode()
	}
	if err := os.Chmod(execPath, mode); err != nil {
		du.restoreExecutable(backupPath, execPath)
		return &types.UpdateError{
			Category: types.ErrCategoryPermission,
			Message:  "failed to set executable permissions",
			Cause:    err,
		}
	}

	// Verify code signature
	if err := du.VerifyCodeSignature(execPath); err != nil {
		du.restoreExecutable(backupPath, execPath)
		return err
	}

	return nil
}

// restoreExecutable restores the executable from backup
func (du *DarwinUpdater) restoreExecutable(backupPath, execPath string) {
	if err := du.Restore(backupPath, execPath); err != nil {
		du.logger.Error().Err(err).Msg("Failed to restore executable from backup")
	}
}

// findExecutable finds the executable file in the extracted directory
func (du *DarwinUpdater) findExecutable(dir string) (string, error) {
	var execPath string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .app bundles (we want the raw executable)
		if info.IsDir() && strings.HasSuffix(info.Name(), ".app") {
			return filepath.SkipDir
		}

		// Look for executable files
		if !info.IsDir() && info.Mode()&0111 != 0 {
			execPath = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to search for executable",
			Cause:    err,
		}
	}

	if execPath == "" {
		return "", &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "no executable found in update archive",
		}
	}

	return execPath, nil
}

// copyFile copies a file from src to dst
func (du *DarwinUpdater) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to open source file",
			Cause:    err,
		}
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create destination file",
			Cause:    err,
		}
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to copy file",
			Cause:    err,
		}
	}

	return nil
}

// validateAppBundle verifies that a path is a valid .app bundle with an executable
func (du *DarwinUpdater) validateAppBundle(appPath string) error {
	if !strings.HasSuffix(appPath, ".app") {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "not a valid .app bundle",
			Context: map[string]interface{}{
				"path": appPath,
			},
		}
	}
	info, err := os.Stat(appPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to stat .app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"path": appPath,
			},
		}
	}
	if !info.IsDir() {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  ".app should be a directory",
			Context: map[string]interface{}{
				"path": appPath,
			},
		}
	}
	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	entries, err := os.ReadDir(macosDir)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "invalid .app structure",
			Cause:    err,
			Context: map[string]interface{}{
				"macos_dir": macosDir,
			},
		}
	}
	if len(entries) == 0 {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "no executable found in .app",
			Context: map[string]interface{}{
				"macos_dir": macosDir,
			},
		}
	}
	return nil
}

// ensureExecutablePermission ensures the primary executable inside .app has execute permission
func (du *DarwinUpdater) ensureExecutablePermission(appPath string) error {
	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	entries, err := os.ReadDir(macosDir)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to read MacOS directory",
			Cause:    err,
			Context: map[string]interface{}{
				"macos_dir": macosDir,
			},
		}
	}
	if len(entries) == 0 {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "no executable found in MacOS directory",
			Context: map[string]interface{}{
				"macos_dir": macosDir,
			},
		}
	}
	execPath := filepath.Join(macosDir, entries[0].Name())
	info, err := os.Stat(execPath)
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to stat executable",
			Cause:    err,
		}
	}
	if info.Mode()&0111 == 0 {
		if err := os.Chmod(execPath, 0755); err != nil {
			return &types.UpdateError{
				Category: types.ErrCategoryPermission,
				Message:  "failed to set execute permission",
				Cause:    err,
				Context: map[string]interface{}{
					"exec": execPath,
				},
			}
		}
	}
	return nil
}

// VerifyCodeSignature verifies the code signature of a macOS executable or .app bundle
func (du *DarwinUpdater) VerifyCodeSignature(path string) error {
	du.logger.Info().
		Str("path", path).
		Msg("Verifying macOS code signature")

	// Run codesign --verify on the path
	cmd := exec.Command("codesign", "--verify", "--verbose", path)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Code signature verification failed
		du.logger.Error().
			Err(err).
			Str("path", path).
			Str("output", string(output)).
			Msg("Code signature verification failed")

		return &types.UpdateError{
			Category: types.ErrCategorySecurity,
			Message:  "code signature verification failed",
			Cause:    err,
			Context: map[string]interface{}{
				"path":   path,
				"output": strings.TrimSpace(string(output)),
			},
		}
	}

	du.logger.Info().
		Str("path", path).
		Msg("Code signature verified successfully")

	return nil
}

// Restart restarts the application
func (du *DarwinUpdater) Restart(execPath string, args []string) error {
	du.logger.Info().
		Str("exec", execPath).
		Strs("args", args).
		Msg("Restarting application")
	du.logs = append(du.logs, fmt.Sprintf("Restarting application: %s %v", execPath, args))
	// Try launching via LaunchServices when inside an .app bundle
	appBundlePath := du.findAppBundlePath(execPath)
	du.logs = append(du.logs, fmt.Sprintf("find App bundle path: %s", appBundlePath))
	if appBundlePath != "" {
		du.logger.Info().
			Str("app_bundle", appBundlePath).
			Msg("Launching via macOS LaunchServices (open)")

		// If Gatekeeper blocks, attempt ad-hoc re-sign to allow dev builds to relaunch
		// if !du.gatekeeperOK(appBundlePath) {
		// 	du.logger.Warn().Str("app_bundle", appBundlePath).Msg("Gatekeeper check failed, attempting ad-hoc re-sign")
		// 	if err := du.adHocResign(appBundlePath); err != nil {
		// 		du.logger.Warn().Err(err).Msg("Ad-hoc re-sign failed")
		// 	}
		// }
		fmt.Println("[]applier  before du.tryOpenLaunch", appBundlePath)
		du.logger.Info().
			Str("appBundlePath", appBundlePath).
			Msg("try open app bundle path")
		if err := du.tryOpenLaunch(appBundlePath, args); err == nil {
			os.Exit(0)
			return nil
		}
		du.logger.Warn().Msg("LaunchServices open strategies failed, falling back to direct exec")
	}

	fmt.Println("[]applier  before exec.Command(execPath)", execPath)
	du.logger.Info().
		Str("appBundlePath", execPath).
		Msg("open exec with exec.Command")

	// Fallback: directly execute the binary (non-.app or open failed)
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
		}
	}

	du.logger.Info().
		Int("pid", cmd.Process.Pid).
		Msg("New process started via exec")

	// Exit current process
	os.Exit(0)

	return nil
}

// gatekeeperOK checks if spctl allows execution
func (du *DarwinUpdater) gatekeeperOK(path string) bool {
	cmd := exec.Command("spctl", "-a", "-t", "exec", "-vv", path)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// adHocResign performs an ad-hoc code signing for development builds
func (du *DarwinUpdater) adHocResign(appBundlePath string) error {
	cmd := exec.Command("codesign", "--force", "--deep", "--sign", "-", "--options", "runtime", appBundlePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategorySecurity,
			Message:  "failed to ad-hoc sign app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"output": strings.TrimSpace(string(output)),
			},
		}
	}
	return nil
}

// tryOpenLaunch attempts multiple open strategies: by path, bundle id, and app name
func (du *DarwinUpdater) tryOpenLaunch(appBundlePath string, args []string) error {
	// Strategy 1: open by bundle path
	if err := du.startOpen([]string{"-n", appBundlePath}, args); err == nil {
		return nil
	}
	// Strategy 2: open by bundle identifier
	if bid := du.readInfoPlistValue(appBundlePath, "CFBundleIdentifier"); bid != "" {
		if err := du.startOpen([]string{"-n", "-b", bid}, args); err == nil {
			return nil
		}
	}
	// Strategy 3: open by app name
	appName := du.readInfoPlistValue(appBundlePath, "CFBundleName")
	if appName == "" {
		appName = strings.TrimSuffix(filepath.Base(appBundlePath), ".app")
	}
	if appName != "" {
		if err := du.startOpen([]string{"-n", "-a", appName}, args); err == nil {
			return nil
		}
	}
	return &types.UpdateError{
		Category: types.ErrCategoryFileSystem,
		Message:  "failed to start via LaunchServices",
	}
}

// startOpen builds and starts an `open` command with optional args
func (du *DarwinUpdater) startOpen(baseArgs []string, extraArgs []string) error {
	final := append([]string{}, baseArgs...)
	if len(extraArgs) > 0 {
		final = append(final, "--args")
		final = append(final, extraArgs...)
	}
	cmd := exec.Command("open", final...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

// readInfoPlistValue reads an Info.plist key using `defaults read`
func (du *DarwinUpdater) readInfoPlistValue(appBundlePath, key string) string {
	plist := filepath.Join(appBundlePath, "Contents", "Info.plist")
	cmd := exec.Command("defaults", "read", plist, key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// newPlatformUpdaterImpl creates a macOS-specific updater
func newPlatformUpdaterImpl(logger *zerolog.Logger) master.UpdateApplier {
	return NewDarwinUpdater(logger)
}

// extractAppFromDmg extracts the .app bundle from a DMG file
func (du *DarwinUpdater) extractAppFromDmg(dmgPath, destDir string) (string, error) {
	du.logger.Info().
		Str("dmg", dmgPath).
		Str("dest", destDir).
		Msg("Extracting .app from DMG")

	mountInfo, err := du.attachDmg(dmgPath)
	if err != nil {
		return "", err
	}
	defer du.detachDmg(mountInfo.device)

	appPath := filepath.Join(mountInfo.mountPoint, du.findAppBundleName(mountInfo.mountPoint))
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return "", &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  ".app bundle not found in DMG",
			Context: map[string]interface{}{
				"dmg_path":     dmgPath,
				"mount_point":  mountInfo.mountPoint,
				"searched_for": appPath,
			},
		}
	}

	newAppPath := filepath.Join(destDir, filepath.Base(appPath))
	if err := du.copyAppBundle(appPath, newAppPath); err != nil {
		return "", err
	}

	du.logger.Info().
		Str("app_path", newAppPath).
		Msg("Successfully extracted .app from DMG")

	return newAppPath, nil
}

type dmgMountInfo struct {
	device     string
	mountPoint string
}

func (du *DarwinUpdater) attachDmg(dmgPath string) (*dmgMountInfo, error) {
	cmd := exec.Command("hdiutil", "attach", "-readonly", "-plist", dmgPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to attach DMG",
			Cause:    err,
			Context: map[string]interface{}{
				"dmg":    dmgPath,
				"output": string(output),
			},
		}
	}

	mountPoint, device, err := du.parseDmgAttachOutput(string(output))
	if err != nil {
		return nil, err
	}

	return &dmgMountInfo{
		device:     device,
		mountPoint: mountPoint,
	}, nil
}

func (du *DarwinUpdater) detachDmg(device string) error {
	cmd := exec.Command("hdiutil", "detach", device)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to detach DMG",
			Cause:    err,
			Context: map[string]interface{}{
				"device": device,
				"output": string(output),
			},
		}
	}
	return nil
}

func (du *DarwinUpdater) parseDmgAttachOutput(output string) (string, string, error) {
	mountPoint := extractPlistStringValue(output, "mount-point")
	device := extractPlistStringValue(output, "dev-entry")

	if mountPoint == "" || device == "" {
		return "", "", &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "failed to parse mount-point or device from hdiutil output",
			Context: map[string]interface{}{
				"mount_point": mountPoint,
				"device":      device,
				"output":      output,
			},
		}
	}

	return mountPoint, device, nil
}

func extractPlistStringValue(output string, key string) string {
	pattern := `<key>` + key + `</key>\s*<string>([^<]+)</string>`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func (du *DarwinUpdater) findAppBundleName(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".app") {
			return entry.Name()
		}
	}
	return ""
}

func (du *DarwinUpdater) copyAppBundle(src, dst string) error {
	cmd := exec.Command("cp", "-R", src, dst)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to copy .app bundle",
			Cause:    err,
			Context: map[string]interface{}{
				"source": src,
				"dest":   dst,
				"output": string(output),
			},
		}
	}
	return nil
}
