package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

const fileName = "settings.json"

const (
	pipScaleSchemaVersion = 2
	legacyPIPMinimumScale = 0.016
	legacyPIPMaximumScale = 0.08
)

type Service struct {
	appData *appdata.Service
	now     func() time.Time
}

func NewService(appData *appdata.Service) *Service {
	return &Service{
		appData: appData,
		now:     time.Now,
	}
}

func (s *Service) Path() (string, error) {
	root, err := s.appData.RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, fileName), nil
}

func (s *Service) Load() (Settings, error) {
	path, err := s.Path()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Settings{}, err
	}

	var result Settings
	if err := json.Unmarshal(data, &result); err != nil {
		return Settings{}, err
	}
	result = migrateLoadedSettings(result)
	return normalize(result), nil
}

func (s *Service) Save(next Settings) (Settings, error) {
	path, err := s.Path()
	if err != nil {
		return Settings{}, err
	}

	next = normalizeForSave(next)
	next.UpdatedAt = s.now().UTC()
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return Settings{}, err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return Settings{}, err
	}
	if err := replaceFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return Settings{}, err
	}
	return next, nil
}

func Default() Settings {
	return Settings{
		SchemaVersion: SchemaVersion,
		Locale:        LocaleZhCN,
		Source: SourceSettings{
			LastSourceType: "screen",
		},
		Recording: recordingprofile.Default(),
		Audio: AudioSettings{
			System:             false,
			SystemDeviceID:     "system-audio:default",
			Microphone:         false,
			MicrophoneDeviceID: "microphone:default",
			NoiseSuppression:   false,
			MicrophoneGain:     1,
		},
		Camera: CameraSettings{
			Enabled:   false,
			DeviceID:  "camera:default",
			PIPPreset: string(pip.DefaultPreset),
			PIP:       pip.DefaultConfig(),
		},
		Whiteboard: WhiteboardSettings{
			Enabled:         true,
			LastMode:        "board",
			LastTool:        "freedraw",
			LastStrokeColor: "#ef4444",
			LastStrokeWidth: "medium",
			LastOpacity:     100,
			CapturePolicy:   "export-compose",
		},
		OCR: OCRSettings{
			AutoRecognizeScreenshots: false,
			Translation: OCRTranslationSettings{
				Provider:       "disabled",
				SourceLanguage: "auto",
				TargetLanguage: "zh-CN",
			},
		},
		Shortcuts: DefaultShortcuts(),
		Window: WindowSettings{
			MinimizeToTray: true,
			Theme:          ThemeNightTeal,
		},
	}
}

func normalize(value Settings) Settings {
	defaults := Default()
	if value.SchemaVersion < SchemaVersion {
		value.SchemaVersion = SchemaVersion
	}
	if value.Locale == "" || !validLocale(value.Locale) {
		value.Locale = defaults.Locale
	}
	if value.Source.LastSourceType == "" {
		value.Source.LastSourceType = defaults.Source.LastSourceType
	}
	value.Storage.DataRootDir = strings.TrimSpace(value.Storage.DataRootDir)
	value.Recording = recordingprofile.Normalize(value.Recording)
	if value.Audio.SystemDeviceID == "" {
		value.Audio.SystemDeviceID = defaults.Audio.SystemDeviceID
	}
	if value.Audio.MicrophoneDeviceID == "" {
		value.Audio.MicrophoneDeviceID = defaults.Audio.MicrophoneDeviceID
	}
	if value.Audio.MicrophoneGain <= 0 {
		value.Audio.MicrophoneGain = defaults.Audio.MicrophoneGain
	}
	if value.Camera.DeviceID == "" {
		value.Camera.DeviceID = defaults.Camera.DeviceID
	}
	value.Camera.PIP = pip.NormalizeConfigForPreset(value.Camera.PIPPreset, value.Camera.PIP)
	value.Camera.PIPPreset = string(value.Camera.PIP.Preset)
	value.Whiteboard = normalizeWhiteboard(value.Whiteboard, defaults.Whiteboard)
	value.OCR = normalizeOCR(value.OCR, defaults.OCR)
	value.Shortcuts = normalizeShortcuts(value.Shortcuts, defaults.Shortcuts)
	if value.Window.Theme == "" || !validTheme(value.Window.Theme) {
		value.Window.Theme = defaults.Window.Theme
	}
	return value
}

