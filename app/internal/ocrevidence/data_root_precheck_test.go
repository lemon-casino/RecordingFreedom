package ocrevidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

func TestAuditDataRootAcceptsCompleteDesktopOCRChain(t *testing.T) {
	root := t.TempDir()
	writeCompleteDataRootFixture(t, root, "")

	report, err := AuditDataRoot(root)
	if err != nil {
		t.Fatalf("AuditDataRoot() error = %v", err)
	}
	if !report.CheckComplete {
		t.Fatalf("CheckComplete = false, missing = %#v", report.MissingRequirements)
	}
	if len(report.Sources) != len(RequiredSourceKinds) {
		t.Fatalf("sources = %d, want %d", len(report.Sources), len(RequiredSourceKinds))
	}
	if !report.Annotation.MatchingPackage {
		t.Fatalf("annotation = %#v, want matching package", report.Annotation)
	}
}

func TestAuditDataRootReportsMissingPerSourceClientRender(t *testing.T) {
	root := t.TempDir()
	writeCompleteDataRootFixture(t, root, string(ocr.SourceWhiteboardSelection))

	report, err := AuditDataRoot(root)
	if err != nil {
		t.Fatalf("AuditDataRoot() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want missing whiteboard-selection render")
	}
	joined := strings.Join(report.MissingRequirements, "\n")
	if !strings.Contains(joined, "missing client rendered sourceKind=whiteboard-selection") {
		t.Fatalf("missing requirements = %#v, want whiteboard-selection client render failure", report.MissingRequirements)
	}
}

func TestAuditDataRootReportsRecordingAnnotationMismatch(t *testing.T) {
	root := t.TempDir()
	writeCompleteDataRootFixture(t, root, "")
	appLogPath := filepath.Join(root, "logs", "recordingfreedom-test.log")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	replaced := strings.Replace(string(content), `"sourceId":"test-recording-package"`, `"sourceId":"other-package"`, 1)
	if err := os.WriteFile(appLogPath, []byte(replaced), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", appLogPath, err)
	}

	report, err := AuditDataRoot(root)
	if err != nil {
		t.Fatalf("AuditDataRoot() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want annotation package mismatch")
	}
	if !strings.Contains(strings.Join(report.Annotation.Missing, "\n"), "matching save-capture packageDir") {
		t.Fatalf("annotation missing = %#v, want matching package failure", report.Annotation.Missing)
	}
}

func TestAuditDataRootRejectsMixedHistoricalResultRuns(t *testing.T) {
	root := t.TempDir()
	createdAt := writeCompleteDataRootFixture(t, root, "")
	resultPath := filepath.Join(root, "data", "ocr", "results", string(ocr.SourceRegionScreenshot)+".json")
	var result ocr.Result
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", resultPath, err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", resultPath, err)
	}
	result.CreatedAt = createdAt.Add(-24 * time.Hour)
	writeDataRootJSON(t, resultPath, result)

	report, err := AuditDataRoot(root)
	if err != nil {
		t.Fatalf("AuditDataRoot() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want mixed historical run rejection")
	}
	if !strings.Contains(strings.Join(report.MissingRequirements, "\n"), "OCR result createdAt span exceeds evidence window max") {
		t.Fatalf("missing requirements = %#v, want createdAt span failure", report.MissingRequirements)
	}
}

func TestAuditDataRootRejectsAppLogEventsWithoutTimestamps(t *testing.T) {
	root := t.TempDir()
	writeCompleteDataRootFixture(t, root, "")
	appLogPath := filepath.Join(root, "logs", "recordingfreedom-test.log")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	replaced := strings.Replace(string(content), `"timestamp":"2026-07-06T08:00:00Z","component":"ocr","event":"queue-request"`, `"component":"ocr","event":"queue-request"`, 1)
	if err := os.WriteFile(appLogPath, []byte(replaced), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", appLogPath, err)
	}

	report, err := AuditDataRoot(root)
	if err != nil {
		t.Fatalf("AuditDataRoot() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want missing app-log timestamp rejection")
	}
	if !strings.Contains(strings.Join(report.MissingRequirements, "\n"), "missing app-log timestamp") {
		t.Fatalf("missing requirements = %#v, want timestamp failure", report.MissingRequirements)
	}
}

func TestAuditDataRootRejectsMissingEvidenceSession(t *testing.T) {
	root := t.TempDir()
	writeCompleteDataRootFixture(t, root, "")
	appLogPath := filepath.Join(root, "logs", "recordingfreedom-test.log")
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
	if err := os.WriteFile(appLogPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", appLogPath, err)
	}

	report, err := AuditDataRoot(root)
	if err != nil {
		t.Fatalf("AuditDataRoot() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want missing evidence session rejection")
	}
	if !strings.Contains(strings.Join(report.MissingRequirements, "\n"), "missing ocr-desktop-evidence session-start/session-end markers") {
		t.Fatalf("missing requirements = %#v, want missing session failure", report.MissingRequirements)
	}
}

