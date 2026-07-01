package video

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestNewDiagnosticsNormalizesCaptureConfig(t *testing.T) {
	diagnostics := NewDiagnostics(CaptureConfig{
		Backend:    " screencapturekit ",
		SourceID:   " screen:display-1 ",
		SourceType: devices.SourceScreen,
		SourceName: " Primary Display ",
		SourceGeometry: &SourceGeometry{
			X:            -1440,
			Y:            0,
			Width:        2560,
			Height:       1440,
			DisplayIndex: 2,
			NativeID:     " display:1 ",
		},
		OutputPath:  " screen.mp4 ",
		SystemAudio: true,
		Profile: recordingprofile.Profile{
			Quality:          recordingprofile.QualityHigh,
			FPS:              60,
			CaptureCursor:    true,
			CountdownSeconds: 3,
		},
	})

	if diagnostics.Backend != "screencapturekit" {
		t.Fatalf("backend = %q, want screencapturekit", diagnostics.Backend)
	}
	if diagnostics.Source.ID != "screen:display-1" || diagnostics.Source.Name != "Primary Display" || diagnostics.Source.Type != devices.SourceScreen {
		t.Fatalf("source = %#v", diagnostics.Source)
	}
	if diagnostics.Source.Geometry == nil ||
		diagnostics.Source.Geometry.X != -1440 ||
		diagnostics.Source.Geometry.Width != 2560 ||
		diagnostics.Source.Geometry.NativeID != "display:1" {
		t.Fatalf("source geometry = %#v, want normalized capture geometry", diagnostics.Source.Geometry)
	}
	if diagnostics.OutputPath != "screen.mp4" {
		t.Fatalf("output path = %q, want screen.mp4", diagnostics.OutputPath)
	}
	if diagnostics.Recording.Quality != recordingprofile.QualityHigh || diagnostics.Recording.FPS != 60 || !diagnostics.Recording.CaptureCursor || diagnostics.Recording.CountdownSeconds != 3 {
		t.Fatalf("recording = %#v", diagnostics.Recording)
	}
	if !diagnostics.Screen.Enabled || diagnostics.Screen.FrameRate != 60 {
		t.Fatalf("screen diagnostics = %#v", diagnostics.Screen)
	}
	if !diagnostics.SystemAudio.Enabled {
		t.Fatalf("system audio diagnostics = %#v, want enabled", diagnostics.SystemAudio)
	}
}

func TestWriteDiagnosticsWritesJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video-diagnostics.json")
	diagnostics := NewDiagnostics(CaptureConfig{
		Backend:    "windows-graphics-capture",
		SourceID:   "screen:primary",
		SourceType: devices.SourceScreen,
		OutputPath: "screen.mp4",
	})
	diagnostics.Screen.FramesWritten = 12

	if err := WriteDiagnostics(path, diagnostics); err != nil {
		t.Fatalf("WriteDiagnostics() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var decoded Diagnostics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("diagnostics JSON invalid: %v", err)
	}
	if decoded.SchemaVersion != 1 || decoded.Screen.FramesWritten != 12 {
		t.Fatalf("decoded diagnostics = %#v", decoded)
	}
}
