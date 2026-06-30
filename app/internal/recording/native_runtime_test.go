package recording

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func TestNativeBackendRuntimeStartsPausesResumesAndStopsMedia(t *testing.T) {
	packages := recpackage.NewService()
	videoDir := t.TempDir()
	audioSession := &fakeNativeAudioSession{
		diagnostics: audio.Diagnostics{Backend: BackendWindowsGraphicsCapture},
	}
	videoSession := &fakeNativeVideoSession{
		diagnostics: video.Diagnostics{Backend: BackendWindowsGraphicsCapture},
	}
	var gotAudioConfig audio.CaptureConfig
	var gotVideoConfig video.CaptureConfig
	runtime, err := NewNativeBackendRuntime(packages, BackendWindowsGraphicsCapture, BackendStartRequest{
		VideoDir:  videoDir,
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:primary",
			SourceType: SourceScreen,
			Audio: AudioRequest{
				System:         true,
				SystemDeviceID: "system-audio:default",
				Microphone:     true,
				MicrophoneID:   "microphone:default",
			},
		},
	}, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(config video.CaptureConfig) (NativeVideoSession, error) {
			gotVideoConfig = config
			return videoSession, nil
		},
		AudioSessionFactory: func(config audio.CaptureConfig, suppressor audio.NoiseSuppressor) (NativeAudioSession, error) {
			gotAudioConfig = config
			if suppressor != nil {
				t.Fatalf("suppressor = %#v, want nil when RNNoise is off", suppressor)
			}
			return audioSession, nil
		},
	})
	if err != nil {
		t.Fatalf("NewNativeBackendRuntime() error = %v", err)
	}

	if filepath.Dir(runtime.Plan.Package.Dir) != videoDir {
		t.Fatalf("package parent = %q, want %q", filepath.Dir(runtime.Plan.Package.Dir), videoDir)
	}
	if gotVideoConfig.Backend != BackendWindowsGraphicsCapture {
		t.Fatalf("video backend = %q, want %q", gotVideoConfig.Backend, BackendWindowsGraphicsCapture)
	}
	if gotVideoConfig.OutputPath != filepath.Join(runtime.Plan.Package.Dir, recpackage.ScreenVideoFile) {
		t.Fatalf("video output path = %q, want package screen media", gotVideoConfig.OutputPath)
	}
	if gotVideoConfig.DiagnosticsPath != filepath.Join(runtime.Plan.Package.Dir, recpackage.VideoDiagnosticsFile) {
		t.Fatalf("video diagnostics path = %q, want package diagnostics", gotVideoConfig.DiagnosticsPath)
	}
	if gotAudioConfig.Backend != BackendWindowsGraphicsCapture {
		t.Fatalf("audio backend = %q, want %q", gotAudioConfig.Backend, BackendWindowsGraphicsCapture)
	}
	if gotAudioConfig.SystemAudioOutputPath != filepath.Join(runtime.Plan.Package.Dir, recpackage.SystemAudioFile) {
		t.Fatalf("system audio path = %q, want package sidecar", gotAudioConfig.SystemAudioOutputPath)
	}
	if gotAudioConfig.MicrophoneAudioPath != filepath.Join(runtime.Plan.Package.Dir, recpackage.MicrophoneAudioFile) {
		t.Fatalf("microphone path = %q, want package sidecar", gotAudioConfig.MicrophoneAudioPath)
	}
	if gotAudioConfig.DiagnosticsPath != filepath.Join(runtime.Plan.Package.Dir, recpackage.AudioDiagnosticsFile) {
		t.Fatalf("audio diagnostics path = %q, want package diagnostics", gotAudioConfig.DiagnosticsPath)
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
	if videoSession.started != 1 || videoSession.paused != 1 || videoSession.resumed != 1 || videoSession.stopped != 1 {
		t.Fatalf("video controls = start:%d pause:%d resume:%d stop:%d, want 1 each", videoSession.started, videoSession.paused, videoSession.resumed, videoSession.stopped)
	}
	if audioSession.started != 1 || audioSession.paused != 1 || audioSession.resumed != 1 || audioSession.stopped != 1 {
		t.Fatalf("audio controls = start:%d pause:%d resume:%d stop:%d, want 1 each", audioSession.started, audioSession.paused, audioSession.resumed, audioSession.stopped)
	}
	if diagnostics, ok := runtime.VideoDiagnostics(); !ok || diagnostics.Backend != BackendWindowsGraphicsCapture {
		t.Fatalf("VideoDiagnostics() = (%#v, %v), want backend diagnostics", diagnostics, ok)
	}
	if diagnostics, ok := runtime.AudioDiagnostics(); !ok || diagnostics.Backend != BackendWindowsGraphicsCapture {
		t.Fatalf("AudioDiagnostics() = (%#v, %v), want backend diagnostics", diagnostics, ok)
	}
}

