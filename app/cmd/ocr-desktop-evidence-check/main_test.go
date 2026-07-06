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

func TestRunAcceptsCompleteDesktopOCREvidence(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	report, err := run(options{
		evidenceDir:        root,
		requireTranslation: true,
		mustContain:        []string{"RecordingFreedom", "文字识别"},
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.OK {
		t.Fatalf("report.OK = false; checks=%#v sourceKinds=%#v", report.Checks, report.SourceKinds)
	}
	if len(report.SourceKinds) != len(ocrevidence.RequiredSourceKinds) {
		t.Fatalf("sourceKinds = %d, want %d", len(report.SourceKinds), len(ocrevidence.RequiredSourceKinds))
	}
}

func TestRunRejectsMissingDesktopOCRSourceKind(t *testing.T) {
	root := createDesktopOCREvidence(t, map[ocr.SourceKind]bool{ocr.SourceWhiteboardSelection: true})
	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing source kind failure")
	}
	found := false
	for _, source := range report.SourceKinds {
		if source.SourceKind == string(ocr.SourceWhiteboardSelection) && strings.Contains(source.Message, "missing OCR result") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing source failure not found: %#v", report.SourceKinds)
	}
}

func TestRunRejectsOutOfBoundsOCRBlock(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	resultPath := filepath.Join(root, "results", string(ocr.SourceRegionScreenshot)+".json")
	var result ocr.Result
	readJSONFile(t, resultPath, &result)
	result.Blocks[0].Box[0].X = float64(result.Width + 10)
	writeJSONFile(t, resultPath, result)

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want out-of-bounds block failure")
	}
	found := false
	for _, source := range report.SourceKinds {
		if source.SourceKind == string(ocr.SourceRegionScreenshot) && strings.Contains(source.Message, "outside image") {
			found = true
		}
	}
	if !found {
		t.Fatalf("out-of-bounds failure not found: %#v", report.SourceKinds)
	}
}

func TestRunRejectsAppLogResultChainMismatch(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	appLogPath := filepath.Join(root, "app-log.jsonl")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	rewritten := strings.Builder{}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `"event":"open-result"`) && strings.Contains(line, `"sourceKind":"region-screenshot"`) {
			line = strings.Replace(line, `"sourceId":"source-region-screenshot"`, `"sourceId":"different-region"`, 1)
		}
		if strings.TrimSpace(line) != "" {
			rewritten.WriteString(line)
			rewritten.WriteByte('\n')
		}
	}
	writeTextFile(t, appLogPath, rewritten.String())

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want app-log/result chain mismatch")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "evidence chain" && strings.Contains(check.Message, "app-log open-result sourceKind=region-screenshot sourceId=source-region-screenshot resultId=result-region-screenshot") {
			found = true
		}
	}
	if !found {
		t.Fatalf("app-log/result chain mismatch not found: %#v", report.Checks)
	}
}

func TestRunRejectsJobEventResultChainMismatch(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	jobEventsPath := filepath.Join(root, "ocr-job-events.jsonl")
	content, err := os.ReadFile(jobEventsPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", jobEventsPath, err)
	}
	rewritten := strings.Builder{}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `"sourceKind":"region-screenshot"`) && strings.Contains(line, `"status":"ready"`) {
			var event ocrJobEvidenceEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				t.Fatalf("Unmarshal(job event) error = %v", err)
			}
			event.ResultID = "different-result"
			if event.Result != nil {
				event.Result.ID = "different-result"
			}
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Marshal(job event) error = %v", err)
			}
			line = string(data)
		}
		if strings.TrimSpace(line) != "" {
			rewritten.WriteString(line)
			rewritten.WriteByte('\n')
		}
	}
	writeTextFile(t, jobEventsPath, rewritten.String())

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want job-event/result chain mismatch")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "evidence chain" && strings.Contains(check.Message, "job-events ready sourceKind=region-screenshot sourceId=source-region-screenshot resultId=result-region-screenshot") {
			found = true
		}
	}
	if !found {
		t.Fatalf("job-event/result chain mismatch not found: %#v", report.Checks)
	}
}

