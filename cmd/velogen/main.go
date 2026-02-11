package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ltaoo/velo/buildcfg"
)

func main() {
	configPath := flag.String("config", "app-config.json", "path to app-config.json")
	outDir := flag.String("out", ".build", "output directory")
	icons := flag.Bool("icons", false, "generate icons from source image")
	flag.Parse()

	cfg, err := buildcfg.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}

	baseDir := filepath.Dir(*configPath)
	if baseDir == "" || baseDir == "." {
		baseDir, _ = os.Getwd()
	} else if !filepath.IsAbs(baseDir) {
		wd, _ := os.Getwd()
		baseDir = filepath.Join(wd, baseDir)
	}

	if *icons {
		fmt.Println("generating icons...")
		if err := buildcfg.GenerateIcons(cfg, baseDir, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "icons: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("  ✓ icons")
	}

	fmt.Println("generating build configs...")

	if err := buildcfg.GenerateGoreleaser(cfg, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "goreleaser: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ .goreleaser.yaml")

	if err := buildcfg.GenerateWinres(cfg, baseDir, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "winres: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ winres/winres.json")

	if err := buildcfg.GenerateDarwinPlist(cfg, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "darwin: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Info.plist.template")
	fmt.Println("  ✓ entitlements.plist")

	if err := buildcfg.GenerateLinuxDesktop(cfg, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "linux: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ app.desktop.template")

	fmt.Println("done!")
}
