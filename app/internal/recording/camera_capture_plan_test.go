package recording

import (
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestCreateCameraCaptureConfigAddsManagedPreviewImagePath(t *testing.T) {
	packageDir := filepath.Join(t.TempDir(), "recording.rfrec")
	plan := recpackage.RecordingWritePlan{
		Package:         recpackage.Package{Dir: packageDir},
		WebcamVideoPath: filepath.Join(packageDir, recpackage.WindowsWebcamVideoFile),
		CacheDir:        filepath.Join(packageDir, recpackage.CacheDir),
	}

	config, err := CreateCameraCaptureConfig(BackendFFmpegDesktopCapture, StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Camera: CameraRequest{
			Enabled:        true,
			DeviceID:       "camera:dshow:integrated-camera",
			DeviceNativeID: "Integrated Camera",
			PIPPreset:      "bottom-right",
		},
	}, plan)
	if err != nil {
		t.Fatalf("CreateCameraCaptureConfig() error = %v", err)
	}
	wantPreview := filepath.Join(packageDir, recpackage.CacheDir, cameraPreviewImageFile)
	if config.PreviewImagePath != wantPreview {
		t.Fatalf("preview image path = %q, want %q", config.PreviewImagePath, wantPreview)
	}
	if CameraPreviewImagePath(packageDir) != wantPreview {
		t.Fatalf("CameraPreviewImagePath() = %q, want %q", CameraPreviewImagePath(packageDir), wantPreview)
	}
}
