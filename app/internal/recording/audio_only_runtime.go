package recording

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type AudioOnlyRuntimeOptions struct {
	AudioSessionFactory    NativeAudioSessionFactory
	NoiseSuppressorFactory NativeNoiseSuppressorFactory
	PostStopProcessor      AudioOnlyPostStopProcessor
}

type AudioOnlyPostStopProcessor func(*AudioOnlyRuntime) error

type AudioOnlyRuntime struct {
	packages *recpackage.Service

	BackendID string
	Plan      recpackage.RecordingWritePlan
	Request   AudioOnlyRequest

	audioSession    NativeAudioSession
	closeSuppressor func()
	audioStarted    bool
	audioStopped    bool
}

func NewAudioOnlyRuntime(packages *recpackage.Service, backendID string, videoDir string, createdAt time.Time, req AudioOnlyRequest, options AudioOnlyRuntimeOptions) (*AudioOnlyRuntime, error) {
	backendID = strings.TrimSpace(backendID)
	if backendID == "" {
		backendID = BackendAudioOnlyNative
	}
	plan, normalized, err := CreateAudioOnlyWritePlan(packages, backendID, videoDir, createdAt, req)
	if err != nil {
		return nil, err
	}
	runtime := &AudioOnlyRuntime{
		packages:  packages,
		BackendID: backendID,
		Plan:      plan,
		Request:   normalized,
	}
	if err := runtime.prepareAudio(options); err != nil {
		return nil, errors.Join(err, runtime.MarkPackageFailed())
	}
	return runtime, nil
}

func (r *AudioOnlyRuntime) Start(ctx context.Context) error {
	if r == nil || r.audioSession == nil {
		return errors.New("audio-only runtime has no audio session")
	}
	if r.audioStarted {
		return errors.New("audio-only runtime is already started")
	}
	if r.audioStopped {
		return errors.New("audio-only runtime is stopped")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.audioSession.Start(ctx); err != nil {
		return errors.Join(fmt.Errorf("start audio-only capture: %w", err), r.Stop(), r.MarkPackageFailed())
	}
	r.audioStarted = true
	return nil
}

func (r *AudioOnlyRuntime) Pause() error {
	if r == nil || r.audioSession == nil || !r.audioStarted || r.audioStopped {
		return nil
	}
	return r.audioSession.Pause()
}

func (r *AudioOnlyRuntime) Resume() error {
	if r == nil || r.audioSession == nil || !r.audioStarted || r.audioStopped {
		return nil
	}
	return r.audioSession.Resume()
}

func (r *AudioOnlyRuntime) Stop() error {
	if r == nil || r.audioSession == nil || r.audioStopped {
		return nil
	}
	r.audioStopped = true
	err := r.audioSession.Stop()
	r.closeAudioSuppressor()
	return err
}

func (r *AudioOnlyRuntime) AudioDiagnostics() (audio.Diagnostics, bool) {
	if r == nil || r.audioSession == nil {
		return audio.Diagnostics{}, false
	}
	return r.audioSession.Diagnostics(), true
}

func (r *AudioOnlyRuntime) SyncDiagnostics() *recpackage.ManifestSyncDiagnostics {
	if r == nil {
		return nil
	}
	diagnostics := &recpackage.ManifestSyncDiagnostics{
		TimelineBase:         recpackage.TimelineBaseMedia,
		AudioDiagnosticsPath: recpackage.AudioDiagnosticsFile,
	}
	if audioDiagnostics, ok := r.AudioDiagnostics(); ok {
		diagnostics.SystemAudio = audioSyncTrack(audioDiagnostics.SystemAudio, r.Plan.Package.Manifest.Media.SystemAudioPath)
		diagnostics.Microphone = audioSyncTrack(audioDiagnostics.Microphone, r.Plan.Package.Manifest.Media.MicrophoneAudioPath)
	}
	return diagnostics
}

func (r *AudioOnlyRuntime) MarkPackageFailed() error {
	if r == nil || r.packages == nil || r.Plan.Package.ManifestPath == "" {
		return nil
	}
	return r.packages.PatchStatus(r.Plan.Package.ManifestPath, recpackage.StatusFailed, nil)
}

func (r *AudioOnlyRuntime) prepareAudio(options AudioOnlyRuntimeOptions) error {
	config, err := CreateAudioOnlyCaptureConfig(r.BackendID, r.Request, r.Plan)
	if err != nil {
		return err
	}
	var suppressor audio.NoiseSuppressor
	if config.NoiseSuppression {
		factory := options.NoiseSuppressorFactory
		if factory == nil {
			factory = defaultNativeNoiseSuppressorFactory
		}
		suppressor, r.closeSuppressor, err = factory(config.MicrophoneGain)
		if err != nil {
			r.closeAudioSuppressor()
			return fmt.Errorf("create RNNoise suppressor: %w", err)
		}
		if suppressor == nil {
			r.closeAudioSuppressor()
			return errors.New("create RNNoise suppressor: factory returned nil")
		}
	}

	sessionFactory := options.AudioSessionFactory
	if sessionFactory == nil {
		sessionFactory = defaultNativeAudioSessionFactory
	}
	session, err := sessionFactory(config, suppressor)
	if err != nil {
		r.closeAudioSuppressor()
		return err
	}
	if session == nil {
		r.closeAudioSuppressor()
		return errors.New("audio-only session factory returned nil")
	}
	r.audioSession = session
	return nil
}

func (r *AudioOnlyRuntime) closeAudioSuppressor() {
	if r.closeSuppressor == nil {
		return
	}
	r.closeSuppressor()
	r.closeSuppressor = nil
}
