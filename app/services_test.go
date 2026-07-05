package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/preflight"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
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

func TestPatchSettingsPreferencesPersistsRecordingAndTheme(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	originalSyncStartAtLogin := syncStartAtLogin
	var syncedStartAtLogin []bool
	syncStartAtLogin = func(enabled bool) error {
		syncedStartAtLogin = append(syncedStartAtLogin, enabled)
		return nil
	}
	t.Cleanup(func() {
		syncStartAtLogin = originalSyncStartAtLogin
	})
	theme := settings.ThemeSageGray
	quality := recordingprofile.QualityHigh
	fps := 60
	captureCursor := false
	countdown := 5
	startAtLogin := true

	saved, err := service.PatchSettingsPreferences(SettingsPreferencesPatchRequest{
		Theme:            &theme,
		RecordingQuality: &quality,
		RecordingFPS:     &fps,
		CaptureCursor:    &captureCursor,
		CountdownSeconds: &countdown,
		StartAtLogin:     &startAtLogin,
	})
	if err != nil {
		t.Fatalf("PatchSettingsPreferences() error = %v", err)
	}
	if saved.Window.Theme != theme || !saved.Window.StartAtLogin || saved.Recording.Quality != quality || saved.Recording.FPS != fps || saved.Recording.CaptureCursor || saved.Recording.CountdownSeconds != countdown {
		t.Fatalf("saved preferences = theme %q recording %#v, want patched preferences", saved.Window.Theme, saved.Recording)
	}
	if len(syncedStartAtLogin) != 1 || !syncedStartAtLogin[0] {
		t.Fatalf("start at login sync calls = %#v, want enabled once", syncedStartAtLogin)
	}
	loaded, err := service.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if loaded.Window.Theme != theme || !loaded.Window.StartAtLogin || loaded.Recording.Quality != quality || loaded.Recording.FPS != fps || loaded.Recording.CaptureCursor || loaded.Recording.CountdownSeconds != countdown {
		t.Fatalf("loaded preferences = theme %q recording %#v, want patched preferences", loaded.Window.Theme, loaded.Recording)
	}
}

func TestPatchSettingsPreferencesRejectsStartAtLoginWhenSystemSyncFails(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	originalSyncStartAtLogin := syncStartAtLogin
	syncStartAtLogin = func(enabled bool) error {
		return errors.New("login item denied")
	}
	t.Cleanup(func() {
		syncStartAtLogin = originalSyncStartAtLogin
	})
	startAtLogin := true

	if _, err := service.PatchSettingsPreferences(SettingsPreferencesPatchRequest{StartAtLogin: &startAtLogin}); err == nil {
		t.Fatal("PatchSettingsPreferences() accepted a failed start at login sync")
	}
	loaded, err := service.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if loaded.Window.StartAtLogin {
		t.Fatal("start at login was persisted even though system sync failed")
	}
}

func TestPatchWhiteboardSettingsPersistsWhiteboardOnly(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	theme := settings.ThemeSageGray
	quality := recordingprofile.QualityHigh
	if _, err := service.PatchSettingsPreferences(SettingsPreferencesPatchRequest{
		Theme:            &theme,
		RecordingQuality: &quality,
	}); err != nil {
		t.Fatalf("PatchSettingsPreferences() error = %v", err)
	}
	tool := "rectangle"
	color := "#38bdf8"
	width := "bold"
	opacity := 140

	saved, err := service.PatchWhiteboardSettings(WhiteboardSettingsPatchRequest{
		LastTool:        &tool,
		LastStrokeColor: &color,
		LastStrokeWidth: &width,
		LastOpacity:     &opacity,
	})
	if err != nil {
		t.Fatalf("PatchWhiteboardSettings() error = %v", err)
	}
	if saved.Whiteboard.LastTool != tool || saved.Whiteboard.LastStrokeColor != color || saved.Whiteboard.LastStrokeWidth != width || saved.Whiteboard.LastOpacity != 100 {
		t.Fatalf("whiteboard settings = %#v, want patched values with opacity clamped", saved.Whiteboard)
	}
	if saved.Window.Theme != theme || saved.Recording.Quality != quality {
		t.Fatalf("whiteboard patch changed unrelated preferences: theme %q recording %#v", saved.Window.Theme, saved.Recording)
	}
}

