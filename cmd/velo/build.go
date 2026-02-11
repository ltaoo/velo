package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ltaoo/velo/buildcfg"
)

type darwinCreds struct {
	AppleID        string
	TeamID         string
	P12File        string
	P12Password    string
	P8File         string
	APIKeyID       string
	APIKeyIssuerID string
}

func loadEnv(projectPath string) map[string]string {
	env := map[string]string{}
	f, err := os.Open(filepath.Join(projectPath, ".env"))
	if err != nil {
		return env
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			env[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), "\"'")
		}
	}
	return env
}

func envOr(env map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := env[k]; v != "" {
			return v
		}
	}
	return ""
}

func decodeBase64ToTempFile(b64, prefix, suffix string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("decoding %s base64: %w", prefix, err)
	}
	f, err := os.CreateTemp("", prefix+"*"+suffix)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func loadDarwinCreds(projectPath string) *darwinCreds {
	env := loadEnv(projectPath)
	c := &darwinCreds{
		AppleID:        env["APPLE_ID"],
		TeamID:         envOr(env, "TEAM_ID", "APNS_TEAM_ID"),
		P12File:        env["P12_FILE"],
		P12Password:    envOr(env, "P12_PASSWORD", "MAC_CERT_PASSWORD"),
		P8File:         env["P8_FILE"],
		APIKeyID:       envOr(env, "API_KEY_ID", "APNS_KEY_ID"),
		APIKeyIssuerID: env["API_KEY_ISSUER_ID"],
	}

	// Support base64-encoded P12
	if c.P12File == "" {
		if b64 := envOr(env, "P12_BASE64", "MAC_CERT_P12_BASE64"); b64 != "" {
			path, err := decodeBase64ToTempFile(b64, "velo-p12-", ".p12")
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			} else {
				c.P12File = path
			}
		}
	}

	// Support base64-encoded P8
	if c.P8File == "" {
		if b64 := envOr(env, "P8_BASE64", "APNS_AUTH_KEY_BASE64", "APNS_AUTH_KEY_P8"); b64 != "" {
			path, err := decodeBase64ToTempFile(b64, "velo-p8-", ".p8")
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			} else {
				c.P8File = path
			}
		}
	}

	if c.AppleID == "" || c.TeamID == "" {
		return nil
	}
	return c
}

type target struct {
	goos   string
	goarch string
	cgo    string
}

var buildTargets = map[string][]target{
	"darwin":  {{goos: "darwin", goarch: "arm64", cgo: "1"}, {goos: "darwin", goarch: "amd64", cgo: "1"}},
	"windows": {{goos: "windows", goarch: "amd64", cgo: "1"}},
	"linux":   {{goos: "linux", goarch: "amd64", cgo: "0"}, {goos: "linux", goarch: "arm64", cgo: "0"}},
}

