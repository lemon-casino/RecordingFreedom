package iconbuilder

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
)

const (
	DefaultSizes       = "16,24,32,48,64,128,256,512,1024"
	DefaultAppIconSize = 1024
)

type Options struct {
	SourcePath        string
	Sizes             []int
	OutputDir         string
	AppIconPath       string
	AppIconSize       int
	FrontendIconPaths []string
	FrontendIconSize  int
}

type GeneratedIcon struct {
	Size int
	Path string
}

type Report struct {
	SourcePath        string
	OutputDir         string
	Generated         []GeneratedIcon
	AppIconPath       string
	FrontendIconPaths []string
}

func ParseSizes(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	sizes := make([]int, 0, len(parts))
	seen := make(map[int]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		size, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("parse icon size %q: %w", part, err)
		}
		if size < 8 || size > 4096 {
			return nil, fmt.Errorf("icon size %d is outside the supported range 8..4096", size)
		}
		if seen[size] {
			continue
		}
		seen[size] = true
		sizes = append(sizes, size)
	}
	if len(sizes) == 0 {
		return nil, fmt.Errorf("at least one icon size is required")
	}
	sort.Ints(sizes)
	return sizes, nil
}

func Run(ctx context.Context, opts Options) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	if strings.TrimSpace(opts.SourcePath) == "" {
		return Report{}, fmt.Errorf("source path is required")
	}
	if len(opts.Sizes) == 0 {
		return Report{}, fmt.Errorf("at least one generated icon size is required")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join("build", "icons")
	}
	if opts.AppIconPath == "" {
		opts.AppIconPath = filepath.Join("build", "appicon.png")
	}
	if opts.AppIconSize == 0 {
		opts.AppIconSize = DefaultAppIconSize
	}
	if opts.FrontendIconSize == 0 {
		opts.FrontendIconSize = 256
	}

	src, err := loadImage(opts.SourcePath)
	if err != nil {
		return Report{}, err
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return Report{}, fmt.Errorf("create output dir: %w", err)
	}

	report := Report{
		SourcePath: opts.SourcePath,
		OutputDir:  opts.OutputDir,
	}
	for _, size := range opts.Sizes {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		outPath := filepath.Join(opts.OutputDir, fmt.Sprintf("icon-%d.png", size))
		if err := writePNG(outPath, resizeContain(src, size)); err != nil {
			return Report{}, err
		}
		report.Generated = append(report.Generated, GeneratedIcon{Size: size, Path: outPath})
	}

	if opts.AppIconPath != "" {
		if err := writePNG(opts.AppIconPath, resizeContain(src, opts.AppIconSize)); err != nil {
			return Report{}, err
		}
		report.AppIconPath = opts.AppIconPath
	}
	for _, path := range opts.FrontendIconPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if err := writePNG(path, resizeContain(src, opts.FrontendIconSize)); err != nil {
			return Report{}, err
		}
		report.FrontendIconPaths = append(report.FrontendIconPaths, path)
	}

	return report, nil
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source icon: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode source icon: %w", err)
	}
	return img, nil
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()

	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(file, img); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func resizeContain(src image.Image, size int) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return dst
	}

	scale := float64(size) / float64(width)
	if hScale := float64(size) / float64(height); hScale < scale {
		scale = hScale
	}
	targetW := max(1, int(float64(width)*scale+0.5))
	targetH := max(1, int(float64(height)*scale+0.5))
	x0 := (size - targetW) / 2
	y0 := (size - targetH) / 2
	rect := image.Rect(x0, y0, x0+targetW, y0+targetH)
	xdraw.CatmullRom.Scale(dst, rect, src, bounds, xdraw.Over, nil)
	return dst
}
