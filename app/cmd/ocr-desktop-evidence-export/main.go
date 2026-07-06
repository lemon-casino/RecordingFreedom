package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocrevidence"
)

type options struct {
	dataRoot            string
	evidenceDir         string
	visualDir           string
	platformFile        string
	version             string
	commit              string
	artifact            string
	knownFailures       string
	displayCount        string
	displayResolution   string
	displayScale        string
	includeTranslations bool
}

type exportReport struct {
	OK                 bool                                 `json:"ok"`
	GeneratedAt        time.Time                            `json:"generatedAt"`
	DataRoot           string                               `json:"dataRoot"`
	EvidenceDir        string                               `json:"evidenceDir"`
	VisualDir          string                               `json:"visualDir"`
	ResultCount        int                                  `json:"resultCount"`
	SourceKinds        []sourceExport                       `json:"sourceKinds"`
	AppLogLines        int                                  `json:"appLogLines"`
	JobEventLines      int                                  `json:"jobEventLines"`
	VisualFiles        int                                  `json:"visualFiles"`
	VisualManifest     string                               `json:"visualManifest,omitempty"`
	VisualRequirements []ocrevidence.VisualRequirementMatch `json:"visualRequirements,omitempty"`
	ChecklistMarkdown  string                               `json:"checklistMarkdown,omitempty"`
	ChecklistJSON      string                               `json:"checklistJson,omitempty"`
	DataRootPrecheck   string                               `json:"dataRootPrecheck,omitempty"`
	TranslationFiles   int                                  `json:"translationFiles,omitempty"`
}

type sourceExport struct {
	SourceKind string `json:"sourceKind"`
	ResultID   string `json:"resultId"`
	ImagePath  string `json:"imagePath"`
	ResultPath string `json:"resultPath"`
}

type visualManifest struct {
	SchemaVersion int                   `json:"schemaVersion"`
	GeneratedAt   time.Time             `json:"generatedAt"`
	Files         []visualManifestEntry `json:"files"`
}

type visualManifestEntry struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type jobEvidenceEvent struct {
	Event      string      `json:"event,omitempty"`
	SourceKind string      `json:"sourceKind,omitempty"`
	SourceID   string      `json:"sourceId,omitempty"`
	Status     string      `json:"status,omitempty"`
	ResultID   string      `json:"resultId,omitempty"`
	Result     *ocr.Result `json:"result,omitempty"`
}

func main() {
	var opts options
	flag.StringVar(&opts.dataRoot, "data-root", "", "RecordingFreedom data root from the same real desktop evidence session")
	flag.StringVar(&opts.evidenceDir, "evidence-dir", "", "output OCR desktop evidence directory")
	flag.StringVar(&opts.visualDir, "visual-dir", "", "directory containing real desktop visual evidence screenshots")
	flag.StringVar(&opts.platformFile, "platform-file", "", "platform.txt captured during the real desktop run")
	flag.StringVar(&opts.version, "version", "manual", "RecordingFreedom version under test")
	flag.StringVar(&opts.commit, "commit", "unknown", "git commit under test")
	flag.StringVar(&opts.artifact, "artifact", "manual desktop run", "release/actions artifact under test")
	flag.StringVar(&opts.knownFailures, "known-failures", "none", "known failures or blockers observed during the run")
	flag.StringVar(&opts.displayCount, "display-count", "", "display count when no platform-file is provided")
	flag.StringVar(&opts.displayResolution, "display-resolution", "", "display resolution such as 1920x1080 when no platform-file is provided")
	flag.StringVar(&opts.displayScale, "display-scale", "", "display scale/DPI when no platform-file is provided")
	flag.BoolVar(&opts.includeTranslations, "include-translations", true, "copy existing OCR translation JSON files when present")
	flag.Parse()

	report, err := run(opts)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode OCR desktop evidence export report: %v\n", err)
		os.Exit(1)
	}
}

