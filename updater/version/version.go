package version

import (
	"fmt"
	"strings"
	"github.com/ltaoo/velo/updater/types"
	"github.com/ltaoo/velo/updater/util"

	"github.com/blang/semver/v4"
)

const DevVersion = "(dev)"

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

func DetectEnvironment(version string) Environment {
	if version == DevVersion || version == "" || version == "unknown" {
		return EnvironmentDevelopment
	}

	if util.IsValidSemver(strings.TrimPrefix(version, "v")) {
		return EnvironmentProduction
	}

	return EnvironmentDevelopment
}

func GetVersionNumber(version string) string {
	return strings.TrimPrefix(version, "v")
}

type UpdateMode string

const (
	UpdateModeDisabled    UpdateMode = "disabled"
	UpdateModeManual      UpdateMode = "manual"
	UpdateModeAutomatic   UpdateMode = "automatic"
	UpdateModeDevelopment UpdateMode = "development"
)

func DetermineUpdateMode(env Environment, cfg *types.UpdateConfig) UpdateMode {
	if cfg == nil {
		return UpdateModeDisabled
	}

	if !cfg.Enabled {
		return UpdateModeDisabled
	}

	switch env {
	case EnvironmentDevelopment:
		if cfg.DevModeEnabled {
			return UpdateModeDevelopment
		}
		return UpdateModeManual

	case EnvironmentProduction:
		switch cfg.CheckFrequency {
		case "manual":
			return UpdateModeManual
		case "startup", "daily", "weekly":
			return UpdateModeAutomatic
		default:
			return UpdateModeManual
		}

	default:
		return UpdateModeManual
	}
}

func (m UpdateMode) String() string {
	switch m {
	case UpdateModeDisabled:
		return "disabled"
	case UpdateModeManual:
		return "manual"
	case UpdateModeAutomatic:
		return "automatic"
	case UpdateModeDevelopment:
		return "development"
	default:
		return "unknown"
	}
}

func (m UpdateMode) ShouldCheckAtStartup() bool {
	return m == UpdateModeAutomatic || m == UpdateModeDevelopment
}

func (m UpdateMode) IsEnabled() bool {
	return m != UpdateModeDisabled
}

type VersionInfo struct {
	Version     string
	Environment Environment
	UpdateMode  UpdateMode
	RawVersion  string
}

func ParseVersionInfo(version string, cfg *types.UpdateConfig) *VersionInfo {
	env := DetectEnvironment(version)
	mode := DetermineUpdateMode(env, cfg)

	return &VersionInfo{
		Version:     GetVersionNumber(version),
		Environment: env,
		UpdateMode:  mode,
		RawVersion:  version,
	}
}

func (vi *VersionInfo) IsDevelopment() bool {
	return vi.Environment == EnvironmentDevelopment
}

func (vi *VersionInfo) CanCheckUpdates() bool {
	return vi.UpdateMode.IsEnabled() && !vi.IsDevelopment()
}

func (vi *VersionInfo) ShouldForceUpdateCheck() bool {
	return vi.Environment == EnvironmentDevelopment && vi.UpdateMode == UpdateModeDevelopment
}

func (vi *VersionInfo) String() string {
	return fmt.Sprintf("VersionInfo{Version=%s, Environment=%s, UpdateMode=%s}",
		vi.Version, vi.Environment, vi.UpdateMode)
}

func CompareVersions(current, latest string) (bool, error) {
	currentVer, err := semver.Parse(normalizeVersion(current))
	if err != nil {
		return false, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "invalid current version format",
			Cause:    err,
			Context: map[string]interface{}{
				"current_version": current,
			},
		}
	}

	latestVer, err := semver.Parse(normalizeVersion(latest))
	if err != nil {
		return false, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "invalid latest version format",
			Cause:    err,
			Context: map[string]interface{}{
				"latest_version": latest,
			},
		}
	}

	return latestVer.GT(currentVer), nil
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}
