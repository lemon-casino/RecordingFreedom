package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestRunAcceptsCompleteEvidenceDirectory(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.OK {
		t.Fatalf("report.OK = false: %#v", report)
	}
	if len(report.Packages) != 4 {
		t.Fatalf("packages = %d, want 4", len(report.Packages))
	}
	if report.Packages[0].SourceType == "" || report.Packages[0].EventsPath == "" || report.Packages[0].DiagnosticsPath == "" || report.Packages[0].Snapshot == "" || report.Packages[0].DurationMs <= 0 || report.Packages[0].ExportDurationMs <= 0 {
		t.Fatalf("package report = %#v, want source/events/snapshot", report.Packages[0])
	}
}

func TestRunRejectsPackageWithoutAnnotations(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, false)

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "annotations.enabled") {
		t.Fatalf("package report = %#v, want missing annotations block", report.Packages)
	}
}

func TestRunRejectsREADMEWithoutEvidenceChecklist(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	writeFile(t, filepath.Join(evidenceDir, "README.md"), "version: test\ncommit: test\n")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Name == "README.md" && check.Status == "blocked" && strings.Contains(check.Message, "missing evidence records") {
			found = true
		}
	}
	if !found {
		t.Fatalf("README check not blocked: %#v", report.Checks)
	}
}

func TestRunRejectsMissingScreenshotEvidenceMatrix(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	if err := os.Remove(filepath.Join(evidenceDir, "screenshots", "pass-through-click-selection.png")); err != nil {
		t.Fatalf("Remove(pass-through screenshot) error = %v", err)
	}

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Name == "screenshots" && check.Status == "blocked" && strings.Contains(check.Message, "pass-through") {
			found = true
		}
	}
	if !found {
		t.Fatalf("screenshots check not blocked: %#v", report.Checks)
	}
}

func TestRunRejectsMissingRecordingEvidenceMatrix(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	if err := os.Remove(filepath.Join(evidenceDir, "recordings", "export-without-annotations.mp4")); err != nil {
		t.Fatalf("Remove(without annotations recording) error = %v", err)
	}

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Name == "recordings" && check.Status == "blocked" && strings.Contains(check.Message, "without annotations") {
			found = true
		}
	}
	if !found {
		t.Fatalf("recordings check not blocked: %#v", report.Checks)
	}
}

func TestRunRejectsIncompleteAppLog(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	writeFile(t, filepath.Join(evidenceDir, "app-log.jsonl"),
		`{"component":"app","event":"startup"}`+"\n"+
			`{"component":"recording","event":"start-request","fields":{"sourceType":"screen"}}`+"\n")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Name == "app-log.jsonl" && check.Status == "blocked" && strings.Contains(check.Message, "annotation-overlay/show") {
			found = true
		}
	}
	if !found {
		t.Fatalf("app log check not blocked: %#v", report.Checks)
	}
}

func TestRunRejectsIncompletePlatformFile(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	writeFile(t, filepath.Join(evidenceDir, "platform.txt"), "windows 11\n")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Name == "platform.txt" && check.Status == "blocked" && strings.Contains(check.Message, "display environment") {
			found = true
		}
	}
	if !found {
		t.Fatalf("platform check not blocked: %#v", report.Checks)
	}
}

func TestRunRejectsPackageWithoutOverlayDiagnostics(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	diagnosticsPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, recpackage.AnnotationOverlayDiagnosticsFile)
	if err := os.Remove(diagnosticsPath); err != nil {
		t.Fatalf("Remove(diagnostics) error = %v", err)
	}

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "overlay diagnostics") {
		t.Fatalf("package report = %#v, want missing diagnostics block", report.Packages)
	}
}

func TestRunRejectsIncompleteOverlayDiagnostics(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	diagnosticsPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, recpackage.AnnotationOverlayDiagnosticsFile)
	writeFile(t, diagnosticsPath, annotationDiagnosticLine("show", "all-screens", "all-screens:virtual-desktop")+"\n")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "missing") {
		t.Fatalf("package report = %#v, want incomplete diagnostics block", report.Packages)
	}
}

