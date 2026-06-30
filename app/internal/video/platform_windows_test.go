//go:build windows

package video

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

func TestWindowsGraphicsCapturePlatformSessionConstructsForScreenSource(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "screen:--display1",
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

func TestWindowsGraphicsCapturePlatformSessionConstructsForWindowSource(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "window:1f04aa",
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

func TestWindowsGraphicsCapturePlatformSessionConstructsForApplicationSource(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
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

func TestWindowsGraphicsCaptureStartFailsUntilWriterLands(t *testing.T) {
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "screen:--display1",
		SourceType:      devices.SourceScreen,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	err = session.Start(nil)
	if err == nil || !strings.Contains(err.Error(), "writer is not implemented") {
		t.Fatalf("Start() error = %v, want writer not implemented", err)
	}
}