func run(opts options) (exportReport, error) {
	if strings.TrimSpace(opts.evidenceDir) == "" {
		return exportReport{}, errors.New("-evidence-dir is required")
	}
	if strings.TrimSpace(opts.visualDir) == "" {
		return exportReport{}, errors.New("-visual-dir with real desktop screenshots is required")
	}
	dataRoot, err := resolveDataRoot(opts.dataRoot)
	if err != nil {
		return exportReport{}, err
	}
	evidenceDir, err := filepath.Abs(opts.evidenceDir)
	if err != nil {
		return exportReport{}, err
	}
	visualDir, err := filepath.Abs(opts.visualDir)
	if err != nil {
		return exportReport{}, err
	}
	if info, err := os.Stat(visualDir); err != nil {
		return exportReport{}, err
	} else if !info.IsDir() {
		return exportReport{}, fmt.Errorf("visual evidence path %q is not a directory", visualDir)
	}
	if err := os.RemoveAll(evidenceDir); err != nil {
		return exportReport{}, err
	}
	for _, dir := range []string{
		evidenceDir,
		filepath.Join(evidenceDir, "results"),
		filepath.Join(evidenceDir, "images"),
		filepath.Join(evidenceDir, "visual"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return exportReport{}, err
		}
	}

	report := exportReport{
		OK:          true,
		GeneratedAt: time.Now().UTC(),
		DataRoot:    dataRoot,
		EvidenceDir: evidenceDir,
		VisualDir:   visualDir,
	}
	if err := writeREADME(evidenceDir, opts); err != nil {
		return exportReport{}, err
	}
	if err := writePlatform(evidenceDir, opts); err != nil {
		return exportReport{}, err
	}
	dataRootPrecheck, err := ocrevidence.AuditDataRoot(dataRoot)
	if err != nil {
		return exportReport{}, err
	}
	if !dataRootPrecheck.CheckComplete {
		return exportReport{}, fmt.Errorf("data root precheck is incomplete: %s", strings.Join(dataRootPrecheck.MissingRequirements, ", "))
	}
	dataRootPrecheckPath := filepath.Join(evidenceDir, "data-root-precheck.json")
	if err := writeJSON(dataRootPrecheckPath, dataRootPrecheck); err != nil {
		return exportReport{}, err
	}
	report.DataRootPrecheck = "data-root-precheck.json"
	logLines, err := copyAppLogs(dataRoot, filepath.Join(evidenceDir, "app-log.jsonl"), dataRootPrecheck.RunWindow)
	if err != nil {
		return exportReport{}, err
	}
	report.AppLogLines = logLines
	jobLines, err := copyJobEvents(dataRoot, filepath.Join(evidenceDir, "ocr-job-events.jsonl"), dataRootPrecheck)
	if err != nil {
		return exportReport{}, err
	}
	report.JobEventLines = jobLines
	manifest, err := copyVisualEvidence(visualDir, filepath.Join(evidenceDir, "visual"))
	if err != nil {
		return exportReport{}, err
	}
	report.VisualFiles = len(manifest.Files)
	report.VisualManifest = filepath.ToSlash(filepath.Join("visual", "visual-manifest.json"))
	visualRequirements, err := matchVisualRequirements(manifest.Files)
	if err != nil {
		return exportReport{}, err
	}
	report.VisualRequirements = visualRequirements
	checklistMarkdown, checklistJSON, err := writeVisualCaptureChecklist(evidenceDir, visualDir, manifest.Files, &dataRootPrecheck)
	if err != nil {
		return exportReport{}, err
	}
	report.ChecklistMarkdown = checklistMarkdown
	report.ChecklistJSON = checklistJSON
	sourceExports, err := exportResults(dataRoot, evidenceDir)
	if err != nil {
		return exportReport{}, err
	}
	report.SourceKinds = sourceExports
	report.ResultCount = len(sourceExports)
	if opts.includeTranslations {
		count, err := copyTranslations(dataRoot, filepath.Join(evidenceDir, "translations"))
		if err != nil {
			return exportReport{}, err
		}
		report.TranslationFiles = count
	}
	if err := writeJSON(filepath.Join(evidenceDir, "export-report.json"), report); err != nil {
		return exportReport{}, err
	}
	return report, nil
}

func resolveDataRoot(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("-data-root is required; use the same RecordingFreedom data root passed to ocr-desktop-evidence-session start/end")
	}
	return filepath.Abs(value)
}

