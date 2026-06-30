package audio

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCaptureSessionProcessesSourceIntoSinkAndDiagnostics(t *testing.T) {
	diagnosticsPath := filepath.Join(t.TempDir(), "audio-diagnostics.json")
	source := &scriptedSource{
		id:   "mic",
		kind: StreamMicrophone,
		frames: []TimedPCMBuffer{{
			Buffer: PCMBuffer{
				Kind:       StreamMicrophone,
				SampleRate: RNNoiseSampleRate,
				Channels:   1,
				Samples:    make([]float32, RNNoiseFrameSamples+9),
			},
			Timestamp: 20 * time.Millisecond,
		}},
	}
	sink := &memorySink{id: "microphone"}
	session, err := NewCaptureSession(CaptureConfig{
		Backend:          "test-audio",
		Microphone:       StreamConfig{Enabled: true, DeviceID: "microphone:default"},
		NoiseSuppression: true,
		DiagnosticsPath:  diagnosticsPath,
	}, NewEnhancer(&spySuppressor{}), []CaptureSource{source}, map[StreamKind]CaptureSink{
		StreamMicrophone: sink,
	})
	if err != nil {
		t.Fatalf("NewCaptureSession() error = %v", err)
	}

	if err := session.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(sink.buffers) != 1 {
		t.Fatalf("sink buffers = %d, want 1", len(sink.buffers))
	}
	if got := len(sink.buffers[0].Buffer.Samples); got != RNNoiseFrameSamples {
		t.Fatalf("processed samples = %d, want one RNNoise frame", got)
	}
	if err := session.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !source.stopped || !sink.closed {
		t.Fatalf("source stopped=%v sink closed=%v, want both true", source.stopped, sink.closed)
	}
	if _, err := os.Stat(diagnosticsPath); err != nil {
		t.Fatalf("diagnostics were not written: %v", err)
	}
}

func TestCaptureSessionStopBeforeStartClosesSinkAndWritesDiagnostics(t *testing.T) {
	diagnosticsPath := filepath.Join(t.TempDir(), "audio-diagnostics.json")
	source := &scriptedSource{id: "mic", kind: StreamMicrophone}
	sink := &memorySink{id: "microphone"}
	session, err := NewCaptureSession(CaptureConfig{
		Backend:         "test-audio",
		Microphone:      StreamConfig{Enabled: true, DeviceID: "microphone:default"},
		DiagnosticsPath: diagnosticsPath,
	}, nil, []CaptureSource{source}, map[StreamKind]CaptureSink{
		StreamMicrophone: sink,
	})
	if err != nil {
		t.Fatalf("NewCaptureSession() error = %v", err)
	}

	if err := session.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if source.stopped {
		t.Fatal("source was stopped even though it was never started")
	}
	if !sink.closed {
		t.Fatal("sink was not closed")
	}
	if _, err := os.Stat(diagnosticsPath); err != nil {
		t.Fatalf("diagnostics were not written: %v", err)
	}
}

func TestCaptureSessionPauseResetsRNNoiseState(t *testing.T) {
	source := &scriptedSource{
		id:   "mic",
		kind: StreamMicrophone,
		frames: []TimedPCMBuffer{{
			Buffer: PCMBuffer{
				Kind:       StreamMicrophone,
				SampleRate: RNNoiseSampleRate,
				Channels:   1,
				Samples:    make([]float32, RNNoiseFrameSamples+17),
			},
		}},
	}
	sink := &memorySink{id: "microphone"}
	suppressor := &spySuppressor{}
	session, err := NewCaptureSession(CaptureConfig{
		Backend:          "test-audio",
		Microphone:       StreamConfig{Enabled: true, DeviceID: "microphone:default"},
		NoiseSuppression: true,
		DiagnosticsPath:  filepath.Join(t.TempDir(), "audio-diagnostics.json"),
	}, NewEnhancer(suppressor), []CaptureSource{source}, map[StreamKind]CaptureSink{
		StreamMicrophone: sink,
	})
	if err != nil {
		t.Fatalf("NewCaptureSession() error = %v", err)
	}
	if err := session.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if pending := session.Diagnostics().Enhancement.PendingSamples; pending != 17 {
		t.Fatalf("pending before pause = %d, want 17", pending)
	}

	if err := session.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	diagnostics := session.Diagnostics()
	if !source.paused {
		t.Fatal("source was not paused")
	}
	if diagnostics.Enhancement.PendingSamples != 0 || diagnostics.Enhancement.ResetCount != 1 {
		t.Fatalf("enhancement after pause = %#v, want pending 0 and reset count 1", diagnostics.Enhancement)
	}
	if suppressor.resets != 1 {
		t.Fatalf("suppressor resets = %d, want 1", suppressor.resets)
	}
	if err := session.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

type scriptedSource struct {
	id      string
	kind    StreamKind
	frames  []TimedPCMBuffer
	paused  bool
	stopped bool
}

func (s *scriptedSource) ID() string {
	return s.id
}

func (s *scriptedSource) Kind() StreamKind {
	return s.kind
}

func (s *scriptedSource) Start(onFrame func(TimedPCMBuffer) error) error {
	for _, frame := range s.frames {
		if err := onFrame(frame); err != nil {
			return err
		}
	}
	return nil
}

func (s *scriptedSource) Pause() error {
	s.paused = true
	return nil
}

func (s *scriptedSource) Resume() error {
	s.paused = false
	return nil
}

func (s *scriptedSource) Stop() error {
	s.stopped = true
	return nil
}

type memorySink struct {
	id      string
	buffers []ProcessedBuffer
	closed  bool
}

func (s *memorySink) ID() string {
	return s.id
}

func (s *memorySink) Append(buffer ProcessedBuffer) error {
	s.buffers = append(s.buffers, buffer)
	return nil
}

func (s *memorySink) Close() error {
	s.closed = true
	return nil
}