func normalizeForSave(value Settings) Settings {
	value = normalize(value)
	if strings.TrimSpace(value.OCR.Translation.APIKey) != "" {
		value.OCR.Translation.APIKeySet = true
	}
	value.OCR.Translation.APIKey = ""
	return value
}

func normalizeOCR(value OCRSettings, defaults OCRSettings) OCRSettings {
	value.Translation = normalizeOCRTranslation(value.Translation, defaults.Translation)
	return value
}

func normalizeOCRTranslation(value OCRTranslationSettings, defaults OCRTranslationSettings) OCRTranslationSettings {
	value.Provider = strings.TrimSpace(value.Provider)
	if !validOCRTranslationProvider(value.Provider) {
		value.Provider = defaults.Provider
	}
	value.BaseURL = strings.TrimSpace(value.BaseURL)
	value.APIKey = strings.TrimSpace(value.APIKey)
	if value.APIKey != "" {
		value.APIKeySet = true
	}
	value.Model = strings.TrimSpace(value.Model)
	value.SourceLanguage = strings.TrimSpace(value.SourceLanguage)
	value.TargetLanguage = strings.TrimSpace(value.TargetLanguage)
	value.PrivacyConfirmedAt = strings.TrimSpace(value.PrivacyConfirmedAt)
	if value.SourceLanguage == "" {
		value.SourceLanguage = defaults.SourceLanguage
	}
	if value.TargetLanguage == "" {
		value.TargetLanguage = defaults.TargetLanguage
	}
	if value.Provider == "disabled" {
		value.PrivacyConfirmed = false
		value.PrivacyConfirmedAt = ""
	}
	if !value.PrivacyConfirmed {
		value.PrivacyConfirmedAt = ""
	}
	return value
}

func validOCRTranslationProvider(provider string) bool {
	switch provider {
	case "disabled", "deepl", "openai-compatible":
		return true
	default:
		return false
	}
}

func migrateLoadedSettings(value Settings) Settings {
	if value.SchemaVersion >= pipScaleSchemaVersion {
		return value
	}
	value.Camera.PIP.Scale = migrateLegacyPIPScale(value.Camera.PIP.Scale)
	return value
}

func migrateLegacyPIPScale(value float64) float64 {
	if value <= 0 {
		return value
	}
	value = clampSettingsFloat(value, legacyPIPMinimumScale, legacyPIPMaximumScale)
	progress := (value - legacyPIPMinimumScale) / (legacyPIPMaximumScale - legacyPIPMinimumScale)
	return pip.MinimumScale + progress*(pip.MaximumScale-pip.MinimumScale)
}

