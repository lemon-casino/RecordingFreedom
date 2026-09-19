package audio

import (
	"bufio"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
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

// TestWAVSinkBuffersSmallAppendsUntilClose keeps every batch under the write
// buffer size so Close's header rewrite runs while PCM is still buffered. It
// catches the regressions where the header rewrite seeks to 0 before the
// buffer flushes and later clobbers the file header with PCM data.
func TestWAVSinkBuffersSmallAppendsUntilClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "microphone.wav")
	sink, err := NewWAVSink("microphone", path)
	if err != nil {
		t.Fatalf("NewWAVSink() error = %v", err)
	}

	var want []float32
	for batch := 0; batch < 40; batch++ {
		samples := make([]float32, 1000)
		for index := range samples {
			samples[index] = float32((batch*1000+index)%256)/128 - 1
		}
		want = append(want, samples...)
		if err := sink.Append(ProcessedBuffer{Buffer: PCMBuffer{Kind: StreamMicrophone, SampleRate: RNNoiseSampleRate, Channels: 1, Samples: samples}}); err != nil {
			t.Fatalf("Append(batch %d) error = %v", batch, err)
		}
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	dataBytes := binary.LittleEndian.Uint32(data[40:44])
	if int(dataBytes) != len(want)*wavFloat32Bytes {
		t.Fatalf("data size = %d, want %d", dataBytes, len(want)*wavFloat32Bytes)
	}
	if len(data) != wavHeaderSize+int(dataBytes) {
		t.Fatalf("file size = %d, want header %d + data %d", len(data), wavHeaderSize, dataBytes)
	}
	got := make([]float32, len(want))
	for index := range got {
		got[index] = math.Float32frombits(binary.LittleEndian.Uint32(data[wavHeaderSize+index*wavFloat32Bytes:]))
	}
	if !slices.Equal(got, want) {
		t.Fatal("decoded samples differ from appended samples; buffered PCM was lost or clobbered")
	}
}

// failingWAVWriter always fails, standing in for a dead disk handle.
type failingWAVWriter struct {
	err error
}

func (w *failingWAVWriter) Write([]byte) (int, error) {
	return 0, w.err
}

// TestWAVSinkSurfacesWriteErrorsThroughClose verifies a write failure is
// sticky in the buffered writer: the batch that forces a flush reports it,
// later appends keep reporting it, and Close's header flush returns it so the
// error reaches the session stop chain.
func TestWAVSinkSurfacesWriteErrorsThroughClose(t *testing.T) {
	sink, err := NewWAVSink("microphone", filepath.Join(t.TempDir(), "microphone.wav"))
	if err != nil {
		t.Fatalf("NewWAVSink() error = %v", err)
	}
	cause := errors.New("disk unavailable")
	sink.writer = bufio.NewWriterSize(&failingWAVWriter{err: cause}, wavWriteBufferSize)

	oversized := make([]float32, wavWriteBufferSize/wavFloat32Bytes+1)
	if err := sink.Append(ProcessedBuffer{Buffer: PCMBuffer{Kind: StreamMicrophone, SampleRate: RNNoiseSampleRate, Channels: 1, Samples: oversized}}); err == nil {
		t.Fatal("Append() error = nil, want the batch that forces a flush to report the writer failure")
	}
	if err := sink.Append(ProcessedBuffer{Buffer: PCMBuffer{Kind: StreamMicrophone, SampleRate: RNNoiseSampleRate, Channels: 1, Samples: []float32{0.5}}}); err == nil {
		t.Fatal("second Append() error = nil, want sticky buffered-writer failure")
	}
	if err := sink.Close(); err == nil {
		t.Fatal("Close() error = nil, want header rewrite to surface the flush failure")
	}
}
