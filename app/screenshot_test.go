package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
)

func TestScreenshotHistoryPersistsSortedUniqueItems(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())

	items := []ScreenshotItem{
		{ID: "older", Path: filepath.Join(mustScreenshotDir(t, service), "older.png"), CreatedAt: "2026-07-04T00:00:00Z", Width: 100, Height: 100},
		{ID: "newer", Path: filepath.Join(mustScreenshotDir(t, service), "newer.png"), CreatedAt: "2026-07-04T00:00:02Z", Width: 200, Height: 120, Mode: "region"},
		{ID: "older", Path: filepath.Join(mustScreenshotDir(t, service), "duplicate.png"), CreatedAt: "2026-07-04T00:00:03Z", Width: 10, Height: 10},
	}
	if err := service.saveScreenshotHistory(items); err != nil {
		t.Fatalf("saveScreenshotHistory() error = %v", err)
	}
	got, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("loadScreenshotHistory() len = %d, want 2", len(got))
	}
	if got[0].ID != "newer" || got[1].ID != "older" {
		t.Fatalf("history order = [%s, %s], want [newer, older]", got[0].ID, got[1].ID)
	}
	if got[1].Mode != "region" {
		t.Fatalf("default mode = %q, want region", got[1].Mode)
	}
}

func TestDeleteScreenshotItemRemovesHistoryAndFiles(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	dir := mustScreenshotDir(t, service)
	thumbDir := filepath.Join(dir, "thumbnails")
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	imagePath := filepath.Join(dir, "shot.png")
	thumbPath := filepath.Join(thumbDir, "shot.png")
	if err := os.WriteFile(imagePath, []byte("image"), 0o644); err != nil {
		t.Fatalf("WriteFile(image) error = %v", err)
	}
	if err := os.WriteFile(thumbPath, []byte("thumb"), 0o644); err != nil {
		t.Fatalf("WriteFile(thumb) error = %v", err)
	}
	if err := service.saveScreenshotHistory([]ScreenshotItem{
		{ID: "shot", Path: imagePath, ThumbnailPath: thumbPath, CreatedAt: "2026-07-04T00:00:02Z", Width: 200, Height: 120, Mode: "region"},
		{ID: "keep", Path: filepath.Join(dir, "keep.png"), CreatedAt: "2026-07-04T00:00:01Z", Width: 100, Height: 100, Mode: "region"},
	}); err != nil {
		t.Fatalf("saveScreenshotHistory() error = %v", err)
	}

	result, err := service.DeleteScreenshotItem("shot")
	if err != nil {
		t.Fatalf("DeleteScreenshotItem() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "keep" {
		t.Fatalf("remaining history = %#v, want keep only", result.Items)
	}
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatalf("deleted image stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Fatalf("deleted thumbnail stat error = %v, want not exist", err)
	}
}

func TestSaveScreenshotAnnotationCaptureWritesHistoryAndFiles(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	region := RegionRect{X: 10, Y: 20, Width: 80, Height: 60}
	service.screenshotAnnotation = ScreenshotWhiteboardContext{
		Available: true,
		Item: ScreenshotItem{
			ID:     "draft",
			Width:  80,
			Height: 60,
			Mode:   "region",
			Region: &region,
		},
		DataURL: testPNGDataURL(t, 80, 60),
	}

	result, err := service.SaveScreenshotAnnotationCapture(AnnotationCaptureRequest{
		SceneJSON:       `{"type":"excalidraw","elements":[],"appState":{},"files":{}}`,
		SnapshotDataURL: testPNGDataURL(t, 80, 60),
	})
	if err != nil {
		t.Fatalf("SaveScreenshotAnnotationCapture() error = %v", err)
	}
	if result.Item.ID == "" || result.Item.Path == "" || result.Item.ThumbnailPath == "" {
		t.Fatalf("saved screenshot item missing file paths: %#v", result.Item)
	}
	if result.Item.Mode != "region" || result.Item.Region == nil || *result.Item.Region != region {
		t.Fatalf("saved screenshot region = %#v, want %#v", result.Item.Region, region)
	}
	if _, err := os.Stat(result.Item.Path); err != nil {
		t.Fatalf("saved image stat error = %v", err)
	}
	if _, err := os.Stat(result.Item.ThumbnailPath); err != nil {
		t.Fatalf("saved thumbnail stat error = %v", err)
	}
	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != result.Item.ID {
		t.Fatalf("history = %#v, want saved item only", history)
	}
}

func TestMapRegionSelectionToCaptureRectScalesOverlayToNativeBounds(t *testing.T) {
	session := RegionSelectionSession{
		Bounds:        RegionRect{X: 10, Y: 20, Width: 1000, Height: 500},
		CaptureBounds: &RegionRect{X: 100, Y: 200, Width: 2000, Height: 1000},
	}
	rect := mapRegionSelectionToCaptureRect(session, RegionRect{X: 250, Y: 100, Width: 300, Height: 200})
	if rect.Min.X != 600 || rect.Min.Y != 400 || rect.Dx() != 600 || rect.Dy() != 400 {
		t.Fatalf("mapped rect = %+v, want min=(600,400) size=600x400", rect)
	}
}

func testPNGDataURL(t *testing.T, width int, height int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 180, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return whiteboardPNGContentPrefix + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func mustScreenshotDir(t *testing.T, service *RecordingFreedomService) string {
	t.Helper()
	dir, err := service.screenshotDir()
	if err != nil {
		t.Fatalf("screenshotDir() error = %v", err)
	}
	return dir
}
