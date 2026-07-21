package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/wailsapp/wails/v3/pkg/application"
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
	annotationBounds := applicationRect(-1820, 148, 640, 360)
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
			service.setAnnotationRegionDIP(session.ID, annotationBounds)
			service.setAnnotationSourceImage(session.ID, "data:image/png;base64,frozen", "2026-07-21T18:00:00Z")

			state, err := service.annotationOverlayState()
			if err != nil {
				t.Fatalf("annotationOverlayState() error = %v", err)
			}
			if state.PackageDir != session.PackageDir || state.ManifestPath != session.Manifest {
				t.Fatalf("state package = %q/%q, want %q/%q", state.PackageDir, state.ManifestPath, session.PackageDir, session.Manifest)
			}
			wantWindow := annotationOverlayWindowBounds(annotationBounds)
			if state.WindowBounds.X != wantWindow.X || state.WindowBounds.Y != wantWindow.Y || state.WindowBounds.Width != wantWindow.Width || state.WindowBounds.Height != wantWindow.Height {
				t.Fatalf("window bounds = %#v, want framed annotation geometry %#v", state.WindowBounds, wantWindow)
			}
			if state.CanvasBounds.X != annotationOverlayFrameInset || state.CanvasBounds.Y != annotationOverlayFrameInset || state.CanvasBounds.Width != annotationBounds.Width || state.CanvasBounds.Height != annotationBounds.Height {
				t.Fatalf("canvas bounds = %#v, want local canvas %dx%d", state.CanvasBounds, annotationBounds.Width, annotationBounds.Height)
			}
			if state.Target.Type != annotationRegionTargetType || state.Target.ID != annotationRegionTargetID || state.Target.Geometry == nil {
				t.Fatalf("target = %#v, want selected annotation target", state.Target)
			}
			if got := *state.Target.Geometry; got.X != annotationBounds.X || got.Y != annotationBounds.Y || got.Width != annotationBounds.Width || got.Height != annotationBounds.Height {
				t.Fatalf("target geometry = %#v, want %#v", got, annotationBounds)
			}
			if state.SourceImageDataURL != "data:image/png;base64,frozen" || state.SourceImageCapturedAt != "2026-07-21T18:00:00Z" {
				t.Fatalf("source image = %q captured at %q, want frozen source", state.SourceImageDataURL, state.SourceImageCapturedAt)
			}
		})
	}
}

func TestAnnotationOverlayLayoutKeepsCompactToolbarOutsideSmallCanvas(t *testing.T) {
	screens := []*application.Screen{{
		Bounds:   applicationRect(0, 0, 1280, 720),
		WorkArea: applicationRect(0, 0, 1280, 680),
	}}
	layout := annotationOverlayLayoutForScreens(applicationRect(600, 320, 120, 80), screens)

	if layout.ToolbarPlacement != "top" {
		t.Fatalf("toolbar placement = %q, want top", layout.ToolbarPlacement)
	}
	if layout.ToolbarBounds.Width < annotationOverlayToolbarMinWidth {
		t.Fatalf("toolbar width = %d, want at least %d", layout.ToolbarBounds.Width, annotationOverlayToolbarMinWidth)
	}
	if layout.ToolbarBounds.Y+layout.ToolbarBounds.Height+annotationOverlayToolbarGap > layout.CanvasBounds.Y {
		t.Fatalf("toolbar and canvas overlap: toolbar=%#v canvas=%#v", layout.ToolbarBounds, layout.CanvasBounds)
	}
	if layout.WindowBounds.Width < layout.ToolbarBounds.Width+annotationOverlayFrameInset*2 || layout.WindowBounds.Height < layout.CanvasBounds.Y+layout.CanvasBounds.Height+annotationOverlayFrameInset {
		t.Fatalf("window does not contain toolbar and canvas: window=%#v toolbar=%#v canvas=%#v", layout.WindowBounds, layout.ToolbarBounds, layout.CanvasBounds)
	}
}

func TestAnnotationOverlayLayoutPlacesToolbarBelowCanvasNearTopEdge(t *testing.T) {
	screens := []*application.Screen{{
		Bounds:   applicationRect(0, 0, 1280, 720),
		WorkArea: applicationRect(0, 0, 1280, 680),
	}}
	layout := annotationOverlayLayoutForScreens(applicationRect(600, 8, 120, 80), screens)

	if layout.ToolbarPlacement != "bottom" {
		t.Fatalf("toolbar placement = %q, want bottom", layout.ToolbarPlacement)
	}
	if layout.CanvasBounds.Y+layout.CanvasBounds.Height+annotationOverlayFrameInset+annotationOverlayToolbarGap > layout.ToolbarBounds.Y {
		t.Fatalf("toolbar and canvas overlap: canvas=%#v toolbar=%#v", layout.CanvasBounds, layout.ToolbarBounds)
	}
	if layout.WindowBounds.Y != 0 {
		t.Fatalf("window y = %d, want 0 for top-edge canvas", layout.WindowBounds.Y)
	}
}

