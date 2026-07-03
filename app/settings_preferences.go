package main

import (
	"strconv"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

type SettingsPreferencesPatchRequest struct {
	Theme            *settings.Theme `json:"theme,omitempty"`
	RecordingQuality *string         `json:"recordingQuality,omitempty"`
	RecordingFPS     *int            `json:"recordingFps,omitempty"`
	CaptureCursor    *bool           `json:"captureCursor,omitempty"`
	CountdownSeconds *int            `json:"countdownSeconds,omitempty"`
}

func (s *RecordingFreedomService) PatchSettingsPreferences(patch SettingsPreferencesPatchRequest) (settings.Settings, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	currentSettings, err := s.settings.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	if patch.Theme != nil {
		currentSettings.Window.Theme = *patch.Theme
	}
	if patch.RecordingQuality != nil {
		currentSettings.Recording.Quality = strings.TrimSpace(*patch.RecordingQuality)
	}
	if patch.RecordingFPS != nil {
		currentSettings.Recording.FPS = *patch.RecordingFPS
	}
	if patch.CaptureCursor != nil {
		currentSettings.Recording.CaptureCursor = *patch.CaptureCursor
	}
	if patch.CountdownSeconds != nil {
		currentSettings.Recording.CountdownSeconds = *patch.CountdownSeconds
	}
	saved, err := s.settings.Save(currentSettings)
	if err != nil {
		return settings.Settings{}, err
	}
	s.logEvent("settings-preferences", "patch", settingsPreferencesPatchFields(patch, saved))
	s.emitSettingsChanged(saved)
	return saved, nil
}

func settingsPreferencesPatchFields(patch SettingsPreferencesPatchRequest, saved settings.Settings) map[string]string {
	fields := map[string]string{
		"savedTheme":            string(saved.Window.Theme),
		"savedRecordingQuality": saved.Recording.Quality,
		"savedRecordingFps":     strconv.Itoa(saved.Recording.FPS),
		"savedCaptureCursor":    strconv.FormatBool(saved.Recording.CaptureCursor),
		"savedCountdownSeconds": strconv.Itoa(saved.Recording.CountdownSeconds),
	}
	if patch.Theme != nil {
		fields["theme"] = string(*patch.Theme)
	}
	if patch.RecordingQuality != nil {
		fields["recordingQuality"] = strings.TrimSpace(*patch.RecordingQuality)
	}
	if patch.RecordingFPS != nil {
		fields["recordingFps"] = strconv.Itoa(*patch.RecordingFPS)
	}
	if patch.CaptureCursor != nil {
		fields["captureCursor"] = strconv.FormatBool(*patch.CaptureCursor)
	}
	if patch.CountdownSeconds != nil {
		fields["countdownSeconds"] = strconv.Itoa(*patch.CountdownSeconds)
	}
	return fields
}
