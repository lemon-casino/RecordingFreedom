package audio

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestWAVSinkWritesFloat32HeaderAndSamples(t *testing.T) {
	path := filepath.Join(t.TempDir(), "microphone.wav")
	sink, err := NewWAVSink("microphone", path)
	if err != nil {
		t.Fatalf("NewWAVSink() error = %v", err)
	}
	if err := sink.Append(ProcessedBuffer{Buffer: PCMBuffer{
		Kind:       StreamMicrophone,
		SampleRate: RNNoiseSampleRate,
		Channels:   1,
		Samples:    []float32{0.25, -0.5},
	}}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" || string(data[36:40]) != "data" {
		t.Fatalf("invalid wav header: %q %q %q", data[0:4], data[8:12], data[36:40])
	}
	if got := binary.LittleEndian.Uint16(data[20:22]); got != wavFormatIEEEFloat {
		t.Fatalf("wav format = %d, want IEEE float", got)
	}
	if got := binary.LittleEndian.Uint32(data[24:28]); got != RNNoiseSampleRate {
		t.Fatalf("sample rate = %d, want %d", got, RNNoiseSampleRate)
	}
	if got := binary.LittleEndian.Uint32(data[40:44]); got != 8 {
		t.Fatalf("data size = %d, want 8", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(data[44:48])); got != 0.25 {
		t.Fatalf("first sample = %f, want 0.25", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(data[48:52])); got != -0.5 {
		t.Fatalf("second sample = %f, want -0.5", got)
	}
}

func TestWAVSinkRejectsFormatChanges(t *testing.T) {
	sink, err := NewWAVSink("microphone", filepath.Join(t.TempDir(), "microphone.wav"))
	if err != nil {
		t.Fatalf("NewWAVSink() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sink.Close()
	})

	if err := sink.Append(ProcessedBuffer{Buffer: PCMBuffer{Kind: StreamMicrophone, SampleRate: 48000, Channels: 1, Samples: []float32{0}}}); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := sink.Append(ProcessedBuffer{Buffer: PCMBuffer{Kind: StreamMicrophone, SampleRate: 44100, Channels: 1, Samples: []float32{0}}}); err == nil {
		t.Fatal("Append() accepted a sample-rate change")
	}
}
