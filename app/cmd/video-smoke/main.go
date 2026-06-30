package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/preflight"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

type smokeReport struct {
	OK                       bool   `json:"ok"`
	DataRoot                 string `json:"dataRoot"`
	VideoDir                 string `json:"videoDir"`
	Backend                  string `json:"backend"`
	SourceID                 string `json:"sourceId"`
	SourceType               string `json:"sourceType"`
	PreflightStatus          string `json:"preflightStatus"`
	DurationMs               int64  `json:"durationMs"`
	SessionID                string `json:"sessionId"`
	PackageDir               string `json:"packageDir"`
	ManifestPath             string `json:"manifestPath"`
	ManifestStatus           string `json:"manifestStatus"`
	ScreenVideoPath          string `json:"screenVideoPath"`
	ScreenVideoBytes         int64  `json:"screenVideoBytes"`
	VideoDiagnosticsPath     string `json:"videoDiagnosticsPath"`
	VideoDiagnosticFrames    int64  `json:"videoDiagnosticFrames"`
	VideoDiagnosticDuration  int64  `json:"videoDiagnosticDurationMs"`
	SyncDiagnosticFrameRate  int    `json:"syncDiagnosticFrameRate"`
	SyncDiagnosticDurationMs int64  `json:"syncDiagnosticDurationMs"`
	Paused                   bool   `json:"paused"`
	KeptDataRoot             bool   `json:"keptDataRoot"`
}

type options struct {
	dataRoot      string
	backend       string
	sourceID      string
	sourceType    devices.CaptureSourceType
	duration      time.Duration
	pauseAfter    time.Duration
	pauseDuration time.Duration
	systemAudio   bool
	microphone    bool
	camera        bool
	quality       string
	fps           int
	captureCursor bool
}

func main() {
	var (
		sourceTypeFlag string
		opts           options
	)
	flag.StringVar(&opts.dataRoot, "data-dir", "", "data root; defaults to the app-managed RecordingFreedom root")
	flag.StringVar(&opts.backend, "backend", "native", "recording backend: native, screencapturekit, windows-graphics-capture, pipewire-portal, or mock-package")
	flag.StringVar(&opts.sourceID, "source-id", "", "capture source id; defaults to the first available source of -source-type")
	flag.StringVar(&sourceTypeFlag, "source-type", string(devices.SourceScreen), "capture source type: screen, window, or application")
	flag.DurationVar(&opts.duration, "duration", 5*time.Second, "recording duration")
	flag.DurationVar(&opts.pauseAfter, "pause-after", 0, "optional pause time after start; 0 disables pause/resume smoke")
	flag.DurationVar(&opts.pauseDuration, "pause-duration", time.Second, "pause duration when -pause-after is set")
	flag.BoolVar(&opts.systemAudio, "system", false, "capture system audio; disabled by default until mux support lands")
	flag.BoolVar(&opts.microphone, "microphone", false, "capture microphone; disabled by default until mux support lands")
	flag.BoolVar(&opts.camera, "camera", false, "capture camera sidecar; disabled by default until sidecar support lands")
	flag.StringVar(&opts.quality, "quality", recordingprofile.QualityBalanced, "recording quality: standard, balanced, or high")
	flag.IntVar(&opts.fps, "fps", recordingprofile.DefaultFPS, "recording fps: 24, 30, or 60")
	flag.BoolVar(&opts.captureCursor, "cursor", recordingprofile.DefaultCaptureCursor, "capture cursor")
	flag.Parse()

	sourceType, err := parseSourceType(sourceTypeFlag)
	if err != nil {
		fail(err)
	}
	opts.sourceType = sourceType

	report, err := run(opts)
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
		fail(err)
	}
}