func runBuild(projectPath, platform, outDir string) error {
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	configPath := filepath.Join(projectPath, "app-config.json")
	cfg, err := buildcfg.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	// Step 1: Generate build configs (icons, winres, plist, desktop)
	buildDir := filepath.Join(projectPath, ".build")
	fmt.Println("generating build configs...")
	if err := buildcfg.GenerateIcons(cfg, projectPath, buildDir); err != nil {
		return fmt.Errorf("generating icons: %w", err)
	}
	fmt.Println("  ✓ icons")

	if err := buildcfg.GenerateWinres(cfg, projectPath, buildDir); err != nil {
		return fmt.Errorf("generating winres: %w", err)
	}
	fmt.Println("  ✓ winres")

	if err := buildcfg.GenerateDarwinPlist(cfg, buildDir); err != nil {
		return fmt.Errorf("generating plist: %w", err)
	}
	fmt.Println("  ✓ Info.plist")

	if err := buildcfg.GenerateLinuxDesktop(cfg, buildDir); err != nil {
		return fmt.Errorf("generating desktop: %w", err)
	}
	fmt.Println("  ✓ desktop entry")

	// Determine platforms to build
	if platform == "" {
		platform = runtime.GOOS
	}

	var platforms []string
	if platform == "all" {
		platforms = []string{"darwin", "windows", "linux"}
	} else {
		if _, ok := buildTargets[platform]; !ok {
			return fmt.Errorf("unsupported platform: %s", platform)
		}
		platforms = []string{platform}
	}

	distDir := filepath.Join(projectPath, outDir)
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return err
	}

	appName := cfg.App.Name
	version := cfg.App.Version

	// Load macOS signing credentials if available
	var creds *darwinCreds
	for _, p := range platforms {
		if p == "darwin" {
			creds = loadDarwinCreds(projectPath)
			if creds != nil {
				fmt.Println("found macOS signing credentials in .env")
			}
			break
		}
	}

	for _, p := range platforms {
		// Step 2: For Windows, generate .syso via go-winres
		if p == "windows" {
			fmt.Println("running go-winres...")
			winresDir := filepath.Join(buildDir, "winres")
			cmd := exec.Command("go-winres", "make", "--in", filepath.Join(winresDir, "winres.json"), "--out", projectPath)
			cmd.Dir = projectPath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go-winres: %w", err)
			}
			fmt.Println("  ✓ .syso generated")
		}

		// Step 3: Build for each arch
		for _, t := range buildTargets[p] {
			binaryName := appName
			if t.goos == "windows" {
				binaryName += ".exe"
			}

			outputName := fmt.Sprintf("%s_%s_%s", appName, t.goos, t.goarch)
			outputPath := filepath.Join(distDir, outputName, binaryName)
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				return err
			}

			ldflags := "-s -w -X main.Mode=release"
			if t.goos == "windows" {
				ldflags += " -H windowsgui"
			}

			fmt.Printf("building %s/%s...\n", t.goos, t.goarch)
			cmd := exec.Command("go", "build", "-o", outputPath, "-ldflags", ldflags, ".")
			cmd.Dir = projectPath
			cmd.Env = append(os.Environ(),
				"GOOS="+t.goos,
				"GOARCH="+t.goarch,
				"CGO_ENABLED="+t.cgo,
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go build %s/%s: %w", t.goos, t.goarch, err)
			}

			// Verify the output is a real binary, not an archive from a non-main package
			if info, err := os.Stat(outputPath); err != nil || info.Size() == 0 {
				return fmt.Errorf("go build produced no output — is there a main package in %s?", projectPath)
			}
			if header, err := os.ReadFile(outputPath); err == nil && len(header) > 8 && string(header[:8]) == "!<arch>\n" {
				return fmt.Errorf("go build produced a library archive, not an executable — %s must contain a main package", projectPath)
			}

			fmt.Printf("  ✓ %s/%s\n", t.goos, t.goarch)

			// Step 4: macOS .app bundle
			if t.goos == "darwin" {
				if err := createDarwinApp(cfg, buildDir, distDir, outputPath, t.goarch, appName, version); err != nil {
					return fmt.Errorf("creating .app bundle: %w", err)
				}
				if creds != nil {
					appDir := filepath.Join(distDir, fmt.Sprintf("%s_%s_%s.app", appName, "darwin", t.goarch))
					if err := signDarwinApp(creds, appDir, appName, filepath.Join(buildDir, "entitlements.plist")); err != nil {
						return fmt.Errorf("signing .app: %w", err)
					}
					if err := notarizeDarwinApp(creds, appDir); err != nil {
						return fmt.Errorf("notarizing .app: %w", err)
					}
				}
			}
		}
	}

	fmt.Println("build complete!")
	return nil
}

