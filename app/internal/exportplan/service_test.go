package exportplan

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestPlanReadyPackageWithPIPAndSyncDiagnostics(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Camera: true})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		Canvas:      pip.Size{Width: 1920, Height: 1080},
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.PackageDir != packageDir {
		t.Fatalf("packageDir = %q, want %q", plan.PackageDir, packageDir)
	}
	if plan.OutputPath != filepath.Join(packageDir, DefaultOutputPath) {
		t.Fatalf("output path = %q, want package exports output", plan.OutputPath)
	}
	if plan.ScreenInputPath != filepath.Join(packageDir, "screen.mp4") {
		t.Fatalf("screen input = %q, want package screen.mp4", plan.ScreenInputPath)
	}
	if plan.WebcamInputPath != filepath.Join(packageDir, "webcam.mov") {
		t.Fatalf("webcam input = %q, want package webcam.mov", plan.WebcamInputPath)
	}
	if plan.WebcamStartOffsetMs != 120 {
		t.Fatalf("webcam offset = %d, want 120", plan.WebcamStartOffsetMs)
	}
	if plan.PIPPreset != string(pip.PresetFree) || !plan.PIPRect.Visible {
		t.Fatalf("pip = preset:%q rect:%#v, want visible free layout", plan.PIPPreset, plan.PIPRect)
	}
	if plan.PIPConfig.Shape != pip.ShapeSquare || plan.PIPConfig.Mirror || plan.PIPConfig.EdgeFeather != 0.2 {
		t.Fatalf("pip config = %#v, want square non-mirrored feathered layout", plan.PIPConfig)
	}
	if !plan.PIPLayout.Visible || plan.PIPLayout.Shape != pip.ShapeSquare || plan.PIPLayout.Mirror {
		t.Fatalf("pip layout = %#v, want visible square non-mirrored layout", plan.PIPLayout)
	}
	if plan.PIPRect.X >= 1920/2 || plan.PIPRect.Y <= 1080/2 {
		t.Fatalf("pip rect = %#v, want lower-left-ish free overlay", plan.PIPRect)
	}
	if plan.TimelineBase != recpackage.TimelineBaseMedia {
		t.Fatalf("timeline base = %q, want media timestamp", plan.TimelineBase)
	}
	if len(plan.PauseSegments) != 1 || plan.PauseSegments[0].DurationMs != 250 {
		t.Fatalf("pause segments = %#v, want one 250ms pause", plan.PauseSegments)
	}
}

func TestPlanUsesManifestSourceGeometryAsPIPCanvasFallback(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Camera: true, SourceGeometry: true})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:   videoDir,
		PackageDir: packageDir,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !plan.PIPRect.Visible || plan.PIPRect.Width <= 0 || plan.PIPRect.X >= 1280 {
		t.Fatalf("pip rect = %#v, want placement derived from manifest source geometry", plan.PIPRect)
	}
}

func TestPlanScreenOnlyPackageHidesPIP(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(screen only) error = %v", err)
	}
	if plan.PIPRect.Visible || plan.WebcamInputPath != "" || plan.WebcamStartOffsetMs != 0 {
		t.Fatalf("screen-only pip fields = rect:%#v webcam:%q offset:%d", plan.PIPRect, plan.WebcamInputPath, plan.WebcamStartOffsetMs)
	}
}

func TestPlanIncludesAnnotationSnapshot(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Annotations: true})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(with annotations) error = %v", err)
	}
	if !plan.AnnotationsVisible || plan.AnnotationInputPath != filepath.Join(packageDir, recpackage.AnnotationSnapshotFile) {
		t.Fatalf("annotation plan = visible:%v path:%q, want package annotation snapshot", plan.AnnotationsVisible, plan.AnnotationInputPath)
	}
}

