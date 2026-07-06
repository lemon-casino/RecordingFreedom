package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocrevidence"
)

func TestRunExportsDesktopOCREvidenceFromDataRoot(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual-source")
	createDesktopDataRoot(t, root)
	createVisualEvidence(t, visualDir)

	evidenceDir := filepath.Join(root, "evidence")
	report, err := run(options{
		dataRoot:          root,
		evidenceDir:       evidenceDir,
		visualDir:         visualDir,
		version:           "v-test",
		commit:            "abc123",
		artifact:          "GitHub Actions release artifact",
		knownFailures:     "none",
		displayCount:      "2",
		displayResolution: "1920x1080",
		displayScale:      "150%",
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.OK || report.ResultCount != len(ocrevidence.RequiredSourceKinds) || report.JobEventLines != len(ocrevidence.RequiredSourceKinds)*2 || report.AppLogLines != expectedAppLogLines() || report.VisualFiles == 0 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.VisualRequirements) != len(ocrevidence.RequiredVisualEvidence) {
		t.Fatalf("visual requirements = %d, want %d: %#v", len(report.VisualRequirements), len(ocrevidence.RequiredVisualEvidence), report.VisualRequirements)
	}
	requireFile(t, filepath.Join(evidenceDir, "README.md"))
	requireFile(t, filepath.Join(evidenceDir, "platform.txt"))
	requireFile(t, filepath.Join(evidenceDir, "app-log.jsonl"))
	exportedAppLog, err := os.ReadFile(filepath.Join(evidenceDir, "app-log.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile(app-log.jsonl) error = %v", err)
	}
	if strings.Contains(string(exportedAppLog), "historical-window-poison") {
		t.Fatalf("exported app-log contains historical out-of-window event:\n%s", string(exportedAppLog))
	}
	requireFile(t, filepath.Join(evidenceDir, "ocr-job-events.jsonl"))
	exportedJobEvents, err := os.ReadFile(filepath.Join(evidenceDir, "ocr-job-events.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile(ocr-job-events.jsonl) error = %v", err)
	}
	if strings.Contains(string(exportedJobEvents), "historical-job-poison") {
		t.Fatalf("exported job events contain historical out-of-scope event:\n%s", string(exportedJobEvents))
	}
	requireFile(t, filepath.Join(evidenceDir, "export-report.json"))
	requireFile(t, filepath.Join(evidenceDir, "data-root-precheck.json"))
	requireFile(t, filepath.Join(evidenceDir, "visual-capture-checklist.md"))
	requireFile(t, filepath.Join(evidenceDir, "visual-capture-checklist.json"))
	if report.ChecklistMarkdown != "visual-capture-checklist.md" || report.ChecklistJSON != "visual-capture-checklist.json" || report.DataRootPrecheck != "data-root-precheck.json" {
		t.Fatalf("report paths checklist=%q/%q dataRootPrecheck=%q", report.ChecklistMarkdown, report.ChecklistJSON, report.DataRootPrecheck)
	}
	var checklist ocrevidence.ChecklistReport
	readJSONTest(t, filepath.Join(evidenceDir, "visual-capture-checklist.json"), &checklist)
	if !checklist.CheckComplete || checklist.DataRootPrecheck == nil || !checklist.DataRootPrecheck.CheckComplete || len(checklist.EvidenceChainRequirements) != len(ocrevidence.EvidenceChainRequirements) {
		t.Fatalf("checklist = %#v", checklist)
	}
	var dataRootPrecheck ocrevidence.DataRootPrecheckReport
	readJSONTest(t, filepath.Join(evidenceDir, "data-root-precheck.json"), &dataRootPrecheck)
	if !dataRootPrecheck.CheckComplete || dataRootPrecheck.RunWindow.ResultStart.IsZero() || dataRootPrecheck.Session.SessionID == "" {
		t.Fatalf("data root precheck = %#v", dataRootPrecheck)
	}
	manifestPath := filepath.Join(evidenceDir, "visual", "visual-manifest.json")
	requireFile(t, manifestPath)
	var manifest visualManifest
	readJSONTest(t, manifestPath, &manifest)
	if manifest.SchemaVersion != 1 || len(manifest.Files) != report.VisualFiles {
		t.Fatalf("visual manifest = %#v, report visual files = %d", manifest, report.VisualFiles)
	}
	for _, entry := range manifest.Files {
		if entry.Path == "" || entry.Bytes <= 0 || entry.SHA256 == "" || entry.Width <= 0 || entry.Height <= 0 {
			t.Fatalf("visual manifest entry = %#v, want path/bytes/sha256/dimensions", entry)
		}
		requireFile(t, filepath.Join(evidenceDir, "visual", filepath.FromSlash(entry.Path)))
	}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		resultPath := filepath.Join(evidenceDir, "results", string(kind)+".json")
		requireFile(t, resultPath)
		var result ocr.Result
		readJSONTest(t, resultPath, &result)
		if result.SourceKind != kind {
			t.Fatalf("%s sourceKind = %s", resultPath, result.SourceKind)
		}
		if !strings.HasPrefix(filepath.ToSlash(result.ImagePath), "images/") {
			t.Fatalf("%s imagePath = %q, want evidence-relative images path", resultPath, result.ImagePath)
		}
		requireFile(t, filepath.Join(evidenceDir, filepath.FromSlash(result.ImagePath)))
	}
}

