package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestPreprocessImageScalesToCHWTensor(t *testing.T) {
	path := writeSolidPNG(t, t.TempDir(), 128, 64, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

	result, err := preprocessImageFile(path, 64)
	if err != nil {
		t.Fatalf("preprocessImageFile() error = %v", err)
	}
	if result.Summary.Format != "png" || result.Summary.OriginalWidth != 128 || result.Summary.OriginalHeight != 64 {
		t.Fatalf("summary = %#v, want png 128x64", result.Summary)
	}
	if result.Summary.InputWidth != 64 || result.Summary.InputHeight != 32 {
		t.Fatalf("input size = %dx%d, want 64x32", result.Summary.InputWidth, result.Summary.InputHeight)
	}
	if result.Summary.ScaleX != 0.5 || result.Summary.ScaleY != 0.5 {
		t.Fatalf("scale = %f/%f, want 0.5/0.5", result.Summary.ScaleX, result.Summary.ScaleY)
	}
	wantShape := []int64{1, 3, 32, 64}
	if !sameInt64Slice(result.Summary.Shape, wantShape) {
		t.Fatalf("shape = %#v, want %#v", result.Summary.Shape, wantShape)
	}
	if len(result.Tensor) != 3*64*32 {
		t.Fatalf("tensor len = %d, want %d", len(result.Tensor), 3*64*32)
	}
	assertClose(t, result.Tensor[0], normalizedChannelForTest(255, ocrImageMean[0], ocrImageStd[0]), 0.0001)
	assertClose(t, result.Tensor[64*32], normalizedChannelForTest(0, ocrImageMean[1], ocrImageStd[1]), 0.0001)
	assertClose(t, result.Tensor[2*64*32], normalizedChannelForTest(0, ocrImageMean[2], ocrImageStd[2]), 0.0001)
}

func TestPreprocessImageCompositesTransparentPixelsOnWhite(t *testing.T) {
	path := writeSolidPNG(t, t.TempDir(), 32, 32, color.NRGBA{R: 0, G: 0, B: 0, A: 0})

	result, err := preprocessImageFile(path, 32)
	if err != nil {
		t.Fatalf("preprocessImageFile() error = %v", err)
	}
	if result.Summary.InputWidth != 32 || result.Summary.InputHeight != 32 {
		t.Fatalf("input size = %dx%d, want 32x32", result.Summary.InputWidth, result.Summary.InputHeight)
	}
	planeSize := 32 * 32
	assertClose(t, result.Tensor[0], normalizedChannelForTest(255, ocrImageMean[0], ocrImageStd[0]), 0.0001)
	assertClose(t, result.Tensor[planeSize], normalizedChannelForTest(255, ocrImageMean[1], ocrImageStd[1]), 0.0001)
	assertClose(t, result.Tensor[2*planeSize], normalizedChannelForTest(255, ocrImageMean[2], ocrImageStd[2]), 0.0001)
}

func TestPreprocessImageSupportsJPEGAndWebP(t *testing.T) {
	root := t.TempDir()
	jpegPath := filepath.Join(root, "sample.jpg")
	jpegImage := image.NewRGBA(image.Rect(0, 0, 40, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 40; x++ {
			jpegImage.SetRGBA(x, y, color.RGBA{R: 32, G: 64, B: 96, A: 255})
		}
	}
	jpegFile, err := os.Create(jpegPath)
	if err != nil {
		t.Fatalf("Create(jpeg) error = %v", err)
	}
	if err := jpeg.Encode(jpegFile, jpegImage, &jpeg.Options{Quality: 90}); err != nil {
		_ = jpegFile.Close()
		t.Fatalf("jpeg.Encode() error = %v", err)
	}
	if err := jpegFile.Close(); err != nil {
		t.Fatalf("jpeg close error = %v", err)
	}

	jpegResult, err := preprocessImageFile(jpegPath, 64)
	if err != nil {
		t.Fatalf("preprocessImageFile(jpeg) error = %v", err)
	}
	if jpegResult.Summary.Format != "jpeg" || len(jpegResult.Tensor) != 3*jpegResult.Summary.InputWidth*jpegResult.Summary.InputHeight {
		t.Fatalf("jpeg summary/tensor = %#v len=%d", jpegResult.Summary, len(jpegResult.Tensor))
	}

	webpPath := filepath.Join(root, "sample.webp")
	webpData, err := base64.StdEncoding.DecodeString(tinyWebPBase64)
	if err != nil {
		t.Fatalf("DecodeString(webp) error = %v", err)
	}
	if err := os.WriteFile(webpPath, webpData, 0o644); err != nil {
		t.Fatalf("WriteFile(webp) error = %v", err)
	}
	webpResult, err := preprocessImageFile(webpPath, 64)
	if err != nil {
		t.Fatalf("preprocessImageFile(webp) error = %v", err)
	}
	if webpResult.Summary.Format != "webp" || webpResult.Summary.OriginalWidth <= 0 || webpResult.Summary.OriginalHeight <= 0 || len(webpResult.Tensor) == 0 {
		t.Fatalf("webp summary/tensor = %#v len=%d", webpResult.Summary, len(webpResult.Tensor))
	}
}

func TestPreprocessImageRejectsMissingPath(t *testing.T) {
	_, err := preprocessImageFile(filepath.Join(t.TempDir(), "missing.png"), 64)
	if err == nil {
		t.Fatal("preprocessImageFile(missing) succeeded, want error")
	}
}

func writeSolidPNG(t *testing.T, root string, width int, height int, fill color.NRGBA) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", root, err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	path := filepath.Join(root, "sample.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	return path
}

func sameInt64Slice(a []int64, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func normalizedChannelForTest(value uint8, mean float32, std float32) float32 {
	return (float32(value)/255.0 - mean) / std
}

func assertClose(t *testing.T, got float32, want float32, tolerance float64) {
	t.Helper()
	if math.Abs(float64(got-want)) > tolerance {
		t.Fatalf("value = %f, want %f", got, want)
	}
}

const tinyWebPBase64 = "UklGRrIBAABXRUJQVlA4TKUBAAAvSsAYAA8w//M///MfeJAkbXvaSG7m8Q3GfYSBJekwQztm/IcZlgwnmWImn2BK7aFmBtnVir6q//8VOkFE/xm4baTIu8c48ArEo6+B3zFKYln3pqClSCKX0begFTAXFOLXHSyF8cCNcZEG4OywuA4KVVfJCiArU7GAgJI8+lJP/OKMT/fBAjevg1cYB7YVkFuWga2lyPi5I0HFy5YTpWIHg0RZpkniRVW9odHAKOwosWuOGdxIyn2OvaCDvhg/we6TwadPBPbqBV58MsLmMJ8yZnOWk8SRz4N+QoyPL+MnamzMvcE1rHNEr91F9GKZPVUcS9w7PhhH36suB9qPeYb/oLk6cuTiJ0wOK3m5h1cKjW6EVZCYMK7dxcKCBdgP9HkKr9gkAO2P8GKZGWVdIAatQa+1IDpt6qyorVwdy01xdW8Jkfk6xjEXmVQQ+HQdFr6OKhIN34dXWq0+0qr6EJSCeeVLH9+gvGTLyqM65PQ44ihzlTXxQKjKbAvshXgir7Lil9w4L2bvMycmjQcqXaMCO6BlY28i+FOLzbfI1vEqxAhotocAAA=="
