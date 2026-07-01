package audio

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Diagnostics struct {
	SchemaVersion int                    `json:"schemaVersion"`
	CreatedAt     time.Time              `json:"createdAt"`
	Backend       string                 `json:"backend"`
	TargetFormat  Format                 `json:"targetFormat"`
	SystemAudio   StreamDiagnostics      `json:"systemAudio"`
	Microphone    StreamDiagnostics      `json:"microphone"`
	Enhancement   EnhancementDiagnostics `json:"enhancement"`
	Queue         QueueDiagnostics       `json:"queue"`
	Mixer         MixerDiagnostics       `json:"mixer"`
	Messages      []string               `json:"messages,omitempty"`
}

type Format struct {
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
	SampleType string `json:"sampleType"`
}

type StreamDiagnostics struct {
	Enabled         bool   `json:"enabled"`
	DeviceID        string `json:"deviceId,omitempty"`
	SampleRate      int    `json:"sampleRate,omitempty"`
	Channels        int    `json:"channels,omitempty"`
	FramesReceived  int64  `json:"framesReceived"`
	SamplesReceived int64  `json:"samplesReceived"`
	SamplesWritten  int64  `json:"samplesWritten"`
	DroppedSamples  int64  `json:"droppedSamples"`
	AppendFailures  int64  `json:"appendFailures"`
	StartOffsetMs   int64  `json:"startOffsetMs"`
	EndOffsetMs     int64  `json:"endOffsetMs"`
	DurationMs      int64  `json:"durationMs"`
	Message         string `json:"message,omitempty"`
}

type EnhancementDiagnostics struct {
	Engine              string `json:"engine"`
	Enabled             bool   `json:"enabled"`
	AppliedTo           string `json:"appliedTo"`
	SystemAudioBypassed bool   `json:"systemAudioBypassed"`
	ProcessedFrames     int64  `json:"processedFrames"`
	ProcessedSamples    int64  `json:"processedSamples"`
	PendingSamples      int    `json:"pendingSamples"`
	ResetCount          int64  `json:"resetCount"`
	RejectedFrames      int64  `json:"rejectedFrames"`
	LastError           string `json:"lastError,omitempty"`
	RequiredSampleRate  int    `json:"requiredSampleRate"`
	RequiredFrameSize   int    `json:"requiredFrameSize"`
	RequiredChannelMode string `json:"requiredChannelMode"`
}

type QueueDiagnostics struct {
	Capacity       int   `json:"capacity"`
	MaxDepth       int   `json:"maxDepth"`
	FlushCount     int64 `json:"flushCount"`
	DroppedFrames  int64 `json:"droppedFrames"`
	DroppedSamples int64 `json:"droppedSamples"`
}

type MixerDiagnostics struct {
	Enabled        bool  `json:"enabled"`
	OutputSamples  int64 `json:"outputSamples"`
	DroppedSamples int64 `json:"droppedSamples"`
	AppendFailures int64 `json:"appendFailures"`
}

func WriteDiagnostics(path string, diagnostics Diagnostics) error {
	if path == "" {
		return errors.New("audio diagnostics path is required")
	}
	if diagnostics.SchemaVersion == 0 {
		diagnostics.SchemaVersion = 1
	}
	if diagnostics.CreatedAt.IsZero() {
		diagnostics.CreatedAt = time.Now()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
