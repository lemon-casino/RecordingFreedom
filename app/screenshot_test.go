package main

import (
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

func mustScreenshotDir(t *testing.T, service *RecordingFreedomService) string {
	t.Helper()
	dir, err := service.screenshotDir()
	if err != nil {
		t.Fatalf("screenshotDir() error = %v", err)
	}
	return dir
}
