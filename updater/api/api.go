package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ltaoo/velo/updater/applier"
	"github.com/ltaoo/velo/updater/cache"
	"github.com/ltaoo/velo/updater/checker"
	"github.com/ltaoo/velo/updater/downloader"
	"github.com/ltaoo/velo/updater/master"
	"github.com/ltaoo/velo/updater/types"
	"github.com/ltaoo/velo/updater/util"

	"github.com/rs/zerolog"
)

type AppUpdater struct {
	cfg                          *types.UpdateConfig
	state                        *types.UpdateState
	statePath                    string
	currentVersion               string
	latestRelease                *types.ReleaseInfo
	downloadedNewVersionFilepath string
	checker                      *checker.UpdateChecker
	downloader                   *downloader.UpdateDownloadManager
	applier                      master.UpdateApplier
	cache                        *cache.CacheManager
	logger                       *zerolog.Logger
}

// UpdateCallback is called to notify about update events

// NewUpdater creates a new update orchestrator with the given configuration
// This is the main entry point for creating an updater instance
func NewUpdater(config *types.UpdateConfig, logger *zerolog.Logger) (*AppUpdater, error) {
	return NewUpdaterWithOptions(&types.UpdaterOptions{
		Config: config,
	}, logger)
}

// NewUpdaterWithOptions creates a new update orchestrator with custom options
// This provides more control over the updater configuration
func NewUpdaterWithOptions(opts *types.UpdaterOptions, logger *zerolog.Logger) (*AppUpdater, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	if opts.Config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	currentVersion := opts.CurrentVersion
	if currentVersion == "" {
		currentVersion = "0.1.0"
	}
	statePath := opts.StatePath
	if statePath == "" {
		statePath = fmt.Sprintf("%s/.app_updater/update_state.json", homeDir)
	}
	// Load existing state
	state, err := types.LoadUpdateState(statePath)
	if err != nil {
		// Log warning but continue with empty state
		logger.Warn().Err(err).Msg("Failed to load update state, starting with empty state")
		state = &types.UpdateState{
			SkippedVersions: []string{},
			CurrentVersion:  currentVersion,
		}
	}
	cacheDir := filepath.Dir(statePath)
	cachePath := filepath.Join(cacheDir, "update_cache.json")
	cacheManager := cache.NewCacheManager(cachePath, 1*time.Hour)
	updateChecker, err := checker.NewUpdateChecker(opts.Config, currentVersion, cacheManager, state, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}
	if state.CurrentVersion == "" || state.CurrentVersion != currentVersion {
		state.CurrentVersion = currentVersion
	}
	updateApplier := applier.NewPlatformUpdater(logger)
	updateDownloader := downloader.NewUpdateDownloadManager(logger)
	return &AppUpdater{
		checker:        updateChecker,
		downloader:     updateDownloader,
		applier:        updateApplier,
		cache:          cacheManager,
		logger:         logger,
		cfg:            opts.Config,
		state:          state,
		statePath:      statePath,
		currentVersion: currentVersion,
	}, nil
}

// PerformUpdate performs the complete update process: check, download, and apply
func (u *AppUpdater) PerformUpdate(ctx context.Context, onProgress types.DownloadCallback) error {
	u.logger.Info().
		Str("current_version", u.currentVersion).
		Msg("Starting complete update process")

	// Phase 1: Check for updates
	u.logger.Info().Msg("Phase 1: Checking for updates")
	releaseInfo, err := u.CheckForUpdates(ctx)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to check for updates")
		return &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to check for updates",
			Cause:    err,
		}
	}
	// Check if update is needed
	if !releaseInfo.IsNewer {
		u.logger.Info().
			Str("current_version", u.currentVersion).
			Str("latest_version", releaseInfo.Version).
			Msg("Already running the latest version")
		return nil
	}
	u.logger.Info().
		Str("current_version", u.currentVersion).
		Str("new_version", releaseInfo.Version).
		Msg("New version available")
	// Store release info for later use
	u.latestRelease = releaseInfo
	// Phase 2: Download update
	u.logger.Info().Msg("Phase 2: Downloading update")
	updatePath, err := u.DownloadUpdate(ctx, releaseInfo, onProgress)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to download update")
		return &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to download update",
			Cause:    err,
			Context: map[string]interface{}{
				"version": releaseInfo.Version,
			},
		}
	}
	// Phase 3: Apply update
	u.logger.Info().Msg("Phase 3: Applying update")
	if err := u.ApplyUpdate(ctx, updatePath); err != nil {
		u.logger.Error().Err(err).Msg("Failed to apply update")
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to apply update",
			Cause:    err,
			Context: map[string]interface{}{
				"version":     releaseInfo.Version,
				"update_path": updatePath,
			},
		}
	}
	u.logger.Info().
		Str("old_version", u.currentVersion).
		Str("new_version", releaseInfo.Version).
		Msg("Update process completed successfully")
	return nil
}

