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

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocrevidence"
)

func TestRunWritesChecklistAndAcceptsCompleteVisualDir(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual")
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		writePlanPNG(t, filepath.Join(visualDir, requirement.RecommendedFile), 800, 520)
	}
	outDir := filepath.Join(root, "checklist")
	report, err := run(options{visualDir: visualDir, outDir: outDir})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.CheckComplete {
		t.Fatalf("CheckComplete = false, missing = %#v", report.MissingVisualRequirements)
	}
	if len(report.MatchedVisualRequirements) != len(ocrevidence.RequiredVisualEvidence) {
		t.Fatalf("matched = %d, want %d", len(report.MatchedVisualRequirements), len(ocrevidence.RequiredVisualEvidence))
	}
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.md"), "recording annotation OCR safety")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.md"), "Session boundary runbook")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.md"), "Capture runbook")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.md"), "Evidence chain requirements")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.md"), "Whiteboard selected image OCR")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "requiredSourceKinds")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "evidenceSessionRunbook")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "captureSteps")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "evidenceChainRequirements")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "existingVisualDimensions")
	if len(report.CaptureSteps) != len(ocrevidence.RequiredCaptureSteps) {
		t.Fatalf("captureSteps = %d, want %d", len(report.CaptureSteps), len(ocrevidence.RequiredCaptureSteps))
	}
	if len(report.EvidenceChainRequirements) != len(ocrevidence.EvidenceChainRequirements) {
		t.Fatalf("evidenceChainRequirements = %d, want %d", len(report.EvidenceChainRequirements), len(ocrevidence.EvidenceChainRequirements))
	}
}

func TestRunReportsMissingVisualRequirements(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual")
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		if requirement.Name == "recording annotation OCR safety" {
			continue
		}
		writePlanPNG(t, filepath.Join(visualDir, requirement.RecommendedFile), 800, 520)
	}
	report, err := run(options{visualDir: visualDir})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want missing recording annotation OCR safety")
	}
	if !containsString(report.MissingVisualRequirements, "recording annotation OCR safety") {
		t.Fatalf("missing requirements = %#v", report.MissingVisualRequirements)
	}
}

func TestRunReportsTooSmallVisualRequirement(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual")
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		width := 800
		height := 520
		if requirement.Name == "OCR result floating panel" {
			width = 120
			height = 80
		}
		writePlanPNG(t, filepath.Join(visualDir, requirement.RecommendedFile), width, height)
	}
	report, err := run(options{visualDir: visualDir})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want too-small OCR result floating panel")
	}
	if len(report.VisualDimensionFailures) == 0 || !strings.Contains(strings.Join(report.VisualDimensionFailures, "\n"), "OCR result floating panel") {
		t.Fatalf("dimension failures = %#v, want OCR result floating panel failure", report.VisualDimensionFailures)
	}
}

func TestRunWritesDataRootPrecheck(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data-root")
	writePlanDataRootFixture(t, dataRoot, "")
	outDir := filepath.Join(root, "checklist")

	report, err := run(options{dataRoot: dataRoot, outDir: outDir})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.CheckComplete || report.DataRootPrecheck == nil || !report.DataRootPrecheck.CheckComplete {
		t.Fatalf("report = %#v, want complete data-root precheck", report)
	}
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.md"), "Data Root Precheck")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.md"), "session id")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "dataRootPrecheck")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "sessionId")
	requirePlanFile(t, filepath.Join(outDir, "visual-capture-checklist.json"), "appLogLines")
}

func TestRunReportsIncompleteDataRootPrecheck(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data-root")
	writePlanDataRootFixture(t, dataRoot, string(ocr.SourcePinnedScreenshot))

	report, err := run(options{dataRoot: dataRoot})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want missing pinned screenshot render")
	}
	if report.DataRootPrecheck == nil || !strings.Contains(strings.Join(report.DataRootPrecheck.MissingRequirements, "\n"), "missing client rendered sourceKind=pinned-screenshot") {
		t.Fatalf("dataRootPrecheck = %#v, want pinned-screenshot render failure", report.DataRootPrecheck)
	}
}

func TestRunCheckRejectsVisualDirWithoutDataRoot(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual")
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		writePlanPNG(t, filepath.Join(visualDir, requirement.RecommendedFile), 800, 520)
	}

	_, err := run(options{visualDir: visualDir, check: true})
	if err == nil || !strings.Contains(err.Error(), "-data-root is required") {
		t.Fatalf("run() error = %v, want required data-root for checked visual precheck", err)
	}
}

func TestRunRejectsNonImageVisualPrecheck(t *testing.T) {
	root := t.TempDir()
	visualDir := filepath.Join(root, "visual")
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		writePlanPNG(t, filepath.Join(visualDir, requirement.RecommendedFile), 800, 520)
	}
	writePlanFile(t, filepath.Join(visualDir, "not-an-image.txt"), "not an image")

	_, err := run(options{visualDir: visualDir})
	if err == nil || !strings.Contains(err.Error(), "not a decodable image") {
		t.Fatalf("run() error = %v, want non-image precheck failure", err)
	}
}

func TestMarkdownChecklistIncludesRecommendedFilenames(t *testing.T) {
	report, err := run(options{})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	markdown := markdownChecklist(report)
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		if !strings.Contains(markdown, requirement.RecommendedFile) {
			t.Fatalf("markdown checklist missing %s", requirement.RecommendedFile)
		}
	}
}

