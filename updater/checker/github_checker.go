package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"github.com/ltaoo/velo/updater/types"
	"github.com/ltaoo/velo/updater/util"

	"github.com/rs/zerolog"
)

// GitHubVersionChecker implements VersionChecker for GitHub Releases
type GitHubVersionChecker struct {
	repo       string // format: "owner/repo"
	token      string // optional GitHub API token
	httpClient *http.Client
	logger     zerolog.Logger
}

// GitHubRelease represents a GitHub release API response
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a release asset from GitHub
type GitHubAsset struct {
	Name               string `json:"name"`
	URL                string `json:"url"` // API URL for downloading
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// NewGitHubVersionChecker creates a new GitHub version checker
func NewGitHubVersionChecker(repo, token string, logger *zerolog.Logger) *GitHubVersionChecker {
	return &GitHubVersionChecker{
		repo:  repo,
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With().Str("component", "github_checker").Logger(),
	}
}

// GetSourceName returns the name of this update source
func (g *GitHubVersionChecker) GetSourceName() string {
	return fmt.Sprintf("github:%s", g.repo)
}

// CheckLatest checks for the latest version from GitHub Releases
func (g *GitHubVersionChecker) CheckLatest(ctx context.Context, currentVersion string) (*types.ReleaseInfo, error) {
	g.logger.Info().
		Str("repo", g.repo).
		Str("current_version", currentVersion).
		Msg("Checking for updates from GitHub")

	// Construct GitHub API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", g.repo)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to create GitHub API request",
			Cause:    err,
			Context: map[string]interface{}{
				"repo": g.repo,
			},
		}
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	}

	// Execute request
	resp, err := g.httpClient.Do(req)
	if err != nil {
		g.logger.Error().Err(err).Msg("Failed to fetch GitHub release")
		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "failed to fetch GitHub release",
			Cause:    err,
			Context: map[string]interface{}{
				"repo": g.repo,
			},
		}
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusForbidden {
		rateLimitRemaining := resp.Header.Get("X-RateLimit-Remaining")
		rateLimitReset := resp.Header.Get("X-RateLimit-Reset")

		g.logger.Warn().
			Str("remaining", rateLimitRemaining).
			Str("reset", rateLimitReset).
			Msg("GitHub API rate limit exceeded")

		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  "GitHub API rate limit exceeded",
			Context: map[string]interface{}{
				"rate_limit_remaining": rateLimitRemaining,
				"rate_limit_reset":     rateLimitReset,
			},
		}
	}

	// Handle other HTTP errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		g.logger.Error().
			Int("status_code", resp.StatusCode).
			Str("body", string(body)).
			Msg("GitHub API returned error")

		return nil, &types.UpdateError{
			Category: types.ErrCategoryNetwork,
			Message:  fmt.Sprintf("GitHub API returned status %d", resp.StatusCode),
			Context: map[string]interface{}{
				"status_code": resp.StatusCode,
				"body":        string(body),
			},
		}
	}

	// Parse response
	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		g.logger.Error().Err(err).Msg("Failed to parse GitHub release response")
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "failed to parse GitHub release response",
			Cause:    err,
		}
	}

	// Validate required fields
	if release.TagName == "" {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "GitHub release missing tag_name field",
		}
	}

	// Validate tag_name is a valid semver format
	normalizedTag := strings.TrimPrefix(release.TagName, "v")
	if normalizedTag == "" || !util.IsValidSemver(normalizedTag) {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  fmt.Sprintf("GitHub release tag_name '%s' is not a valid semver format", release.TagName),
		}
	}

	if len(release.Assets) == 0 {
		return nil, &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "GitHub release has no assets",
		}
	}

	// Parse published time
	publishedAt, err := time.Parse(time.RFC3339, release.PublishedAt)
	if err != nil {
		g.logger.Warn().Err(err).Msg("Failed to parse published_at timestamp")
		publishedAt = time.Now() // fallback to current time
	}
	g.logger.Info().Str("current_version", currentVersion).Str("tag_name", release.TagName).Msg("Comparing versions")
	// Compare versions
	isNewer, err := util.CompareVersions(currentVersion, release.TagName)
	if err != nil {
		g.logger.Error().Err(err).Msg("Failed to compare versions")
		return nil, err
	}

	// Find the appropriate asset for current platform
	platformKey := GetPlatformKey()
	asset, checksum, err := g.findAssetForPlatform(release.Assets, platformKey)
	if err != nil {
		g.logger.Error().Err(err).Str("platform", platformKey).Msg("Failed to find asset for platform")
		return nil, err
	}

	asset_url := asset.BrowserDownloadURL
	// Determine which URL to use
	// For public repos: always use browser_download_url (no auth required)
	// For private repos with token: use API URL (requires Accept: application/octet-stream header)
	headers := make(map[string]string)
	g.logger.Info().
		Str("download_url", asset.BrowserDownloadURL).
		Bool("is_api_url", util.IsGitHubAPIURL(asset.URL)).
		Bool("has_token", g.token != "").
		Msg("Determining download URL")
	if asset.URL != "" && util.IsGitHubAPIURL(asset.URL) {
		// Only use API URL when we have a token
		// For public repos without token, use browser_download_url
		headers["Authorization"] = fmt.Sprintf("token %s", g.token)
		headers["Accept"] = "application/octet-stream"
		asset_url = asset.URL
		g.logger.Debug().
			Str("download_url", asset.URL).
			Str("token", g.token).
			Msg("Using API URL for private repository download")
	}
	releaseInfo := &types.ReleaseInfo{
		AssetName:    asset.Name,
		AssetURL:     asset_url,
		AssetSize:    asset.Size,
		Version:      release.TagName,
		ReleaseNotes: release.Body,
		Headers:      headers,
		PublishedAt:  publishedAt,
		Checksum:     checksum,
		IsNewer:      isNewer,
	}
	g.logger.Info().
		Str("latest_version", release.TagName).
		Bool("is_newer", isNewer).
		Str("asset_url", asset_url).
		Str("asset_name", asset.Name).
		Int("headers_count", len(headers)).
		Msg("Successfully checked for updates")

	return releaseInfo, nil
}