func TestPatchShortcutSettingsPersistsShortcutsOnly(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	theme := settings.ThemeSageGray
	quality := recordingprofile.QualityHigh
	if _, err := service.PatchSettingsPreferences(SettingsPreferencesPatchRequest{
		Theme:            &theme,
		RecordingQuality: &quality,
	}); err != nil {
		t.Fatalf("PatchSettingsPreferences() error = %v", err)
	}
	next := "CmdOrCtrl+OptionOrAlt+R"

	saved, err := service.PatchShortcutSettings(ShortcutSettingsPatchRequest{
		ToggleRecording: &next,
	})
	if err != nil {
		t.Fatalf("PatchShortcutSettings() error = %v", err)
	}
	if saved.Shortcuts.ToggleRecording != next {
		t.Fatalf("toggle recording shortcut = %q, want %q", saved.Shortcuts.ToggleRecording, next)
	}
	if saved.Window.Theme != theme || saved.Recording.Quality != quality {
		t.Fatalf("shortcut patch changed unrelated preferences: theme %q recording %#v", saved.Window.Theme, saved.Recording)
	}
	loaded, err := service.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if loaded.Shortcuts.ToggleRecording != next {
		t.Fatalf("loaded toggle recording shortcut = %q, want %q", loaded.Shortcuts.ToggleRecording, next)
	}
}

func TestPatchShortcutSettingsRejectsDuplicate(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	duplicate := settings.DefaultShortcuts().TogglePause
	if _, err := service.PatchShortcutSettings(ShortcutSettingsPatchRequest{ToggleRecording: &duplicate}); err == nil {
		t.Fatal("PatchShortcutSettings() should reject duplicate shortcuts")
	}
}

func TestSaveSettingsDoesNotOverwritePatchedPreferences(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	theme := settings.ThemeSunsetYellow
	quality := recordingprofile.QualityHigh
	fps := 60
	shortcut := "CmdOrCtrl+OptionOrAlt+B"
	if _, err := service.PatchSettingsPreferences(SettingsPreferencesPatchRequest{
		Theme:            &theme,
		RecordingQuality: &quality,
		RecordingFPS:     &fps,
	}); err != nil {
		t.Fatalf("PatchSettingsPreferences() error = %v", err)
	}
	if _, err := service.PatchShortcutSettings(ShortcutSettingsPatchRequest{OpenWhiteboard: &shortcut}); err != nil {
		t.Fatalf("PatchShortcutSettings() error = %v", err)
	}
	current, err := service.settings.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	current.Window.StartAtLogin = true
	if _, err := service.settings.Save(current); err != nil {
		t.Fatalf("Save(start at login) error = %v", err)
	}
	stale := settings.Default()
	stale.Locale = settings.LocaleEN
	stale.Window.Theme = settings.ThemeSkyBlue
	stale.Window.StartAtLogin = false
	stale.Recording.Quality = recordingprofile.QualityStandard
	stale.Recording.FPS = 24
	stale.Shortcuts.OpenWhiteboard = "CmdOrCtrl+Shift+W"
	saved, err := service.SaveSettings(stale)
	if err != nil {
		t.Fatalf("SaveSettings(stale) error = %v", err)
	}
	if saved.Locale != settings.LocaleEN {
		t.Fatalf("locale = %q, want SaveSettings to still persist unrelated settings", saved.Locale)
	}
	if saved.Window.Theme != theme || saved.Recording.Quality != quality || saved.Recording.FPS != fps {
		t.Fatalf("SaveSettings overwrote patched preferences: theme %q recording %#v", saved.Window.Theme, saved.Recording)
	}
	if !saved.Window.StartAtLogin {
		t.Fatal("SaveSettings overwrote patched start at login preference")
	}
	if saved.Shortcuts.OpenWhiteboard != shortcut {
		t.Fatalf("SaveSettings overwrote patched shortcut: %q, want %q", saved.Shortcuts.OpenWhiteboard, shortcut)
	}
}

