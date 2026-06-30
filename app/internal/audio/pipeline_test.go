package audio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPipelineBypassesSystemAudioRNNoise(t *testing.T) {
	suppressor := &spySuppressor{}
	pipeline, err := NewPipeline(CaptureConfig{
		Backend:          "test-audio",
		SystemAudio:      StreamConfig{Enabled: true, DeviceID: "system-audio:default"},
		Microphone:       StreamConfig{Enabled: true, DeviceID: "microphone:default"},
		NoiseSuppression: true,
	}, NewEnhancer(suppressor))
	if err != nil {
		t.Fatalf("NewPipeline() error = %v", err)
	}

	result, err := pipeline.Process(TimedPCMBuffer{
		Buffer: PCMBuffer{
			Kind:        StreamSystemAudio,
			SampleRate:  RNNoiseSampleRate,
			Channels:    2,
			Samples:     []float32{0.1, 0.2, 0.3, 0.4},
			Enhancement: EnhancementRNNoise,
		},
		Timestamp: 15 * time.Millisecond,
		Duration:  10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Process(system) error = %v", err)
	}
	if suppressor.frames != 0 {
		t.Fatalf("system audio entered RNNoise suppressor %d time(s), want 0", suppressor.frames)
	}
	if result.Buffer.Enhancement != EnhancementOff {
		t.Fatalf("system result enhancement = %q, want off", result.Buffer.Enhancement)
	}
	diagnostics := pipeline.Diagnostics()
	if diagnostics.SystemAudio.SamplesReceived != 4 || diagnostics.SystemAudio.SamplesWritten != 4 {
		t.Fatalf("system diagnostics = %#v, want samples received/written", diagnostics.SystemAudio)
	}
	if !diagnostics.Enhancement.SystemAudioBypassed {
		t.Fatal("system audio bypass policy was not recorded")
	}
}

func TestPipelineProcessesMicrophoneRNNoise(t *testing.T) {
	suppressor := &spySuppressor{}
	pipeline, err := NewPipeline(CaptureConfig{
		Backend:          "test-audio",
		Microphone:       StreamConfig{Enabled: true, DeviceID: "microphone:default"},
		NoiseSuppression: true,
	}, NewEnhancer(suppressor))
	if err != nil {
		t.Fatalf("NewPipeline() error = %v", err)
	}

	result, err := pipeline.Process(TimedPCMBuffer{
		Buffer: PCMBuffer{
			Kind:       StreamMicrophone,
			SampleRate: RNNoiseSampleRate,
			Channels:   1,
			Samples:    make([]float32, RNNoiseFrameSamples*2+17),
		},
		Timestamp: 0,
	})
	if err != nil {
		t.Fatalf("Process(microphone) error = %v", err)
	}
	if suppressor.frames != 2 {
		t.Fatalf("RNNoise frames = %d, want 2", suppressor.frames)
	}
	if len(result.Buffer.Samples) != RNNoiseFrameSamples*2 {
		t.Fatalf("processed samples = %d, want complete RNNoise frames only", len(result.Buffer.Samples))
	}
	diagnostics := pipeline.Diagnostics()
	if diagnostics.Enhancement.ProcessedFrames != 2 || diagnostics.Enhancement.PendingSamples != 17 {
		t.Fatalf("enhancement diagnostics = %#v", diagnostics.Enhancement)
	}
	if diagnostics.Microphone.SamplesReceived != int64(RNNoiseFrameSamples*2+17) {
		t.Fatalf("microphone samples received = %d", diagnostics.Microphone.SamplesReceived)
	}
	if diagnostics.Microphone.SamplesWritten != int64(RNNoiseFrameSamples*2) {
		t.Fatalf("microphone samples written = %d", diagnostics.Microphone.SamplesWritten)
	}
}

func TestPipelineRejectsDisabledStreams(t *testing.T) {
	pipeline, err := NewPipeline(CaptureConfig{}, nil)
	if err != nil {
		t.Fatalf("NewPipeline() error = %v", err)
	}

	_, err = pipeline.Process(TimedPCMBuffer{
		Buffer: PCMBuffer{
			Kind:       StreamMicrophone,
			SampleRate: RNNoiseSampleRate,
			Channels:   1,
			Samples:    make([]float32, RNNoiseFrameSamples),
		},
	})
	if err == nil {
		t.Fatal("Process() accepted microphone frame while microphone is disabled")
	}
}

func TestPipelineResetResetsEnhancerDiagnostics(t *testing.T) {
	pipeline, err := NewPipeline(CaptureConfig{
		Microphone:       StreamConfig{Enabled: true},
		NoiseSuppression: true,
	}, NewEnhancer(&spySuppressor{}))
	if err != nil {
		t.Fatalf("NewPipeline() error = %v", err)
	}
	if _, err := pipeline.Process(TimedPCMBuffer{
		Buffer: PCMBuffer{
			Kind:        StreamMicrophone,
			SampleRate:  RNNoiseSampleRate,
			Channels:    1,
			Samples:     make([]float32, 17),
			Enhancement: EnhancementRNNoise,
		},
	}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if err := pipeline.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	diagnostics := pipeline.Diagnostics()
	if diagnostics.Enhancement.ResetCount != 1 || diagnostics.Enhancement.PendingSamples != 0 {
		t.Fatalf("enhancement after reset = %#v", diagnostics.Enhancement)
	}
}

func TestWriteDiagnosticsWritesReadableJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recording.rfrec", audioDiagnosticsFilenameForTest())
	diagnostics := Diagnostics{
		Backend: "test-audio",
		TargetFormat: Format{
			SampleRate: RNNoiseSampleRate,
			Channels:   2,
			SampleType: "float32",
		},
		Enhancement: EnhancementDiagnostics{
			Engine:              "rnnoise",
			Enabled:             true,
			AppliedTo:           string(StreamMicrophone),
			SystemAudioBypassed: true,
		},
	}
	if err := WriteDiagnostics(path, diagnostics); err != nil {
		t.Fatalf("WriteDiagnostics() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var written Diagnostics
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("diagnostics JSON is invalid: %v", err)
	}
	if written.SchemaVersion != 1 || written.Backend != "test-audio" {
		t.Fatalf("written diagnostics = %#v", written)
	}
}

func audioDiagnosticsFilenameForTest() string {
	return "audio-diagnostics.json"
}