func writeREADME(evidenceDir string, opts options) error {
	var builder strings.Builder
	builder.WriteString("# RecordingFreedom desktop Wails OCR evidence\n\n")
	builder.WriteString("version: " + fallbackText(opts.version, "manual") + "\n")
	builder.WriteString("commit: " + fallbackText(opts.commit, "unknown") + "\n")
	builder.WriteString("artifact: " + fallbackText(opts.artifact, "manual desktop run") + "\n")
	builder.WriteString("known failures: " + fallbackText(opts.knownFailures, "none") + "\n\n")
	builder.WriteString("This evidence was exported from a real Wails desktop OCR run. It contains OCR results, app logs, OCR job events, copied source images, and visual evidence screenshots.\n\n")
	builder.WriteString("Required source kinds:\n")
	for _, kind := range ocrevidence.RequiredSourceKinds {
		builder.WriteString("- " + string(kind) + "\n")
	}
	builder.WriteString("\nOCR / 文字识别 evidence is expected to be verified by ocr-desktop-evidence-check.\n")
	return os.WriteFile(filepath.Join(evidenceDir, "README.md"), []byte(builder.String()), 0o644)
}

func writeVisualCaptureChecklist(evidenceDir string, visualDir string, entries []visualManifestEntry, dataRootPrecheck *ocrevidence.DataRootPrecheckReport) (string, string, error) {
	files := make([]string, 0, len(entries))
	dimensions := make([]ocrevidence.VisualFileDimension, 0, len(entries))
	for _, entry := range entries {
		files = append(files, entry.Path)
		dimensions = append(dimensions, ocrevidence.VisualFileDimension{
			Path:   entry.Path,
			Width:  entry.Width,
			Height: entry.Height,
		})
	}
	report := ocrevidence.NewChecklistReportWithDimensions(time.Now().UTC(), visualDir, "", files, dimensions)
	report.OutputDir = "."
	report.MarkdownChecklistPath = "visual-capture-checklist.md"
	report.JSONChecklistPath = "visual-capture-checklist.json"
	if dataRootPrecheck != nil {
		report.DataRootPrecheck = dataRootPrecheck
		if !dataRootPrecheck.CheckComplete {
			report.CheckComplete = false
		}
	}
	if !report.CheckComplete {
		missing := append([]string{}, report.MissingVisualRequirements...)
		missing = append(missing, report.VisualDimensionFailures...)
		if dataRootPrecheck != nil {
			missing = append(missing, dataRootPrecheck.MissingRequirements...)
		}
		return "", "", fmt.Errorf("visual capture checklist is incomplete: %s", strings.Join(missing, ", "))
	}
	markdownPath := filepath.Join(evidenceDir, report.MarkdownChecklistPath)
	if err := os.WriteFile(markdownPath, []byte(ocrevidence.MarkdownChecklist(report)), 0o644); err != nil {
		return "", "", err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", "", err
	}
	jsonPath := filepath.Join(evidenceDir, report.JSONChecklistPath)
	if err := os.WriteFile(jsonPath, append(data, '\n'), 0o644); err != nil {
		return "", "", err
	}
	return report.MarkdownChecklistPath, report.JSONChecklistPath, nil
}

func writePlatform(evidenceDir string, opts options) error {
	if strings.TrimSpace(opts.platformFile) != "" {
		source, err := filepath.Abs(opts.platformFile)
		if err != nil {
			return err
		}
		return copyFile(source, filepath.Join(evidenceDir, "platform.txt"), 0o644)
	}
	if strings.TrimSpace(opts.displayCount) == "" || strings.TrimSpace(opts.displayResolution) == "" || strings.TrimSpace(opts.displayScale) == "" {
		return errors.New("-platform-file or all of -display-count, -display-resolution, and -display-scale are required")
	}
	content := strings.Join([]string{
		"operating system: " + runtime.GOOS,
		"architecture: " + runtime.GOARCH,
		"version: " + runtime.Version(),
		"display count: " + strings.TrimSpace(opts.displayCount),
		"resolution: " + strings.TrimSpace(opts.displayResolution),
		"scale: " + strings.TrimSpace(opts.displayScale),
		"",
	}, "\n")
	return os.WriteFile(filepath.Join(evidenceDir, "platform.txt"), []byte(content), 0o644)
}