func TestPlanIncludesAnnotationTimelineFromElementEvents(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{
		Annotations:    true,
		SourceGeometry: true,
		AnnotationEventsJSONL: strings.Join([]string{
			`{"type":"scene-snapshot","recordingOffsetMs":1200}`,
			`{"type":"element-created","elementId":"a","elementType":"freedraw","recordingOffsetMs":3456,"sequence":2,"element":{"id":"a","type":"freedraw","version":1}}`,
			`{"type":"element-updated","elementId":"a","elementType":"freedraw","elementVersion":2,"recordingOffsetMs":4567,"sequence":3,"element":{"id":"a","type":"freedraw","version":2}}`,
		}, "\n"),
	})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(with annotation timeline) error = %v", err)
	}
	if plan.AnnotationEventsPath != filepath.Join(packageDir, recpackage.AnnotationEventsFile) {
		t.Fatalf("annotation events path = %q, want package events path", plan.AnnotationEventsPath)
	}
	if plan.AnnotationStartMs != 3456 || plan.AnnotationTimeline != annotationTimelineSnapshotFromFirstElement {
		t.Fatalf("annotation timeline = %q @ %dms, want first element event at 3456ms", plan.AnnotationTimeline, plan.AnnotationStartMs)
	}
	if len(plan.Warnings) != 0 {
		t.Fatalf("warnings = %#v, want none for element timeline", plan.Warnings)
	}
	if plan.AnnotationSummary == nil ||
		plan.AnnotationSummary.ElementEventCount != 2 ||
		plan.AnnotationSummary.ElementTimelineMode != annotationElementTimelineReconstructed ||
		plan.AnnotationSummary.ElementKeyframeCount != 2 ||
		plan.AnnotationSummary.FinalElementCount != 1 ||
		plan.AnnotationSummary.MissingElementPayloads != 0 ||
		plan.AnnotationSummary.SnapshotCount != 1 ||
		plan.AnnotationSummary.ElementTypeCounts["freedraw"] != 1 ||
		len(plan.AnnotationSummary.ElementPreviewFrames) != 2 ||
		plan.AnnotationSummary.ElementPreviewFrames[1].ActiveElementCount != 1 {
		t.Fatalf("annotation summary = %#v, want element and snapshot counts", plan.AnnotationSummary)
	}
}

func TestPlanPreparesAnnotationElementSceneAssetsWhenRequested(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{
		Annotations:    true,
		SourceGeometry: true,
		AnnotationEventsJSONL: strings.Join([]string{
			`{"type":"scene-snapshot","recordingOffsetMs":1200}`,
			`{"type":"element-created","elementId":"a","elementType":"freedraw","recordingOffsetMs":1500,"sequence":2,"element":{"id":"a","type":"freedraw","version":1}}`,
			`{"type":"element-updated","elementId":"a","elementType":"freedraw","recordingOffsetMs":2500,"sequence":3,"element":{"id":"a","type":"freedraw","version":2}}`,
			`{"type":"element-deleted","elementId":"a","elementType":"freedraw","recordingOffsetMs":3500,"sequence":4,"isDeleted":true}`,
		}, "\n"),
	})

	previewPlan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(preview annotation assets) error = %v", err)
	}
	if previewPlan.AnnotationRenderMode != "" || len(previewPlan.AnnotationElementScenes) != 0 {
		t.Fatalf("preview plan render mode/scenes = %q/%#v, want no generated assets", previewPlan.AnnotationRenderMode, previewPlan.AnnotationElementScenes)
	}

	plan, err := NewService(nil).Plan(Request{
		VideoDir:                videoDir,
		PackageDir:              packageDir,
		RequireSync:             true,
		PrepareAnnotationAssets: true,
	})
	if err != nil {
		t.Fatalf("Plan(prepare annotation assets) error = %v", err)
	}
	if plan.AnnotationRenderMode != annotationRenderModeElementScenes {
		t.Fatalf("annotation render mode = %q, want element scene assets", plan.AnnotationRenderMode)
	}
	if len(plan.AnnotationElementScenes) != 3 {
		t.Fatalf("annotation element scenes = %#v, want three scene assets", plan.AnnotationElementScenes)
	}
	first := plan.AnnotationElementScenes[0]
	if first.RelativePath != filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderDir, "scene-000001.excalidraw")) ||
		first.RenderRelativePath != filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, "annotation-000001.png")) ||
		first.StartOffsetMs != 1500 ||
		first.EndOffsetMs != 2500 ||
		first.DurationMs != 1000 ||
		first.CanvasWidth != 1280 ||
		first.CanvasHeight != 720 ||
		first.ElementCount != 1 ||
		first.SourceEventSequence != 2 ||
		first.Bytes == 0 {
		t.Fatalf("first scene = %#v, want first reconstructed scene asset", first)
	}
	data, err := os.ReadFile(first.InputPath)
	if err != nil {
		t.Fatalf("ReadFile(first scene) error = %v", err)
	}
	var scene map[string]any
	if err := json.Unmarshal(data, &scene); err != nil {
		t.Fatalf("first scene JSON invalid: %v", err)
	}
	elements, ok := scene["elements"].([]any)
	if !ok || len(elements) != 1 {
		t.Fatalf("scene elements = %#v, want one reconstructed element", scene["elements"])
	}
	element, ok := elements[0].(map[string]any)
	if !ok || element["id"] != "a" || element["type"] != "freedraw" {
		t.Fatalf("scene element = %#v, want reconstructed freedraw element", elements[0])
	}
	last := plan.AnnotationElementScenes[len(plan.AnnotationElementScenes)-1]
	if last.ElementCount != 0 || last.EndOffsetMs != 0 {
		t.Fatalf("last scene = %#v, want delete keyframe with empty open-ended scene", last)
	}
}

