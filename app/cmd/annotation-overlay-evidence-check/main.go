package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

const (
	minOneMinuteEvidenceDurationMs  = int64(60_000)
	minFiveMinuteEvidenceDurationMs = int64(300_000)
	minExportDurationToleranceMs    = int64(2_000)
)

var displayResolutionPattern = regexp.MustCompile(`\b\d{3,5}\s*x\s*\d{3,5}\b`)

var requiredEvidenceSourceChecks = []struct {
	SourceType string
	CheckName  string
}{
	{SourceType: "all-screens", CheckName: "source all-screens package"},
	{SourceType: "screen", CheckName: "source screen package"},
	{SourceType: "region", CheckName: "source region package"},
	{SourceType: "window", CheckName: "source window package"},
}

var requiredScreenshotEvidence = []namedEvidenceRequirement{
	{Name: "all-screens source screenshot", Terms: []string{"all-screens"}},
	{Name: "single screen source screenshot", Terms: []string{"source-screen", "single-screen", "screen-1"}},
	{Name: "region source screenshot", Terms: []string{"region"}},
	{Name: "window source screenshot", Terms: []string{"window"}},
	{Name: "pass-through click screenshot", Terms: []string{"pass-through", "click-through"}},
	{Name: "drawing state screenshot", Terms: []string{"drawing", "draw"}},
	{Name: "capsule controls screenshot", Terms: []string{"capsule"}},
}

var requiredRecordingEvidence = []namedEvidenceRequirement{
	{Name: "all-screens source recording", Terms: []string{"all-screens"}},
	{Name: "single screen source recording", Terms: []string{"source-screen", "single-screen", "screen-1"}},
	{Name: "region source recording", Terms: []string{"region"}},
	{Name: "window source recording", Terms: []string{"window"}},
	{Name: "export with annotations recording", Terms: []string{"with-annotations", "include-annotations"}},
	{Name: "export without annotations recording", Terms: []string{"without-annotations", "exclude-annotations"}},
}

type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type packageResult struct {
	Dir              string `json:"dir"`
	Manifest         string `json:"manifest"`
	SourceType       string `json:"sourceType"`
	SourceID         string `json:"sourceId"`
	EventsPath       string `json:"eventsPath"`
	DiagnosticsPath  string `json:"diagnosticsPath"`
	Snapshot         string `json:"snapshot"`
	ExportPath       string `json:"exportPath"`
	DurationMs       int64  `json:"durationMs"`
	ExportDurationMs int64  `json:"exportDurationMs"`
	Status           string `json:"status"`
	Message          string `json:"message,omitempty"`
}

type report struct {
	OK          bool            `json:"ok"`
	GeneratedAt time.Time       `json:"generatedAt"`
	EvidenceDir string          `json:"evidenceDir"`
	Checks      []checkResult   `json:"checks"`
	Packages    []packageResult `json:"packages"`
}

type overlayDiagnosticEvent struct {
	Type         string                              `json:"type"`
	WindowBounds diagnosticRect                      `json:"windowBounds"`
	CanvasBounds diagnosticRect                      `json:"canvasBounds"`
	Target       recpackage.ManifestAnnotationTarget `json:"target"`
	HitRegions   *overlayDiagnosticHitRegions        `json:"hitRegions,omitempty"`
}

type diagnosticRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type overlayDiagnosticHitRegions struct {
	Enabled          bool                         `json:"enabled"`
	ViewportWidth    float64                      `json:"viewportWidth"`
	ViewportHeight   float64                      `json:"viewportHeight"`
	DevicePixelRatio float64                      `json:"devicePixelRatio"`
	Regions          []overlayDiagnosticHitRegion `json:"regions"`
}

type overlayDiagnosticHitRegion struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Kind   string  `json:"kind,omitempty"`
	Radius float64 `json:"radius,omitempty"`
}

