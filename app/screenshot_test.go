package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
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
	if got[0].OCRStatus != "none" || got[1].OCRStatus != "none" {
		t.Fatalf("default OCR status = %q/%q, want none", got[0].OCRStatus, got[1].OCRStatus)
	}
}

func TestScreenshotHistoryPreservesReadyOCRStateAndClearsStaleNone(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	items := []ScreenshotItem{
		{
			ID:           "ready",
			Path:         filepath.Join(mustScreenshotDir(t, service), "ready.png"),
			CreatedAt:    "2026-07-04T00:00:02Z",
			Width:        200,
			Height:       120,
			Mode:         "region",
			OCRStatus:    "ready",
			OCRResultID:  "ocr_1",
			OCRModelID:   "ppocrv5-mobile-zh-en",
			OCRLanguage:  "zh-en",
			OCRUpdatedAt: "2026-07-04T00:00:03Z",
		},
		{
			ID:           "stale",
			Path:         filepath.Join(mustScreenshotDir(t, service), "stale.png"),
			CreatedAt:    "2026-07-04T00:00:01Z",
			Width:        100,
			Height:       100,
			Mode:         "region",
			OCRStatus:    "unknown",
			OCRResultID:  "ocr_stale",
			OCRModelID:   "ppocrv5-mobile-zh-en",
			OCRUpdatedAt: "2026-07-04T00:00:03Z",
			OCRError:     "old",
		},
	}
	if err := service.saveScreenshotHistory(items); err != nil {
		t.Fatalf("saveScreenshotHistory() error = %v", err)
	}

	got, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if got[0].OCRStatus != "ready" || got[0].OCRResultID != "ocr_1" {
		t.Fatalf("ready OCR state = %#v, want preserved", got[0])
	}
	if got[1].OCRStatus != "none" || got[1].OCRResultID != "" || got[1].OCRError != "" {
		t.Fatalf("stale OCR state = %#v, want cleared none state", got[1])
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

func TestRecognizeScreenshotRecordsRecoverableOCRFailure(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(80, 60), "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}

	if _, err := service.RecognizeScreenshot(item.ID); err == nil {
		t.Fatal("RecognizeScreenshot() succeeded without installed model, want error")
	}
	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	if history[0].OCRStatus != "failed" || history[0].OCRLanguage != "zh-en" || history[0].OCRError == "" {
		t.Fatalf("OCR failure state = %#v, want failed zh-en with error", history[0])
	}
}

func TestSaveScreenshotImageAutoQueuesOCRWhenEnabled(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := NewRecordingFreedomService()
	service.appData = data
	service.settings = settings.NewService(data)
	service.ocr = ocr.NewService(data)
	service.startOCRJobEventPump()

	current := settings.Default()
	current.OCR.AutoRecognizeScreenshots = true
	if _, err := service.settings.Save(current); err != nil {
		t.Fatalf("Save(settings) error = %v", err)
	}
	item, err := service.saveScreenshotImage(testPatternImage(80, 60), "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}

	var history []ScreenshotItem
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		history, err = service.loadScreenshotHistory()
		if err != nil {
			t.Fatalf("loadScreenshotHistory() error = %v", err)
		}
		if len(history) == 1 && history[0].ID == item.ID && history[0].OCRStatus != "none" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(history) != 1 || history[0].ID != item.ID {
		t.Fatalf("history = %#v, want saved screenshot", history)
	}
	if history[0].OCRStatus == "none" || history[0].OCRLanguage != "zh-en" {
		t.Fatalf("auto OCR history = %#v, want queued/running/ready/failed zh-en state", history[0])
	}
}

func TestQueueRecognizeScreenshotUsesModeSourceKindsAndUpdatesHistory(t *testing.T) {
	cases := []struct {
		name string
		mode string
		want ocr.SourceKind
	}{
		{name: "region", mode: "region", want: ocr.SourceRegionScreenshot},
		{name: "full", mode: "full", want: ocr.SourceFullScreenshot},
		{name: "screen", mode: "screen", want: ocr.SourceFullScreenshot},
		{name: "window", mode: "window", want: ocr.SourceWindowScreenshot},
		{name: "focused window", mode: "focused-window", want: ocr.SourceFocusedWindowScreenshot},
		{name: "scrolling", mode: "scrolling", want: ocr.SourceScrollingScreenshot},
		{name: "whiteboard", mode: "whiteboard", want: ocr.SourceWhiteboard},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewRecordingFreedomService()
			service.appData = appdata.NewService(t.TempDir())
			service.ocr = ocr.NewService(service.appData)
			item, err := service.saveScreenshotImage(testPatternImage(80, 60), tc.mode, nil)
			if err != nil {
				t.Fatalf("saveScreenshotImage() error = %v", err)
			}

			snapshot, err := service.QueueRecognizeScreenshot(item.ID)
			if err != nil {
				t.Fatalf("QueueRecognizeScreenshot() error = %v", err)
			}
			if snapshot.Status != ocr.ResultStatusQueued {
				t.Fatalf("snapshot status = %q, want queued", snapshot.Status)
			}
			if snapshot.Request.SourceKind != tc.want ||
				snapshot.Request.SourceID != item.ID ||
				snapshot.Request.Priority != ocr.JobPriorityInteractive ||
				snapshot.Request.Language != "zh-en" ||
				snapshot.Request.ImagePath != item.Path {
				t.Fatalf("snapshot request = %#v, want %s interactive zh-en request for %s", snapshot.Request, tc.want, item.ID)
			}

			history, err := service.loadScreenshotHistory()
			if err != nil {
				t.Fatalf("loadScreenshotHistory() error = %v", err)
			}
			if len(history) != 1 || history[0].ID != item.ID {
				t.Fatalf("history = %#v, want screenshot item", history)
			}
			if history[0].OCRStatus != ocr.ResultStatusQueued ||
				history[0].OCRLanguage != "zh-en" ||
				history[0].OCRResultID != "" ||
				history[0].OCRError != "" {
				t.Fatalf("manual OCR history = %#v, want queued zh-en without result/error", history[0])
			}
			waitForOCRJobTerminal(t, service.ocr, snapshot.JobID)
		})
	}
}

func TestQueueRecognizeScreenshotRecordsEnqueueFailure(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(80, 60), "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}
	if err := os.Remove(item.Path); err != nil {
		t.Fatalf("Remove(screenshot) error = %v", err)
	}

	if _, err := service.QueueRecognizeScreenshot(item.ID); err == nil {
		t.Fatal("QueueRecognizeScreenshot() succeeded for missing screenshot image, want error")
	}
	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != item.ID {
		t.Fatalf("history = %#v, want screenshot item preserved", history)
	}
	if history[0].OCRStatus != ocr.ResultStatusFailed || history[0].OCRLanguage != "zh-en" || history[0].OCRError == "" {
		t.Fatalf("failed OCR history = %#v, want failed zh-en with error", history[0])
	}
	if _, err := os.Stat(item.ThumbnailPath); err != nil {
		t.Fatalf("thumbnail should remain after OCR enqueue failure, stat error = %v", err)
	}
}

func TestAutoQueueScreenshotOCRRecordsEnqueueFailure(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := NewRecordingFreedomService()
	service.appData = data
	service.settings = settings.NewService(data)
	service.ocr = ocr.NewService(data)

	current := settings.Default()
	current.OCR.AutoRecognizeScreenshots = true
	if _, err := service.settings.Save(current); err != nil {
		t.Fatalf("Save(settings) error = %v", err)
	}
	item := ScreenshotItem{
		ID:        "missing-auto-ocr",
		Path:      filepath.Join(mustScreenshotDir(t, service), "missing-auto-ocr.png"),
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Width:     80,
		Height:    60,
		Mode:      "region",
		OCRStatus: ocr.ResultStatusNone,
	}
	if err := service.saveScreenshotHistory([]ScreenshotItem{item}); err != nil {
		t.Fatalf("saveScreenshotHistory() error = %v", err)
	}

	service.queueScreenshotOCRAfterSave(item)
	var history []ScreenshotItem
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var err error
		history, err = service.loadScreenshotHistory()
		if err != nil {
			t.Fatalf("loadScreenshotHistory() error = %v", err)
		}
		if len(history) == 1 && history[0].OCRStatus == ocr.ResultStatusFailed {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(history) != 1 || history[0].ID != item.ID {
		t.Fatalf("history = %#v, want auto OCR screenshot item preserved", history)
	}
	if history[0].OCRStatus != ocr.ResultStatusFailed || history[0].OCRLanguage != "zh-en" || history[0].OCRError == "" {
		t.Fatalf("auto OCR failure history = %#v, want failed zh-en with error", history[0])
	}
}

func TestScreenshotOCRRealWorkerSmoke(t *testing.T) {
	if os.Getenv("RF_OCR_SCREENSHOT_SMOKE") != "1" {
		t.Skip("set RF_OCR_SCREENSHOT_SMOKE=1 after running scripts/run-local-ocr-smoke.ps1 to exercise the real OCR worker screenshot path")
	}
	fixture := newOCRRealWorkerSmokeFixture(t)
	service := fixture.service
	workerTarget := fixture.workerTarget
	smokeImage := fixture.smokeImage
	longScrollingSmokeImage := buildLongOCRSmokeImage(t, smokeImage, 2800)
	cases := []struct {
		scenario string
		mode     string
		image    image.Image
		want     ocr.SourceKind
	}{
		{scenario: "region", mode: "region", image: smokeImage, want: ocr.SourceRegionScreenshot},
		{scenario: "full", mode: "full", image: smokeImage, want: ocr.SourceFullScreenshot},
		{scenario: "window", mode: "window", image: smokeImage, want: ocr.SourceWindowScreenshot},
		{scenario: "focused-window", mode: "focused-window", image: smokeImage, want: ocr.SourceFocusedWindowScreenshot},
		{scenario: "scrolling", mode: "scrolling", image: smokeImage, want: ocr.SourceScrollingScreenshot},
		{scenario: "scrolling-long", mode: "scrolling", image: longScrollingSmokeImage, want: ocr.SourceScrollingScreenshot},
	}
	items := make([]ScreenshotItem, 0, len(cases))
	saved := make(map[string]struct {
		item ScreenshotItem
		want ocr.SourceKind
	}, len(cases))
	evidence := make([]screenshotOCRSmokeEvidence, 0, len(cases))
	for _, tc := range cases {
		item, err := service.saveScreenshotImage(tc.image, tc.mode, nil)
		if err != nil {
			t.Fatalf("saveScreenshotImage(%s) error = %v", tc.scenario, err)
		}
		snapshot, err := service.QueueRecognizeScreenshot(item.ID)
		if err != nil {
			t.Fatalf("QueueRecognizeScreenshot(%s) error = %v", tc.scenario, err)
		}
		if snapshot.Request.SourceKind != tc.want || snapshot.Request.SourceID != item.ID {
			t.Fatalf("snapshot for %s = %#v, want source kind %s and source id %s", tc.scenario, snapshot.Request, tc.want, item.ID)
		}
		items = append(items, item)
		saved[tc.scenario] = struct {
			item ScreenshotItem
			want ocr.SourceKind
		}{item: item, want: tc.want}
	}

	ready := waitForScreenshotOCRReady(t, service, items, 45*time.Second)
	for _, tc := range cases {
		savedCase := saved[tc.scenario]
		wantItem := savedCase.item
		var readyItem ScreenshotItem
		for _, candidate := range ready {
			if candidate.ID == wantItem.ID {
				readyItem = candidate
				break
			}
		}
		if readyItem.ID == "" {
			t.Fatalf("ready history did not include scenario %s item %s: %#v", tc.scenario, wantItem.ID, ready)
		}
		result, err := service.OpenOcrResult(readyItem.OCRResultID)
		if err != nil {
			t.Fatalf("OpenOcrResult(%s) error = %v", readyItem.OCRResultID, err)
		}
		if result.SourceKind != tc.want || result.SourceID != readyItem.ID || result.ImagePath != readyItem.Path {
			t.Fatalf("OCR result for %s = %#v, want source %s/%s and image path %s", tc.scenario, result, tc.want, readyItem.ID, readyItem.Path)
		}
		if !strings.Contains(result.PlainText, "RecordingFreedom") || !strings.Contains(result.PlainText, "文字识别") {
			t.Fatalf("OCR result for %s plainText = %q, want smoke text", tc.scenario, result.PlainText)
		}
		if tc.scenario == "scrolling-long" {
			if result.Height <= 2400 || result.Height != readyItem.Height {
				t.Fatalf("long scrolling OCR height = result:%d item:%d, want tiled long image height > 2400", result.Height, readyItem.Height)
			}
			if result.Width != readyItem.Width {
				t.Fatalf("long scrolling OCR width = result:%d item:%d, want full long image width", result.Width, readyItem.Width)
			}
		}
		image, err := service.ReadOcrResultImage(result.ID)
		if err != nil {
			t.Fatalf("ReadOcrResultImage(%s) error = %v", result.ID, err)
		}
		if !image.Available || image.Path != readyItem.Path || !strings.HasPrefix(image.DataURL, "data:image/png;base64,") {
			t.Fatalf("OCR result image for %s = %#v, want screenshot image data", tc.scenario, image)
		}
		evidence = append(evidence, screenshotOCRSmokeEvidence{
			Scenario:       tc.scenario,
			Mode:           readyItem.Mode,
			SourceKind:     string(result.SourceKind),
			SourceID:       result.SourceID,
			ScreenshotPath: readyItem.Path,
			ResultID:       result.ID,
			ResultImage:    image.Path,
			ImageWidth:     result.Width,
			ImageHeight:    result.Height,
			ModelID:        result.ModelID,
			Language:       result.Language,
			PlainText:      result.PlainText,
			BlockCount:     len(result.Blocks),
			Blocks:         result.Blocks,
		})
	}

	if err := os.Rename(workerTarget, workerTarget+".disabled"); err != nil {
		t.Fatalf("Rename(worker disabled) error = %v", err)
	}
	cached, err := service.RecognizeScreenshot(ready[0].ID)
	if err != nil {
		t.Fatalf("RecognizeScreenshot(cache hit without worker) error = %v", err)
	}
	if cached.SourceID != ready[0].ID || cached.ImagePath != ready[0].Path ||
		!strings.Contains(cached.PlainText, "RecordingFreedom") ||
		!strings.Contains(cached.PlainText, "文字识别") {
		t.Fatalf("cached OCR result without worker = %#v, want current screenshot source and smoke text", cached)
	}
	persistedCached, err := service.OpenOcrResult(cached.ID)
	if err != nil {
		t.Fatalf("OpenOcrResult(cached %s) error = %v", cached.ID, err)
	}
	if persistedCached.SourceID != cached.SourceID || persistedCached.ImagePath != cached.ImagePath {
		t.Fatalf("persisted cached OCR result = %#v, want current screenshot source", persistedCached)
	}
	for index := range evidence {
		if evidence[index].SourceID == cached.SourceID {
			evidence[index].CacheHitWithoutWorker = true
			evidence[index].CachedResultID = cached.ID
			break
		}
	}
	queuedCacheItem, err := service.saveScreenshotImage(smokeImage, "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage(queued cache hit) error = %v", err)
	}
	queuedCacheSnapshot, err := service.QueueRecognizeScreenshot(queuedCacheItem.ID)
	if err != nil {
		t.Fatalf("QueueRecognizeScreenshot(queued cache hit without worker) error = %v", err)
	}
	if queuedCacheSnapshot.Request.SourceKind != ocr.SourceRegionScreenshot || queuedCacheSnapshot.Request.SourceID != queuedCacheItem.ID {
		t.Fatalf("queued cache snapshot = %#v, want current region screenshot source", queuedCacheSnapshot.Request)
	}
	queuedCacheReady := waitForScreenshotOCRReady(t, service, []ScreenshotItem{queuedCacheItem}, 10*time.Second)[0]
	queuedCacheResult, err := service.OpenOcrResult(queuedCacheReady.OCRResultID)
	if err != nil {
		t.Fatalf("OpenOcrResult(queued cache %s) error = %v", queuedCacheReady.OCRResultID, err)
	}
	if queuedCacheResult.SourceKind != ocr.SourceRegionScreenshot ||
		queuedCacheResult.SourceID != queuedCacheItem.ID ||
		queuedCacheResult.ImagePath != queuedCacheItem.Path ||
		!strings.Contains(queuedCacheResult.PlainText, "RecordingFreedom") ||
		!strings.Contains(queuedCacheResult.PlainText, "文字识别") {
		t.Fatalf("queued cached OCR result without worker = %#v, want current screenshot source and smoke text", queuedCacheResult)
	}
	evidence = append(evidence, screenshotOCRSmokeEvidence{
		Scenario:            "region-queued-cache-hit",
		Mode:                queuedCacheReady.Mode,
		SourceKind:          string(queuedCacheResult.SourceKind),
		SourceID:            queuedCacheResult.SourceID,
		ScreenshotPath:      queuedCacheReady.Path,
		ResultID:            queuedCacheResult.ID,
		ResultImage:         queuedCacheReady.Path,
		ImageWidth:          queuedCacheResult.Width,
		ImageHeight:         queuedCacheResult.Height,
		ModelID:             queuedCacheResult.ModelID,
		Language:            queuedCacheResult.Language,
		PlainText:           queuedCacheResult.PlainText,
		BlockCount:          len(queuedCacheResult.Blocks),
		Blocks:              queuedCacheResult.Blocks,
		QueuedCacheHit:      true,
		QueuedCacheResultID: queuedCacheResult.ID,
	})
	writeScreenshotOCRSmokeEvidence(t, evidence)
}

func TestWhiteboardSelectionOCRRealWorkerSmoke(t *testing.T) {
	if os.Getenv("RF_OCR_WHITEBOARD_SMOKE") != "1" {
		t.Skip("set RF_OCR_WHITEBOARD_SMOKE=1 after running scripts/run-local-ocr-smoke.ps1 to exercise the real OCR worker whiteboard-selection path")
	}
	fixture := newOCRRealWorkerSmokeFixture(t)
	service := fixture.service
	selectionImage, err := service.saveScreenshotImage(fixture.smokeImage, "whiteboard", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage(whiteboard selection smoke) error = %v", err)
	}

	const elementID = "whiteboard-selection-real-worker-image"
	snapshot, err := service.QueueRecognizeWhiteboard(ocr.WhiteboardRequest{
		ImagePath: selectionImage.Path,
		SceneID:   selectionImage.ID,
		ElementID: elementID,
		Language:  "zh-en",
		Priority:  ocr.JobPriorityInteractive,
	})
	if err != nil {
		t.Fatalf("QueueRecognizeWhiteboard(whiteboard selection real worker) error = %v", err)
	}
	if snapshot.Request.SourceKind != ocr.SourceWhiteboardSelection ||
		snapshot.Request.SourceID != selectionImage.ID ||
		snapshot.Request.ImagePath != selectionImage.Path ||
		snapshot.Request.Priority != ocr.JobPriorityInteractive ||
		snapshot.Request.Language != "zh-en" {
		t.Fatalf("whiteboard-selection snapshot request = %#v, want selected image source %s", snapshot.Request, selectionImage.ID)
	}

	ready := waitForScreenshotOCRReady(t, service, []ScreenshotItem{selectionImage}, 45*time.Second)[0]
	result, err := service.OpenOcrResult(ready.OCRResultID)
	if err != nil {
		t.Fatalf("OpenOcrResult(%s) error = %v", ready.OCRResultID, err)
	}
	if result.SourceKind != ocr.SourceWhiteboardSelection ||
		result.SourceID != selectionImage.ID ||
		result.ImagePath != selectionImage.Path ||
		result.Width != selectionImage.Width ||
		result.Height != selectionImage.Height {
		t.Fatalf("whiteboard-selection result = %#v, want selected image source and dimensions %#v", result, selectionImage)
	}
	if !strings.Contains(result.PlainText, "RecordingFreedom") || !strings.Contains(result.PlainText, "文字识别") {
		t.Fatalf("whiteboard-selection plainText = %q, want smoke text", result.PlainText)
	}
	if len(result.Blocks) == 0 {
		t.Fatalf("whiteboard-selection result has no OCR blocks: %#v", result)
	}
	image, err := service.ReadOcrResultImage(result.ID)
	if err != nil {
		t.Fatalf("ReadOcrResultImage(%s) error = %v", result.ID, err)
	}
	if !image.Available || image.Path != selectionImage.Path || !strings.HasPrefix(image.DataURL, "data:image/png;base64,") {
		t.Fatalf("whiteboard-selection result image = %#v, want selected image data", image)
	}

	writeWhiteboardOCRSmokeEvidence(t, []screenshotOCRSmokeEvidence{{
		Scenario:        "whiteboard-selection-real-worker",
		Mode:            "whiteboard-selection",
		SourceKind:      string(result.SourceKind),
		SourceID:        result.SourceID,
		SourceImagePath: selectionImage.Path,
		ElementID:       elementID,
		ScreenshotPath:  selectionImage.Path,
		ResultID:        result.ID,
		ResultImage:     image.Path,
		ImageWidth:      result.Width,
		ImageHeight:     result.Height,
		ModelID:         result.ModelID,
		Language:        result.Language,
		PlainText:       result.PlainText,
		BlockCount:      len(result.Blocks),
		Blocks:          result.Blocks,
	}})
}

func TestQueueRecognizePinnedScreenshotUsesPinnedSourceAndUpdatesHistory(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(80, 60), "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}

	snapshot, err := service.QueueRecognizePinnedScreenshot(item.ID)
	if err != nil {
		t.Fatalf("QueueRecognizePinnedScreenshot() error = %v", err)
	}
	if snapshot.Status != ocr.ResultStatusQueued {
		t.Fatalf("snapshot status = %q, want queued", snapshot.Status)
	}
	if snapshot.Request.SourceKind != ocr.SourcePinnedScreenshot ||
		snapshot.Request.SourceID != item.ID ||
		snapshot.Request.Priority != ocr.JobPriorityInteractive ||
		snapshot.Request.Language != "zh-en" {
		t.Fatalf("snapshot request = %#v, want pinned screenshot interactive zh-en request", snapshot.Request)
	}
	if snapshot.Request.ImagePath != item.Path {
		t.Fatalf("snapshot image path = %q, want %q", snapshot.Request.ImagePath, item.Path)
	}

	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != item.ID {
		t.Fatalf("history = %#v, want pinned OCR screenshot item", history)
	}
	if history[0].OCRStatus != ocr.ResultStatusQueued ||
		history[0].OCRLanguage != "zh-en" ||
		history[0].OCRResultID != "" ||
		history[0].OCRError != "" {
		t.Fatalf("pinned OCR history = %#v, want queued zh-en without result/error", history[0])
	}
	waitForOCRJobTerminal(t, service.ocr, snapshot.JobID)
}

func TestQueueRecognizeWhiteboardSnapshotUsesWhiteboardSourceAndUpdatesHistory(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(120, 80), "whiteboard", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}

	snapshot, err := service.QueueRecognizeWhiteboard(ocr.WhiteboardRequest{
		ImagePath: item.Path,
		SceneID:   item.ID,
		Language:  "zh-en",
	})
	if err != nil {
		t.Fatalf("QueueRecognizeWhiteboard() error = %v", err)
	}
	if snapshot.Status != ocr.ResultStatusQueued {
		t.Fatalf("snapshot status = %q, want queued", snapshot.Status)
	}
	if snapshot.Request.SourceKind != ocr.SourceWhiteboard ||
		snapshot.Request.SourceID != item.ID ||
		snapshot.Request.Priority != ocr.JobPriorityInteractive ||
		snapshot.Request.Language != "zh-en" {
		t.Fatalf("snapshot request = %#v, want whiteboard interactive zh-en request", snapshot.Request)
	}

	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != item.ID {
		t.Fatalf("history = %#v, want whiteboard screenshot item", history)
	}
	if history[0].OCRStatus != ocr.ResultStatusQueued || history[0].OCRLanguage != "zh-en" {
		t.Fatalf("whiteboard OCR history = %#v, want queued zh-en", history[0])
	}
	waitForOCRJobTerminal(t, service.ocr, snapshot.JobID)
}