func (u *AppUpdater) CheckForUpdates(ctx context.Context) (*types.ReleaseInfo, error) {
	return u.checker.CheckForUpdates(ctx)
}
func (u *AppUpdater) CheckForUpdatesForce(ctx context.Context) (*types.ReleaseInfo, error) {
	return u.checker.CheckForUpdatesForce(ctx)
}
func (u *AppUpdater) CheckForUpdatesWithCallback(ctx context.Context, callback types.UpdateCallback) (*types.ReleaseInfo, error) {
	return u.checker.CheckForUpdatesWithCallback(ctx, callback)
}
func (u *AppUpdater) DownloadUpdate(ctx context.Context, release *types.ReleaseInfo, onProgress func(progress types.DownloadProgress)) (string, error) {
	newAppFilepath, err := u.downloader.DownloadUpdate(ctx, release, onProgress)
	if err != nil {
		return "", err
	}
	u.downloadedNewVersionFilepath = newAppFilepath
	return newAppFilepath, nil
}

// ApplyUpdate applies the downloaded update package
func (u *AppUpdater) ApplyUpdate(ctx context.Context, updatePath string) error {
	if updatePath == "" {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "update path cannot be empty",
		}
	}
	u.logger.Info().
		Str("update_path", updatePath).
		Msg("Starting update application")
	// Get current executable path
	execPath, err := util.GetExecutablePath()
	if err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to get executable path",
			Cause:    err,
		}
	}
	// Create backup path
	backupPath := execPath + ".backup"
	// Step 1: Create backup
	u.logger.Info().Msg("Creating backup of current executable")
	fmt.Println("ApplyUpdate - Creating backup of current executable", execPath, backupPath)
	if err := u.applier.Backup(execPath, backupPath); err != nil {
		// logpkg.LogUpdateError(uo.logger, err, "backup")
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "failed to create backup",
			Cause:    err,
			Context: map[string]interface{}{
				"exec_path":   execPath,
				"backup_path": backupPath,
			},
		}
	}
	// logpkg.LogBackupOperation(uo.logger, "create", execPath, backupPath, true)
	// Step 2: Apply update (extract and replace)
	u.logger.Info().Msg("Applying update")
	fmt.Println("ApplyUpdate - Applying update")
	if err := u.applier.Apply(updatePath, execPath); err != nil {
		// logpkg.LogUpdateError(uo.logger, err, "apply")
		// Rollback on failure
		u.logger.Warn().Msg("Update failed, attempting rollback")
		// logpkg.LogRollbackStart(uo.logger, err.Error())
		if rollbackErr := u.applier.Restore(backupPath, execPath); rollbackErr != nil {
			// logpkg.LogRollbackError(uo.logger, rollbackErr)
			return &types.UpdateError{
				Category: types.ErrCategoryFileSystem,
				Message:  "update failed and rollback also failed",
				Cause:    err,
				Context: map[string]interface{}{
					"apply_error":    err.Error(),
					"rollback_error": rollbackErr.Error(),
				},
			}
		}
		// logpkg.LogRollbackComplete(uo.logger)
		return &types.UpdateError{
			Category: types.ErrCategoryFileSystem,
			Message:  "update failed, rolled back to previous version",
			Cause:    err,
		}
	}
	// Step 3: Verify the updated executable
	u.logger.Info().Msg("Verifying updated executable")
	if err := util.ValidateExecutable(execPath); err != nil {
		// logpkg.LogSecurityError(uo.logger, err, "executable_validation")
		// Rollback on validation failure
		u.logger.Warn().Msg("Validation failed, attempting rollback")
		// logpkg.LogRollbackStart(uo.logger, "validation failed")
		if rollbackErr := u.applier.Restore(backupPath, execPath); rollbackErr != nil {
			// logpkg.LogRollbackError(uo.logger, rollbackErr)
			return &types.UpdateError{
				Category: types.ErrCategorySecurity,
				Message:  "validation failed and rollback also failed",
				Cause:    err,
				Context: map[string]interface{}{
					"validation_error": err.Error(),
					"rollback_error":   rollbackErr.Error(),
				},
			}
		}
		// logpkg.LogRollbackComplete(uo.logger)
		return &types.UpdateError{
			Category: types.ErrCategorySecurity,
			Message:  "executable validation failed, rolled back to previous version",
			Cause:    err,
		}
	}
	// Step 4: Cleanup
	u.logger.Info().Msg("Cleaning up temporary files")
	cleanupPaths := []string{updatePath, backupPath}
	if err := u.applier.Cleanup(cleanupPaths...); err != nil {
		// Log cleanup errors but don't fail the update
		u.logger.Warn().Err(err).Msg("Cleanup had errors, but update was successful")
	}
	u.logger.Info().Msg("Update applied successfully")
	// logpkg.LogUpdateComplete(uo.logger, "", 0)
	return nil
}