func clampSettingsFloat(value float64, minimum float64, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func normalizeWhiteboard(value WhiteboardSettings, defaults WhiteboardSettings) WhiteboardSettings {
	if value.LastMode != "board" && value.LastMode != "annotation" {
		value.LastMode = defaults.LastMode
	}
	if !validWhiteboardTool(value.LastTool) {
		value.LastTool = defaults.LastTool
	}
	if strings.TrimSpace(value.LastStrokeColor) == "" {
		value.LastStrokeColor = defaults.LastStrokeColor
	}
	if value.LastStrokeWidth != "thin" && value.LastStrokeWidth != "medium" && value.LastStrokeWidth != "bold" {
		value.LastStrokeWidth = defaults.LastStrokeWidth
	}
	if value.LastOpacity <= 0 {
		value.LastOpacity = defaults.LastOpacity
	}
	if value.LastOpacity < 5 {
		value.LastOpacity = 5
	}
	if value.LastOpacity > 100 {
		value.LastOpacity = 100
	}
	if value.CapturePolicy != "preview-only" && value.CapturePolicy != "export-compose" {
		value.CapturePolicy = defaults.CapturePolicy
	}
	return value
}

func validWhiteboardTool(tool string) bool {
	switch tool {
	case "selection", "hand", "freedraw", "laser", "arrow", "line", "rectangle", "ellipse", "text", "eraser":
		return true
	default:
		return false
	}
}

func validLocale(locale Locale) bool {
	switch locale {
	case LocaleZhCN, LocaleEN:
		return true
	default:
		return false
	}
}

func validTheme(theme Theme) bool {
	switch theme {
	case ThemeNightTeal, ThemeMountainGreen, ThemeSkyBlue, ThemeSunsetYellow, ThemeInkPurple, ThemeSageGray:
		return true
	default:
		return false
	}
}

func DefaultShortcuts() ShortcutSettings {
	return ShortcutSettings{
		ToggleRecording: "CmdOrCtrl+Shift+R",
		TogglePause:     "CmdOrCtrl+Shift+P",
		ToggleCamera:    "CmdOrCtrl+Shift+C",
		OpenWhiteboard:  "CmdOrCtrl+Shift+B",
		OpenScreenshot:  "CmdOrCtrl+Shift+S",
	}
}

func ValidateShortcuts(value ShortcutSettings) (ShortcutSettings, error) {
	normalized, err := normalizeShortcutValues(value)
	if err != nil {
		return ShortcutSettings{}, err
	}
	if err := validateShortcutUniqueness(normalized); err != nil {
		return ShortcutSettings{}, err
	}
	return normalized, nil
}

func ShortcutBindings(value ShortcutSettings) []ShortcutBinding {
	normalized := normalizeShortcuts(value, DefaultShortcuts())
	return []ShortcutBinding{
		{Action: ShortcutActionToggleRecording, Accelerator: normalized.ToggleRecording},
		{Action: ShortcutActionTogglePause, Accelerator: normalized.TogglePause},
		{Action: ShortcutActionToggleCamera, Accelerator: normalized.ToggleCamera},
		{Action: ShortcutActionOpenWhiteboard, Accelerator: normalized.OpenWhiteboard},
		{Action: ShortcutActionOpenScreenshot, Accelerator: normalized.OpenScreenshot},
	}
}

func normalizeShortcuts(value ShortcutSettings, defaults ShortcutSettings) ShortcutSettings {
	normalized, err := normalizeShortcutValues(ShortcutSettings{
		ToggleRecording: shortcutOrDefault(value.ToggleRecording, defaults.ToggleRecording),
		TogglePause:     shortcutOrDefault(value.TogglePause, defaults.TogglePause),
		ToggleCamera:    shortcutOrDefault(value.ToggleCamera, defaults.ToggleCamera),
		OpenWhiteboard:  shortcutOrDefault(value.OpenWhiteboard, defaults.OpenWhiteboard),
		OpenScreenshot:  shortcutOrDefault(value.OpenScreenshot, defaults.OpenScreenshot),
	})
	if err != nil {
		return defaults
	}
	return resolveShortcutDuplicateDefaults(normalized, defaults)
}

func shortcutOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func normalizeShortcutValues(value ShortcutSettings) (ShortcutSettings, error) {
	var err error
	if value.ToggleRecording, err = NormalizeShortcutAccelerator(value.ToggleRecording); err != nil {
		return ShortcutSettings{}, fmt.Errorf("%s: %w", ShortcutActionToggleRecording, err)
	}
	if value.TogglePause, err = NormalizeShortcutAccelerator(value.TogglePause); err != nil {
		return ShortcutSettings{}, fmt.Errorf("%s: %w", ShortcutActionTogglePause, err)
	}
	if value.ToggleCamera, err = NormalizeShortcutAccelerator(value.ToggleCamera); err != nil {
		return ShortcutSettings{}, fmt.Errorf("%s: %w", ShortcutActionToggleCamera, err)
	}
	if value.OpenWhiteboard, err = NormalizeShortcutAccelerator(value.OpenWhiteboard); err != nil {
		return ShortcutSettings{}, fmt.Errorf("%s: %w", ShortcutActionOpenWhiteboard, err)
	}
	if value.OpenScreenshot, err = NormalizeShortcutAccelerator(value.OpenScreenshot); err != nil {
		return ShortcutSettings{}, fmt.Errorf("%s: %w", ShortcutActionOpenScreenshot, err)
	}
	return value, nil
}

func resolveShortcutDuplicateDefaults(value ShortcutSettings, defaults ShortcutSettings) ShortcutSettings {
	seen := map[string]ShortcutAction{}
	for _, binding := range ShortcutBindingsRaw(value) {
		identity := shortcutIdentity(binding.Accelerator)
		if identity == "" {
			value = setShortcutValue(value, binding.Action, defaultShortcutForAction(defaults, binding.Action))
			identity = shortcutIdentity(defaultShortcutForAction(defaults, binding.Action))
		}
		if _, duplicate := seen[identity]; duplicate {
			fallback := defaultShortcutForAction(defaults, binding.Action)
			fallbackIdentity := shortcutIdentity(fallback)
			if _, fallbackDuplicate := seen[fallbackIdentity]; fallbackIdentity != "" && !fallbackDuplicate {
				value = setShortcutValue(value, binding.Action, fallback)
				seen[fallbackIdentity] = binding.Action
			} else {
				value = setShortcutValue(value, binding.Action, "")
			}
			continue
		}
		seen[identity] = binding.Action
	}
	return value
}

func validateShortcutUniqueness(value ShortcutSettings) error {
	seen := map[string]ShortcutAction{}
	for _, binding := range ShortcutBindingsRaw(value) {
		identity := shortcutIdentity(binding.Accelerator)
		if identity == "" {
			return fmt.Errorf("%s shortcut is required", binding.Action)
		}
		if existing, duplicate := seen[identity]; duplicate {
			return fmt.Errorf("%s conflicts with %s", binding.Action, existing)
		}
		seen[identity] = binding.Action
	}
	return nil
}

func ShortcutBindingsRaw(value ShortcutSettings) []ShortcutBinding {
	return []ShortcutBinding{
		{Action: ShortcutActionToggleRecording, Accelerator: value.ToggleRecording},
		{Action: ShortcutActionTogglePause, Accelerator: value.TogglePause},
		{Action: ShortcutActionToggleCamera, Accelerator: value.ToggleCamera},
		{Action: ShortcutActionOpenWhiteboard, Accelerator: value.OpenWhiteboard},
		{Action: ShortcutActionOpenScreenshot, Accelerator: value.OpenScreenshot},
	}
}

func defaultShortcutForAction(defaults ShortcutSettings, action ShortcutAction) string {
	switch action {
	case ShortcutActionToggleRecording:
		return defaults.ToggleRecording
	case ShortcutActionTogglePause:
		return defaults.TogglePause
	case ShortcutActionToggleCamera:
		return defaults.ToggleCamera
	case ShortcutActionOpenWhiteboard:
		return defaults.OpenWhiteboard
	case ShortcutActionOpenScreenshot:
		return defaults.OpenScreenshot
	default:
		return ""
	}
}

func setShortcutValue(value ShortcutSettings, action ShortcutAction, accelerator string) ShortcutSettings {
	switch action {
	case ShortcutActionToggleRecording:
		value.ToggleRecording = accelerator
	case ShortcutActionTogglePause:
		value.TogglePause = accelerator
	case ShortcutActionToggleCamera:
		value.ToggleCamera = accelerator
	case ShortcutActionOpenWhiteboard:
		value.OpenWhiteboard = accelerator
	case ShortcutActionOpenScreenshot:
		value.OpenScreenshot = accelerator
	}
	return value
}

func NormalizeShortcutAccelerator(value string) (string, error) {
	parts := strings.Split(strings.TrimSpace(value), "+")
	if len(parts) == 0 || strings.TrimSpace(value) == "" {
		return "", errors.New("shortcut is required")
	}
	modifiers := map[string]bool{}
	for _, part := range parts[:len(parts)-1] {
		modifier, ok := normalizeShortcutModifier(part)
		if !ok {
			return "", fmt.Errorf("%q is not a supported modifier", strings.TrimSpace(part))
		}
		modifiers[modifier] = true
	}
	key, ok := normalizeShortcutKey(parts[len(parts)-1])
	if !ok {
		return "", fmt.Errorf("%q is not a supported key", strings.TrimSpace(parts[len(parts)-1]))
	}
	if len(modifiers) == 0 {
		return "", errors.New("shortcut must include a modifier")
	}
	if len(modifiers) == 1 && modifiers["Shift"] && isPrintableShortcutKey(key) {
		return "", errors.New("shortcut must include Ctrl, Command, Alt, or Super")
	}
	ordered := make([]string, 0, len(modifiers)+1)
	for _, modifier := range []string{"CmdOrCtrl", "Ctrl", "OptionOrAlt", "Shift", "Super"} {
		if modifiers[modifier] {
			ordered = append(ordered, modifier)
		}
	}
	ordered = append(ordered, key)
	return strings.Join(ordered, "+"), nil
}

func normalizeShortcutModifier(value string) (string, bool) {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", "")) {
	case "cmdorctrl", "cmd", "command", "meta":
		return "CmdOrCtrl", true
	case "ctrl", "control":
		return "Ctrl", true
	case "optionoralt", "option", "alt":
		return "OptionOrAlt", true
	case "shift":
		return "Shift", true
	case "super", "win", "windows":
		return "Super", true
	default:
		return "", false
	}
}

