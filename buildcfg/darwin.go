package buildcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>{{.Name}}</string>
  <key>CFBundleIdentifier</key>
  <string>{{.BundleID}}</string>
  <key>CFBundleName</key>
  <string>{{.Name}}</string>
  <key>CFBundleDisplayName</key>
  <string>{{.DisplayName}}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${BUILD_NUMBER}</string>
  <key>LSMinimumSystemVersion</key>
  <string>{{.MinSystemVersion}}</string>
  <key>NSHighResolutionCapable</key>
  <true/>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleDevelopmentRegion</key>
  <string>zh_CN</string>
  <key>LSApplicationCategoryType</key>
  <string>{{.Category}}</string>
  <key>NSHumanReadableCopyright</key>
  <string>{{.Copyright}}</string>
</dict>
</plist>
`))

var entitlementsTmpl = template.Must(template.New("entitlements").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>{{range .}}
  <key>{{.Key}}</key>
  <{{.Value}}/>{{end}}
</dict>
</plist>
`))

type plistData struct {
	Name             string
	DisplayName      string
	BundleID         string
	MinSystemVersion string
	Category         string
	Copyright        string
}

type entitlementEntry struct {
	Key   string
	Value string
}

func GenerateDarwinPlist(cfg *Config, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	minVer := cfg.Platforms.MacOS.MinSystemVersion
	if minVer == "" {
		minVer = "10.13"
	}
	category := cfg.Platforms.MacOS.Category
	if category == "" {
		category = "public.app-category.utilities"
	}
	bundleID := cfg.Platforms.MacOS.BundleID
	if bundleID == "" {
		bundleID = "com.app." + cfg.App.Name
	}

	data := plistData{
		Name:             cfg.App.Name,
		DisplayName:      cfg.DisplayName(),
		BundleID:         bundleID,
		MinSystemVersion: minVer,
		Category:         category,
		Copyright:        fmt.Sprintf("Copyright © 2024 %s", cfg.App.Author),
	}

	f, err := os.Create(filepath.Join(outDir, "Info.plist.template"))
	if err != nil {
		return fmt.Errorf("creating Info.plist.template: %w", err)
	}
	defer f.Close()

	if err := plistTmpl.Execute(f, data); err != nil {
		return fmt.Errorf("rendering Info.plist.template: %w", err)
	}

	return generateEntitlements(cfg, outDir)
}

func generateEntitlements(cfg *Config, outDir string) error {
	entMap := map[string]string{
		"network_client": "com.apple.security.network.client",
		"network_server": "com.apple.security.network.server",
		"camera":         "com.apple.security.device.camera",
		"microphone":     "com.apple.security.device.microphone",
		"location":       "com.apple.security.personal-information.location",
		"photos":         "com.apple.security.personal-information.photos-library",
		"contacts":       "com.apple.security.personal-information.addressbook",
		"calendars":      "com.apple.security.personal-information.calendars",
	}

	var entries []entitlementEntry
	for key, enabled := range cfg.Platforms.MacOS.Entitlements {
		if !enabled {
			continue
		}
		if entKey, ok := entMap[key]; ok {
			entries = append(entries, entitlementEntry{Key: entKey, Value: "true"})
		}
	}

	f, err := os.Create(filepath.Join(outDir, "entitlements.plist"))
	if err != nil {
		return fmt.Errorf("creating entitlements.plist: %w", err)
	}
	defer f.Close()

	return entitlementsTmpl.Execute(f, entries)
}