func TestNativeBackendRuntimeRequiresAvailableRNNoise(t *testing.T) {
	packages := recpackage.NewService()
	videoDir := t.TempDir()
	_, err := NewNativeBackendRuntime(packages, BackendWindowsGraphicsCapture, BackendStartRequest{
		VideoDir:  videoDir,
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:primary",
			SourceType: SourceScreen,
			Audio: AudioRequest{
				Microphone:       true,
				MicrophoneID:     "microphone:default",
				NoiseSuppression: true,
			},
		},
	}, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(video.CaptureConfig) (NativeVideoSession, error) {
			return &fakeNativeVideoSession{}, nil
		},
		NoiseSuppressorFactory: func(float64) (audio.NoiseSuppressor, func(), error) {
			return nil, nil, errors.New("rnnoise native unavailable")
		},
		AudioSessionFactory: func(audio.CaptureConfig, audio.NoiseSuppressor) (NativeAudioSession, error) {
			t.Fatal("audio session factory was called even though RNNoise was unavailable")
			return nil, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "rnnoise native unavailable") {
		t.Fatalf("NewNativeBackendRuntime() error = %v, want RNNoise unavailable error", err)
	}

	manifestPaths, globErr := filepath.Glob(filepath.Join(videoDir, "*"+recpackage.PackageDirSuffix, recpackage.ManifestFile))
	if globErr != nil {
		t.Fatalf("Glob() error = %v", globErr)
	}
	if len(manifestPaths) != 1 {
		t.Fatalf("manifest count = %d, want 1", len(manifestPaths))
	}
	manifest, readErr := packages.ReadManifest(manifestPaths[0])
	if readErr != nil {
		t.Fatalf("ReadManifest() error = %v", readErr)
	}
	if manifest.Status != recpackage.StatusFailed {
		t.Fatalf("manifest status = %q, want failed after RNNoise setup failure", manifest.Status)
	}
}

func TestNativeBackendRuntimePassesAndClosesRNNoiseSuppressor(t *testing.T) {
	packages := recpackage.NewService()
	suppressor := &fakeNoiseSuppressor{name: "fake-rnnoise"}
	audioSession := &fakeNativeAudioSession{}
	closed := false
	var gotSuppressor audio.NoiseSuppressor

	runtime, err := NewNativeBackendRuntime(packages, BackendWindowsGraphicsCapture, BackendStartRequest{
		VideoDir:  t.TempDir(),
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:primary",
			SourceType: SourceScreen,
			Audio: AudioRequest{
				Microphone:       true,
				MicrophoneID:     "microphone:default",
				NoiseSuppression: true,
				MicrophoneGain:   1.5,
			},
		},
	}, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(video.CaptureConfig) (NativeVideoSession, error) {
			return &fakeNativeVideoSession{}, nil
		},
		NoiseSuppressorFactory: func(outputGain float64) (audio.NoiseSuppressor, func(), error) {
			if outputGain != 1.5 {
				t.Fatalf("outputGain = %v, want 1.5", outputGain)
			}
			return suppressor, func() { closed = true }, nil
		},
		AudioSessionFactory: func(_ audio.CaptureConfig, suppressor audio.NoiseSuppressor) (NativeAudioSession, error) {
			gotSuppressor = suppressor
			return audioSession, nil
		},
	})
	if err != nil {
		t.Fatalf("NewNativeBackendRuntime() error = %v", err)
	}
	if gotSuppressor != suppressor {
		t.Fatalf("suppressor = %#v, want fake suppressor", gotSuppressor)
	}
	if err := runtime.StartAudio(context.Background()); err != nil {
		t.Fatalf("StartAudio() error = %v", err)
	}
	if err := runtime.StopAudio(); err != nil {
		t.Fatalf("StopAudio() error = %v", err)
	}
	if !closed {
		t.Fatal("RNNoise suppressor close callback was not called")
	}
	if err := runtime.StopAudio(); err != nil {
		t.Fatalf("second StopAudio() error = %v", err)
	}
}

