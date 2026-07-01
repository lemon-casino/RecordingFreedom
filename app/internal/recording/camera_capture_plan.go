package recording

import (
	"errors"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

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
	return video.NormalizeCameraCaptureConfig(video.CameraCaptureConfig{
		Backend:        backendID,
		DeviceID:       normalized.Camera.DeviceID,
		DeviceNativeID: normalized.Camera.DeviceNativeID,
		OutputPath:     plan.WebcamVideoPath,
		Profile:        normalized.Recording,
	}), nil
}