func TestWhiteboardSelectionOCRJobEventUpdatesHistory(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(120, 80), "whiteboard", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}

	service.handleOCRJobEvent(ocr.JobEvent{
		JobID:  "ocr-job-whiteboard-selection-ready",
		Status: ocr.ResultStatusReady,
		Request: ocr.RecognizeRequest{
			ImagePath:  item.Path,
			SourceKind: ocr.SourceWhiteboardSelection,
			SourceID:   item.ID,
			Language:   "zh-en",
			Priority:   ocr.JobPriorityInteractive,
		},
		Result: &ocr.Result{
			ID:         "ocr-result-whiteboard-selection-ready",
			SourceKind: ocr.SourceWhiteboardSelection,
			SourceID:   item.ID,
			ImagePath:  item.Path,
			ModelID:    "ppocrv5-mobile-zh-en",
			Language:   "zh-en",
			Width:      item.Width,
			Height:     item.Height,
			PlainText:  "RecordingFreedom",
			CreatedAt:  time.Now().UTC(),
		},
	})

	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != item.ID {
		t.Fatalf("history = %#v, want whiteboard selection item", history)
	}
	if history[0].OCRStatus != ocr.ResultStatusReady ||
		history[0].OCRResultID != "ocr-result-whiteboard-selection-ready" ||
		history[0].OCRModelID != "ppocrv5-mobile-zh-en" ||
		history[0].OCRLanguage != "zh-en" ||
		history[0].OCRError != "" {
		t.Fatalf("whiteboard-selection OCR history = %#v, want ready result", history[0])
	}
}

