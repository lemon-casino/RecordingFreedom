package recording

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type Service struct {
	appData          *appdata.Service
	packages         *recpackage.Service
	backend          Backend
	audioOnlyBackend *AudioOnlyRuntimeBackend
	mu               sync.Mutex
	state            State
	session          *Session
	pausedDuration   time.Duration
	pauseStartedAt   time.Time
}

type ServiceOptions struct {
	Backend                 Backend
	AudioOnlyRuntimeOptions AudioOnlyRuntimeOptions
}

func NewService(appData *appdata.Service) *Service {
	packages := recpackage.NewService()
	return newService(appData, packages, DefaultBackend(packages), AudioOnlyRuntimeOptions{})
}

func NewServiceWithBackend(appData *appdata.Service, backend Backend) *Service {
	packages := recpackage.NewService()
	if backend == nil {
		backend = NewMockBackend(packages)
	}
	return newService(appData, packages, backend, AudioOnlyRuntimeOptions{})
}

func NewServiceWithOptions(appData *appdata.Service, options ServiceOptions) *Service {
	packages := recpackage.NewService()
	backend := options.Backend
	if backend == nil {
		backend = DefaultBackend(packages)
	}
	return newService(appData, packages, backend, options.AudioOnlyRuntimeOptions)
}

func newService(appData *appdata.Service, packages *recpackage.Service, backend Backend, audioOnlyOptions AudioOnlyRuntimeOptions) *Service {
	if packages == nil {
		packages = recpackage.NewService()
	}
	if backend == nil {
		backend = DefaultBackend(packages)
	}
	return &Service{
		appData:          appData,
		packages:         packages,
		backend:          backend,
		audioOnlyBackend: NewAudioOnlyRuntimeBackend(packages, audioOnlyOptions),
		state:            StateIdle,
	}
}

func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Service) BackendID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.ID()
}

func (s *Service) ActiveBackendID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil && s.session.Backend != "" {
		return s.session.Backend
	}
	return s.backend.ID()
}

func (s *Service) ActiveSession() (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil {
		return Session{}, false
	}
	switch s.state {
	case StatePreparing, StateRecording, StatePaused, StateStopping:
		return *s.session, true
	default:
		return Session{}, false
	}
}

func (s *Service) ActiveRecordingOffset(now time.Time) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session == nil || s.session.StartedAt.IsZero() {
		return 0, false
	}
	switch s.state {
	case StatePreparing, StateRecording, StatePaused, StateStopping:
	default:
		return 0, false
	}
	if now.IsZero() {
		now = time.Now()
	}
	elapsed := now.Sub(s.session.StartedAt) - s.pausedDuration
	if s.state == StatePaused && !s.pauseStartedAt.IsZero() {
		elapsed -= now.Sub(s.pauseStartedAt)
	}
	if elapsed < 0 {
		elapsed = 0
	}
	return elapsed.Milliseconds(), true
}

func (s *Service) PatchActiveCameraPIP(config pip.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return nil
	}
	if s.session.RecordingMode != recpackage.RecordingModeScreen {
		return nil
	}
	if s.state != StateRecording && s.state != StatePaused && s.state != StatePreparing {
		return nil
	}
	if s.session.Manifest == "" {
		return nil
	}
	manifest, err := s.packages.ReadManifest(s.session.Manifest)
	if err != nil {
		return err
	}
	if !manifest.Camera.Enabled {
		return nil
	}
	_, err = s.packages.PatchCameraPIP(s.session.Manifest, config)
	return err
}

func (s *Service) ScanPackages() ([]recpackage.RecoverySummary, error) {
	videoDir, err := s.appData.VideoDir()
	if err != nil {
		return nil, err
	}
	return s.packages.Scan(videoDir)
}

func (s *Service) RecoverPackage(packageDir string) (recpackage.RecoverySummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRecording || s.state == StatePaused || s.state == StateStopping || s.state == StatePreparing {
		return recpackage.RecoverySummary{}, errors.New("cannot recover packages while recording is active")
	}
	videoDir, err := s.appData.VideoDir()
	if err != nil {
		return recpackage.RecoverySummary{}, err
	}
	return s.packages.Recover(videoDir, packageDir, time.Now())
}

func (s *Service) StartMockRecording(req StartRequest) (Session, error) {
	return s.StartRecording(req)
}

func (s *Service) StartRecording(req StartRequest) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRecording || s.state == StatePaused || s.state == StateStopping || s.state == StatePreparing {
		return Session{}, errors.New("recording is already active")
	}
	req, err := NormalizeStartRequest(req)
	if err != nil {
		return Session{}, err
	}

	videoDir, err := s.appData.VideoDir()
	if err != nil {
		s.state = StateFailed
		return Session{}, err
	}

	now := time.Now()
	s.state = StatePreparing
	s.pausedDuration = 0
	s.pauseStartedAt = time.Time{}
	result, err := s.backend.Start(context.Background(), BackendStartRequest{
		StartRequest: req,
		VideoDir:     videoDir,
		CreatedAt:    now,
	})
	if err != nil {
		s.state = StateFailed
		return Session{}, err
	}

	session := Session{
		ID:            result.Package.ID,
		PackageDir:    result.Package.Dir,
		Manifest:      result.Package.ManifestPath,
		Backend:       s.backend.ID(),
		RecordingMode: recpackage.RecordingModeScreen,
		Status:        StateRecording,
		StartedAt:     now,
	}
	s.state = StateRecording
	s.session = &session
	return session, nil
}