func TestRunRejectsOverlayDiagnosticsWithoutPassThroughState(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	diagnosticsPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, recpackage.AnnotationOverlayDiagnosticsFile)
	writeFile(t, diagnosticsPath,
		annotationDiagnosticLine("show", "all-screens", "all-screens:virtual-desktop")+"\n"+
			annotationHitRegionsDiagnosticLine("all-screens", "all-screens:virtual-desktop", "drawing")+"\n"+
			annotationDiagnosticLine("save-capture", "all-screens", "all-screens:virtual-desktop")+"\n")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "pass-through") {
		t.Fatalf("package report = %#v, want missing pass-through diagnostics block", report.Packages)
	}
}

func TestRunRejectsOverlayDiagnosticsWithoutDrawingState(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	diagnosticsPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, recpackage.AnnotationOverlayDiagnosticsFile)
	writeFile(t, diagnosticsPath,
		annotationDiagnosticLine("show", "all-screens", "all-screens:virtual-desktop")+"\n"+
			annotationHitRegionsDiagnosticLine("all-screens", "all-screens:virtual-desktop", "pass-through")+"\n"+
			annotationDiagnosticLine("save-capture", "all-screens", "all-screens:virtual-desktop")+"\n")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "drawing") {
		t.Fatalf("package report = %#v, want missing drawing diagnostics block", report.Packages)
	}
}

func TestRunRejectsAnnotationEventsWithoutElementEvent(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	eventsPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, recpackage.AnnotationEventsFile)
	writeFile(t, eventsPath, annotationSceneSnapshotEvent()+"\n")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "element-created") {
		t.Fatalf("package report = %#v, want missing element event block", report.Packages)
	}
}

func TestRunRejectsExportedRecordingThatIsNotMP4(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	exportPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, filepath.FromSlash(exportplan.DefaultOutputPath))
	writeFile(t, exportPath, "mp4")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "ftyp") {
		t.Fatalf("package report = %#v, want invalid mp4 block", report.Packages)
	}
}

func TestRunRejectsExportedRecordingWithoutVideoTrack(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	exportPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, filepath.FromSlash(exportplan.DefaultOutputPath))
	writeMinimalEvidenceMP4(t, exportPath, "soun", 60_000)

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "video track") {
		t.Fatalf("package report = %#v, want missing video track block", report.Packages)
	}
}

func TestRunRejectsExportedRecordingDurationMismatch(t *testing.T) {
	evidenceDir := createEvidenceFixture(t, true)
	exportPath := filepath.Join(evidenceDir, "packages", "recording-overlay-all-screens-1m"+recpackage.PackageDirSuffix, filepath.FromSlash(exportplan.DefaultOutputPath))
	writeMinimalEvidenceMP4(t, exportPath, "vide", 10_000)

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	if len(report.Packages) != 4 || report.Packages[0].Status != "blocked" || !strings.Contains(report.Packages[0].Message, "durationMs") {
		t.Fatalf("package report = %#v, want duration mismatch block", report.Packages)
	}
}

func TestRunRejectsMissingFiveMinutePackage(t *testing.T) {
	evidenceDir := t.TempDir()
	createEvidenceBase(t, evidenceDir)
	createEvidencePackage(t, evidenceDir, "recording-overlay-all-screens-1m", true, 60_000, "all-screens", "all-screens:virtual-desktop")
	createEvidencePackage(t, evidenceDir, "recording-overlay-screen", true, 30_000, "screen", "screen:1")
	createEvidencePackage(t, evidenceDir, "recording-overlay-region", true, 30_000, "region", "region:custom")
	createEvidencePackage(t, evidenceDir, "recording-overlay-window", true, 30_000, "window", "window:focused")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Name == "5m recording package" && check.Status == "blocked" {
			found = true
		}
	}
	if !found {
		t.Fatalf("5m duration check not blocked: %#v", report.Checks)
	}
}

