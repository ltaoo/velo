package checker

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog"

	"github.com/ltaoo/velo/updater/cache"
	"github.com/ltaoo/velo/updater/master"
	"github.com/ltaoo/velo/updater/types"
	"github.com/ltaoo/velo/updater/util"
)

type UpdateChecker struct {
	config       *types.UpdateConfig
	checkers     []master.VersionChecker
	cacheManager *cache.CacheManager
	state        *types.UpdateState
	// statePath      string
	currentVersion string
	logger         *zerolog.Logger
}

func NewUpdateChecker(config *types.UpdateConfig, currentVersion string, cacheManager *cache.CacheManager, state *types.UpdateState, logger *zerolog.Logger) (*UpdateChecker, error) {
	sources := config.Sources

	// In development mode with dev source configured, use dev source
	if config.DevModeEnabled && config.DevUpdateSource != nil {
		logger.Info().Msg("Using development update source")
		sources = []types.UpdateSource{*config.DevUpdateSource}
	}

	if len(sources) == 0 {
		logger.Warn().Msg("No update sources configured")
		return nil, fmt.Errorf("no update sources configured")
	}

	// Sort sources by priority (lower number = higher priority)
	sortedSources := make([]types.UpdateSource, len(sources))
	copy(sortedSources, sources)
	sort.Slice(sortedSources, func(i, j int) bool {
		return sortedSources[i].Priority < sortedSources[j].Priority
	})
	checkers := make([]master.VersionChecker, 0, len(sortedSources))
	// Create checkers for enabled sources
	for _, source := range sortedSources {
		if !source.Enabled {
			logger.Debug().
				Str("type", source.Type).
				Int("priority", source.Priority).
				Msg("Skipping disabled source")
			continue
		}

		var checker master.VersionChecker

		switch source.Type {
		case "github":
			if source.GitHubRepo == "" {
				logger.Warn().Msg("GitHub source missing repo configuration, skipping")
				continue
			}
			checker = NewGitHubVersionChecker(source.GitHubRepo, source.GitHubToken, logger)
			// Save GitHub token for download authentication
			if source.GitHubToken != "" {
				// uo.githubToken = source.GitHubToken
			}

		case "http":
			manifestURL := source.ManifestURL
			if manifestURL == "" {
				logger.Warn().Msg("HTTP source missing manifest URL, skipping")
				continue
			}
			checker = NewHTTPVersionChecker(manifestURL, logger)

		default:
			logger.Warn().
				Str("type", source.Type).
				Msg("Unknown source type, skipping")
			continue
		}

		checkers = append(checkers, checker)
		logger.Info().
			Str("source", checker.GetSourceName()).
			Int("priority", source.Priority).
			Msg("Initialized version checker")
	}

	if len(checkers) == 0 {
		return nil, fmt.Errorf("no valid update sources configured")
	}
	return &UpdateChecker{
		config:         config,
		checkers:       checkers,
		cacheManager:   cacheManager,
		logger:         logger,
		state:          state,
		currentVersion: currentVersion,
	}, nil
}

// ShouldCheckForUpdates determines if an update check should be performed
// based on the configuration and last check time
func (uo *UpdateChecker) ShouldCheckForUpdates() bool {
	// If updates are disabled, never check
	if !uo.config.Enabled {
		uo.logger.Debug().Msg("Updates disabled in configuration")
		return false
	}

	// If check frequency is manual, never auto-check
	if uo.config.CheckFrequency == "manual" {
		uo.logger.Debug().Msg("Check frequency set to manual")
		return false
	}

	// If check frequency is startup, always check
	if uo.config.CheckFrequency == "startup" {
		uo.logger.Debug().Msg("Check frequency set to startup, will check")
		return true
	}

	// For daily and weekly, check the last check time
	now := time.Now()
	lastCheck := uo.state.LastCheckTime

	switch uo.config.CheckFrequency {
	case "daily":
		// If never checked before, check now
		if lastCheck.IsZero() {
			uo.logger.Debug().Msg("No previous check recorded, will check")
			return true
		}
		// Check if more than 24 hours have passed
		if now.Sub(lastCheck) >= 24*time.Hour {
			uo.logger.Debug().
				Time("last_check", lastCheck).
				Dur("elapsed", now.Sub(lastCheck)).
				Msg("Daily check interval elapsed, will check")
			return true
		}
		uo.logger.Debug().
			Time("last_check", lastCheck).
			Dur("elapsed", now.Sub(lastCheck)).
			Msg("Daily check interval not yet elapsed")
		return false

	case "weekly":
		// If never checked before, check now
		if lastCheck.IsZero() {
			uo.logger.Debug().Msg("No previous check recorded, will check")
			return true
		}
		// Check if more than 7 days have passed
		if now.Sub(lastCheck) >= 7*24*time.Hour {
			uo.logger.Debug().
				Time("last_check", lastCheck).
				Dur("elapsed", now.Sub(lastCheck)).
				Msg("Weekly check interval elapsed, will check")
			return true
		}
		uo.logger.Debug().
			Time("last_check", lastCheck).
			Dur("elapsed", now.Sub(lastCheck)).
			Msg("Weekly check interval not yet elapsed")
		return false

	default:
		uo.logger.Warn().
			Str("check_frequency", uo.config.CheckFrequency).
			Msg("Unknown check frequency, defaulting to not checking")
		return false
	}
}

