package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

func TestScreenshotHistoryPersistsSortedUniqueItems(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())

	items := []ScreenshotItem{
		{ID: "older", Path: filepath.Join(mustScreenshotDir(t, service), "older.png"), CreatedAt: "2026-07-04T00:00:00Z", Width: 100, Height: 100},
		{ID: "newer", Path: filepath.Join(mustScreenshotDir(t, service), "newer.png"), CreatedAt: "2026-07-04T00:00:02Z", Width: 200, Height: 120, Mode: "region", Pinned: true, Fixed: true},
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
	if got[0].Pinned {
		t.Fatalf("stale pinned history state was persisted")
	}
	if !got[0].Fixed {
		t.Fatalf("fixed history state was not preserved")
	}
}

func TestPatchScreenshotItemDoesNotPersistPinnedHistoryState(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	shot := ScreenshotItem{
		ID:        "shot",
		Path:      filepath.Join(mustScreenshotDir(t, service), "shot.png"),
		CreatedAt: "2026-07-04T00:00:00Z",
		Width:     100,
		Height:    100,
		Mode:      "region",
	}
	if err := service.saveScreenshotHistory([]ScreenshotItem{shot}); err != nil {
		t.Fatalf("saveScreenshotHistory() error = %v", err)
	}

	fixed := true
	result, err := service.PatchScreenshotItem(ScreenshotItemPatchRequest{ID: "shot", Fixed: &fixed})
	if err != nil {
		t.Fatalf("PatchScreenshotItem(fixed=true) error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("patched history len = %d, want 1", len(result.Items))
	}
	if result.Items[0].Pinned {
		t.Fatalf("fixed screenshot was also persisted as pinned")
	}
	if !result.Items[0].Fixed {
		t.Fatalf("fixed screenshot was not marked fixed")
	}

	pinned := false
	result, err = service.PatchScreenshotItem(ScreenshotItemPatchRequest{ID: "shot", Pinned: &pinned})
	if err != nil {
		t.Fatalf("PatchScreenshotItem(pinned=false) error = %v", err)
	}
	if result.Items[0].Pinned || result.Items[0].Fixed {
		t.Fatalf("cleared screenshot state = pinned %v fixed %v, want both false", result.Items[0].Pinned, result.Items[0].Fixed)
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

func TestSaveWhiteboardSnapshotWritesSceneAndScreenshotHistory(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())

	result, err := service.SaveWhiteboardSnapshot(WhiteboardSnapshotRequest{
		SceneJSON:       `{"type":"excalidraw","elements":[{"id":"a","type":"rectangle"}],"appState":{},"files":{}}`,
		SnapshotDataURL: testPNGDataURL(t, 320, 180),
	})
	if err != nil {
		t.Fatalf("SaveWhiteboardSnapshot() error = %v", err)
	}
	if !result.Scene.Available || result.Scene.ScenePath == "" {
		t.Fatalf("saved scene = %#v, want available scene", result.Scene)
	}
	if result.Item.ID == "" || result.Item.Mode != "whiteboard" || result.Item.Path == "" || result.Item.ThumbnailPath == "" {
		t.Fatalf("saved item = %#v, want whiteboard screenshot history item", result.Item)
	}
	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != result.Item.ID || history[0].Mode != "whiteboard" {
		t.Fatalf("history = %#v, want saved whiteboard item", history)
	}
}

func TestCaptureScreenshotRectUsesFocusedWindowProvider(t *testing.T) {
	originalProvider := focusedWindowScreenshotRectProvider
	focusedWindowScreenshotRectProvider = func() (image.Rectangle, bool) {
		return image.Rect(21, 31, 301, 241), true
	}
	defer func() {
		focusedWindowScreenshotRectProvider = originalProvider
	}()

	service := NewRecordingFreedomService()
	rect, region, err := service.captureScreenshotRect(ScreenshotCaptureRequest{Mode: "focused-window"})
	if err != nil {
		t.Fatalf("captureScreenshotRect(focused-window) error = %v", err)
	}
	if rect.Min.X != 21 || rect.Min.Y != 31 || rect.Dx() != 280 || rect.Dy() != 210 {
		t.Fatalf("focused rect = %v, want (21,31) 280x210", rect)
	}
	if region == nil || region.X != 21 || region.Y != 31 || region.Width != 280 || region.Height != 210 {
		t.Fatalf("focused region = %#v, want provider rectangle", region)
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

func TestMapRegionPointToCapturePointScalesOverlayToNativeBounds(t *testing.T) {
	session := RegionSelectionSession{
		Bounds:        RegionRect{X: 10, Y: 20, Width: 1000, Height: 500},
		CaptureBounds: &RegionRect{X: 100, Y: 200, Width: 2000, Height: 1000},
	}
	got := mapRegionPointToCapturePoint(session, image.Point{X: 250, Y: 125})
	if got.X != 600 || got.Y != 450 {
		t.Fatalf("mapped point = %v, want (600,450)", got)
	}
}

func TestRegionCoordinateMappingUsesPerDisplayScale(t *testing.T) {
	session := RegionSelectionSession{
		Bounds:        RegionRect{X: 0, Y: 0, Width: 3200, Height: 1080},
		CaptureBounds: &RegionRect{X: 0, Y: 0, Width: 4480, Height: 1440},
		DisplayBounds: []RegionDisplayBounds{
			{
				ID:            "primary",
				Bounds:        RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
				CaptureBounds: RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
				ScaleFactor:   1,
			},
			{
				ID:            "scaled",
				Bounds:        RegionRect{X: 1920, Y: 0, Width: 1280, Height: 720},
				CaptureBounds: RegionRect{X: 1920, Y: 0, Width: 2560, Height: 1440},
				ScaleFactor:   2,
			},
		},
	}

	point := mapRegionPointToCapturePoint(session, image.Point{X: 2560, Y: 360})
	if point.X != 3200 || point.Y != 720 {
		t.Fatalf("mapped point = %v, want second-display physical point (3200,720)", point)
	}

	rect := mapRegionSelectionToCaptureRect(session, RegionRect{X: 2240, Y: 180, Width: 320, Height: 180})
	if rect.Min.X != 2560 || rect.Min.Y != 360 || rect.Dx() != 640 || rect.Dy() != 360 {
		t.Fatalf("mapped rect = %v, want second-display physical rect (2560,360) 640x360", rect)
	}

	relative := mapCaptureRectToRegionSelection(session, image.Rect(2560, 360, 3200, 720))
	if relative.X != 2240 || relative.Y != 180 || relative.Width != 320 || relative.Height != 180 {
		t.Fatalf("mapped relative rect = %#v, want second-display DIP rect 2240,180 320x180", relative)
	}
}

func TestRegionCoordinateMappingUsesRightDisplayAtSeam(t *testing.T) {
	session := RegionSelectionSession{
		Bounds: RegionRect{X: 0, Y: 0, Width: 3200, Height: 1080},
		DisplayBounds: []RegionDisplayBounds{
			{
				ID:            "left",
				Bounds:        RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
				CaptureBounds: RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
			},
			{
				ID:            "right",
				Bounds:        RegionRect{X: 1920, Y: 0, Width: 1280, Height: 720},
				CaptureBounds: RegionRect{X: 1920, Y: 0, Width: 2560, Height: 1440},
			},
		},
	}

	point := mapRegionPointToCapturePoint(session, image.Point{X: 1920, Y: 100})
	if point.X != 1920 || point.Y != 200 {
		t.Fatalf("seam point = %v, want right display mapping (1920,200)", point)
	}

	display, ok := regionDisplayForAbsolutePoint(session.DisplayBounds, image.Point{X: 1920, Y: 100})
	if !ok || display.ID != "right" {
		t.Fatalf("seam display = %#v ok=%v, want right display", display, ok)
	}
}

func TestRegionCoordinateMappingSupportsNegativeOriginDisplays(t *testing.T) {
	session := RegionSelectionSession{
		Bounds: RegionRect{X: -1600, Y: -120, Width: 3520, Height: 1200},
		DisplayBounds: []RegionDisplayBounds{
			{
				ID:            "left-retina",
				Bounds:        RegionRect{X: -1600, Y: -120, Width: 800, Height: 600},
				CaptureBounds: RegionRect{X: -3200, Y: -240, Width: 1600, Height: 1200},
				ScaleFactor:   2,
			},
			{
				ID:            "primary",
				Bounds:        RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
				CaptureBounds: RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
				ScaleFactor:   1,
			},
		},
	}

	point := mapRegionPointToCapturePoint(session, image.Point{X: 100, Y: 220})
	if point != (image.Point{X: -3000, Y: 200}) {
		t.Fatalf("negative-origin point = %v, want (-3000,200)", point)
	}

	rect := mapRegionSelectionToCaptureRect(session, RegionRect{X: 120, Y: 240, Width: 200, Height: 120})
	if rect != image.Rect(-2960, 240, -2560, 480) {
		t.Fatalf("negative-origin rect = %v, want (-2960,240)-(-2560,480)", rect)
	}

	relative := mapCaptureRectToRegionSelection(session, image.Rect(-2960, 240, -2560, 480))
	if relative != (RegionRect{X: 120, Y: 240, Width: 200, Height: 120}) {
		t.Fatalf("negative-origin reverse rect = %#v, want 120,240 200x120", relative)
	}
}

func TestRegionCoordinateMappingAcrossDisplaysUsesDisplayEndpoints(t *testing.T) {
	session := RegionSelectionSession{
		Bounds: RegionRect{X: 0, Y: 0, Width: 3200, Height: 1080},
		DisplayBounds: []RegionDisplayBounds{
			{
				ID:            "left",
				Bounds:        RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
				CaptureBounds: RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
			},
			{
				ID:            "right-retina",
				Bounds:        RegionRect{X: 1920, Y: 0, Width: 1280, Height: 720},
				CaptureBounds: RegionRect{X: 1920, Y: 0, Width: 2560, Height: 1440},
			},
		},
	}

	got := mapRegionSelectionToCaptureRect(session, RegionRect{X: 1840, Y: 100, Width: 240, Height: 120})
	if got != image.Rect(1840, 100, 2240, 440) {
		t.Fatalf("cross-display rect = %v, want endpoint-mapped rect (1840,100)-(2240,440)", got)
	}
}

func TestRegionAssistSnapshotBoundsUsesDisplayCaptureUnion(t *testing.T) {
	session := RegionSelectionSession{
		Bounds: RegionRect{X: 0, Y: 0, Width: 3200, Height: 1080},
		DisplayBounds: []RegionDisplayBounds{
			{
				ID:            "left",
				Bounds:        RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
				CaptureBounds: RegionRect{X: 0, Y: 0, Width: 1920, Height: 1080},
			},
			{
				ID:            "right",
				Bounds:        RegionRect{X: 1920, Y: 0, Width: 1280, Height: 720},
				CaptureBounds: RegionRect{X: 1920, Y: 0, Width: 2560, Height: 1440},
			},
		},
	}
	got := regionAssistSnapshotBounds(session)
	want := image.Rect(0, 0, 4480, 1440)
	if got != want {
		t.Fatalf("snapshot bounds = %v, want capture union %v", got, want)
	}
}

func TestRegionAssistImageCropsCachedCleanSnapshot(t *testing.T) {
	service := &RecordingFreedomService{}
	cached := image.NewRGBA(image.Rect(0, 0, 400, 220))
	wantColor := color.RGBA{R: 12, G: 34, B: 210, A: 255}
	cached.Set(15, 25, wantColor)
	session := RegionSelectionSession{ID: "snapshot-crop"}
	service.regionSnapshotCache = regionAssistSnapshotCache{
		SessionID: session.ID,
		Bounds:    image.Rect(100, 200, 500, 420),
		Image:     cached,
	}

	got, ok := service.regionAssistImage(session, image.Rect(110, 220, 130, 240))
	if !ok {
		t.Fatal("regionAssistImage() ok = false, want cached crop")
	}
	if got.Bounds() != image.Rect(0, 0, 20, 20) {
		t.Fatalf("cropped bounds = %v, want 20x20 zero-origin", got.Bounds())
	}
	if got.At(5, 5) != wantColor {
		t.Fatalf("cropped pixel = %v, want %v", got.At(5, 5), wantColor)
	}
}

func TestMapAbsoluteRectToRegionSelectionUsesOverlayOrigin(t *testing.T) {
	session := RegionSelectionSession{
		Bounds: RegionRect{X: -1280, Y: 120, Width: 3200, Height: 1080},
	}
	got := mapAbsoluteRectToRegionSelection(session, image.Rect(-1200, 180, -980, 340))
	if got.X != 80 || got.Y != 60 || got.Width != 220 || got.Height != 160 {
		t.Fatalf("relative rect = %#v, want 80,60 220x160", got)
	}
}

func TestRegionCandidateAtLevelClampsToAvailableCandidates(t *testing.T) {
	candidates := []RegionSmartCandidate{
		{ID: "child", Bounds: RegionRect{X: 10, Y: 10, Width: 80, Height: 80}},
		{ID: "parent", Bounds: RegionRect{X: 0, Y: 0, Width: 200, Height: 200}},
	}
	if got := regionCandidateAtLevel(candidates, -1); got == nil || got.ID != "child" {
		t.Fatalf("negative level candidate = %#v, want child", got)
	}
	if got := regionCandidateAtLevel(candidates, 1); got == nil || got.ID != "parent" {
		t.Fatalf("level 1 candidate = %#v, want parent", got)
	}
	if got := regionCandidateAtLevel(candidates, 12); got == nil || got.ID != "parent" {
		t.Fatalf("overflow level candidate = %#v, want parent", got)
	}
}

func TestRegionWindowCandidatesPreserveTopmostSourceOrder(t *testing.T) {
	session := RegionSelectionSession{
		Bounds:        RegionRect{X: 100, Y: 80, Width: 800, Height: 520},
		MinimumWidth:  24,
		MinimumHeight: 24,
	}
	point := image.Point{X: 180, Y: 150}
	sources := []devices.CaptureSource{
		{
			ID:     "window:top",
			Type:   devices.SourceWindow,
			Name:   "Top window",
			X:      240,
			Y:      190,
			Width:  260,
			Height: 180,
		},
		{
			ID:     "window:behind",
			Type:   devices.SourceWindow,
			Name:   "Behind window",
			X:      220,
			Y:      180,
			Width:  360,
			Height: 260,
		},
		{
			ID:     "window:outside",
			Type:   devices.SourceWindow,
			Name:   "Outside window",
			X:      20,
			Y:      20,
			Width:  120,
			Height: 80,
		},
	}

	got := regionWindowCandidatesFromCaptureSources(sources, session, point)
	if len(got) != 2 {
		t.Fatalf("window candidates = %d, want 2: %#v", len(got), got)
	}
	if got[0].ID != "window:top" || got[1].ID != "window:behind" {
		t.Fatalf("window candidate order = [%s,%s], want topmost source order", got[0].ID, got[1].ID)
	}
	if got[0].Bounds != (RegionRect{X: 140, Y: 110, Width: 260, Height: 180}) {
		t.Fatalf("top candidate bounds = %#v, want overlay-relative 140,110 260x180", got[0].Bounds)
	}
}

func TestAssistRegionSelectionReportsSelectionSource(t *testing.T) {
	service := &RecordingFreedomService{}
	service.regionSession = &RegionSelectionSession{
		ID:            "assist-selection-source",
		Bounds:        RegionRect{X: 0, Y: 0, Width: 400, Height: 300},
		MinimumWidth:  minRegionWidth,
		MinimumHeight: minRegionHeight,
		Purpose:       regionSelectionPurposeCapture,
		Candidates: []RegionSmartCandidate{
			{
				ID:     "window:main",
				Kind:   regionSmartKindWindow,
				Label:  "Main window",
				Bounds: RegionRect{X: 40, Y: 50, Width: 180, Height: 140},
				Score:  0.8,
			},
		},
	}

	result, err := service.AssistRegionSelection(RegionAssistRequest{
		SessionID: "assist-selection-source",
		Purpose:   regionSelectionPurposeCapture,
		Selection: &RegionRect{X: 44, Y: 52, Width: 172, Height: 136},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != "selection" {
		t.Fatalf("assist source = %q, want selection", result.Source)
	}
	if result.Best == nil || result.Best.ID != "window:main" {
		t.Fatalf("best candidate = %#v, want window:main", result.Best)
	}
}

func TestRegionCandidatePointContainmentUsesHalfOpenEdges(t *testing.T) {
	rect := RegionRect{X: 10, Y: 20, Width: 100, Height: 80}
	inside := []image.Point{
		{X: 10, Y: 20},
		{X: 109, Y: 20},
		{X: 10, Y: 99},
		{X: 109, Y: 99},
	}
	for _, point := range inside {
		if !regionRectContainsPoint(rect, point) {
			t.Fatalf("rect %#v should contain %v", rect, point)
		}
	}
	outside := []image.Point{
		{X: 110, Y: 20},
		{X: 10, Y: 100},
		{X: 110, Y: 100},
		{X: 9, Y: 20},
		{X: 10, Y: 19},
	}
	for _, point := range outside {
		if regionRectContainsPoint(rect, point) {
			t.Fatalf("rect %#v should not contain boundary/outside point %v", rect, point)
		}
	}
}

func TestSafeRegionElementLabelSuppressesSensitiveLongText(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		fallback string
		want     string
	}{
		{name: "short", raw: "Save recording", fallback: "Button", want: "Save recording"},
		{name: "empty", raw: " \n\t ", fallback: "Group", want: "Group"},
		{name: "json", raw: `{"accessToken":"secret"}`, fallback: "Document", want: "Document"},
		{name: "token", raw: "Bearer token eyJhbGciOi...", fallback: "Text", want: "Text"},
		{name: "password", raw: "password reset value", fallback: "Edit", want: "Edit"},
		{name: "long", raw: strings.Repeat("a", 81), fallback: "Pane", want: "Pane"},
		{name: "fallback", raw: "", fallback: "", want: "UI element"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safeRegionElementLabel(tc.raw, tc.fallback); got != tc.want {
				t.Fatalf("safeRegionElementLabel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRegionElementCandidateCacheKeepsSmallestContainingElementFirst(t *testing.T) {
	service := &RecordingFreedomService{}
	session := RegionSelectionSession{
		ID:            "element-cache",
		Bounds:        RegionRect{X: 0, Y: 0, Width: 800, Height: 600},
		MinimumWidth:  minRegionWidth,
		MinimumHeight: minRegionHeight,
	}
	service.rememberRegionElementCandidates(session.ID, []RegionSmartCandidate{
		{
			ID:       "element:parent",
			Kind:     regionSmartKindElement,
			Label:    "Parent",
			SourceID: "content:window",
			Bounds:   RegionRect{X: 80, Y: 70, Width: 500, Height: 420},
			Score:    0.7,
		},
		{
			ID:       "element:child",
			Kind:     regionSmartKindElement,
			Label:    "Child",
			SourceID: "raw:point",
			Bounds:   RegionRect{X: 140, Y: 120, Width: 180, Height: 120},
			Score:    0.9,
		},
	}, session)

	candidates := service.regionElementCacheCandidatesAtPoint(session.ID, image.Point{X: 170, Y: 150})
	if len(candidates) != 2 {
		t.Fatalf("cache candidates = %d, want 2: %#v", len(candidates), candidates)
	}
	if candidates[0].ID != "element:child" {
		t.Fatalf("first cache candidate = %q, want child", candidates[0].ID)
	}
	if candidates[1].ID != "element:parent" {
		t.Fatalf("second cache candidate = %q, want parent", candidates[1].ID)
	}

	outsideChild := service.regionElementCacheCandidatesAtPoint(session.ID, image.Point{X: 500, Y: 450})
	if len(outsideChild) != 1 || outsideChild[0].ID != "element:parent" {
		t.Fatalf("outside child cache candidates = %#v, want parent only", outsideChild)
	}

	service.resetRegionElementCache("next-session")
	if stale := service.regionElementCacheCandidatesAtPoint(session.ID, image.Point{X: 170, Y: 150}); len(stale) != 0 {
		t.Fatalf("old session cache candidates after reset = %#v, want none", stale)
	}
	service.rememberRegionElementCandidates("next-session", []RegionSmartCandidate{{
		ID:     "element:next",
		Kind:   regionSmartKindElement,
		Bounds: RegionRect{X: 10, Y: 10, Width: 120, Height: 90},
	}}, session)
	service.clearRegionElementCache("other-session")
	if kept := service.regionElementCacheCandidatesAtPoint("next-session", image.Point{X: 20, Y: 20}); len(kept) != 1 {
		t.Fatalf("cache after clearing other session = %#v, want next candidate", kept)
	}
	service.clearRegionElementCache("next-session")
	if cleared := service.regionElementCacheCandidatesAtPoint("next-session", image.Point{X: 20, Y: 20}); len(cleared) != 0 {
		t.Fatalf("cache after clearing matching session = %#v, want none", cleared)
	}
}

func TestSnapRectToImageEdgesFindsNearestBorder(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 220))
	imagedraw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 245, G: 247, B: 250, A: 255}}, image.Point{}, imagedraw.Src)
	border := color.RGBA{R: 18, G: 24, B: 38, A: 255}
	for x := 40; x <= 260; x++ {
		img.Set(x, 30, border)
		img.Set(x, 170, border)
	}
	for y := 30; y <= 170; y++ {
		img.Set(40, y, border)
		img.Set(260, y, border)
	}

	got, confidence := snapRectToImageEdges(img, image.Rect(47, 37, 253, 163), 18)
	if confidence < 0.5 {
		t.Fatalf("confidence = %.3f, want confident edge snap", confidence)
	}
	if got.Min.X != 40 || got.Min.Y != 30 || got.Max.X != 260 || got.Max.Y != 170 {
		t.Fatalf("snapped rect = %v, want (40,30)-(260,170)", got)
	}
}

func TestDetectImageRegionsAroundPointFindsSidebarPanelWithoutBottomBorder(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 240))
	imagedraw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 252, G: 252, B: 252, A: 255}}, image.Point{}, imagedraw.Src)
	panel := color.RGBA{R: 248, G: 250, B: 252, A: 255}
	imagedraw.Draw(img, image.Rect(0, 40, 96, 240), &image.Uniform{C: panel}, image.Point{}, imagedraw.Src)
	border := color.RGBA{R: 96, G: 132, B: 180, A: 255}
	for y := 40; y < 240; y++ {
		img.Set(96, y, border)
	}
	for x := 0; x < 320; x++ {
		img.Set(x, 40, border)
	}

	got := detectImageRegionsAroundPoint(img, image.Point{X: 48, Y: 180}, []image.Rectangle{img.Bounds()})
	if len(got) == 0 {
		t.Fatal("detectImageRegionsAroundPoint() returned no candidates")
	}
	want := image.Rect(0, 41, 96, 240)
	if got[0].Rect != want {
		t.Fatalf("first image candidate = %v, want sidebar panel %v; all=%#v", got[0].Rect, want, got)
	}
	if got[0].Confidence < 0.5 {
		t.Fatalf("confidence = %.3f, want usable sidebar panel", got[0].Confidence)
	}
}