func TestReadOcrResultImageUsesManagedResultImagePath(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(120, 80), "whiteboard", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}
	result := ocr.Result{
		ID:          "ocr_result_image",
		SourceKind:  ocr.SourceWhiteboard,
		SourceID:    item.ID,
		ImagePath:   item.Path,
		ImageSHA256: "sha-managed-image",
		ModelID:     "ppocrv5-mobile-zh-en",
		Language:    "zh-en",
		Width:       item.Width,
		Height:      item.Height,
		PlainText:   "RecordingFreedom",
		CreatedAt:   time.Now().UTC(),
	}
	if err := service.ocr.WriteResult(result); err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	image, err := service.ReadOcrResultImage(result.ID)
	if err != nil {
		t.Fatalf("ReadOcrResultImage() error = %v", err)
	}
	if !image.Available || !strings.HasPrefix(image.DataURL, "data:image/png;base64,") || image.Path != item.Path || image.Bytes <= 0 {
		t.Fatalf("image = %#v, want managed PNG data URL", image)
	}
}

func TestReadOcrResultImageFallsBackToScreenshotHistorySource(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(96, 64), "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}
	result := ocr.Result{
		ID:          "ocr_legacy_result_image",
		SourceKind:  ocr.SourceRegionScreenshot,
		SourceID:    item.ID,
		ImageSHA256: "sha-legacy-image",
		ModelID:     "ppocrv5-mobile-zh-en",
		Language:    "zh-en",
		Width:       item.Width,
		Height:      item.Height,
		PlainText:   "Legacy result",
		CreatedAt:   time.Now().UTC(),
	}
	if err := service.ocr.WriteResult(result); err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	image, err := service.ReadOcrResultImage(result.ID)
	if err != nil {
		t.Fatalf("ReadOcrResultImage() error = %v", err)
	}
	if !image.Available || !strings.HasPrefix(image.DataURL, "data:image/png;base64,") || image.Path != item.Path || image.Bytes <= 0 {
		t.Fatalf("image = %#v, want fallback screenshot PNG data URL", image)
	}
}

