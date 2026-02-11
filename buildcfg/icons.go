package buildcfg

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nfnt/resize"
)

func GenerateIcons(cfg *Config, baseDir, outDir string) error {
	srcIcon := cfg.App.Icon
	if srcIcon == "" {
		srcIcon = "build/icon.png"
	}
	if !filepath.IsAbs(srcIcon) {
		srcIcon = filepath.Join(baseDir, srcIcon)
	}

	f, err := os.Open(srcIcon)
	if err != nil {
		return fmt.Errorf("opening source icon: %w", err)
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("decoding source icon: %w", err)
	}

	iconsDir := filepath.Join(outDir, "icons")
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		return fmt.Errorf("creating icons dir: %w", err)
	}

	// Resize source if > 1024
	bounds := src.Bounds()
	if bounds.Dx() > 1024 || bounds.Dy() > 1024 {
		src = resize.Resize(1024, 1024, src, resize.Lanczos3)
	}

	// Generate PNG icons at various sizes
	sizes := []int{16, 32, 48, 64, 128, 256, 512, 1024}
	resized := make(map[int]image.Image)
	for _, size := range sizes {
		img := resize.Resize(uint(size), uint(size), src, resize.Lanczos3)
		resized[size] = img
		if err := savePNG(filepath.Join(iconsDir, fmt.Sprintf("icon_%d.png", size)), img); err != nil {
			return err
		}
	}

	// Windows ICO (16, 32, 48, 64, 128, 256)
	icoSizes := []int{16, 32, 48, 64, 128, 256}
	var icoImages []image.Image
	for _, s := range icoSizes {
		icoImages = append(icoImages, resized[s])
	}
	if err := saveICO(filepath.Join(iconsDir, "icon.ico"), icoImages); err != nil {
		return err
	}

	// macOS iconset + icns (only works on macOS with iconutil)
	if err := generateIconset(src, resized, iconsDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: icns generation skipped: %v\n", err)
	}

	// Copy to build/ for compatibility
	buildDir := filepath.Join(baseDir, "build")
	os.MkdirAll(buildDir, 0755)
	copies := map[string]string{
		"icon.ico":      "icon.ico",
		"icon.ico#2":    "appicon.ico",
		"icon_256.png":  "icon_256.png",
		"icon_256.png#2": "appicon.png",
		"icon_16.png":   "icon_16.png",
		"icon_16.png#2": "icon16.png",
	}
	for src, dst := range copies {
		actual := src
		if len(actual) > 2 && actual[len(actual)-2] == '#' {
			actual = actual[:len(actual)-2]
		}
		srcPath := filepath.Join(iconsDir, actual)
		if data, err := os.ReadFile(srcPath); err == nil {
			os.WriteFile(filepath.Join(buildDir, dst), data, 0644)
		}
	}
	if data, err := os.ReadFile(filepath.Join(iconsDir, "AppIcon.icns")); err == nil {
		os.WriteFile(filepath.Join(buildDir, "AppIcon.icns"), data, 0644)
	}

	return nil
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()
	return png.Encode(f, img)
}

func generateIconset(src image.Image, resized map[int]image.Image, outDir string) error {
	iconsetDir := filepath.Join(outDir, "AppIcon.iconset")
	if err := os.MkdirAll(iconsetDir, 0755); err != nil {
		return err
	}

	iconsetSizes := []int{16, 32, 64, 128, 256, 512}
	for _, size := range iconsetSizes {
		if err := savePNG(filepath.Join(iconsetDir, fmt.Sprintf("icon_%dx%d.png", size, size)), resized[size]); err != nil {
			return err
		}
		// @2x version
		size2x := size * 2
		img2x, ok := resized[size2x]
		if !ok {
			img2x = resize.Resize(uint(size2x), uint(size2x), src, resize.Lanczos3)
		}
		if err := savePNG(filepath.Join(iconsetDir, fmt.Sprintf("icon_%dx%d@2x.png", size, size)), img2x); err != nil {
			return err
		}
	}
	// 1024 (no @2x)
	if err := savePNG(filepath.Join(iconsetDir, "icon_512x512@2x.png"), resized[1024]); err != nil {
		return err
	}

	// Try iconutil (macOS only)
	icnsPath := filepath.Join(outDir, "AppIcon.icns")
	cmd := exec.Command("iconutil", "-c", "icns", iconsetDir, "-o", icnsPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("iconutil not available (macOS only): %w", err)
	}
	return nil
}

// saveICO writes a multi-resolution ICO file
func saveICO(path string, images []image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating ico: %w", err)
	}
	defer f.Close()

	count := len(images)

	// Encode each image as PNG first
	type icoEntry struct {
		data   []byte
		width  int
		height int
	}
	var entries []icoEntry
	for _, img := range images {
		tmpF, err := os.CreateTemp("", "ico-*.png")
		if err != nil {
			return err
		}
		if err := png.Encode(tmpF, img); err != nil {
			tmpF.Close()
			os.Remove(tmpF.Name())
			return err
		}
		tmpF.Close()
		data, err := os.ReadFile(tmpF.Name())
		os.Remove(tmpF.Name())
		if err != nil {
			return err
		}
		b := img.Bounds()
		entries = append(entries, icoEntry{data: data, width: b.Dx(), height: b.Dy()})
	}

	// ICO header: reserved(2) + type(2) + count(2)
	binary.Write(f, binary.LittleEndian, uint16(0))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(count))

	// Calculate data offset: header(6) + entries(16 each)
	offset := 6 + count*16

	// Write directory entries
	for _, e := range entries {
		w := uint8(e.width)
		h := uint8(e.height)
		if e.width >= 256 {
			w = 0
		}
		if e.height >= 256 {
			h = 0
		}
		f.Write([]byte{w, h, 0, 0})                              // width, height, colors, reserved
		binary.Write(f, binary.LittleEndian, uint16(1))           // color planes
		binary.Write(f, binary.LittleEndian, uint16(32))          // bits per pixel
		binary.Write(f, binary.LittleEndian, uint32(len(e.data))) // data size
		binary.Write(f, binary.LittleEndian, uint32(offset))      // data offset
		offset += len(e.data)
	}

	// Write image data
	for _, e := range entries {
		f.Write(e.data)
	}

	return nil
}
