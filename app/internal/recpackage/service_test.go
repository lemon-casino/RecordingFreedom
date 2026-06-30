package recpackage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestCreateMockWritesRelativeManifestAndMarker(t *testing.T) {
	root := t.TempDir()
	createdAt := time.Date(2026, 6, 30, 14, 30, 1, 123000000, time.UTC)
	service := NewService()

	pkg, err := service.CreateMock(root, CreateMockRequest{
		CreatedAt: createdAt,
		Source:    ManifestSource{Type: "screen", ID: "screen:primary", Name: "Primary Display"},
		Recording: recordingprofile.Profile{
			Quality:          recordingprofile.QualityHigh,
			FPS:              60,
			CaptureCursor:    true,
			CountdownSeconds: 3,
		},
		Audio: ManifestAudio{
			System:                     true,
			SystemDeviceID:             "system-audio:default",
			Microphone:                 true,
			MicrophoneDeviceID:         "microphone:default",
			MicrophoneNoiseSuppression: NoiseSuppressionOn,
			MicrophoneGain:             1,
			MockPipeline:               true,
		},
		Camera: ManifestCamera{Enabled: true, DeviceID: "camera:default", PIPPreset: "bottom-right"},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	if filepath.Dir(pkg.Dir) != root {
		t.Fatalf("package parent = %q, want %q", filepath.Dir(pkg.Dir), root)
	}
	if pkg.ID != "2026-06-30-14-30-01-123" {
		t.Fatalf("package ID = %q", pkg.ID)
	}
	if _, err := os.Stat(filepath.Join(pkg.Dir, MockScreenFile)); err != nil {
		t.Fatalf("mock marker was not created: %v", err)
	}

	manifest, err := service.ReadManifest(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Media.ScreenVideoPath != MockScreenFile {
		t.Fatalf("screen path = %q, want %q", manifest.Media.ScreenVideoPath, MockScreenFile)
	}
	if filepath.IsAbs(manifest.Media.ScreenVideoPath) {
		t.Fatalf("screen path must be relative: %q", manifest.Media.ScreenVideoPath)
	}
	if manifest.Audio.MicrophoneNoiseSuppression != NoiseSuppressionOn {
		t.Fatalf("noise suppression = %q", manifest.Audio.MicrophoneNoiseSuppression)
	}
	if manifest.Recording.Quality != recordingprofile.QualityHigh || manifest.Recording.FPS != 60 || !manifest.Recording.CaptureCursor || manifest.Recording.CountdownSeconds != 3 {
		t.Fatalf("recording profile = %#v", manifest.Recording)
	}
	if manifest.Audio.SystemDeviceID != "system-audio:default" {
		t.Fatalf("system device id = %q, want system-audio:default", manifest.Audio.SystemDeviceID)
	}
	if !manifest.Diagnostics.Mock {
		t.Fatal("mock package must be marked as mock in diagnostics")
	}
	if manifest.Diagnostics.Sync == nil {
		t.Fatal("mock package must include sync diagnostics contract")
	}
	if manifest.Diagnostics.Sync.TimelineBase != TimelineBaseMock {
		t.Fatalf("sync timeline base = %q, want %q", manifest.Diagnostics.Sync.TimelineBase, TimelineBaseMock)
	}
	if !manifest.Diagnostics.Sync.Screen.Enabled || manifest.Diagnostics.Sync.Screen.Path != MockScreenFile {
		t.Fatalf("screen sync diagnostics = %#v, want enabled mock screen path", manifest.Diagnostics.Sync.Screen)
	}
	if manifest.Diagnostics.Sync.Screen.FrameRate != 60 {
		t.Fatalf("screen sync frame rate = %d, want 60", manifest.Diagnostics.Sync.Screen.FrameRate)
	}
	if !manifest.Diagnostics.Sync.SystemAudio.Enabled || manifest.Diagnostics.Sync.SystemAudio.SampleRate != 48000 {
		t.Fatalf("system audio sync diagnostics = %#v, want enabled 48kHz", manifest.Diagnostics.Sync.SystemAudio)
	}
	if !manifest.Diagnostics.Sync.Microphone.Enabled || manifest.Diagnostics.Sync.Microphone.SampleRate != 48000 {
		t.Fatalf("microphone sync diagnostics = %#v, want enabled 48kHz", manifest.Diagnostics.Sync.Microphone)
	}
	if !manifest.Diagnostics.Sync.Webcam.Enabled {
		t.Fatalf("webcam sync diagnostics = %#v, want enabled mock sidecar contract", manifest.Diagnostics.Sync.Webcam)
	}
}

func TestPatchStatusWritesCompletedAt(t *testing.T) {
	service := NewService()
	pkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	completedAt := time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC)
	if err := service.PatchStatus(pkg.ManifestPath, StatusReady, &completedAt); err != nil {
		t.Fatalf("PatchStatus() error = %v", err)
	}
	manifest, err := service.ReadManifest(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != StatusReady {
		t.Fatalf("status = %q, want %q", manifest.Status, StatusReady)
	}
	if manifest.CompletedAt == nil || !manifest.CompletedAt.Equal(completedAt) {
		t.Fatalf("completedAt = %v, want %v", manifest.CompletedAt, completedAt)
	}
}

func TestCreateNativeInitializesWritePlanWithoutCreatingMedia(t *testing.T) {
	root := t.TempDir()
	createdAt := time.Date(2026, 6, 30, 18, 30, 0, 456000000, time.UTC)
	service := NewService()

	plan, err := service.CreateNative(root, CreateNativeRequest{
		CreatedAt: createdAt,
		Backend:   "screencapturekit",
		Source:    ManifestSource{Type: "screen", ID: "cgdisplay:1", Name: "Built-in Display"},
		Recording: recordingprofile.Profile{Quality: recordingprofile.QualityHigh, FPS: 60, CaptureCursor: true},
		Audio: ManifestAudio{
			System:                     true,
			Microphone:                 true,
			MicrophoneDeviceID:         "microphone:default",
			MicrophoneNoiseSuppression: NoiseSuppressionOn,
		},
		Camera: ManifestCamera{Enabled: true, DeviceID: "camera:default", PIPPreset: "bottom-left"},
	})
	if err != nil {
		t.Fatalf("CreateNative() error = %v", err)
	}
	if filepath.Dir(plan.Package.Dir) != root {
		t.Fatalf("package parent = %q, want %q", filepath.Dir(plan.Package.Dir), root)
	}
	if plan.Package.ID != "2026-06-30-18-30-00-456" {
		t.Fatalf("package ID = %q, want timestamp ID", plan.Package.ID)
	}
	if plan.ScreenVideoPath != filepath.Join(plan.Package.Dir, ScreenVideoFile) {
		t.Fatalf("screen write path = %q, want package screen file", plan.ScreenVideoPath)
	}
	if plan.WebcamVideoPath != filepath.Join(plan.Package.Dir, WebcamVideoFile) {
		t.Fatalf("webcam write path = %q, want package webcam file", plan.WebcamVideoPath)
	}
	if plan.AudioDiagnosticsPath != filepath.Join(plan.Package.Dir, AudioDiagnosticsFile) {
		t.Fatalf("audio diagnostics path = %q, want package diagnostics file", plan.AudioDiagnosticsPath)
	}
	if _, err := os.Stat(plan.CacheDir); err != nil {
		t.Fatalf("cache dir was not created: %v", err)
	}
	if _, err := os.Stat(plan.ExportsDir); err != nil {
		t.Fatalf("exports dir was not created: %v", err)
	}
	if _, err := os.Stat(plan.ScreenVideoPath); err == nil {
		t.Fatal("CreateNative() created screen media before native backend wrote samples")
	}
	if _, err := os.Stat(plan.WebcamVideoPath); err == nil {
		t.Fatal("CreateNative() created webcam media before native backend wrote samples")
	}

	manifest, err := service.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != StatusRecording {
		t.Fatalf("status = %q, want recording", manifest.Status)
	}
	if manifest.Media.ScreenVideoPath != ScreenVideoFile || manifest.Media.WebcamVideoPath != WebcamVideoFile {
		t.Fatalf("media paths = %#v, want native package defaults", manifest.Media)
	}
	if manifest.Diagnostics.Mock {
		t.Fatal("native package must not be marked as mock")
	}
	if manifest.Diagnostics.Sync != nil {
		t.Fatalf("native package should not invent sync diagnostics before samples exist: %#v", manifest.Diagnostics.Sync)
	}
	if manifest.Audio.MicrophoneNoiseSuppression != NoiseSuppressionOn || manifest.Audio.SampleRate != 48000 {
		t.Fatalf("audio manifest = %#v, want RNNoise 48kHz contract", manifest.Audio)
	}
	if manifest.Camera.PIPPreset != "bottom-left" {
		t.Fatalf("camera pip preset = %q, want bottom-left", manifest.Camera.PIPPreset)
	}
}

func TestCreateNativeOmitsWebcamPlanWhenCameraDisabled(t *testing.T) {
	service := NewService()
	plan, err := service.CreateNative(t.TempDir(), CreateNativeRequest{
		Source: ManifestSource{Type: "screen", ID: "cgdisplay:1"},
		Camera: ManifestCamera{Enabled: false, DeviceID: "camera:default", PIPPreset: "bottom-right"},
	})
	if err != nil {
		t.Fatalf("CreateNative() error = %v", err)
	}
	if plan.WebcamVideoPath != "" {
		t.Fatalf("disabled camera write path = %q, want empty", plan.WebcamVideoPath)
	}
	manifest, err := service.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Media.WebcamVideoPath != "" || manifest.Media.WebcamStartOffsetMs != 0 {
		t.Fatalf("disabled camera kept webcam media: %#v", manifest.Media)
	}
	if manifest.Camera.DeviceID != "" || manifest.Camera.PIPPreset != "off" {
		t.Fatalf("disabled camera manifest = %#v, want off without device", manifest.Camera)
	}
}

func TestValidateReadyAcceptsMockPackageMarker(t *testing.T) {
	service := NewService()
	pkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Status: StatusFinalizing,
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
		Camera: ManifestCamera{Enabled: true, DeviceID: "camera:default", PIPPreset: "bottom-right"},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	if err := service.ValidateReady(pkg.ManifestPath); err != nil {
		t.Fatalf("ValidateReady(mock) error = %v", err)
	}
}

func TestValidateReadyRejectsNativePackageWithoutReadableScreen(t *testing.T) {
	service := NewService()
	plan, err := service.CreateNative(t.TempDir(), CreateNativeRequest{
		Backend: "screencapturekit",
		Source:  ManifestSource{Type: "screen", ID: "cgdisplay:1"},
	})
	if err != nil {
		t.Fatalf("CreateNative() error = %v", err)
	}

	if err := service.ValidateReady(plan.Package.ManifestPath); err == nil || !strings.Contains(err.Error(), "screenVideoPath") {
		t.Fatalf("ValidateReady(missing screen) error = %v, want screenVideoPath error", err)
	}
	if err := os.WriteFile(plan.ScreenVideoPath, nil, 0o644); err != nil {
		t.Fatalf("WriteFile(empty screen) error = %v", err)
	}
	if err := service.ValidateReady(plan.Package.ManifestPath); err == nil || !strings.Contains(err.Error(), "not readable media") {
		t.Fatalf("ValidateReady(empty screen) error = %v, want readable media error", err)
	}
	if err := os.WriteFile(plan.ScreenVideoPath, []byte("real screen media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen) error = %v", err)
	}
	if err := service.ValidateReady(plan.Package.ManifestPath); err != nil {
		t.Fatalf("ValidateReady(screen media) error = %v", err)
	}
}

func TestValidateReadyRejectsNonMockMarkerAsNativeMedia(t *testing.T) {
	service := NewService()
	packageDir := filepath.Join(t.TempDir(), "recording-marker"+PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(package) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, MockScreenFile), []byte("mock marker"), 0o644); err != nil {
		t.Fatalf("WriteFile(mock marker) error = %v", err)
	}
	manifestPath := filepath.Join(packageDir, ManifestFile)
	if err := service.WriteManifest(manifestPath, Manifest{
		SchemaVersion: 1,
		App:           AppName,
		Status:        StatusFinalizing,
		Media:         ManifestMedia{ScreenVideoPath: MockScreenFile},
		Camera:        ManifestCamera{PIPPreset: "off"},
	}); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}

	if err := service.ValidateReady(manifestPath); err == nil || !strings.Contains(err.Error(), "mock marker") {
		t.Fatalf("ValidateReady(non-mock marker) error = %v, want mock marker rejection", err)
	}
}

func TestValidateReadyRequiresCameraSidecarWhenCameraEnabled(t *testing.T) {
	service := NewService()
	plan, err := service.CreateNative(t.TempDir(), CreateNativeRequest{
		Backend: "screencapturekit",
		Source:  ManifestSource{Type: "screen", ID: "cgdisplay:1"},
		Camera:  ManifestCamera{Enabled: true, DeviceID: "camera:default", PIPPreset: "bottom-right"},
	})
	if err != nil {
		t.Fatalf("CreateNative() error = %v", err)
	}
	if err := os.WriteFile(plan.ScreenVideoPath, []byte("real screen media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen) error = %v", err)
	}

	if err := service.ValidateReady(plan.Package.ManifestPath); err == nil || !strings.Contains(err.Error(), "webcamVideoPath") {
		t.Fatalf("ValidateReady(missing webcam) error = %v, want webcamVideoPath error", err)
	}
	if err := os.WriteFile(plan.WebcamVideoPath, nil, 0o644); err != nil {
		t.Fatalf("WriteFile(empty webcam) error = %v", err)
	}
	if err := service.ValidateReady(plan.Package.ManifestPath); err == nil || !strings.Contains(err.Error(), "not readable media") {
		t.Fatalf("ValidateReady(empty webcam) error = %v, want readable media error", err)
	}
	if err := os.WriteFile(plan.WebcamVideoPath, []byte("real webcam media"), 0o644); err != nil {
		t.Fatalf("WriteFile(webcam) error = %v", err)
	}
	if err := service.ValidateReady(plan.Package.ManifestPath); err != nil {
		t.Fatalf("ValidateReady(webcam sidecar) error = %v", err)
	}
}

func TestPatchSyncDiagnosticsWritesRelativeTrackDiagnostics(t *testing.T) {
	service := NewService()
	pkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
		Audio:  ManifestAudio{System: true, Microphone: true},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	diagnostics := ManifestSyncDiagnostics{
		TimelineBase:          TimelineBaseMedia,
		TimelineStartUnixNano: time.Date(2026, 6, 30, 15, 30, 0, 0, time.UTC).UnixNano(),
		AudioDiagnosticsPath:  AudioDiagnosticsFile,
		VideoDiagnosticsPath:  filepath.Join("diagnostics", VideoDiagnosticsFile),
		Screen: ManifestTrackDiagnostics{
			Enabled:       true,
			Path:          MockScreenFile,
			Clock:         TimelineBaseMedia,
			EndOffsetMs:   1000,
			DurationMs:    1000,
			DroppedFrames: 1,
			FrameRate:     30,
		},
		SystemAudio: ManifestTrackDiagnostics{
			Enabled:     true,
			Path:        MockScreenFile,
			Clock:       TimelineBaseMedia,
			EndOffsetMs: 1000,
			DurationMs:  1000,
			SampleRate:  48000,
		},
		Microphone: ManifestTrackDiagnostics{
			Enabled:        true,
			Path:           MockScreenFile,
			Clock:          TimelineBaseMedia,
			EndOffsetMs:    1000,
			DurationMs:     1000,
			DroppedSamples: 480,
			SampleRate:     48000,
		},
		PauseSegments: []ManifestPauseSegment{{StartOffsetMs: 400, EndOffsetMs: 650, DurationMs: 250}},
	}
	if err := service.PatchSyncDiagnostics(pkg.ManifestPath, diagnostics); err != nil {
		t.Fatalf("PatchSyncDiagnostics() error = %v", err)
	}

	manifest, err := service.ReadManifest(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Diagnostics.Sync == nil {
		t.Fatal("sync diagnostics were not written")
	}
	if manifest.Diagnostics.Sync.AudioDiagnosticsPath != AudioDiagnosticsFile {
		t.Fatalf("audio diagnostics path = %q, want %q", manifest.Diagnostics.Sync.AudioDiagnosticsPath, AudioDiagnosticsFile)
	}
	if manifest.Diagnostics.Sync.VideoDiagnosticsPath != filepath.Join("diagnostics", VideoDiagnosticsFile) {
		t.Fatalf("video diagnostics path = %q, want nested video diagnostics", manifest.Diagnostics.Sync.VideoDiagnosticsPath)
	}
	if manifest.Diagnostics.Sync.Screen.DroppedFrames != 1 {
		t.Fatalf("screen dropped frames = %d, want 1", manifest.Diagnostics.Sync.Screen.DroppedFrames)
	}
	if len(manifest.Diagnostics.Sync.PauseSegments) != 1 || manifest.Diagnostics.Sync.PauseSegments[0].DurationMs != 250 {
		t.Fatalf("pause segments = %#v, want one 250ms segment", manifest.Diagnostics.Sync.PauseSegments)
	}

	diagnostics.Screen.Path = "../screen.mp4"
	if err := service.PatchSyncDiagnostics(pkg.ManifestPath, diagnostics); err == nil {
		t.Fatal("PatchSyncDiagnostics() accepted an escaping track path")
	}
}

func TestCreateMockNormalizesCameraPIPPreset(t *testing.T) {
	service := NewService()
	disabledPkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
		Camera: ManifestCamera{Enabled: false, DeviceID: "camera:default", PIPPreset: "bottom-left"},
	})
	if err != nil {
		t.Fatalf("CreateMock(disabled camera) error = %v", err)
	}
	disabledManifest, err := service.ReadManifest(disabledPkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest(disabled) error = %v", err)
	}
	if disabledManifest.Camera.PIPPreset != "off" {
		t.Fatalf("disabled camera pip preset = %q, want off", disabledManifest.Camera.PIPPreset)
	}
	if disabledManifest.Camera.DeviceID != "" {
		t.Fatalf("disabled camera kept device id %q", disabledManifest.Camera.DeviceID)
	}

	enabledPkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
		Camera: ManifestCamera{Enabled: true},
	})
	if err != nil {
		t.Fatalf("CreateMock(enabled camera) error = %v", err)
	}
	enabledManifest, err := service.ReadManifest(enabledPkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest(enabled) error = %v", err)
	}
	if enabledManifest.Camera.PIPPreset != "bottom-right" {
		t.Fatalf("enabled camera pip preset = %q, want bottom-right", enabledManifest.Camera.PIPPreset)
	}

	invalidPkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
		Camera: ManifestCamera{Enabled: true, PIPPreset: "top-right"},
	})
	if err != nil {
		t.Fatalf("CreateMock(invalid pip) error = %v", err)
	}
	invalidManifest, err := service.ReadManifest(invalidPkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest(invalid) error = %v", err)
	}
	if invalidManifest.Camera.PIPPreset != "bottom-right" {
		t.Fatalf("invalid pip preset normalized to %q, want bottom-right", invalidManifest.Camera.PIPPreset)
	}
}

