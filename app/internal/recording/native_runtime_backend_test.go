package recording

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func TestNativeRuntimeBackendRunsMediaLifecycleAndReturnsSyncDiagnostics(t *testing.T) {
	packages := recpackage.NewService()
	videoSession := &fakeNativeVideoSession{
		diagnostics: video.Diagnostics{
			Screen: video.TrackDiagnostics{
				Enabled:       true,
				FrameRate:     30,
				FramesWritten: 30,
				EndOffsetMs:   1000,
				DurationMs:    1000,
			},
		},
	}
	audioSession := &fakeNativeAudioSession{
		diagnostics: audio.Diagnostics{
			SystemAudio: audio.StreamDiagnostics{
				Enabled:        true,
				SampleRate:     48000,
				SamplesWritten: 48000,
				EndOffsetMs:    1000,
				DurationMs:     1000,
			},
		},
	}
	backend := NewNativeRuntimeBackend(BackendScreenCaptureKit, packages, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(config video.CaptureConfig) (NativeVideoSession, error) {
			return videoSession, nil
		},
		AudioSessionFactory: func(audio.CaptureConfig, audio.NoiseSuppressor) (NativeAudioSession, error) {
			return audioSession, nil
		},
	})

	started, err := backend.Start(context.Background(), BackendStartRequest{
		VideoDir:  t.TempDir(),
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:display-1",
			SourceType: SourceScreen,
			Audio:      AudioRequest{System: true},
		},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(started.Package.Dir, recpackage.ScreenVideoFile), bytes.Repeat([]byte{1}, 64), 0o644); err != nil {
		t.Fatalf("WriteFile(screen) error = %v", err)
	}
	session := Session{ID: started.Package.ID}
	if err := backend.Pause(context.Background(), BackendControlRequest{Session: session}); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if err := backend.Resume(context.Background(), BackendControlRequest{Session: session}); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	stopped, err := backend.Stop(context.Background(), BackendControlRequest{Session: session})
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if videoSession.started != 1 || videoSession.paused != 1 || videoSession.resumed != 1 || videoSession.stopped != 1 {
		t.Fatalf("video lifecycle = start:%d pause:%d resume:%d stop:%d", videoSession.started, videoSession.paused, videoSession.resumed, videoSession.stopped)
	}
	if audioSession.started != 1 || audioSession.paused != 1 || audioSession.resumed != 1 || audioSession.stopped != 1 {
		t.Fatalf("audio lifecycle = start:%d pause:%d resume:%d stop:%d", audioSession.started, audioSession.paused, audioSession.resumed, audioSession.stopped)
	}
	if stopped.SyncDiagnostics == nil {
		t.Fatal("Stop() did not return sync diagnostics")
	}
	if stopped.SyncDiagnostics.Screen.Path != recpackage.ScreenVideoFile || stopped.SyncDiagnostics.SystemAudio.Path != recpackage.SystemAudioFile {
		t.Fatalf("sync diagnostics = %#v", stopped.SyncDiagnostics)
	}
	if _, err := backend.Stop(context.Background(), BackendControlRequest{Session: session}); err == nil {
		t.Fatal("second Stop() error = nil, want missing runtime")
	}
}

func TestNativeRuntimeBackendMarksPackageFailedWhenStartFails(t *testing.T) {
	packages := recpackage.NewService()
	videoDir := t.TempDir()
	backend := NewNativeRuntimeBackend(BackendScreenCaptureKit, packages, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(video.CaptureConfig) (NativeVideoSession, error) {
			return &fakeNativeVideoSession{startErr: errors.New("screen capture denied")}, nil
		},
	})

	_, err := backend.Start(context.Background(), BackendStartRequest{
		VideoDir:  videoDir,
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:display-1",
			SourceType: SourceScreen,
		},
	})
	if err == nil {
		t.Fatal("Start() error = nil, want capture failure")
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
		t.Fatalf("manifest status = %q, want failed", manifest.Status)
	}
}
