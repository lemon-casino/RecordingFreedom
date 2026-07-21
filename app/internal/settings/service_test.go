package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
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
	if !got.Audio.NoiseSuppression {
		t.Fatal("default settings should enable the RNNoise voice-focus preference")
	}
	if got.Recording != recordingprofile.Default() {
		t.Fatalf("default recording profile = %#v, want %#v", got.Recording, recordingprofile.Default())
	}
	if got.Camera.PIP.Preset != pip.DefaultPreset || got.Camera.PIP.Shape != pip.DefaultShape || !got.Camera.PIP.Mirror {
		t.Fatalf("default pip = %#v, want default preset/shape/mirror", got.Camera.PIP)
	}
	if got.Audio.SystemDeviceID != "system-audio:default" {
		t.Fatalf("default system audio device = %q", got.Audio.SystemDeviceID)
	}
	if !got.Window.MinimizeToTray {
		t.Fatal("default settings should minimize to tray")
	}
	if got.Window.StartAtLogin {
		t.Fatal("default settings should not start at login until the user enables it")
	}
	if got.Window.Theme != ThemeNightTeal {
		t.Fatalf("default theme = %q, want %q", got.Window.Theme, ThemeNightTeal)
	}
	if got.Whiteboard.LastOpacity != 100 {
		t.Fatalf("default whiteboard opacity = %d, want 100", got.Whiteboard.LastOpacity)
	}
	if got.OCR.AutoRecognizeScreenshots {
		t.Fatal("default settings should not auto-run OCR after screenshots")
	}
	if got.OCR.Translation.Provider != "disabled" || got.OCR.Translation.PrivacyConfirmed {
		t.Fatalf("default OCR translation = %#v, want disabled and unconfirmed", got.OCR.Translation)
	}
	if got.Shortcuts.ToggleRecording != "CmdOrCtrl+Shift+R" || got.Shortcuts.OpenWhiteboard != "CmdOrCtrl+Shift+B" {
		t.Fatalf("default shortcuts = %#v, want recording and whiteboard defaults", got.Shortcuts)
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
			PIPPreset: "free",
			PIP: pip.Config{
				Preset:      pip.PresetFree,
				Shape:       pip.ShapeSquare,
				Mirror:      false,
				Position:    pip.Position{X: 0.25, Y: 0.75},
				Scale:       0.28,
				EdgeFeather: 0.22,
			},
		},
		Whiteboard: WhiteboardSettings{
			Enabled:         true,
			LastMode:        "board",
			LastTool:        "arrow",
			LastStrokeColor: "#38bdf8",
			LastStrokeWidth: "bold",
			LastOpacity:     65,
			CapturePolicy:   "export-compose",
		},
		OCR: OCRSettings{
			AutoRecognizeScreenshots: true,
			Translation: OCRTranslationSettings{
				Provider:           "openai-compatible",
				BaseURL:            "  https://translate.example/v1  ",
				APIKey:             "  local-secret  ",
				Model:              "  gpt-4o-mini  ",
				SourceLanguage:     "auto",
				TargetLanguage:     "en",
				PrivacyConfirmed:   true,
				PrivacyConfirmedAt: "2026-07-06T00:00:00Z",
			},
		},
		Shortcuts: ShortcutSettings{
			ToggleRecording: "cmdorctrl + shift + r",
			TogglePause:     "CmdOrCtrl+Alt+P",
			ToggleCamera:    "CmdOrCtrl+Shift+C",
			OpenWhiteboard:  "CmdOrCtrl+Shift+B",
		},
		Window: WindowSettings{MinimizeToTray: true, Theme: ThemeSunsetYellow, StartAtLogin: true},
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
	if loaded.Camera.PIPPreset != "free" || loaded.Camera.PIP.Shape != pip.ShapeSquare || loaded.Camera.PIP.Mirror {
		t.Fatalf("pip settings were not persisted: %#v", loaded.Camera)
	}
	if loaded.Camera.PIP.Position.X != 0.25 || loaded.Camera.PIP.Position.Y != 0.75 || loaded.Camera.PIP.Scale != pip.MaximumScale || loaded.Camera.PIP.EdgeFeather != 0.22 {
		t.Fatalf("pip layout settings were not persisted: %#v", loaded.Camera.PIP)
	}
	if loaded.Whiteboard.LastTool != "arrow" || loaded.Whiteboard.LastStrokeWidth != "bold" || loaded.Whiteboard.LastOpacity != 65 {
		t.Fatalf("whiteboard settings were not persisted: %#v", loaded.Whiteboard)
	}
	if !loaded.OCR.AutoRecognizeScreenshots {
		t.Fatal("OCR auto-recognize setting was not persisted")
	}
	if loaded.OCR.Translation.Provider != "openai-compatible" || loaded.OCR.Translation.BaseURL != "https://translate.example/v1" || loaded.OCR.Translation.APIKey != "" || !loaded.OCR.Translation.APIKeySet || loaded.OCR.Translation.Model != "gpt-4o-mini" || loaded.OCR.Translation.TargetLanguage != "en" || !loaded.OCR.Translation.PrivacyConfirmed {
		t.Fatalf("OCR translation settings were not persisted and normalized: %#v", loaded.OCR.Translation)
	}
	settingsPath, err := service.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	if len(settingsData) == 0 || strings.Contains(string(settingsData), "local-secret") {
		t.Fatalf("settings file should not contain raw OCR translation API key: %s", settingsData)
	}
	if loaded.Window.Theme != ThemeSunsetYellow {
		t.Fatalf("theme = %q, want %q", loaded.Window.Theme, ThemeSunsetYellow)
	}
	if !loaded.Window.StartAtLogin {
		t.Fatal("start at login setting was not persisted")
	}
	if loaded.Shortcuts.ToggleRecording != "CmdOrCtrl+Shift+R" || loaded.Shortcuts.TogglePause != "CmdOrCtrl+OptionOrAlt+P" {
		t.Fatalf("shortcuts were not normalized and persisted: %#v", loaded.Shortcuts)
	}
}

