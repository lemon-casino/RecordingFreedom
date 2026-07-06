package ocrevidence

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ChecklistReport struct {
	SchemaVersion             int                      `json:"schemaVersion"`
	GeneratedAt               time.Time                `json:"generatedAt"`
	VisualDir                 string                   `json:"visualDir,omitempty"`
	OutputDir                 string                   `json:"outputDir,omitempty"`
	RequiredSourceKinds       []string                 `json:"requiredSourceKinds"`
	EvidenceSessionRunbook    []string                 `json:"evidenceSessionRunbook"`
	CaptureSteps              []CaptureStep            `json:"captureSteps"`
	EvidenceChainRequirements []string                 `json:"evidenceChainRequirements"`
	VisualRequirements        []VisualRequirement      `json:"visualRequirements"`
	ExistingVisualFiles       []string                 `json:"existingVisualFiles,omitempty"`
	ExistingVisualDimensions  []VisualFileDimension    `json:"existingVisualDimensions,omitempty"`
	MatchedVisualRequirements []VisualRequirementMatch `json:"matchedVisualRequirements,omitempty"`
	MissingVisualRequirements []string                 `json:"missingVisualRequirements,omitempty"`
	VisualDimensionFailures   []string                 `json:"visualDimensionFailures,omitempty"`
	DataRootPrecheck          *DataRootPrecheckReport  `json:"dataRootPrecheck,omitempty"`
	CheckComplete             bool                     `json:"checkComplete"`
	MarkdownChecklistPath     string                   `json:"markdownChecklistPath,omitempty"`
	JSONChecklistPath         string                   `json:"jsonChecklistPath,omitempty"`
}

func NewChecklistReport(generatedAt time.Time, visualDir string, outputDir string, existingVisualFiles []string) ChecklistReport {
	return NewChecklistReportWithDimensions(generatedAt, visualDir, outputDir, existingVisualFiles, nil)
}

func NewChecklistReportWithDimensions(generatedAt time.Time, visualDir string, outputDir string, existingVisualFiles []string, existingVisualDimensions []VisualFileDimension) ChecklistReport {
	report := ChecklistReport{
		SchemaVersion:             1,
		GeneratedAt:               generatedAt,
		RequiredSourceKinds:       RequiredSourceKindStrings(),
		EvidenceSessionRunbook:    append([]string(nil), EvidenceSessionRunbook...),
		CaptureSteps:              append([]CaptureStep(nil), RequiredCaptureSteps...),
		EvidenceChainRequirements: append([]string(nil), EvidenceChainRequirements...),
		VisualRequirements:        append([]VisualRequirement(nil), RequiredVisualEvidence...),
		CheckComplete:             strings.TrimSpace(visualDir) == "",
	}
	if strings.TrimSpace(visualDir) != "" {
		report.VisualDir = visualDir
		report.ExistingVisualFiles = normalizeVisualFileList(existingVisualFiles)
		report.ExistingVisualDimensions = normalizeVisualDimensions(existingVisualDimensions)
		report.MatchedVisualRequirements, report.MissingVisualRequirements = MatchAndMissingVisualRequirements(report.ExistingVisualFiles)
		report.VisualDimensionFailures = VisualDimensionFailures(report.MatchedVisualRequirements, report.ExistingVisualDimensions)
		report.CheckComplete = len(report.MissingVisualRequirements) == 0 && len(report.VisualDimensionFailures) == 0
	}
	if strings.TrimSpace(outputDir) != "" {
		report.OutputDir = outputDir
		report.MarkdownChecklistPath = filepath.Join(outputDir, "visual-capture-checklist.md")
		report.JSONChecklistPath = filepath.Join(outputDir, "visual-capture-checklist.json")
	}
	return report
}

func RequiredSourceKindStrings() []string {
	values := make([]string, 0, len(RequiredSourceKinds))
	for _, kind := range RequiredSourceKinds {
		values = append(values, string(kind))
	}
	return values
}

func MatchAndMissingVisualRequirements(files []string) ([]VisualRequirementMatch, []string) {
	matches := []VisualRequirementMatch{}
	missing := []string{}
	for _, requirement := range RequiredVisualEvidence {
		path, ok := VisualRequirementMatchPath(files, requirement)
		if !ok {
			missing = append(missing, requirement.Name)
			continue
		}
		matches = append(matches, VisualRequirementMatch{
			Name:            requirement.Name,
			Path:            path,
			Terms:           append([]string(nil), requirement.Terms...),
			Exclude:         append([]string(nil), requirement.Exclude...),
			RecommendedFile: requirement.RecommendedFile,
			Description:     requirement.Description,
			MinWidth:        requirement.MinWidth,
			MinHeight:       requirement.MinHeight,
		})
	}
	return matches, missing
}

