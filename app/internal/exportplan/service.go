package exportplan

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

const (
	annotationTimelineElementPNGs              = "element-pngs"
	annotationTimelineSnapshotSegments         = "snapshot-segments"
	annotationTimelineSnapshotFromFirstElement = "snapshot-from-first-element"
	annotationTimelineSnapshotAnchor           = "snapshot-anchor"
	annotationTimelineSnapshotOnly             = "snapshot-only"
	maxAnnotationTimelineLineBytes             = 1024 * 1024
	maxAnnotationTimelineEvents                = 200000
	maxAnnotationTimelineSnapshots             = 2000
	warnAnnotationTimelineEventBytes           = 64 * 1024 * 1024
	warnAnnotationTimelineSnapshotBytes        = 512 * 1024 * 1024
)

type Service struct {
	packages *recpackage.Service
}

func NewService(packages *recpackage.Service) *Service {
	if packages == nil {
		packages = recpackage.NewService()
	}
	return &Service{packages: packages}
}

func (s *Service) Plan(req Request) (Plan, error) {
	videoDir, packageDir, err := validatePackageLocation(req.VideoDir, req.PackageDir)
	if err != nil {
		return Plan{}, err
	}
	manifestPath := filepath.Join(packageDir, recpackage.ManifestFile)
	manifest, err := s.packages.ReadManifest(manifestPath)
	if err != nil {
		return Plan{}, err
	}
	if manifest.Status != recpackage.StatusReady {
		return Plan{}, fmt.Errorf("recording package must be ready before export, got status %q", manifest.Status)
	}
	if manifest.Diagnostics.Mock && !req.AllowMock {
		return Plan{}, errors.New("mock recording packages cannot be exported as real media")
	}
	if req.RequireSync {
		if manifest.Diagnostics.Sync == nil {
			return Plan{}, errors.New("export requires diagnostics.sync for audio/video alignment")
		}
		if manifest.Diagnostics.Sync.TimelineBase == recpackage.TimelineBaseMock && !req.AllowMock {
			return Plan{}, errors.New("export requires real media sync diagnostics, got mock timeline")
		}
	}

	outputRel := strings.TrimSpace(req.OutputPath)
	if outputRel == "" {
		outputRel = DefaultOutputPath
	}
	if err := validatePackageRelativePath("outputPath", outputRel); err != nil {
		return Plan{}, err
	}
	screenInput, err := mediaInputPath(packageDir, "screenVideoPath", manifest.Media.ScreenVideoPath)
	if err != nil {
		return Plan{}, err
	}

	pipConfig := pip.NormalizeConfigForPreset(manifest.Camera.PIPPreset, manifest.Camera.PIP)
	canvas := canvasForPIP(req.Canvas, pipConfig.Preset)
	if pipConfig.Preset != pip.PresetOff && (canvas.Width <= 0 || canvas.Height <= 0) {
		canvas = canvasFromManifest(manifest)
	}
	placement, err := pip.Place(pipConfig, canvas)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		PackageDir:          packageDir,
		ManifestPath:        manifestPath,
		OutputPath:          filepath.Join(packageDir, filepath.Clean(outputRel)),
		ScreenInputPath:     screenInput,
		WebcamStartOffsetMs: manifest.Media.WebcamStartOffsetMs,
		PIPPreset:           string(pipConfig.Preset),
		PIPConfig:           pipConfig,
		PIPRect:             placement.Rect,
		PIPLayout:           placement,
	}
	if manifest.Diagnostics.Sync != nil {
		plan.TimelineBase = manifest.Diagnostics.Sync.TimelineBase
		audioDiagnosticsPath, err := optionalPackagePath(packageDir, "audioDiagnosticsPath", manifest.Diagnostics.Sync.AudioDiagnosticsPath)
		if err != nil {
			return Plan{}, err
		}
		videoDiagnosticsPath, err := optionalPackagePath(packageDir, "videoDiagnosticsPath", manifest.Diagnostics.Sync.VideoDiagnosticsPath)
		if err != nil {
			return Plan{}, err
		}
		plan.AudioDiagnosticsPath = audioDiagnosticsPath
		plan.VideoDiagnosticsPath = videoDiagnosticsPath
		for _, segment := range manifest.Diagnostics.Sync.PauseSegments {
			plan.PauseSegments = append(plan.PauseSegments, PauseSegmentPlan{
				StartOffsetMs: segment.StartOffsetMs,
				EndOffsetMs:   segment.EndOffsetMs,
				DurationMs:    segment.DurationMs,
			})
		}
	}
	if shouldIncludeAnnotations(req.IncludeAnnotations, manifest.Annotations) {
		annotationInput, err := mediaInputPath(packageDir, "annotations.snapshotPath", manifest.Annotations.SnapshotPath)
		if err != nil {
			return Plan{}, err
		}
		plan.AnnotationInputPath = annotationInput
		plan.AnnotationsVisible = true
		eventsPath, timeline, warnings, err := s.annotationTimeline(packageDir, manifest.Annotations, req.PrepareAnnotationAssets, annotationCanvasSize(manifest, canvas))
		if err != nil {
			return Plan{}, err
		}
		plan.AnnotationEventsPath = eventsPath
		plan.AnnotationStartMs = timeline.StartOffsetMs
		plan.AnnotationTimeline = timeline.Mode
		plan.AnnotationSnapshots = timeline.Snapshots
		plan.AnnotationRenderMode = timeline.RenderMode
		plan.AnnotationElementScenes = timeline.ElementScenes
		plan.AnnotationSummary = &timeline.Summary
		plan.Warnings = append(plan.Warnings, warnings...)
	}

	if pipConfig.Preset == pip.PresetOff || !manifest.Camera.Enabled {
		plan.PIPRect = pip.Rect{Visible: false}
		plan.PIPLayout = pip.Placement{Visible: false, Rect: pip.Rect{Visible: false}, Shape: pipConfig.Shape, Mirror: pipConfig.Mirror, EdgeFeather: pipConfig.EdgeFeather}
		plan.WebcamStartOffsetMs = 0
		return plan, nil
	}
	webcamInput, err := mediaInputPath(packageDir, "webcamVideoPath", manifest.Media.WebcamVideoPath)
	if err != nil {
		return Plan{}, err
	}
	plan.WebcamInputPath = webcamInput
	if manifest.Media.WebcamStartOffsetMs == 0 {
		plan.Warnings = append(plan.Warnings, "webcamStartOffsetMs is zero; export will align webcam at screen start")
	}
	if err := ensureInside(videoDir, packageDir); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

