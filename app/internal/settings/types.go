package settings

import "time"

import (
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

const SchemaVersion = 1

type Locale string

const (
	LocaleZhCN Locale = "zh-CN"
	LocaleEN   Locale = "en"
)

type Settings struct {
	SchemaVersion int               `json:"schemaVersion"`
	Locale        Locale            `json:"locale"`
	Source        SourceSettings    `json:"source"`
	Storage       StorageSettings   `json:"storage"`
	Recording     RecordingSettings `json:"recording"`
	Audio         AudioSettings     `json:"audio"`
	Camera        CameraSettings    `json:"camera"`
	Window        WindowSettings    `json:"window"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type RecordingSettings = recordingprofile.Profile

type SourceSettings struct {
	LastSourceID   string `json:"lastSourceId,omitempty"`
	LastSourceType string `json:"lastSourceType"`
}

type StorageSettings struct {
	DataRootDir string `json:"dataRootDir,omitempty"`
}

type AudioSettings struct {
	System             bool    `json:"system"`
	SystemDeviceID     string  `json:"systemDeviceId,omitempty"`
	Microphone         bool    `json:"microphone"`
	MicrophoneDeviceID string  `json:"microphoneDeviceId,omitempty"`
	NoiseSuppression   bool    `json:"noiseSuppression"`
	MicrophoneGain     float64 `json:"microphoneGain"`
}

type CameraSettings struct {
	Enabled   bool       `json:"enabled"`
	DeviceID  string     `json:"deviceId,omitempty"`
	PIPPreset string     `json:"pipPreset"`
	PIP       pip.Config `json:"pip"`
}

type WindowSettings struct {
	MinimizeToTray bool `json:"minimizeToTray"`
}
