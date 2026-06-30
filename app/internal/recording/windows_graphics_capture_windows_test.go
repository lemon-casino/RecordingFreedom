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

func TestWindowsGraphicsCaptureBackendIsRegistered(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "windows", "native")
	runtimeBackend, ok := backend.(*NativeRuntimeBackend)
	if !ok {
		t.Fatalf("SelectBackend(native windows) = %T, want *NativeRuntimeBackend", backend)
	}
	if runtimeBackend.ID() != BackendWindowsGraphicsCapture {
		t.Fatalf("backend id = %q, want %q", runtimeBackend.ID(), BackendWindowsGraphicsCapture)
	}
}

func TestWindowsGraphicsCaptureRuntimeFailsWithoutFakeMediaUntilWriterLands(t *testing.T) {
	root := t.TempDir()
	service := NewServiceWithBackend(appdata.NewService(root), SelectBackend(recpackage.NewService(), "windows", "native"))

	_, err := service.StartRecording(StartRequest{
		SourceID:   "screen:--display1",
		SourceType: SourceScreen,
	})
	if err == nil || !strings.Contains(err.Error(), "writer is not implemented") {
		t.Fatalf("StartRecording() error = %v, want WGC writer not implemented", err)
	}
	if service.State() != StateFailed {
		t.Fatalf("State() = %q, want %q", service.State(), StateFailed)
	}

	matches, err := filepath.Glob(filepath.Join(root, "data", "video", "*.rfrec"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("packages = %#v, want one failed WGC package", matches)
	}
	manifest, err := recpackage.NewService().ReadManifest(filepath.Join(matches[0], recpackage.ManifestFile))
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != recpackage.StatusFailed {
		t.Fatalf("manifest status = %q, want %q", manifest.Status, recpackage.StatusFailed)
	}
	if exists(filepath.Join(matches[0], recpackage.ScreenVideoFile)) {
		t.Fatal("WGC writer placeholder created fake screen.mp4")
	}
	if !exists(filepath.Join(matches[0], recpackage.VideoDiagnosticsFile)) {
		t.Fatal("failed WGC package did not keep video-diagnostics.json")
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