func TestValidateShortcutsAllowsFunctionKey(t *testing.T) {
	shortcuts := DefaultShortcuts()
	shortcuts.ToggleRecording = "f1"

	normalized, err := ValidateShortcuts(shortcuts)
	if err != nil {
		t.Fatalf("ValidateShortcuts() error = %v", err)
	}
	if normalized.ToggleRecording != "F1" {
		t.Fatalf("function shortcut = %q, want F1", normalized.ToggleRecording)
	}

	shortcuts.ToggleRecording = "A"
	if _, err := ValidateShortcuts(shortcuts); err == nil {
		t.Fatal("plain letter shortcut should still require a modifier")
	}
}

func TestLoadMigratesLegacyPIPScaleRange(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	path, err := service.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	legacy := `{
  "schemaVersion": 1,
  "locale": "zh-CN",
  "camera": {
    "enabled": true,
    "deviceId": "camera:default",
    "pipPreset": "bottom-right",
    "pip": {
      "preset": "bottom-right",
      "shape": "circle",
      "mirror": true,
      "position": {"x": 1, "y": 1},
      "scale": 0.08,
      "edgeFeather": 0.16
    }
  }
}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", loaded.SchemaVersion, SchemaVersion)
	}
	if loaded.Camera.PIP.Scale != pip.MaximumScale {
		t.Fatalf("legacy max scale = %v, want migrated max %v", loaded.Camera.PIP.Scale, pip.MaximumScale)
	}
}

func TestLoadEnablesRNNoiseVoiceFocusOnceForPreV6Settings(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	path, err := service.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	legacy := `{
  "schemaVersion": 5,
  "locale": "zh-CN",
  "audio": {
    "microphone": true,
    "noiseSuppression": false,
    "microphoneGain": 1
  }
}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.SchemaVersion != SchemaVersion || !loaded.Audio.NoiseSuppression {
		t.Fatalf("migrated settings = %#v, want schema v%d with RNNoise enabled", loaded.Audio, SchemaVersion)
	}
}

