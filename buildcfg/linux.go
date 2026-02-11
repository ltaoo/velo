package buildcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

var desktopTmpl = template.Must(template.New("desktop").Parse(`[Desktop Entry]
Version=1.0
Type=Application
Name={{.Name}}
Name[zh_CN]={{.Name}}
Comment={{.Description}}
Comment[zh_CN]={{.Description}}
Exec={{.Name}}
Icon={{.Icon}}
Terminal=false
Categories={{.Categories}}
Keywords={{.Keywords}}
StartupNotify=true
`))

type desktopData struct {
	Name        string
	Description string
	Icon        string
	Categories  string
	Keywords    string
}

func GenerateLinuxDesktop(cfg *Config, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	iconBase := cfg.App.Name
	if cfg.Platforms.Linux.IconFile != "" {
		ext := filepath.Ext(cfg.Platforms.Linux.IconFile)
		iconBase = filepath.Base(cfg.Platforms.Linux.IconFile)
		iconBase = iconBase[:len(iconBase)-len(ext)]
	}

	data := desktopData{
		Name:        cfg.App.Name,
		Description: cfg.App.Description,
		Icon:        iconBase,
		Categories:  cfg.Platforms.Linux.DesktopEntry.Categories,
		Keywords:    cfg.Platforms.Linux.DesktopEntry.Keywords,
	}

	f, err := os.Create(filepath.Join(outDir, "app.desktop.template"))
	if err != nil {
		return fmt.Errorf("creating app.desktop.template: %w", err)
	}
	defer f.Close()

	return desktopTmpl.Execute(f, data)
}
