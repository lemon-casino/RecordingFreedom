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
	"regexp"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocrevidence"
)

var displayResolutionPattern = regexp.MustCompile(`\b\d{3,5}\s*x\s*\d{3,5}\b`)

type options struct {
	evidenceDir        string
	requireTranslation bool
	mustContain        []string
}

type report struct {
	OK          bool           `json:"ok"`
	GeneratedAt time.Time      `json:"generatedAt"`
	EvidenceDir string         `json:"evidenceDir"`
	Checks      []checkResult  `json:"checks"`
	SourceKinds []sourceResult `json:"sourceKinds"`
}

type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type sourceResult struct {
	SourceKind string `json:"sourceKind"`
	ResultID   string `json:"resultId,omitempty"`
	ImagePath  string `json:"imagePath,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	BlockCount int    `json:"blockCount,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

type namedRequirement struct {
	Name    string
	Terms   []string
	Exclude []string
}

type appLogEvent struct {
	Timestamp string            `json:"timestamp,omitempty"`
	Component string            `json:"component"`
	Event     string            `json:"event"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type ocrJobEvidenceEvent struct {
	Event      string      `json:"event,omitempty"`
	JobID      string      `json:"jobId,omitempty"`
	SourceKind string      `json:"sourceKind,omitempty"`
	SourceID   string      `json:"sourceId,omitempty"`
	Status     string      `json:"status,omitempty"`
	ResultID   string      `json:"resultId,omitempty"`
	Result     *ocr.Result `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
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

type repeatedString []string

func (v *repeatedString) String() string {
	return strings.Join(*v, ",")
}

func (v *repeatedString) Set(value string) error {
	value = strings.TrimSpace(value)
	if value != "" {
		*v = append(*v, value)
	}
	return nil
}

func main() {
	var opts options
	var mustContain repeatedString
	flag.StringVar(&opts.evidenceDir, "evidence-dir", "", "real Wails desktop OCR evidence directory")
	flag.BoolVar(&opts.requireTranslation, "require-translation", false, "require translation result evidence under translations/")
	flag.Var(&mustContain, "must-contain", "plain text that every OCR source kind must contain; may be repeated")
	flag.Parse()
	opts.mustContain = []string(mustContain)

	result, err := run(opts)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "encode OCR desktop evidence report: %v\n", err)
		os.Exit(1)
	}
	if !result.OK {
		os.Exit(1)
	}
}

func run(opts options) (report, error) {
	if strings.TrimSpace(opts.evidenceDir) == "" {
		return report{}, errors.New("-evidence-dir is required")
	}
	evidenceDir, err := filepath.Abs(opts.evidenceDir)
	if err != nil {
		return report{}, err
	}
	info, err := os.Stat(evidenceDir)
	if err != nil {
		return report{}, err
	}
	if !info.IsDir() {
		return report{}, fmt.Errorf("evidence directory %q is not a directory", evidenceDir)
	}

	result := report{
		OK:          true,
		GeneratedAt: time.Now().UTC(),
		EvidenceDir: evidenceDir,
	}
	result.addCheck("README.md", validateREADME(filepath.Join(evidenceDir, "README.md")))
	result.addCheck("platform.txt", validatePlatform(filepath.Join(evidenceDir, "platform.txt")))
	result.addCheck("app-log.jsonl", validateAppLog(filepath.Join(evidenceDir, "app-log.jsonl"), opts.requireTranslation))
	result.addCheck("ocr-job-events.jsonl", validateOCRJobEvents(filepath.Join(evidenceDir, "ocr-job-events.jsonl")))
	result.addCheck("visual evidence", requireVisualEvidence(filepath.Join(evidenceDir, "visual")))
	result.addCheck("visual-capture-checklist", validateVisualCaptureChecklist(evidenceDir))
	precheck, precheckErr := loadDataRootPrecheck(filepath.Join(evidenceDir, "data-root-precheck.json"))
	result.addCheck("data-root-precheck", precheckErr)
	if precheckErr == nil {
		result.addCheck("app-log run window", validateAppLogRunWindow(filepath.Join(evidenceDir, "app-log.jsonl"), precheck.RunWindow))
		result.addCheck("desktop evidence session", validateSessionMarkers(filepath.Join(evidenceDir, "app-log.jsonl"), precheck.Session))
	}
	result.addCheck("export-report.json", validateExportReport(filepath.Join(evidenceDir, "export-report.json"), evidenceDir))

	results, err := loadOCRResults(filepath.Join(evidenceDir, "results"))
	if err != nil {
		result.addCheck("OCR results", err)
	} else {
		result.addCheck("evidence chain", validateEvidenceChain(
			filepath.Join(evidenceDir, "app-log.jsonl"),
			filepath.Join(evidenceDir, "ocr-job-events.jsonl"),
			results,
		))
		sourceResults := validateSourceResults(evidenceDir, results, opts.mustContain)
		result.SourceKinds = sourceResults
		for _, source := range sourceResults {
			if source.Status != "ready" {
				result.OK = false
			}
		}
		result.addCheck("OCR results", nil)
	}
	if opts.requireTranslation {
		result.addCheck("translation results", validateTranslations(filepath.Join(evidenceDir, "translations")))
	}
	for _, check := range result.Checks {
		if check.Status != "ready" {
			result.OK = false
			break
		}
	}
	return result, nil
}

func (r *report) addCheck(name string, err error) {
	if err != nil {
		r.Checks = append(r.Checks, checkResult{Name: name, Status: "blocked", Message: err.Error()})
		return
	}
	r.Checks = append(r.Checks, checkResult{Name: name, Status: "ready"})
}

func validateREADME(path string) error {
	content, err := readLowerNonEmpty(path)
	if err != nil {
		return err
	}
	requirements := []namedRequirement{
		{Name: "version", Terms: []string{"version", "版本"}},
		{Name: "commit", Terms: []string{"commit"}},
		{Name: "artifact source", Terms: []string{"artifact", "release", "actions", "产物", "构建"}},
		{Name: "desktop Wails evidence", Terms: []string{"wails", "desktop", "桌面"}},
		{Name: "OCR evidence", Terms: []string{"ocr", "文字识别"}},
		{Name: "known failures", Terms: []string{"known failure", "known failures", "blocked", "blocker", "已知失败", "阻塞"}},
	}
	missing := missingRequirements(content, requirements)
	if len(missing) > 0 {
		return fmt.Errorf("README.md is missing evidence records: %s", strings.Join(missing, ", "))
	}
	if err := validateKnownFailuresAreClear(content); err != nil {
		return err
	}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		if !strings.Contains(content, string(kind)) {
			return fmt.Errorf("README.md is missing source kind %s", kind)
		}
	}
	return nil
}

