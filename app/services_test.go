package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
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
