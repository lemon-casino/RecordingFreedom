package iconbuilder

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseSizesSortsAndDeduplicates(t *testing.T) {
	got, err := ParseSizes("128, 16,32,16")
	if err != nil {
		t.Fatalf("ParseSizes returned error: %v", err)
	}
	want := []int{16, 32, 128}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseSizes = %v, want %v", got, want)
	}
}

func TestRunGeneratesRequestedPNGs(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	writeTestSource(t, source)

	report, err := Run(context.Background(), Options{
		SourcePath:        source,
		Sizes:             []int{16, 64},
		OutputDir:         filepath.Join(dir, "icons"),
		AppIconPath:       filepath.Join(dir, "build", "appicon.png"),
		AppIconSize:       128,
		FrontendIconPaths: []string{filepath.Join(dir, "frontend", "appicon.png")},
		FrontendIconSize:  32,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(report.Generated) != 2 {
		t.Fatalf("generated count = %d, want 2", len(report.Generated))
	}
	assertPNGSize(t, filepath.Join(dir, "icons", "icon-16.png"), 16)
	assertPNGSize(t, filepath.Join(dir, "icons", "icon-64.png"), 64)
	assertPNGSize(t, filepath.Join(dir, "build", "appicon.png"), 128)
	assertPNGSize(t, filepath.Join(dir, "frontend", "appicon.png"), 32)
}

func writeTestSource(t *testing.T, path string) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 10, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.NRGBA{R: 25, G: 120, B: 220, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode source: %v", err)
	}
}

func assertPNGSize(t *testing.T, path string, size int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if got := img.Bounds().Dx(); got != size {
		t.Fatalf("%s width = %d, want %d", path, got, size)
	}
	if got := img.Bounds().Dy(); got != size {
		t.Fatalf("%s height = %d, want %d", path, got, size)
	}
}
