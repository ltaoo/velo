package buildcfg

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ltaoo/velo/updater/types"
)

type AppSection struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Icon        string `json:"icon"`
}

type MacOSSection struct {
	BundleID          string            `json:"bundle_id"`
	IconFile          string            `json:"icon_file"`
	MinSystemVersion  string            `json:"min_system_version"`
	Category          string            `json:"category"`
	Entitlements      map[string]bool   `json:"entitlements"`
	DMG               DMGSection        `json:"dmg"`
}

type DMGSection struct {
	WindowSize struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"window_size"`
	IconSize  int `json:"icon_size"`
	Positions struct {
		App          Position `json:"app"`
		Applications Position `json:"applications"`
	} `json:"positions"`
	Background struct {
		AutoGenerate bool   `json:"auto_generate"`
		CustomPath   string `json:"custom_path"`
	} `json:"background"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type WindowsSection struct {
	CompanyName      string `json:"company_name"`
	ProductName      string `json:"product_name"`
	FileDescription  string `json:"file_description"`
	InternalName     string `json:"internal_name"`
	OriginalFilename string `json:"original_filename"`
	IconFiles        struct {
		PNG   string `json:"png"`
		PNG16 string `json:"png16"`
	} `json:"icon_files"`
}

type LinuxSection struct {
	IconFile     string `json:"icon_file"`
	DesktopEntry struct {
		Categories string `json:"categories"`
		Keywords   string `json:"keywords"`
	} `json:"desktop_entry"`
}

type ConfigFile struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type BuildSection struct {
	ConfigFiles  []ConfigFile `json:"config_files"`
	ExcludeFiles []string     `json:"exclude_files"`
}

type ReleaseSection struct {
	Footer string `json:"footer"`
}

type BinarySection struct {
	ProjectName string `json:"project_name"`
}

type UpdateSourceConfig struct {
	Type              string `json:"type"`
	Priority          int    `json:"priority"`
	Enabled           bool   `json:"enabled"`
	NeedCheckChecksum bool   `json:"need_check_checksum"`
	GitHubRepo        string `json:"github_repo,omitempty"`
	GitHubToken       string `json:"github_token,omitempty"`
	ManifestURL       string `json:"manifest_url,omitempty"`
	SelfURL           string `json:"self_url,omitempty"`
}

type UpdateSection struct {
	Enabled        bool                 `json:"enabled"`
	CheckFrequency string               `json:"check_frequency"`
	Channel        string               `json:"channel"`
	AutoDownload   bool                 `json:"auto_download"`
	Timeout        int                  `json:"timeout"`
	Sources        []UpdateSourceConfig `json:"sources"`
}

func (u *UpdateSection) ToUpdaterConfig() *types.UpdateConfig {
	cfg := &types.UpdateConfig{
		Enabled:        u.Enabled,
		CheckFrequency: u.CheckFrequency,
		Channel:        u.Channel,
		AutoDownload:   u.AutoDownload,
		Timeout:        u.Timeout,
	}
	if cfg.CheckFrequency == "" {
		cfg.CheckFrequency = "startup"
	}
	if cfg.Channel == "" {
		cfg.Channel = "stable"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 300
	}
	for _, s := range u.Sources {
		cfg.Sources = append(cfg.Sources, types.UpdateSource{
			Type:              s.Type,
			Priority:          s.Priority,
			Enabled:           s.Enabled,
			NeedCheckChecksum: s.NeedCheckChecksum,
			GitHubRepo:        s.GitHubRepo,
			GitHubToken:       s.GitHubToken,
			ManifestURL:       s.ManifestURL,
			SelfURL:           s.SelfURL,
		})
	}
	return cfg
}

type Config struct {
	App       AppSection    `json:"app"`
	Binary    BinarySection `json:"binary"`
	Platforms struct {
		MacOS   MacOSSection   `json:"macos"`
		Windows WindowsSection `json:"windows"`
		Linux   LinuxSection   `json:"linux"`
	} `json:"platforms"`
	Build   BuildSection   `json:"build"`
	Release ReleaseSection `json:"release"`
	Update  UpdateSection  `json:"update"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if c.App.Version == "" {
		return fmt.Errorf("app.version is required")
	}
	return nil
}

func (c *Config) DisplayName() string {
	if c.App.DisplayName != "" {
		return c.App.DisplayName
	}
	return c.App.Name
}

func (c *Config) ProjectName() string {
	if c.Binary.ProjectName != "" {
		return c.Binary.ProjectName
	}
	return c.App.Name
}
