package buildcfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type winresJSON struct {
	RTGroupIcon map[string]interface{} `json:"RT_GROUP_ICON"`
	RTVersion   map[string]interface{} `json:"RT_VERSION"`
	RTManifest  map[string]interface{} `json:"RT_MANIFEST"`
}

func GenerateWinres(cfg *Config, baseDir, outDir string) error {
	winresDir := filepath.Join(outDir, "winres")
	if err := os.MkdirAll(winresDir, 0755); err != nil {
		return fmt.Errorf("creating winres dir: %w", err)
	}

	png := filepath.Base(cfg.Platforms.Windows.IconFiles.PNG)
	png16 := filepath.Base(cfg.Platforms.Windows.IconFiles.PNG16)

	data := winresJSON{
		RTGroupIcon: map[string]interface{}{
			"APP": map[string]interface{}{
				"0000": []string{png, png16},
			},
		},
		RTVersion: map[string]interface{}{
			"#1": map[string]interface{}{
				"0000": map[string]interface{}{
					"fixed": map[string]string{
						"file_version":    cfg.App.Version + ".0",
						"product_version": cfg.App.Version + ".0",
					},
					"info": map[string]interface{}{
						"0409": map[string]string{
							"CompanyName":      cfg.Platforms.Windows.CompanyName,
							"FileDescription":  cfg.Platforms.Windows.FileDescription,
							"FileVersion":      cfg.App.Version,
							"InternalName":     cfg.Platforms.Windows.InternalName,
							"LegalCopyright":   fmt.Sprintf("Copyright © 2024 %s", cfg.Platforms.Windows.CompanyName),
							"OriginalFilename": cfg.Platforms.Windows.OriginalFilename,
							"ProductName":      cfg.Platforms.Windows.ProductName,
							"ProductVersion":   cfg.App.Version,
						},
					},
				},
			},
		},
		RTManifest: map[string]interface{}{
			"#1": map[string]interface{}{
				"0409": map[string]interface{}{
					"identity": map[string]string{
						"name":    cfg.App.Name,
						"version": "1.0.0.0",
					},
					"description":     cfg.Platforms.Windows.FileDescription,
					"minimum-os":      "win7",
					"execution-level":  "as invoker",
					"dpi-awareness":   "system",
					"long-path-aware": "true",
				},
			},
		},
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling winres.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(winresDir, "winres.json"), out, 0644); err != nil {
		return fmt.Errorf("writing winres.json: %w", err)
	}

	// Copy icon files to winres directory
	for _, iconSrc := range []string{cfg.Platforms.Windows.IconFiles.PNG, cfg.Platforms.Windows.IconFiles.PNG16} {
		if iconSrc == "" {
			continue
		}
		src := iconSrc
		if !filepath.IsAbs(src) {
			src = filepath.Join(baseDir, src)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: icon file not found: %s\n", src)
			continue
		}
		dst := filepath.Join(winresDir, filepath.Base(iconSrc))
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("copying icon %s: %w", iconSrc, err)
		}
	}

	return nil
}
