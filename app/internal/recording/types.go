package recording

import (
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

type CaptureSourceType = devices.CaptureSourceType

const (
	SourceScreen      = devices.SourceScreen
	SourceAllScreens  = devices.SourceAllScreens
	SourceRegion      = devices.SourceRegion
	SourceWindow      = devices.SourceWindow
	SourceApplication = devices.SourceApplication
)

type State string

const (
	StateIdle      State = "idle"
	StatePreparing State = "preparing"
	StateRecording State = "recording"
	StatePaused    State = "paused"
	StateStopping  State = "stopping"
	StateReady     State = "ready"
	StateFailed    State = "failed"
)

type AudioRequest struct {
	System           bool    `json:"system"`
	SystemDeviceID   string  `json:"systemDeviceId,omitempty"`
	Microphone       bool    `json:"microphone"`
	MicrophoneID     string  `json:"microphoneDeviceId,omitempty"`
	NoiseSuppression bool    `json:"noiseSuppression"`
	MicrophoneGain   float64 `json:"microphoneGain"`
}

type CameraRequest struct {
	Enabled        bool       `json:"enabled"`
	DeviceID       string     `json:"deviceId,omitempty"`
	DeviceNativeID string     `json:"deviceNativeId,omitempty"`
	PIPPreset      string     `json:"pipPreset"`
	PIP            pip.Config `json:"pip"`
}

type SourceGeometry struct {
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	DisplayIndex int    `json:"displayIndex,omitempty"`
	NativeID     string `json:"nativeId,omitempty"`
}

type StartRequest struct {
	SourceID       string                   `json:"sourceId"`
	SourceType     CaptureSourceType        `json:"sourceType"`
	SourceName     string                   `json:"sourceName,omitempty"`
	SourceGeometry *SourceGeometry          `json:"sourceGeometry,omitempty"`
	Recording      recordingprofile.Profile `json:"recording"`
	Audio          AudioRequest             `json:"audio"`
	Camera         CameraRequest            `json:"camera"`
}

type AudioOnlyRequest struct {
	Recording recordingprofile.Profile `json:"recording"`
	Audio     AudioRequest             `json:"audio"`
}

type Session struct {
	ID            string    `json:"id"`
	PackageDir    string    `json:"packageDir"`
	Manifest      string    `json:"manifest"`
	Backend       string    `json:"backend"`
	RecordingMode string    `json:"recordingMode"`
	Status        State     `json:"status"`
	StartedAt     time.Time `json:"startedAt"`
	CompletedAt   time.Time `json:"completedAt,omitempty"`
}

type StatusEvent struct {
	Status     State  `json:"status"`
	SessionID  string `json:"sessionId,omitempty"`
	PackageDir string `json:"packageDir,omitempty"`
	Manifest   string `json:"manifest,omitempty"`
	Backend    string `json:"backend,omitempty"`
	Message    string `json:"message,omitempty"`
}
