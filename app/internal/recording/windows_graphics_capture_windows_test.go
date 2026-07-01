//go:build windows

package recording

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestWindowsDesktopCaptureBackendIsDefaultOnWindows(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "windows", "native")
	runtimeBackend, ok := backend.(*NativeRuntimeBackend)
	if !ok {
		t.Fatalf("SelectBackend(native windows) = %T, want *NativeRuntimeBackend", backend)
	}
	if runtimeBackend.ID() != BackendFFmpegDesktopCapture {
		t.Fatalf("backend id = %q, want %q", runtimeBackend.ID(), BackendFFmpegDesktopCapture)
	}
}

func TestWindowsGraphicsCaptureBackendAliasIsRegistered(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "windows", "wgc")
	runtimeBackend, ok := backend.(*NativeRuntimeBackend)
	if !ok {
		t.Fatalf("SelectBackend(wgc windows) = %T, want *NativeRuntimeBackend", backend)
	}
	if runtimeBackend.ID() != BackendWindowsGraphicsCapture {
		t.Fatalf("backend id = %q, want %q", runtimeBackend.ID(), BackendWindowsGraphicsCapture)
	}
}

func TestWindowsDesktopCaptureRuntimeFailsWithoutFakeMediaWhenFFmpegMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("RECORDINGFREEDOM_FFMPEG_PATH", filepath.Join(root, "missing-ffmpeg.exe"))
	service := NewServiceWithBackend(appdata.NewService(root), SelectBackend(recpackage.NewService(), "windows", "native"))

	_, err := service.StartRecording(StartRequest{
		SourceID:   "screen:--display1",
		SourceType: SourceScreen,
		SourceGeometry: &SourceGeometry{
			Width:  1920,
			Height: 1080,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "FFmpeg") {
		t.Fatalf("StartRecording() error = %v, want FFmpeg dependency error", err)
	}
	if service.State() != StateFailed {
		t.Fatalf("State() = %q, want %q", service.State(), StateFailed)
	}

	matches, err := filepath.Glob(filepath.Join(root, "data", "video", "*.rfrec"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("packages = %#v, want one failed Windows capture package", matches)
	}
	manifest, err := recpackage.NewService().ReadManifest(filepath.Join(matches[0], recpackage.ManifestFile))
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != recpackage.StatusFailed {
		t.Fatalf("manifest status = %q, want %q", manifest.Status, recpackage.StatusFailed)
	}
	if exists(filepath.Join(matches[0], recpackage.ScreenVideoFile)) {
		t.Fatal("Windows capture dependency failure created fake screen.mp4")
	}
	if !exists(filepath.Join(matches[0], recpackage.VideoDiagnosticsFile)) {
		t.Fatal("failed Windows capture package did not keep video-diagnostics.json")
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