func TestDetectImageRegionsAroundPointFindsExplorerSidebarWithFaintDivider(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1328, 1188))
	imagedraw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, imagedraw.Src)
	topChrome := color.RGBA{R: 245, G: 247, B: 249, A: 255}
	imagedraw.Draw(img, image.Rect(0, 0, 1328, 52), &image.Uniform{C: topChrome}, image.Point{}, imagedraw.Src)
	divider := color.RGBA{R: 239, G: 241, B: 244, A: 255}
	for y := 52; y < 1188; y++ {
		img.Set(212, y, divider)
		img.Set(213, y, divider)
	}
	for x := 0; x < 1328; x++ {
		img.Set(x, 52, color.RGBA{R: 232, G: 235, B: 238, A: 255})
	}
	selectionBlue := color.RGBA{R: 222, G: 231, B: 241, A: 255}
	imagedraw.Draw(img, image.Rect(0, 300, 212, 338), &image.Uniform{C: selectionBlue}, image.Point{}, imagedraw.Src)

	got := detectImageRegionsAroundPoint(img, image.Point{X: 127, Y: 726}, []image.Rectangle{img.Bounds()})
	if len(got) == 0 {
		t.Fatal("detectImageRegionsAroundPoint() returned no candidates for explorer sidebar")
	}
	want := image.Rect(0, 53, 212, 1188)
	if got[0].Rect != want {
		t.Fatalf("first image candidate = %v, want explorer sidebar %v; all=%#v", got[0].Rect, want, got)
	}
	if got[0].Confidence < 0.5 {
		t.Fatalf("confidence = %.3f, want usable faint-divider sidebar", got[0].Confidence)
	}
}

