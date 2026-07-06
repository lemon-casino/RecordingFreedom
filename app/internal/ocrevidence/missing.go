package ocrevidence

import (
	"strings"
)

type NextMissingRequirement struct {
	Area                  string `json:"area"`
	SourceKind            string `json:"sourceKind,omitempty"`
	StepID                string `json:"stepId,omitempty"`
	Title                 string `json:"title"`
	Missing               string `json:"missing"`
	RecommendedAction     string `json:"recommendedAction,omitempty"`
	RecommendedVisualFile string `json:"recommendedVisualFile,omitempty"`
}

func BuildNextMissingRequirements(report ChecklistReport) []NextMissingRequirement {
	items := []NextMissingRequirement{}
	visualByName := visualRequirementByName()
	stepByVisual := captureStepByVisualRequirement()
	for _, name := range report.MissingVisualRequirements {
		requirement := visualByName[name]
		step := stepByVisual[name]
		items = append(items, NextMissingRequirement{
			Area:                  "visual",
			SourceKind:            step.SourceKind,
			StepID:                step.ID,
			Title:                 missingTitle(name, step.Title),
			Missing:               "missing visual evidence: " + name,
			RecommendedAction:     missingAction(step.Action, "Capture the real desktop visual scene for "+name+"."),
			RecommendedVisualFile: missingVisualFile(requirement.RecommendedFile, step.RecommendedVisualFile),
		})
	}
	for _, failure := range report.VisualDimensionFailures {
		name := visualRequirementNameFromFailure(failure)
		requirement := visualByName[name]
		step := stepByVisual[name]
		items = append(items, NextMissingRequirement{
			Area:                  "visual-dimensions",
			SourceKind:            step.SourceKind,
			StepID:                step.ID,
			Title:                 missingTitle(name, step.Title),
			Missing:               failure,
			RecommendedAction:     "Retake the visual evidence at or above the required size; do not crop away the OCR panel, image preview, or polygons.",
			RecommendedVisualFile: missingVisualFile(requirement.RecommendedFile, step.RecommendedVisualFile),
		})
	}
	if report.VisualDir != "" && report.DataRootPrecheck == nil {
		items = append(items, NextMissingRequirement{
			Area:              "data-root",
			Title:             "Data root precheck",
			Missing:           "missing data-root precheck",
			RecommendedAction: "Run plan/export/check with -data-root using the same RecordingFreedom data root that was passed to ocr-desktop-evidence-session start/end.",
		})
	}
	if report.DataRootPrecheck != nil {
		items = append(items, BuildDataRootNextMissingRequirements(*report.DataRootPrecheck)...)
	}
	return dedupeNextMissing(items)
}

func BuildDataRootNextMissingRequirements(report DataRootPrecheckReport) []NextMissingRequirement {
	items := []NextMissingRequirement{}
	stepBySource := captureStepBySourceKind()
	seen := map[string]bool{}
	for _, source := range report.Sources {
		step := stepBySource[source.SourceKind]
		for _, missing := range source.Missing {
			item := NextMissingRequirement{
				Area:                  "data-root-source",
				SourceKind:            source.SourceKind,
				StepID:                step.ID,
				Title:                 missingTitle(source.SourceKind, step.Title),
				Missing:               missing,
				RecommendedAction:     missingAction(step.Action, "Repeat the real desktop OCR flow for sourceKind="+source.SourceKind+" inside the active evidence session."),
				RecommendedVisualFile: step.RecommendedVisualFile,
			}
			items = append(items, item)
			seen[missing] = true
		}
	}
	for _, missing := range report.Annotation.Missing {
		step := captureStepByID("recording-annotation")
		items = append(items, NextMissingRequirement{
			Area:                  "recording-annotation",
			SourceKind:            step.SourceKind,
			StepID:                step.ID,
			Title:                 missingTitle("recording annotation OCR safety", step.Title),
			Missing:               missing,
			RecommendedAction:     missingAction(step.Action, "Repeat the recording annotation OCR safety flow while recording is still active."),
			RecommendedVisualFile: step.RecommendedVisualFile,
		})
		seen[missing] = true
	}
	for _, missing := range report.MissingRequirements {
		if seen[missing] {
			continue
		}
		items = append(items, NextMissingRequirement{
			Area:              dataRootMissingArea(missing),
			Title:             dataRootMissingTitle(missing),
			Missing:           missing,
			RecommendedAction: dataRootMissingAction(missing),
		})
	}
	for _, kind := range report.UnexpectedSourceKinds {
		missing := "unexpected OCR result sourceKind: " + kind
		items = append(items, NextMissingRequirement{
			Area:              "data-root-unexpected-source",
			SourceKind:        kind,
			Title:             "Unexpected OCR source kind",
			Missing:           missing,
			RecommendedAction: "Start a fresh evidence session or remove stale OCR results before exporting; A batch evidence must contain only the required user-facing source kinds.",
		})
	}
	return dedupeNextMissing(items)
}