type annotationTimeline struct {
	StartOffsetMs int64
	Mode          string
	Snapshots     []AnnotationSnapshotPlan
	RenderMode    string
	ElementScenes []AnnotationElementScenePlan
	Summary       AnnotationTimelineSummary
}

type annotationSnapshotCandidate struct {
	RelativePath  string
	StartOffsetMs int64
}

type annotationSnapshotSegmentsResult struct {
	Segments     []AnnotationSnapshotPlan
	Warnings     []string
	SkippedCount int
	TotalBytes   int64
}

func (s *Service) annotationTimeline(packageDir string, annotations *recpackage.ManifestAnnotations, prepareAssets bool, canvas pip.Size) (string, annotationTimeline, []string, error) {
	if annotations == nil || strings.TrimSpace(annotations.EventsPath) == "" {
		return "", annotationTimeline{Mode: annotationTimelineSnapshotOnly}, nil, nil
	}
	eventsPath, err := optionalPackagePath(packageDir, "annotations.eventsPath", annotations.EventsPath)
	if err != nil {
		return "", annotationTimeline{}, nil, err
	}
	timeline, warnings, err := readAnnotationTimeline(packageDir, eventsPath, prepareAssets, canvas)
	if errors.Is(err, os.ErrNotExist) {
		return eventsPath, annotationTimeline{Mode: annotationTimelineSnapshotOnly}, []string{"annotation events file is missing; export will compose the final annotation snapshot for the full video"}, nil
	}
	if err != nil {
		return "", annotationTimeline{}, nil, err
	}
	return eventsPath, timeline, warnings, nil
}