func TestReadOcrResultImageRejectsSelectionFallbackWithoutImagePath(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	item, err := service.saveScreenshotImage(testPatternImage(96, 64), "whiteboard", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}
	result := ocr.Result{
		ID:          "ocr_selection_without_image",
		SourceKind:  ocr.SourceWhiteboardSelection,
		SourceID:    item.ID,
		ImageSHA256: "sha-selection-image",
		ModelID:     "ppocrv5-mobile-zh-en",
		Language:    "zh-en",
		PlainText:   "selection",
		CreatedAt:   time.Now().UTC(),
	}
	if err := service.ocr.WriteResult(result); err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	if _, err := service.ReadOcrResultImage(result.ID); err == nil {
		t.Fatal("ReadOcrResultImage() allowed whiteboard-selection fallback without its own image path")
	}
}

func TestReadOcrResultImageRejectsOutsideDataRoot(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
	outsidePath := filepath.Join(t.TempDir(), "outside.png")
	if err := os.WriteFile(outsidePath, []byte("not a real png but outside root"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}
	result := ocr.Result{
		ID:          "ocr_outside_image",
		SourceKind:  ocr.SourceImage,
		SourceID:    "outside",
		ImagePath:   outsidePath,
		ImageSHA256: "sha-outside-image",
		ModelID:     "ppocrv5-mobile-zh-en",
		Language:    "zh-en",
		PlainText:   "outside",
		CreatedAt:   time.Now().UTC(),
	}
	if err := service.ocr.WriteResult(result); err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}

	if _, err := service.ReadOcrResultImage(result.ID); err == nil {
		t.Fatal("ReadOcrResultImage() accepted an OCR result image outside app data root")
	}
}

