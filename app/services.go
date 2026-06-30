package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/preflight"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type BootstrapState struct {
	AppData      appdata.Info                 `json:"appData"`
	Storage      appdata.StorageStatus        `json:"storage"`
	State        recording.State              `json:"state"`
	Backend      string                       `json:"backend"`
	Sources      []devices.CaptureSource      `json:"sources"`
	Media        devices.MediaInventory       `json:"media"`
	Recoveries   []recpackage.RecoverySummary `json:"recoveries"`
	Settings     settings.Settings            `json:"settings"`
	Capabilities capture.Capabilities         `json:"capabilities"`
}

type RecordingFreedomService struct {
	appData   *appdata.Service
	capture   *capture.Service
	devices   *devices.Service
	preflight *preflight.Service
	recorder  *recording.Service
	settings  *settings.Service

	app            *application.App
	settingsWindow *application.WebviewWindow
}

func NewRecordingFreedomService() *RecordingFreedomService {
	data := appdata.NewService("")
	return &RecordingFreedomService{
		appData:   data,
		capture:   capture.NewService(),
		devices:   devices.NewService(),
		preflight: preflight.NewService(),
		recorder:  recording.NewService(data),
		settings:  settings.NewService(data),
	}
}

func (s *RecordingFreedomService) setApp(app *application.App) {
	s.app = app
}

func (s *RecordingFreedomService) setSettingsWindow(window *application.WebviewWindow) {
	s.settingsWindow = window
}

func (s *RecordingFreedomService) ShowSettingsWindow() error {
	if s.settingsWindow == nil {
		return errors.New("settings window is not configured")
	}
	s.settingsWindow.Show()
	s.settingsWindow.Focus()
	return nil
}

func (s *RecordingFreedomService) HideSettingsWindow() error {
	if s.settingsWindow == nil {
		return errors.New("settings window is not configured")
	}
	s.settingsWindow.Hide()
	return nil
}

func (s *RecordingFreedomService) Bootstrap() (BootstrapState, error) {
	info, err := s.appData.Info()
	if err != nil {
		return BootstrapState{}, err
	}
	storage, _ := s.appData.StorageStatus()
	recoveries, err := s.recorder.ScanPackages()
	if err != nil {
		return BootstrapState{}, err
	}
	currentSettings, err := s.settings.Load()
	if err != nil {
		return BootstrapState{}, err
	}
	currentSettings.Storage.DataRootDir = info.RootDir
	return BootstrapState{
		AppData:      info,
		Storage:      storage,
		State:        s.recorder.State(),
		Backend:      s.recorder.BackendID(),
		Sources:      s.devices.ListSources(),
		Media:        s.devices.ListMediaDevices(),
		Recoveries:   recoveries,
		Settings:     currentSettings,
		Capabilities: s.capture.Capabilities(),
	}, nil
}

func (s *RecordingFreedomService) ListSources() []devices.CaptureSource {
	return s.devices.ListSources()
}

func (s *RecordingFreedomService) ListMediaDevices() devices.MediaInventory {
	return s.devices.ListMediaDevices()
}

func (s *RecordingFreedomService) GetCaptureCapabilities() capture.Capabilities {
	return s.capture.Capabilities()
}

func (s *RecordingFreedomService) PreflightRecording(req recording.StartRequest) preflight.Summary {
	storage, _ := s.appData.StorageStatus()
	return s.preflight.Evaluate(req, preflight.Inputs{
		Backend:      s.recorder.BackendID(),
		Sources:      s.devices.ListSources(),
		Media:        s.devices.ListMediaDevices(),
		Capabilities: s.capture.Capabilities(),
		Storage:      storage,
	})
}

func (s *RecordingFreedomService) ScanRecordingPackages() ([]recpackage.RecoverySummary, error) {
	return s.recorder.ScanPackages()
}

func (s *RecordingFreedomService) RecoverRecordingPackage(packageDir string) (recpackage.RecoverySummary, error) {
	return s.recorder.RecoverPackage(packageDir)
}