func readAnnotationTimeline(packageDir string, path string, prepareAssets bool, canvas pip.Size) (annotationTimeline, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return annotationTimeline{}, nil, err
	}
	defer file.Close()
	eventFileBytes := int64(0)
	if info, err := file.Stat(); err == nil && !info.IsDir() {
		eventFileBytes = info.Size()
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxAnnotationTimelineLineBytes)
	eventCount := 0
	snapshotCount := 0
	elementEventCount := 0
	var firstSnapshotOffset int64 = -1
	var firstElementOffset int64 = -1
	var firstOffset int64 = -1
	var lastOffset int64 = -1
	snapshots := make([]annotationSnapshotCandidate, 0)
	elementReconstructor := newAnnotationElementReconstructor(prepareAssets)
	warnings := make([]string, 0)
	if eventFileBytes > warnAnnotationTimelineEventBytes {
		warnings = append(warnings, fmt.Sprintf("annotation events file is %s; long recordings should be checked before export", formatBytes(eventFileBytes)))
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		eventCount++
		if eventCount > maxAnnotationTimelineEvents {
			return annotationTimeline{}, nil, fmt.Errorf("annotation events file has more than %d events", maxAnnotationTimelineEvents)
		}
		event := map[string]any{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return annotationTimeline{}, nil, fmt.Errorf("annotation event line %d is invalid JSON: %w", eventCount, err)
		}
		offset, ok := annotationEventOffsetMs(event)
		if !ok {
			continue
		}
		if firstOffset < 0 {
			firstOffset = offset
		}
		if offset > lastOffset {
			lastOffset = offset
		}
		eventType, _ := event["type"].(string)
		eventType = strings.TrimSpace(eventType)
		if eventType == "scene-snapshot" {
			snapshotCount++
			if firstSnapshotOffset < 0 {
				firstSnapshotOffset = offset
			}
			if snapshotPath := annotationEventSnapshotPath(event); isTimelineSnapshotPath(snapshotPath) {
				snapshots = append(snapshots, annotationSnapshotCandidate{RelativePath: snapshotPath, StartOffsetMs: offset})
			}
			continue
		}
		if strings.HasPrefix(eventType, "element-") {
			elementEventCount++
			if firstElementOffset < 0 {
				firstElementOffset = offset
			}
			elementReconstructor.Apply(event)
		}
	}
	if err := scanner.Err(); err != nil {
		return annotationTimeline{}, nil, fmt.Errorf("read annotation events: %w", err)
	}
	if firstOffset < 0 {
		firstOffset = 0
	}
	if lastOffset < 0 {
		lastOffset = 0
	}
	summary := AnnotationTimelineSummary{
		EventCount:        eventCount,
		SnapshotCount:     snapshotCount,
		ElementEventCount: elementEventCount,
		StartOffsetMs:     firstOffset,
		EndOffsetMs:       lastOffset,
		EventFileBytes:    eventFileBytes,
	}
	elementReconstructor.ApplyToSummary(&summary)
	if summary.MissingElementPayloads > 0 {
		warnings = append(warnings, fmt.Sprintf("annotation element timeline has %d event(s) without element payload; element-level reconstruction is partial", summary.MissingElementPayloads))
	}
	elementScenes, elementSceneWarnings, err := elementReconstructor.BuildSceneAssets(packageDir, canvas.Width, canvas.Height)
	if err != nil {
		return annotationTimeline{}, nil, err
	}
	warnings = append(warnings, elementSceneWarnings...)
	renderMode := ""
	if len(elementScenes) > 0 {
		renderMode = annotationRenderModeElementScenes
	}
	renderedResult, renderedComplete, err := annotationRenderedElementSegments(packageDir, elementScenes)
	if err != nil {
		return annotationTimeline{}, nil, err
	}
	if len(elementScenes) > 0 && !renderedComplete {
		warnings = append(warnings, "annotation element scene PNGs are not fully rendered yet; snapshot timeline will be used until render assets are complete")
	}
	if renderedComplete && len(renderedResult.Segments) > 0 {
		renderMode = annotationRenderModeElementPNGs
		warnings = append(warnings, renderedResult.Warnings...)
		summary.Mode = annotationTimelineElementPNGs
		summary.ExportedSnapshotCount = len(renderedResult.Segments)
		summary.SkippedSnapshotCount = renderedResult.SkippedCount
		summary.SnapshotBytes = renderedResult.TotalBytes
		summary.StartOffsetMs = renderedResult.Segments[0].StartOffsetMs
		summary.EndOffsetMs = renderedResult.Segments[len(renderedResult.Segments)-1].StartOffsetMs
		if len(renderedResult.Segments) > 1 && renderedResult.Segments[len(renderedResult.Segments)-2].EndOffsetMs > summary.EndOffsetMs {
			summary.EndOffsetMs = renderedResult.Segments[len(renderedResult.Segments)-2].EndOffsetMs
		}
		if renderedResult.TotalBytes > warnAnnotationTimelineSnapshotBytes {
			warnings = append(warnings, fmt.Sprintf("annotation rendered PNG timeline uses %s; verify long recording export storage before composing", formatBytes(renderedResult.TotalBytes)))
		}
		return annotationTimeline{
			StartOffsetMs: renderedResult.Segments[0].StartOffsetMs,
			Mode:          annotationTimelineElementPNGs,
			Snapshots:     renderedResult.Segments,
			RenderMode:    renderMode,
			ElementScenes: elementScenes,
			Summary:       summary,
		}, warnings, nil
	}
	segmentResult, err := annotationSnapshotSegments(packageDir, snapshots)
	if err != nil {
		return annotationTimeline{}, nil, err
	}
	warnings = append(warnings, segmentResult.Warnings...)
	summary.ExportedSnapshotCount = len(segmentResult.Segments)
	summary.SkippedSnapshotCount = segmentResult.SkippedCount
	summary.SnapshotBytes = segmentResult.TotalBytes
	if segmentResult.TotalBytes > warnAnnotationTimelineSnapshotBytes {
		warnings = append(warnings, fmt.Sprintf("annotation timeline snapshots use %s; verify long recording export storage before composing", formatBytes(segmentResult.TotalBytes)))
	}
	if len(segmentResult.Segments) > 0 {
		segments := segmentResult.Segments
		summary.Mode = annotationTimelineSnapshotSegments
		summary.StartOffsetMs = segments[0].StartOffsetMs
		summary.EndOffsetMs = segments[len(segments)-1].StartOffsetMs
		if len(segments) > 1 && segments[len(segments)-2].EndOffsetMs > summary.EndOffsetMs {
			summary.EndOffsetMs = segments[len(segments)-2].EndOffsetMs
		}
		return annotationTimeline{
			StartOffsetMs: segments[0].StartOffsetMs,
			Mode:          annotationTimelineSnapshotSegments,
			Snapshots:     segments,
			RenderMode:    renderMode,
			ElementScenes: elementScenes,
			Summary:       summary,
		}, warnings, nil
	}
	if firstElementOffset >= 0 {
		summary.Mode = annotationTimelineSnapshotFromFirstElement
		summary.StartOffsetMs = firstElementOffset
		return annotationTimeline{StartOffsetMs: firstElementOffset, Mode: annotationTimelineSnapshotFromFirstElement, RenderMode: renderMode, ElementScenes: elementScenes, Summary: summary}, warnings, nil
	}
	if firstSnapshotOffset >= 0 {
		summary.Mode = annotationTimelineSnapshotAnchor
		summary.StartOffsetMs = firstSnapshotOffset
		warnings = append(warnings, "annotation events only contain snapshot anchors; export will show the final annotation snapshot from the first saved snapshot")
		return annotationTimeline{StartOffsetMs: firstSnapshotOffset, Mode: annotationTimelineSnapshotAnchor, RenderMode: renderMode, ElementScenes: elementScenes, Summary: summary}, warnings, nil
	}
	summary.Mode = annotationTimelineSnapshotOnly
	warnings = append(warnings, "annotation events file has no timeline offsets; export will compose the final annotation snapshot for the full video")
	return annotationTimeline{Mode: annotationTimelineSnapshotOnly, Summary: summary}, warnings, nil
}

