package recording

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

type NativeAudioSession interface {
	Start(context.Context) error
	Pause() error
	Resume() error
	Stop() error
	Diagnostics() audio.Diagnostics
}

type NativeVideoSession interface {
	Start(context.Context) error
	Pause() error
	Resume() error
	Stop() error
	Diagnostics() video.Diagnostics
}

type NativeAudioSessionFactory func(audio.CaptureConfig, audio.NoiseSuppressor) (NativeAudioSession, error)
type NativeNoiseSuppressorFactory func(outputGain float64) (audio.NoiseSuppressor, func(), error)
type NativeVideoSessionFactory func(video.CaptureConfig) (NativeVideoSession, error)

type NativeBackendRuntimeOptions struct {
	AudioSessionFactory    NativeAudioSessionFactory
	NoiseSuppressorFactory NativeNoiseSuppressorFactory
	VideoSessionFactory    NativeVideoSessionFactory
}

type NativeBackendRuntime struct {
	packages *recpackage.Service

	BackendID string
	Plan      recpackage.RecordingWritePlan

	videoSession NativeVideoSession
	videoStarted bool
	videoStopped bool

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
	if err := runtime.prepareVideo(req.StartRequest, options); err != nil {
		return nil, errors.Join(err, runtime.markPackageFailed())
	}
	if err := runtime.prepareAudio(req.StartRequest, options); err != nil {
		return nil, errors.Join(err, runtime.markPackageFailed())
	}
	return runtime, nil
}

func (r *NativeBackendRuntime) Start(ctx context.Context) error {
	if err := r.StartVideo(ctx); err != nil {
		return err
	}
	if err := r.StartAudio(ctx); err != nil {
		return errors.Join(err, r.StopVideo())
	}
	return nil
}

func (r *NativeBackendRuntime) Pause() error {
	return errors.Join(r.PauseVideo(), r.PauseAudio())
}

func (r *NativeBackendRuntime) Resume() error {
	return errors.Join(r.ResumeVideo(), r.ResumeAudio())
}

func (r *NativeBackendRuntime) Stop() error {
	return errors.Join(r.StopAudio(), r.StopVideo())
}