// CheckForUpdatesInBackground performs an update check in the background
// This is typically called at application startup
// It returns immediately and performs the check asynchronously
func (uo *UpdateChecker) CheckForUpdatesInBackground(ctx context.Context, callback func(*types.ReleaseInfo, error)) {
	// Check if we should perform an update check
	if !uo.ShouldCheckForUpdates() {
		uo.logger.Info().Msg("Skipping background update check based on configuration")
		if callback != nil {
			callback(nil, nil)
		}
		return
	}

	uo.logger.Info().
		Str("check_frequency", uo.config.CheckFrequency).
		Msg("Starting background update check")

	// Perform check in a goroutine
	go func() {
		// Check for updates
		releaseInfo, err := uo.CheckForUpdates(ctx)
		// Update last check time regardless of success/failure
		uo.state.LastCheckTime = time.Now()
		if saveErr := uo.state.Save(); saveErr != nil {
			uo.logger.Warn().Err(saveErr).Msg("Failed to save update state after check")
		}
		if err != nil {
			uo.logger.Error().Err(err).Msg("Background update check failed")
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		// Log the result
		if releaseInfo.IsNewer {
			uo.logger.Info().
				Str("current_version", uo.currentVersion).
				Str("new_version", releaseInfo.Version).
				Msg("New version available")
		} else {
			uo.logger.Info().
				Str("current_version", uo.currentVersion).
				Msg("Already running the latest version")
		}

		// Call the callback if provided
		if callback != nil {
			callback(releaseInfo, nil)
		}
	}()
}
func (uo *UpdateChecker) CheckForUpdatesWithCallback(ctx context.Context, callback types.UpdateCallback) (*types.ReleaseInfo, error) {
	if callback != nil {
		callback(types.UpdateEvent{
			Type:    types.EventCheckStarted,
			Message: "Checking for updates...",
		})
	}

	releaseInfo, err := uo.CheckForUpdates(ctx)

	if err != nil {
		if callback != nil {
			callback(types.UpdateEvent{
				Type:    types.EventError,
				Message: "Failed to check for updates",
				Error:   err,
			})
		}
		return nil, err
	}

	if callback != nil {
		callback(types.UpdateEvent{
			Type:    types.EventCheckCompleted,
			Message: "Update check completed",
		})

		if releaseInfo.IsNewer {
			callback(types.UpdateEvent{
				Type:        types.EventUpdateAvailable,
				Message:     fmt.Sprintf("New version %s is available", releaseInfo.Version),
				ReleaseInfo: releaseInfo,
			})
		} else {
			callback(types.UpdateEvent{
				Type:    types.EventNoUpdateAvailable,
				Message: "You are running the latest version",
			})
		}
	}

	return releaseInfo, nil
}

// CheckForUpdatesAtStartup is a convenience method for checking updates at application startup
// It checks if an update check should be performed based on configuration and performs it in the background
func (uo *UpdateChecker) CheckForUpdatesAtStartup(ctx context.Context, callback func(*types.ReleaseInfo, error)) {
	uo.logger.Info().Msg("Checking for updates at startup")
	uo.CheckForUpdatesInBackground(ctx, callback)
}

// CheckForUpdates is a convenience method that checks for updates and returns the result
// This is an alias for CheckAllSources for backward compatibility
func (uo *UpdateChecker) CheckForUpdates(ctx context.Context) (*types.ReleaseInfo, error) {
	return uo.CheckAllSourcesWithOptions(ctx, false)
}

// // CheckForUpdatesForce checks for updates, ignoring cache
// // Use this for manual update checks triggered by user
func (uo *UpdateChecker) CheckForUpdatesForce(ctx context.Context) (*types.ReleaseInfo, error) {
	return uo.CheckAllSourcesWithOptions(ctx, true)
}

// CheckAllSources checks all configured update sources in priority order
// Returns the first successful result or an error if all sources fail
func (uo *UpdateChecker) CheckAllSources(ctx context.Context) (*types.ReleaseInfo, error) {
	return uo.CheckAllSourcesWithOptions(ctx, false)
}

// CheckAllSourcesWithOptions checks all configured update sources with options
// If forceRefresh is true, cache will be ignored
func (uo *UpdateChecker) CheckAllSourcesWithOptions(ctx context.Context, forceRefresh bool) (*types.ReleaseInfo, error) {
	if len(uo.checkers) == 0 {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryConfiguration,
			Message:  "no update sources configured",
		}
	}
	// Check cache first (unless forceRefresh is true or cacheManager is nil)
	if !forceRefresh && uo.cacheManager != nil {
		if cache, err := uo.cacheManager.Get(); err == nil && cache != nil {
			uo.logger.Info().
				Str("cached_version", cache.LatestVersion).
				Time("cache_time", cache.LastCheck).
				Msg("Using cached update information")

			// Compare with current version
			if cache.CachedManifest != nil {
				isNewer, err := util.CompareVersions(uo.currentVersion, cache.LatestVersion)
				if err == nil {
					cache.CachedManifest.IsNewer = isNewer
					return cache.CachedManifest, nil
				}
			}
		}
	} else if forceRefresh {
		uo.logger.Info().Msg("Force refresh enabled, ignoring cache")
	}

	uo.logger.Info().
		Int("source_count", len(uo.checkers)).
		Str("current_version", uo.currentVersion).
		Msg("Starting multi-source version check")

	var lastError error
	var failedSources []string

	// Try each source in priority order
	for i, checker := range uo.checkers {
		sourceName := checker.GetSourceName()

		uo.logger.Info().
			Int("attempt", i+1).
			Int("total", len(uo.checkers)).
			Str("source", sourceName).
			Msg("Checking update source")

		// Check this source
		releaseInfo, err := checker.CheckLatest(ctx, uo.currentVersion)
		if err != nil {
			// Log the failure and try next source
			uo.logger.Warn().
				Err(err).
				Str("source", sourceName).
				Int("attempt", i+1).
				Msg("Update source check failed, trying next source")

			failedSources = append(failedSources, sourceName)
			lastError = err
			continue
		}

		// Get NeedCheckChecksum from the corresponding source configuration
		for _, source := range uo.config.Sources {
			if source.Type == "github" && sourceName == fmt.Sprintf("github:%s", source.GitHubRepo) {
				releaseInfo.NeedCheckChecksum = source.NeedCheckChecksum
				break
			}
			if source.Type == "http" && sourceName == fmt.Sprintf("http:%s", source.ManifestURL) {
				releaseInfo.NeedCheckChecksum = source.NeedCheckChecksum
				break
			}
		}

		// Success! Cache the result and return
		uo.logger.Info().
			Str("source", sourceName).
			Str("version", releaseInfo.Version).
			Bool("is_newer", releaseInfo.IsNewer).
			Msg("Successfully retrieved version information")

		// Cache the result (if cacheManager is available)
		if uo.cacheManager != nil {
			if err := uo.cacheManager.Set(releaseInfo); err != nil {
				uo.logger.Warn().Err(err).Msg("Failed to cache update information")
			}
		}

		return releaseInfo, nil
	}

	// All sources failed
	uo.logger.Error().
		Strs("failed_sources", failedSources).
		Err(lastError).
		Msg("All update sources failed")

	return nil, &types.UpdateError{
		Category: types.ErrCategoryNetwork,
		Message:  fmt.Sprintf("all %d update sources failed", len(uo.checkers)),
		Cause:    lastError,
		Context: map[string]interface{}{
			"failed_sources": failedSources,
			"source_count":   len(uo.checkers),
		},
	}
}

