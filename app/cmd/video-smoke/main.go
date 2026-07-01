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
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
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
	SystemAudioPath          string `json:"systemAudioPath,omitempty"`
	SystemAudioStorage       string `json:"systemAudioStorage,omitempty"`
	SystemAudioBytes         int64  `json:"systemAudioBytes,omitempty"`
	SystemAudioSamples       int64  `json:"systemAudioSamples,omitempty"`
	MicrophoneAudioPath      string `json:"microphoneAudioPath,omitempty"`
	MicrophoneAudioStorage   string `json:"microphoneAudioStorage,omitempty"`
	MicrophoneAudioBytes     int64  `json:"microphoneAudioBytes,omitempty"`
	MicrophoneSamples        int64  `json:"microphoneSamples,omitempty"`
	AudioDiagnosticsPath     string `json:"audioDiagnosticsPath,omitempty"`
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
	regionX       int
	regionY       int
	regionWidth   int
	regionHeight  int
	regionNative  string
	keep          bool
}

func main() {
	var (
		sourceTypeFlag string
		opts           options
	)
	flag.StringVar(&opts.dataRoot, "data-dir", "", "data root; defaults to the app-managed RecordingFreedom root")
	flag.StringVar(&opts.backend, "backend", "native", "recording backend: native, screencapturekit, ffmpeg-desktop-capture, windows-graphics-capture, pipewire-portal, or mock-package")
	flag.StringVar(&opts.sourceID, "source-id", "", "capture source id; defaults to the first available source of -source-type")
	flag.StringVar(&sourceTypeFlag, "source-type", string(devices.SourceScreen), "capture source type: screen, all-screens, region, window, or application")
	flag.DurationVar(&opts.duration, "duration", 5*time.Second, "recording duration")
	flag.DurationVar(&opts.pauseAfter, "pause-after", 0, "optional pause time after start; 0 disables pause/resume smoke")
	flag.DurationVar(&opts.pauseDuration, "pause-duration", time.Second, "pause duration when -pause-after is set")
	flag.BoolVar(&opts.systemAudio, "system", false, "capture system audio into the primary media when the backend supports muxing")
	flag.BoolVar(&opts.microphone, "microphone", false, "capture microphone; disabled by default until mux support lands")
	flag.BoolVar(&opts.camera, "camera", false, "capture camera sidecar when the platform reports an available native camera writer")
	flag.StringVar(&opts.quality, "quality", recordingprofile.QualityBalanced, "recording quality: standard, balanced, or high")
	flag.IntVar(&opts.fps, "fps", recordingprofile.DefaultFPS, "recording fps: 24, 30, or 60")
	flag.BoolVar(&opts.captureCursor, "cursor", recordingprofile.DefaultCaptureCursor, "capture cursor")
	flag.IntVar(&opts.regionX, "region-x", 0, "region source x coordinate; used with -source-type=region")
	flag.IntVar(&opts.regionY, "region-y", 0, "region source y coordinate; used with -source-type=region")
	flag.IntVar(&opts.regionWidth, "region-width", 0, "region source width; used with -source-type=region")
	flag.IntVar(&opts.regionHeight, "region-height", 0, "region source height; used with -source-type=region")
	flag.StringVar(&opts.regionNative, "region-native-id", "", "native display id for region source, for example cgdisplay:<id>")
	flag.BoolVar(&opts.keep, "keep", false, "accepted for consistency with other smoke commands; video-smoke keeps app-managed or explicit data roots")
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
	req, err := startRequest(source, sources, media, opts)
	if err != nil {
		return smokeReport{}, err
	}

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
		SystemAudioPath:          verification.systemAudioPath,
		SystemAudioStorage:       manifest.Media.SystemAudioStorage,
		SystemAudioBytes:         verification.systemAudioBytes,
		SystemAudioSamples:       verification.systemAudioSamples,
		MicrophoneAudioPath:      verification.microphoneAudioPath,
		MicrophoneAudioStorage:   manifest.Media.MicrophoneAudioStorage,
		MicrophoneAudioBytes:     verification.microphoneAudioBytes,
		MicrophoneSamples:        verification.microphoneSamples,
		AudioDiagnosticsPath:     verification.audioDiagnosticsPath,
		VideoDiagnosticsPath:     verification.videoDiagnosticsPath,
		VideoDiagnosticFrames:    verification.videoFrames,
		VideoDiagnosticDuration:  verification.videoDurationMs,
		SyncDiagnosticFrameRate:  manifest.Diagnostics.Sync.Screen.FrameRate,
		SyncDiagnosticDurationMs: manifest.Diagnostics.Sync.Screen.DurationMs,
		Paused:                   paused,
		KeptDataRoot:             true,
	}, nil
}

