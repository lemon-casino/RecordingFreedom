package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestParseSourceType(t *testing.T) {
	tests := []struct {
		value string
		want  devices.CaptureSourceType
		ok    bool
	}{
		{value: "screen", want: devices.SourceScreen, ok: true},
		{value: "all-screens", want: devices.SourceAllScreens, ok: true},
		{value: "region", want: devices.SourceRegion, ok: true},
		{value: " window ", want: devices.SourceWindow, ok: true},
		{value: "application", want: devices.SourceApplication, ok: true},
		{value: "camera"},
	}
	for _, test := range tests {
		got, err := parseSourceType(test.value)
		if test.ok && (err != nil || got != test.want) {
			t.Fatalf("parseSourceType(%q) = %q, %v; want %q", test.value, got, err, test.want)
		}
		if !test.ok && err == nil {
			t.Fatalf("parseSourceType(%q) error = nil, want error", test.value)
		}
	}
}

func TestRegionGeometryForSmokeDerivesCenteredScreenRegion(t *testing.T) {
	got, err := regionGeometryForSmoke([]devices.CaptureSource{
		{ID: "screen:display-1", Type: devices.SourceScreen, X: 0, Y: 0, Width: 2560, Height: 1440, NativeID: "cgdisplay:42", DisplayIndex: 1, Available: true},
	}, options{})
	if err != nil {
		t.Fatalf("regionGeometryForSmoke() error = %v", err)
	}
	if got.X != 64 || got.Y != 64 || got.Width != 640 || got.Height != 360 || got.NativeID != "cgdisplay:42" {
		t.Fatalf("region geometry = %#v, want conservative 640x360 region on display", got)
	}
}

func TestChooseSourcePrefersAvailableMatchingType(t *testing.T) {
	sources := []devices.CaptureSource{
		{ID: "screen:queued", Type: devices.SourceScreen, Available: false},
		{ID: "window:1", Type: devices.SourceWindow, Available: true},
		{ID: "screen:display-1", Type: devices.SourceScreen, Available: true},
	}
	got, err := chooseSource(sources, "", devices.SourceScreen)
	if err != nil {
		t.Fatalf("chooseSource() error = %v", err)
	}
	if got.ID != "screen:display-1" {
		t.Fatalf("chooseSource() = %q, want screen:display-1", got.ID)
	}
}

func TestChooseSourceHonorsExplicitID(t *testing.T) {
	sources := []devices.CaptureSource{
		{ID: "screen:display-1", Type: devices.SourceScreen, Available: true},
		{ID: "screen:display-2", Type: devices.SourceScreen, Available: true},
	}
	got, err := chooseSource(sources, "screen:display-2", devices.SourceScreen)
	if err != nil {
		t.Fatalf("chooseSource() error = %v", err)
	}
	if got.ID != "screen:display-2" {
		t.Fatalf("chooseSource() = %q, want screen:display-2", got.ID)
	}
}

func TestVerifyManifestAudioTrackAcceptsSidecar(t *testing.T) {
	packageDir := t.TempDir()
	audioPath := recpackage.SystemAudioFile
	if err := os.WriteFile(filepath.Join(packageDir, audioPath), make([]byte, 46), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	path, bytes, err := verifyManifestAudioTrack(packageDir, "systemAudio", audioPath, recpackage.AudioStorageSidecar, recpackage.ScreenVideoFile, recpackage.ManifestTrackDiagnostics{
		Enabled: true,
		Path:    audioPath,
	})
	if err != nil {
		t.Fatalf("verifyManifestAudioTrack() error = %v", err)
	}
	if path != filepath.Join(packageDir, audioPath) || bytes != 46 {
		t.Fatalf("sidecar verification = %q/%d, want package file and size", path, bytes)
	}
}