func TestAuditDataRootRejectsEventsOutsideEvidenceSession(t *testing.T) {
	root := t.TempDir()
	writeCompleteDataRootFixture(t, root, "")
	appLogPath := filepath.Join(root, "logs", "recordingfreedom-test.log")
	content, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", appLogPath, err)
	}
	replaced := strings.Replace(string(content), `"timestamp":"2026-07-06T07:59:00Z","component":"ocr-desktop-evidence","event":"session-start"`, `"timestamp":"2026-07-06T08:00:30Z","component":"ocr-desktop-evidence","event":"session-start"`, 1)
	if err := os.WriteFile(appLogPath, []byte(replaced), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", appLogPath, err)
	}

	report, err := AuditDataRoot(root)
	if err != nil {
		t.Fatalf("AuditDataRoot() error = %v", err)
	}
	if report.CheckComplete {
		t.Fatal("CheckComplete = true, want session outside result rejection")
	}
	if !strings.Contains(strings.Join(report.MissingRequirements, "\n"), "session starts after OCR result window") {
		t.Fatalf("missing requirements = %#v, want session start failure", report.MissingRequirements)
	}
}

func writeCompleteDataRootFixture(t *testing.T, root string, omitClientRenderedSourceKind string) time.Time {
	t.Helper()
	createdAt := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	appLog := strings.Builder{}
	appLog.WriteString(dataRootAppLogLine(t, createdAt.Add(-time.Minute), EvidenceSessionComponent, EvidenceSessionStartEvent, map[string]string{"sessionId": "session-test"}))
	appLog.WriteString(dataRootAppLogLine(t, createdAt, "app", "startup", map[string]string{"platform": "windows"}))
	appLog.WriteString(dataRootAppLogLine(t, createdAt, "floating-panel", "show", map[string]string{"kind": "ocr-result", "contextId": "result-region-screenshot"}))
	appLog.WriteString(dataRootAppLogLine(t, createdAt, "annotation-overlay", "show", map[string]string{"packageDir": "test-recording-package"}))
	appLog.WriteString(dataRootAppLogLine(t, createdAt, "annotation-overlay", "save-capture", map[string]string{"packageDir": "test-recording-package", "bytes": "4096"}))
	jobEvents := strings.Builder{}
	for _, kind := range RequiredSourceKinds {
		result := makeDataRootResult(t, root, kind, createdAt)
		writeDataRootJSON(t, filepath.Join(root, "data", "ocr", "results", string(kind)+".json"), result)
		key := string(kind)
		appLog.WriteString(dataRootAppLogLine(t, createdAt, "ocr", "queue-request", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "priority": "interactive"}))
		appLog.WriteString(dataRootAppLogLine(t, createdAt, "ocr", "open-result", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID}))
		appLog.WriteString(dataRootAppLogLine(t, createdAt, "ocr", "read-result-image", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID, "bytes": "128"}))
		appLog.WriteString(dataRootAppLogLine(t, createdAt, "client.ocr-result", "preview-loaded", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID, "available": "true"}))
		if key != omitClientRenderedSourceKind {
			appLog.WriteString(dataRootAppLogLine(t, createdAt, "client.ocr-result", "rendered", map[string]string{"sourceKind": key, "sourceId": result.SourceID, "resultId": result.ID, "hasPreview": "true"}))
		}
		jobEvents.WriteString(`{"sourceKind":"` + key + `","sourceId":"` + result.SourceID + `","status":"queued"}` + "\n")
		ready := dataRootJobEvent{
			SourceKind: key,
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
	appLog.WriteString(dataRootAppLogLine(t, createdAt, "ocr", "queue-request", map[string]string{"sourceKind": "whiteboard", "sourceId": "test-recording-package", "priority": "background"}))
	appLog.WriteString(dataRootAppLogLine(t, createdAt.Add(time.Minute), EvidenceSessionComponent, EvidenceSessionEndEvent, map[string]string{"sessionId": "session-test"}))
	writeDataRootText(t, filepath.Join(root, "logs", "recordingfreedom-test.log"), appLog.String())
	writeDataRootText(t, filepath.Join(root, "data", "ocr", "evidence", "ocr-job-events.jsonl"), jobEvents.String())
	return createdAt
}

func makeDataRootResult(t *testing.T, root string, kind ocr.SourceKind, createdAt time.Time) ocr.Result {
	t.Helper()
	imagePath := filepath.Join(root, "data", "ocr", "source-images", string(kind)+".png")
	writeDataRootText(t, imagePath, "png bytes")
	return ocr.Result{
		ID:         "result-" + string(kind),
		SourceKind: kind,
		SourceID:   "source-" + string(kind),
		ImagePath:  imagePath,
		ModelID:    "ppocrv5-mobile-zh-en",
		Language:   "zh-en",
		Width:      120,
		Height:     48,
		Blocks: []ocr.Block{{
			ID:         "block-1",
			Text:       "RecordingFreedom 文字识别",
			Confidence: 0.9,
			Box: []ocr.Point{
				{X: 1, Y: 1},
				{X: 100, Y: 1},
				{X: 100, Y: 40},
				{X: 1, Y: 40},
			},
		}},
		PlainText: "RecordingFreedom 文字识别",
		CreatedAt: createdAt,
	}
}

func dataRootAppLogLine(t *testing.T, timestamp time.Time, component string, event string, fields map[string]string) string {
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

func writeDataRootJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(%s) error = %v", path, err)
	}
	writeDataRootText(t, path, string(data)+"\n")
}

func writeDataRootText(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