func TestPlanPrefersRenderedAnnotationElementPNGsWhenComplete(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{
		Annotations:    true,
		SourceGeometry: true,
		AnnotationEventsJSONL: strings.Join([]string{
			`{"type":"element-created","elementId":"a","elementType":"rectangle","recordingOffsetMs":1000,"sequence":1,"element":{"id":"a","type":"rectangle","version":1}}`,
			`{"type":"element-updated","elementId":"a","elementType":"rectangle","recordingOffsetMs":2200,"sequence":2,"element":{"id":"a","type":"rectangle","version":2}}`,
		}, "\n"),
	})
	renderDir := filepath.Join(packageDir, recpackage.AnnotationRenderPNGDir)
	if err := os.MkdirAll(renderDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(render png dir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(renderDir, "annotation-000001.png"), []byte("rendered png 1"), 0o644); err != nil {
		t.Fatalf("WriteFile(rendered png 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(renderDir, "annotation-000002.png"), []byte("rendered png 2"), 0o644); err != nil {
		t.Fatalf("WriteFile(rendered png 2) error = %v", err)
	}

	plan, err := NewService(nil).Plan(Request{
		VideoDir:                videoDir,
		PackageDir:              packageDir,
		RequireSync:             true,
		PrepareAnnotationAssets: true,
	})
	if err != nil {
		t.Fatalf("Plan(rendered annotation PNGs) error = %v", err)
	}
	if plan.AnnotationTimeline != annotationTimelineElementPNGs || plan.AnnotationRenderMode != annotationRenderModeElementPNGs {
		t.Fatalf("annotation timeline/render = %q/%q, want rendered element PNGs", plan.AnnotationTimeline, plan.AnnotationRenderMode)
	}
	if len(plan.AnnotationSnapshots) != 2 {
		t.Fatalf("annotation snapshots = %#v, want rendered PNG timeline", plan.AnnotationSnapshots)
	}
	first := plan.AnnotationSnapshots[0]
	if first.RelativePath != filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, "annotation-000001.png")) ||
		first.StartOffsetMs != 1000 ||
		first.EndOffsetMs != 2200 ||
		first.DurationMs != 1200 ||
		first.Bytes == 0 {
		t.Fatalf("first rendered segment = %#v, want 1000..2200 rendered PNG", first)
	}
	if plan.AnnotationSummary == nil ||
		plan.AnnotationSummary.Mode != annotationTimelineElementPNGs ||
		plan.AnnotationSummary.ExportedSnapshotCount != 2 ||
		plan.AnnotationSummary.ElementTimelineMode != annotationElementTimelineReconstructed {
		t.Fatalf("annotation summary = %#v, want rendered element PNG summary", plan.AnnotationSummary)
	}
}