func validateKnownFailuresAreClear(content string) error {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "known failures") && !strings.HasPrefix(line, "known failure") {
			continue
		}
		_, value, found := strings.Cut(line, ":")
		if !found {
			return errors.New("README.md known failures must use 'known failures: none'")
		}
		value = strings.TrimSpace(value)
		switch value {
		case "", "none", "no", "n/a", "na", "无", "没有":
			return nil
		default:
			return fmt.Errorf("README.md known failures must be none before acceptance, got %q", value)
		}
	}
	return errors.New("README.md is missing known failures value")
}

func validatePlatform(path string) error {
	content, err := readLowerNonEmpty(path)
	if err != nil {
		return err
	}
	requirements := []namedRequirement{
		{Name: "operating system", Terms: []string{"operating system", "platform", "os", "windows", "macos", "darwin", "linux", "操作系统"}},
		{Name: "os version", Terms: []string{"version", "build", "版本"}},
		{Name: "display count", Terms: []string{"display count", "monitor count", "displays", "monitors", "显示器数量"}},
		{Name: "scale", Terms: []string{"scale", "scaling", "dpi", "缩放"}},
	}
	missing := missingRequirements(content, requirements)
	if !displayResolutionPattern.MatchString(content) {
		missing = append(missing, "display resolution")
	}
	if len(missing) > 0 {
		return fmt.Errorf("platform.txt is missing display environment records: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateAppLog(path string, requireTranslation bool) error {
	events, err := readAppLog(path)
	if err != nil {
		return err
	}
	hasStartup := false
	hasOCRPanel := false
	hasTranslateRequest := false
	hasTranslateReady := false
	hasAnnotationShow := false
	annotationCapturePackages := map[string]bool{}
	annotationBackgroundQueueSources := map[string]bool{}
	queuedSources := map[string]bool{}
	openResultSources := map[string]bool{}
	readImageSources := map[string]bool{}
	clientPreviewSources := map[string]bool{}
	clientRenderedSources := map[string]bool{}
	for _, event := range events {
		if event.Component == "app" && event.Event == "startup" {
			hasStartup = true
		}
		if event.Component == "floating-panel" && event.Event == "show" && event.Fields["kind"] == "ocr-result" {
			hasOCRPanel = true
		}
		if event.Component == "annotation-overlay" {
			switch event.Event {
			case "show":
				if strings.TrimSpace(event.Fields["packageDir"]) != "" {
					hasAnnotationShow = true
				}
			case "save-capture":
				packageDir := strings.TrimSpace(event.Fields["packageDir"])
				if packageDir != "" && strings.TrimSpace(event.Fields["bytes"]) != "" {
					annotationCapturePackages[packageDir] = true
				}
			}
		}
		if event.Component == "client.ocr-result" {
			sourceKind := strings.TrimSpace(event.Fields["sourceKind"])
			switch event.Event {
			case "preview-loaded":
				if event.Fields["available"] == "true" && strings.TrimSpace(event.Fields["bytes"]) != "" {
					if sourceKind != "" {
						clientPreviewSources[sourceKind] = true
					}
				}
			case "rendered":
				if event.Fields["hasPreview"] == "true" && strings.TrimSpace(event.Fields["polygonCount"]) != "" {
					if sourceKind != "" {
						clientRenderedSources[sourceKind] = true
					}
				}
			}
		}
		if event.Component != "ocr" {
			continue
		}
		switch event.Event {
		case "queue-request":
			sourceKind := strings.TrimSpace(event.Fields["sourceKind"])
			if sourceKind != "" {
				queuedSources[sourceKind] = true
			}
			if sourceKind == string(ocr.SourceWhiteboard) && strings.TrimSpace(event.Fields["priority"]) == ocr.JobPriorityBackground {
				if sourceID := strings.TrimSpace(event.Fields["sourceId"]); sourceID != "" {
					annotationBackgroundQueueSources[sourceID] = true
				}
			}
		case "open-result":
			if sourceKind := strings.TrimSpace(event.Fields["sourceKind"]); sourceKind != "" {
				openResultSources[sourceKind] = true
			}
		case "read-result-image":
			if sourceKind := strings.TrimSpace(event.Fields["sourceKind"]); sourceKind != "" {
				readImageSources[sourceKind] = true
			}
		case "translate-request":
			hasTranslateRequest = true
		case "translate-ready":
			hasTranslateReady = true
		}
	}
	missing := []string{}
	if !hasStartup {
		missing = append(missing, "app/startup")
	}
	if !hasOCRPanel {
		missing = append(missing, "floating-panel/show kind=ocr-result")
	}
	if !hasAnnotationShow {
		missing = append(missing, "annotation-overlay/show packageDir")
	}
	if len(annotationCapturePackages) == 0 {
		missing = append(missing, "annotation-overlay/save-capture packageDir bytes")
	}
	if len(annotationBackgroundQueueSources) == 0 {
		missing = append(missing, "ocr/queue-request sourceKind=whiteboard priority=background sourceId")
	} else if len(annotationCapturePackages) > 0 && !hasSharedKey(annotationCapturePackages, annotationBackgroundQueueSources) {
		missing = append(missing, "ocr/queue-request sourceKind=whiteboard priority=background sourceId matching annotation-overlay/save-capture packageDir")
	}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		key := string(kind)
		if !queuedSources[key] {
			missing = append(missing, "ocr/queue-request sourceKind="+key)
		}
		if !openResultSources[key] {
			missing = append(missing, "ocr/open-result sourceKind="+key)
		}
		if !readImageSources[key] {
			missing = append(missing, "ocr/read-result-image sourceKind="+key)
		}
		if !clientPreviewSources[key] {
			missing = append(missing, "client.ocr-result/preview-loaded sourceKind="+key+" available=true")
		}
		if !clientRenderedSources[key] {
			missing = append(missing, "client.ocr-result/rendered sourceKind="+key+" hasPreview=true")
		}
	}
	if requireTranslation {
		if !hasTranslateRequest {
			missing = append(missing, "ocr/translate-request")
		}
		if !hasTranslateReady {
			missing = append(missing, "ocr/translate-ready")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("app-log.jsonl is missing required events: %s", strings.Join(missing, ", "))
	}
	return nil
}

func hasSharedKey(left map[string]bool, right map[string]bool) bool {
	for key := range left {
		if right[key] {
			return true
		}
	}
	return false
}

func validateOCRJobEvents(path string) error {
	events, err := readOCRJobEvents(path)
	if err != nil {
		return err
	}
	queued := map[string]bool{}
	ready := map[string]bool{}
	for _, event := range events {
		sourceKind := ocrJobEventSourceKind(event)
		status := strings.TrimSpace(event.Status)
		switch status {
		case ocr.ResultStatusQueued:
			queued[sourceKind] = true
		case ocr.ResultStatusReady:
			ready[sourceKind] = true
		}
		if event.Result != nil {
			ready[string(event.Result.SourceKind)] = true
		}
	}
	missing := []string{}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		key := string(kind)
		if !queued[key] {
			missing = append(missing, key+" queued")
		}
		if !ready[key] {
			missing = append(missing, key+" ready")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("ocr-job-events.jsonl is missing required job states: %s", strings.Join(missing, ", "))
	}
	return nil
}

func readOCRJobEvents(path string) ([]ocrJobEvidenceEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	events := []ocrJobEvidenceEvent{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNumber++
		var event ocrJobEvidenceEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("line %d is invalid JSON: %w", lineNumber, err)
		}
		if ocrJobEventSourceKind(event) == "" {
			return nil, fmt.Errorf("line %d is missing sourceKind", lineNumber)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if lineNumber == 0 {
		return nil, errors.New("ocr-job-events.jsonl contains no events")
	}
	return events, nil
}

func ocrJobEventSourceKind(event ocrJobEvidenceEvent) string {
	sourceKind := strings.TrimSpace(event.SourceKind)
	if sourceKind == "" && event.Result != nil {
		sourceKind = string(event.Result.SourceKind)
	}
	return sourceKind
}

func validateEvidenceChain(appLogPath string, jobEventsPath string, results []ocr.Result) error {
	appEvents, err := readAppLog(appLogPath)
	if err != nil {
		return err
	}
	jobEvents, err := readOCRJobEvents(jobEventsPath)
	if err != nil {
		return err
	}
	byKind, err := requiredResultsByKind(results)
	if err != nil {
		return err
	}
	missing := []string{}
	if scopedMissing := validateScopedJobEvents(jobEvents, byKind); len(scopedMissing) > 0 {
		missing = append(missing, scopedMissing...)
	}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		result := byKind[kind]
		key := string(kind)
		if !appLogHasResultEvent(appEvents, "ocr", "queue-request", result, false) {
			missing = append(missing, "app-log queue-request sourceKind="+key+" sourceId="+result.SourceID)
		}
		if !appLogHasResultEvent(appEvents, "ocr", "open-result", result, true) {
			missing = append(missing, "app-log open-result sourceKind="+key+" sourceId="+result.SourceID+" resultId="+result.ID)
		}
		if !appLogHasResultEvent(appEvents, "ocr", "read-result-image", result, true) {
			missing = append(missing, "app-log read-result-image sourceKind="+key+" sourceId="+result.SourceID+" resultId="+result.ID)
		}
		if !appLogHasResultEvent(appEvents, "client.ocr-result", "preview-loaded", result, true) {
			missing = append(missing, "client preview-loaded sourceKind="+key+" sourceId="+result.SourceID+" resultId="+result.ID)
		}
		if !appLogHasResultEvent(appEvents, "client.ocr-result", "rendered", result, true) {
			missing = append(missing, "client rendered sourceKind="+key+" sourceId="+result.SourceID+" resultId="+result.ID)
		}
		if !jobEventsHaveQueued(jobEvents, result) {
			missing = append(missing, "job-events queued sourceKind="+key+" sourceId="+result.SourceID)
		}
		if !jobEventsHaveReady(jobEvents, result) {
			missing = append(missing, "job-events ready sourceKind="+key+" sourceId="+result.SourceID+" resultId="+result.ID)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("desktop OCR evidence chain is inconsistent: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateScopedJobEvents(events []ocrJobEvidenceEvent, byKind map[ocr.SourceKind]ocr.Result) []string {
	missing := []string{}
	for index, event := range events {
		status := strings.TrimSpace(event.Status)
		sourceKind := ocr.SourceKind(ocrJobEventSourceKind(event))
		result, ok := byKind[sourceKind]
		if !ok {
			missing = append(missing, fmt.Sprintf("job-events line %d unexpected sourceKind=%s", index+1, sourceKind))
			continue
		}
		sourceID := ocrJobEventSourceID(event)
		if sourceID != result.SourceID {
			missing = append(missing, fmt.Sprintf("job-events line %d unexpected sourceId=%s for sourceKind=%s", index+1, sourceID, sourceKind))
			continue
		}
		switch status {
		case ocr.ResultStatusQueued:
			// Queued events do not have a result id yet.
		case ocr.ResultStatusReady:
			if resultID := ocrJobEventResultID(event); resultID != result.ID {
				missing = append(missing, fmt.Sprintf("job-events line %d unexpected resultId=%s for sourceKind=%s", index+1, resultID, sourceKind))
			}
		default:
			missing = append(missing, fmt.Sprintf("job-events line %d unexpected status=%s for sourceKind=%s", index+1, status, sourceKind))
		}
	}
	return missing
}

func requiredResultsByKind(results []ocr.Result) (map[ocr.SourceKind]ocr.Result, error) {
	required := map[ocr.SourceKind]bool{}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		required[kind] = true
	}
	byKind := map[ocr.SourceKind]ocr.Result{}
	for _, result := range results {
		if !required[result.SourceKind] {
			return nil, fmt.Errorf("unexpected OCR result sourceKind %s", result.SourceKind)
		}
		if _, exists := byKind[result.SourceKind]; exists {
			return nil, fmt.Errorf("duplicate OCR result sourceKind %s", result.SourceKind)
		}
		byKind[result.SourceKind] = result
	}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		if _, exists := byKind[kind]; !exists {
			return nil, fmt.Errorf("missing OCR result sourceKind %s", kind)
		}
	}
	return byKind, nil
}

func appLogHasResultEvent(events []appLogEvent, component string, name string, result ocr.Result, requireResultID bool) bool {
	for _, event := range events {
		if event.Component != component || event.Event != name {
			continue
		}
		if strings.TrimSpace(event.Fields["sourceKind"]) != string(result.SourceKind) {
			continue
		}
		if strings.TrimSpace(event.Fields["sourceId"]) != result.SourceID {
			continue
		}
		if requireResultID && strings.TrimSpace(event.Fields["resultId"]) != result.ID {
			continue
		}
		return true
	}
	return false
}

func jobEventsHaveQueued(events []ocrJobEvidenceEvent, result ocr.Result) bool {
	for _, event := range events {
		if strings.TrimSpace(event.Status) != ocr.ResultStatusQueued {
			continue
		}
		if ocrJobEventSourceKind(event) == string(result.SourceKind) && ocrJobEventSourceID(event) == result.SourceID {
			return true
		}
	}
	return false
}

func jobEventsHaveReady(events []ocrJobEvidenceEvent, result ocr.Result) bool {
	for _, event := range events {
		if strings.TrimSpace(event.Status) != ocr.ResultStatusReady {
			continue
		}
		if ocrJobEventSourceKind(event) != string(result.SourceKind) {
			continue
		}
		if ocrJobEventSourceID(event) != result.SourceID {
			continue
		}
		if ocrJobEventResultID(event) != result.ID {
			continue
		}
		return true
	}
	return false
}

func ocrJobEventSourceID(event ocrJobEvidenceEvent) string {
	sourceID := strings.TrimSpace(event.SourceID)
	if sourceID == "" && event.Result != nil {
		sourceID = strings.TrimSpace(event.Result.SourceID)
	}
	return sourceID
}

func ocrJobEventResultID(event ocrJobEvidenceEvent) string {
	resultID := strings.TrimSpace(event.ResultID)
	if resultID == "" && event.Result != nil {
		resultID = strings.TrimSpace(event.Result.ID)
	}
	return resultID
}

func requireVisualEvidence(path string) error {
	if err := requireDirWithFile(path); err != nil {
		return err
	}
	manifest, err := loadVisualManifest(filepath.Join(path, "visual-manifest.json"))
	if err != nil {
		return err
	}
	files := make([]string, 0, len(manifest.Files))
	dimensions := make([]ocrevidence.VisualFileDimension, 0, len(manifest.Files))
	seen := map[string]bool{}
	for _, entry := range manifest.Files {
		key := strings.ToLower(filepath.ToSlash(entry.Path))
		if seen[key] {
			return fmt.Errorf("visual-manifest.json contains duplicate path %s", entry.Path)
		}
		seen[key] = true
		if err := validateVisualManifestEntry(path, entry); err != nil {
			return err
		}
		files = append(files, key)
		dimensions = append(dimensions, ocrevidence.VisualFileDimension{
			Path:   key,
			Width:  entry.Width,
			Height: entry.Height,
		})
	}
	matches, err := ocrevidence.MatchVisualRequirements(files)
	if err != nil {
		return err
	}
	if failures := ocrevidence.VisualDimensionFailures(matches, dimensions); len(failures) > 0 {
		return fmt.Errorf("visual evidence dimensions are too small: %s", strings.Join(failures, ", "))
	}
	return nil
}

func loadVisualManifest(path string) (visualManifest, error) {
	if err := requireNonEmptyFile(path); err != nil {
		return visualManifest{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return visualManifest{}, err
	}
	var manifest visualManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return visualManifest{}, fmt.Errorf("visual-manifest.json is invalid JSON: %w", err)
	}
	if manifest.SchemaVersion != 1 {
		return visualManifest{}, fmt.Errorf("visual-manifest.json schemaVersion = %d, want 1", manifest.SchemaVersion)
	}
	if len(manifest.Files) == 0 {
		return visualManifest{}, errors.New("visual-manifest.json contains no files")
	}
	return manifest, nil
}

func validateVisualManifestEntry(root string, entry visualManifestEntry) error {
	if strings.TrimSpace(entry.Path) == "" {
		return errors.New("visual-manifest.json contains an entry without path")
	}
	if entry.Bytes <= 0 || entry.Width <= 0 || entry.Height <= 0 || strings.TrimSpace(entry.SHA256) == "" {
		return fmt.Errorf("visual-manifest.json entry %s is missing bytes/sha256/width/height", entry.Path)
	}
	path, err := evidencePath(root, entry.Path)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() != entry.Bytes {
		return fmt.Errorf("visual evidence %s bytes = %d, want manifest %d", entry.Path, info.Size(), entry.Bytes)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(sum, entry.SHA256) {
		return fmt.Errorf("visual evidence %s sha256 mismatch", entry.Path)
	}
	width, height, err := imageSize(path)
	if err != nil {
		return err
	}
	if width != entry.Width || height != entry.Height {
		return fmt.Errorf("visual evidence %s dimensions %dx%d do not match manifest %dx%d", entry.Path, width, height, entry.Width, entry.Height)
	}
	return nil
}

func validateVisualCaptureChecklist(evidenceDir string) error {
	markdownPath := filepath.Join(evidenceDir, "visual-capture-checklist.md")
	jsonPath := filepath.Join(evidenceDir, "visual-capture-checklist.json")
	if err := requireNonEmptyFile(markdownPath); err != nil {
		return err
	}
	if err := requireNonEmptyFile(jsonPath); err != nil {
		return err
	}
	markdown, err := os.ReadFile(markdownPath)
	if err != nil {
		return err
	}
	markdownText := string(markdown)
	for _, needle := range []string{"Evidence chain requirements", "Capture runbook", "No duplicate sourceKind", "recording-annotation-ocr-safety.png"} {
		if !strings.Contains(markdownText, needle) {
			return fmt.Errorf("visual-capture-checklist.md is missing %q", needle)
		}
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	var checklist ocrevidence.ChecklistReport
	if err := json.Unmarshal(data, &checklist); err != nil {
		return fmt.Errorf("visual-capture-checklist.json is invalid JSON: %w", err)
	}
	if checklist.SchemaVersion != 1 {
		return fmt.Errorf("visual-capture-checklist.json schemaVersion = %d, want 1", checklist.SchemaVersion)
	}
	if !checklist.CheckComplete {
		return errors.New("visual-capture-checklist.json checkComplete=false")
	}
	if len(checklist.RequiredSourceKinds) != len(ocrevidence.RequiredSourceKinds) {
		return fmt.Errorf("visual-capture-checklist.json requiredSourceKinds = %d, want %d", len(checklist.RequiredSourceKinds), len(ocrevidence.RequiredSourceKinds))
	}
	if len(checklist.CaptureSteps) != len(ocrevidence.RequiredCaptureSteps) {
		return fmt.Errorf("visual-capture-checklist.json captureSteps = %d, want %d", len(checklist.CaptureSteps), len(ocrevidence.RequiredCaptureSteps))
	}
	if len(checklist.EvidenceChainRequirements) != len(ocrevidence.EvidenceChainRequirements) {
		return fmt.Errorf("visual-capture-checklist.json evidenceChainRequirements = %d, want %d", len(checklist.EvidenceChainRequirements), len(ocrevidence.EvidenceChainRequirements))
	}
	if len(checklist.MatchedVisualRequirements) != len(ocrevidence.RequiredVisualEvidence) {
		return fmt.Errorf("visual-capture-checklist.json matchedVisualRequirements = %d, want %d", len(checklist.MatchedVisualRequirements), len(ocrevidence.RequiredVisualEvidence))
	}
	if len(checklist.ExistingVisualDimensions) < len(ocrevidence.RequiredVisualEvidence) {
		return fmt.Errorf("visual-capture-checklist.json existingVisualDimensions = %d, want at least %d", len(checklist.ExistingVisualDimensions), len(ocrevidence.RequiredVisualEvidence))
	}
	if len(checklist.VisualDimensionFailures) != 0 {
		return fmt.Errorf("visual-capture-checklist.json contains visual dimension failures: %s", strings.Join(checklist.VisualDimensionFailures, ", "))
	}
	if checklist.DataRootPrecheck == nil {
		return errors.New("visual-capture-checklist.json is missing dataRootPrecheck")
	}
	if !checklist.DataRootPrecheck.CheckComplete {
		return errors.New("visual-capture-checklist.json dataRootPrecheck checkComplete=false")
	}
	if checklist.DataRootPrecheck.RunWindow.ResultStart.IsZero() || checklist.DataRootPrecheck.RunWindow.ResultEnd.IsZero() {
		return errors.New("visual-capture-checklist.json dataRootPrecheck is missing run window")
	}
	if strings.TrimSpace(checklist.DataRootPrecheck.Session.SessionID) == "" || checklist.DataRootPrecheck.Session.Start.IsZero() || checklist.DataRootPrecheck.Session.End.IsZero() {
		return errors.New("visual-capture-checklist.json dataRootPrecheck is missing ocr-desktop-evidence session")
	}
	return nil
}

func loadDataRootPrecheck(path string) (ocrevidence.DataRootPrecheckReport, error) {
	if err := requireNonEmptyFile(path); err != nil {
		return ocrevidence.DataRootPrecheckReport{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ocrevidence.DataRootPrecheckReport{}, err
	}
	var precheck ocrevidence.DataRootPrecheckReport
	if err := json.Unmarshal(data, &precheck); err != nil {
		return ocrevidence.DataRootPrecheckReport{}, fmt.Errorf("data-root-precheck.json is invalid JSON: %w", err)
	}
	if !precheck.CheckComplete {
		return ocrevidence.DataRootPrecheckReport{}, errors.New("data-root-precheck.json checkComplete=false")
	}
	if len(precheck.Sources) != len(ocrevidence.RequiredSourceKinds) {
		return ocrevidence.DataRootPrecheckReport{}, fmt.Errorf("data-root-precheck.json sources = %d, want %d", len(precheck.Sources), len(ocrevidence.RequiredSourceKinds))
	}
	if precheck.RunWindow.ResultStart.IsZero() || precheck.RunWindow.ResultEnd.IsZero() {
		return ocrevidence.DataRootPrecheckReport{}, errors.New("data-root-precheck.json is missing result run window")
	}
	if strings.TrimSpace(precheck.Session.SessionID) == "" || precheck.Session.Start.IsZero() || precheck.Session.End.IsZero() {
		return ocrevidence.DataRootPrecheckReport{}, errors.New("data-root-precheck.json is missing ocr-desktop-evidence session")
	}
	if precheck.Session.End.Before(precheck.Session.Start) {
		return ocrevidence.DataRootPrecheckReport{}, errors.New("data-root-precheck.json has invalid ocr-desktop-evidence session window")
	}
	if precheck.RunWindow.MaxSpanSeconds <= 0 || precheck.RunWindow.AppEventPaddingSeconds <= 0 {
		return ocrevidence.DataRootPrecheckReport{}, errors.New("data-root-precheck.json is missing run window limits")
	}
	if !precheck.Session.Start.Equal(precheck.RunWindow.AppEventStart) || !precheck.Session.End.Equal(precheck.RunWindow.AppEventEnd) {
		return ocrevidence.DataRootPrecheckReport{}, errors.New("data-root-precheck.json app event window must match ocr-desktop-evidence session")
	}
	if len(precheck.MissingRequirements) > 0 {
		return ocrevidence.DataRootPrecheckReport{}, fmt.Errorf("data-root-precheck.json still has missing requirements: %s", strings.Join(precheck.MissingRequirements, ", "))
	}
	return precheck, nil
}

func validateDataRootPrecheck(path string) error {
	_, err := loadDataRootPrecheck(path)
	return err
}

func validateAppLogRunWindow(path string, window ocrevidence.DataRootPrecheckRunWindow) error {
	if window.AppEventStart.IsZero() || window.AppEventEnd.IsZero() || window.AppEventEnd.Before(window.AppEventStart) {
		return errors.New("data-root-precheck.json has invalid app event window")
	}
	events, err := readAppLog(path)
	if err != nil {
		return err
	}
	for index, event := range events {
		timestampRaw := strings.TrimSpace(event.Timestamp)
		if timestampRaw == "" {
			return fmt.Errorf("app-log.jsonl event %d is missing timestamp", index+1)
		}
		timestamp, err := time.Parse(time.RFC3339Nano, timestampRaw)
		if err != nil {
			return fmt.Errorf("app-log.jsonl event %d has invalid timestamp %q: %w", index+1, timestampRaw, err)
		}
		if timestamp.Before(window.AppEventStart) || timestamp.After(window.AppEventEnd) {
			return fmt.Errorf("app-log.jsonl event %d timestamp %s is outside data-root precheck app event window %s -> %s", index+1, timestamp.Format(time.RFC3339Nano), window.AppEventStart.Format(time.RFC3339Nano), window.AppEventEnd.Format(time.RFC3339Nano))
		}
	}
	return nil
}

func validateSessionMarkers(path string, session ocrevidence.DataRootPrecheckSession) error {
	events, err := readAppLog(path)
	if err != nil {
		return err
	}
	hasStart := false
	hasEnd := false
	for _, event := range events {
		if event.Component != ocrevidence.EvidenceSessionComponent {
			continue
		}
		if strings.TrimSpace(event.Fields["sessionId"]) != session.SessionID {
			continue
		}
		timestamp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(event.Timestamp))
		if err != nil {
			return fmt.Errorf("ocr-desktop-evidence %s has invalid timestamp: %w", event.Event, err)
		}
		switch event.Event {
		case ocrevidence.EvidenceSessionStartEvent:
			if timestamp.Equal(session.Start) {
				hasStart = true
			}
		case ocrevidence.EvidenceSessionEndEvent:
			if timestamp.Equal(session.End) {
				hasEnd = true
			}
		}
	}
	missing := []string{}
	if !hasStart {
		missing = append(missing, "ocr-desktop-evidence/session-start sessionId="+session.SessionID)
	}
	if !hasEnd {
		missing = append(missing, "ocr-desktop-evidence/session-end sessionId="+session.SessionID)
	}
	if len(missing) > 0 {
		return fmt.Errorf("app-log.jsonl is missing required session markers: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateExportReport(path string, evidenceDir string) error {
	if err := requireNonEmptyFile(path); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report exportReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("export-report.json is invalid JSON: %w", err)
	}
	if !report.OK {
		return errors.New("export-report.json ok=false")
	}
	if report.GeneratedAt.IsZero() {
		return errors.New("export-report.json is missing generatedAt")
	}
	if strings.TrimSpace(report.DataRoot) == "" || strings.TrimSpace(report.EvidenceDir) == "" || strings.TrimSpace(report.VisualDir) == "" {
		return errors.New("export-report.json is missing dataRoot/evidenceDir/visualDir")
	}
	if filepath.ToSlash(strings.TrimSpace(report.VisualManifest)) != "visual/visual-manifest.json" {
		return fmt.Errorf("export-report.json visualManifest = %q, want visual/visual-manifest.json", report.VisualManifest)
	}
	if filepath.ToSlash(strings.TrimSpace(report.ChecklistMarkdown)) != "visual-capture-checklist.md" {
		return fmt.Errorf("export-report.json checklistMarkdown = %q, want visual-capture-checklist.md", report.ChecklistMarkdown)
	}
	if filepath.ToSlash(strings.TrimSpace(report.ChecklistJSON)) != "visual-capture-checklist.json" {
		return fmt.Errorf("export-report.json checklistJson = %q, want visual-capture-checklist.json", report.ChecklistJSON)
	}
	if filepath.ToSlash(strings.TrimSpace(report.DataRootPrecheck)) != "data-root-precheck.json" {
		return fmt.Errorf("export-report.json dataRootPrecheck = %q, want data-root-precheck.json", report.DataRootPrecheck)
	}
	if report.ResultCount != len(ocrevidence.RequiredSourceKinds) {
		return fmt.Errorf("export-report.json resultCount = %d, want %d", report.ResultCount, len(ocrevidence.RequiredSourceKinds))
	}
	if len(report.SourceKinds) != len(ocrevidence.RequiredSourceKinds) {
		return fmt.Errorf("export-report.json sourceKinds = %d, want %d", len(report.SourceKinds), len(ocrevidence.RequiredSourceKinds))
	}
	appLogLines, err := countNonEmptyLines(filepath.Join(evidenceDir, "app-log.jsonl"))
	if err != nil {
		return err
	}
	if report.AppLogLines != appLogLines {
		return fmt.Errorf("export-report.json appLogLines = %d, actual %d", report.AppLogLines, appLogLines)
	}
	jobEventLines, err := countNonEmptyLines(filepath.Join(evidenceDir, "ocr-job-events.jsonl"))
	if err != nil {
		return err
	}
	if report.JobEventLines != jobEventLines {
		return fmt.Errorf("export-report.json jobEventLines = %d, actual %d", report.JobEventLines, jobEventLines)
	}
	manifest, err := loadVisualManifest(filepath.Join(evidenceDir, "visual", "visual-manifest.json"))
	if err != nil {
		return err
	}
	if report.VisualFiles != len(manifest.Files) {
		return fmt.Errorf("export-report.json visualFiles = %d, actual %d", report.VisualFiles, len(manifest.Files))
	}
	visualFiles := make([]string, 0, len(manifest.Files))
	for _, entry := range manifest.Files {
		visualFiles = append(visualFiles, strings.ToLower(filepath.ToSlash(entry.Path)))
	}
	expectedVisualRequirements, err := ocrevidence.MatchVisualRequirements(visualFiles)
	if err != nil {
		return err
	}
	visualDimensions := make([]ocrevidence.VisualFileDimension, 0, len(manifest.Files))
	for _, entry := range manifest.Files {
		visualDimensions = append(visualDimensions, ocrevidence.VisualFileDimension{
			Path:   entry.Path,
			Width:  entry.Width,
			Height: entry.Height,
		})
	}
	if failures := ocrevidence.VisualDimensionFailures(expectedVisualRequirements, visualDimensions); len(failures) > 0 {
		return fmt.Errorf("export-report.json visualRequirements dimensions are too small: %s", strings.Join(failures, ", "))
	}
	if err := validateExportedVisualRequirements(report.VisualRequirements, expectedVisualRequirements); err != nil {
		return err
	}
	translationFiles, err := countJSONFiles(filepath.Join(evidenceDir, "translations"))
	if err != nil {
		return err
	}
	if report.TranslationFiles != translationFiles {
		return fmt.Errorf("export-report.json translationFiles = %d, actual %d", report.TranslationFiles, translationFiles)
	}

	required := map[string]bool{}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		required[string(kind)] = false
	}
	seen := map[string]bool{}
	for _, source := range report.SourceKinds {
		kind := strings.TrimSpace(source.SourceKind)
		if _, ok := required[kind]; !ok {
			return fmt.Errorf("export-report.json contains unknown sourceKind %q", kind)
		}
		if seen[kind] {
			return fmt.Errorf("export-report.json contains duplicate sourceKind %s", kind)
		}
		seen[kind] = true
		required[kind] = true
		if strings.TrimSpace(source.ResultID) == "" || strings.TrimSpace(source.ImagePath) == "" || strings.TrimSpace(source.ResultPath) == "" {
			return fmt.Errorf("export-report.json sourceKind %s is missing resultId/imagePath/resultPath", kind)
		}
		if err := validateSourceExport(evidenceDir, source); err != nil {
			return err
		}
	}
	for kind, found := range required {
		if !found {
			return fmt.Errorf("export-report.json is missing sourceKind %s", kind)
		}
	}
	return nil
}

func validateExportedVisualRequirements(actual []ocrevidence.VisualRequirementMatch, expected []ocrevidence.VisualRequirementMatch) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("export-report.json visualRequirements = %d, want %d", len(actual), len(expected))
	}
	byName := map[string]ocrevidence.VisualRequirementMatch{}
	for _, item := range actual {
		name := strings.TrimSpace(item.Name)
		if name == "" || strings.TrimSpace(item.Path) == "" {
			return errors.New("export-report.json visualRequirements contains an entry without name/path")
		}
		if _, exists := byName[name]; exists {
			return fmt.Errorf("export-report.json visualRequirements contains duplicate %q", name)
		}
		byName[name] = item
	}
	for _, expectedItem := range expected {
		actualItem, ok := byName[expectedItem.Name]
		if !ok {
			return fmt.Errorf("export-report.json visualRequirements is missing %q", expectedItem.Name)
		}
		if filepath.ToSlash(actualItem.Path) != filepath.ToSlash(expectedItem.Path) {
			return fmt.Errorf("export-report.json visualRequirements %q path = %q, want %q", expectedItem.Name, actualItem.Path, expectedItem.Path)
		}
		if actualItem.MinWidth != expectedItem.MinWidth || actualItem.MinHeight != expectedItem.MinHeight {
			return fmt.Errorf("export-report.json visualRequirements %q min size = %dx%d, want %dx%d", expectedItem.Name, actualItem.MinWidth, actualItem.MinHeight, expectedItem.MinWidth, expectedItem.MinHeight)
		}
	}
	return nil
}

func validateSourceExport(evidenceDir string, source sourceExport) error {
	resultPath, err := evidenceFilePath(evidenceDir, source.ResultPath, "resultPath")
	if err != nil {
		return err
	}
	imagePath, err := evidenceFilePath(evidenceDir, source.ImagePath, "imagePath")
	if err != nil {
		return err
	}
	if _, _, err := imageSize(imagePath); err != nil {
		return err
	}
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return err
	}
	var result ocr.Result
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("export-report.json resultPath %s is not an OCR result JSON: %w", source.ResultPath, err)
	}
	if string(result.SourceKind) != source.SourceKind {
		return fmt.Errorf("export-report.json sourceKind %s points to result sourceKind %s", source.SourceKind, result.SourceKind)
	}
	if result.ID != source.ResultID {
		return fmt.Errorf("export-report.json sourceKind %s resultId = %s, actual %s", source.SourceKind, source.ResultID, result.ID)
	}
	if filepath.ToSlash(result.ImagePath) != filepath.ToSlash(source.ImagePath) {
		return fmt.Errorf("export-report.json sourceKind %s imagePath = %s, actual %s", source.SourceKind, source.ImagePath, result.ImagePath)
	}
	return nil
}

func loadOCRResults(path string) ([]ocr.Result, error) {
	if err := requireDirWithFile(path); err != nil {
		return nil, err
	}
	results := []ocr.Result{}
	err := filepath.WalkDir(path, func(itemPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			return nil
		}
		data, err := os.ReadFile(itemPath)
		if err != nil {
			return err
		}
		var result ocr.Result
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("%s is not an OCR result JSON: %w", itemPath, err)
		}
		results = append(results, result)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("%s contains no OCR result JSON files", path)
	}
	return results, nil
}