func TestRunRejectsMissingSourceMatrixPackage(t *testing.T) {
	evidenceDir := t.TempDir()
	createEvidenceBase(t, evidenceDir)
	createEvidencePackage(t, evidenceDir, "recording-overlay-all-screens-1m", true, 60_000, "all-screens", "all-screens:virtual-desktop")
	createEvidencePackage(t, evidenceDir, "recording-overlay-screen-5m", true, 300_000, "screen", "screen:1")
	createEvidencePackage(t, evidenceDir, "recording-overlay-region", true, 30_000, "region", "region:custom")

	report, err := run(evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Name == "source window package" && check.Status == "blocked" {
			found = true
		}
	}
	if !found {
		t.Fatalf("window source check not blocked: %#v", report.Checks)
	}
}

func TestPackageRelativePathRejectsEscapes(t *testing.T) {
	if _, err := packageRelativePath(t.TempDir(), "../outside"); err == nil {
		t.Fatal("packageRelativePath accepted escaping path")
	}
}

func createEvidenceFixture(t *testing.T, withAnnotations bool) string {
	t.Helper()
	evidenceDir := t.TempDir()
	createEvidenceBase(t, evidenceDir)
	createEvidencePackage(t, evidenceDir, "recording-overlay-all-screens-1m", withAnnotations, 60_000, "all-screens", "all-screens:virtual-desktop")
	createEvidencePackage(t, evidenceDir, "recording-overlay-screen-5m", withAnnotations, 300_000, "screen", "screen:1")
	createEvidencePackage(t, evidenceDir, "recording-overlay-region", withAnnotations, 30_000, "region", "region:custom")
	createEvidencePackage(t, evidenceDir, "recording-overlay-window", withAnnotations, 30_000, "window", "window:focused")
	return evidenceDir
}

func createEvidenceBase(t *testing.T, evidenceDir string) {
	t.Helper()
	writeFile(t, filepath.Join(evidenceDir, "README.md"), strings.Join([]string{
		"version: v0.0.0-test",
		"commit: abcdef123456",
		"artifact: local smoke artifact",
		"operating system: windows 11",
		"display count: 2",
		"resolution: 1920x1080 + 2560x1440",
		"scale: 100% / 150%",
		"source matrix: all-screens ready; screen ready; region ready; window ready",
		"click-through: selection pass; drawing pass; capsule control pass",
		"export results: with annotations pass; without annotations pass",
		"known failures: none; blockers: none",
		"",
	}, "\n"))
	writeFile(t, filepath.Join(evidenceDir, "platform.txt"), strings.Join([]string{
		"platform: windows",
		"version: windows 11 23h2 build 22631",
		"display count: 2",
		"display 1: 1920x1080 scale 100%",
		"display 2: 2560x1440 scale 150%",
		"",
	}, "\n"))
	writeFile(t, filepath.Join(evidenceDir, "app-log.jsonl"), appLogJSONL())
	writeFile(t, filepath.Join(evidenceDir, "screenshots", "source-all-screens.png"), "screenshot")
	writeFile(t, filepath.Join(evidenceDir, "screenshots", "source-screen-1.png"), "screenshot")
	writeFile(t, filepath.Join(evidenceDir, "screenshots", "source-region.png"), "screenshot")
	writeFile(t, filepath.Join(evidenceDir, "screenshots", "source-window.png"), "screenshot")
	writeFile(t, filepath.Join(evidenceDir, "screenshots", "pass-through-click-selection.png"), "screenshot")
	writeFile(t, filepath.Join(evidenceDir, "screenshots", "drawing-state.png"), "screenshot")
	writeFile(t, filepath.Join(evidenceDir, "screenshots", "capsule-controls.png"), "screenshot")
	writeFile(t, filepath.Join(evidenceDir, "recordings", "source-all-screens.mp4"), "recording")
	writeFile(t, filepath.Join(evidenceDir, "recordings", "source-screen-1.mp4"), "recording")
	writeFile(t, filepath.Join(evidenceDir, "recordings", "source-region.mp4"), "recording")
	writeFile(t, filepath.Join(evidenceDir, "recordings", "source-window.mp4"), "recording")
	writeFile(t, filepath.Join(evidenceDir, "recordings", "export-with-annotations.mp4"), "recording")
	writeFile(t, filepath.Join(evidenceDir, "recordings", "export-without-annotations.mp4"), "recording")
}