func TestPlanIncludesExecutableCaptureRunbook(t *testing.T) {
	report, err := run(options{})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(report.CaptureSteps) == 0 {
		t.Fatal("CaptureSteps is empty")
	}
	markdown := markdownChecklist(report)
	for _, needle := range []string{
		"Session boundary runbook",
		"ocr-desktop-evidence-session.exe -event start",
		"ocr-desktop-evidence-session -event end",
		"Do not create session markers during export",
		"Capture runbook",
		"Evidence chain requirements",
		"sourceId, and resultId",
		"No duplicate sourceKind",
		"Region screenshot OCR",
		"Whiteboard selected image OCR",
		"ocr/open-result sourceKind=whiteboard-selection",
		"client.ocr-result/rendered sourceKind=whiteboard-selection",
		"recording-annotation-ocr-safety.png",
	} {
		if !strings.Contains(markdown, needle) {
			t.Fatalf("capture runbook markdown missing %q", needle)
		}
	}
}

func writePlanFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func requirePlanFile(t *testing.T, path string, mustContain string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if !strings.Contains(string(data), mustContain) {
		t.Fatalf("%s is missing %q", path, mustContain)
	}
}

func writePlanPNG(t *testing.T, path string, width int, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 240, G: 244, B: 248, A: 255})
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writePlanDataRootFixture(t *testing.T, root string, omitClientRenderedSourceKind string) {
	t.Helper()
	createdAt := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	appLog := strings.Builder{}
	appLog.WriteString(planAppLogLine(t, createdAt.Add(-time.Minute), ocrevidence.EvidenceSessionComponent, ocrevidence.EvidenceSessionStartEvent, map[string]string{"sessionId": "plan-session"}))
	appLog.WriteString(planAppLogLine(t, createdAt, "annotation-overlay", "show", map[string]string{"packageDir": "plan-recording-package"}))
	appLog.WriteString(planAppLogLine(t, createdAt, "annotation-overlay", "save-capture", map[string]string{"packageDir": "plan-recording-package", "bytes": "2048"}))
	jobEvents := strings.Builder{}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		result := makePlanOCRResult(t, root, kind, createdAt)
		writePlanJSON(t, filepath.Join(root, "data", "ocr", "results", string(kind)+".json"), result)
		key := string(kind)
		appLog.WriteString(planAppLogLine(t, createdAt, "ocr", "queue-request", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "priority": "interactive"}))
		appLog.WriteString(planAppLogLine(t, createdAt, "ocr", "open-result", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID}))
		appLog.WriteString(planAppLogLine(t, createdAt, "ocr", "read-result-image", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID}))
		appLog.WriteString(planAppLogLine(t, createdAt, "client.ocr-result", "preview-loaded", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID}))
		if key != omitClientRenderedSourceKind {
			appLog.WriteString(planAppLogLine(t, createdAt, "client.ocr-result", "rendered", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID}))
		}
		jobEvents.WriteString(`{"sourceKind":"` + key + `","sourceId":"` + result.SourceID + `","status":"queued"}` + "\n")
		ready := map[string]any{
			"sourceKind": key,
			"sourceId":   result.SourceID,
			"status":     ocr.ResultStatusReady,
			"resultId":   result.ID,
		}
		data, err := json.Marshal(ready)
		if err != nil {
			t.Fatalf("Marshal(ready) error = %v", err)
		}
		jobEvents.Write(data)
		jobEvents.WriteByte('\n')
	}
	appLog.WriteString(planAppLogLine(t, createdAt, "ocr", "queue-request", map[string]string{"sourceKind": "whiteboard", "sourceId": "plan-recording-package", "priority": "background"}))
	appLog.WriteString(planAppLogLine(t, createdAt.Add(time.Minute), ocrevidence.EvidenceSessionComponent, ocrevidence.EvidenceSessionEndEvent, map[string]string{"sessionId": "plan-session"}))
	writePlanFile(t, filepath.Join(root, "logs", "recordingfreedom-plan.log"), appLog.String())
	writePlanFile(t, filepath.Join(root, "data", "ocr", "evidence", "ocr-job-events.jsonl"), jobEvents.String())
}

func makePlanOCRResult(t *testing.T, root string, kind ocr.SourceKind, createdAt time.Time) ocr.Result {
	t.Helper()
	imagePath := filepath.Join(root, "data", "ocr", "source-images", string(kind)+".png")
	writePlanFile(t, imagePath, "image bytes")
	return ocr.Result{
		ID:         "result-" + string(kind),
		SourceKind: kind,
		SourceID:   "source-" + string(kind),
		ImagePath:  imagePath,
		ModelID:    "ppocrv5-mobile-zh-en",
		Language:   "zh-en",
		Width:      120,
		Height:     48,
		PlainText:  "RecordingFreedom 文字识别",
		CreatedAt:  createdAt,
	}
}

func planAppLogLine(t *testing.T, timestamp time.Time, component string, event string, fields map[string]string) string {
	t.Helper()
	line, err := json.Marshal(struct {
		Timestamp string            `json:"timestamp"`
		Component string            `json:"component"`
		Event     string            `json:"event"`
		Fields    map[string]string `json:"fields"`
	}{
		Timestamp: timestamp.Format(time.RFC3339Nano),
		Component: component,
		Event:     event,
		Fields:    fields,
	})
	if err != nil {
		t.Fatalf("Marshal app log line error = %v", err)
	}
	return string(line) + "\n"
}

func writePlanJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(%s) error = %v", path, err)
	}
	writePlanFile(t, path, string(data)+"\n")
}
