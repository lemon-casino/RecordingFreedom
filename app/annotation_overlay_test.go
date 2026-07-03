package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestAnnotationOverlayStateUsesActiveVideoRecordingGeometry(t *testing.T) {
	geometry := recording.SourceGeometry{
		X:            -1920,
		Y:            48,
		Width:        1440,
		Height:       900,
		DisplayIndex: 2,
		NativeID:     "display-2",
	}
	for _, tc := range []struct {
		name       string
		sourceID   string
		sourceType recording.CaptureSourceType
	}{
		{name: "all screens", sourceID: "all-screens:virtual-desktop", sourceType: recording.SourceAllScreens},
		{name: "single screen", sourceID: "screen:2", sourceType: recording.SourceScreen},
		{name: "region", sourceID: "region:custom", sourceType: recording.SourceRegion},
		{name: "window", sourceID: "window:focused", sourceType: recording.SourceWindow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := newAnnotationOverlayTestService(t)
			session, err := service.recorder.StartRecording(recording.StartRequest{
				SourceID:       tc.sourceID,
				SourceType:     tc.sourceType,
				SourceName:     "Presentation Target",
				SourceGeometry: &geometry,
			})
			if err != nil {
				t.Fatalf("StartRecording() error = %v", err)
			}

			state, err := service.annotationOverlayState()
			if err != nil {
				t.Fatalf("annotationOverlayState() error = %v", err)
			}
			if state.PackageDir != session.PackageDir || state.ManifestPath != session.Manifest {
				t.Fatalf("state package = %q/%q, want %q/%q", state.PackageDir, state.ManifestPath, session.PackageDir, session.Manifest)
			}
			if state.WindowBounds.X != geometry.X || state.WindowBounds.Y != geometry.Y || state.WindowBounds.Width != geometry.Width || state.WindowBounds.Height != geometry.Height {
				t.Fatalf("window bounds = %#v, want recording geometry %#v", state.WindowBounds, geometry)
			}
			if state.CanvasBounds.X != 0 || state.CanvasBounds.Y != 0 || state.CanvasBounds.Width != geometry.Width || state.CanvasBounds.Height != geometry.Height {
				t.Fatalf("canvas bounds = %#v, want local canvas %dx%d", state.CanvasBounds, geometry.Width, geometry.Height)
			}
			if state.Target.Type != string(tc.sourceType) || state.Target.ID != tc.sourceID || state.Target.Geometry == nil {
				t.Fatalf("target = %#v, want source type/id with geometry", state.Target)
			}
			if got := *state.Target.Geometry; got.X != geometry.X || got.Y != geometry.Y || got.Width != geometry.Width || got.Height != geometry.Height || got.DisplayIndex != geometry.DisplayIndex || got.NativeID != geometry.NativeID {
				t.Fatalf("target geometry = %#v, want %#v", got, geometry)
			}
		})
	}
}

func TestAnnotationOverlayStateAllowsPausedVideoRecording(t *testing.T) {
	service := newAnnotationOverlayTestService(t)
	if _, err := service.recorder.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		SourceGeometry: &recording.SourceGeometry{
			Width:  1280,
			Height: 720,
		},
	}); err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	if _, err := service.recorder.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	state, err := service.annotationOverlayState()
	if err != nil {
		t.Fatalf("annotationOverlayState() error = %v", err)
	}
	if state.WindowBounds.Width != 1280 || state.WindowBounds.Height != 720 {
		t.Fatalf("window bounds = %#v, want paused video geometry", state.WindowBounds)
	}
}

func TestAnnotationOverlayStateRejectsNonVideoRecordingModes(t *testing.T) {
	t.Run("no active recording", func(t *testing.T) {
		service := newAnnotationOverlayTestService(t)
		if _, err := service.annotationOverlayState(); err == nil || !strings.Contains(err.Error(), "active screen recording") {
			t.Fatalf("annotationOverlayState() error = %v, want active screen recording error", err)
		}
	})

	t.Run("audio only recording", func(t *testing.T) {
		service := newAnnotationOverlayTestService(t)
		if _, err := service.recorder.StartAudioOnlyRecording(recording.AudioOnlyRequest{
			Audio: recording.AudioRequest{Microphone: true},
		}); err != nil {
			t.Fatalf("StartAudioOnlyRecording() error = %v", err)
		}
		defer func() {
			_, _ = service.recorder.Stop()
		}()
		if _, err := service.annotationOverlayState(); err == nil || !strings.Contains(err.Error(), "active screen recording") {
			t.Fatalf("annotationOverlayState() error = %v, want active screen recording error", err)
		}
	})
}