func validateSourceResults(evidenceDir string, results []ocr.Result, mustContain []string) []sourceResult {
	byKind := map[ocr.SourceKind]ocr.Result{}
	for _, result := range results {
		if _, exists := byKind[result.SourceKind]; !exists {
			byKind[result.SourceKind] = result
		}
	}
	sourceResults := []sourceResult{}
	for _, kind := range ocrevidence.RequiredSourceKinds {
		result, exists := byKind[kind]
		item := sourceResult{SourceKind: string(kind), Status: "ready"}
		if !exists {
			item.Status = "blocked"
			item.Message = "missing OCR result"
			sourceResults = append(sourceResults, item)
			continue
		}
		item.ResultID = result.ID
		item.ImagePath = result.ImagePath
		item.Width = result.Width
		item.Height = result.Height
		item.BlockCount = len(result.Blocks)
		if err := validateOCRResult(evidenceDir, result, mustContain); err != nil {
			item.Status = "blocked"
			item.Message = err.Error()
		}
		sourceResults = append(sourceResults, item)
	}
	return sourceResults
}

func validateOCRResult(evidenceDir string, result ocr.Result, mustContain []string) error {
	if strings.TrimSpace(result.ID) == "" {
		return errors.New("result id is required")
	}
	if strings.TrimSpace(result.SourceID) == "" {
		return errors.New("sourceId is required")
	}
	if strings.TrimSpace(result.ModelID) == "" {
		return errors.New("modelId is required")
	}
	if strings.TrimSpace(result.Language) == "" {
		return errors.New("language is required")
	}
	if result.Width <= 0 || result.Height <= 0 {
		return errors.New("result width/height must be positive")
	}
	if strings.TrimSpace(result.PlainText) == "" {
		return errors.New("plainText is required")
	}
	for _, expected := range mustContain {
		if !strings.Contains(result.PlainText, expected) {
			return fmt.Errorf("plainText is missing required text %q", expected)
		}
	}
	if len(result.Blocks) == 0 {
		return errors.New("at least one OCR block is required")
	}
	for _, block := range result.Blocks {
		if err := validateOCRBlock(result, block); err != nil {
			return err
		}
	}
	imagePath, err := evidencePath(evidenceDir, result.ImagePath)
	if err != nil {
		return err
	}
	width, height, err := imageSize(imagePath)
	if err != nil {
		return err
	}
	if width != result.Width || height != result.Height {
		return fmt.Errorf("image dimensions %dx%d do not match OCR result %dx%d", width, height, result.Width, result.Height)
	}
	return nil
}

