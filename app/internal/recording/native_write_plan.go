package recording

import (
	"errors"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func CreateNativeWritePlan(packages *recpackage.Service, backendID string, req BackendStartRequest) (recpackage.RecordingWritePlan, error) {
	if packages == nil {
		return recpackage.RecordingWritePlan{}, errors.New("recpackage service is required")
	}
	normalized, err := NormalizeStartRequest(req.StartRequest)
	if err != nil {
		return recpackage.RecordingWritePlan{}, err
	}
	backendID = strings.TrimSpace(backendID)
	if backendID == "" {
		backendID = BackendNativeUnsupported
	}
	return packages.CreateNative(req.VideoDir, recpackage.CreateNativeRequest{
		CreatedAt: req.CreatedAt,
		Status:    recpackage.StatusRecording,
		Backend:   backendID,
		Source: recpackage.ManifestSource{
			Type: string(normalized.SourceType),
			ID:   normalized.SourceID,
			Name: normalized.SourceName,
		},
		Recording: normalized.Recording,
		Audio: recpackage.ManifestAudio{
			System:                     normalized.Audio.System,
			SystemDeviceID:             normalized.Audio.SystemDeviceID,
			Microphone:                 normalized.Audio.Microphone,
			MicrophoneDeviceID:         normalized.Audio.MicrophoneID,
			MicrophoneNoiseSuppression: noiseSuppressionLabel(normalized.Audio.NoiseSuppression),
			MicrophoneGain:             normalized.Audio.MicrophoneGain,
		},
		Camera: recpackage.ManifestCamera{
			Enabled:   normalized.Camera.Enabled,
			DeviceID:  normalized.Camera.DeviceID,
			PIPPreset: normalized.Camera.PIPPreset,
		},
	})
}
