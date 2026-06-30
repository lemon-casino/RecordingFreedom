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
	req.Recording = recordingprofile.Normalize(req.Recording)

	req.Audio.SystemDeviceID = strings.TrimSpace(req.Audio.SystemDeviceID)
	req.Audio.MicrophoneID = strings.TrimSpace(req.Audio.MicrophoneID)
	if req.Audio.System {
		if req.Audio.SystemDeviceID == "" {
			req.Audio.SystemDeviceID = defaultSystemAudioID
		}
	} else {
		req.Audio.SystemDeviceID = ""
	}
	if req.Audio.Microphone {
		if req.Audio.MicrophoneID == "" {
			req.Audio.MicrophoneID = defaultMicrophoneID
		}
		if req.Audio.MicrophoneGain == 0 {
			req.Audio.MicrophoneGain = defaultMicrophoneGain
		}
		if req.Audio.MicrophoneGain < 0 || req.Audio.MicrophoneGain > maxMicrophoneGain {
			return StartRequest{}, fmt.Errorf("microphoneGain must be between 0 and %d", maxMicrophoneGain)
		}
	} else {
		req.Audio.MicrophoneID = ""
		req.Audio.NoiseSuppression = false
		req.Audio.MicrophoneGain = 0
	}

	req.Camera.DeviceID = strings.TrimSpace(req.Camera.DeviceID)
	if req.Camera.Enabled {
		if req.Camera.DeviceID == "" {
			req.Camera.DeviceID = defaultCameraID
		}
		req.Camera.PIPPreset = string(pip.Normalize(req.Camera.PIPPreset))
	} else {
		req.Camera.DeviceID = ""
		req.Camera.PIPPreset = string(pip.PresetOff)
	}
	return req, nil
}

func validSourceType(sourceType CaptureSourceType) bool {
	switch sourceType {
	case SourceScreen, SourceWindow, SourceApplication:
		return true
	default:
		return false
	}
}
