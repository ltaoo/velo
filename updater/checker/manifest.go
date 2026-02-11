package checker

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/ltaoo/velo/updater/types"
)

// ParseManifest parses a release manifest from JSON data
func ParseManifest(data []byte) (*types.ReleaseManifest, error) {
	if len(data) == 0 {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest data is empty",
		}
	}

	var manifest types.ReleaseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "failed to parse manifest JSON",
			Cause:    err,
		}
	}

	// Validate the parsed manifest
	if err := ValidateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ValidateManifest validates that a manifest contains all required fields
func ValidateManifest(manifest *types.ReleaseManifest) error {
	if manifest == nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest is nil",
		}
	}

	// Validate version field
	if manifest.Version == "" {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest missing required field: version",
		}
	}

	// Validate published_at field
	if manifest.PublishedAt == "" {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest missing required field: published_at",
		}
	}

	// Validate published_at is valid RFC3339 timestamp
	if _, err := time.Parse(time.RFC3339, manifest.PublishedAt); err != nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest published_at is not valid RFC3339 format",
			Cause:    err,
		}
	}

	// Validate release_notes field (can be empty but must exist)
	// Note: ReleaseNotes is allowed to be empty string, so we don't check for emptiness

	// Validate assets field
	if manifest.Assets == nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest missing required field: assets",
		}
	}

	if len(manifest.Assets) == 0 {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest assets map is empty",
		}
	}

	// Validate each asset
	for platform, asset := range manifest.Assets {
		if err := validateAsset(platform, &asset); err != nil {
			return err
		}
	}

	return nil
}

// validateAsset validates a single asset entry
func validateAsset(platform string, asset *types.AssetInfo) error {
	if asset == nil {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  fmt.Sprintf("asset for platform %s is nil", platform),
		}
	}

	if asset.URL == "" {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  fmt.Sprintf("asset for platform %s missing URL", platform),
		}
	}

	if asset.Size <= 0 {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  fmt.Sprintf("asset for platform %s has invalid size: %d", platform, asset.Size),
		}
	}

	if asset.Checksum == "" {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  fmt.Sprintf("asset for platform %s missing checksum", platform),
		}
	}

	if asset.Name == "" {
		return &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  fmt.Sprintf("asset for platform %s missing name", platform),
		}
	}

	return nil
}

// GetPlatformKey returns the platform key for the current system in the format "{os}_{arch}"
func GetPlatformKey() string {
	return fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
}

// NormalizePlatformKey normalizes a platform key to the standard format
func NormalizePlatformKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// GetAssetForCurrentPlatform retrieves the asset for the current platform from a manifest
func GetAssetForCurrentPlatform(manifest *types.ReleaseManifest) (*types.AssetInfo, error) {
	if manifest == nil {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "manifest is nil",
		}
	}

	platformKey := GetPlatformKey()
	asset, exists := manifest.Assets[platformKey]
	if !exists {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  fmt.Sprintf("no asset found for platform: %s", platformKey),
			Context: map[string]interface{}{
				"platform":         platformKey,
				"available_assets": getAssetKeys(manifest.Assets),
			},
		}
	}

	return &asset, nil
}

// getAssetKeys returns a list of all asset keys in the manifest
func getAssetKeys(assets map[string]types.AssetInfo) []string {
	keys := make([]string, 0, len(assets))
	for key := range assets {
		keys = append(keys, key)
	}
	return keys
}
