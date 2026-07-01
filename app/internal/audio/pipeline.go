package audio

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type CaptureConfig struct {
	Backend                    string
	SystemAudio                StreamConfig
	Microphone                 StreamConfig
	NoiseSuppression           bool
	MicrophoneGain             float64
	TargetSampleRate           int
	TargetChannels             int
	MaxQueuedFrames            int
	SystemAudioOutputPath      string
	MicrophoneAudioPath        string
	DiagnosticsPath            string
	SystemAudioIsNeverDenoised bool
}

type StreamConfig struct {
	Enabled  bool
	DeviceID string
}

type TimedPCMBuffer struct {
	Buffer    PCMBuffer
	Timestamp time.Duration
	Duration  time.Duration
}

type ProcessedBuffer struct {
	Buffer    PCMBuffer
	Timestamp time.Duration
	Duration  time.Duration
}

type CaptureSource interface {
	ID() string
	Kind() StreamKind
	Start(onFrame func(TimedPCMBuffer) error) error
	Pause() error
	Resume() error
	Stop() error
}

type CaptureSink interface {
	ID() string
	Append(ProcessedBuffer) error
	Close() error
}

type Pipeline struct {
	config      CaptureConfig
	enhancer    *Enhancer
	diagnostics Diagnostics
}

func NewPipeline(config CaptureConfig, enhancer *Enhancer) (*Pipeline, error) {
	config = normalizeCaptureConfig(config)
	if config.NoiseSuppression && config.Microphone.Enabled && enhancer == nil {
		return nil, errors.New("rnnoise pipeline requires an enhancer when microphone noise suppression is enabled")
	}
	diagnostics := Diagnostics{
		SchemaVersion: 1,
		CreatedAt:     time.Now(),
		Backend:       config.Backend,
		TargetFormat: Format{
			SampleRate: config.TargetSampleRate,
			Channels:   config.TargetChannels,
			SampleType: "float32",
		},
		SystemAudio: StreamDiagnostics{
			Enabled:  config.SystemAudio.Enabled,
			DeviceID: config.SystemAudio.DeviceID,
		},
		Microphone: StreamDiagnostics{
			Enabled:  config.Microphone.Enabled,
			DeviceID: config.Microphone.DeviceID,
		},
		Enhancement: EnhancementDiagnostics{
			Engine:              "rnnoise",
			Enabled:             config.Microphone.Enabled && config.NoiseSuppression,
			AppliedTo:           string(StreamMicrophone),
			SystemAudioBypassed: true,
			RequiredSampleRate:  RNNoiseSampleRate,
			RequiredFrameSize:   RNNoiseFrameSamples,
			RequiredChannelMode: "mono",
		},
		Queue: QueueDiagnostics{
			Capacity: config.MaxQueuedFrames,
		},
		Mixer: MixerDiagnostics{
			Enabled: config.SystemAudio.Enabled && config.Microphone.Enabled,
		},
	}
	return &Pipeline{config: config, enhancer: enhancer, diagnostics: diagnostics}, nil
}

func (p *Pipeline) Process(input TimedPCMBuffer) (ProcessedBuffer, error) {
	if err := p.ensureEnabled(input.Buffer.Kind); err != nil {
		return ProcessedBuffer{}, err
	}
	p.recordInput(input)

	output := input.Buffer
	if input.Buffer.Kind == StreamMicrophone && p.config.NoiseSuppression {
		microphone := input.Buffer
		microphone.Enhancement = EnhancementRNNoise
		result, err := p.enhancer.Process(microphone)
		p.updateEnhancementDiagnostics()
		if err != nil {
			p.recordAppendFailure(input.Buffer.Kind)
			return ProcessedBuffer{}, err
		}
		output = PCMBuffer{
			Kind:        result.Kind,
			SampleRate:  result.SampleRate,
			Channels:    result.Channels,
			Samples:     result.Samples,
			Enhancement: result.EnhancementApplied,
		}
	} else {
		if input.Buffer.Kind == StreamSystemAudio && input.Buffer.Enhancement == EnhancementRNNoise {
			p.diagnostics.Messages = append(p.diagnostics.Messages, "system audio requested RNNoise but was bypassed by policy")
		}
		output.Enhancement = EnhancementOff
	}

	p.recordOutput(output.Kind, len(output.Samples))
	return ProcessedBuffer{
		Buffer:    output,
		Timestamp: input.Timestamp,
		Duration:  input.Duration,
	}, nil
}

func (p *Pipeline) Reset() error {
	if p.enhancer == nil {
		return nil
	}
	err := p.enhancer.Reset()
	p.updateEnhancementDiagnostics()
	return err
}

func (p *Pipeline) RecordDroppedInput(kind StreamKind, samples int, reason string) {
	diagnostics := p.streamDiagnostics(kind)
	if diagnostics != nil {
		diagnostics.DroppedSamples += int64(samples)
		if diagnostics.Message == "" {
			diagnostics.Message = reason
		}
	}
	p.diagnostics.Queue.DroppedFrames++
	p.diagnostics.Queue.DroppedSamples += int64(samples)
	p.diagnostics.Mixer.DroppedSamples += int64(samples)
	if reason != "" && !containsMessage(p.diagnostics.Messages, reason) {
		p.diagnostics.Messages = append(p.diagnostics.Messages, reason)
	}
}