func TestWhiteboardOCRRequestUsesSelectionSourceForElement(t *testing.T) {
	req, err := whiteboardOCRRequest(ocr.WhiteboardRequest{
		ImagePath: "selection.png",
		SceneID:   "scene-1",
		ElementID: "image-element-1",
	})
	if err != nil {
		t.Fatalf("whiteboardOCRRequest() error = %v", err)
	}
	if req.SourceKind != ocr.SourceWhiteboardSelection {
		t.Fatalf("source kind = %q, want whiteboard-selection", req.SourceKind)
	}
	if req.SourceID != "scene-1" {
		t.Fatalf("source id = %q, want scene id", req.SourceID)
	}
}

func TestWhiteboardOCRRequestAllowsBackgroundPriorityForRecordingAnnotation(t *testing.T) {
	req, err := whiteboardOCRRequest(ocr.WhiteboardRequest{
		ImagePath: "annotation.png",
		SceneID:   "recording-package",
		Priority:  ocr.JobPriorityBackground,
	})
	if err != nil {
		t.Fatalf("whiteboardOCRRequest() error = %v", err)
	}
	if req.SourceKind != ocr.SourceWhiteboard ||
		req.SourceID != "recording-package" ||
		req.Priority != ocr.JobPriorityBackground {
		t.Fatalf("request = %#v, want whiteboard background request for recording annotation", req)
	}
}

