package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ltaoo/velo/updater/types"
)

// UpdateCache represents cached update check information
type UpdateCache struct {
	LastCheck      time.Time          `json:"last_check"`
	LatestVersion  string             `json:"latest_version"`
	CachedManifest *types.ReleaseInfo `json:"cached_manifest,omitempty"`
	ChecksumValid  bool               `json:"checksum_valid"`
	CacheExpiry    time.Time          `json:"cache_expiry"`
}

// CacheManager manages update check caching
type CacheManager struct {
	cachePath string
	ttl       time.Duration // Time-to-live for cache
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cachePath string, ttl time.Duration) *CacheManager {
	return &CacheManager{
		cachePath: cachePath,
		ttl:       ttl,
	}
}

// Get retrieves cached update information if valid
func (cm *CacheManager) Get() (*UpdateCache, error) {
	// Check if cache file exists
	if _, err := os.Stat(cm.cachePath); os.IsNotExist(err) {
		return nil, nil // No cache available
	}

	// Read cache file
	data, err := os.ReadFile(cm.cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse cache
	var cache UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	// Check if cache is expired
	if time.Now().After(cache.CacheExpiry) {
		return nil, nil // Cache expired
	}

	// Validate cached version format
	if cache.LatestVersion == "" {
		return nil, nil // Invalid cache, treat as no cache
	}

	return &cache, nil
}

// Set stores update information in cache
func (cm *CacheManager) Set(releaseInfo *types.ReleaseInfo) error {
	cache := UpdateCache{
		LastCheck:      time.Now(),
		LatestVersion:  releaseInfo.Version,
		CachedManifest: releaseInfo,
		ChecksumValid:  true,
		CacheExpiry:    time.Now().Add(cm.ttl),
	}

	// Ensure directory exists
	dir := filepath.Dir(cm.cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to temporary file first
	tempPath := cm.cachePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Atomically rename
	if err := os.Rename(tempPath, cm.cachePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// Invalidate removes the cache
func (cm *CacheManager) Invalidate() error {
	if err := os.Remove(cm.cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

// IsValid checks if cached information is still valid
func (cm *CacheManager) IsValid() bool {
	cache, err := cm.Get()
	if err != nil || cache == nil {
		return false
	}
	return time.Now().Before(cache.CacheExpiry)
}
