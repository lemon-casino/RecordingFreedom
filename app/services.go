package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/exporter"
	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
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

type ExportRecordingRequest struct {
	PackageDir   string `json:"packageDir"`
	OutputPath   string `json:"outputPath,omitempty"`
	CanvasWidth  int    `json:"canvasWidth,omitempty"`
	CanvasHeight int    `json:"canvasHeight,omitempty"`
}

type ExportRecordingResult struct {
	Plan   exportplan.Plan `json:"plan"`
	Export exporter.Result `json:"export"`
}

type PIPPreviewImageRequest struct {
	Path                  string `json:"path"`
	KnownModifiedUnixNano int64  `json:"knownModifiedUnixNano,omitempty"`
}

type PIPPreviewImageResult struct {
	Available        bool   `json:"available"`
	DataURL          string `json:"dataUrl,omitempty"`
	ModifiedUnixNano int64  `json:"modifiedUnixNano,omitempty"`
}

type ClientLogEvent struct {
	Component string            `json:"component"`
	Event     string            `json:"event"`
	Message   string            `json:"message,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

const maxPIPPreviewImageBytes = 2 * 1024 * 1024

type RecordingFreedomService struct {
	appData   *appdata.Service
	capture   *capture.Service
	devices   *devices.Service
	preflight *preflight.Service
	recorder  *recording.Service
	settings  *settings.Service

	app               *application.App
	capsuleWindow     *application.WebviewWindow
	settingsWindow    *application.WebviewWindow
	regionOverlay     *application.WebviewWindow
	screenIndicator   *application.WebviewWindow
	pipOverlay        *application.WebviewWindow
	trayLocale        func(settings.Locale)
	capsuleHitRegions capsuleWindowHitRegions
	regionMu          sync.Mutex
	regionSession     *RegionSelectionSession
	selectedRegionDIP application.Rect
	pipOverlayMu      sync.Mutex
	pipOverlayToken   uint64
	micLevelMu        sync.Mutex
	micLevelSource    audioLevelCaptureSource
	micLevelDevice    string
	micLevelToken     uint64
	logMu             sync.Mutex
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

func (s *RecordingFreedomService) setCapsuleWindow(window *application.WebviewWindow) {
	s.capsuleWindow = window
}

func (s *RecordingFreedomService) setSettingsWindow(window *application.WebviewWindow) {
	s.settingsWindow = window
}

func (s *RecordingFreedomService) setRegionOverlayWindow(window *application.WebviewWindow) {
	s.regionOverlay = window
}

func (s *RecordingFreedomService) setScreenIndicatorWindow(window *application.WebviewWindow) {
	s.screenIndicator = window
}

func (s *RecordingFreedomService) setPIPOverlayWindow(window *application.WebviewWindow) {
	s.pipOverlay = window
}

func (s *RecordingFreedomService) setTrayLocaleUpdater(update func(settings.Locale)) {
	s.trayLocale = update
}

func (s *RecordingFreedomService) LogClientEvent(event ClientLogEvent) error {
	return s.writeLog("client."+strings.TrimSpace(event.Component), strings.TrimSpace(event.Event), strings.TrimSpace(event.Message), event.Fields)
}

func (s *RecordingFreedomService) ReadPIPPreviewImage(req PIPPreviewImageRequest) (PIPPreviewImageResult, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return PIPPreviewImageResult{}, errors.New("PIP preview image path is required")
	}
	if s.appData == nil {
		return PIPPreviewImageResult{}, errors.New("app data service is not initialized")
	}
	videoDir, err := s.appData.VideoDir()
	if err != nil {
		return PIPPreviewImageResult{}, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return PIPPreviewImageResult{}, err
	}
	videoDir, err = filepath.Abs(videoDir)
	if err != nil {
		return PIPPreviewImageResult{}, err
	}
	rel, err := filepath.Rel(videoDir, path)
	if err != nil {
		return PIPPreviewImageResult{}, err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return PIPPreviewImageResult{}, fmt.Errorf("PIP preview image %q is outside the managed video directory", path)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".jpg" && ext != ".jpeg" {
		return PIPPreviewImageResult{}, fmt.Errorf("PIP preview image %q must be a JPEG", path)
	}

	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return PIPPreviewImageResult{Available: false}, nil
	}
	if err != nil {
		return PIPPreviewImageResult{}, err
	}
	if info.IsDir() {
		return PIPPreviewImageResult{}, fmt.Errorf("PIP preview image %q is a directory", path)
	}
	modified := info.ModTime().UnixNano()
	if req.KnownModifiedUnixNano > 0 && modified <= req.KnownModifiedUnixNano {
		return PIPPreviewImageResult{Available: false, ModifiedUnixNano: modified}, nil
	}
	if info.Size() <= 0 {
		return PIPPreviewImageResult{Available: false, ModifiedUnixNano: modified}, nil
	}
	if info.Size() > maxPIPPreviewImageBytes {
		return PIPPreviewImageResult{}, fmt.Errorf("PIP preview image %q is too large", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return PIPPreviewImageResult{}, err
	}
	if len(data) == 0 {
		return PIPPreviewImageResult{Available: false, ModifiedUnixNano: modified}, nil
	}
	return PIPPreviewImageResult{
		Available:        true,
		DataURL:          "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data),
		ModifiedUnixNano: modified,
	}, nil
}

func (s *RecordingFreedomService) logEvent(component string, event string, fields map[string]string) {
	_ = s.writeLog(component, event, "", fields)
}

func (s *RecordingFreedomService) writeLog(component string, event string, message string, fields map[string]string) error {
	component = strings.TrimSpace(component)
	event = strings.TrimSpace(event)
	if component == "" {
		component = "app"
	}
	if event == "" {
		event = "event"
	}
	dir := s.logDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	now := time.Now()
	entry := map[string]any{
		"timestamp": now.Format(time.RFC3339Nano),
		"component": component,
		"event":     event,
	}
	if message != "" {
		entry["message"] = message
	}
	if len(fields) > 0 {
		entry["fields"] = fields
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "recordingfreedom-"+now.Format("2006-01-02")+".log")
	s.logMu.Lock()
	defer s.logMu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *RecordingFreedomService) logDir() string {
	root := ""
	if s != nil && s.appData != nil {
		root, _ = s.appData.RootDir()
	}
	if strings.TrimSpace(root) == "" {
		root = softwareRootFallback()
	}
	return filepath.Join(root, "logs")
}

func softwareRootFallback() string {
	if executable, err := os.Executable(); err == nil {
		if dir := strings.TrimSpace(filepath.Dir(executable)); dir != "" && dir != "." {
			return dir
		}
	}
	if workingDir, err := os.Getwd(); err == nil && strings.TrimSpace(workingDir) != "" {
		return workingDir
	}
	return "."
}

func (s *RecordingFreedomService) restoreCapsuleWindow() {
	if s == nil || s.capsuleWindow == nil {
		return
	}
	s.capsuleWindow.SetAlwaysOnTop(true)
	s.capsuleWindow.Show()
	s.capsuleWindow.UnMinimise()
	s.capsuleWindow.Focus()
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
	media := s.devices.ListMediaDevices()
	req = enrichRecordingCameraRequest(req, media)
	return s.evaluateRecordingPreflight(req, media)
}

func (s *RecordingFreedomService) evaluateRecordingPreflight(req recording.StartRequest, media devices.MediaInventory) preflight.Summary {
	storage, _ := s.appData.StorageStatus()
	return s.preflight.Evaluate(req, preflight.Inputs{
		Backend:      s.recorder.BackendID(),
		Sources:      s.devices.ListSources(),
		Media:        media,
		Capabilities: s.capture.Capabilities(),
		Storage:      storage,
	})
}

func (s *RecordingFreedomService) PreflightAudioOnlyRecording(req recording.AudioOnlyRequest) preflight.Summary {
	storage, _ := s.appData.StorageStatus()
	return s.preflight.EvaluateAudioOnly(req, preflight.Inputs{
		Backend:      recording.BackendAudioOnlyNative,
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

func (s *RecordingFreedomService) OpenVideoDirectory() (appdata.Info, error) {
	info, err := s.appData.Info()
	if err != nil {
		return appdata.Info{}, err
	}
	if err := openPath(info.VideoDir); err != nil {
		return appdata.Info{}, err
	}
	return info, nil
}

func (s *RecordingFreedomService) OpenRecordingPackage(packageDir string) (recpackage.RecoverySummary, error) {
	info, err := s.appData.Info()
	if err != nil {
		return recpackage.RecoverySummary{}, err
	}
	summary, err := managedRecordingPackageSummary(info.VideoDir, packageDir)
	if err != nil {
		return recpackage.RecoverySummary{}, err
	}
	if err := openPath(summary.PackageDir); err != nil {
		return recpackage.RecoverySummary{}, err
	}
	return summary, nil
}

func (s *RecordingFreedomService) ExportRecordingPackage(req ExportRecordingRequest) (ExportRecordingResult, error) {
	if recorderIsActive(s.recorder.State()) {
		return ExportRecordingResult{}, errors.New("cannot export a recording package while recording is active")
	}
	info, err := s.appData.Info()
	if err != nil {
		return ExportRecordingResult{}, err
	}
	packageDir, err := managedRecordingPackageDir(info.VideoDir, req.PackageDir)
	if err != nil {
		return ExportRecordingResult{}, err
	}
	outputPath := strings.TrimSpace(req.OutputPath)
	if outputPath == "" {
		outputPath = exportplan.DefaultOutputPath
	}
	plan, err := exportplan.NewService(nil).Plan(exportplan.Request{
		VideoDir:    info.VideoDir,
		PackageDir:  packageDir,
		OutputPath:  outputPath,
		Canvas:      pip.Size{Width: req.CanvasWidth, Height: req.CanvasHeight},
		RequireSync: true,
	})
	if err != nil {
		return ExportRecordingResult{}, err
	}
	result, err := exporter.NewService().Export(nil, plan, exporter.Options{})
	if err != nil {
		return ExportRecordingResult{}, err
	}
	return ExportRecordingResult{Plan: plan, Export: result}, nil
}

func managedRecordingPackageSummary(videoDir string, packageDir string) (recpackage.RecoverySummary, error) {
	target, err := managedRecordingPackageDir(videoDir, packageDir)
	if err != nil {
		return recpackage.RecoverySummary{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return recpackage.RecoverySummary{}, err
	}
	if !info.IsDir() {
		return recpackage.RecoverySummary{}, fmt.Errorf("packageDir %q is not a directory", packageDir)
	}
	manifestPath := filepath.Join(target, recpackage.ManifestFile)
	summary := recpackage.RecoverySummary{
		PackageDir:   target,
		ManifestPath: manifestPath,
	}
	manifest, err := recpackage.NewService().ReadManifest(manifestPath)
	if err != nil {
		summary.Status = recpackage.StatusFailed
		summary.Reason = fmt.Sprintf("manifest is missing or unreadable: %v", err)
		return summary, nil
	}
	summary.Status = manifest.Status
	if summary.Status == "" {
		summary.Status = recpackage.StatusFailed
		summary.Reason = "manifest status is empty"
	}
	return summary, nil
}

func managedRecordingPackageDir(videoDir string, packageDir string) (string, error) {
	if strings.TrimSpace(videoDir) == "" {
		return "", errors.New("videoDir is required")
	}
	if strings.TrimSpace(packageDir) == "" {
		return "", errors.New("packageDir is required")
	}
	videoRoot, err := filepath.Abs(videoDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(packageDir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(videoRoot, target)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("packageDir %q must be inside videoDir %q", packageDir, videoDir)
	}
	if !strings.HasSuffix(filepath.Base(target), recpackage.PackageDirSuffix) {
		return "", fmt.Errorf("packageDir %q must end with %s", packageDir, recpackage.PackageDirSuffix)
	}
	return target, nil
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

var openPath = defaultOpenPath

func defaultOpenPath(path string) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return errors.New("path is required")
	}
	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absoluteTarget); err != nil {
		return fmt.Errorf("cannot open %q: %w", absoluteTarget, err)
	}
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{absoluteTarget}
	case "windows":
		command = "explorer.exe"
		args = []string{absoluteTarget}
	default:
		command = "xdg-open"
		args = []string{absoluteTarget}
	}
	if err := exec.Command(command, args...).Start(); err != nil {
		if runtime.GOOS == "windows" {
			return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", absoluteTarget).Start()
		}
		return err
	}
	return nil
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
	saved, err := s.settings.Save(next)
	if err != nil {
		return settings.Settings{}, err
	}
	s.refreshTrayLocale(saved.Locale)
	s.emitSettingsChanged(saved)
	return saved, nil
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
	saved, err := s.settings.Save(currentSettings)
	if err != nil {
		return appdata.Info{}, err
	}
	s.refreshTrayLocale(saved.Locale)
	s.emitSettingsChanged(saved)
	return info, nil
}

func (s *RecordingFreedomService) refreshTrayLocale(locale settings.Locale) {
	if s.trayLocale == nil {
		return
	}
	s.trayLocale(locale)
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
	_ = s.StopMicrophoneLevelMonitor()
	media := devices.MediaInventory{}
	if s.devices != nil {
		media = s.devices.ListMediaDevices()
		req = enrichRecordingCameraRequest(req, media)
	}
	s.logEvent("recording", "start-request", map[string]string{
		"sourceType":        string(req.SourceType),
		"cameraEnabled":     fmt.Sprint(req.Camera.Enabled),
		"cameraDeviceId":    strings.TrimSpace(req.Camera.DeviceID),
		"cameraNativeId":    strings.TrimSpace(req.Camera.DeviceNativeID),
		"cameraPipPreset":   strings.TrimSpace(req.Camera.PIPPreset),
		"microphoneEnabled": fmt.Sprint(req.Audio.Microphone),
		"systemAudio":       fmt.Sprint(req.Audio.System),
	})
	if summary, blocked := s.blockingRecordingPreflight(req, media); blocked {
		err := fmt.Errorf("preflight blocked: %s", firstBlockedPreflightReason(summary))
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.BackendID(),
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
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
	s.lockRegionFrameForRecording(req)
	s.showRecordingPIPOverlay(req, session)
	s.emitSessionStatus(session, "Recording started")
	return session, nil
}

func (s *RecordingFreedomService) StartMockRecording(req recording.StartRequest) (recording.Session, error) {
	return s.StartRecording(req)
}

func (s *RecordingFreedomService) blockingRecordingPreflight(req recording.StartRequest, media devices.MediaInventory) (preflight.Summary, bool) {
	if s.preflight == nil || s.devices == nil || s.capture == nil || s.appData == nil {
		return preflight.Summary{}, false
	}
	if media.Cameras == nil && media.SystemAudio == nil && media.Microphones == nil {
		media = s.devices.ListMediaDevices()
	}
	summary := s.evaluateRecordingPreflight(req, media)
	return summary, summary.Status == preflight.StatusBlocked
}

func enrichRecordingCameraRequest(req recording.StartRequest, media devices.MediaInventory) recording.StartRequest {
	normalized, err := recording.NormalizeStartRequest(req)
	if err != nil || !normalized.Camera.Enabled {
		return req
	}
	deviceID := strings.TrimSpace(normalized.Camera.DeviceID)
	selected := devices.MediaDevice{}
	for _, camera := range media.Cameras {
		if camera.ID == deviceID {
			selected = camera
			break
		}
	}
	if !usableCameraSidecarDevice(selected) {
		for _, candidate := range media.Cameras {
			if usableCameraSidecarDevice(candidate) {
				selected = candidate
				break
			}
		}
	}
	if selected.ID == "" {
		return req
	}
	req.Camera.DeviceID = selected.ID
	req.Camera.DeviceNativeID = selected.NativeID
	return req
}

func usableCameraSidecarDevice(camera devices.MediaDevice) bool {
	return camera.ID != "" &&
		camera.Available &&
		camera.SidecarEligible &&
		strings.TrimSpace(camera.NativeID) != ""
}

func (s *RecordingFreedomService) blockingAudioOnlyPreflight(req recording.AudioOnlyRequest) (preflight.Summary, bool) {
	if s.preflight == nil || s.devices == nil || s.capture == nil || s.appData == nil {
		return preflight.Summary{}, false
	}
	summary := s.PreflightAudioOnlyRecording(req)
	return summary, summary.Status == preflight.StatusBlocked
}

func firstBlockedPreflightReason(summary preflight.Summary) string {
	for _, check := range summary.Checks {
		if check.Status == preflight.StatusBlocked && check.Reason != "" {
			return check.Reason
		}
	}
	if summary.Message != "" {
		return summary.Message
	}
	return "recording preflight failed"
}

func (s *RecordingFreedomService) StartAudioOnlyRecording(req recording.AudioOnlyRequest) (recording.Session, error) {
	_ = s.StopMicrophoneLevelMonitor()
	if summary, blocked := s.blockingAudioOnlyPreflight(req); blocked {
		err := fmt.Errorf("preflight blocked: %s", firstBlockedPreflightReason(summary))
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: recording.BackendAudioOnlyNative,
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
	s.emitRecordingStatus(recording.StatusEvent{
		Status:  recording.StatePreparing,
		Backend: recording.BackendAudioOnlyNative,
		Message: "Preparing audio-only recording package",
	})
	session, err := s.recorder.StartAudioOnlyRecording(req)
	if err != nil {
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: recording.BackendAudioOnlyNative,
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
	s.emitSessionStatus(session, "Audio-only recording started")
	return session, nil
}

func (s *RecordingFreedomService) PauseRecording() (recording.Session, error) {
	session, err := s.recorder.Pause()
	if err != nil {
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.ActiveBackendID(),
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
			Backend: s.recorder.ActiveBackendID(),
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
		Backend: s.recorder.ActiveBackendID(),
		Message: "Finalizing recording package",
	})
	defer s.restoreCapsuleWindow()
	session, err := s.recorder.Stop()
	if err != nil {
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.ActiveBackendID(),
			Message: err.Error(),
		})
		return session, err
	}
	_ = s.HidePIPOverlay()
	s.emitSessionStatus(session, "Recording package ready")
	return session, nil
}

func (s *RecordingFreedomService) lockRegionFrameForRecording(req recording.StartRequest) {
	if req.SourceType != recording.SourceRegion || req.SourceGeometry == nil {
		return
	}
	rect := s.selectedRegionDisplayBounds()
	if rect.Width <= 0 || rect.Height <= 0 {
		rect = application.Rect{
			X:      req.SourceGeometry.X,
			Y:      req.SourceGeometry.Y,
			Width:  req.SourceGeometry.Width,
			Height: req.SourceGeometry.Height,
		}
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	_ = s.showRegionFrame(rect)
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

func (s *RecordingFreedomService) emitSettingsChanged(next settings.Settings) {
	if s.app == nil {
		return
	}
	s.app.Event.Emit("settings.changed", next)
}
