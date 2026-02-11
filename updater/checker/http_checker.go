package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"github.com/ltaoo/velo/updater/types"
	"github.com/ltaoo/velo/updater/util"

	"github.com/rs/zerolog"
)

// HTTPVersionChecker implements VersionChecker for custom HTTP/HTTPS servers
type HTTPVersionChecker struct {
	manifestURL string
	httpClient  *http.Client
	logger      zerolog.Logger
}

// NewHTTPVersionChecker creates a new HTTP version checker
func NewHTTPVersionChecker(manifestURL string, logger *zerolog.Logger) *HTTPVersionChecker {
	return &HTTPVersionChecker{
		manifestURL: manifestURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With().Str("component", "http_checker").Logger(),
	}
}

// GetSourceName returns the name of this update source
func (h *HTTPVersionChecker) GetSourceName() string {
	return fmt.Sprintf("http:%s", h.manifestURL)
}

// CheckLatest checks for the latest version from a custom HTTP server
func (h *HTTPVersionChecker) CheckLatest(ctx context.Context, currentVersion string) (*types.ReleaseInfo, error) {
	h.logger.Info().
		Str("manifest_url", h.manifestURL).
		Str("current_version", currentVersion).
		Msg("Checking for updates from HTTP source")

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", h.manifestURL, nil)
	if err != nil {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to create HTTP request",
			Cause:    err,
			Context: map[string]interface{}{
				"manifest_url": h.manifestURL,
			},
		}
	}

	// Set headers
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to fetch manifest from HTTP source")
		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to fetch manifest from HTTP source",
			Cause:    err,
			Context: map[string]interface{}{
				"manifest_url": h.manifestURL,
			},
		}
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.logger.Error().
			Int("status_code", resp.StatusCode).
			Str("body", string(body)).
			Msg("HTTP server returned error")

		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  fmt.Sprintf("HTTP server returned status %d", resp.StatusCode),
			Context: map[string]interface{}{
				"status_code":  resp.StatusCode,
				"body":         string(body),
				"manifest_url": h.manifestURL,
			},
		}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to read manifest response")
		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to read manifest response",
			Cause:    err,
		}
	}

	// Parse manifest
	manifest, err := ParseManifest(body)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse manifest")
		return nil, err
	}

	// Parse published time
	publishedAt, err := time.Parse(time.RFC3339, manifest.PublishedAt)
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to parse published_at timestamp")
		publishedAt = time.Now() // fallback to current time
	}

	// Compare versions
	isNewer, err := util.CompareVersions(currentVersion, manifest.Version)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to compare versions")
		return nil, err
	}

	// Get asset for current platform
	asset, err := GetAssetForCurrentPlatform(manifest)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find asset for current platform")
		return nil, err
	}

	releaseInfo := &types.ReleaseInfo{
		Version:      manifest.Version,
		PublishedAt:  publishedAt,
		ReleaseNotes: manifest.ReleaseNotes,
		AssetURL:     asset.URL,
		AssetSize:    asset.Size,
		Checksum:     asset.Checksum,
		AssetName:    asset.Name,
		IsNewer:      isNewer,
	}

	h.logger.Info().
		Str("latest_version", manifest.Version).
		Bool("is_newer", isNewer).
		Str("asset_url", asset.URL).
		Msg("Successfully checked for updates")

	return releaseInfo, nil
}