func annotationSnapshotSegments(packageDir string, candidates []annotationSnapshotCandidate) (annotationSnapshotSegmentsResult, error) {
	if len(candidates) == 0 {
		return annotationSnapshotSegmentsResult{}, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].StartOffsetMs == candidates[j].StartOffsetMs {
			return candidates[i].RelativePath < candidates[j].RelativePath
		}
		return candidates[i].StartOffsetMs < candidates[j].StartOffsetMs
	})
	segments := make([]AnnotationSnapshotPlan, 0, len(candidates))
	warnings := make([]string, 0)
	seen := map[string]bool{}
	skippedCount := 0
	totalBytes := int64(0)
	for index, candidate := range candidates {
		key := fmt.Sprintf("%013d:%s", candidate.StartOffsetMs, candidate.RelativePath)
		if seen[key] {
			continue
		}
		seen[key] = true
		if len(segments) >= maxAnnotationTimelineSnapshots {
			warnings = append(warnings, fmt.Sprintf("annotation snapshot timeline has more than %d snapshots; later snapshots are skipped for export", maxAnnotationTimelineSnapshots))
			skippedCount += len(candidates) - index
			break
		}
		inputPath, err := optionalPackagePath(packageDir, "annotation snapshotPath", candidate.RelativePath)
		if err != nil {
			return annotationSnapshotSegmentsResult{}, err
		}
		info, err := os.Stat(inputPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("annotation snapshot %q is missing; it will be skipped", candidate.RelativePath))
			skippedCount++
			continue
		}
		if info.IsDir() || info.Size() == 0 {
			warnings = append(warnings, fmt.Sprintf("annotation snapshot %q is empty; it will be skipped", candidate.RelativePath))
			skippedCount++
			continue
		}
		totalBytes += info.Size()
		segments = append(segments, AnnotationSnapshotPlan{
			InputPath:     inputPath,
			RelativePath:  filepath.ToSlash(candidate.RelativePath),
			StartOffsetMs: candidate.StartOffsetMs,
			Bytes:         info.Size(),
		})
	}
	for index := 0; index < len(segments)-1; index++ {
		nextStart := segments[index+1].StartOffsetMs
		if nextStart > segments[index].StartOffsetMs {
			segments[index].EndOffsetMs = nextStart
			segments[index].DurationMs = nextStart - segments[index].StartOffsetMs
		}
	}
	return annotationSnapshotSegmentsResult{
		Segments:     segments,
		Warnings:     warnings,
		SkippedCount: skippedCount,
		TotalBytes:   totalBytes,
	}, nil
}