func (s *RecordingFreedomService) GetSettings() (settings.Settings, error) {
	info, err := s.appData.Info()
	if err != nil {
		return settings.Settings{}, err
	}
	currentSettings, err := s.settings.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	currentSettings.Storage.DataRootDir = info.RootDir
	return currentSettings, nil
}

func (s *RecordingFreedomService) SaveSettings(next settings.Settings) (settings.Settings, error) {
	if err := s.applyDataRootFromSettings(next); err != nil {
		return settings.Settings{}, err
	}
	info, err := s.appData.Info()
	if err != nil {
		return settings.Settings{}, err
	}
	next.Storage.DataRootDir = info.RootDir
	return s.settings.Save(next)
}

func (s *RecordingFreedomService) SetDataRoot(rootDir string) (appdata.Info, error) {
	if recorderIsActive(s.recorder.State()) {
		return appdata.Info{}, errors.New("cannot change data root while recording is active")
	}
	info, err := s.appData.SetRootDir(rootDir)
	if err != nil {
		return appdata.Info{}, err
	}
	currentSettings, err := s.settings.Load()
	if err != nil {
		return appdata.Info{}, err
	}
	currentSettings.Storage.DataRootDir = info.RootDir
	if _, err := s.settings.Save(currentSettings); err != nil {
		return appdata.Info{}, err
	}
	return info, nil
}

func (s *RecordingFreedomService) applyDataRootFromSettings(next settings.Settings) error {
	target := strings.TrimSpace(next.Storage.DataRootDir)
	if target == "" {
		return nil
	}
	currentRoot, err := s.appData.RootDir()
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if targetAbs == currentRoot {
		return nil
	}
	if recorderIsActive(s.recorder.State()) {
		return fmt.Errorf("cannot change data root from %q to %q while recording is active", currentRoot, targetAbs)
	}
	_, err = s.appData.SetRootDir(targetAbs)
	return err
}

func recorderIsActive(state recording.State) bool {
	switch state {
	case recording.StatePreparing, recording.StateRecording, recording.StatePaused, recording.StateStopping:
		return true
	default:
		return false
	}
}

func (s *RecordingFreedomService) StartRecording(req recording.StartRequest) (recording.Session, error) {
	s.emitRecordingStatus(recording.StatusEvent{
		Status:  recording.StatePreparing,
		Backend: s.recorder.BackendID(),
		Message: "Preparing recording package",
	})
	session, err := s.recorder.StartRecording(req)
	if err != nil {
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.BackendID(),
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
	s.emitSessionStatus(session, "Recording started")
	return session, nil
}

func (s *RecordingFreedomService) StartMockRecording(req recording.StartRequest) (recording.Session, error) {
	return s.StartRecording(req)
}

func (s *RecordingFreedomService) PauseRecording() (recording.Session, error) {
	session, err := s.recorder.Pause()
	if err != nil {
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.BackendID(),
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
	s.emitSessionStatus(session, "Recording paused")
	return session, nil
}

func (s *RecordingFreedomService) ResumeRecording() (recording.Session, error) {
	session, err := s.recorder.Resume()
	if err != nil {
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.BackendID(),
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
	s.emitSessionStatus(session, "Recording resumed")
	return session, nil
}

func (s *RecordingFreedomService) StopRecording() (recording.Session, error) {
	s.emitRecordingStatus(recording.StatusEvent{
		Status:  recording.StateStopping,
		Backend: s.recorder.BackendID(),
		Message: "Finalizing recording package",
	})
	session, err := s.recorder.Stop()
	if err != nil {
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.BackendID(),
			Message: err.Error(),
		})
		return session, err
	}
	s.emitSessionStatus(session, "Recording package ready")
	return session, nil
}

func (s *RecordingFreedomService) emitSessionStatus(session recording.Session, message string) {
	s.emitRecordingStatus(recording.StatusEvent{
		Status:     session.Status,
		SessionID:  session.ID,
		PackageDir: session.PackageDir,
		Manifest:   session.Manifest,
		Backend:    session.Backend,
		Message:    message,
	})
}

func (s *RecordingFreedomService) emitRecordingStatus(event recording.StatusEvent) {
	if s.app == nil {
		return
	}
	s.app.Event.Emit("recording.status", event)
}
