package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ltaoo/velo/updater/types"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BaseDir  string
	Filename string
	FullPath string
	Existing bool
	Error    error
	Debug    bool
}

func New() (*Config, error) {
	exe, _ := os.Executable()
	exe_dir := filepath.Dir(exe)

	// Determine base directory based on whether we're running via `go run` or compiled binary
	var base_dir string

	// Check if we were relaunched from temp app bundle (macOS dev mode)
	if originalCwd := os.Getenv("__WEBVIEW_ORIGINAL_CWD"); originalCwd != "" {
		base_dir = originalCwd
	} else if filepath.Base(exe_dir) == "exe" || strings.Contains(exe, "go-build") {
		// Running via `go run` - use source directory
		if _, this_file, _, ok := runtime.Caller(0); ok {
			cfg_dir := filepath.Dir(this_file)
			proj_root := filepath.Dir(cfg_dir) // Go up from config to project root
			base_dir = proj_root
		} else {
			base_dir = exe_dir // Fallback
		}
	} else {
		// Running as compiled executable - use executable directory
		base_dir = exe_dir
	}

	// Look for config file
	filename := "config.yaml"
	config_filepath := filepath.Join(base_dir, filename)
	var has_config bool

	if _, err := os.Stat(config_filepath); err == nil {
		has_config = true
	}
	c := &Config{
		BaseDir:  base_dir,
		Filename: filename,
		FullPath: config_filepath,
		Existing: has_config,
	}
	return c, nil
}

// GetDebugInfo returns debug information about how the base directory was determined
func (c *Config) GetDebugInfo() map[string]string {
	exe, _ := os.Executable()
	exe_dir := filepath.Dir(exe)

	info := map[string]string{
		"executable":    exe,
		"exe_dir":       exe_dir,
		"base_dir":      c.BaseDir,
		"config_path":   c.FullPath,
		"config_exists": fmt.Sprintf("%v", c.Existing),
	}

	// Determine run mode
	if filepath.Base(exe_dir) == "exe" || strings.Contains(exe, "go-build") {
		info["run_mode"] = "go run (development)"
	} else {
		info["run_mode"] = "compiled binary"
	}

	return info
}

func (c *Config) ReadFromConfig() error {
	return nil
}

func EnsureDirIfMissing(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return err
}

// UpdateConfig defines the configuration for the update system
type UpdateConfig struct {
	// Enabled controls whether auto-update is enabled
	Enabled bool `yaml:"enabled"`

	// CheckFrequency defines when to check for updates: "startup", "daily", "weekly", "manual"
	CheckFrequency string `yaml:"check_frequency"`

	// Channel defines the update channel: "stable", "beta"
	Channel string `yaml:"channel"`

	// AutoDownload controls whether to automatically download updates (false = notify only)
	AutoDownload bool `yaml:"auto_download"`

	// Sources is the list of update sources (ordered by priority)
	Sources []UpdateSource `yaml:"sources"`

	// Timeout is the timeout in seconds for update operations
	Timeout int `yaml:"timeout"`
}

// UpdateSource represents a single update source
type UpdateSource struct {
	// Type is the source type: "github", "http"
	Type string `yaml:"type"`

	// Priority determines the order (lower number = higher priority)
	Priority int `yaml:"priority"`

	// GitHubRepo is the GitHub repository in "owner/repo" format
	GitHubRepo string `yaml:"github_repo,omitempty"`

	// GitHubToken is the optional GitHub API token
	GitHubToken string `yaml:"github_token,omitempty"`

	// ManifestURL is the URL for custom HTTP manifest
	ManifestURL string `yaml:"manifest_url,omitempty"`

	// SelfURL is the URL for self-hosted update source
	SelfURL string `yaml:"self_url,omitempty"`

	// Enabled controls whether this source is active
	Enabled bool `yaml:"enabled"`
}

// DefaultUpdateConfig returns the default update configuration
func DefaultUpdateConfig() *UpdateConfig {
	return &UpdateConfig{
		Enabled:        true,
		CheckFrequency: "startup",
		Channel:        "stable",
		AutoDownload:   false,
		Timeout:        300, // 5 minutes
		Sources:        []UpdateSource{},
	}
}

// LoadUpdateConfig loads update configuration from a YAML file
// If the file doesn't exist or has format errors, it returns default config with a warning
func LoadUpdateConfig(path string) (*UpdateConfig, error) {
	// Check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default config
			return DefaultUpdateConfig(), fmt.Errorf("config file not found at %s, using defaults: %w", path, err)
		}
		// Other read error
		return DefaultUpdateConfig(), fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config UpdateConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		// Format error, return default config
		return DefaultUpdateConfig(), fmt.Errorf("failed to parse config file (invalid YAML format), using defaults: %w", err)
	}

	// Validate and apply defaults for missing fields
	if config.CheckFrequency == "" {
		config.CheckFrequency = "startup"
	}
	if config.Channel == "" {
		config.Channel = "stable"
	}
	if config.Timeout <= 0 {
		config.Timeout = 300
	}

	return &config, nil
}

// SaveUpdateConfig saves update configuration to a YAML file
func SaveUpdateConfig(path string, config *UpdateConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := EnsureDirIfMissing(dir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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
