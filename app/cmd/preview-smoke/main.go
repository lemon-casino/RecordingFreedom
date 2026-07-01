package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/preflight"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

type smokeReport struct {
	OK              bool   `json:"ok"`
	DataRoot        string `json:"dataRoot"`
	VideoDir        string `json:"videoDir"`
	Backend         string `json:"backend"`
	SourceID        string `json:"sourceId"`
	SourceType      string `json:"sourceType"`
	PreflightStatus string `json:"preflightStatus"`
	StorageStatus   string `json:"storageStatus"`
	SessionID       string `json:"sessionId"`
	PackageDir      string `json:"packageDir"`
	ManifestPath    string `json:"manifestPath"`
	ScreenMarker    string `json:"screenMarker"`
	ManifestStatus  string `json:"manifestStatus"`
	LocaleChecked   string `json:"localeChecked"`
	KeptDataRoot    bool   `json:"keptDataRoot"`
}

func main() {
	var dataRoot string
	var keep bool
	flag.StringVar(&dataRoot, "data-dir", "", "data root for the smoke run; defaults to a temporary directory")
	flag.BoolVar(&keep, "keep", false, "keep the generated data root for manual inspection")
	flag.Parse()

	report, err := run(dataRoot, keep)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode smoke report: %v\n", err)
		os.Exit(1)
	}
}

func run(dataRoot string, keep bool) (smokeReport, error) {
	if strings.TrimSpace(dataRoot) == "" {
		tempRoot, err := os.MkdirTemp("", "recordingfreedom-preview-smoke-*")
		if err != nil {
			return smokeReport{}, err
		}
		dataRoot = tempRoot
		if !keep {
			defer os.RemoveAll(tempRoot)
		}
	}

	data := appdata.NewService(dataRoot)
	settingsService := settings.NewService(data)
	recorder := recording.NewServiceWithBackend(data, recording.SelectBackend(recpackage.NewService(), "", recording.BackendMockPackage))
	deviceService := devices.NewService()
	captureService := capture.NewService()
	preflightService := preflight.NewService()

	info, err := data.Info()
	if err != nil {
		return smokeReport{}, fmt.Errorf("app data info: %w", err)
	}
	if want := filepath.Join(info.RootDir, "data", "video"); info.VideoDir != want {
		return smokeReport{}, fmt.Errorf("video dir = %q, want %q", info.VideoDir, want)
	}
	storage, err := data.StorageStatus()
	if err != nil {
		return smokeReport{}, fmt.Errorf("storage status: %w", err)
	}
	if !storage.Writable || storage.Status == appdata.StorageStatusBlocked {
		return smokeReport{}, fmt.Errorf("storage is not ready for preview smoke: %#v", storage)
	}

	if err := verifySettings(settingsService, info.RootDir); err != nil {
		return smokeReport{}, err
	}

	sources := deviceService.ListSources()
	if len(sources) == 0 {
		return smokeReport{}, errors.New("device service returned no capture sources")
	}
	source := selectSource(sources)
	media := deviceService.ListMediaDevices()
	req := smokeStartRequest(source, media)

	preflightSummary := preflightService.Evaluate(req, preflight.Inputs{
		Backend:      recorder.BackendID(),
		Sources:      sources,
		Media:        media,
		Capabilities: captureService.Capabilities(),
		Storage:      storage,
	})
	if preflightSummary.Status == preflight.StatusBlocked {
		return smokeReport{}, fmt.Errorf("preflight blocked preview smoke: %#v", preflightSummary.Checks)
	}

	session, err := recorder.StartRecording(req)
	if err != nil {
		return smokeReport{}, fmt.Errorf("start recording: %w", err)
	}
	if !isInside(info.VideoDir, session.PackageDir) {
		return smokeReport{}, fmt.Errorf("package dir %q is outside video dir %q", session.PackageDir, info.VideoDir)
	}
	if session.Backend != recording.BackendMockPackage {
		return smokeReport{}, fmt.Errorf("backend = %q, want %q", session.Backend, recording.BackendMockPackage)
	}
	if paused, err := recorder.Pause(); err != nil || paused.Status != recording.StatePaused {
		return smokeReport{}, fmt.Errorf("pause = %#v, %w", paused, err)
	}
	if resumed, err := recorder.Resume(); err != nil || resumed.Status != recording.StateRecording {
		return smokeReport{}, fmt.Errorf("resume = %#v, %w", resumed, err)
	}
	session, err = recorder.Stop()
	if err != nil {
		return smokeReport{}, fmt.Errorf("stop recording: %w", err)
	}
	if session.Status != recording.StateReady {
		return smokeReport{}, fmt.Errorf("session status = %q, want %q", session.Status, recording.StateReady)
	}

	packageService := recpackage.NewService()
	manifest, err := packageService.ReadManifest(session.Manifest)
	if err != nil {
		return smokeReport{}, fmt.Errorf("read manifest: %w", err)
	}
	if err := verifyMockManifest(manifest, req); err != nil {
		return smokeReport{}, err
	}
	marker := filepath.Join(session.PackageDir, recpackage.MockScreenFile)
	if err := requireNonEmptyFile(marker); err != nil {
		return smokeReport{}, err
	}
	recoveries, err := recorder.ScanPackages()
	if err != nil {
		return smokeReport{}, fmt.Errorf("scan packages: %w", err)
	}
	if !hasReadyPackage(recoveries, session.PackageDir) {
		return smokeReport{}, fmt.Errorf("scan did not report ready package %q: %#v", session.PackageDir, recoveries)
	}

	return smokeReport{
		OK:              true,
		DataRoot:        info.RootDir,
		VideoDir:        info.VideoDir,
		Backend:         session.Backend,
		SourceID:        string(source.ID),
		SourceType:      string(source.Type),
		PreflightStatus: string(preflightSummary.Status),
		StorageStatus:   storage.Status,
		SessionID:       session.ID,
		PackageDir:      session.PackageDir,
		ManifestPath:    session.Manifest,
		ScreenMarker:    marker,
		ManifestStatus:  manifest.Status,
		LocaleChecked:   string(settings.LocaleEN),
		KeptDataRoot:    keep,
	}, nil
}