func createDarwinApp(cfg *buildcfg.Config, buildDir, distDir, binaryPath, arch, appName, version string) error {
	appDir := filepath.Join(distDir, fmt.Sprintf("%s_%s_%s.app", appName, "darwin", arch))
	contentsDir := filepath.Join(appDir, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	resourcesDir := filepath.Join(contentsDir, "Resources")

	for _, d := range []string{macosDir, resourcesDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Copy binary
	binData, err := os.ReadFile(binaryPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(macosDir, appName), binData, 0755); err != nil {
		return err
	}

	// Generate Info.plist from template (replace version placeholders)
	plistTemplate, err := os.ReadFile(filepath.Join(buildDir, "Info.plist.template"))
	if err != nil {
		return fmt.Errorf("reading Info.plist.template: %w", err)
	}
	plist := strings.ReplaceAll(string(plistTemplate), "${VERSION}", version)
	plist = strings.ReplaceAll(plist, "${BUILD_NUMBER}", "1")
	if err := os.WriteFile(filepath.Join(contentsDir, "Info.plist"), []byte(plist), 0644); err != nil {
		return err
	}

	// Copy icon
	icnsPath := filepath.Join(buildDir, "icons", "AppIcon.icns")
	if data, err := os.ReadFile(icnsPath); err == nil {
		os.WriteFile(filepath.Join(resourcesDir, "AppIcon.icns"), data, 0644)
	}

	fmt.Printf("  ✓ %s\n", filepath.Base(appDir))
	return nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCmdOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return out.String(), err
}

func signDarwinApp(creds *darwinCreds, appDir, appName, entitlementsPath string) error {
	identity := fmt.Sprintf("Developer ID Application: %s (%s)", creds.AppleID, creds.TeamID)

	// Setup temporary keychain if P12 is provided
	keychainPath := "/tmp/velo-build.keychain"
	keychainPass := "velo_build_temp"
	if creds.P12File != "" {
		fmt.Println("setting up signing keychain...")
		exec.Command("security", "delete-keychain", keychainPath).Run()
		if err := runCmd("security", "create-keychain", "-p", keychainPass, keychainPath); err != nil {
			return fmt.Errorf("creating keychain: %w", err)
		}
		runCmd("security", "default-keychain", "-s", keychainPath)
		runCmd("security", "unlock-keychain", "-p", keychainPass, keychainPath)
		runCmd("security", "import", creds.P12File, "-P", creds.P12Password, "-k", keychainPath, "-T", "/usr/bin/codesign")
		runCmd("security", "set-key-partition-list", "-S", "apple-tool:,apple:", "-s", "-k", keychainPass, keychainPath)
		runCmd("security", "list-keychains", "-s", keychainPath, "/Library/Keychains/System.keychain")
	}

	fmt.Println("signing .app...")

	// Sign the main binary with entitlements + hardened runtime
	binaryPath := filepath.Join(appDir, "Contents", "MacOS", appName)
	if err := runCmd("codesign", "--force", "--sign", identity, "--options", "runtime", "--timestamp", "--entitlements", entitlementsPath, binaryPath); err != nil {
		return fmt.Errorf("signing binary: %w", err)
	}

	// Sign the .app bundle
	if err := runCmd("codesign", "--force", "--sign", identity, "--options", "runtime", "--timestamp", "--entitlements", entitlementsPath, appDir); err != nil {
		return fmt.Errorf("signing app bundle: %w", err)
	}

	fmt.Println("  ✓ signed")

	// Cleanup keychain
	if creds.P12File != "" {
		runCmd("security", "list-keychains", "-s", "/Library/Keychains/System.keychain")
		runCmd("security", "delete-keychain", keychainPath)
	}

	return nil
}

func notarizeDarwinApp(creds *darwinCreds, appDir string) error {
	if creds.P8File == "" || creds.APIKeyID == "" || creds.APIKeyIssuerID == "" {
		fmt.Println("  skipping notarization (P8_FILE, API_KEY_ID, or API_KEY_ISSUER_ID not set)")
		return nil
	}

	// Create a zip for notarization
	zipPath := appDir + ".zip"
	if err := runCmd("ditto", "-c", "-k", "--keepParent", appDir, zipPath); err != nil {
		return fmt.Errorf("creating zip for notarization: %w", err)
	}
	defer os.Remove(zipPath)

	fmt.Println("submitting for notarization...")
	output, err := runCmdOutput("xcrun", "notarytool", "submit", zipPath,
		"-k", creds.P8File,
		"-d", creds.APIKeyID,
		"-i", creds.APIKeyIssuerID,
		"--wait")
	if err != nil {
		return fmt.Errorf("notarytool submit: %w", err)
	}
	fmt.Print(output)

	if !strings.Contains(output, "status: Accepted") {
		return fmt.Errorf("notarization was not accepted")
	}
	fmt.Println("  ✓ notarized")

	// Staple
	fmt.Println("stapling...")
	if err := runCmd("xcrun", "stapler", "staple", appDir); err != nil {
		return fmt.Errorf("stapling: %w", err)
	}
	fmt.Println("  ✓ stapled")

	return nil
}
