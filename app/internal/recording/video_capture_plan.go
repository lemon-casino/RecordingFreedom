package recording

import (
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func CreateVideoCaptureConfig(backendID string, req StartRequest, plan recpackage.RecordingWritePlan) (video.CaptureConfig, error) {
	normalized, err := NormalizeStartRequest(req)
	if err != nil {
		return video.CaptureConfig{}, err
	}
	return video.NormalizeCaptureConfig(video.CaptureConfig{
		Backend:         backendID,
		SourceID:        normalized.SourceID,
		SourceType:      normalized.SourceType,
		SourceName:      normalized.SourceName,
		OutputPath:      plan.ScreenVideoPath,
		DiagnosticsPath: plan.VideoDiagnosticsPath,
		Profile:         normalized.Recording,
	}), nil
}
