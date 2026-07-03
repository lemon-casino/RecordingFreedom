package recording

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestStartMockRecordingCreatesPackageInDataVideo(t *testing.T) {
	root := t.TempDir()
	service := newMockRecordingService(root)

	session, err := service.StartMockRecording(StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		SourceName: "Primary Display",
		SourceGeometry: &SourceGeometry{
			X:            -1920,
			Y:            0,
			Width:        1920,
			Height:       1080,
			DisplayIndex: 2,
			NativeID:     "DISPLAY2",
		},
		Recording: recordingprofile.Profile{
			Quality:          recordingprofile.QualityHigh,
			FPS:              60,
			CaptureCursor:    true,
			CountdownSeconds: 3,
		},
		Audio: AudioRequest{
			System:           true,
			SystemDeviceID:   "system-audio:default",
			Microphone:       true,
			MicrophoneID:     "microphone:default",
			NoiseSuppression: true,
			MicrophoneGain:   1.0,
		},
		Camera: CameraRequest{Enabled: true, DeviceID: "camera:default", PIPPreset: "bottom-right"},
	})
	if err != nil {
		t.Fatalf("StartMockRecording() error = %v", err)
	}

	wantParent := filepath.Join(root, "data", "video")
	if filepath.Dir(session.PackageDir) != wantParent {
		t.Fatalf("package parent = %q, want %q", filepath.Dir(session.PackageDir), wantParent)
	}
	if _, err := os.Stat(filepath.Join(session.PackageDir, "manifest.json")); err != nil {
		t.Fatalf("manifest was not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(session.PackageDir, "screen.mock.txt")); err != nil {
		t.Fatalf("mock media marker was not created: %v", err)
	}

	data, err := os.ReadFile(session.Manifest)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("manifest JSON is invalid: %v", err)
	}
	audio := manifest["audio"].(map[string]any)
	if audio["systemDeviceId"] != "system-audio:default" {
		t.Fatalf("systemDeviceId = %v, want system-audio:default", audio["systemDeviceId"])
	}
	if audio["microphoneNoiseSuppression"] != "rnnoise" {
		t.Fatalf("microphoneNoiseSuppression = %v, want rnnoise", audio["microphoneNoiseSuppression"])
	}
	if audio["microphoneDeviceId"] != "microphone:default" {
		t.Fatalf("microphoneDeviceId = %v, want microphone:default", audio["microphoneDeviceId"])
	}
	source := manifest["source"].(map[string]any)
	if source["name"] != "Primary Display" {
		t.Fatalf("source name = %v, want Primary Display", source["name"])
	}
	geometry := source["geometry"].(map[string]any)
	if geometry["x"] != float64(-1920) || geometry["width"] != float64(1920) || geometry["nativeId"] != "DISPLAY2" {
		t.Fatalf("source geometry = %#v, want selected display geometry", geometry)
	}
	recording := manifest["recording"].(map[string]any)
	if recording["quality"] != recordingprofile.QualityHigh || recording["fps"] != float64(60) || recording["countdownSeconds"] != float64(3) {
		t.Fatalf("recording profile = %#v", recording)
	}
	camera := manifest["camera"].(map[string]any)
	if camera["deviceId"] != "camera:default" {
		t.Fatalf("camera deviceId = %v, want camera:default", camera["deviceId"])
	}
}

func TestPauseResumeStopPatchManifestStatus(t *testing.T) {
	service := newMockRecordingService(t.TempDir())
	if _, err := service.StartMockRecording(StartRequest{SourceID: "screen:primary", SourceType: SourceScreen}); err != nil {
		t.Fatalf("StartMockRecording() error = %v", err)
	}
	if session, err := service.Pause(); err != nil || session.Status != StatePaused {
		t.Fatalf("Pause() = (%v, %v), want paused", session.Status, err)
	}
	if session, err := service.Resume(); err != nil || session.Status != StateRecording {
		t.Fatalf("Resume() = (%v, %v), want recording", session.Status, err)
	}
	if session, err := service.Stop(); err != nil || session.Status != StateReady {
		t.Fatalf("Stop() = (%v, %v), want ready", session.Status, err)
	}
}