func TestLoadPreservesExplicitRNNoiseOffInV6Settings(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	path, err := service.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	current := `{
  "schemaVersion": 6,
  "locale": "zh-CN",
  "audio": {
    "microphone": true,
    "noiseSuppression": false,
    "microphoneGain": 1
  }
}`
	if err := os.WriteFile(path, []byte(current), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Audio.NoiseSuppression {
		t.Fatalf("RNNoise preference = true, want explicit v6 off preserved")
	}
}

func TestSavePreservesNewPIPMinimumScale(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))

	saved, err := service.Save(Settings{
		Camera: CameraSettings{
			PIPPreset: string(pip.PresetBottomRight),
			PIP: pip.Config{
				Preset:      pip.PresetBottomRight,
				Shape:       pip.ShapeCircle,
				Mirror:      true,
				Position:    pip.Position{X: 1, Y: 1},
				Scale:       pip.MinimumScale,
				EdgeFeather: pip.DefaultEdgeFeather,
			},
		},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.Camera.PIP.Scale != pip.MinimumScale {
		t.Fatalf("minimum scale = %v, want preserved %v", saved.Camera.PIP.Scale, pip.MinimumScale)
	}

	loaded, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Camera.PIP.Scale != pip.MinimumScale {
		t.Fatalf("loaded minimum scale = %v, want preserved %v", loaded.Camera.PIP.Scale, pip.MinimumScale)
	}
}

func TestSaveNormalizesInvalidSettings(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))

	saved, err := service.Save(Settings{
		Locale:    Locale("fr"),
		Recording: RecordingSettings{Quality: "cinema", FPS: 120, CountdownSeconds: -4},
		Audio:     AudioSettings{MicrophoneGain: -2},
		Camera:    CameraSettings{PIPPreset: "top-right", PIP: pip.Config{Shape: pip.Shape("triangle"), Scale: 3, EdgeFeather: 2}},
		Whiteboard: WhiteboardSettings{
			LastMode:        "floating",
			LastTool:        "spray",
			LastStrokeWidth: "giant",
			LastOpacity:     120,
			CapturePolicy:   "capture-window",
		},
		Shortcuts: ShortcutSettings{
			ToggleRecording: "",
			TogglePause:     "R",
			ToggleCamera:    "Shift+C",
			OpenWhiteboard:  "CmdOrCtrl+Shift+B",
		},
		OCR: OCRSettings{
			Translation: OCRTranslationSettings{
				Provider:           "cloud-mystery",
				SourceLanguage:     "",
				TargetLanguage:     "",
				PrivacyConfirmed:   true,
				PrivacyConfirmedAt: "2026-07-06T00:00:00Z",
			},
		},
		Window: WindowSettings{Theme: Theme("neon")},
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
	if saved.Camera.PIP.Shape != pip.DefaultShape || saved.Camera.PIP.Scale != pip.MaximumScale || saved.Camera.PIP.EdgeFeather != pip.MaximumEdgeFeather {
		t.Fatalf("normalized pip = %#v, want default shape and clamped ratios", saved.Camera.PIP)
	}
	if saved.Window.Theme != ThemeNightTeal {
		t.Fatalf("invalid theme normalized to %q, want %q", saved.Window.Theme, ThemeNightTeal)
	}
	if saved.Whiteboard.LastMode != "board" || saved.Whiteboard.LastTool != "freedraw" || saved.Whiteboard.LastStrokeWidth != "medium" || saved.Whiteboard.LastOpacity != 100 || saved.Whiteboard.CapturePolicy != "export-compose" {
		t.Fatalf("normalized whiteboard = %#v, want defaults with opacity clamped", saved.Whiteboard)
	}
	if saved.Shortcuts != DefaultShortcuts() {
		t.Fatalf("invalid shortcuts normalized to %#v, want defaults %#v", saved.Shortcuts, DefaultShortcuts())
	}
	if saved.OCR.Translation.Provider != "disabled" || saved.OCR.Translation.SourceLanguage != "auto" || saved.OCR.Translation.TargetLanguage != "zh-CN" || saved.OCR.Translation.PrivacyConfirmed || saved.OCR.Translation.PrivacyConfirmedAt != "" {
		t.Fatalf("invalid OCR translation normalized to %#v, want disabled defaults", saved.OCR.Translation)
	}
}

func TestValidateShortcutsRejectsDuplicatesAndPlainKeys(t *testing.T) {
	duplicates := DefaultShortcuts()
	duplicates.TogglePause = duplicates.ToggleRecording
	if _, err := ValidateShortcuts(duplicates); err == nil {
		t.Fatal("ValidateShortcuts() should reject duplicate shortcuts")
	}
	plain := DefaultShortcuts()
	plain.ToggleRecording = "R"
	if _, err := ValidateShortcuts(plain); err == nil {
		t.Fatal("ValidateShortcuts() should reject plain letter shortcuts")
	}
	shiftOnly := DefaultShortcuts()
	shiftOnly.ToggleRecording = "Shift+R"
	if _, err := ValidateShortcuts(shiftOnly); err == nil {
		t.Fatal("ValidateShortcuts() should reject shift-only printable shortcuts")
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