func TestAnnotationOverlayLayoutPreservesWideCanvasAcrossDisplayBounds(t *testing.T) {
	screens := []*application.Screen{{
		Bounds:   applicationRect(0, 0, 1280, 720),
		WorkArea: applicationRect(0, 0, 1280, 680),
	}}
	canvas := applicationRect(-400, 80, 1800, 500)
	layout := annotationOverlayLayoutForScreens(canvas, screens)

	if layout.CanvasBounds.Width != canvas.Width || layout.CanvasBounds.Height != canvas.Height {
		t.Fatalf("canvas size = %dx%d, want original %dx%d", layout.CanvasBounds.Width, layout.CanvasBounds.Height, canvas.Width, canvas.Height)
	}
	if layout.WindowBounds.Width < canvas.Width+annotationOverlayFrameInset*2 {
		t.Fatalf("window width = %d, want room for wide canvas %d", layout.WindowBounds.Width, canvas.Width+annotationOverlayFrameInset*2)
	}
}

func TestAnnotationOverlayStateRequiresSelectedAnnotationRegion(t *testing.T) {
	service := newAnnotationOverlayTestService(t)
	if _, err := service.recorder.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		SourceGeometry: &recording.SourceGeometry{
			X:      0,
			Y:      0,
			Width:  1280,
			Height: 720,
		},
	}); err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}

	if _, err := service.annotationOverlayState(); err == nil || !strings.Contains(err.Error(), "selected annotation region") {
		t.Fatalf("annotationOverlayState() error = %v, want selected annotation region error", err)
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
	session, ok := service.recorder.ActiveSession()
	if !ok {
		t.Fatal("ActiveSession() = false")
	}
	service.setAnnotationRegionDIP(session.ID, applicationRect(10, 20, 640, 360))

	state, err := service.annotationOverlayState()
	if err != nil {
		t.Fatalf("annotationOverlayState() error = %v", err)
	}
	wantWindow := annotationOverlayWindowBounds(applicationRect(10, 20, 640, 360))
	if state.WindowBounds.X != wantWindow.X || state.WindowBounds.Y != wantWindow.Y || state.WindowBounds.Width != wantWindow.Width || state.WindowBounds.Height != wantWindow.Height {
		t.Fatalf("window bounds = %#v, want paused framed annotation geometry %#v", state.WindowBounds, wantWindow)
	}
	if state.CanvasBounds.X != annotationOverlayFrameInset || state.CanvasBounds.Y != annotationOverlayFrameInset || state.CanvasBounds.Width != 640 || state.CanvasBounds.Height != 360 {
		t.Fatalf("canvas bounds = %#v, want inset paused annotation geometry", state.CanvasBounds)
	}
}

func TestHideAnnotationOverlayRestoresActiveRegionRecordingFrame(t *testing.T) {
	service := newAnnotationOverlayTestService(t)
	recordingBounds := applicationRect(120, 90, 800, 450)
	if _, err := service.recorder.StartRecording(recording.StartRequest{
		SourceID:   "region:custom",
		SourceType: recording.SourceRegion,
		SourceGeometry: &recording.SourceGeometry{
			X:      recordingBounds.X,
			Y:      recordingBounds.Y,
			Width:  recordingBounds.Width,
			Height: recordingBounds.Height,
		},
	}); err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	service.setSelectedRegionDIP(recordingBounds)
	if err := service.showRegionFrame(recordingBounds); err != nil {
		t.Fatalf("showRegionFrame() error = %v", err)
	}

	annotationBounds := applicationRect(180, 130, 320, 240)
	if _, err := service.showRegionEditorWithPurpose(annotationBounds, regionSelectionPurposeAnnotation); err != nil {
		t.Fatalf("showRegionEditorWithPurpose() error = %v", err)
	}
	state, ok := service.currentRegionFrameState()
	if !ok || state.Mode != "edit" || state.Purpose != regionSelectionPurposeAnnotation {
		t.Fatalf("annotation frame state = %#v, visible %v", state, ok)
	}
	service.clearRegionFrameState()

	if err := service.HideAnnotationOverlay(); err != nil {
		t.Fatalf("HideAnnotationOverlay() error = %v", err)
	}
	state, ok = service.currentRegionFrameState()
	if !ok {
		t.Fatal("recording region frame was not restored")
	}
	if state.Mode != "recording" || state.Purpose != regionSelectionPurposeCapture {
		t.Fatalf("restored frame mode/purpose = %#v, want recording capture", state)
	}
	if state.Bounds.X != recordingBounds.X || state.Bounds.Y != recordingBounds.Y || state.Bounds.Width != recordingBounds.Width || state.Bounds.Height != recordingBounds.Height {
		t.Fatalf("restored frame bounds = %#v, want %#v", state.Bounds, recordingBounds)
	}
}