func TestPatchCameraStatePersistsCameraIntent(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	enabled := true
	deviceID := "camera:dshow:hd-webcam"
	preset := string(pip.PresetFree)
	pipConfig := pip.Config{
		Preset:      pip.PresetFree,
		Shape:       pip.ShapeSquare,
		Mirror:      false,
		Position:    pip.Position{X: 0.25, Y: 0.75},
		Scale:       pip.MaximumScale,
		EdgeFeather: 0.2,
	}
	saved, err := service.PatchCameraState(CameraStatePatchRequest{
		Enabled:   &enabled,
		DeviceID:  &deviceID,
		PIPPreset: &preset,
		PIP:       &pipConfig,
	})
	if err != nil {
		t.Fatalf("PatchCameraState() error = %v", err)
	}
	if !saved.Camera.Enabled || saved.Camera.DeviceID != deviceID {
		t.Fatalf("saved camera = %#v, want enabled %q", saved.Camera, deviceID)
	}
	if saved.Camera.PIPPreset != string(pip.PresetFree) || saved.Camera.PIP.Shape != pip.ShapeSquare || saved.Camera.PIP.Mirror {
		t.Fatalf("saved camera pip = %#v, want free square non-mirrored", saved.Camera)
	}
	loaded, err := service.settings.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded.Camera.Enabled || loaded.Camera.DeviceID != deviceID || loaded.Camera.PIPPreset != string(pip.PresetFree) {
		t.Fatalf("loaded camera = %#v, want patched camera intent", loaded.Camera)
	}
}

func TestSaveSettingsDoesNotOverwritePatchedCameraState(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		settings: settings.NewService(data),
	}
	enabled := true
	deviceID := "camera:dshow:hd-webcam"
	if _, err := service.PatchCameraState(CameraStatePatchRequest{
		Enabled:  &enabled,
		DeviceID: &deviceID,
	}); err != nil {
		t.Fatalf("PatchCameraState() error = %v", err)
	}
	stale := settings.Default()
	stale.Locale = settings.LocaleEN
	stale.Camera.Enabled = false
	stale.Camera.DeviceID = "camera:default"
	stale.Camera.PIPPreset = string(pip.PresetOff)
	stale.Camera.PIP = pip.OffConfig()
	saved, err := service.SaveSettings(stale)
	if err != nil {
		t.Fatalf("SaveSettings(stale) error = %v", err)
	}
	if saved.Locale != settings.LocaleEN {
		t.Fatalf("locale = %q, want SaveSettings to still persist unrelated settings", saved.Locale)
	}
	if !saved.Camera.Enabled || saved.Camera.DeviceID != deviceID {
		t.Fatalf("SaveSettings overwrote patched camera: %#v", saved.Camera)
	}
	if saved.Camera.PIPPreset == string(pip.PresetOff) || saved.Camera.PIP.Preset == pip.PresetOff {
		t.Fatalf("SaveSettings disabled patched camera pip: %#v", saved.Camera)
	}
}

func TestSaveWhiteboardExportWritesExcalidrawScene(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{appData: data}

	result, err := service.SaveWhiteboardExport(WhiteboardExportRequest{
		Format:  "excalidraw",
		Payload: `{"type":"excalidraw","version":2,"source":"RecordingFreedom","elements":[],"appState":{},"files":{}}`,
	})
	if err != nil {
		t.Fatalf("SaveWhiteboardExport(excalidraw) error = %v", err)
	}
	if result.Format != "excalidraw" || filepath.Ext(result.OutputPath) != ".excalidraw" {
		t.Fatalf("export result = %#v, want .excalidraw output", result)
	}
	root, err := data.RootDir()
	if err != nil {
		t.Fatalf("RootDir() error = %v", err)
	}
	rel, err := filepath.Rel(root, result.OutputPath)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		t.Fatalf("export path %q escaped root %q", result.OutputPath, root)
	}
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Fatalf("export file was not written: %v", err)
	}
}

func TestSaveWhiteboardExportRejectsInvalidExcalidrawScene(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{appData: data}

	if _, err := service.SaveWhiteboardExport(WhiteboardExportRequest{Format: "excalidraw", Payload: "{invalid"}); err == nil {
		t.Fatal("SaveWhiteboardExport(excalidraw invalid JSON) succeeded, want error")
	}
}