func TestPlanReportsPartialAnnotationElementReconstruction(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{
		Annotations: true,
		AnnotationEventsJSONL: strings.Join([]string{
			`{"type":"scene-snapshot","recordingOffsetMs":1200}`,
			`{"type":"element-created","elementId":"a","elementType":"freedraw","recordingOffsetMs":3456}`,
			`{"type":"element-deleted","elementId":"a","recordingOffsetMs":4567}`,
		}, "\n"),
	})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(partial annotation reconstruction) error = %v", err)
	}
	if plan.AnnotationSummary == nil ||
		plan.AnnotationSummary.ElementTimelineMode != annotationElementTimelinePartial ||
		plan.AnnotationSummary.ElementKeyframeCount != 2 ||
		plan.AnnotationSummary.FinalElementCount != 0 ||
		plan.AnnotationSummary.DeletedElementCount != 1 ||
		plan.AnnotationSummary.MissingElementPayloads != 1 {
		t.Fatalf("annotation summary = %#v, want partial element reconstruction", plan.AnnotationSummary)
	}
	if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "element payload") {
		t.Fatalf("warnings = %#v, want partial reconstruction warning", plan.Warnings)
	}
}

func TestPlanIncludesAnnotationSnapshotSegments(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{
		Annotations: true,
		AnnotationEventsJSONL: strings.Join([]string{
			`{"type":"scene-snapshot","recordingOffsetMs":1200,"snapshotPath":"annotations/snapshots/annotation-000001.png"}`,
			`{"type":"element-created","elementId":"a","elementType":"rectangle","recordingOffsetMs":1200,"sequence":2,"element":{"id":"a","type":"rectangle","version":1}}`,
			`{"type":"scene-snapshot","recordingOffsetMs":3456,"snapshotPath":"annotations/snapshots/annotation-000003.png"}`,
			`{"type":"element-updated","elementId":"a","elementType":"rectangle","recordingOffsetMs":3456,"sequence":4,"element":{"id":"a","type":"rectangle","version":2}}`,
		}, "\n"),
	})
	for _, name := range []string{"annotation-000001.png", "annotation-000003.png"} {
		path := filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(snapshot) error = %v", err)
		}
		if err := os.WriteFile(path, []byte("png media"), 0o644); err != nil {
			t.Fatalf("WriteFile(snapshot) error = %v", err)
		}
	}

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(with annotation snapshot segments) error = %v", err)
	}
	if plan.AnnotationTimeline != annotationTimelineSnapshotSegments || plan.AnnotationStartMs != 1200 {
		t.Fatalf("annotation timeline = %q @ %dms, want snapshot segments from 1200ms", plan.AnnotationTimeline, plan.AnnotationStartMs)
	}
	if len(plan.AnnotationSnapshots) != 2 {
		t.Fatalf("annotation snapshots = %#v, want two segment snapshots", plan.AnnotationSnapshots)
	}
	if plan.AnnotationSnapshots[0].InputPath != filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir, "annotation-000001.png") ||
		plan.AnnotationSnapshots[0].RelativePath != filepath.ToSlash(filepath.Join(recpackage.AnnotationSnapshotsDir, "annotation-000001.png")) ||
		plan.AnnotationSnapshots[0].StartOffsetMs != 1200 ||
		plan.AnnotationSnapshots[0].EndOffsetMs != 3456 ||
		plan.AnnotationSnapshots[0].DurationMs != 2256 ||
		plan.AnnotationSnapshots[0].Bytes == 0 {
		t.Fatalf("first annotation segment = %#v, want 1200..3456 snapshot", plan.AnnotationSnapshots[0])
	}
	if plan.AnnotationSnapshots[1].InputPath != filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir, "annotation-000003.png") ||
		plan.AnnotationSnapshots[1].RelativePath != filepath.ToSlash(filepath.Join(recpackage.AnnotationSnapshotsDir, "annotation-000003.png")) ||
		plan.AnnotationSnapshots[1].StartOffsetMs != 3456 ||
		plan.AnnotationSnapshots[1].EndOffsetMs != 0 ||
		plan.AnnotationSnapshots[1].DurationMs != 0 ||
		plan.AnnotationSnapshots[1].Bytes == 0 {
		t.Fatalf("second annotation segment = %#v, want open-ended 3456ms snapshot", plan.AnnotationSnapshots[1])
	}
	if plan.AnnotationSummary == nil ||
		plan.AnnotationSummary.Mode != annotationTimelineSnapshotSegments ||
		plan.AnnotationSummary.SnapshotCount != 2 ||
		plan.AnnotationSummary.ExportedSnapshotCount != 2 ||
		plan.AnnotationSummary.SkippedSnapshotCount != 0 ||
		plan.AnnotationSummary.ElementEventCount != 2 ||
		plan.AnnotationSummary.ElementTimelineMode != annotationElementTimelineReconstructed ||
		plan.AnnotationSummary.ElementKeyframeCount != 2 ||
		plan.AnnotationSummary.FinalElementCount != 1 ||
		plan.AnnotationSummary.ElementTypeCounts["rectangle"] != 1 ||
		plan.AnnotationSummary.EventFileBytes == 0 ||
		plan.AnnotationSummary.SnapshotBytes == 0 {
		t.Fatalf("annotation summary = %#v, want segment summary", plan.AnnotationSummary)
	}
	if len(plan.Warnings) != 0 {
		t.Fatalf("warnings = %#v, want none for readable snapshot timeline", plan.Warnings)
	}
}

