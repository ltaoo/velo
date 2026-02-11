package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"github.com/ltaoo/velo/updater/types"
)

func TestCacheManager_SetAndGet(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")

	// Create cache manager with 1 hour TTL
	cm := NewCacheManager(cachePath, 1*time.Hour)

	// Create test release info
	releaseInfo := &types.ReleaseInfo{
		Version:      "2.0.0",
		PublishedAt:  time.Now(),
		ReleaseNotes: "Test release notes",
		AssetURL:     "https://example.com/release.zip",
		AssetSize:    1024,
		Checksum:     "abc123",
		AssetName:    "test_asset.exe",
		IsNewer:      true,
	}

	// Set cache
	if err := cm.Set(releaseInfo); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Verify cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}

	// Get cache
	cache, err := cm.Get()
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if cache == nil {
		t.Fatal("Cache is nil")
	}

	// Verify cache contents
	if cache.LatestVersion != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", cache.LatestVersion)
	}

	if cache.CachedManifest == nil {
		t.Fatal("Cached manifest is nil")
	}

	if cache.CachedManifest.Version != "2.0.0" {
		t.Errorf("Expected cached version 2.0.0, got %s", cache.CachedManifest.Version)
	}
}

func TestCacheManager_Expiry(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")

	// Create cache manager with very short TTL
	cm := NewCacheManager(cachePath, 100*time.Millisecond)

	// Create test release info
	releaseInfo := &types.ReleaseInfo{
		Version:      "2.0.0",
		PublishedAt:  time.Now(),
		ReleaseNotes: "Test release notes",
		AssetName:    "test_asset.exe",
		IsNewer:      true,
	}

	// Set cache
	if err := cm.Set(releaseInfo); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Verify cache is valid immediately
	if !cm.IsValid() {
		t.Error("Cache should be valid immediately after setting")
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Verify cache is expired
	if cm.IsValid() {
		t.Error("Cache should be expired after TTL")
	}

	// Get should return nil for expired cache
	cache, err := cm.Get()
	if err != nil {
		t.Fatalf("Get should not return error for expired cache: %v", err)
	}

	if cache != nil {
		t.Error("Get should return nil for expired cache")
	}
}

func TestCacheManager_Invalidate(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")

	// Create cache manager
	cm := NewCacheManager(cachePath, 1*time.Hour)

	// Create test release info
	releaseInfo := &types.ReleaseInfo{
		Version:   "2.0.0",
		AssetName: "test_asset.exe",
		IsNewer:   true,
	}

	// Set cache
	if err := cm.Set(releaseInfo); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Verify cache exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file should exist")
	}

	// Invalidate cache
	if err := cm.Invalidate(); err != nil {
		t.Fatalf("Failed to invalidate cache: %v", err)
	}

	// Verify cache file is removed
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("Cache file should be removed after invalidation")
	}

	// Get should return nil after invalidation
	cache, err := cm.Get()
	if err != nil {
		t.Fatalf("Get should not return error after invalidation: %v", err)
	}

	if cache != nil {
		t.Error("Get should return nil after invalidation")
	}
}

func TestCacheManager_NoCache(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent_cache.json")

	// Create cache manager
	cm := NewCacheManager(cachePath, 1*time.Hour)

	// Get should return nil when no cache exists
	cache, err := cm.Get()
	if err != nil {
		t.Fatalf("Get should not return error when cache doesn't exist: %v", err)
	}

	if cache != nil {
		t.Error("Get should return nil when cache doesn't exist")
	}

	// IsValid should return false when no cache exists
	if cm.IsValid() {
		t.Error("IsValid should return false when cache doesn't exist")
	}
}