func TestSaveAnnotationCaptureWritesActivePackageAnnotations(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(recpackage.NewService())),
	}
	session, err := service.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		SourceName: "Primary",
		SourceGeometry: &recording.SourceGeometry{
			Width:  1280,
			Height: 720,
		},
		Recording: recordingprofile.Default(),
	})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	annotationBounds := application.Rect{X: 80, Y: 90, Width: 640, Height: 360}
	service.setAnnotationRegionDIP(session.ID, annotationBounds)

	result, err := service.SaveAnnotationCapture(AnnotationCaptureRequest{
		SceneJSON:       `{"type":"excalidraw","elements":[],"appState":{},"files":{}}`,
		SnapshotDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png")),
	})
	if err != nil {
		t.Fatalf("SaveAnnotationCapture() error = %v", err)
	}
	if result.PackageDir != session.PackageDir {
		t.Fatalf("annotation packageDir = %q, want active package %q", result.PackageDir, session.PackageDir)
	}
	for _, path := range []string{result.ScenePath, result.EventsPath, result.SnapshotPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("annotation artifact %q missing: %v", path, err)
		}
	}
	if result.TimelineSnapshotPath == "" {
		t.Fatal("TimelineSnapshotPath is empty, want per-save annotation snapshot")
	}
	if _, err := os.Stat(result.TimelineSnapshotPath); err != nil {
		t.Fatalf("timeline annotation snapshot %q missing: %v", result.TimelineSnapshotPath, err)
	}
	manifest, err := recpackage.NewService().ReadManifest(session.Manifest)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Annotations == nil || !manifest.Annotations.Enabled || manifest.Annotations.SnapshotPath != recpackage.AnnotationSnapshotFile || manifest.Annotations.DiagnosticsPath != recpackage.AnnotationOverlayDiagnosticsFile {
		t.Fatalf("manifest annotations = %#v, want enabled snapshot contract", manifest.Annotations)
	}
	if manifest.Annotations.Target.Type != annotationRegionTargetType || manifest.Annotations.Target.ID != annotationRegionTargetID ||
		manifest.Annotations.Target.Geometry == nil ||
		manifest.Annotations.Target.Geometry.X != annotationBounds.X ||
		manifest.Annotations.Target.Geometry.Y != annotationBounds.Y ||
		manifest.Annotations.Target.Geometry.Width != annotationBounds.Width ||
		manifest.Annotations.Target.Geometry.Height != annotationBounds.Height {
		t.Fatalf("annotation target = %#v, want selected annotation region", manifest.Annotations.Target)
	}
	loaded, err := service.LoadAnnotationCapture()
	if err != nil {
		t.Fatalf("LoadAnnotationCapture() error = %v", err)
	}
	if !loaded.Available || loaded.ScenePath != result.ScenePath || !strings.Contains(loaded.SceneJSON, `"type":"excalidraw"`) {
		t.Fatalf("loaded annotation = %#v, want saved active scene", loaded)
	}
	eventsData, err := os.ReadFile(result.EventsPath)
	if err != nil {
		t.Fatalf("ReadFile(annotation events) error = %v", err)
	}
	var event map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(eventsData), &event); err != nil {
		t.Fatalf("annotation event JSON = %q, error = %v", eventsData, err)
	}
	snapshotPath, _ := event["snapshotPath"].(string)
	if event["type"] != "scene-snapshot" || event["scenePath"] != recpackage.AnnotationSceneFile || !strings.HasPrefix(snapshotPath, recpackage.AnnotationSnapshotsDir+"/") {
		t.Fatalf("annotation event = %#v, want scene snapshot with timeline snapshot path", event)
	}
	if _, ok := event["recordingOffsetMs"].(float64); !ok {
		t.Fatalf("annotation event = %#v, want recordingOffsetMs", event)
	}
	diagnosticsPath := filepath.Join(session.PackageDir, recpackage.AnnotationOverlayDiagnosticsFile)
	diagnosticsData, err := os.ReadFile(diagnosticsPath)
	if err != nil {
		t.Fatalf("ReadFile(annotation overlay diagnostics) error = %v", err)
	}
	if !strings.Contains(string(diagnosticsData), `"type":"save-capture"`) || !strings.Contains(string(diagnosticsData), `"windowBounds"`) {
		t.Fatalf("overlay diagnostics = %s, want save-capture bounds evidence", diagnosticsData)
	}
}