func TestPlanReportsSkippedAnnotationSnapshotSegments(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{
		Annotations: true,
		AnnotationEventsJSONL: strings.Join([]string{
			`{"type":"scene-snapshot","recordingOffsetMs":1000,"snapshotPath":"annotations/snapshots/annotation-000001.png"}`,
			`{"type":"scene-snapshot","recordingOffsetMs":2000,"snapshotPath":"annotations/snapshots/annotation-000002.png"}`,
		}, "\n"),
	})
	firstSnapshot := filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir, "annotation-000001.png")
	if err := os.MkdirAll(filepath.Dir(firstSnapshot), 0o755); err != nil {
		t.Fatalf("MkdirAll(snapshot) error = %v", err)
	}
	if err := os.WriteFile(firstSnapshot, []byte("png media"), 0o644); err != nil {
		t.Fatalf("WriteFile(snapshot) error = %v", err)
	}

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(with skipped annotation snapshot) error = %v", err)
	}
	if len(plan.AnnotationSnapshots) != 1 {
		t.Fatalf("annotation snapshots = %#v, want one readable segment", plan.AnnotationSnapshots)
	}
	if plan.AnnotationSummary == nil || plan.AnnotationSummary.ExportedSnapshotCount != 1 || plan.AnnotationSummary.SkippedSnapshotCount != 1 {
		t.Fatalf("annotation summary = %#v, want exported/skipped snapshot counts", plan.AnnotationSummary)
	}
	if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "annotation-000002.png") {
		t.Fatalf("warnings = %#v, want missing snapshot warning", plan.Warnings)
	}
}

