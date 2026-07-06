package ocrevidence

import (
	"strings"
	"testing"
)

func TestMatchVisualRequirementsRequiresDistinctWindowAndFocusedWindow(t *testing.T) {
	files := []string{
		"region-screenshot-capture.png",
		"full-screen-capture.png",
		"focused-window-screenshot-capture.png",
		"scrolling-screenshot-capture.png",
		"ocr-result-floating-panel.png",
		"screenshot-history-ready.png",
		"pinned-screenshot-ocr-highlight.png",
		"whiteboard-ocr.png",
		"whiteboard-selection-ocr.png",
		"recording-annotation-ocr-safety.png",
	}
	_, err := MatchVisualRequirements(files)
	if err == nil || !strings.Contains(err.Error(), "window screenshot capture") {
		t.Fatalf("MatchVisualRequirements() error = %v, want ordinary window requirement missing", err)
	}
}

func TestMatchVisualRequirementsAcceptsCompleteChecklist(t *testing.T) {
	files := []string{}
	for _, requirement := range RequiredVisualEvidence {
		files = append(files, requirement.RecommendedFile)
	}
	matches, err := MatchVisualRequirements(files)
	if err != nil {
		t.Fatalf("MatchVisualRequirements() error = %v", err)
	}
	if len(matches) != len(RequiredVisualEvidence) {
		t.Fatalf("matches = %d, want %d", len(matches), len(RequiredVisualEvidence))
	}
	for _, match := range matches {
		if match.MinWidth <= 0 || match.MinHeight <= 0 {
			t.Fatalf("match %s is missing minimum visual dimensions: %#v", match.Name, match)
		}
	}
}

func TestVisualDimensionFailuresRejectsTinyPlaceholderImages(t *testing.T) {
	files := []string{}
	dimensions := []VisualFileDimension{}
	for _, requirement := range RequiredVisualEvidence {
		files = append(files, requirement.RecommendedFile)
		width := requirement.MinWidth
		height := requirement.MinHeight
		if requirement.Name == "OCR result floating panel" {
			width = 120
			height = 80
		}
		dimensions = append(dimensions, VisualFileDimension{
			Path:   requirement.RecommendedFile,
			Width:  width,
			Height: height,
		})
	}
	matches, err := MatchVisualRequirements(files)
	if err != nil {
		t.Fatalf("MatchVisualRequirements() error = %v", err)
	}
	failures := VisualDimensionFailures(matches, dimensions)
	if len(failures) == 0 || !strings.Contains(strings.Join(failures, "\n"), "OCR result floating panel") {
		t.Fatalf("failures = %#v, want OCR result floating panel size failure", failures)
	}
}

func TestVisualFileMatchesRequirementHonorsExclusions(t *testing.T) {
	var windowRequirement VisualRequirement
	for _, requirement := range RequiredVisualEvidence {
		if requirement.Name == "window screenshot capture" {
			windowRequirement = requirement
			break
		}
	}
	if windowRequirement.Name == "" {
		t.Fatal("window screenshot requirement not found")
	}
	if VisualFileMatchesRequirement("focused-window-screenshot-capture.png", windowRequirement) {
		t.Fatal("window screenshot requirement should exclude focused-window")
	}
	if !VisualFileMatchesRequirement("window-screenshot-capture.png", windowRequirement) {
		t.Fatal("ordinary window visual should satisfy ordinary window screenshot requirement")
	}
}

func TestRequiredCaptureStepsCoverDesktopOCRAcceptance(t *testing.T) {
	if len(RequiredCaptureSteps) == 0 {
		t.Fatal("RequiredCaptureSteps is empty")
	}
	byID := map[string]CaptureStep{}
	for _, step := range RequiredCaptureSteps {
		byID[step.ID] = step
		if step.VisualRequirement == "" || step.RecommendedVisualFile == "" || step.Action == "" {
			t.Fatalf("capture step is missing visual/action contract: %#v", step)
		}
		if len(step.Acceptance) == 0 {
			t.Fatalf("capture step %s has no acceptance criteria", step.ID)
		}
	}
	selection := byID["whiteboard-selection"]
	if selection.SourceKind != "whiteboard-selection" {
		t.Fatalf("whiteboard-selection sourceKind = %q", selection.SourceKind)
	}
	for _, needle := range []string{
		"ocr/open-result sourceKind=whiteboard-selection",
		"ocr/read-result-image sourceKind=whiteboard-selection",
		"client.ocr-result/rendered sourceKind=whiteboard-selection",
	} {
		if !containsCaptureStepString(selection.RequiredLogEvents, needle) {
			t.Fatalf("whiteboard-selection required logs missing %q: %#v", needle, selection.RequiredLogEvents)
		}
	}
	annotation := byID["recording-annotation"]
	if !strings.Contains(strings.Join(annotation.Acceptance, " "), "Recording remains active") {
		t.Fatalf("recording annotation acceptance does not require active recording: %#v", annotation.Acceptance)
	}
	for _, needle := range []string{
		"annotation-overlay/show packageDir",
		"annotation-overlay/save-capture packageDir bytes",
		"ocr/queue-request sourceKind=whiteboard priority=background sourceId=packageDir",
	} {
		if !containsCaptureStepString(annotation.RequiredLogEvents, needle) {
			t.Fatalf("recording annotation required logs missing %q: %#v", needle, annotation.RequiredLogEvents)
		}
	}
}

func TestEvidenceChainRequirementsProtectResultIdentity(t *testing.T) {
	joined := strings.Join(EvidenceChainRequirements, " ")
	for _, needle := range []string{"sourceId", "resultId", "No duplicate sourceKind"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("EvidenceChainRequirements missing %q: %#v", needle, EvidenceChainRequirements)
		}
	}
}

func containsCaptureStepString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
