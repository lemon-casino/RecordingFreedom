package settings

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestLoadMissingSettingsReturnsDefaults(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))

	got, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Locale != LocaleZhCN {
		t.Fatalf("default locale = %q, want %q", got.Locale, LocaleZhCN)
	}
	if got.Audio.System {
		t.Fatal("default settings should keep system audio disabled until the user enables it")
	}
	if got.Audio.Microphone {
		t.Fatal("default settings should keep microphone disabled until the user enables it")
	}
	if got.Audio.NoiseSuppression {
		t.Fatal("default settings should keep microphone noise suppression disabled until the user enables it")
	}
	if got.Recording != recordingprofile.Default() {
		t.Fatalf("default recording profile = %#v, want %#v", got.Recording, recordingprofile.Default())
	}
	if got.Audio.SystemDeviceID != "system-audio:default" {
		t.Fatalf("default system audio device = %q", got.Audio.SystemDeviceID)
	}
	if !got.Window.MinimizeToTray {
		t.Fatal("default settings should minimize to tray")
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))
	now := time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	saved, err := service.Save(Settings{
		Locale: LocaleEN,
		Source: SourceSettings{
			LastSourceID:   "screen:primary",
			LastSourceType: "screen",
		},
		Storage: StorageSettings{
			DataRootDir: "  /Users/lemon/RecordingFreedomData  ",
		},
		Recording: RecordingSettings{
			Quality:          recordingprofile.QualityHigh,
			FPS:              60,
			CaptureCursor:    false,
			CountdownSeconds: 5,
		},
		Audio: AudioSettings{
			System:             false,
			SystemDeviceID:     "system-audio:blackhole",
			Microphone:         true,
			MicrophoneDeviceID: "microphone:studio",
			NoiseSuppression:   false,
			MicrophoneGain:     1.25,
		},
		Camera: CameraSettings{
			Enabled:   true,
			DeviceID:  "camera:default",
			PIPPreset: "bottom-right",
		},
		Window: WindowSettings{MinimizeToTray: true},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !saved.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %v, want %v", saved.UpdatedAt, now)
	}

	loaded, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Locale != LocaleEN {
		t.Fatalf("locale = %q, want %q", loaded.Locale, LocaleEN)
	}
	if loaded.Audio.MicrophoneDeviceID != "microphone:studio" {
		t.Fatalf("microphone device = %q", loaded.Audio.MicrophoneDeviceID)
	}
	if loaded.Audio.SystemDeviceID != "system-audio:blackhole" {
		t.Fatalf("system audio device = %q", loaded.Audio.SystemDeviceID)
	}
	if loaded.Storage.DataRootDir != "/Users/lemon/RecordingFreedomData" {
		t.Fatalf("data root dir = %q, want trimmed custom root", loaded.Storage.DataRootDir)
	}
	if loaded.Recording.Quality != recordingprofile.QualityHigh || loaded.Recording.FPS != 60 || loaded.Recording.CaptureCursor || loaded.Recording.CountdownSeconds != 5 {
		t.Fatalf("recording profile was not persisted: %#v", loaded.Recording)
	}
	if !loaded.Camera.Enabled {
		t.Fatal("camera setting was not persisted")
	}
}

func TestSaveNormalizesInvalidSettings(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))

	saved, err := service.Save(Settings{
		Locale:    Locale("fr"),
		Recording: RecordingSettings{Quality: "cinema", FPS: 120, CountdownSeconds: -4},
		Audio:     AudioSettings{MicrophoneGain: -2},
		Camera:    CameraSettings{PIPPreset: "top-right"},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.Locale != LocaleZhCN {
		t.Fatalf("invalid locale normalized to %q, want %q", saved.Locale, LocaleZhCN)
	}
	if saved.Audio.MicrophoneGain != 1 {
		t.Fatalf("microphone gain = %v, want 1", saved.Audio.MicrophoneGain)
	}
	if saved.Recording.Quality != recordingprofile.DefaultQuality || saved.Recording.FPS != recordingprofile.DefaultFPS || saved.Recording.CountdownSeconds != recordingprofile.DefaultCountdownSeconds {
		t.Fatalf("recording profile = %#v, want normalized quality/fps/countdown", saved.Recording)
	}
	if saved.Camera.PIPPreset != "bottom-right" {
		t.Fatalf("pip preset = %q, want bottom-right", saved.Camera.PIPPreset)
	}
}

func TestPathUsesAppDataRoot(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))

	path, err := service.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if path != filepath.Join(root, fileName) {
		t.Fatalf("path = %q, want %q", path, filepath.Join(root, fileName))
	}
	if _, err := service.Save(Default()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file was not created: %v", err)
	}
}