func TestOCRJobEventUpdatesScreenshotHistoryState(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	item, err := service.saveScreenshotImage(testPatternImage(80, 60), "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}

	service.handleOCRJobEvent(ocr.JobEvent{
		JobID:  "ocr-job-1",
		Status: ocr.ResultStatusRunning,
		Request: ocr.RecognizeRequest{
			ImagePath:  item.Path,
			SourceKind: ocr.SourceRegionScreenshot,
			SourceID:   item.ID,
			Language:   "zh-en",
		},
	})
	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if history[0].OCRStatus != "running" || history[0].OCRLanguage != "zh-en" {
		t.Fatalf("running OCR history = %#v, want running zh-en", history[0])
	}

	service.handleOCRJobEvent(ocr.JobEvent{
		JobID:  "ocr-job-1",
		Status: ocr.ResultStatusReady,
		Request: ocr.RecognizeRequest{
			ImagePath:  item.Path,
			SourceKind: ocr.SourceRegionScreenshot,
			SourceID:   item.ID,
			Language:   "zh-en",
		},
		Result: &ocr.Result{
			ID:       "ocr_result_1",
			ModelID:  "ppocrv5-mobile-zh-en",
			Language: "zh-en",
		},
	})
	history, err = service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if history[0].OCRStatus != "ready" ||
		history[0].OCRResultID != "ocr_result_1" ||
		history[0].OCRModelID != "ppocrv5-mobile-zh-en" ||
		history[0].OCRError != "" {
		t.Fatalf("ready OCR history = %#v, want ready result metadata", history[0])
	}
}

func TestCancelledOCRJobEventDoesNotAllowLateReadyHistoryState(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	item, err := service.saveScreenshotImage(testPatternImage(80, 60), "region", nil)
	if err != nil {
		t.Fatalf("saveScreenshotImage() error = %v", err)
	}
	request := ocr.RecognizeRequest{
		ImagePath:  item.Path,
		SourceKind: ocr.SourceRegionScreenshot,
		SourceID:   item.ID,
		Language:   "zh-en",
	}

	service.handleOCRJobEvent(ocr.JobEvent{JobID: "ocr-job-cancelled", Status: ocr.ResultStatusRunning, Request: request})
	service.handleOCRJobEvent(ocr.JobEvent{JobID: "ocr-job-cancelled", Status: ocr.ResultStatusCancelled, Request: request})
	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if history[0].OCRStatus != ocr.ResultStatusNone || history[0].OCRResultID != "" || history[0].OCRError != "" {
		t.Fatalf("cancelled OCR history = %#v, want none without stale result/error", history[0])
	}

	service.handleOCRJobEvent(ocr.JobEvent{
		JobID:   "ocr-job-cancelled",
		Status:  ocr.ResultStatusReady,
		Request: request,
		Result: &ocr.Result{
			ID:       "late_cancelled_result",
			ModelID:  "ppocrv5-mobile-zh-en",
			Language: "zh-en",
		},
	})
	history, err = service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if history[0].OCRStatus != ocr.ResultStatusNone || history[0].OCRResultID != "" || history[0].OCRModelID != "" {
		t.Fatalf("late ready after cancel changed OCR history = %#v, want cancelled none state preserved", history[0])
	}

	service.handleOCRJobEvent(ocr.JobEvent{JobID: "ocr-job-new", Status: ocr.ResultStatusQueued, Request: request})
	service.handleOCRJobEvent(ocr.JobEvent{
		JobID:   "ocr-job-new",
		Status:  ocr.ResultStatusReady,
		Request: request,
		Result: &ocr.Result{
			ID:       "new_result",
			ModelID:  "ppocrv5-mobile-zh-en",
			Language: "zh-en",
		},
	})
	history, err = service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if history[0].OCRStatus != ocr.ResultStatusReady ||
		history[0].OCRResultID != "new_result" ||
		history[0].OCRModelID != "ppocrv5-mobile-zh-en" {
		t.Fatalf("new OCR job history = %#v, want new ready result after old cancellation", history[0])
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

func TestCaptureScrollingScreenshotStaticTargetQueuesRegionOCR(t *testing.T) {
	service := NewRecordingFreedomService()
	service.appData = appdata.NewService(t.TempDir())
	service.ocr = ocr.NewService(service.appData)
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
	region := RegionRect{X: 10, Y: 20, Width: 80, Height: 120}

	item, err := service.captureScrollingScreenshotWith(image.Rect(10, 20, 90, 140), &region, capture, scroll, func(time.Duration) {})
	if err != nil {
		t.Fatalf("captureScrollingScreenshotWith() error = %v", err)
	}
	if item.Mode != "region" {
		t.Fatalf("item mode = %q, want region for static scrolling target fallback", item.Mode)
	}
	if item.Width != 80 || item.Height != 120 {
		t.Fatalf("item size = %dx%d, want direct region 80x120", item.Width, item.Height)
	}
	if item.Region == nil || *item.Region != region {
		t.Fatalf("item region = %#v, want %#v", item.Region, region)
	}
	if scrolls == 0 {
		t.Fatal("scroll automation was not attempted")
	}

	snapshot, err := service.QueueRecognizeScreenshot(item.ID)
	if err != nil {
		t.Fatalf("QueueRecognizeScreenshot() error = %v", err)
	}
	if snapshot.Request.SourceKind != ocr.SourceRegionScreenshot {
		t.Fatalf("OCR source kind = %q, want region-screenshot for no-scroll fallback", snapshot.Request.SourceKind)
	}
	if snapshot.Request.SourceID != item.ID || snapshot.Request.ImagePath != item.Path {
		t.Fatalf("OCR request = %#v, want fallback screenshot item path/id", snapshot.Request)
	}
	history, err := service.loadScreenshotHistory()
	if err != nil {
		t.Fatalf("loadScreenshotHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != item.ID {
		t.Fatalf("history = %#v, want fallback screenshot item", history)
	}
	if history[0].Mode != "region" || history[0].OCRStatus != ocr.ResultStatusQueued || history[0].OCRLanguage != "zh-en" {
		t.Fatalf("fallback OCR history = %#v, want queued region screenshot OCR", history[0])
	}
	waitForOCRJobTerminal(t, service.ocr, snapshot.JobID)
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

type ocrRealWorkerSmokeFixture struct {
	service      *RecordingFreedomService
	root         string
	workerTarget string
	smokeImage   image.Image
}

func newOCRRealWorkerSmokeFixture(t *testing.T) ocrRealWorkerSmokeFixture {
	t.Helper()
	repoRoot := mustRepoRoot(t)
	target := runtime.GOOS + "-" + runtime.GOARCH
	workerName := "rf-ocr-worker"
	if runtime.GOOS == "windows" {
		workerName += ".exe"
	}
	workerSource := strings.TrimSpace(os.Getenv("RF_OCR_WORKER_PATH"))
	if workerSource == "" {
		workerSource = filepath.Join(repoRoot, "app", "tools", "ocr-worker", target, workerName)
	}
	runtimeSource := strings.TrimSpace(os.Getenv("RF_OCR_RUNTIME_DIR"))
	if runtimeSource == "" {
		runtimeSource = filepath.Join(repoRoot, "app", "tools", "onnxruntime", target)
	}
	modelPackage := strings.TrimSpace(os.Getenv("RF_OCR_MODEL_PACKAGE"))
	if modelPackage == "" {
		modelPackage = newestModelPackageForSmoke(t, filepath.Join(repoRoot, "release-out", "ocr-models"))
	}
	requireSmokeFile(t, workerSource)
	requireSmokeDir(t, runtimeSource)
	requireSmokeFile(t, modelPackage)

	data := appdata.NewService(t.TempDir())
	service := NewRecordingFreedomService()
	service.appData = data
	service.settings = settings.NewService(data)
	service.ocr = ocr.NewService(data)
	service.startOCRJobEventPump()
	root, err := data.RootDir()
	if err != nil {
		t.Fatalf("RootDir() error = %v", err)
	}
	workerTarget := filepath.Join(root, "tools", "ocr-worker", target, workerName)
	copySmokeFile(t, workerSource, workerTarget)
	copySmokeDir(t, runtimeSource, filepath.Join(root, "tools", "onnxruntime", target))

	installedModel, err := service.InstallOcrModelPackage(modelPackage)
	if err != nil {
		t.Fatalf("InstallOcrModelPackage() error = %v", err)
	}
	if installedModel.ID == "" {
		t.Fatalf("InstallOcrModelPackage() returned empty model id for %s", modelPackage)
	}
	if status, err := service.SetActiveOcrModel(installedModel.ID); err != nil {
		t.Fatalf("SetActiveOcrModel(%s) error = %v", installedModel.ID, err)
	} else if status.Status != ocr.StatusReady {
		t.Fatalf("OCR status = %#v, want ready real worker status", status)
	}

	smokeImagePath := filepath.Join(root, "data", "models", "ocr", installedModel.ID, "smoke.png")
	return ocrRealWorkerSmokeFixture{
		service:      service,
		root:         root,
		workerTarget: workerTarget,
		smokeImage:   readSmokeImage(t, smokeImagePath),
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "third_party", "ocr-models", "manifest.json")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatalf("could not locate repository root from %s", wd)
		}
		wd = parent
	}
}

func newestModelPackageForSmoke(t *testing.T, dir string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "ppocrv5-mobile-zh-en-*.zip"))
	if err != nil {
		t.Fatalf("Glob(model packages) error = %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no OCR model package found under %s; run scripts/run-local-ocr-smoke.ps1 first or set RF_OCR_MODEL_PACKAGE", dir)
	}
	newest := matches[0]
	newestInfo, err := os.Stat(newest)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", newest, err)
	}
	for _, match := range matches[1:] {
		info, err := os.Stat(match)
		if err != nil {
			t.Fatalf("Stat(%s) error = %v", match, err)
		}
		if info.ModTime().After(newestInfo.ModTime()) {
			newest = match
			newestInfo = info
		}
	}
	return newest
}

func requireSmokeFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("required OCR smoke file is unavailable at %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("required OCR smoke file path is a directory: %s", path)
	}
}

func requireSmokeDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("required OCR smoke directory is unavailable at %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("required OCR smoke directory path is not a directory: %s", path)
	}
}

func copySmokeFile(t *testing.T, source string, target string) {
	t.Helper()
	info, err := os.Stat(source)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", source, err)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, data, info.Mode()); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", target, err)
	}
}

func copySmokeDir(t *testing.T, source string, target string) {
	t.Helper()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v", source, err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", target, err)
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		targetPath := filepath.Join(target, entry.Name())
		if entry.IsDir() {
			copySmokeDir(t, sourcePath, targetPath)
			continue
		}
		copySmokeFile(t, sourcePath, targetPath)
	}
}

func readSmokeImage(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", path, err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("Decode(%s) error = %v", path, err)
	}
	return img
}

func buildLongOCRSmokeImage(t *testing.T, source image.Image, minHeight int) image.Image {
	t.Helper()
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		t.Fatalf("source smoke image has invalid bounds: %v", bounds)
	}
	repeats := (minHeight + height - 1) / height
	if repeats < 2 {
		repeats = 2
	}
	out := image.NewRGBA(image.Rect(0, 0, width, height*repeats))
	for index := 0; index < repeats; index++ {
		target := image.Rect(0, index*height, width, (index+1)*height)
		imagedraw.Draw(out, target, source, bounds.Min, imagedraw.Src)
	}
	return out
}

type screenshotOCRSmokeEvidence struct {
	Scenario              string      `json:"scenario,omitempty"`
	Mode                  string      `json:"mode"`
	SourceKind            string      `json:"sourceKind"`
	SourceID              string      `json:"sourceId"`
	SourceImagePath       string      `json:"sourceImagePath,omitempty"`
	ElementID             string      `json:"elementId,omitempty"`
	ScreenshotPath        string      `json:"screenshotPath"`
	ResultID              string      `json:"resultId"`
	ResultImage           string      `json:"resultImage"`
	EvidenceImage         string      `json:"evidenceImage,omitempty"`
	EvidenceOverlay       string      `json:"evidenceOverlay,omitempty"`
	ImageWidth            int         `json:"imageWidth"`
	ImageHeight           int         `json:"imageHeight"`
	ModelID               string      `json:"modelId"`
	Language              string      `json:"language"`
	PlainText             string      `json:"plainText"`
	BlockCount            int         `json:"blockCount"`
	Blocks                []ocr.Block `json:"blocks"`
	CacheHitWithoutWorker bool        `json:"cacheHitWithoutWorker,omitempty"`
	CachedResultID        string      `json:"cachedResultId,omitempty"`
	QueuedCacheHit        bool        `json:"queuedCacheHit,omitempty"`
	QueuedCacheResultID   string      `json:"queuedCacheResultId,omitempty"`
}

