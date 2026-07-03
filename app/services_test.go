package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/preflight"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

func TestBootstrapIncludesStorageStatus(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:   data,
		capture:   capture.NewService(),
		devices:   devices.NewService(),
		preflight: preflight.NewService(),
		recorder:  recording.NewService(data),
		settings:  settings.NewService(data),
	}

	bootstrap, err := service.Bootstrap()
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if bootstrap.Storage.VideoDir != bootstrap.AppData.VideoDir {
		t.Fatalf("storage video dir = %q, want appData video dir %q", bootstrap.Storage.VideoDir, bootstrap.AppData.VideoDir)
	}
	if !bootstrap.Storage.Writable {
		t.Fatalf("storage should be writable: %#v", bootstrap.Storage)
	}
	if bootstrap.Storage.Status == "" {
		t.Fatalf("storage status is empty: %#v", bootstrap.Storage)
	}
}

func TestLogClientEventWritesRootLogFile(t *testing.T) {
	root := t.TempDir()
	service := &RecordingFreedomService{appData: appdata.NewService(root)}

	if err := service.LogClientEvent(ClientLogEvent{
		Component: "pip-camera",
		Event:     "stream-error",
		Fields: map[string]string{
			"error": "NotReadableError",
		},
	}); err != nil {
		t.Fatalf("LogClientEvent() error = %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "logs", "recordingfreedom-*.log"))
	if err != nil {
		t.Fatalf("Glob(logs) error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("log files = %#v, want one root logs file", matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile(log) error = %v", err)
	}
	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &entry); err != nil {
		t.Fatalf("log JSON = %q, error = %v", data, err)
	}
	if entry["component"] != "client.pip-camera" || entry["event"] != "stream-error" {
		t.Fatalf("entry = %#v, want client pip-camera stream-error", entry)
	}
	fields, ok := entry["fields"].(map[string]any)
	if !ok || fields["error"] != "NotReadableError" {
		t.Fatalf("fields = %#v, want error field", entry["fields"])
	}
}

func TestReadPIPPreviewImageReadsManagedJPEG(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	previewPath := filepath.Join(info.VideoDir, "recording-preview.rfrec", recpackage.CacheDir, "pip-camera-preview.jpg")
	if err := os.MkdirAll(filepath.Dir(previewPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(preview) error = %v", err)
	}
	if err := os.WriteFile(previewPath, []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644); err != nil {
		t.Fatalf("WriteFile(preview) error = %v", err)
	}

	service := &RecordingFreedomService{appData: data}
	result, err := service.ReadPIPPreviewImage(PIPPreviewImageRequest{Path: previewPath})
	if err != nil {
		t.Fatalf("ReadPIPPreviewImage() error = %v", err)
	}
	if !result.Available || !strings.HasPrefix(result.DataURL, "data:image/jpeg;base64,") || result.ModifiedUnixNano <= 0 {
		t.Fatalf("result = %#v, want available JPEG data URL with modified time", result)
	}

	unchanged, err := service.ReadPIPPreviewImage(PIPPreviewImageRequest{
		Path:                  previewPath,
		KnownModifiedUnixNano: result.ModifiedUnixNano,
	})
	if err != nil {
		t.Fatalf("ReadPIPPreviewImage(known) error = %v", err)
	}
	if unchanged.Available || unchanged.DataURL != "" || unchanged.ModifiedUnixNano != result.ModifiedUnixNano {
		t.Fatalf("unchanged result = %#v, want unavailable without re-reading data URL", unchanged)
	}
}

func TestReadPIPPreviewImageRejectsOutsideManagedVideoDir(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	outsidePath := filepath.Join(t.TempDir(), "pip-camera-preview.jpg")
	if err := os.WriteFile(outsidePath, []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}

	service := &RecordingFreedomService{appData: data}
	if _, err := service.ReadPIPPreviewImage(PIPPreviewImageRequest{Path: outsidePath}); err == nil {
		t.Fatal("ReadPIPPreviewImage() accepted a path outside managed data/video")
	}
}

func TestSetDataRootUpdatesManagedVideoDirAndSettings(t *testing.T) {
	t.Setenv(appdata.EnvDataDir, "")
	data := appdata.NewServiceWithPointerBase("", t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(recpackage.NewService())),
		settings: settings.NewService(data),
	}
	customRoot := filepath.Join(t.TempDir(), "custom-root")

	info, err := service.SetDataRoot(customRoot)
	if err != nil {
		t.Fatalf("SetDataRoot() error = %v", err)
	}
	wantRoot, err := filepath.Abs(customRoot)
	if err != nil {
		t.Fatalf("Abs(customRoot) error = %v", err)
	}
	if info.RootDir != wantRoot {
		t.Fatalf("root = %q, want %q", info.RootDir, wantRoot)
	}
	if info.VideoDir != filepath.Join(wantRoot, "data", "video") {
		t.Fatalf("video dir = %q, want data/video under custom root", info.VideoDir)
	}

	currentSettings, err := service.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if currentSettings.Storage.DataRootDir != wantRoot {
		t.Fatalf("settings data root = %q, want %q", currentSettings.Storage.DataRootDir, wantRoot)
	}
}

func TestSetDataRootRejectsActiveRecording(t *testing.T) {
	t.Setenv(appdata.EnvDataDir, "")
	data := appdata.NewServiceWithPointerBase("", t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(recpackage.NewService())),
		settings: settings.NewService(data),
	}

	if _, err := service.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
	}); err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	if _, err := service.SetDataRoot(t.TempDir()); err == nil {
		t.Fatal("SetDataRoot() accepted a data root change while recording is active")
	}
}