func TestAnnotationTimelineCapsSnapshotSegmentsForLongRecordings(t *testing.T) {
	packageDir := t.TempDir()
	eventsPath := filepath.Join(packageDir, recpackage.AnnotationEventsFile)
	snapshotsDir := filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir)
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(snapshots) error = %v", err)
	}
	eventsFile, err := os.Create(eventsPath)
	if err != nil {
		t.Fatalf("Create(events) error = %v", err)
	}
	writer := bufio.NewWriter(eventsFile)
	snapshotTotal := maxAnnotationTimelineSnapshots + 3
	for index := 1; index <= snapshotTotal; index++ {
		name := fmt.Sprintf("annotation-%06d.png", index)
		relativePath := filepath.ToSlash(filepath.Join(recpackage.AnnotationSnapshotsDir, name))
		if err := os.WriteFile(filepath.Join(snapshotsDir, name), []byte("png media"), 0o644); err != nil {
			t.Fatalf("WriteFile(snapshot %d) error = %v", index, err)
		}
		if _, err := fmt.Fprintf(writer, `{"type":"scene-snapshot","recordingOffsetMs":%d,"snapshotPath":%q}`+"\n", index*1000, relativePath); err != nil {
			t.Fatalf("Write(events %d) error = %v", index, err)
		}
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush(events) error = %v", err)
	}
	if err := eventsFile.Close(); err != nil {
		t.Fatalf("Close(events) error = %v", err)
	}

	timeline, warnings, err := readAnnotationTimeline(packageDir, eventsPath, false, pip.Size{})
	if err != nil {
		t.Fatalf("readAnnotationTimeline() error = %v", err)
	}
	if timeline.Mode != annotationTimelineSnapshotSegments || len(timeline.Snapshots) != maxAnnotationTimelineSnapshots {
		t.Fatalf("timeline = %q with %d snapshots, want capped snapshot segments", timeline.Mode, len(timeline.Snapshots))
	}
	if timeline.Summary.SnapshotCount != snapshotTotal ||
		timeline.Summary.ExportedSnapshotCount != maxAnnotationTimelineSnapshots ||
		timeline.Summary.SkippedSnapshotCount != snapshotTotal-maxAnnotationTimelineSnapshots {
		t.Fatalf("summary = %#v, want exported capped and skipped overflow snapshots", timeline.Summary)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], fmt.Sprintf("more than %d snapshots", maxAnnotationTimelineSnapshots)) {
		t.Fatalf("warnings = %#v, want snapshot cap warning", warnings)
	}
}

func TestAnnotationTimelineRejectsExcessiveEventCount(t *testing.T) {
	packageDir := t.TempDir()
	eventsPath := filepath.Join(packageDir, recpackage.AnnotationEventsFile)
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(events) error = %v", err)
	}
	eventsFile, err := os.Create(eventsPath)
	if err != nil {
		t.Fatalf("Create(events) error = %v", err)
	}
	writer := bufio.NewWriter(eventsFile)
	for index := 1; index <= maxAnnotationTimelineEvents+1; index++ {
		if _, err := fmt.Fprintf(writer, `{"type":"scene-snapshot","recordingOffsetMs":%d}`+"\n", index); err != nil {
			t.Fatalf("Write(events %d) error = %v", index, err)
		}
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush(events) error = %v", err)
	}
	if err := eventsFile.Close(); err != nil {
		t.Fatalf("Close(events) error = %v", err)
	}

	_, _, err = readAnnotationTimeline(packageDir, eventsPath, false, pip.Size{})
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("more than %d events", maxAnnotationTimelineEvents)) {
		t.Fatalf("readAnnotationTimeline() error = %v, want event cap error", err)
	}
}

func TestAnnotationElementSceneAssetsAreCappedForLongRecordings(t *testing.T) {
	reconstructor := newAnnotationElementReconstructor(true)
	for index := 1; index <= maxAnnotationElementSceneAssets+1; index++ {
		reconstructor.Apply(map[string]any{
			"type":              "element-updated",
			"elementId":         "active-element",
			"elementType":       "rectangle",
			"elementVersion":    index,
			"recordingOffsetMs": index * 1000,
			"sequence":          index,
			"element": map[string]any{
				"id":      "active-element",
				"type":    "rectangle",
				"version": index,
			},
		})
	}

	scenes, warnings, err := reconstructor.BuildSceneAssets(t.TempDir(), 1280, 720)
	if err != nil {
		t.Fatalf("BuildSceneAssets() error = %v", err)
	}
	if len(scenes) != 0 {
		t.Fatalf("scenes = %#v, want no generated scenes beyond long recording cap", scenes)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], fmt.Sprintf("exceeds the %d scene asset limit", maxAnnotationElementSceneAssets)) {
		t.Fatalf("warnings = %#v, want scene asset cap warning", warnings)
	}
}

func TestPlanSkipsAnnotationSnapshotWhenRequestDisablesAnnotations(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Annotations: true})
	includeAnnotations := false

	plan, err := NewService(nil).Plan(Request{
		VideoDir:           videoDir,
		PackageDir:         packageDir,
		RequireSync:        true,
		IncludeAnnotations: &includeAnnotations,
	})
	if err != nil {
		t.Fatalf("Plan(with annotations disabled) error = %v", err)
	}
	if plan.AnnotationsVisible || plan.AnnotationInputPath != "" {
		t.Fatalf("annotation plan = visible:%v path:%q, want annotations skipped", plan.AnnotationsVisible, plan.AnnotationInputPath)
	}
}