func TestNativeBackendRuntimeSkipsAudioWhenNoStreamsEnabled(t *testing.T) {
	runtime, err := NewNativeBackendRuntime(recpackage.NewService(), BackendScreenCaptureKit, BackendStartRequest{
		VideoDir:  t.TempDir(),
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:primary",
			SourceType: SourceScreen,
		},
	}, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(video.CaptureConfig) (NativeVideoSession, error) {
			return &fakeNativeVideoSession{}, nil
		},
		AudioSessionFactory: func(audio.CaptureConfig, audio.NoiseSuppressor) (NativeAudioSession, error) {
			t.Fatal("audio session factory was called with all audio streams disabled")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("NewNativeBackendRuntime() error = %v", err)
	}
	if err := runtime.StartAudio(context.Background()); err != nil {
		t.Fatalf("StartAudio() error = %v", err)
	}
	if err := runtime.StopAudio(); err != nil {
		t.Fatalf("StopAudio() error = %v", err)
	}
	if _, ok := runtime.AudioDiagnostics(); ok {
		t.Fatal("AudioDiagnostics() reported diagnostics with no audio session")
	}
}

func TestNativeBackendRuntimeMarksPackageFailedWhenVideoSessionFails(t *testing.T) {
	packages := recpackage.NewService()
	videoDir := t.TempDir()
	_, err := NewNativeBackendRuntime(packages, BackendScreenCaptureKit, BackendStartRequest{
		VideoDir:  videoDir,
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:display-1",
			SourceType: SourceScreen,
		},
	}, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(video.CaptureConfig) (NativeVideoSession, error) {
			return nil, errors.New("screencapturekit unavailable")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "screencapturekit unavailable") {
		t.Fatalf("NewNativeBackendRuntime() error = %v, want video setup error", err)
	}

	manifestPaths, globErr := filepath.Glob(filepath.Join(videoDir, "*"+recpackage.PackageDirSuffix, recpackage.ManifestFile))
	if globErr != nil {
		t.Fatalf("Glob() error = %v", globErr)
	}
	if len(manifestPaths) != 1 {
		t.Fatalf("manifest count = %d, want 1", len(manifestPaths))
	}
	manifest, readErr := packages.ReadManifest(manifestPaths[0])
	if readErr != nil {
		t.Fatalf("ReadManifest() error = %v", readErr)
	}
	if manifest.Status != recpackage.StatusFailed {
		t.Fatalf("manifest status = %q, want failed after video setup failure", manifest.Status)
	}
}

type fakeNativeAudioSession struct {
	startErr    error
	started     int
	paused      int
	resumed     int
	stopped     int
	diagnostics audio.Diagnostics
}

func (s *fakeNativeAudioSession) Start(context.Context) error {
	s.started++
	return s.startErr
}

func (s *fakeNativeAudioSession) Pause() error {
	s.paused++
	return nil
}

func (s *fakeNativeAudioSession) Resume() error {
	s.resumed++
	return nil
}

func (s *fakeNativeAudioSession) Stop() error {
	s.stopped++
	return nil
}

func (s *fakeNativeAudioSession) Diagnostics() audio.Diagnostics {
	return s.diagnostics
}

type fakeNativeVideoSession struct {
	startErr    error
	started     int
	paused      int
	resumed     int
	stopped     int
	diagnostics video.Diagnostics
}

func (s *fakeNativeVideoSession) Start(context.Context) error {
	s.started++
	return s.startErr
}

func (s *fakeNativeVideoSession) Pause() error {
	s.paused++
	return nil
}

func (s *fakeNativeVideoSession) Resume() error {
	s.resumed++
	return nil
}

func (s *fakeNativeVideoSession) Stop() error {
	s.stopped++
	return nil
}

func (s *fakeNativeVideoSession) Diagnostics() video.Diagnostics {
	return s.diagnostics
}

type fakeNoiseSuppressor struct {
	name string
}

func (s *fakeNoiseSuppressor) Name() string {
	return s.name
}

func (s *fakeNoiseSuppressor) ProcessFrame([]float32) error {
	return nil
}

func (s *fakeNoiseSuppressor) Reset() error {
	return nil
}
