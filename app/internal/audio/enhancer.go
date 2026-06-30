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
}

type Enhancer struct {
	suppressor NoiseSuppressor
	pending    []float32
}

func NewEnhancer(suppressor NoiseSuppressor) *Enhancer {
	return &Enhancer{suppressor: suppressor}
}

func (e *Enhancer) Process(buffer PCMBuffer) (ProcessResult, error) {
	if buffer.SampleRate == 0 {
		return ProcessResult{}, errors.New("audio sample rate is required")
	}
	if buffer.Channels == 0 {
		return ProcessResult{}, errors.New("audio channel count is required")
	}
	if buffer.Kind != StreamMicrophone || buffer.Enhancement != EnhancementRNNoise {
		return ProcessResult{
			Kind:               buffer.Kind,
			SampleRate:         buffer.SampleRate,
			Channels:           buffer.Channels,
			Samples:            cloneSamples(buffer.Samples),
			EnhancementApplied: EnhancementOff,
		}, nil
	}
	if e.suppressor == nil {
		return ProcessResult{}, errors.New("rnnoise enhancement requested without a noise suppressor")
	}
	if buffer.SampleRate != RNNoiseSampleRate {
		return ProcessResult{}, fmt.Errorf("rnnoise requires %d Hz microphone PCM, got %d Hz", RNNoiseSampleRate, buffer.SampleRate)
	}
	if buffer.Channels != rnnoiseRequiredTracks {
		return ProcessResult{}, fmt.Errorf("rnnoise requires mono microphone PCM, got %d channels", buffer.Channels)
	}

	combined := make([]float32, 0, len(e.pending)+len(buffer.Samples))
	combined = append(combined, e.pending...)
	combined = append(combined, buffer.Samples...)

	completeSamples := len(combined) / RNNoiseFrameSamples * RNNoiseFrameSamples
	processed := make([]float32, completeSamples)
	copy(processed, combined[:completeSamples])
	for offset := 0; offset < completeSamples; offset += RNNoiseFrameSamples {
		if err := e.suppressor.ProcessFrame(processed[offset : offset+RNNoiseFrameSamples]); err != nil {
			return ProcessResult{}, err
		}
	}

	e.pending = append(e.pending[:0], combined[completeSamples:]...)
	return ProcessResult{
		Kind:               buffer.Kind,
		SampleRate:         buffer.SampleRate,
		Channels:           buffer.Channels,
		Samples:            processed,
		EnhancementApplied: EnhancementRNNoise,
		PendingSamples:     len(e.pending),
	}, nil
}

func (e *Enhancer) Reset() error {
	e.pending = nil
	if e.suppressor == nil {
		return nil
	}
	return e.suppressor.Reset()
}

func cloneSamples(samples []float32) []float32 {
	cloned := make([]float32, len(samples))
	copy(cloned, samples)
	return cloned
}