func run(opts options) (smokeReport, error) {
	if opts.duration <= 0 {
		return smokeReport{}, errors.New("-duration must be positive")
	}
	if opts.pauseAfter < 0 || opts.pauseDuration < 0 {
		return smokeReport{}, errors.New("-pause-after and -pause-duration cannot be negative")
	}

	data := appdata.NewService(opts.dataRoot)
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
		return smokeReport{}, fmt.Errorf("storage is not ready for video smoke: %#v", storage)
	}

	deviceService := devices.NewService()
	sources := deviceService.ListSources()
	source, err := chooseSource(sources, opts.sourceID, opts.sourceType)
	if err != nil {
		return smokeReport{}, err
	}

	packages := recpackage.NewService()
	backend := recording.SelectBackend(packages, runtime.GOOS, opts.backend)
	recorder := recording.NewServiceWithBackend(data, backend)
	capabilities := capture.NewService().Capabilities()
	media := deviceService.ListMediaDevices()
	req := startRequest(source, opts)

	preflightSummary := preflight.NewService().Evaluate(req, preflight.Inputs{
		Backend:      recorder.BackendID(),
		Sources:      sources,
		Media:        media,
		Capabilities: capabilities,
		Storage:      storage,
	})
	if preflightSummary.Status == preflight.StatusBlocked {
		return smokeReport{}, fmt.Errorf("preflight blocked video smoke: %s", describeChecks(preflightSummary.Checks))
	}

	startedAt := time.Now()
	session, err := recorder.StartRecording(req)
	if err != nil {
		return smokeReport{}, fmt.Errorf("start recording: %w", err)
	}
	paused := false
	if opts.pauseAfter > 0 && opts.pauseAfter < opts.duration {
		time.Sleep(opts.pauseAfter)
		if _, err := recorder.Pause(); err != nil {
			return smokeReport{}, fmt.Errorf("pause recording: %w", err)
		}
		paused = true
		time.Sleep(opts.pauseDuration)
		if _, err := recorder.Resume(); err != nil {
			return smokeReport{}, fmt.Errorf("resume recording: %w", err)
		}
		time.Sleep(opts.duration - opts.pauseAfter)
	} else {
		time.Sleep(opts.duration)
	}
	session, err = recorder.Stop()
	if err != nil {
		return smokeReport{}, fmt.Errorf("stop recording: %w", err)
	}

	manifest, err := packages.ReadManifest(session.Manifest)
	if err != nil {
		return smokeReport{}, fmt.Errorf("read manifest: %w", err)
	}
	verification, err := verifyPackage(session, manifest)
	if err != nil {
		return smokeReport{}, err
	}

	return smokeReport{
		OK:                       true,
		DataRoot:                 info.RootDir,
		VideoDir:                 info.VideoDir,
		Backend:                  session.Backend,
		SourceID:                 source.ID,
		SourceType:               string(source.Type),
		PreflightStatus:          string(preflightSummary.Status),
		DurationMs:               time.Since(startedAt).Milliseconds(),
		SessionID:                session.ID,
		PackageDir:               session.PackageDir,
		ManifestPath:             session.Manifest,
		ManifestStatus:           manifest.Status,
		ScreenVideoPath:          verification.screenVideoPath,
		ScreenVideoBytes:         verification.screenVideoBytes,
		VideoDiagnosticsPath:     verification.videoDiagnosticsPath,
		VideoDiagnosticFrames:    verification.videoFrames,
		VideoDiagnosticDuration:  verification.videoDurationMs,
		SyncDiagnosticFrameRate:  manifest.Diagnostics.Sync.Screen.FrameRate,
		SyncDiagnosticDurationMs: manifest.Diagnostics.Sync.Screen.DurationMs,
		Paused:                   paused,
		KeptDataRoot:             true,
	}, nil
}

func startRequest(source devices.CaptureSource, opts options) recording.StartRequest {
	return recording.StartRequest{
		SourceID:   source.ID,
		SourceType: source.Type,
		SourceName: source.Name,
		Recording: recordingprofile.Profile{
			Quality:       opts.quality,
			FPS:           opts.fps,
			CaptureCursor: opts.captureCursor,
		},
		Audio: recording.AudioRequest{
			System:     opts.systemAudio,
			Microphone: opts.microphone,
		},
		Camera: recording.CameraRequest{
			Enabled:   opts.camera,
			PIPPreset: "bottom-right",
		},
	}
}

type packageVerification struct {
	screenVideoPath      string
	screenVideoBytes     int64
	videoDiagnosticsPath string
	videoFrames          int64
	videoDurationMs      int64
}

