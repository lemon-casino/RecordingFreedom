package recording

import (
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func CreateAudioCaptureConfig(backendID string, req StartRequest, plan recpackage.RecordingWritePlan) (audio.CaptureConfig, error) {
	normalized, err := NormalizeStartRequest(req)
	if err != nil {
		return audio.CaptureConfig{}, err
	}
	config := audio.CaptureConfig{
		Backend:          backendID,
		TargetSampleRate: audio.RNNoiseSampleRate,
		TargetChannels:   2,
		SystemAudio: audio.StreamConfig{
			Enabled:  normalized.Audio.System,
			DeviceID: normalized.Audio.SystemDeviceID,
		},
		Microphone: audio.StreamConfig{
			Enabled:  normalized.Audio.Microphone,
			DeviceID: normalized.Audio.MicrophoneID,
		},
		NoiseSuppression:           normalized.Audio.Microphone && normalized.Audio.NoiseSuppression,
		MicrophoneGain:             normalized.Audio.MicrophoneGain,
		SystemAudioOutputPath:      plan.ScreenVideoPath,
		MicrophoneAudioPath:        plan.ScreenVideoPath,
		DiagnosticsPath:            plan.AudioDiagnosticsPath,
		SystemAudioIsNeverDenoised: true,
	}
	if !config.SystemAudio.Enabled {
		config.SystemAudioOutputPath = ""
	}
	if !config.Microphone.Enabled {
		config.MicrophoneAudioPath = ""
		config.NoiseSuppression = false
	}
	return config, nil
}
