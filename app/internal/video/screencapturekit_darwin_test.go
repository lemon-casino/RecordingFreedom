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
