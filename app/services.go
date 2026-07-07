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
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/preflight"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	secretstore "github.com/lemon-casino/RecordingFreedom/app/internal/secrets"
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
	PackageDir         string `json:"packageDir"`
	OutputPath         string `json:"outputPath,omitempty"`
	CanvasWidth        int    `json:"canvasWidth,omitempty"`
	CanvasHeight       int    `json:"canvasHeight,omitempty"`
	IncludeAnnotations *bool  `json:"includeAnnotations,omitempty"`
}

type ExportRecordingResult struct {
	Plan   exportplan.Plan `json:"plan"`
	Export exporter.Result `json:"export"`
}

type ExportRecordingPlanResult struct {
	Plan exportplan.Plan `json:"plan"`
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

type AnnotationPreviewImageRequest struct {
	PackageDir   string `json:"packageDir"`
	SnapshotPath string `json:"snapshotPath"`
}

type AnnotationPreviewImageResult struct {
	Available    bool   `json:"available"`
	DataURL      string `json:"dataUrl,omitempty"`
	RelativePath string `json:"relativePath,omitempty"`
	Bytes        int64  `json:"bytes,omitempty"`
}

type ClientLogEvent struct {
	Component string            `json:"component"`
	Event     string            `json:"event"`
	Message   string            `json:"message,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type CameraStatePatchRequest struct {
	Enabled   *bool       `json:"enabled,omitempty"`
	DeviceID  *string     `json:"deviceId,omitempty"`
	PIPPreset *string     `json:"pipPreset,omitempty"`
	PIP       *pip.Config `json:"pip,omitempty"`
}

const (
	maxPIPPreviewImageBytes        = 2 * 1024 * 1024
	maxAnnotationPreviewImageBytes = 8 * 1024 * 1024
)

type RecordingFreedomService struct {
	appData   *appdata.Service
	capture   *capture.Service
	devices   *devices.Service
	preflight *preflight.Service
	recorder  *recording.Service
	settings  *settings.Service
	secrets   *secretstore.Store
	ocr       *ocr.Service

	app                      *application.App
	capsuleWindow            *application.WebviewWindow
	settingsWindow           *application.WebviewWindow
	whiteboardWindow         *application.WebviewWindow
	annotationOverlay        *application.WebviewWindow
	annotationRenderer       *application.WebviewWindow
	regionOverlay            *application.WebviewWindow
	screenIndicator          *application.WebviewWindow
	pipOverlay               *application.WebviewWindow
	screenshotPinWindow      *application.WebviewWindow
	floatingPanelWindow      *application.WebviewWindow
	floatingSelectWindow     *application.WebviewWindow
	trayLocale               func(settings.Locale)
	capsuleHitRegions        capsuleWindowHitRegions
	annotationHitRegions     capsuleWindowHitRegions
	floatingPanelRegions     capsuleWindowHitRegions
	floatingSelectRegions    capsuleWindowHitRegions
	floatingMu               sync.Mutex
	floatingPanelState       FloatingPanelState
	floatingSelectState      FloatingSelectState
	floatingOutsideClickMu   sync.Mutex
	floatingOutsideClickOn   bool
	sourceMu                 sync.Mutex
	sourceState              SourceControlState
	settingsMu               sync.Mutex
	whiteboardMu             sync.Mutex
	whiteboardVisible        bool
	annotationMu             sync.Mutex
	annotationToken          uint64
	annotationSessionID      string
	annotationRegionDIP      application.Rect
	annotationRenderMu       sync.Mutex
	annotationRenderBatch    *annotationRenderBatch
	regionMu                 sync.Mutex
	regionSession            *RegionSelectionSession
	regionElementCacheMu     sync.Mutex
	regionElementCache       regionElementCandidateCache
	regionSnapshotMu         sync.Mutex
	regionSnapshotCache      regionAssistSnapshotCache
	regionAssistLogMu        sync.Mutex
	regionAssistLogKey       string
	regionAssistLogAt        time.Time
	selectedRegionDIP        application.Rect
	screenshotRegionDIP      application.Rect
	pipOverlayMu             sync.Mutex
	pipOverlayToken          uint64
	micLevelMu               sync.Mutex
	micLevelSource           audioLevelCaptureSource
	micLevelDevice           string
	micLevelToken            uint64
	logMu                    sync.Mutex
	shortcutMu               sync.Mutex
	registeredShortcuts      map[settings.ShortcutAction]string
	screenshotMu             sync.Mutex
	screenshotPinState       ScreenshotPinState
	whiteboardScreenshot     ScreenshotWhiteboardContext
	screenshotAnnotation     ScreenshotWhiteboardContext
	ocrCancelledSources      map[string]map[string]struct{}
	ocrPumpOnce              sync.Once
	ocrModelDownloadPumpOnce sync.Once
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
		secrets:   secretstore.NewStore(data),
		ocr:       ocr.NewService(data),
	}
}

func (s *RecordingFreedomService) setApp(app *application.App) {
	s.app = app
	s.startOCRJobEventPump()
	s.startOCRModelDownloadEventPump()
}

func (s *RecordingFreedomService) setCapsuleWindow(window *application.WebviewWindow) {
	s.capsuleWindow = window
}

func (s *RecordingFreedomService) setSettingsWindow(window *application.WebviewWindow) {
	s.settingsWindow = window
}

func (s *RecordingFreedomService) setWhiteboardWindow(window *application.WebviewWindow) {
	s.whiteboardWindow = window
}

func (s *RecordingFreedomService) setAnnotationOverlayWindow(window *application.WebviewWindow) {
	s.annotationOverlay = window
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

func (s *RecordingFreedomService) setScreenshotPinWindow(window *application.WebviewWindow) {
	s.screenshotPinWindow = window
}

func (s *RecordingFreedomService) setFloatingPanelWindow(window *application.WebviewWindow) {
	s.floatingPanelWindow = window
}

func (s *RecordingFreedomService) setFloatingSelectWindow(window *application.WebviewWindow) {
	s.floatingSelectWindow = window
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

	image, err := readPreviewImageDataURL(path, "PIP preview image", "image/jpeg", maxPIPPreviewImageBytes, req.KnownModifiedUnixNano)
	if err != nil {
		return PIPPreviewImageResult{}, err
	}
	return PIPPreviewImageResult{
		Available:        image.Available,
		DataURL:          image.DataURL,
		ModifiedUnixNano: image.ModifiedUnixNano,
	}, nil
}

func (s *RecordingFreedomService) ReadAnnotationPreviewImage(req AnnotationPreviewImageRequest) (AnnotationPreviewImageResult, error) {
	if s.appData == nil {
		return AnnotationPreviewImageResult{}, errors.New("app data service is not initialized")
	}
	info, err := s.appData.Info()
	if err != nil {
		return AnnotationPreviewImageResult{}, err
	}
	packageDir, err := managedRecordingPackageDir(info.VideoDir, req.PackageDir)
	if err != nil {
		return AnnotationPreviewImageResult{}, err
	}
	target, relativePath, err := annotationPreviewSnapshotPath(packageDir, req.SnapshotPath)
	if err != nil {
		return AnnotationPreviewImageResult{}, err
	}
	image, err := readPreviewImageDataURL(target, "annotation preview image", "image/png", maxAnnotationPreviewImageBytes, 0)
	if err != nil {
		return AnnotationPreviewImageResult{}, err
	}
	return AnnotationPreviewImageResult{
		Available:    image.Available,
		DataURL:      image.DataURL,
		RelativePath: relativePath,
		Bytes:        image.Bytes,
	}, nil
}

type previewImageDataURL struct {
	Available        bool
	DataURL          string
	ModifiedUnixNano int64
	Bytes            int64
}

func readPreviewImageDataURL(path string, label string, mimeType string, maxBytes int64, knownModifiedUnixNano int64) (previewImageDataURL, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return previewImageDataURL{Available: false}, nil
	}
	if err != nil {
		return previewImageDataURL{}, err
	}
	if info.IsDir() {
		return previewImageDataURL{}, fmt.Errorf("%s %q is a directory", label, path)
	}
	modified := info.ModTime().UnixNano()
	if knownModifiedUnixNano > 0 && modified <= knownModifiedUnixNano {
		return previewImageDataURL{Available: false, ModifiedUnixNano: modified}, nil
	}
	if info.Size() <= 0 {
		return previewImageDataURL{Available: false, ModifiedUnixNano: modified}, nil
	}
	if info.Size() > maxBytes {
		return previewImageDataURL{}, fmt.Errorf("%s %q is too large", label, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return previewImageDataURL{}, err
	}
	if len(data) == 0 {
		return previewImageDataURL{Available: false, ModifiedUnixNano: modified}, nil
	}
	return previewImageDataURL{
		Available:        true,
		DataURL:          "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data),
		ModifiedUnixNano: modified,
		Bytes:            info.Size(),
	}, nil
}

func annotationPreviewSnapshotPath(packageDir string, value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", errors.New("annotation snapshot path is required")
	}
	packageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return "", "", err
	}
	var target string
	if filepath.IsAbs(value) {
		target, err = filepath.Abs(value)
		if err != nil {
			return "", "", err
		}
	} else {
		if err := validateAnnotationPreviewRelativePath(value); err != nil {
			return "", "", err
		}
		target = filepath.Join(packageDir, filepath.Clean(value))
	}
	relativePath, err := filepath.Rel(packageDir, target)
	if err != nil {
		return "", "", err
	}
	if relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		return "", "", fmt.Errorf("annotation preview image %q must stay inside package %q", value, packageDir)
	}
	relativePath = filepath.ToSlash(filepath.Clean(relativePath))
	if !annotationPreviewSnapshotAllowed(relativePath) {
		return "", "", fmt.Errorf("annotation preview image %q must be an annotation snapshot", value)
	}
	if strings.ToLower(filepath.Ext(relativePath)) != ".png" {
		return "", "", fmt.Errorf("annotation preview image %q must be a PNG", value)
	}
	return target, relativePath, nil
}

func annotationPreviewSnapshotAllowed(relativePath string) bool {
	relativePath = filepath.ToSlash(strings.TrimSpace(relativePath))
	if relativePath == recpackage.AnnotationSnapshotFile {
		return true
	}
	for _, dir := range []string{recpackage.AnnotationSnapshotsDir, recpackage.AnnotationRenderPNGDir} {
		prefix := filepath.ToSlash(dir) + "/"
		if strings.HasPrefix(relativePath, prefix) {
			return true
		}
	}
	return false
}

func validateAnnotationPreviewRelativePath(value string) error {
	if value == "" {
		return nil
	}
	if filepath.IsAbs(value) {
		return fmt.Errorf("annotation snapshot path must be package-relative, got absolute path %q", value)
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("annotation snapshot path must stay inside the recording package, got %q", value)
	}
	return nil
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
	currentSettings, err := s.loadSettingsForClient()
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
	plan, err := s.exportRecordingPlan(req, true)
	if err != nil {
		return ExportRecordingResult{}, err
	}
	plan, err = s.ensureAnnotationRenderedAssets(req, plan)
	if err != nil {
		return ExportRecordingResult{}, err
	}
	result, err := exporter.NewService().Export(nil, plan, exporter.Options{})
	if err != nil {
		return ExportRecordingResult{}, err
	}
	return ExportRecordingResult{Plan: plan, Export: result}, nil
}

func (s *RecordingFreedomService) PreviewExportRecordingPackage(req ExportRecordingRequest) (ExportRecordingPlanResult, error) {
	plan, err := s.exportRecordingPlan(req, false)
	if err != nil {
		return ExportRecordingPlanResult{}, err
	}
	return ExportRecordingPlanResult{Plan: plan}, nil
}

func (s *RecordingFreedomService) exportRecordingPlan(req ExportRecordingRequest, prepareAnnotationAssets bool) (exportplan.Plan, error) {
	if recorderIsActive(s.recorder.State()) {
		return exportplan.Plan{}, errors.New("cannot export a recording package while recording is active")
	}
	info, err := s.appData.Info()
	if err != nil {
		return exportplan.Plan{}, err
	}
	packageDir, err := managedRecordingPackageDir(info.VideoDir, req.PackageDir)
	if err != nil {
		return exportplan.Plan{}, err
	}
	outputPath := strings.TrimSpace(req.OutputPath)
	if outputPath == "" {
		outputPath = exportplan.DefaultOutputPath
	}
	return exportplan.NewService(nil).Plan(exportplan.Request{
		VideoDir:                info.VideoDir,
		PackageDir:              packageDir,
		OutputPath:              outputPath,
		Canvas:                  pip.Size{Width: req.CanvasWidth, Height: req.CanvasHeight},
		RequireSync:             true,
		IncludeAnnotations:      req.IncludeAnnotations,
		PrepareAnnotationAssets: prepareAnnotationAssets,
	})
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
	currentSettings, err := s.loadSettingsForClient()
	if err != nil {
		return settings.Settings{}, err
	}
	currentSettings.Storage.DataRootDir = info.RootDir
	return currentSettings, nil
}

var openPath = defaultOpenPath
var openFileLocation = defaultOpenFileLocation

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
	cmd := exec.Command(command, args...)
	configureBackgroundCommand(cmd)
	if err := cmd.Start(); err != nil {
		if runtime.GOOS == "windows" {
			fallback := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", absoluteTarget)
			configureBackgroundCommand(fallback)
			return fallback.Start()
		}
		return err
	}
	return nil
}

func defaultOpenFileLocation(path string) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return errors.New("path is required")
	}
	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	info, err := os.Stat(absoluteTarget)
	if err != nil {
		parent := filepath.Dir(absoluteTarget)
		if _, parentErr := os.Stat(parent); parentErr == nil {
			return openPath(parent)
		}
		return fmt.Errorf("cannot open %q: %w", absoluteTarget, err)
	}
	if info.IsDir() {
		return openPath(absoluteTarget)
	}
	if runtime.GOOS == "windows" {
		return openPath(filepath.Dir(absoluteTarget))
	}
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{"-R", absoluteTarget}
	case "windows":
		command = "explorer.exe"
		args = []string{"/select," + absoluteTarget}
	default:
		return openPath(filepath.Dir(absoluteTarget))
	}
	cmd := exec.Command(command, args...)
	configureBackgroundCommand(cmd)
	if err := cmd.Start(); err != nil {
		return openPath(filepath.Dir(absoluteTarget))
	}
	return nil
}

func (s *RecordingFreedomService) SaveSettings(next settings.Settings) (settings.Settings, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	if err := s.applyDataRootFromSettings(next); err != nil {
		return settings.Settings{}, err
	}
	info, err := s.appData.Info()
	if err != nil {
		return settings.Settings{}, err
	}
	if currentSettings, err := s.loadSettingsForMutation(); err == nil {
		next.Recording = currentSettings.Recording
		next.Audio = currentSettings.Audio
		next.Camera = currentSettings.Camera
		next.Whiteboard = currentSettings.Whiteboard
		next.OCR = currentSettings.OCR
		next.Window.Theme = currentSettings.Window.Theme
		next.Window.StartAtLogin = currentSettings.Window.StartAtLogin
		next.Shortcuts = currentSettings.Shortcuts
	}
	next.Storage.DataRootDir = info.RootDir
	saved, err := s.settings.Save(next)
	if err != nil {
		return settings.Settings{}, err
	}
	s.refreshTrayLocale(saved.Locale)
	s.emitSettingsChanged(saved)
	s.emitAudioState(saved.Audio)
	return saved, nil
}

func (s *RecordingFreedomService) PatchCameraState(patch CameraStatePatchRequest) (settings.Settings, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	saved, err := s.patchCameraStateLocked(patch)
	if err != nil {
		return settings.Settings{}, err
	}
	s.emitSettingsChanged(saved)
	s.logEvent("camera", "patch", map[string]string{
		"enabled":   fmt.Sprint(saved.Camera.Enabled),
		"deviceId":  strings.TrimSpace(saved.Camera.DeviceID),
		"pipPreset": strings.TrimSpace(saved.Camera.PIPPreset),
	})
	return saved, nil
}

func (s *RecordingFreedomService) patchCameraStateLocked(patch CameraStatePatchRequest) (settings.Settings, error) {
	if s.settings == nil {
		return settings.Settings{}, errors.New("settings service is not initialized")
	}
	currentSettings, err := s.loadSettingsForMutation()
	if err != nil {
		return settings.Settings{}, err
	}
	currentSettings.Camera = applyCameraStatePatch(currentSettings.Camera, patch)
	return s.settings.Save(currentSettings)
}

func applyCameraStatePatch(current settings.CameraSettings, patch CameraStatePatchRequest) settings.CameraSettings {
	next := current
	if patch.DeviceID != nil {
		if deviceID := strings.TrimSpace(*patch.DeviceID); deviceID != "" {
			next.DeviceID = deviceID
		}
	}
	if patch.PIP != nil {
		next.PIP = *patch.PIP
	}
	if patch.PIPPreset != nil {
		next.PIPPreset = strings.TrimSpace(*patch.PIPPreset)
	}
	if next.PIPPreset == "" {
		next.PIPPreset = string(next.PIP.Preset)
	}
	next.PIP = pip.NormalizeConfigForPreset(next.PIPPreset, next.PIP)
	next.PIPPreset = string(next.PIP.Preset)
	if patch.Enabled != nil {
		next.Enabled = *patch.Enabled
	}
	if next.Enabled && next.PIP.Preset == pip.PresetOff {
		next.PIP = pip.DefaultConfig()
		next.PIPPreset = string(next.PIP.Preset)
	}
	if !next.Enabled {
		next.PIP = pip.OffConfig()
		next.PIPPreset = string(pip.PresetOff)
	}
	return next
}

func (s *RecordingFreedomService) PatchAudioState(patch AudioStatePatchRequest) (AudioState, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	currentSettings, err := s.loadSettingsForMutation()
	if err != nil {
		return AudioState{}, err
	}
	currentSettings.Audio = applyAudioStatePatch(currentSettings.Audio, patch)
	saved, err := s.settings.Save(currentSettings)
	if err != nil {
		return AudioState{}, err
	}
	s.emitSettingsChanged(saved)
	state := audioStateFromSettings(saved.Audio)
	s.emitAudioState(saved.Audio)
	return state, nil
}

func (s *RecordingFreedomService) SetDataRoot(rootDir string) (appdata.Info, error) {
	if recorderIsActive(s.recorder.State()) {
		return appdata.Info{}, errors.New("cannot change data root while recording is active")
	}
	info, err := s.appData.SetRootDir(rootDir)
	if err != nil {
		return appdata.Info{}, err
	}
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	currentSettings, err := s.loadSettingsForMutation()
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
	s.emitAudioState(saved.Audio)
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
	_ = s.HideFloatingPanel(0)
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
		s.logEvent("recording", "start-blocked", map[string]string{
			"reason": err.Error(),
		})
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.BackendID(),
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
	s.releasePIPOverlayMediaForRecording(req)
	s.emitRecordingStatus(recording.StatusEvent{
		Status:  recording.StatePreparing,
		Backend: s.recorder.BackendID(),
		Message: "Preparing recording package",
	})
	if req.Camera.Enabled {
		s.logEvent("camera", "native-start-request", map[string]string{
			"deviceId":  strings.TrimSpace(req.Camera.DeviceID),
			"nativeId":  strings.TrimSpace(req.Camera.DeviceNativeID),
			"pipPreset": strings.TrimSpace(req.Camera.PIPPreset),
		})
	}
	session, err := s.recorder.StartRecording(req)
	if err != nil {
		s.logEvent("recording", "start-error", map[string]string{
			"error": err.Error(),
		})
		s.emitRecordingStatus(recording.StatusEvent{
			Status:  recording.StateFailed,
			Backend: s.recorder.BackendID(),
			Message: err.Error(),
		})
		return recording.Session{}, err
	}
	s.lockRegionFrameForRecording(req)
	if req.Camera.Enabled {
		s.logEvent("camera", "native-started", map[string]string{
			"deviceId":         strings.TrimSpace(req.Camera.DeviceID),
			"nativeId":         strings.TrimSpace(req.Camera.DeviceNativeID),
			"packageDir":       strings.TrimSpace(session.PackageDir),
			"previewImagePath": recording.CameraPreviewImagePath(session.PackageDir),
		})
	}
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
	_ = s.HideFloatingPanel(0)
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
	_ = s.HideFloatingPanel(0)
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
	_ = s.HideAnnotationOverlay()
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

func (s *RecordingFreedomService) emitWhiteboardVisibility(visible bool, mode string) {
	if s.app == nil {
		return
	}
	s.app.Event.Emit("whiteboard.visibility", WhiteboardVisibilityEvent{
		Visible: visible,
		Mode:    strings.TrimSpace(mode),
	})
}

func (s *RecordingFreedomService) emitAudioState(audio settings.AudioSettings) {
	if s.app == nil {
		return
	}
	s.app.Event.Emit("audio.state", audioStateFromSettings(audio))
}
