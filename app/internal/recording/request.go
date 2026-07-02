package recording

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

const (
	defaultSystemAudioID  = "system-audio:default"
	defaultMicrophoneID   = "microphone:default"
	defaultCameraID       = "camera:default"
	defaultMicrophoneGain = 1
	maxMicrophoneGain     = 4
)

func NormalizeStartRequest(req StartRequest) (StartRequest, error) {
	req.SourceID = strings.TrimSpace(req.SourceID)
	req.SourceName = strings.TrimSpace(req.SourceName)
	if req.SourceID == "" {
		return StartRequest{}, errors.New("sourceId is required")
	}
	if !validSourceType(req.SourceType) {
		return StartRequest{}, fmt.Errorf("unsupported sourceType %q", req.SourceType)
	}
	if req.SourceGeometry != nil {
		req.SourceGeometry.NativeID = strings.TrimSpace(req.SourceGeometry.NativeID)
		if req.SourceGeometry.Width < 0 || req.SourceGeometry.Height < 0 {
			return StartRequest{}, errors.New("sourceGeometry width and height must be non-negative")
		}
	}
	req.Recording = recordingprofile.Normalize(req.Recording)

	audioRequest, err := normalizeAudioRequest(req.Audio)
	if err != nil {
		return StartRequest{}, err
	}
	req.Audio = audioRequest

	req.Camera.DeviceID = strings.TrimSpace(req.Camera.DeviceID)
	req.Camera.DeviceNativeID = strings.TrimSpace(req.Camera.DeviceNativeID)
	if req.Camera.Enabled {
		if req.Camera.DeviceID == "" {
			req.Camera.DeviceID = defaultCameraID
		}
		req.Camera.PIP = pip.NormalizeConfigForPreset(req.Camera.PIPPreset, req.Camera.PIP)
		req.Camera.PIPPreset = string(req.Camera.PIP.Preset)
	} else {
		req.Camera.DeviceID = ""
		req.Camera.DeviceNativeID = ""
		req.Camera.PIPPreset = string(pip.PresetOff)
		req.Camera.PIP = pip.OffConfig()
	}
	return req, nil
}

func NormalizeAudioOnlyRequest(req AudioOnlyRequest) (AudioOnlyRequest, error) {
	req.Recording = recordingprofile.Normalize(req.Recording)
	audioRequest, err := normalizeAudioRequest(req.Audio)
	if err != nil {
		return AudioOnlyRequest{}, err
	}
	if !audioRequest.System && !audioRequest.Microphone {
		return AudioOnlyRequest{}, errors.New("audio-only recording requires system audio or microphone")
	}
	req.Audio = audioRequest
	return req, nil
}

func normalizeAudioRequest(req AudioRequest) (AudioRequest, error) {
	req.SystemDeviceID = strings.TrimSpace(req.SystemDeviceID)
	req.MicrophoneID = strings.TrimSpace(req.MicrophoneID)
	if req.System {
		if req.SystemDeviceID == "" {
			req.SystemDeviceID = defaultSystemAudioID
		}
	} else {
		req.SystemDeviceID = ""
	}
	if req.Microphone {
		if req.MicrophoneID == "" {
			req.MicrophoneID = defaultMicrophoneID
		}
		if req.MicrophoneGain == 0 {
			req.MicrophoneGain = defaultMicrophoneGain
		}
		if req.MicrophoneGain < 0 || req.MicrophoneGain > maxMicrophoneGain {
			return AudioRequest{}, fmt.Errorf("microphoneGain must be between 0 and %d", maxMicrophoneGain)
		}
	} else {
		req.MicrophoneID = ""
		req.NoiseSuppression = false
		req.MicrophoneGain = 0
	}
	return req, nil
}

func validSourceType(sourceType CaptureSourceType) bool {
	switch sourceType {
	case SourceScreen, SourceAllScreens, SourceRegion, SourceWindow, SourceApplication:
		return true
	default:
		return false
	}
}
