package buildcfg

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateIconsKeepsGeneratedFilesInOutputDirectory(t *testing.T) {
	projectDir := t.TempDir()
	assetsDir := filepath.Join(projectDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatal(err)
	}

	iconPath := filepath.Join(assetsDir, "appicon.png")
	f, err := os.Create(iconPath)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 32, G: 96, B: 192, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{App: AppSection{Icon: "assets/appicon.png"}}
	outDir := filepath.Join(projectDir, ".build")
	if err := GenerateIcons(cfg, projectDir, outDir); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"icon.ico", "icon_16.png", "icon_256.png"} {
		if _, err := os.Stat(filepath.Join(outDir, "icons", name)); err != nil {
			t.Errorf("expected generated icon %s: %v", name, err)
		}
	}

	if _, err := os.Stat(filepath.Join(projectDir, "build")); !os.IsNotExist(err) {
		t.Errorf("legacy build directory should not be created; stat error: %v", err)
	}
}