func TestClearAnnotationCaptureForSessionRemovesArtifactsAndRegion(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(recpackage.NewService())),
	}
	session, err := service.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		SourceGeometry: &recording.SourceGeometry{
			Width:  1280,
			Height: 720,
		},
		Recording: recordingprofile.Default(),
	})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	service.setAnnotationRegionDIP(session.ID, application.Rect{X: 80, Y: 90, Width: 640, Height: 360})
	result, err := service.SaveAnnotationCapture(AnnotationCaptureRequest{
		SceneJSON:       `{"type":"excalidraw","elements":[{"id":"a","type":"freedraw"}],"appState":{},"files":{}}`,
		SnapshotDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png")),
	})
	if err != nil {
		t.Fatalf("SaveAnnotationCapture() error = %v", err)
	}
	if _, err := os.Stat(result.ScenePath); err != nil {
		t.Fatalf("annotation scene missing before clear: %v", err)
	}

	if err := service.clearAnnotationCaptureForSession(session); err != nil {
		t.Fatalf("clearAnnotationCaptureForSession() error = %v", err)
	}
	service.clearAnnotationRegionDIP(session.ID)
	if _, err := os.Stat(filepath.Join(session.PackageDir, recpackage.AnnotationsDir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("annotations dir stat error = %v, want not exist", err)
	}
	manifest, err := recpackage.NewService().ReadManifest(session.Manifest)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Annotations != nil {
		t.Fatalf("manifest annotations = %#v, want nil after clear", manifest.Annotations)
	}
	if _, ok := service.annotationRegionDIPForSession(session.ID); ok {
		t.Fatal("annotation region still present after clear")
	}
}

func TestSaveAnnotationCaptureUsesRecordingOffsetAfterPauseResume(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(recpackage.NewService())),
	}
	session, err := service.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Recording:  recordingprofile.Default(),
	})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	service.setAnnotationRegionDIP(session.ID, application.Rect{X: 0, Y: 0, Width: 800, Height: 450})
	if _, err := service.recorder.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	time.Sleep(80 * time.Millisecond)
	if _, err := service.recorder.Resume(); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}

	result, err := service.SaveAnnotationCapture(AnnotationCaptureRequest{
		SceneJSON:       `{"type":"excalidraw","elements":[{"id":"a","type":"freedraw"}],"appState":{},"files":{}}`,
		SnapshotDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png")),
	})
	if err != nil {
		t.Fatalf("SaveAnnotationCapture() error = %v", err)
	}
	eventsData, err := os.ReadFile(result.EventsPath)
	if err != nil {
		t.Fatalf("ReadFile(annotation events) error = %v", err)
	}
	var event map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(eventsData), &event); err != nil {
		t.Fatalf("annotation event JSON = %q, error = %v", eventsData, err)
	}
	wallOffsetMs, wallOK := event["wallOffsetMs"].(float64)
	recordingOffsetMs, recordingOK := event["recordingOffsetMs"].(float64)
	if !wallOK || !recordingOK {
		t.Fatalf("annotation event = %#v, want wallOffsetMs and recordingOffsetMs", event)
	}
	if recordingOffsetMs >= wallOffsetMs {
		t.Fatalf("recordingOffsetMs = %.0f, wallOffsetMs = %.0f, want recording offset to subtract pause duration", recordingOffsetMs, wallOffsetMs)
	}
	if wallOffsetMs-recordingOffsetMs < 40 {
		t.Fatalf("offset delta = %.0fms, want pause duration reflected in annotation event", wallOffsetMs-recordingOffsetMs)
	}
}

