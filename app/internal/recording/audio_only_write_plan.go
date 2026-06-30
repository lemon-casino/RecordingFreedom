package recording

import (
	"errors"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func CreateAudioOnlyWritePlan(packages *recpackage.Service, backendID string, videoDir string, createdAt time.Time, req AudioOnlyRequest) (recpackage.RecordingWritePlan, AudioOnlyRequest, error) {
	if packages == nil {
		return recpackage.RecordingWritePlan{}, AudioOnlyRequest{}, errors.New("recpackage service is required")
	}
	normalized, err := NormalizeAudioOnlyRequest(req)
	if err != nil {
		return recpackage.RecordingWritePlan{}, AudioOnlyRequest{}, err
	}
	backendID = strings.TrimSpace(backendID)
	if backendID == "" {
		backendID = BackendAudioOnlyNative
	}
	createRequest := recpackage.CreateAudioOnlyRequest{
		CreatedAt: createdAt,
		Status:    recpackage.StatusRecording,
		Backend:   backendID,
		Recording: normalized.Recording,
		Audio: recpackage.ManifestAudio{
			System:                     normalized.Audio.System,
			SystemDeviceID:             normalized.Audio.SystemDeviceID,
			Microphone:                 normalized.Audio.Microphone,
			MicrophoneDeviceID:         normalized.Audio.MicrophoneID,
			MicrophoneNoiseSuppression: noiseSuppressionLabel(normalized.Audio.NoiseSuppression),
			MicrophoneGain:             normalized.Audio.MicrophoneGain,
		},
	}
	applyAudioOnlyWAVFallback(&createRequest, normalized.Audio)
	plan, err := packages.CreateAudioOnly(videoDir, createRequest)
	if err != nil {
		return recpackage.RecordingWritePlan{}, AudioOnlyRequest{}, err
	}
	return plan, normalized, nil
}

func CreateAudioOnlyCaptureConfig(backendID string, req AudioOnlyRequest, plan recpackage.RecordingWritePlan) (audio.CaptureConfig, error) {
	normalized, err := NormalizeAudioOnlyRequest(req)
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
		SystemAudioOutputPath:      plan.SystemAudioPath,
		MicrophoneAudioPath:        plan.MicrophoneAudioPath,
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

func applyAudioOnlyWAVFallback(request *recpackage.CreateAudioOnlyRequest, audioRequest AudioRequest) {
	if request == nil {
		return
	}
	if audioRequest.Microphone && audioRequest.System {
		request.AudioPath = recpackage.MicrophoneAudioFile
		request.MicrophoneAudioPath = recpackage.MicrophoneAudioFile
		request.MicrophoneAudioStorage = recpackage.AudioStorageSidecar
		request.SystemAudioPath = recpackage.SystemAudioFile
		request.SystemAudioStorage = recpackage.AudioStorageSidecar
		return
	}
	request.AudioPath = recpackage.AudioOnlyWAVFile
	if audioRequest.Microphone {
		request.MicrophoneAudioPath = recpackage.AudioOnlyWAVFile
		request.MicrophoneAudioStorage = recpackage.AudioStorageSidecar
	}
	if audioRequest.System {
		request.SystemAudioPath = recpackage.AudioOnlyWAVFile
		request.SystemAudioStorage = recpackage.AudioStorageSidecar
	}
}
