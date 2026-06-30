package video

import (
	"context"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

type SourceType = devices.CaptureSourceType

type CaptureConfig struct {
	Backend         string
	SourceID        string
	SourceType      SourceType
	SourceName      string
	OutputPath      string
	DiagnosticsPath string
	Profile         recordingprofile.Profile
	SystemAudio     bool
}

type Session interface {
	Start(context.Context) error
	Pause() error
	Resume() error
	Stop() error
	Diagnostics() Diagnostics
}

type Diagnostics struct {
	SchemaVersion int              `json:"schemaVersion"`
	CreatedAt     time.Time        `json:"createdAt"`
	Backend       string           `json:"backend"`
	Source        SourceDiagnostic `json:"source"`
	Recording     RecordingProfile `json:"recording"`
	OutputPath    string           `json:"outputPath,omitempty"`
	Screen        TrackDiagnostics `json:"screen"`
	SystemAudio   TrackDiagnostics `json:"systemAudio"`
	Messages      []string         `json:"messages,omitempty"`
}

type SourceDiagnostic struct {
	ID   string     `json:"id"`
	Type SourceType `json:"type"`
	Name string     `json:"name,omitempty"`
}

type RecordingProfile struct {
	Quality          string `json:"quality"`
	FPS              int    `json:"fps"`
	CaptureCursor    bool   `json:"captureCursor"`
	CountdownSeconds int    `json:"countdownSeconds"`
}

type TrackDiagnostics struct {
	Enabled        bool   `json:"enabled"`
	Path           string `json:"path,omitempty"`
	Clock          string `json:"clock,omitempty"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	FrameRate      int    `json:"frameRate,omitempty"`
	FramesWritten  int64  `json:"framesWritten"`
	DroppedFrames  int64  `json:"droppedFrames"`
	SamplesWritten int64  `json:"samplesWritten,omitempty"`
	DroppedSamples int64  `json:"droppedSamples,omitempty"`
	AppendFailures int64  `json:"appendFailures"`
	SampleRate     int    `json:"sampleRate,omitempty"`
	StartOffsetMs  int64  `json:"startOffsetMs"`
	EndOffsetMs    int64  `json:"endOffsetMs"`
	DurationMs     int64  `json:"durationMs"`
	Message        string `json:"message,omitempty"`
}

func NormalizeCaptureConfig(config CaptureConfig) CaptureConfig {
	config.Backend = strings.TrimSpace(config.Backend)
	if config.Backend == "" {
		config.Backend = "native-video"
	}
	config.SourceID = strings.TrimSpace(config.SourceID)
	config.SourceName = strings.TrimSpace(config.SourceName)
	config.OutputPath = strings.TrimSpace(config.OutputPath)
	config.DiagnosticsPath = strings.TrimSpace(config.DiagnosticsPath)
	config.Profile = recordingprofile.Normalize(config.Profile)
	return config
}

func NewDiagnostics(config CaptureConfig) Diagnostics {
	config = NormalizeCaptureConfig(config)
	return Diagnostics{
		SchemaVersion: 1,
		CreatedAt:     time.Now(),
		Backend:       config.Backend,
		Source: SourceDiagnostic{
			ID:   config.SourceID,
			Type: config.SourceType,
			Name: config.SourceName,
		},
		Recording: RecordingProfile{
			Quality:          config.Profile.Quality,
			FPS:              config.Profile.FPS,
			CaptureCursor:    config.Profile.CaptureCursor,
			CountdownSeconds: config.Profile.CountdownSeconds,
		},
		OutputPath: config.OutputPath,
		Screen: TrackDiagnostics{
			Enabled:   true,
			FrameRate: config.Profile.FPS,
		},
		SystemAudio: TrackDiagnostics{
			Enabled: config.SystemAudio,
		},
	}
}
