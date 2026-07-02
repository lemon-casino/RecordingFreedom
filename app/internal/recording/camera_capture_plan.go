package recording

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

const cameraPreviewImageFile = "pip-camera-preview.jpg"

func CreateCameraCaptureConfig(backendID string, req StartRequest, plan recpackage.RecordingWritePlan) (video.CameraCaptureConfig, error) {
	normalized, err := NormalizeStartRequest(req)
	if err != nil {
		return video.CameraCaptureConfig{}, err
	}
	if !normalized.Camera.Enabled {
		return video.CameraCaptureConfig{}, nil
	}
	if strings.TrimSpace(plan.WebcamVideoPath) == "" {
		return video.CameraCaptureConfig{}, errors.New("camera sidecar output path is required")
	}
	if strings.TrimSpace(normalized.Camera.DeviceNativeID) == "" {
		return video.CameraCaptureConfig{}, errors.New("camera deviceNativeId is required for native sidecar capture")
	}
	previewImagePath := ""
	if strings.TrimSpace(plan.CacheDir) != "" {
		previewImagePath = filepath.Join(plan.CacheDir, cameraPreviewImageFile)
	} else {
		previewImagePath = CameraPreviewImagePath(plan.Package.Dir)
	}
	return video.NormalizeCameraCaptureConfig(video.CameraCaptureConfig{
		Backend:          backendID,
		DeviceID:         normalized.Camera.DeviceID,
		DeviceNativeID:   normalized.Camera.DeviceNativeID,
		OutputPath:       plan.WebcamVideoPath,
		PreviewImagePath: previewImagePath,
		Profile:          normalized.Recording,
	}), nil
}

func CameraPreviewImagePath(packageDir string) string {
	packageDir = strings.TrimSpace(packageDir)
	if packageDir == "" {
		return ""
	}
	return filepath.Join(packageDir, recpackage.CacheDir, cameraPreviewImageFile)
}