func startRequest(source devices.CaptureSource, sources []devices.CaptureSource, media devices.MediaInventory, opts options) (recording.StartRequest, error) {
	req := recording.StartRequest{
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
	if opts.camera {
		camera, ok := firstSmokeCamera(media.Cameras)
		if !ok {
			return recording.StartRequest{}, fmt.Errorf("camera sidecar requested but no available sidecar camera was returned by DeviceService")
		}
		req.Camera.DeviceID = camera.ID
		req.Camera.DeviceNativeID = camera.NativeID
	}
	if source.Type == devices.SourceRegion {
		geometry, err := regionGeometryForSmoke(sources, opts)
		if err != nil {
			return recording.StartRequest{}, err
		}
		req.SourceGeometry = &geometry
	} else if source.Width > 0 && source.Height > 0 {
		req.SourceGeometry = &recording.SourceGeometry{
			X:            source.X,
			Y:            source.Y,
			Width:        source.Width,
			Height:       source.Height,
			DisplayIndex: source.DisplayIndex,
			NativeID:     source.NativeID,
		}
	}
	return req, nil
}

func firstSmokeCamera(cameras []devices.MediaDevice) (devices.MediaDevice, bool) {
	for _, camera := range cameras {
		if camera.Available && camera.SidecarEligible && strings.TrimSpace(camera.NativeID) != "" {
			return camera, true
		}
	}
	return devices.MediaDevice{}, false
}

func regionGeometryForSmoke(sources []devices.CaptureSource, opts options) (recording.SourceGeometry, error) {
	if opts.regionWidth > 0 && opts.regionHeight > 0 {
		return recording.SourceGeometry{
			X:        opts.regionX,
			Y:        opts.regionY,
			Width:    opts.regionWidth,
			Height:   opts.regionHeight,
			NativeID: strings.TrimSpace(opts.regionNative),
		}, nil
	}
	for _, source := range sources {
		if source.Type != devices.SourceScreen || !source.Available || source.Width <= 0 || source.Height <= 0 {
			continue
		}
		width := minInt(640, source.Width)
		height := minInt(360, source.Height)
		return recording.SourceGeometry{
			X:            source.X + minInt(64, maxInt(0, source.Width-width)),
			Y:            source.Y + minInt(64, maxInt(0, source.Height-height)),
			Width:        width,
			Height:       height,
			DisplayIndex: source.DisplayIndex,
			NativeID:     source.NativeID,
		}, nil
	}
	return recording.SourceGeometry{}, errors.New("region smoke requires -region-width/-region-height or an available screen source")
}

type packageVerification struct {
	screenVideoPath      string
	screenVideoBytes     int64
	systemAudioPath      string
	systemAudioBytes     int64
	systemAudioSamples   int64
	microphoneAudioPath  string
	microphoneAudioBytes int64
	microphoneSamples    int64
	audioDiagnosticsPath string
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

	verification := packageVerification{
		screenVideoPath:      screenPath,
		screenVideoBytes:     screenInfo.Size(),
		videoDiagnosticsPath: diagnosticsPath,
		videoFrames:          diagnostics.Screen.FramesWritten,
		videoDurationMs:      diagnostics.Screen.DurationMs,
	}

	audioDiagnostics, hasAudioDiagnostics, err := verifyAudioTracks(session.PackageDir, manifest)
	if err != nil {
		return packageVerification{}, err
	}
	if hasAudioDiagnostics {
		verification.audioDiagnosticsPath = filepath.Join(session.PackageDir, recpackage.AudioDiagnosticsFile)
	}
	if manifest.Audio.System {
		mediaPath, bytes, err := verifyManifestAudioTrack(session.PackageDir, "systemAudio", manifest.Media.SystemAudioPath, manifest.Media.SystemAudioStorage, manifest.Media.ScreenVideoPath, manifest.Diagnostics.Sync.SystemAudio)
		if err != nil {
			return packageVerification{}, err
		}
		verification.systemAudioPath = mediaPath
		verification.systemAudioBytes = bytes
		if hasAudioDiagnostics {
			verification.systemAudioSamples = audioDiagnostics.SystemAudio.SamplesWritten
		}
	}
	if manifest.Audio.Microphone {
		mediaPath, bytes, err := verifyManifestAudioTrack(session.PackageDir, "microphoneAudio", manifest.Media.MicrophoneAudioPath, manifest.Media.MicrophoneAudioStorage, manifest.Media.ScreenVideoPath, manifest.Diagnostics.Sync.Microphone)
		if err != nil {
			return packageVerification{}, err
		}
		verification.microphoneAudioPath = mediaPath
		verification.microphoneAudioBytes = bytes
		if hasAudioDiagnostics {
			verification.microphoneSamples = audioDiagnostics.Microphone.SamplesWritten
		}
	}
	return verification, nil
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

func verifyAudioTracks(packageDir string, manifest recpackage.Manifest) (audio.Diagnostics, bool, error) {
	path := filepath.Join(packageDir, recpackage.AudioDiagnosticsFile)
	diagnostics, err := readAudioDiagnostics(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) &&
			manifest.Media.SystemAudioStorage != recpackage.AudioStorageSidecar &&
			manifest.Media.MicrophoneAudioStorage != recpackage.AudioStorageSidecar {
			return audio.Diagnostics{}, false, nil
		}
		return audio.Diagnostics{}, false, err
	}
	if manifest.Audio.System && diagnostics.SystemAudio.SamplesWritten <= 0 {
		return audio.Diagnostics{}, false, fmt.Errorf("audio diagnostics systemAudio.samplesWritten = %d, want > 0", diagnostics.SystemAudio.SamplesWritten)
	}
	if manifest.Audio.Microphone && diagnostics.Microphone.SamplesWritten <= 0 {
		return audio.Diagnostics{}, false, fmt.Errorf("audio diagnostics microphone.samplesWritten = %d, want > 0", diagnostics.Microphone.SamplesWritten)
	}
	return diagnostics, true, nil
}

func verifyManifestAudioTrack(packageDir string, label string, mediaPath string, storage string, primaryPath string, syncTrack recpackage.ManifestTrackDiagnostics) (string, int64, error) {
	if !syncTrack.Enabled {
		return "", 0, fmt.Errorf("manifest diagnostics.sync.%s is not enabled", label)
	}
	if syncTrack.Path != mediaPath {
		return "", 0, fmt.Errorf("diagnostics.sync.%s.path = %q, want %q", label, syncTrack.Path, mediaPath)
	}
	switch storage {
	case recpackage.AudioStorageSidecar:
		path := filepath.Join(packageDir, filepath.Clean(mediaPath))
		info, err := os.Stat(path)
		if err != nil {
			return "", 0, fmt.Errorf("stat %s sidecar: %w", label, err)
		}
		if info.IsDir() || info.Size() <= 45 {
			return "", 0, fmt.Errorf("%s sidecar %q is empty or not readable media", label, path)
		}
		return path, info.Size(), nil
	case recpackage.AudioStorageMuxed:
		if filepath.Clean(mediaPath) != filepath.Clean(primaryPath) {
			return "", 0, fmt.Errorf("%s muxed path %q must match screenVideoPath %q", label, mediaPath, primaryPath)
		}
		path := filepath.Join(packageDir, filepath.Clean(primaryPath))
		info, err := os.Stat(path)
		if err != nil {
			return "", 0, fmt.Errorf("stat %s muxed media: %w", label, err)
		}
		return path, info.Size(), nil
	default:
		return "", 0, fmt.Errorf("%s storage %q is not supported", label, storage)
	}
}

func readAudioDiagnostics(path string) (audio.Diagnostics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return audio.Diagnostics{}, fmt.Errorf("read audio diagnostics: %w", err)
	}
	var diagnostics audio.Diagnostics
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		return audio.Diagnostics{}, fmt.Errorf("decode audio diagnostics: %w", err)
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
	case devices.SourceAllScreens:
		return devices.SourceAllScreens, nil
	case devices.SourceRegion:
		return devices.SourceRegion, nil
	case devices.SourceWindow:
		return devices.SourceWindow, nil
	case devices.SourceApplication:
		return devices.SourceApplication, nil
	default:
		return "", fmt.Errorf("unsupported source type %q", value)
	}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
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
