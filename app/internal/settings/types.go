package settings

import "time"

import (
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

const SchemaVersion = 2

type Locale string

const (
	LocaleZhCN Locale = "zh-CN"
	LocaleEN   Locale = "en"
)

type Theme string

const (
	ThemeNightTeal     Theme = "night-teal"
	ThemeMountainGreen Theme = "mountain-green"
	ThemeSkyBlue       Theme = "sky-blue"
	ThemeSunsetYellow  Theme = "sunset-yellow"
	ThemeInkPurple     Theme = "ink-purple"
	ThemeSageGray      Theme = "sage-gray"
)

type Settings struct {
	SchemaVersion int                `json:"schemaVersion"`
	Locale        Locale             `json:"locale"`
	Source        SourceSettings     `json:"source"`
	Storage       StorageSettings    `json:"storage"`
	Recording     RecordingSettings  `json:"recording"`
	Audio         AudioSettings      `json:"audio"`
	Camera        CameraSettings     `json:"camera"`
	Whiteboard    WhiteboardSettings `json:"whiteboard"`
	Shortcuts     ShortcutSettings   `json:"shortcuts"`
	Window        WindowSettings     `json:"window"`
	UpdatedAt     time.Time          `json:"updatedAt"`
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

type WhiteboardSettings struct {
	Enabled         bool   `json:"enabled"`
	LastMode        string `json:"lastMode"`
	LastTool        string `json:"lastTool"`
	LastStrokeColor string `json:"lastStrokeColor"`
	LastStrokeWidth string `json:"lastStrokeWidth"`
	LastOpacity     int    `json:"lastOpacity"`
	CapturePolicy   string `json:"capturePolicy"`
}

type ShortcutAction string

const (
	ShortcutActionToggleRecording ShortcutAction = "toggleRecording"
	ShortcutActionTogglePause     ShortcutAction = "togglePause"
	ShortcutActionToggleCamera    ShortcutAction = "toggleCamera"
	ShortcutActionOpenWhiteboard  ShortcutAction = "openWhiteboard"
	ShortcutActionOpenScreenshot  ShortcutAction = "openScreenshot"
)

type ShortcutSettings struct {
	ToggleRecording string `json:"toggleRecording"`
	TogglePause     string `json:"togglePause"`
	ToggleCamera    string `json:"toggleCamera"`
	OpenWhiteboard  string `json:"openWhiteboard"`
	OpenScreenshot  string `json:"openScreenshot"`
}

type ShortcutBinding struct {
	Action      ShortcutAction
	Accelerator string
}

type WindowSettings struct {
	MinimizeToTray bool  `json:"minimizeToTray"`
	Theme          Theme `json:"theme"`
	StartAtLogin   bool  `json:"startAtLogin"`
}
