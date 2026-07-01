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
		SourceGeometry:  videoSourceGeometry(normalized.SourceGeometry),
		OutputPath:      plan.ScreenVideoPath,
		DiagnosticsPath: plan.VideoDiagnosticsPath,
		Profile:         normalized.Recording,
		SystemAudio:     normalized.Audio.System && plan.Package.Manifest.Media.SystemAudioStorage == recpackage.AudioStorageMuxed,
	}), nil
}

func videoSourceGeometry(geometry *SourceGeometry) *video.SourceGeometry {
	if geometry == nil {
		return nil
	}
	return &video.SourceGeometry{
		X:            geometry.X,
		Y:            geometry.Y,
		Width:        geometry.Width,
		Height:       geometry.Height,
		DisplayIndex: geometry.DisplayIndex,
		NativeID:     geometry.NativeID,
	}
}