func TestCreateMockNormalizesDisabledAudioDevices(t *testing.T) {
	service := NewService()
	pkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
		Audio: ManifestAudio{
			System:                     false,
			SystemDeviceID:             "system-audio:default",
			Microphone:                 false,
			MicrophoneDeviceID:         "microphone:default",
			MicrophoneNoiseSuppression: NoiseSuppressionOn,
			MicrophoneGain:             2,
		},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}
	manifest, err := service.ReadManifest(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Audio.SystemDeviceID != "" {
		t.Fatalf("disabled system audio kept device id %q", manifest.Audio.SystemDeviceID)
	}
	if manifest.Audio.MicrophoneDeviceID != "" || manifest.Audio.MicrophoneNoiseSuppression != NoiseSuppressionOff || manifest.Audio.MicrophoneGain != 0 {
		t.Fatalf("disabled microphone was not normalized: %#v", manifest.Audio)
	}
}

func TestCreateMockNormalizesRecordingProfile(t *testing.T) {
	service := NewService()
	pkg, err := service.CreateMock(t.TempDir(), CreateMockRequest{
		Source:    ManifestSource{Type: "screen", ID: "screen:primary"},
		Recording: recordingprofile.Profile{Quality: "cinema", FPS: 120, CountdownSeconds: 99},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}
	manifest, err := service.ReadManifest(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Recording.Quality != recordingprofile.DefaultQuality || manifest.Recording.FPS != recordingprofile.DefaultFPS || manifest.Recording.CountdownSeconds != recordingprofile.MaxCountdownSeconds {
		t.Fatalf("recording profile = %#v", manifest.Recording)
	}
}

func TestWriteManifestRejectsEscapingMediaPath(t *testing.T) {
	service := NewService()
	err := service.WriteManifest(filepath.Join(t.TempDir(), ManifestFile), Manifest{
		SchemaVersion: 1,
		App:           AppName,
		Status:        StatusRecording,
		Media:         ManifestMedia{ScreenVideoPath: "../screen.mp4"},
	})
	if err == nil {
		t.Fatal("WriteManifest() accepted an escaping media path")
	}
}

func TestWriteManifestNormalizesDisabledCameraSyncDiagnostics(t *testing.T) {
	service := NewService()
	manifestPath := filepath.Join(t.TempDir(), ManifestFile)
	if err := service.WriteManifest(manifestPath, Manifest{
		SchemaVersion: 1,
		App:           AppName,
		Status:        StatusRecording,
		Media: ManifestMedia{
			ScreenVideoPath:     MockScreenFile,
			WebcamVideoPath:     "webcam.mov",
			WebcamStartOffsetMs: 120,
		},
		Camera: ManifestCamera{Enabled: false, DeviceID: "camera:default", PIPPreset: "bottom-right"},
		Diagnostics: ManifestDiagnostics{
			Sync: &ManifestSyncDiagnostics{
				TimelineBase: TimelineBaseMedia,
				Screen:       ManifestTrackDiagnostics{Enabled: true},
				Webcam: ManifestTrackDiagnostics{
					Enabled:       true,
					Path:          "webcam.mov",
					StartOffsetMs: 120,
					DurationMs:    1000,
				},
			},
		},
	}); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}

	manifest, err := service.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Media.WebcamVideoPath != "" || manifest.Media.WebcamStartOffsetMs != 0 {
		t.Fatalf("disabled camera kept media fields: %#v", manifest.Media)
	}
	if manifest.Camera.PIPPreset != "off" || manifest.Camera.DeviceID != "" {
		t.Fatalf("disabled camera was not normalized: %#v", manifest.Camera)
	}
	if manifest.Diagnostics.Sync == nil {
		t.Fatal("sync diagnostics missing")
	}
	if manifest.Diagnostics.Sync.Webcam.Enabled || manifest.Diagnostics.Sync.Webcam.Path != "" || manifest.Diagnostics.Sync.Webcam.StartOffsetMs != 0 {
		t.Fatalf("disabled camera kept webcam sync diagnostics: %#v", manifest.Diagnostics.Sync.Webcam)
	}
	if !manifest.Diagnostics.Sync.Screen.Enabled || manifest.Diagnostics.Sync.Screen.Path != MockScreenFile {
		t.Fatalf("screen sync diagnostics = %#v, want normalized screen path", manifest.Diagnostics.Sync.Screen)
	}
}

func TestWriteManifestRejectsEscapingSyncDiagnosticsPath(t *testing.T) {
	service := NewService()
	err := service.WriteManifest(filepath.Join(t.TempDir(), ManifestFile), Manifest{
		SchemaVersion: 1,
		App:           AppName,
		Status:        StatusRecording,
		Media:         ManifestMedia{ScreenVideoPath: MockScreenFile},
		Diagnostics: ManifestDiagnostics{
			Sync: &ManifestSyncDiagnostics{
				TimelineBase:         TimelineBaseMedia,
				AudioDiagnosticsPath: "../audio-diagnostics.json",
				Screen:               ManifestTrackDiagnostics{Enabled: true},
			},
		},
	})
	if err == nil {
		t.Fatal("WriteManifest() accepted an escaping sync diagnostics path")
	}
}

func TestScanMarksRecoverablePackages(t *testing.T) {
	videoDir := t.TempDir()
	service := NewService()
	recordingPkg, err := service.CreateMock(videoDir, CreateMockRequest{
		Status: StatusRecording,
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
	})
	if err != nil {
		t.Fatalf("CreateMock(recording) error = %v", err)
	}
	readyPkg, err := service.CreateMock(videoDir, CreateMockRequest{
		Status: StatusReady,
		Source: ManifestSource{Type: "screen", ID: "screen:secondary"},
	})
	if err != nil {
		t.Fatalf("CreateMock(ready) error = %v", err)
	}
	missingManifestDir := filepath.Join(videoDir, "recording-missing-manifest"+PackageDirSuffix)
	if err := os.MkdirAll(missingManifestDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(missing manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(missingManifestDir, "screen.mp4"), []byte("media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen.mp4) error = %v", err)
	}

	summaries, err := service.Scan(videoDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	byDir := map[string]RecoverySummary{}
	for _, summary := range summaries {
		byDir[summary.PackageDir] = summary
	}
	if !byDir[recordingPkg.Dir].Recoverable {
		t.Fatalf("recording package should be recoverable: %#v", byDir[recordingPkg.Dir])
	}
	if byDir[readyPkg.Dir].Recoverable {
		t.Fatalf("ready package should not be recoverable: %#v", byDir[readyPkg.Dir])
	}
	if !byDir[missingManifestDir].Recoverable {
		t.Fatalf("missing manifest with media should be recoverable: %#v", byDir[missingManifestDir])
	}
}

func TestRecoverMarksActiveManifestReady(t *testing.T) {
	videoDir := t.TempDir()
	service := NewService()
	pkg, err := service.CreateMock(videoDir, CreateMockRequest{
		Status: StatusRecording,
		Source: ManifestSource{Type: "screen", ID: "screen:primary"},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	completedAt := time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC)
	summary, err := service.Recover(videoDir, pkg.Dir, completedAt)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if summary.Status != StatusReady || summary.Recoverable {
		t.Fatalf("Recover() summary = %#v, want ready and not recoverable", summary)
	}

	manifest, err := service.ReadManifest(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != StatusReady {
		t.Fatalf("status = %q, want %q", manifest.Status, StatusReady)
	}
	if manifest.CompletedAt == nil || !manifest.CompletedAt.Equal(completedAt) {
		t.Fatalf("completedAt = %v, want %v", manifest.CompletedAt, completedAt)
	}
	if !manifest.Diagnostics.Recovered {
		t.Fatal("recovered manifest must mark diagnostics.recovered")
	}
}

func TestRecoverRebuildsMissingManifestFromScreenMedia(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := filepath.Join(videoDir, "recording-missing-manifest"+PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(package) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "screen.mp4"), []byte("media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen.mp4) error = %v", err)
	}

	completedAt := time.Date(2026, 6, 30, 16, 5, 0, 0, time.UTC)
	service := NewService()
	summary, err := service.Recover(videoDir, packageDir, completedAt)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if summary.ManifestPath != filepath.Join(packageDir, ManifestFile) {
		t.Fatalf("manifest path = %q, want package manifest", summary.ManifestPath)
	}

	manifest, err := service.ReadManifest(summary.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest(rebuilt) error = %v", err)
	}
	if manifest.Media.ScreenVideoPath != "screen.mp4" {
		t.Fatalf("screen path = %q, want screen.mp4", manifest.Media.ScreenVideoPath)
	}
	if manifest.Status != StatusReady || !manifest.Diagnostics.Recovered {
		t.Fatalf("rebuilt manifest = %#v, want ready recovered", manifest)
	}
	if manifest.Audio.SampleRate != 48000 || !manifest.Audio.SystemAudioIsNeverDenoised {
		t.Fatalf("audio defaults = %#v", manifest.Audio)
	}
}

func TestRecoverRejectsPackageOutsideVideoDir(t *testing.T) {
	videoDir := t.TempDir()
	outsideDir := filepath.Join(t.TempDir(), "recording-outside"+PackageDirSuffix)
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(outside) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "screen.mp4"), []byte("media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen.mp4) error = %v", err)
	}

	if _, err := NewService().Recover(videoDir, outsideDir, time.Now()); err == nil {
		t.Fatal("Recover() accepted a package outside videoDir")
	}
}