func appLogJSONL() string {
	lines := []string{
		`{"component":"app","event":"startup","fields":{"platform":"windows"}}`,
	}
	for _, sourceType := range []string{"all-screens", "screen", "region", "window"} {
		lines = append(lines,
			fmt.Sprintf(`{"component":"recording","event":"start-request","fields":{"sourceType":%q}}`, sourceType),
			fmt.Sprintf(`{"component":"annotation-overlay","event":"show","fields":{"targetType":%q,"targetId":%q}}`, sourceType, sourceType+":test"),
		)
	}
	lines = append(lines, `{"component":"annotation-overlay","event":"save-capture","fields":{"bytes":"1234"}}`)
	return strings.Join(lines, "\n") + "\n"
}

func createEvidencePackage(t *testing.T, evidenceDir string, name string, withAnnotations bool, durationMs int64, sourceType string, sourceID string) {
	t.Helper()
	packageDir := filepath.Join(evidenceDir, "packages", name+recpackage.PackageDirSuffix)
	writeFile(t, filepath.Join(packageDir, recpackage.ScreenVideoFile), "screen")
	writeFile(t, filepath.Join(packageDir, recpackage.AnnotationEventsFile), annotationEventsJSONL())
	writeFile(t, filepath.Join(packageDir, recpackage.AnnotationOverlayDiagnosticsFile),
		annotationDiagnosticLine("show", sourceType, sourceID)+"\n"+
			annotationHitRegionsDiagnosticLine(sourceType, sourceID, "pass-through")+"\n"+
			annotationHitRegionsDiagnosticLine(sourceType, sourceID, "drawing")+"\n"+
			annotationDiagnosticLine("save-capture", sourceType, sourceID)+"\n")
	writeFile(t, filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir, "annotation-000001.png"), "png")
	writeMinimalEvidenceMP4(t, filepath.Join(packageDir, filepath.FromSlash(exportplan.DefaultOutputPath)), "vide", durationMs)

	geometry := &recpackage.ManifestSourceGeometry{
		X:            -1280,
		Y:            48,
		Width:        1024,
		Height:       576,
		DisplayIndex: 2,
		NativeID:     "display-2",
	}
	createdAt := time.Now().UTC()
	completedAt := createdAt.Add(time.Duration(durationMs) * time.Millisecond)
	manifest := recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		CreatedAt:     createdAt,
		CompletedAt:   &completedAt,
		Status:        recpackage.StatusReady,
		RecordingMode: recpackage.RecordingModeScreen,
		Media: recpackage.ManifestMedia{
			ScreenVideoPath: recpackage.ScreenVideoFile,
		},
		Source: recpackage.ManifestSource{
			Type:     sourceType,
			ID:       sourceID,
			Name:     "Smoke " + sourceType,
			Geometry: geometry,
		},
		Recording: recordingprofile.Profile{
			FPS: 30,
		},
		Audio:  recpackage.ManifestAudio{MicrophoneNoiseSuppression: recpackage.NoiseSuppressionOff, SampleRate: 48000},
		Camera: recpackage.ManifestCamera{PIPPreset: "off"},
		Diagnostics: recpackage.ManifestDiagnostics{
			Sync: &recpackage.ManifestSyncDiagnostics{
				TimelineBase:          recpackage.TimelineBaseMedia,
				TimelineStartUnixNano: createdAt.UnixNano(),
				VideoDiagnosticsPath:  recpackage.VideoDiagnosticsFile,
				Screen: recpackage.ManifestTrackDiagnostics{
					Enabled:     true,
					Path:        recpackage.ScreenVideoFile,
					Clock:       recpackage.TimelineBaseMedia,
					EndOffsetMs: durationMs,
					DurationMs:  durationMs,
					FrameRate:   30,
				},
			},
		},
	}
	if withAnnotations {
		manifest.Annotations = &recpackage.ManifestAnnotations{
			Enabled:         true,
			Mode:            "overlay",
			ScenePath:       recpackage.AnnotationSceneFile,
			EventsPath:      recpackage.AnnotationEventsFile,
			SnapshotPath:    recpackage.AnnotationSnapshotFile,
			DiagnosticsPath: recpackage.AnnotationOverlayDiagnosticsFile,
			CapturePolicy:   "export-compose",
			Target: recpackage.ManifestAnnotationTarget{
				Type:     sourceType,
				ID:       sourceID,
				Geometry: geometry,
			},
		}
		writeFile(t, filepath.Join(packageDir, recpackage.AnnotationSnapshotFile), "png")
	}
	if err := recpackage.NewService().WriteManifest(filepath.Join(packageDir, recpackage.ManifestFile), manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}
}