func MarkdownChecklist(report ChecklistReport) string {
	var builder strings.Builder
	builder.WriteString("# RecordingFreedom OCR desktop visual evidence checklist\n\n")
	builder.WriteString("This checklist is for real Wails desktop OCR evidence. Do not create placeholder images. Capture each scene from a real app run, then pass the visual directory to `ocr-desktop-evidence-export`.\n\n")
	if report.VisualDir != "" {
		builder.WriteString("- visual-dir: `" + report.VisualDir + "`\n")
		if report.CheckComplete {
			builder.WriteString("- precheck: complete\n\n")
		} else {
			builder.WriteString("- precheck: missing required visual evidence\n\n")
		}
	}
	builder.WriteString("## Required source kinds\n\n")
	for _, kind := range report.RequiredSourceKinds {
		builder.WriteString("- `" + kind + "`\n")
	}
	builder.WriteString("\n## Session boundary runbook\n\n")
	builder.WriteString("Start and end markers must wrap the real desktop OCR actions. Save the returned `sessionId` from the start command and reuse it for the end command.\n\n")
	builder.WriteString("Windows packaged tools:\n\n")
	builder.WriteString("```powershell\n")
	builder.WriteString(".\\tools\\ocr-desktop-evidence-session.exe -event start -data-root <DataRoot>\n")
	builder.WriteString(".\\tools\\ocr-desktop-evidence-session.exe -event end -data-root <DataRoot> -session-id <sessionId>\n")
	builder.WriteString("```\n\n")
	builder.WriteString("macOS/Linux packaged tools:\n\n")
	builder.WriteString("```bash\n")
	builder.WriteString("./tools/ocr-desktop-evidence-session -event start -data-root <DataRoot>\n")
	builder.WriteString("./tools/ocr-desktop-evidence-session -event end -data-root <DataRoot> -session-id <sessionId>\n")
	builder.WriteString("```\n\n")
	for _, item := range report.EvidenceSessionRunbook {
		builder.WriteString("- " + item + "\n")
	}
	builder.WriteString("\n## Evidence chain requirements\n\n")
	for _, item := range report.EvidenceChainRequirements {
		builder.WriteString("- " + item + "\n")
	}
	builder.WriteString("\n## Required visual evidence\n\n")
	matched := map[string]string{}
	for _, item := range report.MatchedVisualRequirements {
		matched[item.Name] = item.Path
	}
	for _, requirement := range report.VisualRequirements {
		status := "missing"
		if path := matched[requirement.Name]; path != "" {
			status = "ready: `" + path + "`"
		}
		builder.WriteString("- [ ] `" + requirement.RecommendedFile + "` - " + requirement.Name + " (" + status + ")\n")
		builder.WriteString("  - " + requirement.Description + "\n")
		builder.WriteString("  - terms: `" + strings.Join(requirement.Terms, "`, `") + "`\n")
		if requirement.MinWidth > 0 || requirement.MinHeight > 0 {
			builder.WriteString("  - minimum size: `" + minSizeText(requirement.MinWidth, requirement.MinHeight) + "`\n")
		}
		if len(requirement.Exclude) > 0 {
			builder.WriteString("  - exclude: `" + strings.Join(requirement.Exclude, "`, `") + "`\n")
		}
	}
	builder.WriteString("\n## Capture runbook\n\n")
	for _, step := range report.CaptureSteps {
		builder.WriteString("### " + step.Title + "\n\n")
		builder.WriteString("- id: `" + step.ID + "`\n")
		if step.SourceKind != "" {
			builder.WriteString("- sourceKind: `" + step.SourceKind + "`\n")
		}
		builder.WriteString("- visual requirement: `" + step.VisualRequirement + "`\n")
		builder.WriteString("- recommended file: `" + step.RecommendedVisualFile + "`\n")
		builder.WriteString("- action: " + step.Action + "\n")
		if len(step.RequiredLogEvents) > 0 {
			builder.WriteString("- required log events:\n")
			for _, event := range step.RequiredLogEvents {
				builder.WriteString("  - `" + event + "`\n")
			}
		}
		if len(step.Acceptance) > 0 {
			builder.WriteString("- acceptance:\n")
			for _, item := range step.Acceptance {
				builder.WriteString("  - " + item + "\n")
			}
		}
		builder.WriteString("\n")
	}
	if len(report.MissingVisualRequirements) > 0 {
		builder.WriteString("\n## Missing\n\n")
		for _, name := range report.MissingVisualRequirements {
			builder.WriteString("- " + name + "\n")
		}
	}
	if len(report.VisualDimensionFailures) > 0 {
		builder.WriteString("\n## Invalid Visual Dimensions\n\n")
		for _, failure := range report.VisualDimensionFailures {
			builder.WriteString("- " + failure + "\n")
		}
	}
	if report.DataRootPrecheck != nil {
		writeDataRootPrecheckMarkdown(&builder, *report.DataRootPrecheck)
	}
	return builder.String()
}