func (r *NativeBackendRuntime) StartVideo(ctx context.Context) error {
	if r == nil || r.videoSession == nil {
		return nil
	}
	if r.videoStarted {
		return errors.New("native backend video is already started")
	}
	if r.videoStopped {
		return errors.New("native backend video is stopped")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.videoSession.Start(ctx); err != nil {
		return errors.Join(fmt.Errorf("start native backend video: %w", err), r.Stop(), r.markPackageFailed())
	}
	r.videoStarted = true
	return nil
}

func (r *NativeBackendRuntime) PauseVideo() error {
	if r == nil || r.videoSession == nil || !r.videoStarted || r.videoStopped {
		return nil
	}
	return r.videoSession.Pause()
}

func (r *NativeBackendRuntime) ResumeVideo() error {
	if r == nil || r.videoSession == nil || !r.videoStarted || r.videoStopped {
		return nil
	}
	return r.videoSession.Resume()
}

func (r *NativeBackendRuntime) StopVideo() error {
	if r == nil || r.videoSession == nil || r.videoStopped {
		return nil
	}
	r.videoStopped = true
	return r.videoSession.Stop()
}

func (r *NativeBackendRuntime) VideoDiagnostics() (video.Diagnostics, bool) {
	if r == nil || r.videoSession == nil {
		return video.Diagnostics{}, false
	}
	return r.videoSession.Diagnostics(), true
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

func (r *NativeBackendRuntime) SyncDiagnostics() *recpackage.ManifestSyncDiagnostics {
	if r == nil {
		return nil
	}
	diagnostics := &recpackage.ManifestSyncDiagnostics{
		TimelineBase:         recpackage.TimelineBaseMedia,
		VideoDiagnosticsPath: recpackage.VideoDiagnosticsFile,
	}
	if videoDiagnostics, ok := r.VideoDiagnostics(); ok {
		diagnostics.Screen = screenSyncTrack(videoDiagnostics, r.Plan.Package.Manifest.Media.ScreenVideoPath)
		if videoDiagnostics.SystemAudio.Enabled {
			diagnostics.SystemAudio = videoSystemAudioSyncTrack(videoDiagnostics.SystemAudio, r.Plan.Package.Manifest.Media.SystemAudioPath)
		}
	}
	if audioDiagnostics, ok := r.AudioDiagnostics(); ok {
		diagnostics.AudioDiagnosticsPath = recpackage.AudioDiagnosticsFile
		if !diagnostics.SystemAudio.Enabled {
			diagnostics.SystemAudio = audioSyncTrack(audioDiagnostics.SystemAudio, r.Plan.Package.Manifest.Media.SystemAudioPath)
		}
		diagnostics.Microphone = audioSyncTrack(audioDiagnostics.Microphone, r.Plan.Package.Manifest.Media.MicrophoneAudioPath)
	}
	return diagnostics
}

func (r *NativeBackendRuntime) MarkPackageFailed() error {
	if r == nil {
		return nil
	}
	return r.markPackageFailed()
}

func (r *NativeBackendRuntime) prepareVideo(req StartRequest, options NativeBackendRuntimeOptions) error {
	config, err := CreateVideoCaptureConfig(r.BackendID, req, r.Plan)
	if err != nil {
		return err
	}
	factory := options.VideoSessionFactory
	if factory == nil {
		factory = defaultNativeVideoSessionFactory
	}
	session, err := factory(config)
	if err != nil {
		return err
	}
	if session == nil {
		return errors.New("native video session factory returned nil")
	}
	r.videoSession = session
	return nil
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

func defaultNativeVideoSessionFactory(config video.CaptureConfig) (NativeVideoSession, error) {
	return video.NewPlatformSession(config)
}

func defaultNativeNoiseSuppressorFactory(outputGain float64) (audio.NoiseSuppressor, func(), error) {
	suppressor, err := rnnoise.New(outputGain)
	if err != nil {
		return nil, nil, err
	}
	return suppressor, suppressor.Close, nil
}

func screenSyncTrack(diagnostics video.Diagnostics, defaultPath string) recpackage.ManifestTrackDiagnostics {
	track := diagnostics.Screen
	if defaultPath == "" {
		defaultPath = recpackage.ScreenVideoFile
	}
	return recpackage.ManifestTrackDiagnostics{
		Enabled:        track.Enabled,
		Path:           defaultPath,
		Clock:          recpackage.TimelineBaseMedia,
		StartOffsetMs:  track.StartOffsetMs,
		EndOffsetMs:    track.EndOffsetMs,
		DurationMs:     track.DurationMs,
		DroppedFrames:  track.DroppedFrames,
		AppendFailures: track.AppendFailures,
		FrameRate:      track.FrameRate,
		Message:        track.Message,
	}
}

func audioSyncTrack(diagnostics audio.StreamDiagnostics, defaultPath string) recpackage.ManifestTrackDiagnostics {
	if !diagnostics.Enabled {
		return recpackage.ManifestTrackDiagnostics{}
	}
	return recpackage.ManifestTrackDiagnostics{
		Enabled:        true,
		Path:           defaultPath,
		Clock:          recpackage.TimelineBaseMedia,
		StartOffsetMs:  diagnostics.StartOffsetMs,
		EndOffsetMs:    diagnostics.EndOffsetMs,
		DurationMs:     diagnostics.DurationMs,
		DroppedSamples: diagnostics.DroppedSamples,
		AppendFailures: diagnostics.AppendFailures,
		SampleRate:     diagnostics.SampleRate,
		Message:        diagnostics.Message,
	}
}

func videoSystemAudioSyncTrack(track video.TrackDiagnostics, defaultPath string) recpackage.ManifestTrackDiagnostics {
	if !track.Enabled {
		return recpackage.ManifestTrackDiagnostics{}
	}
	if defaultPath == "" {
		defaultPath = recpackage.ScreenVideoFile
	}
	return recpackage.ManifestTrackDiagnostics{
		Enabled:        true,
		Path:           defaultPath,
		Clock:          recpackage.TimelineBaseMedia,
		StartOffsetMs:  track.StartOffsetMs,
		EndOffsetMs:    track.EndOffsetMs,
		DurationMs:     track.DurationMs,
		DroppedSamples: track.DroppedSamples,
		AppendFailures: track.AppendFailures,
		SampleRate:     track.SampleRate,
		Message:        track.Message,
	}
}