func copyAppLogs(dataRoot string, target string, window ocrevidence.DataRootPrecheckRunWindow) (int, error) {
	matches, err := filepath.Glob(filepath.Join(dataRoot, "logs", "recordingfreedom-*.log"))
	if err != nil {
		return 0, err
	}
	if len(matches) == 0 {
		return 0, errors.New("no app log files found under data root logs/")
	}
	sort.Strings(matches)
	return copyJSONLLinesInWindow(matches, target, window.AppEventStart, window.AppEventEnd)
}

func copyJobEvents(dataRoot string, target string, precheck ocrevidence.DataRootPrecheckReport) (int, error) {
	source := filepath.Join(dataRoot, "data", "ocr", "evidence", "ocr-job-events.jsonl")
	if _, err := os.Stat(source); err != nil {
		return 0, fmt.Errorf("OCR job events file is required: %w", err)
	}
	return copyScopedJobEvents(source, target, precheck)
}

func copyJSONLLines(sources []string, target string) (int, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	lines := 0
	for _, source := range sources {
		file, err := os.Open(source)
		if err != nil {
			return 0, err
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if _, err := out.WriteString(line + "\n"); err != nil {
				_ = file.Close()
				return 0, err
			}
			lines++
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil {
			return 0, scanErr
		}
		if closeErr != nil {
			return 0, closeErr
		}
	}
	if lines == 0 {
		return 0, fmt.Errorf("%s has no JSONL lines", target)
	}
	return lines, nil
}

func copyScopedJobEvents(source string, target string, precheck ocrevidence.DataRootPrecheckReport) (int, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	required := jobEventRequirements(precheck)
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	file, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	written := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNumber++
		var event jobEvidenceEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return 0, fmt.Errorf("%s line %d is invalid JSON: %w", source, lineNumber, err)
		}
		key, ok := scopedJobEventKey(event)
		if !ok {
			continue
		}
		if !required[key] {
			continue
		}
		if _, err := out.WriteString(line + "\n"); err != nil {
			return 0, err
		}
		delete(required, key)
		written++
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if len(required) > 0 {
		missing := make([]string, 0, len(required))
		for key := range required {
			missing = append(missing, key)
		}
		sort.Strings(missing)
		return 0, fmt.Errorf("OCR job events are missing required scoped events: %s", strings.Join(missing, ", "))
	}
	if written == 0 {
		return 0, fmt.Errorf("%s has no scoped OCR job event lines", target)
	}
	return written, nil
}

func jobEventRequirements(precheck ocrevidence.DataRootPrecheckReport) map[string]bool {
	required := map[string]bool{}
	for _, source := range precheck.Sources {
		sourceKind := strings.TrimSpace(source.SourceKind)
		sourceID := strings.TrimSpace(source.SourceID)
		resultID := strings.TrimSpace(source.ResultID)
		if sourceKind == "" || sourceID == "" || resultID == "" {
			continue
		}
		required[jobEventKey(ocr.ResultStatusQueued, sourceKind, sourceID, "")] = true
		required[jobEventKey(ocr.ResultStatusReady, sourceKind, sourceID, resultID)] = true
	}
	return required
}

func scopedJobEventKey(event jobEvidenceEvent) (string, bool) {
	status := strings.TrimSpace(event.Status)
	switch status {
	case ocr.ResultStatusQueued:
		sourceKind := jobEventSourceKind(event)
		sourceID := jobEventSourceID(event)
		return jobEventKey(status, sourceKind, sourceID, ""), sourceKind != "" && sourceID != ""
	case ocr.ResultStatusReady:
		sourceKind := jobEventSourceKind(event)
		sourceID := jobEventSourceID(event)
		resultID := jobEventResultID(event)
		return jobEventKey(status, sourceKind, sourceID, resultID), sourceKind != "" && sourceID != "" && resultID != ""
	default:
		return "", false
	}
}

func jobEventKey(status string, sourceKind string, sourceID string, resultID string) string {
	return strings.Join([]string{status, sourceKind, sourceID, resultID}, "\x00")
}

func jobEventSourceKind(event jobEvidenceEvent) string {
	sourceKind := strings.TrimSpace(event.SourceKind)
	if sourceKind == "" && event.Result != nil {
		sourceKind = string(event.Result.SourceKind)
	}
	return sourceKind
}