func validateOCRBlock(result ocr.Result, block ocr.Block) error {
	if strings.TrimSpace(block.ID) == "" {
		return errors.New("OCR block id is required")
	}
	if strings.TrimSpace(block.Text) == "" {
		return fmt.Errorf("OCR block %s text is required", block.ID)
	}
	if block.Confidence <= 0 || block.Confidence > 1 {
		return fmt.Errorf("OCR block %s confidence %.3f is outside 0..1", block.ID, block.Confidence)
	}
	if len(block.Box) < 4 {
		return fmt.Errorf("OCR block %s requires at least four polygon points", block.ID)
	}
	for _, point := range block.Box {
		if point.X < 0 || point.Y < 0 || point.X > float64(result.Width) || point.Y > float64(result.Height) {
			return fmt.Errorf("OCR block %s point %.2f,%.2f is outside image %dx%d", block.ID, point.X, point.Y, result.Width, result.Height)
		}
	}
	return nil
}

func validateTranslations(path string) error {
	if err := requireDirWithFile(path); err != nil {
		return err
	}
	found := 0
	err := filepath.WalkDir(path, func(itemPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			return nil
		}
		data, err := os.ReadFile(itemPath)
		if err != nil {
			return err
		}
		var result ocr.TranslationResult
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("%s is not an OCR translation JSON: %w", itemPath, err)
		}
		if strings.TrimSpace(result.OcrResultID) == "" || strings.TrimSpace(result.Provider) == "" || result.Provider == "disabled" {
			return fmt.Errorf("%s is missing translation provider/result id", itemPath)
		}
		if strings.TrimSpace(result.TargetLanguage) == "" || len(result.Blocks) == 0 {
			return fmt.Errorf("%s is missing target language or translated blocks", itemPath)
		}
		for _, block := range result.Blocks {
			if strings.TrimSpace(block.BlockID) == "" || strings.TrimSpace(block.Source) == "" || strings.TrimSpace(block.Translated) == "" {
				return fmt.Errorf("%s contains an incomplete translation block", itemPath)
			}
		}
		found++
		return nil
	})
	if err != nil {
		return err
	}
	if found == 0 {
		return fmt.Errorf("%s contains no translation result JSON files", path)
	}
	return nil
}

