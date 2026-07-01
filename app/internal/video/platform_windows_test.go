//go:build windows

package video

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

func TestWindowsGraphicsCapturePlatformSessionConstructsForScreenSource(t *testing.T) {
	withFakeFFmpeg(t)
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "screen:--display1",
		SourceType:      devices.SourceScreen,
		SourceGeometry:  &SourceGeometry{Width: 1920, Height: 1080},
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
	withFakeFFmpeg(t)
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
	withFakeFFmpeg(t)
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

func TestWindowsGraphicsCapturePlatformSessionRequiresFFmpeg(t *testing.T) {
	t.Setenv(EnvFFmpegPath, filepath.Join(t.TempDir(), "missing-ffmpeg.exe"))
	_, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "screen:--display1",
		SourceType:      devices.SourceScreen,
		SourceGeometry:  &SourceGeometry{Width: 1920, Height: 1080},
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "FFmpeg") {
		t.Fatalf("NewPlatformSession() error = %v, want FFmpeg dependency error", err)
	}
}

func TestWindowsGraphicsCaptureStartFailsForProgramCapture(t *testing.T) {
	withFakeFFmpeg(t)
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
	err = session.Start(nil)
	if err == nil || !strings.Contains(err.Error(), "program capture is queued") {
		t.Fatalf("Start() error = %v, want program capture queued", err)
	}
}

func withFakeFFmpeg(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg.exe")
	if err := os.WriteFile(path, []byte("fake ffmpeg"), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
	t.Setenv(EnvFFmpegPath, path)
}