func TestPlanIncludesPreviewOnlyAnnotationWhenRequestEnablesAnnotations(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Annotations: true, AnnotationCapturePolicy: "preview-only"})
	includeAnnotations := true

	plan, err := NewService(nil).Plan(Request{
		VideoDir:           videoDir,
		PackageDir:         packageDir,
		RequireSync:        true,
		IncludeAnnotations: &includeAnnotations,
	})
	if err != nil {
		t.Fatalf("Plan(preview-only annotations explicitly enabled) error = %v", err)
	}
	if !plan.AnnotationsVisible || plan.AnnotationInputPath != filepath.Join(packageDir, recpackage.AnnotationSnapshotFile) {
		t.Fatalf("annotation plan = visible:%v path:%q, want explicit annotation snapshot", plan.AnnotationsVisible, plan.AnnotationInputPath)
	}
}

func TestPlanRejectsPackageOutsideVideoDir(t *testing.T) {
	videoDir := t.TempDir()
	outsideDir := createReadyPackage(t, t.TempDir(), readyPackageOptions{})

	if _, err := NewService(nil).Plan(Request{VideoDir: videoDir, PackageDir: outsideDir}); err == nil {
		t.Fatal("Plan() accepted a package outside videoDir")
	}
}

func TestPlanRejectsEscapingOutputPath(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{})

	if _, err := NewService(nil).Plan(Request{
		VideoDir:   videoDir,
		PackageDir: packageDir,
		OutputPath: "../export.mp4",
	}); err == nil {
		t.Fatal("Plan() accepted an escaping output path")
	}
}

func TestPlanRejectsMockPackageByDefault(t *testing.T) {
	videoDir := t.TempDir()
	pkg, err := recpackage.NewService().CreateMock(videoDir, recpackage.CreateMockRequest{
		Status: recpackage.StatusReady,
		Source: recpackage.ManifestSource{Type: "screen", ID: "screen:primary"},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	if _, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  pkg.Dir,
		RequireSync: true,
	}); err == nil {
		t.Fatal("Plan() accepted a mock package as real media")
	}
}

func TestPlanRejectsMissingWebcamSidecarForVisiblePIP(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Camera: true, MissingWebcam: true})

	if _, err := NewService(nil).Plan(Request{
		VideoDir:   videoDir,
		PackageDir: packageDir,
		Canvas:     pip.Size{Width: 1920, Height: 1080},
	}); err == nil || !strings.Contains(err.Error(), "webcamVideoPath") {
		t.Fatalf("Plan() error = %v, want webcamVideoPath error", err)
	}
}

func TestPlanRejectsEscapingDiagnosticsPath(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{EscapingDiagnostics: true})

	if _, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	}); err == nil {
		t.Fatal("Plan() accepted an escaping diagnostics path")
	}
}

type readyPackageOptions struct {
	Camera                  bool
	MissingWebcam           bool
	EscapingDiagnostics     bool
	SourceGeometry          bool
	Annotations             bool
	AnnotationCapturePolicy string
	AnnotationEventsJSONL   string
}