func TestPatchAudioStatePersistsAudioControls(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	systemOn := true
	systemDevice := "system-audio:display"
	micOn := true
	micDevice := "microphone:studio"
	rnnoiseOn := true
	gain := 1.5

	state, err := service.PatchAudioState(AudioStatePatchRequest{
		System:             &systemOn,
		SystemDeviceID:     &systemDevice,
		Microphone:         &micOn,
		MicrophoneDeviceID: &micDevice,
		NoiseSuppression:   &rnnoiseOn,
		MicrophoneGain:     &gain,
	})
	if err != nil {
		t.Fatalf("PatchAudioState() error = %v", err)
	}
	if !state.System || state.SystemDeviceID != systemDevice || !state.Microphone || state.MicrophoneDeviceID != micDevice || !state.NoiseSuppression || state.MicrophoneGain != gain {
		t.Fatalf("audio state = %#v, want patched audio controls", state)
	}
	saved, err := service.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if saved.Audio.SystemDeviceID != systemDevice || saved.Audio.MicrophoneDeviceID != micDevice || !saved.Audio.NoiseSuppression {
		t.Fatalf("saved audio = %#v, want patched audio settings", saved.Audio)
	}
}

func TestPatchAudioStateDisablesRNNoiseWithMicrophone(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	micOn := true
	rnnoiseOn := true
	if _, err := service.PatchAudioState(AudioStatePatchRequest{
		Microphone:       &micOn,
		NoiseSuppression: &rnnoiseOn,
	}); err != nil {
		t.Fatalf("PatchAudioState(enable) error = %v", err)
	}
	micOff := false
	state, err := service.PatchAudioState(AudioStatePatchRequest{Microphone: &micOff})
	if err != nil {
		t.Fatalf("PatchAudioState(disable) error = %v", err)
	}
	if state.Microphone || state.NoiseSuppression {
		t.Fatalf("audio state = %#v, want microphone and rnnoise disabled", state)
	}
	saved, err := service.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if saved.Audio.Microphone || saved.Audio.NoiseSuppression {
		t.Fatalf("saved audio = %#v, want microphone and rnnoise disabled", saved.Audio)
	}
}

func TestOpenVideoDirectoryUsesManagedDataVideo(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{appData: data}
	var opened string
	originalOpenPath := openPath
	openPath = func(path string) error {
		opened = path
		return nil
	}
	t.Cleanup(func() {
		openPath = originalOpenPath
	})

	info, err := service.OpenVideoDirectory()
	if err != nil {
		t.Fatalf("OpenVideoDirectory() error = %v", err)
	}
	if opened != info.VideoDir {
		t.Fatalf("opened path = %q, want %q", opened, info.VideoDir)
	}
	if filepath.Base(opened) != "video" || filepath.Base(filepath.Dir(opened)) != "data" {
		t.Fatalf("opened path = %q, want managed data/video directory", opened)
	}
}

func TestDefaultOpenPathRejectsMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := defaultOpenPath(missing); err == nil {
		t.Fatal("defaultOpenPath() accepted a missing path")
	}
}