func jobEventSourceID(event jobEvidenceEvent) string {
	sourceID := strings.TrimSpace(event.SourceID)
	if sourceID == "" && event.Result != nil {
		sourceID = strings.TrimSpace(event.Result.SourceID)
	}
	return sourceID
}

func jobEventResultID(event jobEvidenceEvent) string {
	resultID := strings.TrimSpace(event.ResultID)
	if resultID == "" && event.Result != nil {
		resultID = strings.TrimSpace(event.Result.ID)
	}
	return resultID
}

func copyJSONLLinesInWindow(sources []string, target string, start time.Time, end time.Time) (int, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0, errors.New("app-log export requires a valid data-root precheck app event window")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	lines := 0
	for _, source := range sources {
		file, err := os.Open(source)
		if err != nil {
			return 0, err
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNumber := 0
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			lineNumber++
			timestamp, err := appLogLineTimestamp(line)
			if err != nil {
				_ = file.Close()
				return 0, fmt.Errorf("%s line %d: %w", source, lineNumber, err)
			}
			if timestamp.Before(start) || timestamp.After(end) {
				continue
			}
			if _, err := out.WriteString(line + "\n"); err != nil {
				_ = file.Close()
				return 0, err
			}
			lines++
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil {
			return 0, scanErr
		}
		if closeErr != nil {
			return 0, closeErr
		}
	}
	if lines == 0 {
		return 0, fmt.Errorf("%s has no JSONL lines inside data-root precheck app event window", target)
	}
	return lines, nil
}

func appLogLineTimestamp(line string) (time.Time, error) {
	var event struct {
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return time.Time{}, fmt.Errorf("invalid JSON: %w", err)
	}
	timestampRaw := strings.TrimSpace(event.Timestamp)
	if timestampRaw == "" {
		return time.Time{}, errors.New("missing timestamp")
	}
	timestamp, err := time.Parse(time.RFC3339Nano, timestampRaw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", timestampRaw, err)
	}
	return timestamp, nil
}

func exportResults(dataRoot string, evidenceDir string) ([]sourceExport, error) {
	results, err := loadResults(filepath.Join(dataRoot, "data", "ocr", "results"))
	if err != nil {
		return nil, err
	}
	byKind := latestResultsByKind(results)
	exports := make([]sourceExport, 0, len(ocrevidence.RequiredSourceKinds))
	for _, kind := range ocrevidence.RequiredSourceKinds {
		result, ok := byKind[kind]
		if !ok {
			return nil, fmt.Errorf("missing OCR result for source kind %s", kind)
		}
		imageSource, err := managedDataRootFile(dataRoot, result.ImagePath)
		if err != nil {
			return nil, fmt.Errorf("source kind %s image: %w", kind, err)
		}
		imageName := safeEvidenceName(string(kind) + "-" + filepath.Base(imageSource))
		imageTargetRel := filepath.ToSlash(filepath.Join("images", imageName))
		imageTarget := filepath.Join(evidenceDir, filepath.FromSlash(imageTargetRel))
		if err := copyFile(imageSource, imageTarget, 0o644); err != nil {
			return nil, err
		}
		result.ImagePath = imageTargetRel
		resultPathRel := filepath.ToSlash(filepath.Join("results", safeEvidenceName(string(kind)+".json")))
		resultPath := filepath.Join(evidenceDir, filepath.FromSlash(resultPathRel))
		if err := writeJSON(resultPath, result); err != nil {
			return nil, err
		}
		exports = append(exports, sourceExport{
			SourceKind: string(kind),
			ResultID:   result.ID,
			ImagePath:  imageTargetRel,
			ResultPath: resultPathRel,
		})
	}
	return exports, nil
}

func loadResults(dir string) ([]ocr.Result, error) {
	if info, err := os.Stat(dir); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	results := []ocr.Result{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var result ocr.Result
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("%s is not an OCR result JSON: %w", path, err)
		}
		if result.SourceKind != "" {
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("%s contains no OCR result JSON files", dir)
	}
	return results, nil
}

func latestResultsByKind(results []ocr.Result) map[ocr.SourceKind]ocr.Result {
	byKind := map[ocr.SourceKind]ocr.Result{}
	for _, result := range results {
		current, exists := byKind[result.SourceKind]
		if !exists || result.CreatedAt.After(current.CreatedAt) {
			byKind[result.SourceKind] = result
		}
	}
	return byKind
}

func managedDataRootFile(dataRoot string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("path is required")
	}
	target, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(dataRoot)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%q must stay inside data root %q", value, dataRoot)
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%q is a directory", target)
	}
	return target, nil
}

