package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	secretstore "github.com/lemon-casino/RecordingFreedom/app/internal/secrets"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

const ocrTranslationAPIKeySecretName = "ocr.translation.api-key"

func (s *RecordingFreedomService) loadSettingsForClient() (settings.Settings, error) {
	current, err := s.loadSettingsForMutation()
	if err != nil {
		return settings.Settings{}, err
	}
	return sanitizeSettingsForClient(current), nil
}

func (s *RecordingFreedomService) loadSettingsForMutation() (settings.Settings, error) {
	if s == nil || s.settings == nil {
		return settings.Settings{}, errors.New("settings service is not initialized")
	}
	current, err := s.settings.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	current, err = s.migrateLegacyOCRTranslationAPIKey(current)
	if err != nil {
		return settings.Settings{}, err
	}
	return current, nil
}

func (s *RecordingFreedomService) migrateLegacyOCRTranslationAPIKey(current settings.Settings) (settings.Settings, error) {
	legacyKey := strings.TrimSpace(current.OCR.Translation.APIKey)
	if legacyKey == "" {
		current.OCR.Translation.APIKey = ""
		return current, nil
	}
	store := s.ocrTranslationSecretStore()
	if err := store.Save(ocrTranslationAPIKeySecretName, legacyKey); err != nil {
		return settings.Settings{}, fmt.Errorf("migrate OCR translation API key: %w", err)
	}
	current.OCR.Translation.APIKey = ""
	current.OCR.Translation.APIKeySet = true
	saved, err := s.settings.Save(current)
	if err != nil {
		return settings.Settings{}, err
	}
	return saved, nil
}

func sanitizeSettingsForClient(current settings.Settings) settings.Settings {
	if strings.TrimSpace(current.OCR.Translation.APIKey) != "" {
		current.OCR.Translation.APIKeySet = true
	}
	current.OCR.Translation.APIKey = ""
	return current
}

func (s *RecordingFreedomService) ocrTranslationSecretStore() *secretstore.Store {
	if s.secrets == nil {
		s.secrets = secretstore.NewStore(s.appData)
	}
	return s.secrets
}

func (s *RecordingFreedomService) saveOCRTranslationAPIKey(secret string) (bool, error) {
	secret = strings.TrimSpace(secret)
	store := s.ocrTranslationSecretStore()
	if secret == "" {
		if err := store.Delete(ocrTranslationAPIKeySecretName); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := store.Save(ocrTranslationAPIKeySecretName, secret); err != nil {
		return false, err
	}
	return true, nil
}

func (s *RecordingFreedomService) loadOCRTranslationAPIKey() (string, bool, error) {
	return s.ocrTranslationSecretStore().Load(ocrTranslationAPIKeySecretName)
}

func (s *RecordingFreedomService) resolveOCRTranslateRequest(req ocr.TranslateRequest) (ocr.TranslateRequest, error) {
	req.Provider = strings.TrimSpace(req.Provider)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.APIKey = strings.TrimSpace(req.APIKey)
	req.Model = strings.TrimSpace(req.Model)
	req.SourceLanguage = strings.TrimSpace(req.SourceLanguage)
	req.TargetLanguage = strings.TrimSpace(req.TargetLanguage)
	if req.Provider == "" || req.Provider == "disabled" || req.APIKey != "" {
		return req, nil
	}
	current, err := s.loadSettingsForMutation()
	if err != nil {
		return req, err
	}
	translation := current.OCR.Translation
	if strings.TrimSpace(translation.Provider) != req.Provider || !translation.APIKeySet {
		return req, nil
	}
	if req.BaseURL == "" {
		req.BaseURL = strings.TrimSpace(translation.BaseURL)
	} else if req.BaseURL != strings.TrimSpace(translation.BaseURL) {
		return req, nil
	}
	if req.Model == "" {
		req.Model = strings.TrimSpace(translation.Model)
	} else if req.Provider == "openai-compatible" && req.Model != strings.TrimSpace(translation.Model) {
		return req, nil
	}
	if req.SourceLanguage == "" {
		req.SourceLanguage = strings.TrimSpace(translation.SourceLanguage)
	}
	if req.TargetLanguage == "" {
		req.TargetLanguage = strings.TrimSpace(translation.TargetLanguage)
	}
	apiKey, ok, err := s.loadOCRTranslationAPIKey()
	if err != nil {
		return req, err
	}
	if ok {
		req.APIKey = apiKey
	}
	return req, nil
}