// func (a *UpdateChecker) CheckForUpdates(ctx context.Context) (*types.UpdateCheckResult, error) {
// 	releaseInfo, err := a.orchestrator.CheckForUpdates(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &types.UpdateCheckResult{
// 		Version:      releaseInfo.Version,
// 		IsNewer:      releaseInfo.IsNewer,
// 		ReleaseNotes: releaseInfo.ReleaseNotes,
// 	}, nil
// }

// func (a *UpdateChecker) CheckForUpdatesForce(ctx context.Context) (*types.UpdateCheckResult, error) {
// 	releaseInfo, err := a.orchestrator.CheckForUpdatesForce(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &types.UpdateCheckResult{
// 		Version:      releaseInfo.Version,
// 		IsNewer:      releaseInfo.IsNewer,
// 		ReleaseNotes: releaseInfo.ReleaseNotes,
// 	}, nil
// }

// func (a *UpdateChecker) CheckForUpdatesWithCallback(ctx context.Context, callback types.UpdateCallback) (*types.UpdateCheckResult, error) {
// 	releaseInfo, err := a.orchestrator.CheckForUpdatesWithCallback(ctx, callback)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &types.UpdateCheckResult{
// 		Version:      releaseInfo.Version,
// 		IsNewer:      releaseInfo.IsNewer,
// 		ReleaseNotes: releaseInfo.ReleaseNotes,
// 	}, nil
// }