func writeScreenshotOCRSmokeEvidence(t *testing.T, evidence []screenshotOCRSmokeEvidence) {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("RF_OCR_EVIDENCE_DIR"))
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(RF_OCR_EVIDENCE_DIR=%s) error = %v", dir, err)
	}
	for index := range evidence {
		stem := evidence[index].Mode
		if strings.TrimSpace(evidence[index].Scenario) != "" {
			stem = evidence[index].Scenario
		}
		imagePath := filepath.Join(dir, stem+".png")
		copySmokeFile(t, evidence[index].ResultImage, imagePath)
		evidence[index].EvidenceImage = imagePath
		overlayPath := filepath.Join(dir, stem+"-ocr-overlay.png")
		writeScreenshotOCRSmokeOverlay(t, evidence[index].ResultImage, overlayPath, evidence[index].Blocks)
		evidence[index].EvidenceOverlay = overlayPath
	}
	path := filepath.Join(dir, "screenshot-ocr-real-worker-smoke.json")
	data, err := json.MarshalIndent(struct {
		SchemaVersion int                          `json:"schemaVersion"`
		GeneratedAt   string                       `json:"generatedAt"`
		Entries       []screenshotOCRSmokeEvidence `json:"entries"`
	}{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Entries:       evidence,
	}, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(screenshot OCR smoke evidence) error = %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	t.Logf("wrote screenshot OCR smoke evidence: %s", path)
}

func writeWhiteboardOCRSmokeEvidence(t *testing.T, evidence []screenshotOCRSmokeEvidence) {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("RF_OCR_EVIDENCE_DIR"))
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(RF_OCR_EVIDENCE_DIR=%s) error = %v", dir, err)
	}
	for index := range evidence {
		stem := evidence[index].Mode
		if strings.TrimSpace(evidence[index].Scenario) != "" {
			stem = evidence[index].Scenario
		}
		imagePath := filepath.Join(dir, stem+".png")
		copySmokeFile(t, evidence[index].ResultImage, imagePath)
		evidence[index].EvidenceImage = imagePath
		overlayPath := filepath.Join(dir, stem+"-ocr-overlay.png")
		writeScreenshotOCRSmokeOverlay(t, evidence[index].ResultImage, overlayPath, evidence[index].Blocks)
		evidence[index].EvidenceOverlay = overlayPath
	}
	path := filepath.Join(dir, "whiteboard-ocr-real-worker-smoke.json")
	data, err := json.MarshalIndent(struct {
		SchemaVersion int                          `json:"schemaVersion"`
		GeneratedAt   string                       `json:"generatedAt"`
		Entries       []screenshotOCRSmokeEvidence `json:"entries"`
	}{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Entries:       evidence,
	}, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(whiteboard OCR smoke evidence) error = %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	t.Logf("wrote whiteboard OCR smoke evidence: %s", path)
}

func writeScreenshotOCRSmokeOverlay(t *testing.T, source string, target string, blocks []ocr.Block) {
	t.Helper()
	file, err := os.Open(source)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", source, err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("Decode(%s) error = %v", source, err)
	}
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	imagedraw.Draw(out, bounds, img, bounds.Min, imagedraw.Src)
	red := color.RGBA{R: 236, G: 54, B: 54, A: 255}
	for _, block := range blocks {
		minX, minY, maxX, maxY, ok := ocrBlockPixelBounds(block, bounds)
		if ok {
			drawOCRSmokeRect(out, minX, minY, maxX, maxY, red)
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(target), err)
	}
	outFile, err := os.Create(target)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", target, err)
	}
	defer outFile.Close()
	if err := png.Encode(outFile, out); err != nil {
		t.Fatalf("Encode(%s) error = %v", target, err)
	}
}

func ocrBlockPixelBounds(block ocr.Block, bounds image.Rectangle) (int, int, int, int, bool) {
	if len(block.Box) == 0 {
		return 0, 0, 0, 0, false
	}
	minX, maxX := block.Box[0].X, block.Box[0].X
	minY, maxY := block.Box[0].Y, block.Box[0].Y
	for _, point := range block.Box[1:] {
		if point.X < minX {
			minX = point.X
		}
		if point.X > maxX {
			maxX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	left := clampOCRSmokeCoordinate(int(minX+0.5), bounds.Min.X, bounds.Max.X-1)
	top := clampOCRSmokeCoordinate(int(minY+0.5), bounds.Min.Y, bounds.Max.Y-1)
	right := clampOCRSmokeCoordinate(int(maxX+0.5), bounds.Min.X, bounds.Max.X-1)
	bottom := clampOCRSmokeCoordinate(int(maxY+0.5), bounds.Min.Y, bounds.Max.Y-1)
	if right <= left || bottom <= top {
		return 0, 0, 0, 0, false
	}
	return left, top, right, bottom, true
}

func clampOCRSmokeCoordinate(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func drawOCRSmokeRect(img *image.RGBA, left int, top int, right int, bottom int, c color.RGBA) {
	const thickness = 3
	for offset := 0; offset < thickness; offset++ {
		for x := left; x <= right; x++ {
			img.Set(x, top+offset, c)
			img.Set(x, bottom-offset, c)
		}
		for y := top; y <= bottom; y++ {
			img.Set(left+offset, y, c)
			img.Set(right-offset, y, c)
		}
	}
}

func waitForScreenshotOCRReady(t *testing.T, service *RecordingFreedomService, items []ScreenshotItem, timeout time.Duration) []ScreenshotItem {
	t.Helper()
	want := map[string]bool{}
	for _, item := range items {
		want[item.ID] = true
	}
	deadline := time.Now().Add(timeout)
	var last []ScreenshotItem
	for time.Now().Before(deadline) {
		history, err := service.loadScreenshotHistory()
		if err != nil {
			t.Fatalf("loadScreenshotHistory() error = %v", err)
		}
		last = history
		ready := make([]ScreenshotItem, 0, len(items))
		for _, item := range history {
			if !want[item.ID] {
				continue
			}
			if item.OCRStatus == ocr.ResultStatusFailed {
				t.Fatalf("screenshot %s OCR failed: %s", item.ID, item.OCRError)
			}
			if item.OCRStatus == ocr.ResultStatusReady && item.OCRResultID != "" {
				ready = append(ready, item)
			}
		}
		if len(ready) == len(items) {
			return ready
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for real OCR screenshot smoke; last history = %#v", last)
	return nil
}

func waitForOCRJobTerminal(t *testing.T, service *ocr.Service, jobID string) {
	t.Helper()
	if service == nil {
		t.Fatal("OCR service is nil")
	}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-service.Events():
			if event.JobID != jobID {
				continue
			}
			switch event.Status {
			case ocr.ResultStatusReady, ocr.ResultStatusFailed, ocr.ResultStatusCancelled:
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for OCR job %q to settle", jobID)
		}
	}
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