func TestActiveRecordingOffsetSubtractsPausedDuration(t *testing.T) {
	started := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	service := &Service{
		state: StatePaused,
		session: &Session{
			ID:            "session-1",
			RecordingMode: recpackage.RecordingModeScreen,
			Status:        StatePaused,
			StartedAt:     started,
		},
		pausedDuration: 2 * time.Second,
		pauseStartedAt: started.Add(5 * time.Second),
	}

	offset, ok := service.ActiveRecordingOffset(started.Add(8 * time.Second))
	if !ok {
		t.Fatal("ActiveRecordingOffset() ok = false, want true")
	}
	if offset != 3000 {
		t.Fatalf("offset = %dms, want 3000ms after subtracting completed and active pauses", offset)
	}
}

func TestStartRecordingDelegatesToBackend(t *testing.T) {
	backend := &trackingBackend{packages: recpackage.NewService()}
	service := NewServiceWithBackend(appdata.NewService(t.TempDir()), backend)

	session, err := service.StartRecording(StartRequest{SourceID: "screen:primary", SourceType: SourceScreen, Audio: AudioRequest{System: true}})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	if backend.started != 1 {
		t.Fatalf("backend started = %d, want 1", backend.started)
	}
	if session.Backend != backend.ID() {
		t.Fatalf("session backend = %q, want %q", session.Backend, backend.ID())
	}
	if backend.lastStart.StartRequest.Audio.SystemDeviceID != defaultSystemAudioID {
		t.Fatalf("backend system audio device = %q, want %q", backend.lastStart.StartRequest.Audio.SystemDeviceID, defaultSystemAudioID)
	}
	if backend.lastStart.StartRequest.Recording != recordingprofile.Default() {
		t.Fatalf("backend recording profile = %#v, want default", backend.lastStart.StartRequest.Recording)
	}
	if _, err := service.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if _, err := service.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if _, err := service.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if backend.paused != 1 || backend.resumed != 1 || backend.stopped != 1 {
		t.Fatalf("backend controls = pause:%d resume:%d stop:%d, want 1 each", backend.paused, backend.resumed, backend.stopped)
	}
}

func TestPatchActiveCameraPIPUpdatesRecordingManifest(t *testing.T) {
	service := newMockRecordingService(t.TempDir())
	session, err := service.StartMockRecording(StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Camera: CameraRequest{
			Enabled:   true,
			DeviceID:  "camera:default",
			PIPPreset: string(pip.PresetBottomRight),
		},
	})
	if err != nil {
		t.Fatalf("StartMockRecording() error = %v", err)
	}

	next := pip.Config{
		Preset:      pip.PresetFree,
		Shape:       pip.ShapeSquare,
		Mirror:      false,
		Position:    pip.Position{X: 0.25, Y: 0.6},
		Scale:       0.32,
		EdgeFeather: 0.22,
	}
	if err := service.PatchActiveCameraPIP(next); err != nil {
		t.Fatalf("PatchActiveCameraPIP() error = %v", err)
	}

	manifest, err := recpackage.NewService().ReadManifest(session.Manifest)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Camera.PIPPreset != string(pip.PresetFree) || manifest.Camera.PIP.Preset != pip.PresetFree {
		t.Fatalf("manifest pip preset = %q/%q, want free", manifest.Camera.PIPPreset, manifest.Camera.PIP.Preset)
	}
	if manifest.Camera.PIP.Shape != pip.ShapeSquare || manifest.Camera.PIP.Mirror {
		t.Fatalf("manifest pip shape/mirror = %#v, want square non-mirrored", manifest.Camera.PIP)
	}
	if manifest.Camera.PIP.Position.X != 0.25 || manifest.Camera.PIP.Position.Y != 0.6 || manifest.Camera.PIP.Scale != pip.MaximumScale || manifest.Camera.PIP.EdgeFeather != 0.22 {
		t.Fatalf("manifest pip layout = %#v, want patched config", manifest.Camera.PIP)
	}
}

