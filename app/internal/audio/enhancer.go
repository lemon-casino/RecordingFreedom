package audio

import (
	"errors"
	"fmt"
	"time"
)

const (
	RNNoiseSampleRate     = 48000
	RNNoiseFrameDuration  = 10 * time.Millisecond
	RNNoiseFrameSamples   = 480
	rnnoiseRequiredTracks = 1
)

type StreamKind string

const (
	StreamSystemAudio StreamKind = "system-audio"
	StreamMicrophone  StreamKind = "microphone"
)

type Enhancement string

const (
	EnhancementOff     Enhancement = "off"
	EnhancementRNNoise Enhancement = "rnnoise"
)

type NoiseSuppressor interface {
	Name() string
	ProcessFrame(frame []float32) error
	Reset() error
}

type PCMBuffer struct {
	Kind        StreamKind
	SampleRate  int
	Channels    int
	Samples     []float32
	Enhancement Enhancement
}

type ProcessResult struct {
	Kind               StreamKind
	SampleRate         int
	Channels           int
	Samples            []float32
	EnhancementApplied Enhancement
	PendingSamples     int
	ProcessedFrames    int64
}

type Enhancer struct {
	suppressor NoiseSuppressor
	pending    []float32
	stats      EnhancerStats
}

type EnhancerStats struct {
	Engine              string      `json:"engine"`
	ProcessedFrames     int64       `json:"processedFrames"`
	ProcessedSamples    int64       `json:"processedSamples"`
	PendingSamples      int         `json:"pendingSamples"`
	ResetCount          int64       `json:"resetCount"`
	BypassedSamples     int64       `json:"bypassedSamples"`
	RejectedFrames      int64       `json:"rejectedFrames"`
	LastApplied         Enhancement `json:"lastApplied"`
	LastError           string      `json:"lastError,omitempty"`
	RequiredSampleRate  int         `json:"requiredSampleRate"`
	RequiredFrameSize   int         `json:"requiredFrameSize"`
	RequiredChannelMode string      `json:"requiredChannelMode"`
}

func NewEnhancer(suppressor NoiseSuppressor) *Enhancer {
	stats := EnhancerStats{
		Engine:              "rnnoise",
		LastApplied:         EnhancementOff,
		RequiredSampleRate:  RNNoiseSampleRate,
		RequiredFrameSize:   RNNoiseFrameSamples,
		RequiredChannelMode: "mono",
	}
	if suppressor != nil && suppressor.Name() != "" {
		stats.Engine = suppressor.Name()
	}
	return &Enhancer{suppressor: suppressor, stats: stats}
}

func (e *Enhancer) Process(buffer PCMBuffer) (ProcessResult, error) {
	if buffer.SampleRate == 0 {
		return e.reject("audio sample rate is required")
	}
	if buffer.Channels == 0 {
		return e.reject("audio channel count is required")
	}
	if buffer.Kind != StreamMicrophone || buffer.Enhancement != EnhancementRNNoise {
		e.stats.BypassedSamples += int64(len(buffer.Samples))
		e.stats.LastApplied = EnhancementOff
		e.stats.LastError = ""
		return ProcessResult{
			Kind:               buffer.Kind,
			SampleRate:         buffer.SampleRate,
			Channels:           buffer.Channels,
			Samples:            cloneSamples(buffer.Samples),
			EnhancementApplied: EnhancementOff,
		}, nil
	}
	if e.suppressor == nil {
		return e.reject("rnnoise enhancement requested without a noise suppressor")
	}
	if buffer.SampleRate != RNNoiseSampleRate {
		return e.reject(fmt.Sprintf("rnnoise requires %d Hz microphone PCM, got %d Hz", RNNoiseSampleRate, buffer.SampleRate))
	}
	if buffer.Channels != rnnoiseRequiredTracks {
		return e.reject(fmt.Sprintf("rnnoise requires mono microphone PCM, got %d channels", buffer.Channels))
	}

	combined := make([]float32, 0, len(e.pending)+len(buffer.Samples))
	combined = append(combined, e.pending...)
	combined = append(combined, buffer.Samples...)

	completeSamples := len(combined) / RNNoiseFrameSamples * RNNoiseFrameSamples
	processed := make([]float32, completeSamples)
	copy(processed, combined[:completeSamples])
	for offset := 0; offset < completeSamples; offset += RNNoiseFrameSamples {
		if err := e.suppressor.ProcessFrame(processed[offset : offset+RNNoiseFrameSamples]); err != nil {
			e.stats.LastError = err.Error()
			e.stats.RejectedFrames++
			return ProcessResult{}, err
		}
	}

	e.pending = append(e.pending[:0], combined[completeSamples:]...)
	processedFrames := int64(completeSamples / RNNoiseFrameSamples)
	e.stats.ProcessedFrames += processedFrames
	e.stats.ProcessedSamples += int64(completeSamples)
	e.stats.PendingSamples = len(e.pending)
	e.stats.LastApplied = EnhancementRNNoise
	e.stats.LastError = ""
	return ProcessResult{
		Kind:               buffer.Kind,
		SampleRate:         buffer.SampleRate,
		Channels:           buffer.Channels,
		Samples:            processed,
		EnhancementApplied: EnhancementRNNoise,
		PendingSamples:     len(e.pending),
		ProcessedFrames:    processedFrames,
	}, nil
}

func (e *Enhancer) Reset() error {
	e.pending = nil
	e.stats.PendingSamples = 0
	e.stats.ResetCount++
	if e.suppressor == nil {
		return nil
	}
	return e.suppressor.Reset()
}

func (e *Enhancer) Stats() EnhancerStats {
	stats := e.stats
	stats.PendingSamples = len(e.pending)
	return stats
}

func (e *Enhancer) reject(message string) (ProcessResult, error) {
	e.stats.RejectedFrames++
	e.stats.LastError = message
	return ProcessResult{}, errors.New(message)
}

func cloneSamples(samples []float32) []float32 {
	cloned := make([]float32, len(samples))
	copy(cloned, samples)
	return cloned
}
