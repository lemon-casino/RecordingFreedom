package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/autostart"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

type SettingsPreferencesPatchRequest struct {
	Locale           *settings.Locale                       `json:"locale,omitempty"`
	Theme            *settings.Theme                        `json:"theme,omitempty"`
	RecordingQuality *string                                `json:"recordingQuality,omitempty"`
	RecordingFPS     *int                                   `json:"recordingFps,omitempty"`
	CaptureCursor    *bool                                  `json:"captureCursor,omitempty"`
	CountdownSeconds *int                                   `json:"countdownSeconds,omitempty"`
	StartAtLogin     *bool                                  `json:"startAtLogin,omitempty"`
	AutoOCR          *bool                                  `json:"autoOcr,omitempty"`
	OCRTranslation   *OCRTranslationPreferencesPatchRequest `json:"ocrTranslation,omitempty"`
}

type OCRTranslationPreferencesPatchRequest struct {
	Provider           *string `json:"provider,omitempty"`
	BaseURL            *string `json:"baseUrl,omitempty"`
	APIKey             *string `json:"apiKey,omitempty"`
	Model              *string `json:"model,omitempty"`
	SourceLanguage     *string `json:"sourceLanguage,omitempty"`
	TargetLanguage     *string `json:"targetLanguage,omitempty"`
	PrivacyConfirmed   *bool   `json:"privacyConfirmed,omitempty"`
	PrivacyConfirmedAt *string `json:"privacyConfirmedAt,omitempty"`
}

var syncStartAtLogin = autostart.SetEnabled

func (s *RecordingFreedomService) PatchSettingsPreferences(patch SettingsPreferencesPatchRequest) (settings.Settings, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	currentSettings, err := s.loadSettingsForMutation()
	if err != nil {
		return settings.Settings{}, err
	}
	if patch.Theme != nil {
		currentSettings.Window.Theme = *patch.Theme
	}
	if patch.Locale != nil {
		currentSettings.Locale = *patch.Locale
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
	if patch.StartAtLogin != nil {
		if err := syncStartAtLogin(*patch.StartAtLogin); err != nil {
			s.logEvent("settings-preferences", "start-at-login-error", map[string]string{
				"enabled": strconv.FormatBool(*patch.StartAtLogin),
				"error":   err.Error(),
			})
			return settings.Settings{}, fmt.Errorf("set start at login: %w", err)
		}
		currentSettings.Window.StartAtLogin = *patch.StartAtLogin
	}
	if patch.AutoOCR != nil {
		currentSettings.OCR.AutoRecognizeScreenshots = *patch.AutoOCR
	}
	if patch.OCRTranslation != nil {
		nextTranslation := applyOCRTranslationPreferencesPatch(currentSettings.OCR.Translation, *patch.OCRTranslation)
		if patch.OCRTranslation.APIKey != nil {
			apiKeySet, err := s.saveOCRTranslationAPIKey(*patch.OCRTranslation.APIKey)
			if err != nil {
				return settings.Settings{}, fmt.Errorf("save OCR translation API key: %w", err)
			}
			nextTranslation.APIKeySet = apiKeySet
		}
		nextTranslation.APIKey = ""
		currentSettings.OCR.Translation = nextTranslation
	}
	saved, err := s.settings.Save(currentSettings)
	if err != nil {
		return settings.Settings{}, err
	}
	sanitized := sanitizeSettingsForClient(saved)
	s.refreshTrayLocale(saved.Locale)
	s.logEvent("settings-preferences", "patch", settingsPreferencesPatchFields(patch, sanitized))
	s.emitSettingsChanged(sanitized)
	return sanitized, nil
}

func settingsPreferencesPatchFields(patch SettingsPreferencesPatchRequest, saved settings.Settings) map[string]string {
	fields := map[string]string{
		"savedLocale":           string(saved.Locale),
		"savedTheme":            string(saved.Window.Theme),
		"savedRecordingQuality": saved.Recording.Quality,
		"savedRecordingFps":     strconv.Itoa(saved.Recording.FPS),
		"savedCaptureCursor":    strconv.FormatBool(saved.Recording.CaptureCursor),
		"savedCountdownSeconds": strconv.Itoa(saved.Recording.CountdownSeconds),
		"savedStartAtLogin":     strconv.FormatBool(saved.Window.StartAtLogin),
		"savedAutoOcr":          strconv.FormatBool(saved.OCR.AutoRecognizeScreenshots),
	}
	if patch.Locale != nil {
		fields["locale"] = string(*patch.Locale)
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
	if patch.StartAtLogin != nil {
		fields["startAtLogin"] = strconv.FormatBool(*patch.StartAtLogin)
	}
	if patch.AutoOCR != nil {
		fields["autoOcr"] = strconv.FormatBool(*patch.AutoOCR)
	}
	if patch.OCRTranslation != nil {
		fields["ocrTranslationProvider"] = saved.OCR.Translation.Provider
		fields["ocrTranslationApiKeySet"] = strconv.FormatBool(saved.OCR.Translation.APIKeySet)
		fields["ocrTranslationPrivacyConfirmed"] = strconv.FormatBool(saved.OCR.Translation.PrivacyConfirmed)
		if patch.OCRTranslation.Provider != nil {
			fields["ocrTranslationProviderRequested"] = strings.TrimSpace(*patch.OCRTranslation.Provider)
		}
	}
	return fields
}

func applyOCRTranslationPreferencesPatch(current settings.OCRTranslationSettings, patch OCRTranslationPreferencesPatchRequest) settings.OCRTranslationSettings {
	if patch.Provider != nil {
		current.Provider = strings.TrimSpace(*patch.Provider)
	}
	if patch.BaseURL != nil {
		current.BaseURL = strings.TrimSpace(*patch.BaseURL)
	}
	if patch.APIKey != nil {
		current.APIKey = strings.TrimSpace(*patch.APIKey)
	}
	if patch.Model != nil {
		current.Model = strings.TrimSpace(*patch.Model)
	}
	if patch.SourceLanguage != nil {
		current.SourceLanguage = strings.TrimSpace(*patch.SourceLanguage)
	}
	if patch.TargetLanguage != nil {
		current.TargetLanguage = strings.TrimSpace(*patch.TargetLanguage)
	}
	if patch.PrivacyConfirmed != nil {
		current.PrivacyConfirmed = *patch.PrivacyConfirmed
		if current.PrivacyConfirmed && strings.TrimSpace(current.PrivacyConfirmedAt) == "" {
			current.PrivacyConfirmedAt = timeNowUTCString()
		}
		if !current.PrivacyConfirmed {
			current.PrivacyConfirmedAt = ""
		}
	}
	if patch.PrivacyConfirmedAt != nil {
		current.PrivacyConfirmedAt = strings.TrimSpace(*patch.PrivacyConfirmedAt)
	}
	return current
}

func timeNowUTCString() string {
	return strings.TrimSpace(time.Now().UTC().Format(time.RFC3339Nano))
}