func TestRunExportedEvidencePassesDesktopEvidenceChecker(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual-source")
	createDesktopDataRoot(t, root)
	createVisualEvidence(t, visualDir)

	evidenceDir := filepath.Join(root, "evidence")
	_, err := run(options{
		dataRoot:          root,
		evidenceDir:       evidenceDir,
		visualDir:         visualDir,
		version:           "v-test",
		commit:            "abc123",
		artifact:          "GitHub Actions release artifact",
		knownFailures:     "none",
		displayCount:      "2",
		displayResolution: "1920x1080",
		displayScale:      "150%",
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	cmd := exec.Command("go", "run", "../ocr-desktop-evidence-check", "-evidence-dir", evidenceDir, "-must-contain", "RecordingFreedom", "-must-contain", "文字识别")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ocr-desktop-evidence-check failed: %v\n%s", err, string(output))
	}
	if !strings.Contains(string(output), `"ok": true`) {
		t.Fatalf("checker output missing ok=true:\n%s", string(output))
	}
}

func TestRunRejectsNonImageVisualEvidence(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual-source")
	createDesktopDataRoot(t, root)
	createVisualEvidence(t, visualDir)
	writeText(t, filepath.Join(visualDir, "not-an-image.txt"), "not an image")

	_, err := run(options{
		dataRoot:          root,
		evidenceDir:       filepath.Join(root, "evidence"),
		visualDir:         visualDir,
		displayCount:      "1",
		displayResolution: "1920x1080",
		displayScale:      "100%",
	})
	if err == nil || !strings.Contains(err.Error(), "not a decodable image") {
		t.Fatalf("run() error = %v, want non-image visual evidence failure", err)
	}
}

func TestRunRejectsMissingRequiredVisualScene(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual-source")
	createDesktopDataRoot(t, root)
	createVisualEvidence(t, visualDir)
	if err := os.Remove(filepath.Join(visualDir, "recording-annotation-ocr-safety.png")); err != nil {
		t.Fatalf("Remove(recording annotation visual) error = %v", err)
	}

	_, err := run(options{
		dataRoot:          root,
		evidenceDir:       filepath.Join(root, "evidence"),
		visualDir:         visualDir,
		displayCount:      "1",
		displayResolution: "1920x1080",
		displayScale:      "100%",
	})
	if err == nil || !strings.Contains(err.Error(), "recording annotation OCR safety") {
		t.Fatalf("run() error = %v, want missing recording annotation visual requirement", err)
	}
}

func TestRunRejectsTooSmallVisualEvidence(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual-source")
	createDesktopDataRoot(t, root)
	createVisualEvidence(t, visualDir)
	writePNGTest(t, filepath.Join(visualDir, "ocr-result-floating-panel.png"), 120, 80)

	_, err := run(options{
		dataRoot:          root,
		evidenceDir:       filepath.Join(root, "evidence"),
		visualDir:         visualDir,
		displayCount:      "1",
		displayResolution: "1920x1080",
		displayScale:      "100%",
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least") {
		t.Fatalf("run() error = %v, want too-small visual evidence failure", err)
	}
}

func TestRunRejectsMissingVisualEvidenceDirectory(t *testing.T) {
	root := t.TempDir()
	createDesktopDataRoot(t, root)
	_, err := run(options{
		dataRoot:          root,
		evidenceDir:       filepath.Join(root, "evidence"),
		displayCount:      "1",
		displayResolution: "1920x1080",
		displayScale:      "100%",
	})
	if err == nil || !strings.Contains(err.Error(), "-visual-dir") {
		t.Fatalf("run() error = %v, want missing visual-dir failure", err)
	}
}

func TestRunRejectsMissingDataRoot(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual-source")
	createVisualEvidence(t, visualDir)

	_, err := run(options{
		evidenceDir:       filepath.Join(root, "evidence"),
		visualDir:         visualDir,
		displayCount:      "1",
		displayResolution: "1920x1080",
		displayScale:      "100%",
	})
	if err == nil || !strings.Contains(err.Error(), "-data-root is required") {
		t.Fatalf("run() error = %v, want missing data-root failure", err)
	}
}

func createDesktopDataRoot(t *testing.T, root string) {
	t.Helper()
	createdAt := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	appLog := strings.Builder{}
	appLog.WriteString(exportAppLogLine(createdAt.Add(-time.Minute), "ocr-desktop-evidence", "session-start", `"sessionId":"export-session"`))
	appLog.WriteString(exportAppLogLine(createdAt, "app", "startup", `"platform":"windows"`))
	appLog.WriteString(exportAppLogLine(createdAt, "floating-panel", "show", `"kind":"ocr-result","contextId":"result-region-screenshot"`))
	appLog.WriteString(exportAppLogLine(createdAt, "annotation-overlay", "show", `"packageDir":"test-recording-package","targetType":"screen","targetId":"screen-1"`))
	appLog.WriteString(exportAppLogLine(createdAt, "annotation-overlay", "save-capture", `"packageDir":"test-recording-package","bytes":"4096"`))
	jobEvents := strings.Builder{}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		result := makeResult(t, root, kind, createdAt)
		writeJSONTest(t, filepath.Join(root, "data", "ocr", "results", string(kind)+".json"), result)
		appLog.WriteString(exportAppLogLine(createdAt, "ocr", "queue-request", `"sourceKind":"`+string(kind)+`","sourceId":"`+result.SourceID+`","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","priority":"interactive","force":"false"`))
		jobEvents.WriteString(`{"event":"ocr.job.queued","sourceKind":"` + string(kind) + `","sourceId":"` + result.SourceID + `","status":"queued"}` + "\n")
		ready := map[string]any{
			"event":      "ocr.job.finished",
			"sourceKind": string(kind),
			"sourceId":   result.SourceID,
			"status":     ocr.ResultStatusReady,
			"result":     result,
		}
		data, err := json.Marshal(ready)
		if err != nil {
			t.Fatalf("Marshal(ready) error = %v", err)
		}
		jobEvents.Write(data)
		jobEvents.WriteByte('\n')
	}
	appLog.WriteString(exportAppLogLine(createdAt, "ocr", "queue-request", `"sourceKind":"whiteboard","sourceId":"test-recording-package","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","priority":"background","force":"false"`))
	for _, kind := range ocrevidence.RequiredSourceKinds {
		resultPath := filepath.Join(root, "data", "ocr", "results", string(kind)+".json")
		var result ocr.Result
		readJSONTest(t, resultPath, &result)
		key := string(kind)
		appLog.WriteString(exportAppLogLine(createdAt, "ocr", "open-result", `"resultId":"`+result.ID+`","sourceKind":"`+key+`","sourceId":"`+result.SourceID+`","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","blockCount":"1"`))
		appLog.WriteString(exportAppLogLine(createdAt, "ocr", "read-result-image", `"resultId":"`+result.ID+`","sourceKind":"`+key+`","sourceId":"`+result.SourceID+`","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","bytes":"192"`))
		appLog.WriteString(exportAppLogLine(createdAt, "client.ocr-result", "preview-loaded", `"resultId":"`+result.ID+`","sourceKind":"`+key+`","sourceId":"`+result.SourceID+`","width":"120","height":"48","blockCount":"1","available":"true","bytes":"192"`))
		appLog.WriteString(exportAppLogLine(createdAt, "client.ocr-result", "rendered", `"resultId":"`+result.ID+`","sourceKind":"`+key+`","sourceId":"`+result.SourceID+`","width":"120","height":"48","blockCount":"1","polygonCount":"1","hasPreview":"true"`))
	}
	appLog.WriteString(exportAppLogLine(createdAt.Add(time.Minute), "ocr-desktop-evidence", "session-end", `"sessionId":"export-session"`))
	writeText(t, filepath.Join(root, "logs", "recordingfreedom-2026-07-06.log"), appLog.String())
	writeText(t, filepath.Join(root, "logs", "recordingfreedom-2026-07-05.log"),
		exportAppLogLine(createdAt.Add(-24*time.Hour), "ocr", "open-result", `"resultId":"result-region-screenshot","sourceKind":"region-screenshot","sourceId":"source-region-screenshot","marker":"historical-window-poison"`))
	jobEvents.WriteString(`{"event":"ocr.job.finished","sourceKind":"region-screenshot","sourceId":"source-region-screenshot","status":"ready","resultId":"historical-job-poison"}` + "\n")
	writeText(t, filepath.Join(root, "data", "ocr", "evidence", "ocr-job-events.jsonl"), jobEvents.String())
}

func expectedAppLogLines() int {
	return 7 + len(ocrevidence.RequiredSourceKinds)*5
}

func makeResult(t *testing.T, root string, kind ocr.SourceKind, createdAt time.Time) ocr.Result {
	t.Helper()
	imagePath := filepath.Join(root, "data", "ocr", "source-images", string(kind)+".png")
	writePNGTest(t, imagePath, 120, 48)
	return ocr.Result{
		ID:          "result-" + string(kind),
		SourceKind:  kind,
		SourceID:    "source-" + string(kind),
		ImagePath:   imagePath,
		ImageSHA256: strings.Repeat("a", 64),
		ModelID:     "ppocrv5-mobile-zh-en",
		Language:    "zh-en",
		Width:       120,
		Height:      48,
		Blocks: []ocr.Block{{
			ID:         "block-1",
			Text:       "RecordingFreedom 文字识别",
			Confidence: 0.92,
			Box: []ocr.Point{
				{X: 4, Y: 4},
				{X: 116, Y: 4},
				{X: 116, Y: 44},
				{X: 4, Y: 44},
			},
			LineIndex: 0,
		}},
		PlainText:  "RecordingFreedom\n文字识别",
		CreatedAt:  createdAt,
		DurationMS: 120,
	}
}

func exportAppLogLine(timestamp time.Time, component string, event string, fields string) string {
	return `{"timestamp":"` + timestamp.Format(time.RFC3339Nano) + `","component":"` + component + `","event":"` + event + `","fields":{` + fields + `}}` + "\n"
}

func createVisualEvidence(t *testing.T, dir string) {
	t.Helper()
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		writePNGTest(t, filepath.Join(dir, requirement.RecommendedFile), 800, 520)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("%s is not a non-empty file", path)
	}
}

func writeText(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func writeJSONTest(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(%s) error = %v", path, err)
	}
	writeText(t, path, string(data)+"\n")
}

func readJSONTest(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
}

func writePNGTest(t *testing.T, path string, width int, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 238, G: 242, B: 247, A: 255})
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