func copyTranslations(dataRoot string, targetDir string) (int, error) {
	sourceDir := filepath.Join(dataRoot, "data", "ocr", "translations")
	if _, err := os.Stat(sourceDir); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return copyDirectoryFiles(sourceDir, targetDir)
}

func copyVisualEvidence(sourceDir string, targetDir string) (visualManifest, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return visualManifest{}, err
	}
	manifest := visualManifest{SchemaVersion: 1, GeneratedAt: time.Now().UTC()}
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if isIgnoredVisualMetadataFile(entry.Name()) {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return fmt.Errorf("unsafe visual evidence path %q", rel)
		}
		targetRel := filepath.ToSlash(rel)
		target := filepath.Join(targetDir, filepath.FromSlash(targetRel))
		if err := copyFile(path, target, 0o644); err != nil {
			return err
		}
		visualEntry, err := visualEvidenceEntry(targetDir, targetRel)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, visualEntry)
		return nil
	})
	if err != nil {
		return visualManifest{}, err
	}
	if len(manifest.Files) == 0 {
		return visualManifest{}, fmt.Errorf("%s contains no visual evidence image files", sourceDir)
	}
	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})
	if err := writeJSON(filepath.Join(targetDir, "visual-manifest.json"), manifest); err != nil {
		return visualManifest{}, err
	}
	return manifest, nil
}

func isIgnoredVisualMetadataFile(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".ds_store", "thumbs.db", "visual-manifest.json":
		return true
	default:
		return false
	}
}

func visualEvidenceEntry(root string, rel string) (visualManifestEntry, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		return visualManifestEntry{}, err
	}
	if info.IsDir() || info.Size() <= 0 {
		return visualManifestEntry{}, fmt.Errorf("visual evidence %s is not a non-empty file", rel)
	}
	width, height, err := imageSize(path)
	if err != nil {
		return visualManifestEntry{}, fmt.Errorf("visual evidence %s is not a decodable image: %w", rel, err)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return visualManifestEntry{}, err
	}
	return visualManifestEntry{
		Path:   filepath.ToSlash(rel),
		Bytes:  info.Size(),
		SHA256: sum,
		Width:  width,
		Height: height,
	}, nil
}

func matchVisualRequirements(files []visualManifestEntry) ([]ocrevidence.VisualRequirementMatch, error) {
	names := make([]string, 0, len(files))
	dimensions := make([]ocrevidence.VisualFileDimension, 0, len(files))
	for _, file := range files {
		names = append(names, strings.ToLower(filepath.ToSlash(file.Path)))
		dimensions = append(dimensions, ocrevidence.VisualFileDimension{
			Path:   file.Path,
			Width:  file.Width,
			Height: file.Height,
		})
	}
	matches, err := ocrevidence.MatchVisualRequirements(names)
	if err != nil {
		return nil, err
	}
	if failures := ocrevidence.VisualDimensionFailures(matches, dimensions); len(failures) > 0 {
		return nil, fmt.Errorf("visual evidence dimensions are too small: %s", strings.Join(failures, ", "))
	}
	return matches, nil
}

func copyDirectoryFiles(sourceDir string, targetDir string) (int, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return 0, err
	}
	count := 0
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return fmt.Errorf("unsafe evidence path %q", rel)
		}
		target := filepath.Join(targetDir, rel)
		if err := copyFile(path, target, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, fmt.Errorf("%s contains no evidence files", sourceDir)
	}
	return count, nil
}

func copyFile(source string, target string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func fallbackText(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func safeEvidenceName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "evidence"
	}
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(value)
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func imageSize(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	if config.Width <= 0 || config.Height <= 0 {
		return 0, 0, fmt.Errorf("invalid image dimensions %dx%d", config.Width, config.Height)
	}
	return config.Width, config.Height, nil
}