func TestPatchActiveCameraPIPIsNoopWithoutScreenCameraRecording(t *testing.T) {
	service := newMockRecordingService(t.TempDir())
	if err := service.PatchActiveCameraPIP(pip.Config{Preset: pip.PresetFree}); err != nil {
		t.Fatalf("PatchActiveCameraPIP() with no session error = %v", err)
	}
	service.state = StateRecording
	service.session = &Session{ID: "audio-session", RecordingMode: recpackage.RecordingModeAudio, Status: StateRecording}
	if err := service.PatchActiveCameraPIP(pip.Config{Preset: pip.PresetFree}); err != nil {
		t.Fatalf("PatchActiveCameraPIP() during audio-only error = %v", err)
	}
}

func TestStartAudioOnlyRecordingCreatesReadyPackage(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))
	audioSession := &fileWritingAudioSession{
		diagnostics: audio.Diagnostics{
			Backend: BackendAudioOnlyNative,
			Microphone: audio.StreamDiagnostics{
				Enabled:        true,
				SampleRate:     audio.RNNoiseSampleRate,
				SamplesWritten: 48000,
				EndOffsetMs:    1000,
				DurationMs:     1000,
			},
		},
	}
	service.audioOnlyBackend = NewAudioOnlyRuntimeBackend(service.packages, AudioOnlyRuntimeOptions{
		AudioSessionFactory: func(config audio.CaptureConfig, suppressor audio.NoiseSuppressor) (NativeAudioSession, error) {
			if suppressor != nil {
				t.Fatalf("suppressor = %#v, want nil", suppressor)
			}
			audioSession.path = config.MicrophoneAudioPath
			return audioSession, nil
		},
		PostStopProcessor: func(runtime *AudioOnlyRuntime) error {
			writeMinimalMP4File(t, runtime.Plan.AudioOnlyPath, "soun")
			manifest, err := runtime.packages.PatchAudioOnlyMuxed(runtime.Plan.Package.ManifestPath, false, true)
			if err != nil {
				return err
			}
			runtime.Plan.Package.Manifest = manifest
			runtime.Plan.MicrophoneAudioPath = runtime.Plan.AudioOnlyPath
			return nil
		},
	})

	session, err := service.StartAudioOnlyRecording(AudioOnlyRequest{
		Audio: AudioRequest{Microphone: true},
	})
	if err != nil {
		t.Fatalf("StartAudioOnlyRecording() error = %v", err)
	}
	if session.RecordingMode != recpackage.RecordingModeAudio || session.Backend != BackendAudioOnlyNative {
		t.Fatalf("session = %#v, want audio-only backend", session)
	}
	if _, err := os.Stat(filepath.Join(session.PackageDir, recpackage.ScreenVideoFile)); err == nil {
		t.Fatal("audio-only recording created screen media")
	}
	if _, err := service.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if _, err := service.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	stopped, err := service.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if stopped.Status != StateReady || service.State() != StateReady {
		t.Fatalf("stop state = session:%q service:%q, want ready", stopped.Status, service.State())
	}
	if audioSession.started != 1 || audioSession.paused != 1 || audioSession.resumed != 1 || audioSession.stopped != 1 {
		t.Fatalf("audio lifecycle = start:%d pause:%d resume:%d stop:%d", audioSession.started, audioSession.paused, audioSession.resumed, audioSession.stopped)
	}

	manifest, err := recpackage.NewService().ReadManifest(session.Manifest)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != recpackage.StatusReady || manifest.RecordingMode != recpackage.RecordingModeAudio {
		t.Fatalf("manifest status/mode = %q/%q, want ready audio-only", manifest.Status, manifest.RecordingMode)
	}
	if manifest.Media.ScreenVideoPath != "" || manifest.Media.AudioPath != recpackage.AudioOnlyFile || manifest.Media.MicrophoneAudioPath != recpackage.AudioOnlyFile {
		t.Fatalf("audio-only media = %#v", manifest.Media)
	}
	if manifest.Media.MicrophoneAudioStorage != recpackage.AudioStorageMuxed {
		t.Fatalf("microphone storage = %q, want muxed", manifest.Media.MicrophoneAudioStorage)
	}
	if manifest.Diagnostics.Sync == nil || manifest.Diagnostics.Sync.Microphone.Path != recpackage.AudioOnlyFile {
		t.Fatalf("sync diagnostics = %#v, want microphone audio.m4a", manifest.Diagnostics.Sync)
	}
}

