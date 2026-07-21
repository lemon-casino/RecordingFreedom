package main

import "github.com/lemon-casino/RecordingFreedom/app/internal/settings"

type AudioState struct {
	System             bool    `json:"system"`
	SystemDeviceID     string  `json:"systemDeviceId,omitempty"`
	Microphone         bool    `json:"microphone"`
	MicrophoneDeviceID string  `json:"microphoneDeviceId,omitempty"`
	NoiseSuppression   bool    `json:"noiseSuppression"`
	MicrophoneGain     float64 `json:"microphoneGain"`
}

type AudioStatePatchRequest struct {
	System                *bool    `json:"system,omitempty"`
	SystemDeviceID        *string  `json:"systemDeviceId,omitempty"`
	Microphone            *bool    `json:"microphone,omitempty"`
	MicrophoneDeviceID    *string  `json:"microphoneDeviceId,omitempty"`
	NoiseSuppression      *bool    `json:"noiseSuppression,omitempty"`
	MicrophoneGain        *float64 `json:"microphoneGain,omitempty"`
	ClearSystemDevice     bool     `json:"clearSystemDevice,omitempty"`
	ClearMicrophoneDevice bool     `json:"clearMicrophoneDevice,omitempty"`
}

func audioStateFromSettings(audio settings.AudioSettings) AudioState {
	return AudioState{
		System:             audio.System,
		SystemDeviceID:     audio.SystemDeviceID,
		Microphone:         audio.Microphone,
		MicrophoneDeviceID: audio.MicrophoneDeviceID,
		NoiseSuppression:   audio.NoiseSuppression,
		MicrophoneGain:     audio.MicrophoneGain,
	}
}

func applyAudioStatePatch(audio settings.AudioSettings, patch AudioStatePatchRequest) settings.AudioSettings {
	if patch.System != nil {
		audio.System = *patch.System
	}
	if patch.ClearSystemDevice {
		audio.SystemDeviceID = ""
	}
	if patch.SystemDeviceID != nil {
		audio.SystemDeviceID = *patch.SystemDeviceID
	}
	if patch.Microphone != nil {
		audio.Microphone = *patch.Microphone
	}
	if patch.ClearMicrophoneDevice {
		audio.MicrophoneDeviceID = ""
	}
	if patch.MicrophoneDeviceID != nil {
		audio.MicrophoneDeviceID = *patch.MicrophoneDeviceID
	}
	if patch.NoiseSuppression != nil {
		audio.NoiseSuppression = *patch.NoiseSuppression
	}
	if patch.MicrophoneGain != nil {
		audio.MicrophoneGain = *patch.MicrophoneGain
	}
	return audio
}