func (u *AppUpdater) ApplyUpdateThenRestartApplication(ctx context.Context) error {
	if u.downloadedNewVersionFilepath == "" {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "no update file path available",
		}
	}
	if err := u.ApplyUpdate(ctx, u.downloadedNewVersionFilepath); err != nil {
		return err
	}
	if err := u.RestartApplication([]string{"--update"}); err != nil {
		return err
	}
	return nil
}

// CheckForUpdatesWithCallback checks for updates and calls the callback with events
// This is a convenience method that wraps CheckForUpdates with event notifications

// // DownloadUpdateWithCallback downloads an update and calls the callback with events
// // This is a convenience method that wraps DownloadUpdate with event notifications
// func (uo *UpdateOrchestrator) DownloadUpdateWithCallback(
// 	ctx context.Context,
// 	release *types.ReleaseInfo,
// 	callback UpdateCallback,
// ) (string, error) {
// 	if callback != nil {
// 		callback(UpdateEvent{
// 			Type:        EventDownloadStarted,
// 			Message:     fmt.Sprintf("Starting download of version %s", release.Version),
// 			ReleaseInfo: release,
// 		})
// 	}

// 	// Create progress callback that wraps the update callback
// 	progressCallback := func(progress types.DownloadProgress) {
// 		if callback != nil {
// 			callback(UpdateEvent{
// 				Type:     EventDownloadProgress,
// 				Message:  fmt.Sprintf("Downloading... %.1f%%", progress.Percentage),
// 				Progress: &progress,
// 			})
// 		}
// 	}

// 	updatePath, err := uo.DownloadUpdate(ctx, release, progressCallback)

// 	if err != nil {
// 		if callback != nil {
// 			callback(UpdateEvent{
// 				Type:    EventError,
// 				Message: "Failed to download update",
// 				Error:   err,
// 			})
// 		}
// 		return "", err
// 	}

// 	if callback != nil {
// 		callback(UpdateEvent{
// 			Type:    EventDownloadCompleted,
// 			Message: "Download completed successfully",
// 		})
// 	}

// 	return updatePath, nil
// }

// // ApplyUpdateWithCallback applies an update and calls the callback with events
// // This is a convenience method that wraps ApplyUpdate with event notifications
// func (uo *UpdateOrchestrator) ApplyUpdateWithCallback(
// 	ctx context.Context,
// 	updatePath string,
// 	callback UpdateCallback,
// ) error {
// 	if callback != nil {
// 		callback(UpdateEvent{
// 			Type:    EventApplyStarted,
// 			Message: "Applying update...",
// 		})
// 	}

// 	err := uo.ApplyUpdate(ctx, updatePath)

// 	if err != nil {
// 		if callback != nil {
// 			callback(UpdateEvent{
// 				Type:    EventError,
// 				Message: "Failed to apply update",
// 				Error:   err,
// 			})
// 		}
// 		return err
// 	}