func TestStopPatchesBackendSyncDiagnosticsBeforeReady(t *testing.T) {
	backend := &trackingBackend{
		packages: recpackage.NewService(),
		stopSync: &recpackage.ManifestSyncDiagnostics{
			TimelineBase:         recpackage.TimelineBaseMedia,
			AudioDiagnosticsPath: recpackage.AudioDiagnosticsFile,
			VideoDiagnosticsPath: recpackage.VideoDiagnosticsFile,
			Screen: recpackage.ManifestTrackDiagnostics{
				Enabled:     true,
				Path:        recpackage.MockScreenFile,
				Clock:       recpackage.TimelineBaseMedia,
				EndOffsetMs: 1000,
				DurationMs:  1000,
				FrameRate:   30,
			},
			SystemAudio: recpackage.ManifestTrackDiagnostics{
				Enabled:     true,
				Clock:       recpackage.TimelineBaseMedia,
				EndOffsetMs: 1000,
				DurationMs:  1000,
				SampleRate:  48000,
			},
			Microphone: recpackage.ManifestTrackDiagnostics{
				Enabled:     true,
				Clock:       recpackage.TimelineBaseMedia,
				EndOffsetMs: 1000,
				DurationMs:  1000,
				SampleRate:  48000,
			},
			PauseSegments: []recpackage.ManifestPauseSegment{{StartOffsetMs: 400, EndOffsetMs: 550, DurationMs: 150}},
		},
	}
	service := NewServiceWithBackend(appdata.NewService(t.TempDir()), backend)
	session, err := service.StartRecording(StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Audio:      AudioRequest{System: true, Microphone: true, NoiseSuppression: true},
	})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	if _, err := service.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	manifest, err := recpackage.NewService().ReadManifest(session.Manifest)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != recpackage.StatusReady {
		t.Fatalf("manifest status = %q, want ready", manifest.Status)
	}
	if manifest.Diagnostics.Sync == nil {
		t.Fatal("backend sync diagnostics were not written")
	}
	if manifest.Diagnostics.Sync.TimelineBase != recpackage.TimelineBaseMedia {
		t.Fatalf("timeline base = %q, want media timestamp", manifest.Diagnostics.Sync.TimelineBase)
	}
	if len(manifest.Diagnostics.Sync.PauseSegments) != 1 || manifest.Diagnostics.Sync.PauseSegments[0].DurationMs != 150 {
		t.Fatalf("pause segments = %#v, want backend pause segment", manifest.Diagnostics.Sync.PauseSegments)
	}
	if manifest.Diagnostics.Sync.AudioDiagnosticsPath != recpackage.AudioDiagnosticsFile {
		t.Fatalf("audio diagnostics path = %q, want package relative path", manifest.Diagnostics.Sync.AudioDiagnosticsPath)
	}
}