func createReadyPackage(t *testing.T, videoDir string, opts readyPackageOptions) string {
	t.Helper()
	packageDir := filepath.Join(videoDir, "recording-export-test"+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(package) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "screen.mp4"), []byte("screen media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen) error = %v", err)
	}
	if opts.Annotations {
		if err := os.MkdirAll(filepath.Join(packageDir, recpackage.AnnotationExportsDir), 0o755); err != nil {
			t.Fatalf("MkdirAll(annotation exports) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(packageDir, recpackage.AnnotationSnapshotFile), []byte("png media"), 0o644); err != nil {
			t.Fatalf("WriteFile(annotation) error = %v", err)
		}
		if opts.AnnotationEventsJSONL != "" {
			if err := os.WriteFile(filepath.Join(packageDir, recpackage.AnnotationEventsFile), []byte(opts.AnnotationEventsJSONL+"\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(annotation events) error = %v", err)
			}
		}
	}
	media := recpackage.ManifestMedia{ScreenVideoPath: "screen.mp4"}
	camera := recpackage.ManifestCamera{PIPPreset: string(pip.PresetOff)}
	webcamSync := recpackage.ManifestTrackDiagnostics{}
	if opts.Camera {
		camera = recpackage.ManifestCamera{
			Enabled:   true,
			DeviceID:  "camera:default",
			PIPPreset: string(pip.PresetBottomLeft),
			PIP: pip.Config{
				Preset:      pip.PresetFree,
				Shape:       pip.ShapeSquare,
				Mirror:      false,
				Position:    pip.Position{X: 0.1, Y: 0.9},
				Scale:       0.22,
				EdgeFeather: 0.2,
			},
		}
		media.WebcamVideoPath = "webcam.mov"
		media.WebcamStartOffsetMs = 120
		webcamSync = recpackage.ManifestTrackDiagnostics{
			Enabled:       true,
			Path:          "webcam.mov",
			Clock:         recpackage.TimelineBaseMedia,
			StartOffsetMs: 120,
			EndOffsetMs:   6120,
			DurationMs:    6000,
			FrameRate:     30,
		}
		if !opts.MissingWebcam {
			if err := os.WriteFile(filepath.Join(packageDir, "webcam.mov"), []byte("webcam media"), 0o644); err != nil {
				t.Fatalf("WriteFile(webcam) error = %v", err)
			}
		}
	}
	audioDiagnosticsPath := recpackage.AudioDiagnosticsFile
	if opts.EscapingDiagnostics {
		audioDiagnosticsPath = "../audio-diagnostics.json"
	}
	source := recpackage.ManifestSource{Type: "screen", ID: "screen:primary"}
	if opts.SourceGeometry {
		source.Geometry = &recpackage.ManifestSourceGeometry{Width: 1280, Height: 720}
	}
	manifest := recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		CreatedAt:     time.Date(2026, 6, 30, 18, 0, 0, 0, time.UTC),
		Status:        recpackage.StatusReady,
		Media:         media,
		Source:        source,
		Recording:     recordingprofile.Profile{Quality: recordingprofile.QualityHigh, FPS: 30, CaptureCursor: true},
		Camera:        camera,
		Diagnostics: recpackage.ManifestDiagnostics{
			Sync: &recpackage.ManifestSyncDiagnostics{
				TimelineBase:         recpackage.TimelineBaseMedia,
				AudioDiagnosticsPath: audioDiagnosticsPath,
				VideoDiagnosticsPath: recpackage.VideoDiagnosticsFile,
				Screen: recpackage.ManifestTrackDiagnostics{
					Enabled:       true,
					Path:          "screen.mp4",
					Clock:         recpackage.TimelineBaseMedia,
					EndOffsetMs:   6000,
					DurationMs:    6000,
					DroppedFrames: 0,
					FrameRate:     30,
				},
				Webcam:        webcamSync,
				PauseSegments: []recpackage.ManifestPauseSegment{{StartOffsetMs: 1000, EndOffsetMs: 1250, DurationMs: 250}},
			},
		},
	}
	if opts.Annotations {
		capturePolicy := opts.AnnotationCapturePolicy
		if capturePolicy == "" {
			capturePolicy = "export-compose"
		}
		manifest.Annotations = &recpackage.ManifestAnnotations{
			Enabled:       true,
			Mode:          "overlay",
			ScenePath:     recpackage.AnnotationSceneFile,
			EventsPath:    recpackage.AnnotationEventsFile,
			SnapshotPath:  recpackage.AnnotationSnapshotFile,
			CapturePolicy: capturePolicy,
			Target:        recpackage.ManifestAnnotationTarget{Type: "screen", ID: "screen:primary"},
		}
	}
	if opts.EscapingDiagnostics {
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatalf("MarshalIndent(manifest) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(packageDir, recpackage.ManifestFile), append(data, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile(manifest) error = %v", err)
		}
		return packageDir
	}
	if err := recpackage.NewService().WriteManifest(filepath.Join(packageDir, recpackage.ManifestFile), manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}
	return packageDir
}
