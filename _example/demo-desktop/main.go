package main

import (
	"embed"
	"path/filepath"
	"runtime"

	"example/simple/internal/desktopapp"
)

//go:embed frontend
var frontendFS embed.FS

//go:embed app-config.json
var appConfigData []byte

//go:embed assets/appicon.png
var appIcon []byte

var Version = "1.0.0"
var Mode = "dev"

func projectDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filename)
}

func main() {
	desktopapp.Run(desktopapp.Assets{
		AppConfigData: appConfigData,
		AppIcon:       appIcon,
		FrontendFS:    frontendFS,
		Mode:          Mode,
		ProjectDir:    projectDir(),
		Version:       Version,
	})
}