func TestSaveAnnotationCaptureUsesWhiteboardCapturePolicy(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	settingsService := settings.NewService(data)
	current := settings.Default()
	current.Whiteboard.CapturePolicy = "preview-only"
	if _, err := settingsService.Save(current); err != nil {
		t.Fatalf("Save(settings) error = %v", err)
	}
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(recpackage.NewService())),
		settings: settingsService,
	}
	session, err := service.StartRecording(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Recording:  recordingprofile.Default(),
	})
	if err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}
	service.setAnnotationRegionDIP(session.ID, application.Rect{X: 0, Y: 0, Width: 800, Height: 450})
	if _, err := service.SaveAnnotationCapture(AnnotationCaptureRequest{
		SceneJSON:       `{"type":"excalidraw","elements":[],"appState":{},"files":{}}`,
		SnapshotDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png")),
	}); err != nil {
		t.Fatalf("SaveAnnotationCapture() error = %v", err)
	}
	manifest, err := recpackage.NewService().ReadManifest(session.Manifest)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Annotations == nil || manifest.Annotations.CapturePolicy != "preview-only" {
		t.Fatalf("manifest annotations = %#v, want preview-only policy from settings", manifest.Annotations)
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

func TestPreviewExportRecordingPackageReturnsAnnotationPlan(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	packageDir := createReadyExportPackage(t, info.VideoDir, true)
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewService(data),
	}

	result, err := service.PreviewExportRecordingPackage(ExportRecordingRequest{
		PackageDir: packageDir,
	})
	if err != nil {
		t.Fatalf("PreviewExportRecordingPackage() error = %v", err)
	}
	if result.Plan.OutputPath != filepath.Join(packageDir, exportplan.DefaultOutputPath) {
		t.Fatalf("output path = %q, want package default export", result.Plan.OutputPath)
	}
	if !result.Plan.AnnotationsVisible || result.Plan.AnnotationTimeline != "snapshot-segments" {
		t.Fatalf("annotation plan = visible:%v timeline:%q, want snapshot segment preview", result.Plan.AnnotationsVisible, result.Plan.AnnotationTimeline)
	}
	if len(result.Plan.AnnotationSnapshots) != 2 {
		t.Fatalf("annotation snapshots = %#v, want two preview segments", result.Plan.AnnotationSnapshots)
	}
	if result.Plan.AnnotationSummary == nil || result.Plan.AnnotationSummary.SnapshotCount != 2 || result.Plan.AnnotationSummary.ElementEventCount != 1 {
		t.Fatalf("annotation summary = %#v, want snapshot and element counts", result.Plan.AnnotationSummary)
	}
}

func TestPreviewExportRecordingPackageHonorsAnnotationToggle(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	packageDir := createReadyExportPackage(t, info.VideoDir, true)
	includeAnnotations := false
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewService(data),
	}

	result, err := service.PreviewExportRecordingPackage(ExportRecordingRequest{
		PackageDir:         packageDir,
		IncludeAnnotations: &includeAnnotations,
	})
	if err != nil {
		t.Fatalf("PreviewExportRecordingPackage(disabled annotations) error = %v", err)
	}
	if result.Plan.AnnotationsVisible || len(result.Plan.AnnotationSnapshots) != 0 || result.Plan.AnnotationSummary != nil {
		t.Fatalf("annotation plan = visible:%v snapshots:%#v summary:%#v, want annotations skipped", result.Plan.AnnotationsVisible, result.Plan.AnnotationSnapshots, result.Plan.AnnotationSummary)
	}
}

func TestReadAnnotationPreviewImageReturnsPackageSnapshotDataURL(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	packageDir := createReadyExportPackage(t, info.VideoDir, true)
	service := &RecordingFreedomService{appData: data}

	result, err := service.ReadAnnotationPreviewImage(AnnotationPreviewImageRequest{
		PackageDir:   packageDir,
		SnapshotPath: filepath.ToSlash(filepath.Join(recpackage.AnnotationSnapshotsDir, "annotation-000001.png")),
	})
	if err != nil {
		t.Fatalf("ReadAnnotationPreviewImage() error = %v", err)
	}
	if !result.Available || result.RelativePath != "annotations/snapshots/annotation-000001.png" || result.Bytes <= 0 {
		t.Fatalf("result = %#v, want readable package annotation snapshot", result)
	}
	wantDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("timeline annotation png"))
	if result.DataURL != wantDataURL {
		t.Fatalf("data URL = %q, want encoded annotation snapshot", result.DataURL)
	}
}