// findAssetForPlatform finds the appropriate asset for the current platform
// It looks for assets matching the platform key (e.g., "windows_amd64")
// and tries to find a corresponding checksum file
func (g *GitHubVersionChecker) findAssetForPlatform(assets []GitHubAsset, platformKey string) (*GitHubAsset, string, error) {
	var target_asset *GitHubAsset
	var checksum_asset *GitHubAsset

	// Build list of platform keys to search for
	// For macOS, we need to handle both Go's naming (darwin_amd64) and common naming (darwin_x86_64)
	platform_keys := []string{platformKey}
	if strings.HasPrefix(platformKey, "darwin_") {
		// Add alternative naming conventions for macOS
		if strings.HasSuffix(platformKey, "_amd64") {
			platform_keys = append(platform_keys, strings.Replace(platformKey, "_amd64", "_x86_64", 1))
		} else if strings.HasSuffix(platformKey, "_arm64") {
			// arm64 is commonly used as-is
			platform_keys = append(platform_keys, platformKey)
		}
	}

	// First pass: find the platform-specific asset and checksum file
	for i := range assets {
		asset := &assets[i]
		// Look for platform-specific archive
		for _, pk := range platform_keys {
			if strings.Contains(asset.Name, pk) &&
				(strings.HasSuffix(asset.Name, ".zip") ||
					strings.HasSuffix(asset.Name, ".tar.gz") ||
					strings.HasSuffix(asset.Name, ".tar.xz") ||
					strings.HasSuffix(asset.Name, ".dmg")) {
				target_asset = asset
				break
			}
		}

		// Look for checksums file
		if strings.Contains(strings.ToLower(asset.Name), "checksum") ||
			strings.HasSuffix(asset.Name, ".sha256") ||
			strings.HasSuffix(asset.Name, "_checksums.txt") {
			checksum_asset = asset
		}
	}

	if target_asset == nil {
		return nil, "", &types.UpdateError{
			Category: types.ErrCategoryValidation,
			Message:  "no asset found for current platform",
			Context: map[string]interface{}{
				"platform": platformKey,
			},
		}
	}
	headers := make(map[string]string)
	if g.token != "" {
		headers["Authorization"] = fmt.Sprintf("token %s", g.token)
	}
	// Try to extract checksum from checksum file if available
	checksum := ""
	if checksum_asset != nil {
		// Use API URL for private repos, browser URL for public repos
		checksum_url := checksum_asset.BrowserDownloadURL
		if g.token != "" && checksum_asset.BrowserDownloadURL != "" {
			checksum_url = checksum_asset.BrowserDownloadURL
		}
		checksum, _ = g.extractChecksumForAsset(checksum_url, headers, target_asset.Name)
	}

	return target_asset, checksum, nil
}

// extractChecksumForAsset downloads and parses a checksum file to find the checksum for a specific asset
func (g *GitHubVersionChecker) extractChecksumForAsset(checksumURL string, headers map[string]string, assetName string) (string, error) {
	// Create request with proper headers
	req, err := http.NewRequest("GET", checksumURL, nil)
	if err != nil {
		return "", err
	}

	// Add authentication for private repos
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	// if g.token != "" {
	// 	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	// }

	// For API URLs, add Accept header
	if strings.Contains(checksumURL, "api.github.com") {
		req.Header.Set("Accept", "application/octet-stream")
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download checksum file: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse checksum file (format: "checksum  filename" or "checksum filename")
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by whitespace
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			checksum := parts[0]
			filename := parts[len(parts)-1]

			// Check if this line is for our asset
			if filename == assetName || strings.Contains(filename, assetName) {
				return checksum, nil
			}
		}
	}

	return "", fmt.Errorf("checksum not found for asset %s", assetName)
}