func normalizeShortcutKey(value string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(value))
	key = strings.ReplaceAll(key, " ", "")
	switch key {
	case "esc":
		key = "escape"
	case "del":
		key = "delete"
	case "return":
		key = "enter"
	case "pageup":
		key = "page up"
	case "pagedown":
		key = "page down"
	case "plus":
		return "Plus", true
	}
	if len([]rune(key)) == 1 {
		return strings.ToUpper(key), true
	}
	if _, ok := shortcutNamedKeys[key]; ok {
		return shortcutDisplayKey(key), true
	}
	return "", false
}

func shortcutDisplayKey(key string) string {
	if strings.HasPrefix(key, "f") && len(key) <= 3 {
		return strings.ToUpper(key)
	}
	words := strings.Fields(key)
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	if len(words) > 0 {
		return strings.Join(words, " ")
	}
	return strings.ToUpper(key[:1]) + key[1:]
}

func isPrintableShortcutKey(key string) bool {
	return len([]rune(key)) == 1 || key == "Plus" || key == "Space"
}

var shortcutNamedKeys = map[string]struct{}{
	"backspace": {}, "tab": {}, "enter": {}, "escape": {}, "left": {}, "right": {}, "up": {}, "down": {},
	"space": {}, "delete": {}, "home": {}, "end": {}, "page up": {}, "page down": {}, "numlock": {},
	"f1": {}, "f2": {}, "f3": {}, "f4": {}, "f5": {}, "f6": {}, "f7": {}, "f8": {}, "f9": {}, "f10": {}, "f11": {}, "f12": {},
	"f13": {}, "f14": {}, "f15": {}, "f16": {}, "f17": {}, "f18": {}, "f19": {}, "f20": {}, "f21": {}, "f22": {}, "f23": {}, "f24": {},
	"f25": {}, "f26": {}, "f27": {}, "f28": {}, "f29": {}, "f30": {}, "f31": {}, "f32": {}, "f33": {}, "f34": {}, "f35": {},
}