func TestReadAnnotationPreviewImageReturnsRenderedPNGDataURL(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	packageDir := createReadyExportPackage(t, info.VideoDir, true)
	renderPath := filepath.Join(packageDir, recpackage.AnnotationRenderPNGDir, "annotation-000001.png")
	if err := os.MkdirAll(filepath.Dir(renderPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(render path) error = %v", err)
	}
	if err := os.WriteFile(renderPath, []byte("rendered annotation png"), 0o644); err != nil {
		t.Fatalf("WriteFile(rendered annotation) error = %v", err)
	}
	service := &RecordingFreedomService{appData: data}

	result, err := service.ReadAnnotationPreviewImage(AnnotationPreviewImageRequest{
		PackageDir:   packageDir,
		SnapshotPath: filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, "annotation-000001.png")),
	})
	if err != nil {
		t.Fatalf("ReadAnnotationPreviewImage(rendered) error = %v", err)
	}
	if !result.Available || result.RelativePath != "annotations/reconstructed/png/annotation-000001.png" || result.Bytes <= 0 {
		t.Fatalf("result = %#v, want readable rendered annotation PNG", result)
	}
	wantDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("rendered annotation png"))
	if result.DataURL != wantDataURL {
		t.Fatalf("data URL = %q, want encoded rendered annotation", result.DataURL)
	}
}

func TestReadAnnotationPreviewImageRejectsEscapingSnapshotPath(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	packageDir := createReadyExportPackage(t, info.VideoDir, true)
	outside := filepath.Join(t.TempDir(), "annotation.png")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}
	service := &RecordingFreedomService{appData: data}

	if _, err := service.ReadAnnotationPreviewImage(AnnotationPreviewImageRequest{
		PackageDir:   packageDir,
		SnapshotPath: outside,
	}); err == nil {
		t.Fatal("ReadAnnotationPreviewImage() accepted an absolute path outside the package")
	}
	if _, err := service.ReadAnnotationPreviewImage(AnnotationPreviewImageRequest{
		PackageDir:   packageDir,
		SnapshotPath: recpackage.ScreenVideoFile,
	}); err == nil {
		t.Fatal("ReadAnnotationPreviewImage() accepted a non-annotation package path")
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

func TestRecoverRecordingPackagePreservesAnnotationExportPlan(t *testing.T) {
	data := appdata.NewService(t.TempDir())
	info, err := data.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	packageDir := createReadyExportPackage(t, info.VideoDir, true)
	manifestPath := filepath.Join(packageDir, recpackage.ManifestFile)
	packageService := recpackage.NewService()
	manifest, err := packageService.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	manifest.Status = recpackage.StatusRecording
	manifest.CompletedAt = nil
	if err := packageService.WriteManifest(manifestPath, manifest); err != nil {
		t.Fatalf("WriteManifest(recording) error = %v", err)
	}
	service := &RecordingFreedomService{
		appData:  data,
		recorder: recording.NewServiceWithBackend(data, recording.NewMockBackend(packageService)),
	}

	summary, err := service.RecoverRecordingPackage(packageDir)
	if err != nil {
		t.Fatalf("RecoverRecordingPackage() error = %v", err)
	}
	if summary.Status != recpackage.StatusReady || summary.Recoverable {
		t.Fatalf("summary = %#v, want ready recovered package", summary)
	}
	recovered, err := packageService.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest(recovered) error = %v", err)
	}
	if !recovered.Diagnostics.Recovered {
		t.Fatal("recovered manifest must mark diagnostics.recovered")
	}
	if recovered.Annotations == nil || !recovered.Annotations.Enabled || recovered.Annotations.EventsPath != recpackage.AnnotationEventsFile {
		t.Fatalf("recovered annotations = %#v, want preserved annotation contract", recovered.Annotations)
	}

	preview, err := service.PreviewExportRecordingPackage(ExportRecordingRequest{PackageDir: packageDir})
	if err != nil {
		t.Fatalf("PreviewExportRecordingPackage(recovered annotations) error = %v", err)
	}
	if !preview.Plan.AnnotationsVisible || preview.Plan.AnnotationTimeline != "snapshot-segments" || len(preview.Plan.AnnotationSnapshots) != 2 {
		t.Fatalf("annotation plan = visible:%v timeline:%q snapshots:%#v, want recovered snapshot annotation timeline", preview.Plan.AnnotationsVisible, preview.Plan.AnnotationTimeline, preview.Plan.AnnotationSnapshots)
	}
	if preview.Plan.AnnotationSummary == nil || preview.Plan.AnnotationSummary.EventCount != 3 || preview.Plan.AnnotationSummary.ExportedSnapshotCount != 2 {
		t.Fatalf("annotation summary = %#v, want recovered annotation events and snapshots", preview.Plan.AnnotationSummary)
	}
}