func verifyPackage(session recording.Session, manifest recpackage.Manifest) (packageVerification, error) {
	if session.Status != recording.StateReady {
		return packageVerification{}, fmt.Errorf("session status = %q, want %q", session.Status, recording.StateReady)
	}
	if manifest.Status != recpackage.StatusReady {
		return packageVerification{}, fmt.Errorf("manifest status = %q, want %q", manifest.Status, recpackage.StatusReady)
	}
	if manifest.Diagnostics.Mock {
		return packageVerification{}, errors.New("video smoke must produce real media, got mock diagnostics")
	}
	if manifest.Diagnostics.Sync == nil {
		return packageVerification{}, errors.New("manifest diagnostics.sync is missing")
	}
	if !manifest.Diagnostics.Sync.Screen.Enabled {
		return packageVerification{}, errors.New("manifest diagnostics.sync.screen is not enabled")
	}
	if manifest.Diagnostics.Sync.Screen.Path != recpackage.ScreenVideoFile {
		return packageVerification{}, fmt.Errorf("diagnostics.sync.screen.path = %q, want %q", manifest.Diagnostics.Sync.Screen.Path, recpackage.ScreenVideoFile)
	}
	if manifest.Diagnostics.Sync.Screen.FrameRate <= 0 {
		return packageVerification{}, fmt.Errorf("diagnostics.sync.screen.frameRate = %d, want > 0", manifest.Diagnostics.Sync.Screen.FrameRate)
	}
	if manifest.Diagnostics.Sync.VideoDiagnosticsPath != recpackage.VideoDiagnosticsFile {
		return packageVerification{}, fmt.Errorf("diagnostics.sync.videoDiagnosticsPath = %q, want %q", manifest.Diagnostics.Sync.VideoDiagnosticsPath, recpackage.VideoDiagnosticsFile)
	}
	if manifest.Media.ScreenVideoPath != recpackage.ScreenVideoFile {
		return packageVerification{}, fmt.Errorf("screenVideoPath = %q, want %q", manifest.Media.ScreenVideoPath, recpackage.ScreenVideoFile)
	}

	screenPath := filepath.Join(session.PackageDir, filepath.Clean(manifest.Media.ScreenVideoPath))
	screenInfo, err := os.Stat(screenPath)
	if err != nil {
		return packageVerification{}, fmt.Errorf("stat screen media: %w", err)
	}
	if screenInfo.IsDir() || screenInfo.Size() <= 0 {
		return packageVerification{}, fmt.Errorf("screen media %q is empty or not a file", screenPath)
	}

	diagnosticsPath := filepath.Join(session.PackageDir, recpackage.VideoDiagnosticsFile)
	diagnostics, err := readVideoDiagnostics(diagnosticsPath)
	if err != nil {
		return packageVerification{}, err
	}
	if !diagnostics.Screen.Enabled {
		return packageVerification{}, errors.New("video diagnostics screen track is not enabled")
	}
	if diagnostics.Screen.FramesWritten <= 0 {
		return packageVerification{}, fmt.Errorf("video diagnostics framesWritten = %d, want > 0", diagnostics.Screen.FramesWritten)
	}

	return packageVerification{
		screenVideoPath:      screenPath,
		screenVideoBytes:     screenInfo.Size(),
		videoDiagnosticsPath: diagnosticsPath,
		videoFrames:          diagnostics.Screen.FramesWritten,
		videoDurationMs:      diagnostics.Screen.DurationMs,
	}, nil
}

func readVideoDiagnostics(path string) (video.Diagnostics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return video.Diagnostics{}, fmt.Errorf("read video diagnostics: %w", err)
	}
	var diagnostics video.Diagnostics
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		return video.Diagnostics{}, fmt.Errorf("decode video diagnostics: %w", err)
	}
	return diagnostics, nil
}

func chooseSource(sources []devices.CaptureSource, sourceID string, sourceType devices.CaptureSourceType) (devices.CaptureSource, error) {
	sourceID = strings.TrimSpace(sourceID)
	var fallback *devices.CaptureSource
	for index := range sources {
		source := sources[index]
		if source.Type != sourceType {
			continue
		}
		if sourceID != "" {
			if source.ID == sourceID {
				return source, nil
			}
			continue
		}
		if source.Available {
			return source, nil
		}
		if fallback == nil {
			fallback = &sources[index]
		}
	}
	if sourceID != "" {
		return devices.CaptureSource{}, fmt.Errorf("source %q of type %q was not returned by DeviceService", sourceID, sourceType)
	}
	if fallback != nil {
		return *fallback, nil
	}
	return devices.CaptureSource{}, fmt.Errorf("no source of type %q was returned by DeviceService", sourceType)
}

func parseSourceType(value string) (devices.CaptureSourceType, error) {
	switch devices.CaptureSourceType(strings.TrimSpace(value)) {
	case devices.SourceScreen:
		return devices.SourceScreen, nil
	case devices.SourceWindow:
		return devices.SourceWindow, nil
	case devices.SourceApplication:
		return devices.SourceApplication, nil
	default:
		return "", fmt.Errorf("unsupported source type %q", value)
	}
}

func describeChecks(checks []preflight.Check) string {
	if len(checks) == 0 {
		return "no checks returned"
	}
	parts := make([]string, 0, len(checks))
	for _, check := range checks {
		if check.Status == preflight.StatusReady {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s (%s)", check.ID, check.Status, check.Reason))
	}
	if len(parts) == 0 {
		return "all checks ready"
	}
	return strings.Join(parts, "; ")
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "video-smoke failed: %v\n", err)
	os.Exit(1)
}