func shortcutIdentity(value string) string {
	normalized, err := NormalizeShortcutAccelerator(value)
	if err != nil {
		return ""
	}
	parts := strings.Split(normalized, "+")
	if len(parts) == 0 {
		return ""
	}
	key := strings.ToLower(parts[len(parts)-1])
	modifiers := make([]string, 0, len(parts)-1)
	for _, part := range parts[:len(parts)-1] {
		if identity, ok := shortcutPlatformModifierIdentity(part); ok {
			modifiers = append(modifiers, identity)
		}
	}
	sort.Strings(modifiers)
	modifiers = append(modifiers, key)
	return strings.Join(modifiers, "+")
}

func shortcutPlatformModifierIdentity(value string) (string, bool) {
	modifier, ok := normalizeShortcutModifier(value)
	if !ok {
		return "", false
	}
	switch runtime.GOOS {
	case "darwin":
		switch modifier {
		case "CmdOrCtrl", "Super":
			return "cmd", true
		case "Ctrl":
			return "ctrl", true
		case "OptionOrAlt":
			return "alt", true
		case "Shift":
			return "shift", true
		}
	default:
		switch modifier {
		case "CmdOrCtrl", "Ctrl":
			return "ctrl", true
		case "OptionOrAlt":
			return "alt", true
		case "Shift":
			return "shift", true
		case "Super":
			return "super", true
		}
	}
	return "", false
}

func replaceFile(tmp string, target string) error {
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tmp, target)
}