func createReadyExportPackage(t *testing.T, videoDir string, annotations bool) string {
	t.Helper()
	packageDir := filepath.Join(videoDir, "recording-preview-plan-test"+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(package) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, recpackage.ScreenVideoFile), []byte("screen media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen) error = %v", err)
	}
	manifest := recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		CreatedAt:     time.Date(2026, 7, 3, 22, 0, 0, 0, time.UTC),
		Status:        recpackage.StatusReady,
		RecordingMode: recpackage.RecordingModeScreen,
		Media:         recpackage.ManifestMedia{ScreenVideoPath: recpackage.ScreenVideoFile},
		Source:        recpackage.ManifestSource{Type: "screen", ID: "screen:primary"},
		Recording:     recordingprofile.Profile{Quality: recordingprofile.QualityHigh, FPS: 30, CaptureCursor: true},
		Diagnostics: recpackage.ManifestDiagnostics{
			Sync: &recpackage.ManifestSyncDiagnostics{
				TimelineBase: recpackage.TimelineBaseMedia,
				Screen: recpackage.ManifestTrackDiagnostics{
					Enabled:     true,
					Path:        recpackage.ScreenVideoFile,
					Clock:       recpackage.TimelineBaseMedia,
					EndOffsetMs: 6000,
					DurationMs:  6000,
					FrameRate:   30,
				},
			},
		},
	}
	if annotations {
		if err := os.MkdirAll(filepath.Join(packageDir, recpackage.AnnotationExportsDir), 0o755); err != nil {
			t.Fatalf("MkdirAll(annotation exports) error = %v", err)
		}
		if err := os.MkdirAll(filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir), 0o755); err != nil {
			t.Fatalf("MkdirAll(annotation snapshots) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(packageDir, recpackage.AnnotationSnapshotFile), []byte("final annotation png"), 0o644); err != nil {
			t.Fatalf("WriteFile(annotation final snapshot) error = %v", err)
		}
		for _, name := range []string{"annotation-000001.png", "annotation-000003.png"} {
			if err := os.WriteFile(filepath.Join(packageDir, recpackage.AnnotationSnapshotsDir, name), []byte("timeline annotation png"), 0o644); err != nil {
				t.Fatalf("WriteFile(annotation timeline snapshot) error = %v", err)
			}
		}
		events := strings.Join([]string{
			`{"type":"scene-snapshot","recordingOffsetMs":1000,"snapshotPath":"annotations/snapshots/annotation-000001.png"}`,
			`{"type":"element-created","elementId":"a","recordingOffsetMs":1000}`,
			`{"type":"scene-snapshot","recordingOffsetMs":2500,"snapshotPath":"annotations/snapshots/annotation-000003.png"}`,
		}, "\n")
		if err := os.WriteFile(filepath.Join(packageDir, recpackage.AnnotationEventsFile), []byte(events+"\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(annotation events) error = %v", err)
		}
		manifest.Annotations = &recpackage.ManifestAnnotations{
			Enabled:       true,
			Mode:          "overlay",
			ScenePath:     recpackage.AnnotationSceneFile,
			EventsPath:    recpackage.AnnotationEventsFile,
			SnapshotPath:  recpackage.AnnotationSnapshotFile,
			CapturePolicy: "export-compose",
			Target:        recpackage.ManifestAnnotationTarget{Type: "screen", ID: "screen:primary"},
		}
	}
	if err := recpackage.NewService().WriteManifest(filepath.Join(packageDir, recpackage.ManifestFile), manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}
	return packageDir
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