func annotationDiagnosticLine(eventType string, sourceType string, sourceID string) string {
	return fmt.Sprintf(`{"schemaVersion":1,"type":%q,"windowBounds":{"x":-1280,"y":48,"width":1024,"height":576},"canvasBounds":{"x":0,"y":0,"width":1024,"height":576},"target":{"type":%q,"id":%q,"geometry":{"x":-1280,"y":48,"width":1024,"height":576,"displayIndex":2,"nativeId":"display-2"}}}`, eventType, sourceType, sourceID)
}

func annotationEventsJSONL() string {
	return annotationSceneSnapshotEvent() + "\n" +
		`{"type":"element-created","schemaVersion":1,"sequence":2,"eventId":"annotation-client-2","recordingOffsetMs":1200,"wallOffsetMs":1200,"scenePath":"annotations/scene.excalidraw","snapshotPath":"annotations/snapshots/annotation-000001.png","elementId":"stroke-1","elementType":"freedraw","elementVersion":1}` + "\n"
}

func annotationSceneSnapshotEvent() string {
	return `{"type":"scene-snapshot","schemaVersion":1,"sequence":1,"eventId":"annotation-000001","recordingOffsetMs":1000,"wallOffsetMs":1000,"scenePath":"annotations/scene.excalidraw","snapshotPath":"annotations/snapshots/annotation-000001.png"}`
}

func annotationHitRegionsDiagnosticLine(sourceType string, sourceID string, mode string) string {
	regions := `[{"x":430,"y":14,"width":420,"height":52,"kind":"pill","radius":999}]`
	if mode == "drawing" {
		regions = `[{"x":430,"y":14,"width":420,"height":52,"kind":"pill","radius":999},{"x":0,"y":0,"width":1024,"height":576,"kind":"rect"}]`
	}
	return fmt.Sprintf(`{"schemaVersion":1,"type":"hit-regions","windowBounds":{"x":-1280,"y":48,"width":1024,"height":576},"canvasBounds":{"x":0,"y":0,"width":1024,"height":576},"target":{"type":%q,"id":%q,"geometry":{"x":-1280,"y":48,"width":1024,"height":576,"displayIndex":2,"nativeId":"display-2"}},"hitRegions":{"enabled":true,"viewportWidth":1024,"viewportHeight":576,"devicePixelRatio":1,"regions":%s}}`, sourceType, sourceID, regions)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeMinimalEvidenceMP4(t *testing.T, path string, handlerType string, durationMs int64) {
	t.Helper()
	mvhdPayload := make([]byte, 20)
	binary.BigEndian.PutUint32(mvhdPayload[12:16], 1000)
	binary.BigEndian.PutUint32(mvhdPayload[16:20], uint32(durationMs))

	hdlrPayload := make([]byte, 0, 12)
	hdlrPayload = append(hdlrPayload, 0, 0, 0, 0)
	hdlrPayload = append(hdlrPayload, 0, 0, 0, 0)
	hdlrPayload = append(hdlrPayload, []byte(handlerType)...)

	moovPayload := mp4EvidenceBox("mvhd", mvhdPayload)
	moovPayload = append(moovPayload, mp4EvidenceBox("trak", mp4EvidenceBox("mdia", mp4EvidenceBox("hdlr", hdlrPayload)))...)
	data := mp4EvidenceBox("ftyp", []byte("isom0000"))
	data = append(data, mp4EvidenceBox("moov", moovPayload)...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mp4EvidenceBox(kind string, payload []byte) []byte {
	box := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(box)))
	copy(box[4:8], []byte(kind))
	copy(box[8:], payload)
	return box
}
