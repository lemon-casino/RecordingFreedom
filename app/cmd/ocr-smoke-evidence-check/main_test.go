package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunAcceptsCompleteOCREvidence(t *testing.T) {
	dir := t.TempDir()
	writeEvidenceFixture(t, dir, nil)

	report, err := run(dir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.OK {
		t.Fatalf("report.OK = false: %#v", report.Checks)
	}
	if len(report.Checks) != len(requiredScenarios)+5 {
		t.Fatalf("checks = %d, want %d", len(report.Checks), len(requiredScenarios)+5)
	}
}

func TestRunAcceptsExplicitExpectedModel(t *testing.T) {
	dir := t.TempDir()
	const modelID = "ppocrv6-mobile-zh-en"
	writeEvidenceFixture(t, dir, func(doc *evidenceDocument) {
		for index := range doc.Entries {
			doc.Entries[index].ModelID = modelID
		}
	})
	whiteboardDoc := completeWhiteboardEvidenceDocument(t, dir)
	whiteboardDoc.Entries[0].ModelID = modelID
	writeEvidenceDocument(t, filepath.Join(dir, whiteboardEvidenceFileName), whiteboardDoc)

	defaultReport, err := run(dir)
	if err != nil {
		t.Fatalf("run(default model) error = %v", err)
	}
	if defaultReport.OK {
		t.Fatalf("default report OK = true, want blocked for non-default model")
	}
	report, err := runWithExpectedModel(dir, modelID)
	if err != nil {
		t.Fatalf("runWithExpectedModel() error = %v", err)
	}
	if !report.OK {
		t.Fatalf("report.OK = false: %#v", report.Checks)
	}
}

func TestRunRejectsMissingCacheHitWithoutWorkerProof(t *testing.T) {
	dir := t.TempDir()
	writeEvidenceFixture(t, dir, func(doc *evidenceDocument) {
		for index := range doc.Entries {
			doc.Entries[index].CacheHitWithoutWorker = false
			doc.Entries[index].CachedResultID = ""
		}
	})

	report, err := run(dir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked without cache hit proof")
	}
	if !reportHasMessage(report, "no scenario proves cacheHitWithoutWorker") {
		t.Fatalf("cache hit proof failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingQueuedCacheHitProof(t *testing.T) {
	dir := t.TempDir()
	writeEvidenceFixture(t, dir, func(doc *evidenceDocument) {
		for index := range doc.Entries {
			if doc.Entries[index].Scenario == "region-queued-cache-hit" {
				doc.Entries[index].QueuedCacheHit = false
				doc.Entries[index].QueuedCacheResultID = ""
			}
		}
	})

	report, err := run(dir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked without queued cache proof")
	}
	if !reportHasMessage(report, "region-queued-cache-hit must prove queuedCacheHit") {
		t.Fatalf("queued cache proof failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsOverlayOutsideEvidenceDir(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside-overlay.png")
	writePNG(t, outside, 900, 280)
	writeEvidenceFixture(t, dir, func(doc *evidenceDocument) {
		for index := range doc.Entries {
			if doc.Entries[index].Scenario == "region" {
				doc.Entries[index].EvidenceOverlay = outside
			}
		}
	})

	report, err := run(dir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked for evidence outside directory")
	}
	if !reportHasMessage(report, "must stay inside evidence directory") {
		t.Fatalf("outside evidence failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsShortScrollingLongEvidence(t *testing.T) {
	dir := t.TempDir()
	writeEvidenceFixture(t, dir, func(doc *evidenceDocument) {
		for index := range doc.Entries {
			if doc.Entries[index].Scenario == "scrolling-long" {
				doc.Entries[index].ImageHeight = 280
				doc.Entries[index].EvidenceImage = "scrolling-long.png"
				doc.Entries[index].EvidenceOverlay = "scrolling-long-ocr-overlay.png"
				writePNG(t, filepath.Join(dir, "scrolling-long.png"), 900, 280)
				writePNG(t, filepath.Join(dir, "scrolling-long-ocr-overlay.png"), 900, 280)
			}
		}
	})

	report, err := run(dir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked for short scrolling-long evidence")
	}
	if !reportHasMessage(report, "want > 2400") {
		t.Fatalf("scrolling-long height failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsWhiteboardSelectionWithoutElementID(t *testing.T) {
	dir := t.TempDir()
	writeEvidenceFixture(t, dir, nil)
	whiteboardDoc := completeWhiteboardEvidenceDocument(t, dir)
	whiteboardDoc.Entries[0].ElementID = ""
	writeEvidenceDocument(t, filepath.Join(dir, whiteboardEvidenceFileName), whiteboardDoc)

	report, err := run(dir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked without whiteboard selection element id")
	}
	if !reportHasMessage(report, "elementId is required") {
		t.Fatalf("whiteboard selection element id failure not found: %#v", report.Checks)
	}
}

func writeEvidenceFixture(t *testing.T, dir string, mutate func(*evidenceDocument)) {
	t.Helper()
	doc := evidenceDocument{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
	for scenario, sourceKind := range requiredScenarios {
		width := 900
		height := 280
		if scenario == "scrolling-long" {
			height = 2800
		}
		imageName := scenario + ".png"
		overlayName := scenario + "-ocr-overlay.png"
		writePNG(t, filepath.Join(dir, imageName), width, height)
		writePNG(t, filepath.Join(dir, overlayName), width, height)
		doc.Entries = append(doc.Entries, evidenceEntry{
			Scenario:        scenario,
			Mode:            modeForScenario(scenario),
			SourceKind:      sourceKind,
			SourceID:        "source-" + scenario,
			ScreenshotPath:  filepath.Join(dir, imageName),
			ResultID:        "result-" + scenario,
			ResultImage:     filepath.Join(dir, imageName),
			EvidenceImage:   imageName,
			EvidenceOverlay: overlayName,
			ImageWidth:      width,
			ImageHeight:     height,
			ModelID:         "ppocrv5-mobile-zh-en",
			Language:        "zh-en",
			PlainText:       "RecordingFreedom\n文字识别",
			BlockCount:      2,
			Blocks: []ocrBlock{
				{
					ID:         "block-" + scenario + "-1",
					Text:       "RecordingFreedom",
					Confidence: 0.98,
					Box: []ocrPoint{
						{X: 10, Y: 10},
						{X: 320, Y: 10},
						{X: 320, Y: 70},
						{X: 10, Y: 70},
					},
				},
				{
					ID:         "block-" + scenario + "-2",
					Text:       "文字识别",
					Confidence: 0.97,
					Box: []ocrPoint{
						{X: 10, Y: 90},
						{X: 220, Y: 90},
						{X: 220, Y: 150},
						{X: 10, Y: 150},
					},
					LineIndex: 1,
				},
			},
			CacheHitWithoutWorker: scenario == "region",
			CachedResultID:        cacheResultIDForScenario(scenario),
			QueuedCacheHit:        scenario == "region-queued-cache-hit",
			QueuedCacheResultID:   queuedCacheResultIDForScenario(scenario),
		})
	}
	if mutate != nil {
		mutate(&doc)
	}
	writeEvidenceDocument(t, filepath.Join(dir, screenshotEvidenceFileName), doc)
	writeEvidenceDocument(t, filepath.Join(dir, whiteboardEvidenceFileName), completeWhiteboardEvidenceDocument(t, dir))
}

func completeWhiteboardEvidenceDocument(t *testing.T, dir string) evidenceDocument {
	t.Helper()
	const scenario = "whiteboard-selection-real-worker"
	imageName := scenario + ".png"
	overlayName := scenario + "-ocr-overlay.png"
	writePNG(t, filepath.Join(dir, imageName), 900, 280)
	writePNG(t, filepath.Join(dir, overlayName), 900, 280)
	return evidenceDocument{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Entries: []evidenceEntry{
			{
				Scenario:        scenario,
				Mode:            "whiteboard-selection",
				SourceKind:      "whiteboard-selection",
				SourceID:        "whiteboard-scene",
				SourceImagePath: filepath.Join(dir, "whiteboard-selection-source.png"),
				ElementID:       "selected-image-element",
				ScreenshotPath:  filepath.Join(dir, imageName),
				ResultID:        "result-" + scenario,
				ResultImage:     filepath.Join(dir, imageName),
				EvidenceImage:   imageName,
				EvidenceOverlay: overlayName,
				ImageWidth:      900,
				ImageHeight:     280,
				ModelID:         "ppocrv5-mobile-zh-en",
				Language:        "zh-en",
				PlainText:       "RecordingFreedom\n文字识别",
				BlockCount:      2,
				Blocks: []ocrBlock{
					{
						ID:         "block-" + scenario + "-1",
						Text:       "RecordingFreedom",
						Confidence: 0.98,
						Box: []ocrPoint{
							{X: 10, Y: 10},
							{X: 320, Y: 10},
							{X: 320, Y: 70},
							{X: 10, Y: 70},
						},
					},
					{
						ID:         "block-" + scenario + "-2",
						Text:       "文字识别",
						Confidence: 0.97,
						Box: []ocrPoint{
							{X: 10, Y: 90},
							{X: 220, Y: 90},
							{X: 220, Y: 150},
							{X: 10, Y: 150},
						},
						LineIndex: 1,
					},
				},
			},
		},
	}
}

func writeEvidenceDocument(t *testing.T, path string, doc evidenceDocument) {
	t.Helper()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(evidence) error = %v", err)
	}
}

func modeForScenario(scenario string) string {
	switch scenario {
	case "focused-window":
		return "focused-window"
	case "scrolling", "scrolling-long":
		return "scrolling"
	case "region-queued-cache-hit":
		return "region"
	default:
		return scenario
	}
}

func cacheResultIDForScenario(scenario string) string {
	if scenario == "region" {
		return "cached-region-result"
	}
	return ""
}

func queuedCacheResultIDForScenario(scenario string) string {
	if scenario == "region-queued-cache-hit" {
		return "queued-cache-result"
	}
	return ""
}

func writePNG(t *testing.T, path string, width int, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("Encode(%s) error = %v", path, err)
	}
}

func reportHasMessage(report report, needle string) bool {
	for _, check := range report.Checks {
		if strings.Contains(check.Message, needle) {
			return true
		}
	}
	return false
}