func verifySettings(service *settings.Service, root string) error {
	next := settings.Default()
	next.Locale = settings.LocaleEN
	next.Storage.DataRootDir = root
	next.Recording = recordingprofile.Profile{
		Quality:          recordingprofile.QualityHigh,
		FPS:              60,
		CaptureCursor:    true,
		CountdownSeconds: 3,
	}
	saved, err := service.Save(next)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	if saved.Locale != settings.LocaleEN {
		return fmt.Errorf("saved locale = %q, want %q", saved.Locale, settings.LocaleEN)
	}
	loaded, err := service.Load()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	if loaded.Locale != settings.LocaleEN {
		return fmt.Errorf("loaded locale = %q, want %q", loaded.Locale, settings.LocaleEN)
	}
	return nil
}

func selectSource(sources []devices.CaptureSource) devices.CaptureSource {
	for _, source := range sources {
		if source.Type == devices.SourceScreen {
			return source
		}
	}
	return sources[0]
}

func smokeStartRequest(source devices.CaptureSource, media devices.MediaInventory) recording.StartRequest {
	return recording.StartRequest{
		SourceID:   source.ID,
		SourceType: source.Type,
		SourceName: source.Name,
		Recording: recordingprofile.Profile{
			Quality:          recordingprofile.QualityBalanced,
			FPS:              30,
			CaptureCursor:    true,
			CountdownSeconds: 0,
		},
		Audio: recording.AudioRequest{
			System:           true,
			SystemDeviceID:   firstMediaID(media.SystemAudio),
			Microphone:       true,
			MicrophoneID:     firstMediaID(media.Microphones),
			NoiseSuppression: true,
			MicrophoneGain:   1,
		},
		Camera: recording.CameraRequest{
			Enabled:   true,
			DeviceID:  firstMediaID(media.Cameras),
			PIPPreset: "bottom-right",
		},
	}
}

func firstMediaID(devices []devices.MediaDevice) string {
	if len(devices) == 0 {
		return ""
	}
	return devices[0].ID
}

func verifyMockManifest(manifest recpackage.Manifest, req recording.StartRequest) error {
	if manifest.App != recpackage.AppName {
		return fmt.Errorf("manifest app = %q, want %q", manifest.App, recpackage.AppName)
	}
	if manifest.Status != recpackage.StatusReady {
		return fmt.Errorf("manifest status = %q, want %q", manifest.Status, recpackage.StatusReady)
	}
	if manifest.Media.ScreenVideoPath != recpackage.MockScreenFile {
		return fmt.Errorf("screen path = %q, want %q", manifest.Media.ScreenVideoPath, recpackage.MockScreenFile)
	}
	if !manifest.Diagnostics.Mock || manifest.Diagnostics.Sync == nil || manifest.Diagnostics.Sync.TimelineBase != recpackage.TimelineBaseMock {
		return fmt.Errorf("manifest mock diagnostics are incomplete: %#v", manifest.Diagnostics)
	}
	if !manifest.Audio.System || !manifest.Audio.Microphone {
		return fmt.Errorf("manifest audio flags = %#v, want system and microphone enabled", manifest.Audio)
	}
	if manifest.Audio.MicrophoneNoiseSuppression != recpackage.NoiseSuppressionOn {
		return fmt.Errorf("noise suppression = %q, want %q", manifest.Audio.MicrophoneNoiseSuppression, recpackage.NoiseSuppressionOn)
	}
	if !manifest.Audio.SystemAudioIsNeverDenoised {
		return errors.New("manifest must state system audio is never denoised")
	}
	if !manifest.Camera.Enabled || manifest.Camera.PIPPreset != req.Camera.PIPPreset {
		return fmt.Errorf("manifest camera = %#v, want enabled PIP %q", manifest.Camera, req.Camera.PIPPreset)
	}
	if manifest.CompletedAt == nil || manifest.CompletedAt.IsZero() {
		return errors.New("manifest completedAt was not written")
	}
	if time.Since(manifest.CreatedAt) < 0 {
		return errors.New("manifest createdAt is in the future")
	}
	return nil
}

func requireNonEmptyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %q: %w", path, err)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("%q is empty", path)
	}
	return nil
}

func hasReadyPackage(recoveries []recpackage.RecoverySummary, packageDir string) bool {
	for _, summary := range recoveries {
		if filepath.Clean(summary.PackageDir) == filepath.Clean(packageDir) &&
			summary.Status == recpackage.StatusReady &&
			!summary.Recoverable {
			return true
		}
	}
	return false
}

func isInside(root string, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return relative == "." || (!filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
}