func TestAnnotationWindowBoundsFallsBackWhenManifestHasNoGeometry(t *testing.T) {
	bounds := (&RecordingFreedomService{}).annotationWindowBounds(recpackage.Manifest{})
	if bounds.X != 0 || bounds.Y != 0 || bounds.Width != 1280 || bounds.Height != 720 {
		t.Fatalf("fallback bounds = %#v, want 1280x720 at origin", bounds)
	}
}

func TestNormalizeAnnotationEventsAddsSequenceAndTimelineDefaults(t *testing.T) {
	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	fallback := `{"type":"scene-snapshot","sessionId":"session-1","recordingOffsetMs":42,"wallOffsetMs":50,"scenePath":"annotations/scene.excalidraw","snapshotPath":"annotations/exports/annotation.png"}`
	events := strings.Join([]string{
		`{"type":"element-created","elementId":"a","elementType":"freedraw"}`,
		`{"type":"element-updated","eventId":"client-2","elementId":"a","elementVersion":2}`,
	}, "\n")

	normalized, err := normalizeAnnotationEvents(eventsPath, events, fallback)
	if err != nil {
		t.Fatalf("normalizeAnnotationEvents() error = %v", err)
	}
	lines := nonEmptyJSONLLines(normalized)
	if len(lines) != 3 {
		t.Fatalf("lines = %#v, want snapshot plus two normalized events", lines)
	}
	first := mustJSONLine(t, lines[0])
	second := mustJSONLine(t, lines[1])
	third := mustJSONLine(t, lines[2])
	if first["sequence"] != float64(1) || second["sequence"] != float64(2) || third["sequence"] != float64(3) {
		t.Fatalf("sequences = %#v / %#v / %#v, want 1 / 2 / 3", first["sequence"], second["sequence"], third["sequence"])
	}
	if first["type"] != "scene-snapshot" || second["type"] != "element-created" || third["type"] != "element-updated" {
		t.Fatalf("types = %#v / %#v / %#v, want snapshot then element events", first["type"], second["type"], third["type"])
	}
	if first["eventId"] == "" || second["eventId"] == "" || third["eventId"] != "client-2" {
		t.Fatalf("event ids = %#v / %#v / %#v, want generated first/second and preserved third", first["eventId"], second["eventId"], third["eventId"])
	}
	if second["recordingOffsetMs"] != float64(42) || third["scenePath"] != "annotations/scene.excalidraw" {
		t.Fatalf("timeline defaults = %#v / %#v", second, third)
	}
}

func TestNormalizeAnnotationEventsContinuesExistingSequence(t *testing.T) {
	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(eventsPath, []byte(`{"sequence":1}`+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(events) error = %v", err)
	}
	fallback := `{"type":"scene-snapshot","recordingOffsetMs":1}`

	normalized, err := normalizeAnnotationEvents(eventsPath, `{"type":"element-created","elementId":"b"}`, fallback)
	if err != nil {
		t.Fatalf("normalizeAnnotationEvents() error = %v", err)
	}
	lines := nonEmptyJSONLLines(normalized)
	if len(lines) != 2 {
		t.Fatalf("lines = %#v, want snapshot plus client event", lines)
	}
	event := mustJSONLine(t, lines[0])
	if event["sequence"] != float64(2) || event["type"] != "scene-snapshot" {
		t.Fatalf("event = %#v, want next sequence snapshot first", event)
	}
}

func TestNormalizeAnnotationEventsRejectsInvalidClientJSON(t *testing.T) {
	fallback := `{"type":"scene-snapshot","recordingOffsetMs":1}`
	if _, err := normalizeAnnotationEvents(filepath.Join(t.TempDir(), "events.jsonl"), "{invalid", fallback); err == nil {
		t.Fatal("normalizeAnnotationEvents() accepted invalid JSON")
	}
}

func mustJSONLine(t *testing.T, line string) map[string]any {
	t.Helper()
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("json line %q error = %v", line, err)
	}
	return event
}

func newAnnotationOverlayTestService(t *testing.T) *RecordingFreedomService {
	t.Helper()
	data := appdata.NewService(t.TempDir())
	return &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(recpackage.NewService())),
	}
}