func TestStopFailsWhenBackendSyncDiagnosticsAreInvalid(t *testing.T) {
	backend := &trackingBackend{
		packages: recpackage.NewService(),
		stopSync: &recpackage.ManifestSyncDiagnostics{
			TimelineBase:         recpackage.TimelineBaseMedia,
			AudioDiagnosticsPath: "../audio-diagnostics.json",
			Screen: recpackage.ManifestTrackDiagnostics{
				Enabled:    true,
				Path:       recpackage.MockScreenFile,
				DurationMs: 1000,
			},
		},
	}
	service := NewServiceWithBackend(appdata.NewService(t.TempDir()), backend)
	session, err := service.StartRecording(StartRequest{SourceID: "screen:primary", SourceType: SourceScreen})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}

	stopped, err := service.Stop()
	if err == nil {
		t.Fatal("Stop() accepted escaping backend sync diagnostics")
	}
	if stopped.Status != StateFailed || service.State() != StateFailed {
		t.Fatalf("stop state = session:%q service:%q, want failed", stopped.Status, service.State())
	}
	manifest, readErr := recpackage.NewService().ReadManifest(session.Manifest)
	if readErr != nil {
		t.Fatalf("ReadManifest() error = %v", readErr)
	}
	if manifest.Status != recpackage.StatusFailed {
		t.Fatalf("manifest status = %q, want failed", manifest.Status)
	}
}

func TestStopFailsWhenNativeBackendDidNotWriteScreenMedia(t *testing.T) {
	backend := &nativeProbeBackend{packages: recpackage.NewService()}
	service := NewServiceWithBackend(appdata.NewService(t.TempDir()), backend)
	session, err := service.StartRecording(StartRequest{SourceID: "cgdisplay:1", SourceType: SourceScreen})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}

	stopped, err := service.Stop()
	if err == nil {
		t.Fatal("Stop() accepted a native package without screen media")
	}
	if stopped.Status != StateFailed || service.State() != StateFailed {
		t.Fatalf("stop state = session:%q service:%q, want failed", stopped.Status, service.State())
	}
	manifest, readErr := recpackage.NewService().ReadManifest(session.Manifest)
	if readErr != nil {
		t.Fatalf("ReadManifest() error = %v", readErr)
	}
	if manifest.Status != recpackage.StatusFailed {
		t.Fatalf("manifest status = %q, want failed", manifest.Status)
	}
}

func TestStopMarksNativeBackendReadyAfterMediaProbe(t *testing.T) {
	backend := &nativeProbeBackend{
		packages:    recpackage.NewService(),
		writeScreen: true,
		writeWebcam: true,
		stopSync:    nativeSyncDiagnostics(true),
	}
	service := NewServiceWithBackend(appdata.NewService(t.TempDir()), backend)
	session, err := service.StartRecording(StartRequest{
		SourceID:   "cgdisplay:1",
		SourceType: SourceScreen,
		Camera:     CameraRequest{Enabled: true, DeviceID: "camera:default", PIPPreset: "bottom-right"},
	})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}

	stopped, err := service.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if stopped.Status != StateReady || service.State() != StateReady {
		t.Fatalf("stop state = session:%q service:%q, want ready", stopped.Status, service.State())
	}
	manifest, readErr := recpackage.NewService().ReadManifest(session.Manifest)
	if readErr != nil {
		t.Fatalf("ReadManifest() error = %v", readErr)
	}
	if manifest.Status != recpackage.StatusReady {
		t.Fatalf("manifest status = %q, want ready", manifest.Status)
	}
	if manifest.Diagnostics.Sync == nil || manifest.Diagnostics.Sync.Webcam.Path != recpackage.WebcamVideoFile {
		t.Fatalf("sync diagnostics = %#v, want webcam sidecar path", manifest.Diagnostics.Sync)
	}
	if _, err := os.Stat(filepath.Join(session.PackageDir, recpackage.ScreenVideoFile)); err != nil {
		t.Fatalf("screen media missing after native probe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(session.PackageDir, recpackage.WebcamVideoFile)); err != nil {
		t.Fatalf("webcam media missing after native probe: %v", err)
	}
}