func (p *Pipeline) RecordQueueDepth(depth int, capacity int) {
	if capacity > 0 {
		p.diagnostics.Queue.Capacity = capacity
	}
	if depth > p.diagnostics.Queue.MaxDepth {
		p.diagnostics.Queue.MaxDepth = depth
	}
}

func (p *Pipeline) RecordQueueFlush() {
	p.diagnostics.Queue.FlushCount++
}

func (p *Pipeline) Diagnostics() Diagnostics {
	p.updateEnhancementDiagnostics()
	return p.diagnostics
}

func (p *Pipeline) WriteDiagnostics() error {
	return WriteDiagnostics(p.config.DiagnosticsPath, p.Diagnostics())
}

func (p *Pipeline) ensureEnabled(kind StreamKind) error {
	switch kind {
	case StreamSystemAudio:
		if !p.config.SystemAudio.Enabled {
			return errors.New("system audio frame received while system audio is disabled")
		}
	case StreamMicrophone:
		if !p.config.Microphone.Enabled {
			return errors.New("microphone frame received while microphone is disabled")
		}
	default:
		return fmt.Errorf("unsupported audio stream kind %q", kind)
	}
	return nil
}

func (p *Pipeline) recordInput(input TimedPCMBuffer) {
	diagnostics := p.streamDiagnostics(input.Buffer.Kind)
	if diagnostics == nil {
		return
	}
	if diagnostics.FramesReceived == 0 {
		diagnostics.StartOffsetMs = input.Timestamp.Milliseconds()
	}
	diagnostics.SampleRate = input.Buffer.SampleRate
	diagnostics.Channels = input.Buffer.Channels
	diagnostics.FramesReceived++
	diagnostics.SamplesReceived += int64(len(input.Buffer.Samples))
	end := input.Timestamp + input.Duration
	if input.Duration == 0 && input.Buffer.SampleRate > 0 && input.Buffer.Channels > 0 {
		frames := len(input.Buffer.Samples) / input.Buffer.Channels
		end = input.Timestamp + time.Duration(frames)*time.Second/time.Duration(input.Buffer.SampleRate)
	}
	diagnostics.EndOffsetMs = end.Milliseconds()
	diagnostics.DurationMs = diagnostics.EndOffsetMs - diagnostics.StartOffsetMs
}

func (p *Pipeline) recordOutput(kind StreamKind, samples int) {
	diagnostics := p.streamDiagnostics(kind)
	if diagnostics == nil {
		return
	}
	diagnostics.SamplesWritten += int64(samples)
	if kind == StreamSystemAudio {
		return
	}
	p.diagnostics.Mixer.OutputSamples += int64(samples)
}

func (p *Pipeline) recordAppendFailure(kind StreamKind) {
	diagnostics := p.streamDiagnostics(kind)
	if diagnostics != nil {
		diagnostics.AppendFailures++
	}
	p.diagnostics.Mixer.AppendFailures++
}

func (p *Pipeline) streamDiagnostics(kind StreamKind) *StreamDiagnostics {
	switch kind {
	case StreamSystemAudio:
		return &p.diagnostics.SystemAudio
	case StreamMicrophone:
		return &p.diagnostics.Microphone
	default:
		return nil
	}
}

func (p *Pipeline) updateEnhancementDiagnostics() {
	if p.enhancer == nil {
		return
	}
	stats := p.enhancer.Stats()
	p.diagnostics.Enhancement.Engine = stats.Engine
	p.diagnostics.Enhancement.ProcessedFrames = stats.ProcessedFrames
	p.diagnostics.Enhancement.ProcessedSamples = stats.ProcessedSamples
	p.diagnostics.Enhancement.PendingSamples = stats.PendingSamples
	p.diagnostics.Enhancement.ResetCount = stats.ResetCount
	p.diagnostics.Enhancement.RejectedFrames = stats.RejectedFrames
	p.diagnostics.Enhancement.LastError = stats.LastError
	p.diagnostics.Enhancement.RequiredSampleRate = stats.RequiredSampleRate
	p.diagnostics.Enhancement.RequiredFrameSize = stats.RequiredFrameSize
	p.diagnostics.Enhancement.RequiredChannelMode = stats.RequiredChannelMode
}

func normalizeCaptureConfig(config CaptureConfig) CaptureConfig {
	config.Backend = strings.TrimSpace(config.Backend)
	if config.Backend == "" {
		config.Backend = "native-audio"
	}
	if config.TargetSampleRate == 0 {
		config.TargetSampleRate = RNNoiseSampleRate
	}
	if config.TargetChannels == 0 {
		config.TargetChannels = 2
	}
	if config.MaxQueuedFrames <= 0 {
		config.MaxQueuedFrames = 128
	}
	config.SystemAudioIsNeverDenoised = true
	return config
}

func containsMessage(messages []string, needle string) bool {
	for _, message := range messages {
		if message == needle {
			return true
		}
	}
	return false
}