func TestOpenRecordingPackageUsesManagedPackageDir(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	pkg, err := recpackage.NewService().CreateMock(info.VideoDir, recpackage.CreateMockRequest{
		CreatedAt: time.Now(),
		Status:    recpackage.StatusReady,
		Source: recpackage.ManifestSource{
			Type: "screen",
			ID:   "screen:primary",
		},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	service := &RecordingFreedomService{appData: data}
	var opened string
	originalOpenPath := openPath
	openPath = func(path string) error {
		opened = path
		return nil
	}
	t.Cleanup(func() {
		openPath = originalOpenPath
	})

	summary, err := service.OpenRecordingPackage(pkg.Dir)
	if err != nil {
		t.Fatalf("OpenRecordingPackage() error = %v", err)
	}
	if opened != pkg.Dir {
		t.Fatalf("opened path = %q, want %q", opened, pkg.Dir)
	}
	if summary.PackageDir != pkg.Dir || summary.ManifestPath != pkg.ManifestPath || summary.Status != recpackage.StatusReady {
		t.Fatalf("summary = %#v, want ready package", summary)
	}
}

func TestOpenRecordingPackageAllowsMissingManifestForDiagnostics(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	packageDir := filepath.Join(info.VideoDir, "recording-missing-manifest"+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	service := &RecordingFreedomService{appData: data}
	var opened string
	originalOpenPath := openPath
	openPath = func(path string) error {
		opened = path
		return nil
	}
	t.Cleanup(func() {
		openPath = originalOpenPath
	})

	summary, err := service.OpenRecordingPackage(packageDir)
	if err != nil {
		t.Fatalf("OpenRecordingPackage() error = %v", err)
	}
	if opened != packageDir {
		t.Fatalf("opened path = %q, want %q", opened, packageDir)
	}
	if summary.Status != recpackage.StatusFailed || summary.Reason == "" {
		t.Fatalf("summary = %#v, want failed diagnostic summary", summary)
	}
}

func TestOpenRecordingPackageRejectsPathsOutsideManagedDataVideo(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	outside := filepath.Join(t.TempDir(), "recording-outside"+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll(outside) error = %v", err)
	}

	service := &RecordingFreedomService{appData: data}
	opened := false
	originalOpenPath := openPath
	openPath = func(path string) error {
		opened = true
		return nil
	}
	t.Cleanup(func() {
		openPath = originalOpenPath
	})

	if _, err := service.OpenRecordingPackage(outside); err == nil {
		t.Fatal("OpenRecordingPackage() accepted package outside managed data/video")
	}
	if opened {
		t.Fatal("OpenRecordingPackage() called openPath for rejected outside package")
	}
}

func TestOpenRecordingPackageRejectsNonPackageDirectory(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	nonPackageDir := filepath.Join(info.VideoDir, "not-a-recording")
	if err := os.MkdirAll(nonPackageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(nonPackageDir) error = %v", err)
	}

	service := &RecordingFreedomService{appData: data}
	opened := false
	originalOpenPath := openPath
	openPath = func(path string) error {
		opened = true
		return nil
	}
	t.Cleanup(func() {
		openPath = originalOpenPath
	})

	if _, err := service.OpenRecordingPackage(nonPackageDir); err == nil {
		t.Fatal("OpenRecordingPackage() accepted a directory without the .rfrec suffix")
	}
	if opened {
		t.Fatal("OpenRecordingPackage() called openPath for rejected non-package directory")
	}
}

func TestStartRecordingRejectsBlockedPreflightBeforeCreatingPackage(t *testing.T) {
	t.Setenv(appdata.EnvDataDir, "")
	root := t.TempDir()
	data := appdata.NewService(root)
	service := &RecordingFreedomService{
		appData:   data,
		capture:   capture.NewService(),
		devices:   devices.NewService(),
		preflight: preflight.NewService(),
		recorder:  recording.NewService(data),
		settings:  settings.NewService(data),
	}

	if _, err := service.StartRecording(recording.StartRequest{
		SourceID:   "screen:not-returned-by-device-service",
		SourceType: recording.SourceScreen,
	}); err == nil {
		t.Fatal("StartRecording() accepted a blocked preflight")
	}

	matches, err := filepath.Glob(filepath.Join(root, "data", "video", "*.rfrec"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("blocked preflight created packages = %#v, want none", matches)
	}
}

func TestEnrichRecordingCameraRequestUsesAvailableNativeCamera(t *testing.T) {
	req := enrichRecordingCameraRequest(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Camera: recording.CameraRequest{
			Enabled:   true,
			PIPPreset: "bottom-right",
		},
	}, devices.MediaInventory{
		Cameras: []devices.MediaDevice{
			{
				ID:                "camera:queued",
				Type:              devices.DeviceCamera,
				Name:              "Queued Camera",
				Available:         false,
				SidecarEligible:   true,
				UnavailableReason: "queued",
			},
			{
				ID:              "camera:dshow:integrated-camera",
				Type:            devices.DeviceCamera,
				Name:            "Integrated Camera",
				NativeID:        "Integrated Camera",
				Available:       true,
				SidecarEligible: true,
			},
		},
	})
	if req.Camera.DeviceID != "camera:dshow:integrated-camera" || req.Camera.DeviceNativeID != "Integrated Camera" {
		t.Fatalf("enriched camera = %#v, want available DirectShow camera with native id", req.Camera)
	}
}

func TestEnrichRecordingCameraRequestSkipsUnavailableDefaultCamera(t *testing.T) {
	req := enrichRecordingCameraRequest(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Camera: recording.CameraRequest{
			Enabled:   true,
			DeviceID:  "camera:default",
			PIPPreset: "bottom-right",
		},
	}, devices.MediaInventory{
		Cameras: []devices.MediaDevice{
			{
				ID:                "camera:default",
				Type:              devices.DeviceCamera,
				Name:              "Default Camera",
				NativeID:          "default",
				IsDefault:         true,
				Available:         false,
				SidecarEligible:   true,
				UnavailableReason: "DirectShow returned no default camera",
			},
			{
				ID:              "camera:dshow:usb-camera",
				Type:            devices.DeviceCamera,
				Name:            "USB Camera",
				NativeID:        "USB Camera",
				Available:       true,
				SidecarEligible: true,
			},
		},
	})
	if req.Camera.DeviceID != "camera:dshow:usb-camera" || req.Camera.DeviceNativeID != "USB Camera" {
		t.Fatalf("enriched camera = %#v, want fallback to available sidecar camera", req.Camera)
	}
}

func TestEnrichRecordingCameraRequestSkipsStaleUnavailableCamera(t *testing.T) {
	req := enrichRecordingCameraRequest(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Camera: recording.CameraRequest{
			Enabled:   true,
			DeviceID:  "camera:dshow:old-camera",
			PIPPreset: "bottom-right",
		},
	}, devices.MediaInventory{
		Cameras: []devices.MediaDevice{
			{
				ID:                "camera:dshow:old-camera",
				Type:              devices.DeviceCamera,
				Name:              "Old Camera",
				NativeID:          "Old Camera",
				Available:         false,
				SidecarEligible:   true,
				UnavailableReason: "camera is no longer connected",
			},
			{
				ID:              "camera:dshow:integrated-camera",
				Type:            devices.DeviceCamera,
				Name:            "Integrated Camera",
				NativeID:        "Integrated Camera",
				Available:       true,
				SidecarEligible: true,
			},
		},
	})
	if req.Camera.DeviceID != "camera:dshow:integrated-camera" || req.Camera.DeviceNativeID != "Integrated Camera" {
		t.Fatalf("enriched camera = %#v, want stale camera replaced by available sidecar camera", req.Camera)
	}
}

func TestPersistCameraPIPConfigOffDisablesCamera(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	current, err := service.settings.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	current.Camera.Enabled = true
	current.Camera.PIPPreset = string(pip.PresetBottomRight)
	current.Camera.PIP = pip.ConfigFromPreset(pip.PresetBottomRight)
	if _, err := service.settings.Save(current); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := service.persistCameraPIPConfig(pip.OffConfig()); err != nil {
		t.Fatalf("persistCameraPIPConfig(off) error = %v", err)
	}
	loaded, err := service.settings.Load()
	if err != nil {
		t.Fatalf("Load() after persist error = %v", err)
	}
	if loaded.Camera.Enabled {
		t.Fatal("camera remained enabled after persisting PIP off")
	}
	if loaded.Camera.PIPPreset != string(pip.PresetOff) || loaded.Camera.PIP.Preset != pip.PresetOff {
		t.Fatalf("camera pip = %q/%q, want off", loaded.Camera.PIPPreset, loaded.Camera.PIP.Preset)
	}
}

func TestStartAudioOnlyRejectsBlockedPreflightBeforeCreatingPackage(t *testing.T) {
	t.Setenv(appdata.EnvDataDir, "")
	root := t.TempDir()
	data := appdata.NewService(root)
	service := &RecordingFreedomService{
		appData:   data,
		capture:   capture.NewService(),
		devices:   devices.NewService(),
		preflight: preflight.NewService(),
		recorder:  recording.NewService(data),
		settings:  settings.NewService(data),
	}

	if _, err := service.StartAudioOnlyRecording(recording.AudioOnlyRequest{}); err == nil {
		t.Fatal("StartAudioOnlyRecording() accepted a blocked preflight")
	}

	matches, err := filepath.Glob(filepath.Join(root, "data", "video", "*.rfrec"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("blocked audio-only preflight created packages = %#v, want none", matches)
	}
}
