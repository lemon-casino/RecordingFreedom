//go:build darwin && cgo

package video

import (
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

func TestScreenCaptureKitPlatformSessionConstructsForDisplaySource(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "screencapturekit",
		SourceID:        "screen:display-1",
		SourceType:      devices.SourceScreen,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}

func TestScreenCaptureKitPlatformSessionConstructsForWindowSource(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "screencapturekit",
		SourceID:        "window:42",
		SourceType:      devices.SourceWindow,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}

func TestScreenCaptureKitPlatformSessionConstructsForApplicationSource(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "screencapturekit",
		SourceID:        "application:420",
		SourceType:      devices.SourceApplication,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}

func TestScreenCaptureKitPlatformSessionConstructsForRegionSource(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "screencapturekit",
		SourceID:        "region:custom",
		SourceType:      devices.SourceRegion,
		SourceGeometry:  &SourceGeometry{X: 120, Y: 80, Width: 1280, Height: 720, NativeID: "cgdisplay:42"},
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}

func TestScreenCaptureKitPlatformSessionRejectsRegionWithoutDisplayTarget(t *testing.T) {
	if _, err := NewPlatformSession(CaptureConfig{
		Backend:         "screencapturekit",
		SourceID:        "region:custom",
		SourceType:      devices.SourceRegion,
		SourceGeometry:  &SourceGeometry{X: 120, Y: 80, Width: 1280, Height: 720, NativeID: "region:virtual-desktop"},
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	}); err == nil {
		t.Fatal("NewPlatformSession() accepted region without display target")
	}
}

func TestScreenCaptureKitPlatformSessionConstructsWithSystemAudio(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "screencapturekit",
		SourceID:        "screen:display-1",
		SourceType:      devices.SourceScreen,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
		SystemAudio:     true,
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}