// 	if callback != nil {
// 		callback(UpdateEvent{
// 			Type:    EventApplyCompleted,
// 			Message: "Update applied successfully",
// 		})
// 	}

// 	return nil
// }

// // PerformUpdateWithCallback performs the complete update process with event notifications
// // This is a convenience method that combines check, download, and apply with callbacks
// func (uo *UpdateOrchestrator) PerformUpdateWithCallback(
// 	ctx context.Context,
// 	callback UpdateCallback,
// ) error {
// 	// Check for updates
// 	releaseInfo, err := uo.CheckForUpdatesWithCallback(ctx, callback)
// 	if err != nil {
// 		return err
// 	}

// 	// If no update available, return
// 	if !releaseInfo.IsNewer {
// 		return nil
// 	}

// 	// Download update
// 	updatePath, err := uo.DownloadUpdateWithCallback(ctx, releaseInfo, callback)
// 	if err != nil {
// 		return err
// 	}

// 	// Apply update
// 	return uo.ApplyUpdateWithCallback(ctx, updatePath, callback)
// }

// GetLatestRelease returns the latest release information from the last check
// Returns nil if no check has been performed yet
func (uo *AppUpdater) GetLatestRelease() *types.ReleaseInfo {
	return uo.latestRelease
}

// GetUpdateState returns the current update state
func (uo *AppUpdater) GetUpdateState() *types.UpdateState {
	return uo.state
}

// GetCurrentVersion returns the current application version
func (uo *AppUpdater) GetCurrentVersion() string {
	return uo.currentVersion
}

// // SkipVersion marks a version as skipped so it won't be offered again
func (uo *AppUpdater) SkipVersion(version string) error {
	// Add to skipped versions if not already present
	for _, v := range uo.state.SkippedVersions {
		if v == version {
			return nil // Already skipped
		}
	}

	uo.state.SkippedVersions = append(uo.state.SkippedVersions, version)

	// Save state
	if err := uo.state.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	uo.logger.Info().
		Str("version", version).
		Msg("Version marked as skipped")

	return nil
}

// IsVersionSkipped checks if a version has been marked as skipped
func (uo *AppUpdater) IsVersionSkipped(version string) bool {
	for _, v := range uo.state.SkippedVersions {
		if v == version {
			return true
		}
	}
	return false
}

// ClearSkippedVersions clears all skipped versions
func (uo *AppUpdater) ClearSkippedVersions() error {
	uo.state.SkippedVersions = []string{}

	// Save state
	if err := uo.state.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	uo.logger.Info().Msg("Cleared all skipped versions")

	return nil
}

// RestartApplication restarts the application after an update
// This should be called after a successful update
func (uo *AppUpdater) RestartApplication(args []string) error {
	execPath, err := util.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	uo.logger.Info().
		Str("exec_path", execPath).
		Strs("args", args).
		Msg("Restarting application")

	return uo.applier.Restart(execPath, args)
}

// DefaultUpdaterConfig returns the default updater configuration
func DefaultUpdaterConfig() *types.UpdateConfig {
	return &types.UpdateConfig{
		Enabled:        true,
		CheckFrequency: "startup",
		Channel:        "stable",
		AutoDownload:   false,
		Timeout:        300,
		DevModeEnabled: false,
		DevVersion:     "0.1.0",
		Sources:        []types.UpdateSource{},
	}
}

// DevelopmentConfig returns a configuration suitable for development/testing
func DevelopmentConfig() *types.UpdateConfig {
	return &types.UpdateConfig{
		Enabled:        true,
		CheckFrequency: "startup",
		Channel:        "stable",
		AutoDownload:   false,
		Timeout:        60,
		DevModeEnabled: true,
		DevVersion:     "0.1.0",
		DevUpdateSource: &types.UpdateSource{
			Type:        "http",
			Priority:    1,
			Enabled:     true,
			ManifestURL: "http://localhost:8080/manifest.json",
		},
		Sources: []types.UpdateSource{
			{
				Type:              "github",
				Priority:          1,
				Enabled:           true,
				NeedCheckChecksum: true,
				GitHubRepo:        "ltaoo/velo",
				GitHubToken:       "",
			},
		},
	}
}