func TestRegionImagePointCandidatesUsesImagePanelFallbackInsideWindow(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 240))
	imagedraw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 252, G: 252, B: 252, A: 255}}, image.Point{}, imagedraw.Src)
	imagedraw.Draw(img, image.Rect(0, 40, 96, 240), &image.Uniform{C: color.RGBA{R: 248, G: 250, B: 252, A: 255}}, image.Point{}, imagedraw.Src)
	border := color.RGBA{R: 96, G: 132, B: 180, A: 255}
	for y := 40; y < 240; y++ {
		img.Set(96, y, border)
	}
	for x := 0; x < 320; x++ {
		img.Set(x, 40, border)
	}

	session := RegionSelectionSession{
		ID:            "assist-image-panel",
		Bounds:        RegionRect{X: 0, Y: 0, Width: 320, Height: 240},
		CaptureBounds: &RegionRect{X: 0, Y: 0, Width: 320, Height: 240},
		MinimumWidth:  64,
		MinimumHeight: 64,
		Purpose:       regionSelectionPurposeScreenshot,
		Candidates: []RegionSmartCandidate{{
			ID:     "window:test",
			Kind:   regionSmartKindWindow,
			Label:  "Window",
			Bounds: RegionRect{X: 0, Y: 0, Width: 320, Height: 240},
			Score:  0.82,
		}},
	}
	service := &RecordingFreedomService{}
	service.regionSession = &session
	service.regionSnapshotCache = regionAssistSnapshotCache{
		SessionID: session.ID,
		Bounds:    image.Rect(0, 0, 320, 240),
		Image:     img,
	}

	got := service.regionImagePointCandidates(session, image.Point{X: 48, Y: 180}, session.Candidates)
	if len(got) == 0 {
		t.Fatal("regionImagePointCandidates() returned no candidates")
	}
	if got[0].Kind != regionSmartKindEdge || got[0].Bounds != (RegionRect{X: 0, Y: 41, Width: 96, Height: 199}) {
		t.Fatalf("first image panel = %#v, want edge candidate 0,41 96x199", got[0])
	}
}