func readAppLog(path string) ([]appLogEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	events := []appLogEvent{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNumber++
		var event appLogEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("line %d is invalid JSON: %w", lineNumber, err)
		}
		if strings.TrimSpace(event.Component) == "" || strings.TrimSpace(event.Event) == "" {
			return nil, fmt.Errorf("line %d is missing component/event", lineNumber)
		}
		if event.Fields == nil {
			event.Fields = map[string]string{}
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, errors.New("app-log.jsonl contains no events")
	}
	return events, nil
}

func readLowerNonEmpty(path string) (string, error) {
	if err := requireNonEmptyFile(path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.ToLower(string(data)), nil
}

func missingRequirements(content string, requirements []namedRequirement) []string {
	missing := []string{}
	for _, requirement := range requirements {
		if !containsAny(content, requirement.Terms) {
			missing = append(missing, requirement.Name)
		}
	}
	return missing
}

func containsAny(content string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(content, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func requireDirWithFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	hasFile := false
	err = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			hasFile = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !hasFile {
		return fmt.Errorf("%s contains no files", path)
	}
	return nil
}

func requireNonEmptyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%s is empty", path)
	}
	return nil
}

func countNonEmptyLines(path string) (int, error) {
	if err := requireNonEmptyFile(path); err != nil {
		return 0, err
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	count := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func countJSONFiles(path string) (int, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("%s is not a directory", path)
	}
	count := 0
	err = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.ToLower(filepath.Ext(entry.Name())) == ".json" {
			count++
		}
		return nil
	})
	return count, err
}

func evidenceFileNames(path string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(path, func(itemPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(path, itemPath)
		if err != nil {
			return err
		}
		files = append(files, strings.ToLower(filepath.ToSlash(rel)))
		return nil
	})
	return files, err
}

func evidenceFilePath(root string, value string, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	candidate := value
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, filepath.FromSlash(value))
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absCandidate)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("evidence %s %q must stay inside %q", field, value, root)
	}
	if err := requireNonEmptyFile(absCandidate); err != nil {
		return "", err
	}
	return absCandidate, nil
}

func evidencePath(root string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("imagePath is required")
	}
	candidate := value
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, filepath.FromSlash(value))
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absCandidate)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("evidence image path %q must stay inside %q", value, root)
	}
	if err := requireNonEmptyFile(absCandidate); err != nil {
		return "", err
	}
	return absCandidate, nil
}

func imageSize(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("decode image config %s: %w", path, err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return 0, 0, fmt.Errorf("image %s has invalid dimensions %dx%d", path, config.Width, config.Height)
	}
	return config.Width, config.Height, nil
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
