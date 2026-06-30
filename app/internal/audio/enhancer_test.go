package audio

import (
	"reflect"
	"testing"
)

func TestSystemAudioBypassesRNNoise(t *testing.T) {
	suppressor := &spySuppressor{}
	enhancer := NewEnhancer(suppressor)
	input := []float32{0.1, 0.2, 0.3, 0.4}

	result, err := enhancer.Process(PCMBuffer{
		Kind:        StreamSystemAudio,
		SampleRate:  RNNoiseSampleRate,
		Channels:    2,
		Samples:     input,
		Enhancement: EnhancementRNNoise,
	})
	if err != nil {
		t.Fatalf("Process(system) error = %v", err)
	}
	if suppressor.frames != 0 {
		t.Fatalf("system audio entered suppressor %d time(s), want 0", suppressor.frames)
	}
	if result.EnhancementApplied != EnhancementOff {
		t.Fatalf("system enhancement = %q, want off", result.EnhancementApplied)
	}
	if !reflect.DeepEqual(result.Samples, input) {
		t.Fatalf("system samples changed: got %#v want %#v", result.Samples, input)
	}
}

func TestMicrophoneRNNoiseUsesTenMillisecondFrames(t *testing.T) {
	suppressor := &spySuppressor{}
	enhancer := NewEnhancer(suppressor)

	result, err := enhancer.Process(PCMBuffer{
		Kind:        StreamMicrophone,
		SampleRate:  RNNoiseSampleRate,
		Channels:    1,
		Samples:     make([]float32, RNNoiseFrameSamples*2),
		Enhancement: EnhancementRNNoise,
	})
	if err != nil {
		t.Fatalf("Process(microphone) error = %v", err)
	}
	if suppressor.frames != 2 {
		t.Fatalf("frames = %d, want 2", suppressor.frames)
	}
	if suppressor.lastFrameLength != RNNoiseFrameSamples {
		t.Fatalf("frame length = %d, want %d", suppressor.lastFrameLength, RNNoiseFrameSamples)
	}
	if len(result.Samples) != RNNoiseFrameSamples*2 {
		t.Fatalf("processed samples = %d, want %d", len(result.Samples), RNNoiseFrameSamples*2)
	}
	if result.EnhancementApplied != EnhancementRNNoise {
		t.Fatalf("enhancement = %q, want rnnoise", result.EnhancementApplied)
	}
}

func TestMicrophoneRNNoiseCarriesPartialFrame(t *testing.T) {
	suppressor := &spySuppressor{}
	enhancer := NewEnhancer(suppressor)

	first, err := enhancer.Process(PCMBuffer{
		Kind:        StreamMicrophone,
		SampleRate:  RNNoiseSampleRate,
		Channels:    1,
		Samples:     make([]float32, RNNoiseFrameSamples+20),
		Enhancement: EnhancementRNNoise,
	})
	if err != nil {
		t.Fatalf("Process(first) error = %v", err)
	}
	if len(first.Samples) != RNNoiseFrameSamples || first.PendingSamples != 20 {
		t.Fatalf("first result = samples:%d pending:%d", len(first.Samples), first.PendingSamples)
	}

	second, err := enhancer.Process(PCMBuffer{
		Kind:        StreamMicrophone,
		SampleRate:  RNNoiseSampleRate,
		Channels:    1,
		Samples:     make([]float32, RNNoiseFrameSamples-20),
		Enhancement: EnhancementRNNoise,
	})
	if err != nil {
		t.Fatalf("Process(second) error = %v", err)
	}
	if len(second.Samples) != RNNoiseFrameSamples || second.PendingSamples != 0 {
		t.Fatalf("second result = samples:%d pending:%d", len(second.Samples), second.PendingSamples)
	}
	if suppressor.frames != 2 {
		t.Fatalf("frames = %d, want 2", suppressor.frames)
	}
}

func TestResetClearsPendingAndSuppressorState(t *testing.T) {
	suppressor := &spySuppressor{}
	enhancer := NewEnhancer(suppressor)
	if _, err := enhancer.Process(PCMBuffer{
		Kind:        StreamMicrophone,
		SampleRate:  RNNoiseSampleRate,
		Channels:    1,
		Samples:     make([]float32, 17),
		Enhancement: EnhancementRNNoise,
	}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if err := enhancer.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if suppressor.resets != 1 {
		t.Fatalf("resets = %d, want 1", suppressor.resets)
	}
	result, err := enhancer.Process(PCMBuffer{
		Kind:        StreamMicrophone,
		SampleRate:  RNNoiseSampleRate,
		Channels:    1,
		Samples:     make([]float32, RNNoiseFrameSamples-1),
		Enhancement: EnhancementRNNoise,
	})
	if err != nil {
		t.Fatalf("Process(after reset) error = %v", err)
	}
	if result.PendingSamples != RNNoiseFrameSamples-1 || suppressor.frames != 0 {
		t.Fatalf("after reset = pending:%d frames:%d", result.PendingSamples, suppressor.frames)
	}
}

func TestRNNoiseRejectsNonMonoMicrophonePCM(t *testing.T) {
	_, err := NewEnhancer(&spySuppressor{}).Process(PCMBuffer{
		Kind:        StreamMicrophone,
		SampleRate:  RNNoiseSampleRate,
		Channels:    2,
		Samples:     make([]float32, RNNoiseFrameSamples),
		Enhancement: EnhancementRNNoise,
	})
	if err == nil {
		t.Fatal("Process() accepted stereo microphone PCM for RNNoise")
	}
}

type spySuppressor struct {
	frames          int
	lastFrameLength int
	resets          int
}

func (s *spySuppressor) Name() string {
	return "spy"
}

func (s *spySuppressor) ProcessFrame(frame []float32) error {
	s.frames++
	s.lastFrameLength = len(frame)
	return nil
}

func (s *spySuppressor) Reset() error {
	s.frames = 0
	s.resets++
	return nil
}