func TestHideAnnotationOverlayDoesNotRestoreFrameForNonRegionRecording(t *testing.T) {
	service := newAnnotationOverlayTestService(t)
	staleRegionBounds := applicationRect(50, 60, 640, 360)
	if _, err := service.recorder.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		SourceGeometry: &recording.SourceGeometry{
			X:      0,
			Y:      0,
			Width:  1280,
			Height: 720,
		},
	}); err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	annotationBounds := applicationRect(120, 90, 320, 240)
	if _, err := service.showRegionEditorWithPurpose(annotationBounds, regionSelectionPurposeAnnotation); err != nil {
		t.Fatalf("showRegionEditorWithPurpose() error = %v", err)
	}
	service.setSelectedRegionDIP(staleRegionBounds)
	service.clearRegionFrameState()

	if err := service.HideAnnotationOverlay(); err != nil {
		t.Fatalf("HideAnnotationOverlay() error = %v", err)
	}
	state, ok := service.currentRegionFrameState()
	if ok {
		t.Fatalf("frame state = %#v, want no restored region frame for screen recording", state)
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
	audioSession := &annotationOverlayAudioSession{}
	return &RecordingFreedomService{
		appData: data,
		recorder: recording.NewServiceWithOptions(data, recording.ServiceOptions{
			Backend: recording.NewMockBackend(recpackage.NewService()),
			AudioOnlyRuntimeOptions: recording.AudioOnlyRuntimeOptions{
				AudioSessionFactory: func(config audio.CaptureConfig, suppressor audio.NoiseSuppressor) (recording.NativeAudioSession, error) {
					if suppressor != nil {
						t.Fatalf("suppressor = %#v, want nil", suppressor)
					}
					audioSession.path = config.MicrophoneAudioPath
					return audioSession, nil
				},
				PostStopProcessor: func(runtime *recording.AudioOnlyRuntime) error {
					return os.WriteFile(runtime.Plan.AudioOnlyPath, annotationOverlayMinimalMP4("soun"), 0o644)
				},
			},
		}),
	}
}

type annotationOverlayAudioSession struct {
	path string
}

func applicationRect(x int, y int, width int, height int) application.Rect {
	return application.Rect{X: x, Y: y, Width: width, Height: height}
}

func (s *annotationOverlayAudioSession) Start(context.Context) error {
	return nil
}

func (s *annotationOverlayAudioSession) Pause() error {
	return nil
}

func (s *annotationOverlayAudioSession) Resume() error {
	return nil
}

func (s *annotationOverlayAudioSession) Stop() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	return os.WriteFile(s.path, make([]byte, 64), 0o644)
}

func (s *annotationOverlayAudioSession) Diagnostics() audio.Diagnostics {
	return audio.Diagnostics{
		Backend: recording.BackendAudioOnlyNative,
		Microphone: audio.StreamDiagnostics{
			Enabled:        true,
			SampleRate:     audio.RNNoiseSampleRate,
			SamplesWritten: 48000,
			EndOffsetMs:    1000,
			DurationMs:     1000,
		},
	}
}

func annotationOverlayMinimalMP4(handlerType string) []byte {
	payload := make([]byte, 0, 12)
	payload = append(payload, 0, 0, 0, 0)
	payload = append(payload, 0, 0, 0, 0)
	payload = append(payload, []byte(handlerType)...)
	data := annotationOverlayMP4Box("ftyp", []byte("isom0000"))
	data = append(data, annotationOverlayMP4Box("moov", annotationOverlayMP4Box("trak", annotationOverlayMP4Box("mdia", annotationOverlayMP4Box("hdlr", payload))))...)
	return data
}

func annotationOverlayMP4Box(kind string, payload []byte) []byte {
	box := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(box)))
	copy(box[4:8], []byte(kind))
	copy(box[8:], payload)
	return box
}