type appLogEvent struct {
	Component string            `json:"component"`
	Event     string            `json:"event"`
	Message   string            `json:"message,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

func main() {
	var evidenceDir string
	flag.StringVar(&evidenceDir, "evidence-dir", "", "annotation overlay evidence directory")
	flag.Parse()

	result, err := run(evidenceDir)
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
		fmt.Fprintf(os.Stderr, "encode annotation overlay evidence report: %v\n", err)
		os.Exit(1)
	}
	if !result.OK {
		os.Exit(1)
	}
}

func run(evidenceDir string) (report, error) {
	if strings.TrimSpace(evidenceDir) == "" {
		return report{}, fmt.Errorf("-evidence-dir is required")
	}
	resolved, err := filepath.Abs(evidenceDir)
	if err != nil {
		return report{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return report{}, err
	}
	if !info.IsDir() {
		return report{}, fmt.Errorf("evidence directory %q is not a directory", resolved)
	}

	result := report{
		OK:          true,
		GeneratedAt: time.Now().UTC(),
		EvidenceDir: resolved,
	}
	result.addCheck("README.md", validateEvidenceREADME(filepath.Join(resolved, "README.md")))
	result.addCheck("platform.txt", validatePlatformFile(filepath.Join(resolved, "platform.txt")))
	result.addCheck("app-log.jsonl", validateAppLog(filepath.Join(resolved, "app-log.jsonl")))
	result.addCheck("screenshots", requireEvidenceNamedFiles(filepath.Join(resolved, "screenshots"), requiredScreenshotEvidence))
	result.addCheck("recordings", requireEvidenceNamedFiles(filepath.Join(resolved, "recordings"), requiredRecordingEvidence))

	packagesDir := filepath.Join(resolved, "packages")
	result.addCheck("packages", requireDirWithFile(packagesDir))
	packageDirs, err := filepath.Glob(filepath.Join(packagesDir, "*"+recpackage.PackageDirSuffix))
	if err != nil {
		return report{}, err
	}
	if len(packageDirs) == 0 {
		result.addCheck("recording packages", fmt.Errorf("no *%s package found under %s", recpackage.PackageDirSuffix, packagesDir))
	}
	readyPackages := []packageResult{}
	for _, packageDir := range packageDirs {
		item := inspectPackage(packageDir)
		if item.Status != "ready" {
			result.OK = false
		} else {
			readyPackages = append(readyPackages, item)
		}
		result.Packages = append(result.Packages, item)
	}
	result.addCheck("1m recording package", requirePackageDuration(readyPackages, minOneMinuteEvidenceDurationMs))
	result.addCheck("5m recording package", requirePackageDuration(readyPackages, minFiveMinuteEvidenceDurationMs))
	for _, sourceCheck := range requiredEvidenceSourceChecks {
		result.addCheck(sourceCheck.CheckName, requireSourceType(readyPackages, sourceCheck.SourceType))
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

func inspectPackage(packageDir string) packageResult {
	manifestPath := filepath.Join(packageDir, recpackage.ManifestFile)
	item := packageResult{
		Dir:      packageDir,
		Manifest: manifestPath,
		Status:   "ready",
		ExportPath: filepath.Join(packageDir,
			filepath.FromSlash(exportplan.DefaultOutputPath)),
	}
	manifest, err := recpackage.NewService().ReadManifest(manifestPath)
	if err != nil {
		item.Status = "blocked"
		item.Message = err.Error()
		return item
	}
	item.SourceType = manifest.Source.Type
	item.SourceID = manifest.Source.ID
	if err := validateAnnotationEvidence(packageDir, manifest, &item); err != nil {
		item.Status = "blocked"
		item.Message = err.Error()
	}
	return item
}

func validateAnnotationEvidence(packageDir string, manifest recpackage.Manifest, item *packageResult) error {
	if manifest.RecordingMode != recpackage.RecordingModeScreen {
		return fmt.Errorf("recordingMode %q is not a screen recording", manifest.RecordingMode)
	}
	if manifest.Source.Type == "" || manifest.Source.ID == "" {
		return fmt.Errorf("manifest source type/id is required")
	}
	if manifest.Source.Geometry == nil || manifest.Source.Geometry.Width <= 0 || manifest.Source.Geometry.Height <= 0 {
		return fmt.Errorf("manifest source.geometry with positive width/height is required")
	}
	durationMs, err := screenRecordingDurationMs(manifest)
	if err != nil {
		return err
	}
	item.DurationMs = durationMs
	annotations := manifest.Annotations
	if annotations == nil || !annotations.Enabled {
		return fmt.Errorf("manifest annotations.enabled is required")
	}
	if annotations.Mode != "overlay" {
		return fmt.Errorf("annotations.mode = %q, want overlay", annotations.Mode)
	}
	if annotations.Target.Type != manifest.Source.Type || annotations.Target.ID != manifest.Source.ID {
		return fmt.Errorf("annotation target %s/%s does not match source %s/%s", annotations.Target.Type, annotations.Target.ID, manifest.Source.Type, manifest.Source.ID)
	}
	if !geometryEqual(annotations.Target.Geometry, manifest.Source.Geometry) {
		return fmt.Errorf("annotation target geometry does not match source geometry")
	}

	eventsRel := annotations.EventsPath
	if strings.TrimSpace(eventsRel) == "" {
		eventsRel = recpackage.AnnotationEventsFile
	}
	eventsPath, err := packageRelativePath(packageDir, eventsRel)
	if err != nil {
		return err
	}
	if err := requireNonEmptyFile(eventsPath); err != nil {
		return fmt.Errorf("annotation events: %w", err)
	}
	if err := validateAnnotationEvents(eventsPath); err != nil {
		return fmt.Errorf("annotation events: %w", err)
	}
	item.EventsPath = eventsPath
	diagnosticsRel := annotations.DiagnosticsPath
	if strings.TrimSpace(diagnosticsRel) == "" {
		diagnosticsRel = recpackage.AnnotationOverlayDiagnosticsFile
	}
	diagnosticsPath, err := packageRelativePath(packageDir, diagnosticsRel)
	if err != nil {
		return err
	}
	if err := requireNonEmptyFile(diagnosticsPath); err != nil {
		return fmt.Errorf("annotation overlay diagnostics: %w", err)
	}
	if err := validateOverlayDiagnostics(diagnosticsPath, manifest); err != nil {
		return fmt.Errorf("annotation overlay diagnostics: %w", err)
	}
	item.DiagnosticsPath = diagnosticsPath

	snapshotPath, err := findAnnotationSnapshot(packageDir, annotations.SnapshotPath)
	if err != nil {
		return err
	}
	item.Snapshot = snapshotPath
	if err := requireNonEmptyFile(item.ExportPath); err != nil {
		return fmt.Errorf("exported recording: %w", err)
	}
	exportDurationMs, err := validateExportedRecordingMP4(item.ExportPath, durationMs)
	if err != nil {
		return fmt.Errorf("exported recording: %w", err)
	}
	item.ExportDurationMs = exportDurationMs
	return nil
}

func validateExportedRecordingMP4(path string, expectedDurationMs int64) (int64, error) {
	probe, err := recpackage.ProbeMP4(path)
	if err != nil {
		return 0, fmt.Errorf("MP4 probe failed: %w", err)
	}
	if !probe.HasFileType {
		return 0, fmt.Errorf("MP4 ftyp box is missing")
	}
	if !probe.HasMovie {
		return 0, fmt.Errorf("MP4 moov box is missing")
	}
	if !probe.HasVideoTrack {
		return 0, fmt.Errorf("MP4 video track is missing")
	}
	if probe.DurationMs <= 0 {
		return 0, fmt.Errorf("MP4 duration is missing")
	}
	if expectedDurationMs > 0 {
		toleranceMs := exportDurationToleranceMs(expectedDurationMs)
		diff := absInt64(probe.DurationMs - expectedDurationMs)
		if diff > toleranceMs {
			return 0, fmt.Errorf("MP4 durationMs %d differs from diagnostics.sync.screen.durationMs %d by %dms, tolerance %dms", probe.DurationMs, expectedDurationMs, diff, toleranceMs)
		}
	}
	return probe.DurationMs, nil
}

func exportDurationToleranceMs(expectedDurationMs int64) int64 {
	tolerance := expectedDurationMs / 20
	if tolerance < minExportDurationToleranceMs {
		return minExportDurationToleranceMs
	}
	return tolerance
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

type readmeRequirement struct {
	Name  string
	Terms []string
}

type namedEvidenceRequirement struct {
	Name  string
	Terms []string
}

func validateEvidenceREADME(path string) error {
	if err := requireNonEmptyFile(path); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ToLower(string(data))
	requirements := []readmeRequirement{
		{Name: "software version", Terms: []string{"version", "版本"}},
		{Name: "commit", Terms: []string{"commit"}},
		{Name: "artifact source", Terms: []string{"artifact", "release", "actions", "产物", "构建"}},
		{Name: "operating system", Terms: []string{"operating system", "os", "操作系统"}},
		{Name: "display count", Terms: []string{"display count", "monitor count", "显示器数量"}},
		{Name: "resolution", Terms: []string{"resolution", "分辨率"}},
		{Name: "scale", Terms: []string{"scale", "缩放"}},
		{Name: "source all-screens", Terms: []string{"all-screens", "全部屏幕"}},
		{Name: "source screen", Terms: []string{"source screen", "screen ready", "单屏"}},
		{Name: "source region", Terms: []string{"source region", "region ready", "区域"}},
		{Name: "source window", Terms: []string{"source window", "window ready", "锁定窗口"}},
		{Name: "click-through", Terms: []string{"click-through", "点击穿透"}},
		{Name: "selection state", Terms: []string{"selection", "选择态"}},
		{Name: "drawing state", Terms: []string{"drawing", "绘制态"}},
		{Name: "capsule control", Terms: []string{"capsule", "胶囊"}},
		{Name: "export with annotations", Terms: []string{"with annotations", "include annotations", "包含标注"}},
		{Name: "export without annotations", Terms: []string{"without annotations", "exclude annotations", "不包含标注"}},
		{Name: "known failures", Terms: []string{"known failure", "known failures", "blocked", "blocker", "已知失败", "阻塞"}},
	}
	missing := make([]string, 0)
	for _, requirement := range requirements {
		if !containsAny(content, requirement.Terms) {
			missing = append(missing, requirement.Name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("README.md is missing evidence records: %s", strings.Join(missing, ", "))
	}
	return nil
}

func containsAny(content string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(content, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func validatePlatformFile(path string) error {
	if err := requireNonEmptyFile(path); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ToLower(string(data))
	requirements := []readmeRequirement{
		{Name: "operating system", Terms: []string{"operating system", "platform", "os", "windows", "macos", "darwin", "linux", "操作系统"}},
		{Name: "os version", Terms: []string{"version", "build", "版本"}},
		{Name: "display count", Terms: []string{"display count", "monitor count", "displays", "monitors", "显示器数量"}},
		{Name: "scale", Terms: []string{"scale", "scaling", "dpi", "缩放"}},
	}
	missing := make([]string, 0)
	for _, requirement := range requirements {
		if !containsAny(content, requirement.Terms) {
			missing = append(missing, requirement.Name)
		}
	}
	if !displayResolutionPattern.MatchString(content) {
		missing = append(missing, "display resolution")
	}
	if len(missing) > 0 {
		return fmt.Errorf("platform.txt is missing display environment records: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateAppLog(path string) error {
	if err := requireNonEmptyFile(path); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	lineNumber := 0
	events := 0
	hasStartup := false
	hasAnnotationSave := false
	recordingSources := map[string]bool{}
	annotationTargets := map[string]bool{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNumber++
		var event appLogEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("line %d is invalid JSON: %w", lineNumber, err)
		}
		if strings.TrimSpace(event.Component) == "" || strings.TrimSpace(event.Event) == "" {
			return fmt.Errorf("line %d is missing component/event", lineNumber)
		}
		switch {
		case event.Component == "app" && event.Event == "startup":
			hasStartup = true
		case event.Component == "recording" && event.Event == "start-request":
			if sourceType := strings.TrimSpace(event.Fields["sourceType"]); sourceType != "" {
				recordingSources[sourceType] = true
			}
		case event.Component == "annotation-overlay" && event.Event == "show":
			if targetType := strings.TrimSpace(event.Fields["targetType"]); targetType != "" {
				annotationTargets[targetType] = true
			}
		case event.Component == "annotation-overlay" && event.Event == "save-capture":
			hasAnnotationSave = true
		}
		events++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if events == 0 {
		return fmt.Errorf("no app log events found")
	}
	missing := make([]string, 0)
	if !hasStartup {
		missing = append(missing, "app/startup")
	}
	if !hasAnnotationSave {
		missing = append(missing, "annotation-overlay/save-capture")
	}
	for _, sourceCheck := range requiredEvidenceSourceChecks {
		if !recordingSources[sourceCheck.SourceType] {
			missing = append(missing, "recording/start-request sourceType="+sourceCheck.SourceType)
		}
		if !annotationTargets[sourceCheck.SourceType] {
			missing = append(missing, "annotation-overlay/show targetType="+sourceCheck.SourceType)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("app-log.jsonl is missing required events: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateAnnotationEvents(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNumber := 0
	events := 0
	hasSceneSnapshot := false
	hasElementEvent := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNumber++
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("line %d is invalid JSON: %w", lineNumber, err)
		}
		eventType, _ := event["type"].(string)
		if strings.TrimSpace(eventType) == "" {
			return fmt.Errorf("line %d is missing type", lineNumber)
		}
		if !hasNonNegativeNumber(event, "recordingOffsetMs") {
			return fmt.Errorf("line %d is missing non-negative recordingOffsetMs", lineNumber)
		}
		switch eventType {
		case "scene-snapshot":
			snapshotPath, _ := event["snapshotPath"].(string)
			if !strings.HasPrefix(strings.TrimSpace(snapshotPath), recpackage.AnnotationsDir+"/") {
				return fmt.Errorf("line %d scene-snapshot has invalid snapshotPath %q", lineNumber, snapshotPath)
			}
			hasSceneSnapshot = true
		case "element-created", "element-updated", "element-deleted":
			elementID, _ := event["elementId"].(string)
			if strings.TrimSpace(elementID) == "" {
				return fmt.Errorf("line %d %s is missing elementId", lineNumber, eventType)
			}
			if eventType != "element-deleted" {
				elementType, _ := event["elementType"].(string)
				if strings.TrimSpace(elementType) == "" {
					return fmt.Errorf("line %d %s is missing elementType", lineNumber, eventType)
				}
			}
			hasElementEvent = true
		}
		events++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if events == 0 {
		return fmt.Errorf("no annotation events found")
	}
	if !hasSceneSnapshot {
		return fmt.Errorf("missing scene-snapshot event")
	}
	if !hasElementEvent {
		return fmt.Errorf("missing element-created/element-updated/element-deleted event")
	}
	return nil
}

func hasNonNegativeNumber(event map[string]any, key string) bool {
	value, ok := event[key]
	if !ok {
		return false
	}
	number, ok := value.(float64)
	return ok && number >= 0
}

func screenRecordingDurationMs(manifest recpackage.Manifest) (int64, error) {
	if manifest.Diagnostics.Sync == nil {
		return 0, fmt.Errorf("diagnostics.sync is required to prove recording duration")
	}
	if !manifest.Diagnostics.Sync.Screen.Enabled {
		return 0, fmt.Errorf("diagnostics.sync.screen.enabled is required to prove recording duration")
	}
	if manifest.Diagnostics.Sync.Screen.DurationMs <= 0 {
		return 0, fmt.Errorf("diagnostics.sync.screen.durationMs must be positive")
	}
	return manifest.Diagnostics.Sync.Screen.DurationMs, nil
}

func requirePackageDuration(packages []packageResult, minDurationMs int64) error {
	for _, item := range packages {
		if item.DurationMs >= minDurationMs {
			return nil
		}
	}
	return fmt.Errorf("no ready annotation overlay package with durationMs >= %d", minDurationMs)
}

func requireSourceType(packages []packageResult, sourceType string) error {
	for _, item := range packages {
		if item.SourceType == sourceType {
			return nil
		}
	}
	return fmt.Errorf("no ready annotation overlay package for sourceType %q", sourceType)
}

func validateOverlayDiagnostics(path string, manifest recpackage.Manifest) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	required := map[string]bool{
		"show":         false,
		"hit-regions":  false,
		"save-capture": false,
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNumber := 0
	events := 0
	drawingHitRegions := false
	passThroughHitRegions := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNumber++
		var event overlayDiagnosticEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("line %d is invalid JSON: %w", lineNumber, err)
		}
		if strings.TrimSpace(event.Type) == "" {
			return fmt.Errorf("line %d is missing type", lineNumber)
		}
		if _, ok := required[event.Type]; ok {
			if event.WindowBounds.Width <= 0 || event.WindowBounds.Height <= 0 {
				return fmt.Errorf("line %d has empty windowBounds", lineNumber)
			}
			if event.CanvasBounds.Width <= 0 || event.CanvasBounds.Height <= 0 {
				return fmt.Errorf("line %d has empty canvasBounds", lineNumber)
			}
			if !rectMatchesGeometry(event.WindowBounds, manifest.Source.Geometry, true) {
				return fmt.Errorf("line %d windowBounds does not match source geometry", lineNumber)
			}
			if !rectMatchesGeometry(event.CanvasBounds, manifest.Source.Geometry, false) {
				return fmt.Errorf("line %d canvasBounds does not match source geometry size", lineNumber)
			}
			if event.Target.Type != manifest.Source.Type || event.Target.ID != manifest.Source.ID {
				return fmt.Errorf("line %d target %s/%s does not match source %s/%s", lineNumber, event.Target.Type, event.Target.ID, manifest.Source.Type, manifest.Source.ID)
			}
			if !geometryEqual(event.Target.Geometry, manifest.Source.Geometry) {
				return fmt.Errorf("line %d target geometry does not match source geometry", lineNumber)
			}
			if event.Type == "hit-regions" && event.HitRegions == nil {
				return fmt.Errorf("line %d hit-regions event is missing hitRegions payload", lineNumber)
			}
			if event.Type == "hit-regions" {
				if event.HitRegions.ViewportWidth <= 0 || event.HitRegions.ViewportHeight <= 0 {
					return fmt.Errorf("line %d hit-regions viewport must be positive", lineNumber)
				}
				hasCanvas := hitRegionsContainCanvasRect(event.HitRegions, manifest.Source.Geometry)
				hasPill := hitRegionsContainPill(event.HitRegions)
				if event.HitRegions.Enabled && hasCanvas {
					drawingHitRegions = true
				}
				if event.HitRegions.Enabled && hasPill && !hasCanvas {
					passThroughHitRegions = true
				}
			}
			required[event.Type] = true
		}
		events++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if events == 0 {
		return fmt.Errorf("no diagnostic events found")
	}
	for eventType, seen := range required {
		if !seen {
			return fmt.Errorf("missing %s diagnostic event", eventType)
		}
	}
	if !drawingHitRegions {
		return fmt.Errorf("missing drawing hit-regions event with full canvas rect")
	}
	if !passThroughHitRegions {
		return fmt.Errorf("missing pass-through hit-regions event without full canvas rect")
	}
	return nil
}

func hitRegionsContainCanvasRect(hitRegions *overlayDiagnosticHitRegions, geometry *recpackage.ManifestSourceGeometry) bool {
	if hitRegions == nil || geometry == nil {
		return false
	}
	for _, region := range hitRegions.Regions {
		if region.Kind != "rect" {
			continue
		}
		if nearZero(region.X) &&
			nearZero(region.Y) &&
			nearFloat(region.Width, float64(geometry.Width)) &&
			nearFloat(region.Height, float64(geometry.Height)) {
			return true
		}
	}
	return false
}

func hitRegionsContainPill(hitRegions *overlayDiagnosticHitRegions) bool {
	if hitRegions == nil {
		return false
	}
	for _, region := range hitRegions.Regions {
		if region.Kind == "pill" && region.Width > 0 && region.Height > 0 {
			return true
		}
	}
	return false
}

func nearZero(value float64) bool {
	return value > -0.5 && value < 0.5
}

func nearFloat(got float64, want float64) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	return diff <= 1
}

func rectMatchesGeometry(rect diagnosticRect, geometry *recpackage.ManifestSourceGeometry, includePosition bool) bool {
	if geometry == nil {
		return false
	}
	if rect.Width != geometry.Width || rect.Height != geometry.Height {
		return false
	}
	if includePosition && (rect.X != geometry.X || rect.Y != geometry.Y) {
		return false
	}
	return true
}

func geometryEqual(a *recpackage.ManifestSourceGeometry, b *recpackage.ManifestSourceGeometry) bool {
	if a == nil || b == nil {
		return false
	}
	return a.X == b.X &&
		a.Y == b.Y &&
		a.Width == b.Width &&
		a.Height == b.Height &&
		a.DisplayIndex == b.DisplayIndex &&
		a.NativeID == b.NativeID
}

func findAnnotationSnapshot(packageDir string, manifestSnapshotPath string) (string, error) {
	candidates := []string{}
	if strings.TrimSpace(manifestSnapshotPath) != "" {
		path, err := packageRelativePath(packageDir, manifestSnapshotPath)
		if err != nil {
			return "", err
		}
		candidates = append(candidates, path)
	}
	for _, relDir := range []string{
		recpackage.AnnotationSnapshotsDir,
		recpackage.AnnotationRenderPNGDir,
		recpackage.AnnotationExportsDir,
	} {
		dir, err := packageRelativePath(packageDir, relDir)
		if err != nil {
			return "", err
		}
		matches, err := filepath.Glob(filepath.Join(dir, "*.png"))
		if err != nil {
			return "", err
		}
		candidates = append(candidates, matches...)
	}
	for _, path := range candidates {
		if requireNonEmptyFile(path) == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no non-empty annotation PNG found under annotations snapshots, reconstructed png, or exports")
}

func packageRelativePath(packageDir string, relative string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("package path %q must stay inside package directory", relative)
	}
	joined := filepath.Join(packageDir, clean)
	absPackage, err := filepath.Abs(packageDir)
	if err != nil {
		return "", err
	}
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	if absJoined != absPackage && !strings.HasPrefix(absJoined, absPackage+string(filepath.Separator)) {
		return "", fmt.Errorf("package path %q escapes package directory", relative)
	}
	return absJoined, nil
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

func requireEvidenceNamedFiles(path string, requirements []namedEvidenceRequirement) error {
	if err := requireDirWithFile(path); err != nil {
		return err
	}
	files, err := evidenceFileNames(path)
	if err != nil {
		return err
	}
	missing := make([]string, 0)
	for _, requirement := range requirements {
		if !evidenceFilesContainAny(files, requirement.Terms) {
			missing = append(missing, requirement.Name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s is missing evidence files: %s", path, strings.Join(missing, ", "))
	}
	return nil
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

func evidenceFilesContainAny(files []string, terms []string) bool {
	for _, file := range files {
		if containsAny(file, terms) {
			return true
		}
	}
	return false
}