func annotationRenderedElementSegments(packageDir string, scenes []AnnotationElementScenePlan) (annotationSnapshotSegmentsResult, bool, error) {
	if len(scenes) == 0 {
		return annotationSnapshotSegmentsResult{}, false, nil
	}
	segments := make([]AnnotationSnapshotPlan, 0, len(scenes))
	warnings := make([]string, 0)
	totalBytes := int64(0)
	for index, scene := range scenes {
		relativePath := strings.TrimSpace(scene.RenderRelativePath)
		if relativePath == "" {
			relativePath = annotationElementSceneRenderPath(index + 1)
		}
		if err := validatePackageRelativePath("annotation rendered PNG path", relativePath); err != nil {
			return annotationSnapshotSegmentsResult{}, false, err
		}
		if !isRenderedAnnotationPNGPath(relativePath) {
			return annotationSnapshotSegmentsResult{}, false, fmt.Errorf("annotation rendered PNG path %q must stay under %s", relativePath, recpackage.AnnotationRenderPNGDir)
		}
		inputPath := filepath.Join(packageDir, filepath.Clean(relativePath))
		info, err := os.Stat(inputPath)
		if err != nil {
			return annotationSnapshotSegmentsResult{}, false, nil
		}
		if info.IsDir() || info.Size() == 0 {
			warnings = append(warnings, fmt.Sprintf("annotation rendered PNG %q is empty; snapshot timeline will be used", relativePath))
			return annotationSnapshotSegmentsResult{Warnings: warnings, SkippedCount: 1}, false, nil
		}
		totalBytes += info.Size()
		segments = append(segments, AnnotationSnapshotPlan{
			InputPath:     inputPath,
			RelativePath:  filepath.ToSlash(relativePath),
			StartOffsetMs: scene.StartOffsetMs,
			EndOffsetMs:   scene.EndOffsetMs,
			DurationMs:    scene.DurationMs,
			Bytes:         info.Size(),
		})
	}
	for index := 0; index < len(segments)-1; index++ {
		nextStart := segments[index+1].StartOffsetMs
		if nextStart > segments[index].StartOffsetMs {
			segments[index].EndOffsetMs = nextStart
			segments[index].DurationMs = nextStart - segments[index].StartOffsetMs
		}
	}
	return annotationSnapshotSegmentsResult{
		Segments:   segments,
		Warnings:   warnings,
		TotalBytes: totalBytes,
	}, true, nil
}

func isRenderedAnnotationPNGPath(value string) bool {
	value = filepath.ToSlash(strings.TrimSpace(value))
	prefix := filepath.ToSlash(recpackage.AnnotationRenderPNGDir) + "/"
	return strings.HasPrefix(value, prefix) && strings.ToLower(filepath.Ext(value)) == ".png"
}

func formatBytes(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KiB", "MiB", "GiB"}
	amount := float64(value)
	for _, unit := range units {
		amount = amount / 1024
		if amount < 1024 || unit == units[len(units)-1] {
			return fmt.Sprintf("%.1f %s", amount, unit)
		}
	}
	return fmt.Sprintf("%d B", value)
}

func annotationEventSnapshotPath(event map[string]any) string {
	value, ok := event["snapshotPath"].(string)
	if !ok {
		return ""
	}
	return filepath.ToSlash(strings.TrimSpace(value))
}