func TestScanPackagesReportsRecoverableActivePackage(t *testing.T) {
	service := newMockRecordingService(t.TempDir())
	session, err := service.StartMockRecording(StartRequest{SourceID: "screen:primary", SourceType: SourceScreen})
	if err != nil {
		t.Fatalf("StartMockRecording() error = %v", err)
	}

	summaries, err := service.ScanPackages()
	if err != nil {
		t.Fatalf("ScanPackages() error = %v", err)
	}

	for _, summary := range summaries {
		if summary.PackageDir == session.PackageDir {
			if !summary.Recoverable || summary.Status != recpackage.StatusRecoverable {
				t.Fatalf("active package summary = %#v, want recoverable", summary)
			}
			return
		}
	}
	t.Fatalf("ScanPackages() did not include active package %q: %#v", session.PackageDir, summaries)
}

func newMockRecordingService(root string) *Service {
	return NewServiceWithBackend(appdata.NewService(root), NewMockBackend(recpackage.NewService()))
}

func TestRecoverPackageMarksDataVideoPackageReady(t *testing.T) {
	t.Setenv(EnvRecordingBackend, "")
	root := t.TempDir()
	data := appdata.NewService(root)
	videoDir, err := data.VideoDir()
	if err != nil {
		t.Fatalf("VideoDir() error = %v", err)
	}
	pkg, err := recpackage.NewService().CreateMock(videoDir, recpackage.CreateMockRequest{
		Status: recpackage.StatusRecording,
		Source: recpackage.ManifestSource{Type: "screen", ID: "screen:primary"},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	service := NewService(data)
	summary, err := service.RecoverPackage(pkg.Dir)
	if err != nil {
		t.Fatalf("RecoverPackage() error = %v", err)
	}
	if summary.Status != recpackage.StatusReady {
		t.Fatalf("recovery status = %q, want ready", summary.Status)
	}

	manifest, err := recpackage.NewService().ReadManifest(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != recpackage.StatusReady || !manifest.Diagnostics.Recovered {
		t.Fatalf("manifest recovery = status:%q recovered:%v", manifest.Status, manifest.Diagnostics.Recovered)
	}
}

type trackingBackend struct {
	packages  *recpackage.Service
	started   int
	paused    int
	resumed   int
	stopped   int
	lastStart BackendStartRequest
	stopSync  *recpackage.ManifestSyncDiagnostics
}

func (b *trackingBackend) ID() string {
	return "tracking-backend"
}

func (b *trackingBackend) Start(_ context.Context, req BackendStartRequest) (BackendStartResult, error) {
	b.started++
	b.lastStart = req
	pkg, err := b.packages.CreateMock(req.VideoDir, recpackage.CreateMockRequest{
		CreatedAt: req.CreatedAt,
		Status:    recpackage.StatusRecording,
		Source:    manifestSourceFromStartRequest(req.StartRequest),
		Recording: req.StartRequest.Recording,
	})
	if err != nil {
		return BackendStartResult{}, err
	}
	return BackendStartResult{Package: pkg}, nil
}

func (b *trackingBackend) Pause(context.Context, BackendControlRequest) error {
	b.paused++
	return nil
}

func (b *trackingBackend) Resume(context.Context, BackendControlRequest) error {
	b.resumed++
	return nil
}

func (b *trackingBackend) Stop(context.Context, BackendControlRequest) (BackendStopResult, error) {
	b.stopped++
	return BackendStopResult{SyncDiagnostics: b.stopSync}, nil
}

type nativeProbeBackend struct {
	packages    *recpackage.Service
	writeScreen bool
	writeWebcam bool
	stopSync    *recpackage.ManifestSyncDiagnostics
}

func (b *nativeProbeBackend) ID() string {
	return "native-probe"
}

func (b *nativeProbeBackend) Start(_ context.Context, req BackendStartRequest) (BackendStartResult, error) {
	plan, err := CreateNativeWritePlan(b.packages, b.ID(), req)
	if err != nil {
		return BackendStartResult{}, err
	}
	if b.writeScreen {
		if err := os.WriteFile(plan.ScreenVideoPath, []byte("native screen media"), 0o644); err != nil {
			return BackendStartResult{}, err
		}
	}
	if b.writeWebcam && plan.WebcamVideoPath != "" {
		if err := os.WriteFile(plan.WebcamVideoPath, minimalMP4Data("vide"), 0o644); err != nil {
			return BackendStartResult{}, err
		}
	}
	return BackendStartResult{Package: plan.Package}, nil
}

func (b *nativeProbeBackend) Pause(context.Context, BackendControlRequest) error {
	return nil
}

func (b *nativeProbeBackend) Resume(context.Context, BackendControlRequest) error {
	return nil
}

func (b *nativeProbeBackend) Stop(context.Context, BackendControlRequest) (BackendStopResult, error) {
	return BackendStopResult{SyncDiagnostics: b.stopSync}, nil
}

type fileWritingAudioSession struct {
	path        string
	started     int
	paused      int
	resumed     int
	stopped     int
	diagnostics audio.Diagnostics
}

func (s *fileWritingAudioSession) Start(context.Context) error {
	s.started++
	return nil
}

func (s *fileWritingAudioSession) Pause() error {
	s.paused++
	return nil
}

func (s *fileWritingAudioSession) Resume() error {
	s.resumed++
	return nil
}

func (s *fileWritingAudioSession) Stop() error {
	s.stopped++
	return os.WriteFile(s.path, bytes.Repeat([]byte{1}, 45), 0o644)
}

func (s *fileWritingAudioSession) Diagnostics() audio.Diagnostics {
	return s.diagnostics
}

func nativeSyncDiagnostics(includeWebcam bool) *recpackage.ManifestSyncDiagnostics {
	diagnostics := &recpackage.ManifestSyncDiagnostics{
		TimelineBase:         recpackage.TimelineBaseMedia,
		AudioDiagnosticsPath: recpackage.AudioDiagnosticsFile,
		VideoDiagnosticsPath: recpackage.VideoDiagnosticsFile,
		Screen: recpackage.ManifestTrackDiagnostics{
			Enabled:     true,
			Path:        recpackage.ScreenVideoFile,
			Clock:       recpackage.TimelineBaseMedia,
			EndOffsetMs: 1000,
			DurationMs:  1000,
			FrameRate:   30,
		},
	}
	if includeWebcam {
		diagnostics.Webcam = recpackage.ManifestTrackDiagnostics{
			Enabled:       true,
			Path:          recpackage.WebcamVideoFile,
			Clock:         recpackage.TimelineBaseMedia,
			StartOffsetMs: 100,
			EndOffsetMs:   1100,
			DurationMs:    1000,
			FrameRate:     30,
		}
	}
	return diagnostics
}

func writeMinimalMP4File(t *testing.T, path string, handlerType string) {
	t.Helper()
	data := minimalMP4Data(handlerType)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(minimal mp4) error = %v", err)
	}
}

func minimalMP4Data(handlerType string) []byte {
	payload := make([]byte, 0, 12)
	payload = append(payload, 0, 0, 0, 0)
	payload = append(payload, 0, 0, 0, 0)
	payload = append(payload, []byte(handlerType)...)
	data := mp4Box("ftyp", []byte("isom0000"))
	data = append(data, mp4Box("moov", mp4Box("trak", mp4Box("mdia", mp4Box("hdlr", payload))))...)
	return data
}

func mp4Box(kind string, payload []byte) []byte {
	box := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(box)))
	copy(box[4:8], []byte(kind))
	copy(box[8:], payload)
	return box
}