func normalizeVisualFileList(files []string) []string {
	values := make([]string, 0, len(files))
	for _, file := range files {
		file = strings.ToLower(filepath.ToSlash(strings.TrimSpace(file)))
		if file != "" {
			values = append(values, file)
		}
	}
	return values
}

func normalizeVisualDimensions(dimensions []VisualFileDimension) []VisualFileDimension {
	values := make([]VisualFileDimension, 0, len(dimensions))
	for _, dimension := range dimensions {
		path := strings.ToLower(filepath.ToSlash(strings.TrimSpace(dimension.Path)))
		if path == "" {
			continue
		}
		values = append(values, VisualFileDimension{
			Path:   path,
			Width:  dimension.Width,
			Height: dimension.Height,
		})
	}
	return values
}

func minSizeText(width int, height int) string {
	if width <= 0 {
		return "height " + intText(height)
	}
	if height <= 0 {
		return "width " + intText(width)
	}
	return intText(width) + "x" + intText(height)
}

func intText(value int) string {
	return strconv.Itoa(value)
}

func writeDataRootPrecheckMarkdown(builder *strings.Builder, report DataRootPrecheckReport) {
	builder.WriteString("\n## Data Root Precheck\n\n")
	builder.WriteString("- data-root: `" + report.DataRoot + "`\n")
	if report.CheckComplete {
		builder.WriteString("- precheck: complete\n")
	} else {
		builder.WriteString("- precheck: missing required OCR runtime evidence\n")
	}
	builder.WriteString("- app log lines: `" + intText(report.AppLogLines) + "`\n")
	builder.WriteString("- OCR job event lines: `" + intText(report.JobEventLines) + "`\n")
	builder.WriteString("- OCR result files: `" + intText(report.ResultFiles) + "`\n\n")
	builder.WriteString("### Evidence run window\n\n")
	builder.WriteString("- session id: `" + report.Session.SessionID + "`\n")
	builder.WriteString("- session window: `" + timeText(report.Session.Start) + "` -> `" + timeText(report.Session.End) + "`\n")
	builder.WriteString("- session duration seconds: `" + strconv.FormatInt(report.Session.DurationSeconds, 10) + "`\n")
	builder.WriteString("- result start: `" + timeText(report.RunWindow.ResultStart) + "`\n")
	builder.WriteString("- result end: `" + timeText(report.RunWindow.ResultEnd) + "`\n")
	builder.WriteString("- result span seconds: `" + strconv.FormatInt(report.RunWindow.ResultSpanSeconds, 10) + "`\n")
	builder.WriteString("- max span seconds: `" + strconv.FormatInt(report.RunWindow.MaxSpanSeconds, 10) + "`\n")
	builder.WriteString("- app event window: `" + timeText(report.RunWindow.AppEventStart) + "` -> `" + timeText(report.RunWindow.AppEventEnd) + "`\n\n")
	builder.WriteString("### Source chain\n\n")
	for _, source := range report.Sources {
		status := "ready"
		if len(source.Missing) > 0 {
			status = "missing"
		}
		builder.WriteString("- `" + source.SourceKind + "` - " + status + "\n")
		builder.WriteString("  - result: `" + source.ResultID + "` source: `" + source.SourceID + "`\n")
		for _, missing := range source.Missing {
			builder.WriteString("  - missing: " + missing + "\n")
		}
	}
	builder.WriteString("\n### Recording annotation chain\n\n")
	annotationStatus := "ready"
	if len(report.Annotation.Missing) > 0 {
		annotationStatus = "missing"
	}
	builder.WriteString("- status: " + annotationStatus + "\n")
	builder.WriteString("- save-capture packages: `" + strings.Join(report.Annotation.SaveCapturePackageDirs, "`, `") + "`\n")
	builder.WriteString("- background queue sources: `" + strings.Join(report.Annotation.BackgroundQueueSources, "`, `") + "`\n")
	for _, missing := range report.Annotation.Missing {
		builder.WriteString("- missing: " + missing + "\n")
	}
	if len(report.MissingRequirements) > 0 {
		builder.WriteString("\n### Missing Data Root Requirements\n\n")
		for _, missing := range report.MissingRequirements {
			builder.WriteString("- " + missing + "\n")
		}
	}
}

func timeText(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