func TestRunRejectsUnexpectedScopedJobEvent(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	jobEventsPath := filepath.Join(root, "ocr-job-events.jsonl")
	content, err := os.ReadFile(jobEventsPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", jobEventsPath, err)
	}
	content = append(content, []byte(`{"event":"ocr.job.finished","sourceKind":"region-screenshot","sourceId":"source-region-screenshot","status":"ready","resultId":"unexpected-result"}`+"\n")...)
	writeTextFile(t, jobEventsPath, string(content))

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want unexpected scoped job event failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "evidence chain" && strings.Contains(check.Message, "unexpected resultId=unexpected-result") {
			found = true
		}
	}
	if !found {
		t.Fatalf("unexpected scoped job event failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsDuplicateOCRResultSourceKind(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	resultPath := filepath.Join(root, "results", string(ocr.SourceRegionScreenshot)+".json")
	var result ocr.Result
	readJSONFile(t, resultPath, &result)
	result.ID = "result-region-screenshot-duplicate"
	writeJSONFile(t, filepath.Join(root, "results", "region-screenshot-duplicate.json"), result)

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want duplicate result source kind failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "evidence chain" && strings.Contains(check.Message, "duplicate OCR result sourceKind region-screenshot") {
			found = true
		}
	}
	if !found {
		t.Fatalf("duplicate result source kind failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingOCROperationAppLog(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	writeTextFile(t, filepath.Join(root, "app-log.jsonl"),
		`{"component":"app","event":"startup","fields":{"platform":"windows"}}`+"\n"+
			`{"component":"floating-panel","event":"show","fields":{"kind":"ocr-result","contextId":"result-region-screenshot"}}`+"\n")

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing OCR operation app-log failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "app-log.jsonl" &&
			strings.Contains(check.Message, "ocr/queue-request sourceKind=region-screenshot") &&
			strings.Contains(check.Message, "ocr/open-result sourceKind=region-screenshot") &&
			strings.Contains(check.Message, "ocr/read-result-image sourceKind=region-screenshot") &&
			strings.Contains(check.Message, "client.ocr-result/preview-loaded sourceKind=region-screenshot") &&
			strings.Contains(check.Message, "client.ocr-result/rendered sourceKind=region-screenshot") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing OCR operation log failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingPerSourceResultReadLog(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	appLogPath := filepath.Join(root, "app-log.jsonl")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	filtered := strings.Builder{}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `"event":"read-result-image"`) && strings.Contains(line, `"sourceKind":"whiteboard-selection"`) {
			continue
		}
		if strings.TrimSpace(line) != "" {
			filtered.WriteString(line)
			filtered.WriteByte('\n')
		}
	}
	writeTextFile(t, appLogPath, filtered.String())

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing per-source read-result-image failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "app-log.jsonl" && strings.Contains(check.Message, "ocr/read-result-image sourceKind=whiteboard-selection") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing per-source read-result-image failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingRecordingAnnotationOperationLog(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	appLogPath := filepath.Join(root, "app-log.jsonl")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	filtered := strings.Builder{}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `"component":"annotation-overlay"`) {
			continue
		}
		if strings.Contains(line, `"sourceKind":"whiteboard"`) && strings.Contains(line, `"priority":"background"`) {
			continue
		}
		if strings.TrimSpace(line) != "" {
			filtered.WriteString(line)
			filtered.WriteByte('\n')
		}
	}
	writeTextFile(t, appLogPath, filtered.String())

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing recording annotation app-log failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "app-log.jsonl" &&
			strings.Contains(check.Message, "annotation-overlay/show packageDir") &&
			strings.Contains(check.Message, "annotation-overlay/save-capture packageDir bytes") &&
			strings.Contains(check.Message, "ocr/queue-request sourceKind=whiteboard priority=background sourceId") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing recording annotation operation log failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsRecordingAnnotationQueueNotTiedToSavedPackage(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	appLogPath := filepath.Join(root, "app-log.jsonl")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	replaced := strings.Replace(string(content), `"sourceId":"test-recording-package","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","priority":"background"`, `"sourceId":"different-package","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","priority":"background"`, 1)
	writeTextFile(t, appLogPath, replaced)

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want mismatched recording annotation app-log failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "app-log.jsonl" && strings.Contains(check.Message, "sourceId matching annotation-overlay/save-capture packageDir") {
			found = true
		}
	}
	if !found {
		t.Fatalf("mismatched recording annotation sourceId failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsKnownFailures(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	readmePath := filepath.Join(root, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", readmePath, err)
	}
	writeTextFile(t, readmePath, strings.Replace(string(content), "known failures: none", "known failures: whiteboard OCR blocked", 1))

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want known failures rejection")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "README.md" && strings.Contains(check.Message, "known failures must be none") {
			found = true
		}
	}
	if !found {
		t.Fatalf("known failures rejection not found: %#v", report.Checks)
	}
}