func (s *Service) StartAudioOnlyRecording(req AudioOnlyRequest) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRecording || s.state == StatePaused || s.state == StateStopping || s.state == StatePreparing {
		return Session{}, errors.New("recording is already active")
	}
	req, err := NormalizeAudioOnlyRequest(req)
	if err != nil {
		return Session{}, err
	}

	videoDir, err := s.appData.VideoDir()
	if err != nil {
		s.state = StateFailed
		return Session{}, err
	}
	if s.audioOnlyBackend == nil {
		s.audioOnlyBackend = NewAudioOnlyRuntimeBackend(s.packages, AudioOnlyRuntimeOptions{})
	}

	now := time.Now()
	s.state = StatePreparing
	s.pausedDuration = 0
	s.pauseStartedAt = time.Time{}
	result, err := s.audioOnlyBackend.Start(context.Background(), videoDir, now, req)
	if err != nil {
		s.state = StateFailed
		return Session{}, err
	}

	session := Session{
		ID:            result.Package.ID,
		PackageDir:    result.Package.Dir,
		Manifest:      result.Package.ManifestPath,
		Backend:       s.audioOnlyBackend.ID(),
		RecordingMode: recpackage.RecordingModeAudio,
		Status:        StateRecording,
		StartedAt:     now,
	}
	s.state = StateRecording
	s.session = &session
	return session, nil
}

func (s *Service) Pause() (Session, error) {
	return s.setActiveState(StateRecording, StatePaused)
}

func (s *Service) Resume() (Session, error) {
	return s.setActiveState(StatePaused, StateRecording)
}

func (s *Service) Stop() (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return Session{}, errors.New("no active recording")
	}
	if s.state != StateRecording && s.state != StatePaused {
		return Session{}, fmt.Errorf("cannot stop recording from state %q", s.state)
	}

	completed := time.Now()
	s.state = StateStopping
	s.session.Status = StateStopping
	if err := s.packages.PatchStatus(s.session.Manifest, recpackage.StatusFinalizing, nil); err != nil {
		s.state = StateFailed
		s.session.Status = StateFailed
		return *s.session, err
	}
	stopResult, err := s.stopActiveBackend()
	if err != nil {
		s.state = StateFailed
		s.session.Status = StateFailed
		_ = s.packages.PatchStatus(s.session.Manifest, recpackage.StatusFailed, nil)
		return *s.session, err
	}
	if stopResult.SyncDiagnostics != nil {
		if err := s.packages.PatchSyncDiagnostics(s.session.Manifest, *stopResult.SyncDiagnostics); err != nil {
			s.state = StateFailed
			s.session.Status = StateFailed
			_ = s.packages.PatchStatus(s.session.Manifest, recpackage.StatusFailed, nil)
			return *s.session, err
		}
	}
	if err := s.packages.ValidateReady(s.session.Manifest); err != nil {
		s.state = StateFailed
		s.session.Status = StateFailed
		_ = s.packages.PatchStatus(s.session.Manifest, recpackage.StatusFailed, nil)
		return *s.session, err
	}

	s.state = StateReady
	s.session.Status = StateReady
	s.session.CompletedAt = completed
	if err := s.packages.PatchStatus(s.session.Manifest, recpackage.StatusReady, &completed); err != nil {
		s.state = StateFailed
		s.session.Status = StateFailed
		return *s.session, err
	}
	return *s.session, nil
}

func (s *Service) setActiveState(from State, to State) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return Session{}, errors.New("no active recording")
	}
	if s.state != from {
		return Session{}, fmt.Errorf("cannot transition from %q to %q", s.state, to)
	}
	if err := s.applyBackendTransition(to); err != nil {
		s.state = StateFailed
		s.session.Status = StateFailed
		_ = s.packages.PatchStatus(s.session.Manifest, recpackage.StatusFailed, nil)
		return *s.session, err
	}
	transitionedAt := time.Now()
	s.state = to
	s.session.Status = to
	switch to {
	case StatePaused:
		s.pauseStartedAt = transitionedAt
	case StateRecording:
		if !s.pauseStartedAt.IsZero() {
			s.pausedDuration += transitionedAt.Sub(s.pauseStartedAt)
			s.pauseStartedAt = time.Time{}
		}
	}
	if err := s.packages.PatchStatus(s.session.Manifest, recpackageStatus(to), nil); err != nil {
		return Session{}, err
	}
	return *s.session, nil
}

func (s *Service) applyBackendTransition(to State) error {
	switch to {
	case StatePaused:
		if s.session != nil && s.session.RecordingMode == recpackage.RecordingModeAudio {
			return s.audioOnlyBackend.Pause(context.Background(), BackendControlRequest{Session: *s.session})
		}
		return s.backend.Pause(context.Background(), BackendControlRequest{Session: *s.session})
	case StateRecording:
		if s.session != nil && s.session.RecordingMode == recpackage.RecordingModeAudio {
			return s.audioOnlyBackend.Resume(context.Background(), BackendControlRequest{Session: *s.session})
		}
		return s.backend.Resume(context.Background(), BackendControlRequest{Session: *s.session})
	default:
		return nil
	}
}

func (s *Service) stopActiveBackend() (BackendStopResult, error) {
	if s.session != nil && s.session.RecordingMode == recpackage.RecordingModeAudio {
		return s.audioOnlyBackend.Stop(context.Background(), BackendControlRequest{Session: *s.session})
	}
	return s.backend.Stop(context.Background(), BackendControlRequest{Session: *s.session})
}

func noiseSuppressionLabel(enabled bool) string {
	if enabled {
		return recpackage.NoiseSuppressionOn
	}
	return recpackage.NoiseSuppressionOff
}

func recpackageStatus(state State) string {
	switch state {
	case StateRecording:
		return recpackage.StatusRecording
	case StatePaused:
		return recpackage.StatusPaused
	case StateReady:
		return recpackage.StatusReady
	case StateFailed:
		return recpackage.StatusFailed
	default:
		return string(state)
	}
}
