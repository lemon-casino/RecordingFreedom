package main

import (
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/preflight"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
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
		recorder: recording.NewService(data),
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
		recorder: recording.NewService(data),
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