func TestRunRejectsTamperedVisualManifest(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	manifestPath := filepath.Join(root, "visual", "visual-manifest.json")
	var manifest visualManifest
	readJSONFile(t, manifestPath, &manifest)
	manifest.Files[0].SHA256 = strings.Repeat("0", 64)
	writeJSONFile(t, manifestPath, manifest)

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want visual manifest sha256 failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "visual evidence" && strings.Contains(check.Message, "sha256 mismatch") {
			found = true
		}
	}
	if !found {
		t.Fatalf("visual manifest mismatch failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMismatchedExportReport(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	reportPath := filepath.Join(root, "export-report.json")
	var exported exportReport
	readJSONFile(t, reportPath, &exported)
	exported.ResultCount--
	writeJSONFile(t, reportPath, exported)

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want export-report mismatch failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "export-report.json" && strings.Contains(check.Message, "resultCount") {
			found = true
		}
	}
	if !found {
		t.Fatalf("export-report mismatch failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingVisualRequirement(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	visualPath := filepath.Join(root, "visual", "recording-annotation-ocr-safety.png")
	if err := os.Remove(visualPath); err != nil {
		t.Fatalf("Remove(%s) error = %v", visualPath, err)
	}
	manifestPath := filepath.Join(root, "visual", "visual-manifest.json")
	var manifest visualManifest
	readJSONFile(t, manifestPath, &manifest)
	filtered := manifest.Files[:0]
	for _, entry := range manifest.Files {
		if entry.Path != "recording-annotation-ocr-safety.png" {
			filtered = append(filtered, entry)
		}
	}
	manifest.Files = filtered
	writeJSONFile(t, manifestPath, manifest)

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing visual requirement failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "visual evidence" && strings.Contains(check.Message, "recording annotation OCR safety") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing visual requirement failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsTooSmallVisualRequirement(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	visualPath := filepath.Join(root, "visual", "ocr-result-floating-panel.png")
	writePNG(t, visualPath, 120, 80)
	writeVisualManifest(t, filepath.Join(root, "visual"), fixtureVisualNames())

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want too-small visual requirement failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "visual evidence" && strings.Contains(check.Message, "requires at least") {
			found = true
		}
	}
	if !found {
		t.Fatalf("too-small visual requirement failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingVisualCaptureChecklist(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	if err := os.Remove(filepath.Join(root, "visual-capture-checklist.json")); err != nil {
		t.Fatalf("Remove(visual-capture-checklist.json) error = %v", err)
	}

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing visual capture checklist failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "visual-capture-checklist" && strings.Contains(check.Message, "visual-capture-checklist.json") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing visual capture checklist failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingDataRootPrecheck(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	if err := os.Remove(filepath.Join(root, "data-root-precheck.json")); err != nil {
		t.Fatalf("Remove(data-root-precheck.json) error = %v", err)
	}

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing data-root-precheck failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "data-root-precheck" && strings.Contains(check.Message, "data-root-precheck.json") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing data-root-precheck failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsAppLogOutsideDataRootPrecheckWindow(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	appLogPath := filepath.Join(root, "app-log.jsonl")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	outside := checkerAppLogLine(time.Date(2026, 7, 5, 8, 0, 0, 0, time.UTC), "ocr", "open-result", `"resultId":"result-region-screenshot","sourceKind":"region-screenshot","sourceId":"source-region-screenshot"`)
	writeTextFile(t, appLogPath, string(content)+outside)

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want app-log run window failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "app-log run window" && strings.Contains(check.Message, "outside data-root precheck app event window") {
			found = true
		}
	}
	if !found {
		t.Fatalf("app-log run window failure not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingEvidenceSessionMarkers(t *testing.T) {
	root := createDesktopOCREvidence(t, nil)
	appLogPath := filepath.Join(root, "app-log.jsonl")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	lines := []string{}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `"component":"ocr-desktop-evidence"`) {
			continue
		}
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	writeTextFile(t, appLogPath, strings.Join(lines, "\n")+"\n")

	report, err := run(options{evidenceDir: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want missing session marker failure")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "desktop evidence session" && strings.Contains(check.Message, "session-start") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing session marker failure not found: %#v", report.Checks)
	}
}

func createDesktopOCREvidence(t *testing.T, omit map[ocr.SourceKind]bool) string {
	t.Helper()
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "results"))
	mkdirAll(t, filepath.Join(root, "images"))
	mkdirAll(t, filepath.Join(root, "visual"))
	mkdirAll(t, filepath.Join(root, "translations"))

	readme := `RecordingFreedom desktop Wails OCR evidence
version: test
commit: abc123
artifact: GitHub Actions release artifact
known failures: none
region-screenshot
full-screenshot
scrolling-screenshot
pinned-screenshot
whiteboard
whiteboard-selection
文字识别 OCR
	`
	writeTextFile(t, filepath.Join(root, "README.md"), readme)
	writeTextFile(t, filepath.Join(root, "platform.txt"), "operating system: windows\nversion: 11 build test\ndisplay count: 2\nresolution: 1920x1080\nscale: 150%\n")
	createdAt := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	appLog := strings.Builder{}
	appLog.WriteString(checkerAppLogLine(createdAt.Add(-time.Minute), "ocr-desktop-evidence", "session-start", `"sessionId":"checker-session"`))
	appLog.WriteString(checkerAppLogLine(createdAt, "app", "startup", `"platform":"windows"`))
	appLog.WriteString(checkerAppLogLine(createdAt, "floating-panel", "show", `"kind":"ocr-result","contextId":"result-region-screenshot"`))
	appLog.WriteString(checkerAppLogLine(createdAt, "annotation-overlay", "show", `"packageDir":"test-recording-package","targetType":"screen","targetId":"screen-1"`))
	appLog.WriteString(checkerAppLogLine(createdAt, "annotation-overlay", "save-capture", `"packageDir":"test-recording-package","bytes":"4096"`))
	for _, kind := range ocrevidence.RequiredSourceKinds {
		if omit[kind] {
			continue
		}
		appLog.WriteString(checkerAppLogLine(createdAt, "ocr", "queue-request", `"sourceKind":"`+string(kind)+`","sourceId":"source-`+string(kind)+`","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","priority":"interactive","force":"false"`))
	}
	appLog.WriteString(checkerAppLogLine(createdAt, "ocr", "queue-request", `"sourceKind":"whiteboard","sourceId":"test-recording-package","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","priority":"background","force":"false"`))
	for _, kind := range ocrevidence.RequiredSourceKinds {
		if omit[kind] {
			continue
		}
		key := string(kind)
		appLog.WriteString(checkerAppLogLine(createdAt, "ocr", "open-result", `"resultId":"result-`+key+`","sourceKind":"`+key+`","sourceId":"source-`+key+`","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","blockCount":"1"`))
		appLog.WriteString(checkerAppLogLine(createdAt, "ocr", "read-result-image", `"resultId":"result-`+key+`","sourceKind":"`+key+`","sourceId":"source-`+key+`","language":"zh-en","modelId":"ppocrv5-mobile-zh-en","bytes":"192"`))
		appLog.WriteString(checkerAppLogLine(createdAt, "client.ocr-result", "preview-loaded", `"resultId":"result-`+key+`","sourceKind":"`+key+`","sourceId":"source-`+key+`","width":"120","height":"48","blockCount":"1","available":"true","bytes":"192"`))
		appLog.WriteString(checkerAppLogLine(createdAt, "client.ocr-result", "rendered", `"resultId":"result-`+key+`","sourceKind":"`+key+`","sourceId":"source-`+key+`","width":"120","height":"48","blockCount":"1","polygonCount":"1","hasPreview":"true"`))
	}
	appLog.WriteString(checkerAppLogLine(createdAt, "ocr", "translate-request", `"ocrResultId":"result-region-screenshot","provider":"openai-compatible","sourceLanguage":"zh-en","targetLanguage":"en","model":"rf-translator","blockCount":"0","force":"false"`))
	appLog.WriteString(checkerAppLogLine(createdAt, "ocr", "translate-ready", `"ocrResultId":"result-region-screenshot","provider":"openai-compatible","sourceLanguage":"zh-en","targetLanguage":"en","model":"rf-translator","blockCount":"1"`))
	appLog.WriteString(checkerAppLogLine(createdAt.Add(time.Minute), "ocr-desktop-evidence", "session-end", `"sessionId":"checker-session"`))
	writeTextFile(t, filepath.Join(root, "app-log.jsonl"), appLog.String())

	jobEvents := strings.Builder{}
	sourceExports := []sourceExport{}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		if omit[kind] {
			continue
		}
		result := makeOCRResult(t, root, kind)
		result.CreatedAt = createdAt
		writeJSONFile(t, filepath.Join(root, "results", string(kind)+".json"), result)
		sourceExports = append(sourceExports, sourceExport{
			SourceKind: string(kind),
			ResultID:   result.ID,
			ImagePath:  result.ImagePath,
			ResultPath: filepath.ToSlash(filepath.Join("results", string(kind)+".json")),
		})
		jobEvents.WriteString(`{"sourceKind":"` + string(kind) + `","sourceId":"` + result.SourceID + `","status":"queued"}` + "\n")
		ready := ocrJobEvidenceEvent{
			SourceKind: string(kind),
			SourceID:   result.SourceID,
			Status:     ocr.ResultStatusReady,
			ResultID:   result.ID,
			Result:     &result,
		}
		data, err := json.Marshal(ready)
		if err != nil {
			t.Fatalf("Marshal(ready) error = %v", err)
		}
		jobEvents.Write(data)
		jobEvents.WriteByte('\n')
	}
	writeTextFile(t, filepath.Join(root, "ocr-job-events.jsonl"), jobEvents.String())

	visualNames := fixtureVisualNames()
	for _, name := range visualNames {
		writePNG(t, filepath.Join(root, "visual", name), 800, 520)
	}
	writeVisualManifest(t, filepath.Join(root, "visual"), visualNames)
	visualRequirementMatches, err := ocrevidence.MatchVisualRequirements(lowerFixtureNames(visualNames))
	if err != nil {
		t.Fatalf("matchVisualRequirements() error = %v", err)
	}
	precheck := makeFixtureDataRootPrecheck(root, sourceExports, countFixtureLines(appLog.String()), countFixtureLines(jobEvents.String()), omit)
	writeJSONFile(t, filepath.Join(root, "data-root-precheck.json"), precheck)
	writeDesktopChecklist(t, root, visualNames, precheck)
	writeJSONFile(t, filepath.Join(root, "translations", "translation.json"), ocr.TranslationResult{
		OcrResultID:    "result-region-screenshot",
		Provider:       "openai-compatible",
		SourceLanguage: "zh-en",
		TargetLanguage: "en",
		Blocks: []ocr.TranslationBlock{{
			BlockID:    "block-1",
			Source:     "文字识别",
			Translated: "Text recognition",
		}},
		CreatedAt: time.Now().UTC(),
	})
	writeJSONFile(t, filepath.Join(root, "export-report.json"), exportReport{
		OK:                 true,
		GeneratedAt:        time.Now().UTC(),
		DataRoot:           filepath.Join(root, "data-root"),
		EvidenceDir:        root,
		VisualDir:          filepath.Join(root, "visual-source"),
		ResultCount:        len(sourceExports),
		SourceKinds:        sourceExports,
		AppLogLines:        countFixtureLines(appLog.String()),
		JobEventLines:      countFixtureLines(jobEvents.String()),
		VisualFiles:        len(visualNames),
		VisualManifest:     "visual/visual-manifest.json",
		VisualRequirements: visualRequirementMatches,
		ChecklistMarkdown:  "visual-capture-checklist.md",
		ChecklistJSON:      "visual-capture-checklist.json",
		DataRootPrecheck:   "data-root-precheck.json",
		TranslationFiles:   1,
	})
	return root
}

func fixtureVisualNames() []string {
	names := make([]string, 0, len(ocrevidence.RequiredVisualEvidence))
	for _, requirement := range ocrevidence.RequiredVisualEvidence {
		names = append(names, requirement.RecommendedFile)
	}
	return names
}

func makeFixtureDataRootPrecheck(root string, sourceExports []sourceExport, appLogLines int, jobEventLines int, omit map[ocr.SourceKind]bool) ocrevidence.DataRootPrecheckReport {
	createdAt := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	precheck := ocrevidence.DataRootPrecheckReport{
		DataRoot:      filepath.Join(root, "data-root"),
		CheckComplete: len(omit) == 0,
		AppLogLines:   appLogLines,
		JobEventLines: jobEventLines,
		ResultFiles:   len(sourceExports),
		Session: ocrevidence.DataRootPrecheckSession{
			SessionID:       "checker-session",
			Start:           createdAt.Add(-time.Minute),
			End:             createdAt.Add(time.Minute),
			DurationSeconds: 120,
			StartLog:        filepath.Join(root, "app-log.jsonl"),
			EndLog:          filepath.Join(root, "app-log.jsonl"),
		},
		RunWindow: ocrevidence.DataRootPrecheckRunWindow{
			ResultStart:            createdAt,
			ResultEnd:              createdAt,
			MaxSpanSeconds:         21600,
			AppEventStart:          createdAt.Add(-time.Minute),
			AppEventEnd:            createdAt.Add(time.Minute),
			AppEventPaddingSeconds: 7200,
		},
		Annotation: ocrevidence.DataRootAnnotationPrecheck{
			ShowPackageDir:         true,
			SaveCapturePackageDirs: []string{"test-recording-package"},
			BackgroundQueueSources: []string{"test-recording-package"},
			MatchingPackage:        true,
		},
		Files: ocrevidence.DataRootPrecheckFiles{
			AppLogs:     []string{filepath.Join(root, "app-log.jsonl")},
			JobEvents:   filepath.Join(root, "ocr-job-events.jsonl"),
			ResultsRoot: filepath.Join(root, "results"),
		},
	}
	byKind := map[string]sourceExport{}
	for _, source := range sourceExports {
		byKind[source.SourceKind] = source
	}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		exported, ok := byKind[string(kind)]
		source := ocrevidence.DataRootPrecheckSource{
			SourceKind: string(kind),
		}
		if ok {
			source.SourceID = "source-" + string(kind)
			source.ResultID = exported.ResultID
			source.ResultCreatedAt = createdAt
			source.ResultReady = true
			source.ImageReady = true
			source.JobQueued = true
			source.JobReady = true
			source.AppQueueRequest = true
			source.AppOpenResult = true
			source.AppReadResultImage = true
			source.ClientPreview = true
			source.ClientRendered = true
		} else {
			source.Missing = []string{"missing OCR result sourceKind=" + string(kind)}
			precheck.MissingRequirements = append(precheck.MissingRequirements, source.Missing...)
		}
		precheck.Sources = append(precheck.Sources, source)
	}
	return precheck
}

func writeDesktopChecklist(t *testing.T, root string, visualNames []string, precheck ocrevidence.DataRootPrecheckReport) {
	t.Helper()
	dimensions := make([]ocrevidence.VisualFileDimension, 0, len(visualNames))
	for _, name := range visualNames {
		path := filepath.Join(root, "visual", name)
		width, height, err := imageSize(path)
		if err != nil {
			t.Fatalf("imageSize(%s) error = %v", path, err)
		}
		dimensions = append(dimensions, ocrevidence.VisualFileDimension{
			Path:   name,
			Width:  width,
			Height: height,
		})
	}
	report := ocrevidence.NewChecklistReportWithDimensions(time.Now().UTC(), filepath.Join(root, "visual-source"), "", lowerFixtureNames(visualNames), dimensions)
	report.OutputDir = "."
	report.MarkdownChecklistPath = "visual-capture-checklist.md"
	report.JSONChecklistPath = "visual-capture-checklist.json"
	report.DataRootPrecheck = &precheck
	if !precheck.CheckComplete {
		report.CheckComplete = false
	}
	if len(report.MissingVisualRequirements) > 0 || len(report.VisualDimensionFailures) > 0 {
		t.Fatalf("visual checklist incomplete: missing=%#v dimensions=%#v", report.MissingVisualRequirements, report.VisualDimensionFailures)
	}
	writeTextFile(t, filepath.Join(root, "visual-capture-checklist.md"), ocrevidence.MarkdownChecklist(report))
	writeJSONFile(t, filepath.Join(root, "visual-capture-checklist.json"), report)
}

func lowerFixtureNames(names []string) []string {
	values := make([]string, 0, len(names))
	for _, name := range names {
		values = append(values, strings.ToLower(filepath.ToSlash(name)))
	}
	return values
}

func countFixtureLines(value string) int {
	count := 0
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func checkerAppLogLine(timestamp time.Time, component string, event string, fields string) string {
	return `{"timestamp":"` + timestamp.Format(time.RFC3339Nano) + `","component":"` + component + `","event":"` + event + `","fields":{` + fields + `}}` + "\n"
}

func writeVisualManifest(t *testing.T, root string, names []string) {
	t.Helper()
	manifest := visualManifest{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC(),
		Files:         make([]visualManifestEntry, 0, len(names)),
	}
	for _, name := range names {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%s) error = %v", path, err)
		}
		width, height, err := imageSize(path)
		if err != nil {
			t.Fatalf("imageSize(%s) error = %v", path, err)
		}
		sum, err := fileSHA256(path)
		if err != nil {
			t.Fatalf("fileSHA256(%s) error = %v", path, err)
		}
		manifest.Files = append(manifest.Files, visualManifestEntry{
			Path:   filepath.ToSlash(name),
			Bytes:  info.Size(),
			SHA256: sum,
			Width:  width,
			Height: height,
		})
	}
	writeJSONFile(t, filepath.Join(root, "visual-manifest.json"), manifest)
}

func makeOCRResult(t *testing.T, root string, kind ocr.SourceKind) ocr.Result {
	t.Helper()
	imageRel := filepath.ToSlash(filepath.Join("images", string(kind)+".png"))
	imagePath := filepath.Join(root, filepath.FromSlash(imageRel))
	writePNG(t, imagePath, 120, 48)
	return ocr.Result{
		ID:          "result-" + string(kind),
		SourceKind:  kind,
		SourceID:    "source-" + string(kind),
		ImagePath:   imageRel,
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
		CreatedAt:  time.Now().UTC(),
		DurationMS: 120,
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
}

func writeTextFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
}

func writePNG(t *testing.T, path string, width int, height int) {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
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
