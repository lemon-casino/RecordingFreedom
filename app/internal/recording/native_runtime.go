package recording

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type NativeAudioSession interface {
	Start(context.Context) error
	Pause() error
	Resume() error
	Stop() error
	Diagnostics() audio.Diagnostics
}

type NativeAudioSessionFactory func(audio.CaptureConfig, audio.NoiseSuppressor) (NativeAudioSession, error)
type NativeNoiseSuppressorFactory func(outputGain float64) (audio.NoiseSuppressor, func(), error)

type NativeBackendRuntimeOptions struct {
	AudioSessionFactory    NativeAudioSessionFactory
	NoiseSuppressorFactory NativeNoiseSuppressorFactory
}

type NativeBackendRuntime struct {
	packages *recpackage.Service

	BackendID string
	Plan      recpackage.RecordingWritePlan

	audioSession    NativeAudioSession
	closeSuppressor func()
	audioStarted    bool
	audioStopped    bool
}

func NewNativeBackendRuntime(packages *recpackage.Service, backendID string, req BackendStartRequest, options NativeBackendRuntimeOptions) (*NativeBackendRuntime, error) {
	backendID = strings.TrimSpace(backendID)
	if backendID == "" {
		backendID = BackendNativeUnsupported
	}
	plan, err := CreateNativeWritePlan(packages, backendID, req)
	if err != nil {
		return nil, err
	}
	runtime := &NativeBackendRuntime{
		packages:  packages,
		BackendID: backendID,
		Plan:      plan,
	}
	if err := runtime.prepareAudio(req.StartRequest, options); err != nil {
		return nil, errors.Join(err, runtime.markPackageFailed())
	}
	return runtime, nil
}

func (r *NativeBackendRuntime) StartAudio(ctx context.Context) error {
	if r == nil || r.audioSession == nil {
		return nil
	}
	if r.audioStarted {
		return errors.New("native backend audio is already started")
	}
	if r.audioStopped {
		return errors.New("native backend audio is stopped")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.audioSession.Start(ctx); err != nil {
		return errors.Join(fmt.Errorf("start native backend audio: %w", err), r.StopAudio(), r.markPackageFailed())
	}
	r.audioStarted = true
	return nil
}

func (r *NativeBackendRuntime) PauseAudio() error {
	if r == nil || r.audioSession == nil || !r.audioStarted || r.audioStopped {
		return nil
	}
	return r.audioSession.Pause()
}

func (r *NativeBackendRuntime) ResumeAudio() error {
	if r == nil || r.audioSession == nil || !r.audioStarted || r.audioStopped {
		return nil
	}
	return r.audioSession.Resume()
}

func (r *NativeBackendRuntime) StopAudio() error {
	if r == nil || r.audioSession == nil || r.audioStopped {
		return nil
	}
	r.audioStopped = true
	err := r.audioSession.Stop()
	r.closeAudioSuppressor()
	return err
}

func (r *NativeBackendRuntime) AudioDiagnostics() (audio.Diagnostics, bool) {
	if r == nil || r.audioSession == nil {
		return audio.Diagnostics{}, false
	}
	return r.audioSession.Diagnostics(), true
}

func (r *NativeBackendRuntime) MarkPackageFailed() error {
	if r == nil {
		return nil
	}
	return r.markPackageFailed()
}

func (r *NativeBackendRuntime) prepareAudio(req StartRequest, options NativeBackendRuntimeOptions) error {
	config, err := CreateAudioCaptureConfig(r.BackendID, req, r.Plan)
	if err != nil {
		return err
	}
	if !config.SystemAudio.Enabled && !config.Microphone.Enabled {
		return nil
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
		return errors.New("native audio session factory returned nil")
	}
	r.audioSession = session
	return nil
}

func (r *NativeBackendRuntime) closeAudioSuppressor() {
	if r.closeSuppressor == nil {
		return
	}
	r.closeSuppressor()
	r.closeSuppressor = nil
}

func (r *NativeBackendRuntime) markPackageFailed() error {
	if r == nil || r.packages == nil || r.Plan.Package.ManifestPath == "" {
		return nil
	}
	return r.packages.PatchStatus(r.Plan.Package.ManifestPath, recpackage.StatusFailed, nil)
}

func defaultNativeAudioSessionFactory(config audio.CaptureConfig, suppressor audio.NoiseSuppressor) (NativeAudioSession, error) {
	return audio.NewNativeCaptureSession(config, suppressor)
}

func defaultNativeNoiseSuppressorFactory(outputGain float64) (audio.NoiseSuppressor, func(), error) {
	suppressor, err := rnnoise.New(outputGain)
	if err != nil {
		return nil, nil, err
	}
	return suppressor, suppressor.Close, nil
}