func isTimelineSnapshotPath(value string) bool {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	prefix := filepath.ToSlash(recpackage.AnnotationSnapshotsDir) + "/"
	return strings.HasPrefix(value, prefix)
}

func annotationEventOffsetMs(event map[string]any) (int64, bool) {
	if offset, ok := annotationNumberMs(event["recordingOffsetMs"]); ok {
		return offset, true
	}
	return annotationNumberMs(event["wallOffsetMs"])
}

func annotationNumberMs(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		if typed < 0 {
			return 0, false
		}
		return int64(typed), true
	case int64:
		if typed < 0 {
			return 0, false
		}
		return typed, true
	case int:
		if typed < 0 {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

func shouldIncludeAnnotations(explicit *bool, annotations *recpackage.ManifestAnnotations) bool {
	if annotations == nil || !annotations.Enabled {
		return false
	}
	if explicit != nil {
		return *explicit
	}
	return annotations.CapturePolicy == "export-compose"
}

func validatePackageLocation(videoDir string, packageDir string) (string, string, error) {
	if strings.TrimSpace(videoDir) == "" {
		return "", "", errors.New("videoDir is required")
	}
	if strings.TrimSpace(packageDir) == "" {
		return "", "", errors.New("packageDir is required")
	}
	videoRoot, err := filepath.Abs(videoDir)
	if err != nil {
		return "", "", err
	}
	target, err := filepath.Abs(packageDir)
	if err != nil {
		return "", "", err
	}
	if err := ensureInside(videoRoot, target); err != nil {
		return "", "", err
	}
	if !strings.HasSuffix(filepath.Base(target), recpackage.PackageDirSuffix) {
		return "", "", fmt.Errorf("packageDir %q must end with %s", packageDir, recpackage.PackageDirSuffix)
	}
	return videoRoot, target, nil
}

func ensureInside(root string, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("packageDir %q must be inside videoDir %q", target, root)
	}
	return nil
}

func validatePackageRelativePath(field string, value string) error {
	if value == "" {
		return nil
	}
	if filepath.IsAbs(value) {
		return fmt.Errorf("%s must be package-relative, got absolute path %q", field, value)
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s must stay inside the recording package, got %q", field, value)
	}
	return nil
}

func mediaInputPath(packageDir string, field string, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", fmt.Errorf("%s is required for export", field)
	}
	if err := validatePackageRelativePath(field, relativePath); err != nil {
		return "", err
	}
	target := filepath.Join(packageDir, filepath.Clean(relativePath))
	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("%s %q is not readable: %w", field, relativePath, err)
	}
	if info.IsDir() || info.Size() == 0 {
		return "", fmt.Errorf("%s %q is not readable media", field, relativePath)
	}
	return target, nil
}

func optionalPackagePath(packageDir string, field string, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", nil
	}
	if err := validatePackageRelativePath(field, relativePath); err != nil {
		return "", err
	}
	return filepath.Join(packageDir, filepath.Clean(relativePath)), nil
}

func canvasForPIP(canvas pip.Size, preset pip.Preset) pip.Size {
	if preset == pip.PresetOff {
		return pip.Size{Width: 1, Height: 1}
	}
	return canvas
}

func canvasFromManifest(manifest recpackage.Manifest) pip.Size {
	if manifest.Source.Geometry != nil && manifest.Source.Geometry.Width > 0 && manifest.Source.Geometry.Height > 0 {
		return pip.Size{Width: manifest.Source.Geometry.Width, Height: manifest.Source.Geometry.Height}
	}
	return pip.Size{}
}

func annotationCanvasSize(manifest recpackage.Manifest, fallback pip.Size) pip.Size {
	if manifest.Annotations != nil && manifest.Annotations.Target.Geometry != nil &&
		manifest.Annotations.Target.Geometry.Width > 0 && manifest.Annotations.Target.Geometry.Height > 0 {
		return pip.Size{Width: manifest.Annotations.Target.Geometry.Width, Height: manifest.Annotations.Target.Geometry.Height}
	}
	if manifest.Source.Geometry != nil && manifest.Source.Geometry.Width > 0 && manifest.Source.Geometry.Height > 0 {
		return pip.Size{Width: manifest.Source.Geometry.Width, Height: manifest.Source.Geometry.Height}
	}
	if fallback.Width > 1 && fallback.Height > 1 {
		return fallback
	}
	return canvasFromManifest(manifest)
}