func TestAssistRegionSelectionKeepsImagePanelLevelWhenElementExists(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 240))
	imagedraw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 252, G: 252, B: 252, A: 255}}, image.Point{}, imagedraw.Src)
	imagedraw.Draw(img, image.Rect(0, 40, 96, 240), &image.Uniform{C: color.RGBA{R: 248, G: 250, B: 252, A: 255}}, image.Point{}, imagedraw.Src)
	border := color.RGBA{R: 96, G: 132, B: 180, A: 255}
	for y := 40; y < 240; y++ {
		img.Set(96, y, border)
	}
	for x := 0; x < 320; x++ {
		img.Set(x, 40, border)
	}

	session := RegionSelectionSession{
		ID:            "assist-element-plus-panel",
		Bounds:        RegionRect{X: 100000, Y: 200000, Width: 320, Height: 240},
		CaptureBounds: &RegionRect{X: 100000, Y: 200000, Width: 320, Height: 240},
		MinimumWidth:  12,
		MinimumHeight: 12,
		Purpose:       regionSelectionPurposeScreenshot,
	}
	service := &RecordingFreedomService{}
	service.regionSession = &session
	service.regionElementCache = regionElementCandidateCache{
		SessionID: session.ID,
		Candidates: []RegionSmartCandidate{{
			ID:     "element:leaf",
			Kind:   regionSmartKindElement,
			Label:  "Leaf item",
			Bounds: RegionRect{X: 36, Y: 168, Width: 46, Height: 24},
			Score:  0.94,
		}},
	}
	service.regionSnapshotCache = regionAssistSnapshotCache{
		SessionID: session.ID,
		Bounds:    image.Rect(100000, 200000, 100320, 200240),
		Image:     img,
	}

	first, err := service.AssistRegionSelection(RegionAssistRequest{
		SessionID: session.ID,
		Purpose:   regionSelectionPurposeScreenshot,
		PointerX:  48,
		PointerY:  180,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Source != "element" || first.Best == nil || first.Best.Kind != regionSmartKindElement {
		t.Fatalf("level 0 assist = source %q best %#v, want element leaf", first.Source, first.Best)
	}

	second, err := service.AssistRegionSelection(RegionAssistRequest{
		SessionID:      session.ID,
		Purpose:        regionSelectionPurposeScreenshot,
		PointerX:       48,
		PointerY:       180,
		CandidateLevel: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Source != "image-hover" || second.Best == nil || second.Best.Kind != regionSmartKindEdge {
		t.Fatalf("level 1 assist = source %q best %#v, want image-hover panel", second.Source, second.Best)
	}
	if second.Best.Bounds != (RegionRect{X: 0, Y: 41, Width: 96, Height: 199}) {
		t.Fatalf("level 1 bounds = %#v, want sidebar panel 0,41 96x199", second.Best.Bounds)
	}
}

func TestCaptureScrollingScreenshotImageStitchesOverlappingFrames(t *testing.T) {
	source := testPatternImage(64, 220)
	offsets := []int{0, 60, 120}
	index := 0
	capture := func(rect image.Rectangle) (*image.RGBA, error) {
		frame := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		imagedraw.Draw(frame, frame.Bounds(), source, image.Point{X: 0, Y: offsets[index]}, imagedraw.Src)
		return frame, nil
	}
	scroll := func(rect image.Rectangle) error {
		if index < len(offsets)-1 {
			index++
		}
		return nil
	}

	got, frames, scrolled, err := captureScrollingScreenshotImage(image.Rect(0, 0, 64, 100), capture, scroll, func(time.Duration) {})
	if err != nil {
		t.Fatalf("captureScrollingScreenshotImage() error = %v", err)
	}
	if !scrolled {
		t.Fatal("scrolled = false, want true for overlapping frames")
	}
	if frames < 3 {
		t.Fatalf("frames = %d, want at least 3", frames)
	}
	if got.Bounds().Dx() != 64 || got.Bounds().Dy() != 220 {
		t.Fatalf("stitched bounds = %v, want 64x220", got.Bounds())
	}
	for _, point := range []image.Point{{X: 12, Y: 20}, {X: 33, Y: 118}, {X: 41, Y: 207}} {
		if got.At(point.X, point.Y) != source.At(point.X, point.Y) {
			t.Fatalf("stitched pixel at %v = %v, want %v", point, got.At(point.X, point.Y), source.At(point.X, point.Y))
		}
	}
}

func TestCaptureScrollingScreenshotImageFallsBackToDirectShotForStaticTarget(t *testing.T) {
	frame := testPatternImage(80, 120)
	scrolls := 0
	capture := func(rect image.Rectangle) (*image.RGBA, error) {
		next := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		imagedraw.Draw(next, next.Bounds(), frame, image.Point{}, imagedraw.Src)
		return next, nil
	}
	scroll := func(rect image.Rectangle) error {
		scrolls++
		return nil
	}

	got, frames, scrolled, err := captureScrollingScreenshotImage(image.Rect(0, 0, 80, 120), capture, scroll, func(time.Duration) {})
	if err != nil {
		t.Fatalf("captureScrollingScreenshotImage() error = %v", err)
	}
	if scrolled {
		t.Fatal("scrolled = true, want false for static target")
	}
	if frames < 2 {
		t.Fatalf("frames = %d, want at least 2", frames)
	}
	if got.Bounds().Dx() != 80 || got.Bounds().Dy() != 120 {
		t.Fatalf("fallback bounds = %v, want direct 80x120 screenshot", got.Bounds())
	}
	if scrolls == 0 {
		t.Fatal("scroll automation was not attempted")
	}
}

func testPNGDataURL(t *testing.T, width int, height int) string {
	t.Helper()
	img := testPatternImage(width, height)
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return whiteboardPNGContentPrefix + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func testPatternImage(width int, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8((x*3 + y) % 255), G: uint8((y*5 + x) % 255), B: uint8((x + y*2) % 255), A: 255})
		}
	}
	return img
}

func mustScreenshotDir(t *testing.T, service *RecordingFreedomService) string {
	t.Helper()
	dir, err := service.screenshotDir()
	if err != nil {
		t.Fatalf("screenshotDir() error = %v", err)
	}
	return dir
}
