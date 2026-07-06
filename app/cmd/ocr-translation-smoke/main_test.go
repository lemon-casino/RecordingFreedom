package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesTranslationSmokeEvidenceWithoutRawSecret(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "data-root")
	evidenceDir := filepath.Join(t.TempDir(), "evidence")

	report, err := run(dataRoot, evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.OK {
		t.Fatalf("report.OK = false: %#v", report)
	}
	if report.Provider != smokeProvider || report.Model != smokeModel || report.TargetLanguage != smokeTargetLanguage {
		t.Fatalf("translation report = %#v, want configured provider/model/target", report)
	}
	if report.ProviderMode != providerModeLocal || !report.ProviderRequestCountKnown || report.ProviderRequestCount != 1 || !report.ProviderAuthHeaderOK || !report.ProviderRequestVerified || !report.CacheHitAfterProviderDown || report.ExternalProviderCacheHitNoAPIKey {
		t.Fatalf("provider/cache report = %#v, want one verified request and cache hit", report)
	}
	if len(report.TranslationFiles) == 0 || len(report.TranslatedBlocks) != 2 {
		t.Fatalf("translation artifacts = files %#v blocks %#v, want cache file and two blocks", report.TranslationFiles, report.TranslatedBlocks)
	}
	data, err := os.ReadFile(filepath.Join(evidenceDir, evidenceFileName))
	if err != nil {
		t.Fatalf("ReadFile(evidence) error = %v", err)
	}
	if strings.Contains(string(data), "rf-translation-smoke-secret") {
		t.Fatalf("evidence contains raw translation API key: %s", data)
	}
	var decoded smokeReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("evidence JSON is invalid: %v", err)
	}
	if !decoded.OK || decoded.OcrResultID != "ocr_translation_smoke_result" {
		t.Fatalf("decoded evidence = %#v, want successful OCR translation smoke report", decoded)
	}
}

func TestRunExternalOpenAICompatibleRequiresAPIKeyEnv(t *testing.T) {
	keyEnv := "RF_TEST_OCR_TRANSLATION_API_KEY_MISSING"
	t.Setenv(keyEnv, "")

	_, err := runSmoke(runOptions{
		DataRoot:       filepath.Join(t.TempDir(), "data-root"),
		EvidenceDir:    filepath.Join(t.TempDir(), "evidence"),
		ProviderMode:   providerModeExternalOpenAI,
		BaseURL:        "https://translation.example.invalid/v1",
		Model:          "external-smoke-model",
		APIKeyEnv:      keyEnv,
		SourceLanguage: smokeSourceLanguage,
		TargetLanguage: smokeTargetLanguage,
		ForceProvider:  true,
	})
	if err == nil {
		t.Fatal("runSmoke() succeeded without external provider API key")
	}
	if !strings.Contains(err.Error(), keyEnv) {
		t.Fatalf("runSmoke() error = %v, want missing key env name", err)
	}
}

func TestRunExternalOpenAICompatibleWritesSafeEvidenceAndHitsCacheWithoutAPIKey(t *testing.T) {
	keyEnv := "RF_TEST_OCR_TRANSLATION_API_KEY"
	apiKey := "rf-external-openai-compatible-secret"
	t.Setenv(keyEnv, apiKey)
	recorder := newProviderRecorder(apiKey, "external-smoke-model", smokeTargetLanguage)
	server := httptest.NewServer(recorder.handler())
	defer server.Close()

	report, err := runSmoke(runOptions{
		DataRoot:       filepath.Join(t.TempDir(), "data-root"),
		EvidenceDir:    filepath.Join(t.TempDir(), "evidence"),
		ProviderMode:   providerModeExternalOpenAI,
		BaseURL:        server.URL,
		Model:          "external-smoke-model",
		APIKeyEnv:      keyEnv,
		SourceLanguage: smokeSourceLanguage,
		TargetLanguage: smokeTargetLanguage,
		ForceProvider:  true,
	})
	if err != nil {
		t.Fatalf("runSmoke() error = %v", err)
	}
	if !report.OK || report.ProviderMode != providerModeExternalOpenAI || !report.ExternalProviderRequestForced || !report.ExternalProviderCacheHitNoAPIKey {
		t.Fatalf("external provider report = %#v, want forced request and cache hit without API key", report)
	}
	if report.ProviderBaseURL != server.URL {
		t.Fatalf("ProviderBaseURL = %q, want provider URL", report.ProviderBaseURL)
	}
	if strings.Contains(report.ProviderBaseURL, apiKey) {
		t.Fatalf("ProviderBaseURL contains raw API key: %q", report.ProviderBaseURL)
	}
	if got := sanitizeBaseURLForReport(server.URL + "/v1?api_key=" + apiKey + "#fragment"); got != server.URL+"/v1" {
		t.Fatalf("sanitizeBaseURLForReport() = %q, want URL without query/fragment", got)
	}
	snapshot := recorder.snapshot()
	if snapshot.count != 1 {
		t.Fatalf("provider request count = %d, want one forced provider request before cache hit", snapshot.count)
	}
	if !snapshot.authHeaderOK || !snapshot.requestVerified {
		t.Fatalf("provider snapshot = %#v, want verified Authorization header and prompt body", snapshot)
	}
	data, err := os.ReadFile(report.EvidencePath)
	if err != nil {
		t.Fatalf("ReadFile(evidence) error = %v", err)
	}
	if strings.Contains(string(data), apiKey) {
		t.Fatalf("evidence contains raw external provider API key: %s", data)
	}
	if strings.Contains(string(data), "RecordingFreedom 已翻译") {
		t.Fatalf("evidence should not persist full translated text: %s", data)
	}
}