func visualRequirementByName() map[string]VisualRequirement {
	values := map[string]VisualRequirement{}
	for _, requirement := range RequiredVisualEvidence {
		values[requirement.Name] = requirement
	}
	return values
}

func captureStepByVisualRequirement() map[string]CaptureStep {
	values := map[string]CaptureStep{}
	for _, step := range RequiredCaptureSteps {
		values[step.VisualRequirement] = step
	}
	return values
}

func captureStepBySourceKind() map[string]CaptureStep {
	values := map[string]CaptureStep{}
	for _, step := range RequiredCaptureSteps {
		if strings.TrimSpace(step.SourceKind) != "" {
			values[step.SourceKind] = step
		}
	}
	return values
}

func captureStepByID(id string) CaptureStep {
	for _, step := range RequiredCaptureSteps {
		if step.ID == id {
			return step
		}
	}
	return CaptureStep{}
}

func visualRequirementNameFromFailure(failure string) string {
	if before, _, found := strings.Cut(failure, ":"); found {
		return strings.TrimSpace(before)
	}
	return ""
}

func missingTitle(fallback string, title string) string {
	if strings.TrimSpace(title) != "" {
		return title
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "OCR desktop evidence requirement"
}

func missingAction(action string, fallback string) string {
	if strings.TrimSpace(action) != "" {
		return action
	}
	return fallback
}

func missingVisualFile(left string, right string) string {
	if strings.TrimSpace(left) != "" {
		return left
	}
	return right
}

func dataRootMissingArea(missing string) string {
	switch {
	case strings.Contains(missing, "session"):
		return "session"
	case strings.Contains(missing, "app-log"):
		return "app-log"
	case strings.Contains(missing, "job events") || strings.Contains(missing, "ocr-job-events"):
		return "ocr-job-events"
	case strings.Contains(missing, "results"):
		return "ocr-results"
	default:
		return "data-root"
	}
}

func dataRootMissingTitle(missing string) string {
	switch dataRootMissingArea(missing) {
	case "session":
		return "Evidence session boundary"
	case "app-log":
		return "Application log evidence"
	case "ocr-job-events":
		return "OCR job event evidence"
	case "ocr-results":
		return "OCR result files"
	default:
		return "Data root evidence"
	}
}

func dataRootMissingAction(missing string) string {
	switch dataRootMissingArea(missing) {
	case "session":
		return "Start and end the same real desktop run with ocr-desktop-evidence-session; do not add session markers during export."
	case "app-log":
		return "Rerun the desktop app with logging enabled and perform the required OCR actions inside the evidence session."
	case "ocr-job-events":
		return "Queue the required OCR jobs from the desktop app and wait for queued and ready events before ending the session."
	case "ocr-results":
		return "Open each required OCR result from the desktop app so result JSON and source images are persisted before export."
	default:
		return "Repeat the real desktop OCR evidence run using the runbook and export with the same data root."
	}
}

func dedupeNextMissing(values []NextMissingRequirement) []NextMissingRequirement {
	result := make([]NextMissingRequirement, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		key := value.Area + "\x00" + value.SourceKind + "\x00" + value.StepID + "\x00" + value.Missing
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}
