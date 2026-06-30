package recording

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestAudioOnlyRuntimeStartsPausesResumesAndStopsAudio(t *testing.T) {
	audioSession := &fakeNativeAudioSession{
		diagnostics: audio.Diagnostics{
			Backend: BackendAudioOnlyNative,
			Microphone: audio.StreamDiagnostics{
				Enabled:        true,
				SampleRate:     audio.RNNoiseSampleRate,
				SamplesWritten: 48000,
				EndOffsetMs:    1000,
				DurationMs:     1000,
			},
		},
	}
	var gotConfig audio.CaptureConfig
	runtime, err := NewAudioOnlyRuntime(recpackage.NewService(), BackendAudioOnlyNative, t.TempDir(), time.Now(), AudioOnlyRequest{
		Audio: AudioRequest{Microphone: true},
	}, AudioOnlyRuntimeOptions{
		AudioSessionFactory: func(config audio.CaptureConfig, suppressor audio.NoiseSuppressor) (NativeAudioSession, error) {
			gotConfig = config
			if suppressor != nil {
				t.Fatalf("suppressor = %#v, want nil when RNNoise is off", suppressor)
			}
			return audioSession, nil
		},
	})
	if err != nil {
		t.Fatalf("NewAudioOnlyRuntime() error = %v", err)
	}
	if filepath.Base(runtime.Plan.AudioOnlyPath) != recpackage.AudioOnlyWAVFile {
		t.Fatalf("audio-only path = %q, want audio.wav fallback", runtime.Plan.AudioOnlyPath)
	}
	if gotConfig.MicrophoneAudioPath != runtime.Plan.AudioOnlyPath || gotConfig.SystemAudioOutputPath != "" {
		t.Fatalf("audio config paths = mic:%q system:%q", gotConfig.MicrophoneAudioPath, gotConfig.SystemAudioOutputPath)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := runtime.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if err := runtime.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if audioSession.started != 1 || audioSession.paused != 1 || audioSession.resumed != 1 || audioSession.stopped != 1 {
		t.Fatalf("audio lifecycle = start:%d pause:%d resume:%d stop:%d", audioSession.started, audioSession.paused, audioSession.resumed, audioSession.stopped)
	}
	sync := runtime.SyncDiagnostics()
	if sync == nil || sync.Screen.Enabled || sync.Microphone.Path != recpackage.AudioOnlyWAVFile || sync.Microphone.SampleRate != audio.RNNoiseSampleRate {
		t.Fatalf("sync diagnostics = %#v", sync)
	}
}

func TestAudioOnlyRuntimePassesAndClosesRNNoiseSuppressor(t *testing.T) {
	suppressor := &fakeNoiseSuppressor{name: "fake-rnnoise"}
	audioSession := &fakeNativeAudioSession{}
	closed := false
	var gotSuppressor audio.NoiseSuppressor

	runtime, err := NewAudioOnlyRuntime(recpackage.NewService(), BackendAudioOnlyNative, t.TempDir(), time.Now(), AudioOnlyRequest{
		Audio: AudioRequest{Microphone: true, NoiseSuppression: true, MicrophoneGain: 1.75},
	}, AudioOnlyRuntimeOptions{
		NoiseSuppressorFactory: func(outputGain float64) (audio.NoiseSuppressor, func(), error) {
			if outputGain != 1.75 {
				t.Fatalf("outputGain = %v, want 1.75", outputGain)
			}
			return suppressor, func() { closed = true }, nil
		},
		AudioSessionFactory: func(_ audio.CaptureConfig, suppressor audio.NoiseSuppressor) (NativeAudioSession, error) {
			gotSuppressor = suppressor
			return audioSession, nil
		},
	})
	if err != nil {
		t.Fatalf("NewAudioOnlyRuntime() error = %v", err)
	}
	if gotSuppressor != suppressor {
		t.Fatalf("suppressor = %#v, want fake suppressor", gotSuppressor)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !closed {
		t.Fatal("RNNoise suppressor close callback was not called")
	}
}
