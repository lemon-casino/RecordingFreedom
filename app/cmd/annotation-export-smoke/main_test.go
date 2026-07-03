package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestBuildAnnotationTimelineCreatesMidpointSamples(t *testing.T) {
	timeline := buildAnnotationTimeline(5*time.Second, 5)
	if timeline.SwitchMs != 1000 {
		t.Fatalf("switch ms = %d, want 1000", timeline.SwitchMs)
	}
	if len(timeline.Samples) != 5 {
		t.Fatalf("samples = %#v, want five segments", timeline.Samples)
	}
	for index, sample := range timeline.Samples {
		wantSegment := index + 1
		wantAt := time.Duration(500+index*1000) * time.Millisecond
		if sample.Segment != wantSegment || sample.At != wantAt {
			t.Fatalf("sample[%d] = segment %d at %s, want segment %d at %s", index, sample.Segment, sample.At, wantSegment, wantAt)
		}
	}
	if first, second := timeline.Samples[0].Expected, timeline.Samples[1].Expected; first.R < 200 || second.G < 150 {
		t.Fatalf("first colors = %#v / %#v, want red then green defaults for compatibility", first, second)
	}
}

func TestWriteAnnotationAssetsCreatesRequestedSnapshotTimeline(t *testing.T) {
	packageDir := t.TempDir()
	timeline, err := writeAnnotationAssets(packageDir, 64, 48, 4*time.Second, 4, timelineModeSnapshot)
	if err != nil {
		t.Fatalf("writeAnnotationAssets() error = %v", err)
	}
	if len(timeline.Samples) != 4 {
		t.Fatalf("timeline samples = %#v, want four", timeline.Samples)
	}
	for index := 1; index <= 4; index++ {
		path := filepath.Join(packageDir, filepath.FromSlash(timelineSnapshotRelativePath(index)))
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("snapshot %d stat = %#v, error = %v", index, info, err)
		}
	}
	if info, err := os.Stat(filepath.Join(packageDir, recpackage.AnnotationSnapshotFile)); err != nil || info.Size() == 0 {
		t.Fatalf("final snapshot stat = %#v, error = %v", info, err)
	}
	eventsData, err := os.ReadFile(filepath.Join(packageDir, recpackage.AnnotationEventsFile))
	if err != nil {
		t.Fatalf("ReadFile(events) error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(eventsData)), "\n")
	if len(lines) != 4 {
		t.Fatalf("events = %#v, want four scene-snapshot events", lines)
	}
	if !strings.Contains(lines[0], `"recordingOffsetMs":0`) || !strings.Contains(lines[3], `"recordingOffsetMs":3000`) {
		t.Fatalf("event offsets = %#v, want 0ms through 3000ms", lines)
	}
	if !strings.Contains(lines[3], `"snapshotPath":"annotations/snapshots/annotation-000004.png"`) {
		t.Fatalf("last event = %q, want fourth snapshot path", lines[3])
	}
}

func TestWriteAnnotationAssetsCreatesRenderedElementPNGTimeline(t *testing.T) {
	packageDir := t.TempDir()
	timeline, err := writeAnnotationAssets(packageDir, 64, 48, 4*time.Second, 4, timelineModeElementPNGs)
	if err != nil {
		t.Fatalf("writeAnnotationAssets(element-pngs) error = %v", err)
	}
	if len(timeline.Samples) != 4 {
		t.Fatalf("timeline samples = %#v, want four", timeline.Samples)
	}
	for index := 1; index <= 4; index++ {
		path := filepath.Join(packageDir, filepath.FromSlash(renderedTimelineRelativePath(index)))
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("rendered png %d stat = %#v, error = %v", index, info, err)
		}
	}
	eventsData, err := os.ReadFile(filepath.Join(packageDir, recpackage.AnnotationEventsFile))
	if err != nil {
		t.Fatalf("ReadFile(events) error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(eventsData)), "\n")
	if len(lines) != 4 {
		t.Fatalf("events = %#v, want four element events", lines)
	}
	if !strings.Contains(lines[0], `"type":"element-created"`) || !strings.Contains(lines[3], `"type":"element-updated"`) {
		t.Fatalf("event types = %#v, want created then updated events", lines)
	}
	if strings.Contains(string(eventsData), `"snapshotPath"`) {
		t.Fatalf("element-png events unexpectedly contain snapshotPath: %s", eventsData)
	}
}

func TestAnnotationSmokeSourceDefaultsAndCopiesGeometryToTarget(t *testing.T) {
	source, target, err := annotationSmokeSource(sourceOptions{}, 320, 180)
	if err != nil {
		t.Fatalf("annotationSmokeSource(default) error = %v", err)
	}
	if source.Type != "screen" || source.ID != "screen:annotation-export-smoke" || target.Type != source.Type || target.ID != source.ID {
		t.Fatalf("source/target = %#v / %#v, want default screen source mirrored into target", source, target)
	}
	if source.Geometry == nil || target.Geometry == nil || target.Geometry == source.Geometry {
		t.Fatalf("source/target geometry = %#v / %#v, want copied geometry pointers", source.Geometry, target.Geometry)
	}
	if target.Geometry.Width != 320 || target.Geometry.Height != 180 || target.Geometry.DisplayIndex != 1 {
		t.Fatalf("target geometry = %#v, want default 320x180 display 1", target.Geometry)
	}
}

func TestAnnotationSmokeSourceSupportsRegionAndWindowGeometry(t *testing.T) {
	source, target, err := annotationSmokeSource(sourceOptions{
		Type:         "region",
		ID:           "region:custom",
		Name:         "Custom Region",
		X:            -1920,
		Y:            48,
		DisplayIndex: 2,
		NativeID:     "display-2",
	}, 1440, 900)
	if err != nil {
		t.Fatalf("annotationSmokeSource(region) error = %v", err)
	}
	if source.Type != "region" || source.ID != "region:custom" || source.Name != "Custom Region" {
		t.Fatalf("source = %#v, want custom region source", source)
	}
	if target.Geometry == nil || target.Geometry.X != -1920 || target.Geometry.Y != 48 || target.Geometry.Width != 1440 || target.Geometry.Height != 900 || target.Geometry.NativeID != "display-2" {
		t.Fatalf("target geometry = %#v, want copied region geometry", target.Geometry)
	}

	windowSource, windowTarget, err := annotationSmokeSource(sourceOptions{Type: "window"}, 800, 600)
	if err != nil {
		t.Fatalf("annotationSmokeSource(window) error = %v", err)
	}
	if windowSource.ID != "window:annotation-export-smoke" || windowTarget.ID != windowSource.ID {
		t.Fatalf("window source/target = %#v / %#v, want default window id", windowSource, windowTarget)
	}
}

func TestAnnotationSmokeSourceRejectsUnsupportedType(t *testing.T) {
	if _, _, err := annotationSmokeSource(sourceOptions{Type: "application"}, 320, 180); err == nil {
		t.Fatal("annotationSmokeSource(application) succeeded, want unsupported source error")
	}
}
