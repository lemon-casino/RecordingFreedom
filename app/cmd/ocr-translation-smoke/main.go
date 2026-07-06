package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	secretstore "github.com/lemon-casino/RecordingFreedom/app/internal/secrets"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

const (
	ocrTranslationSecretName     = "ocr.translation.api-key"
	evidenceFileName             = "translation-smoke.json"
	providerModeLocal            = "local"
	providerModeExternalOpenAI   = "external-openai-compatible"
	smokeProvider                = "openai-compatible"
	smokeModel                   = "rf-translation-smoke"
	smokeSourceLanguage          = "auto"
	smokeTargetLanguage          = "zh-CN"
	defaultTranslationAPIKeyEnv  = "RF_OCR_TRANSLATION_API_KEY"
	defaultTranslationBaseURLEnv = "RF_OCR_TRANSLATION_BASE_URL"
	defaultTranslationModelEnv   = "RF_OCR_TRANSLATION_MODEL"
	localSmokeAPIKey             = "rf-translation-smoke-secret"
)

type runOptions struct {
	DataRoot       string
	EvidenceDir    string
	ProviderMode   string
	BaseURL        string
	Model          string
	APIKeyEnv      string
	SourceLanguage string
	TargetLanguage string
	ForceProvider  bool
}

type translationSettingsInput struct {
	BaseURL        string
	APIKey         string
	Model          string
	SourceLanguage string
	TargetLanguage string
}

type smokeReport struct {
	SchemaVersion                    int                        `json:"schemaVersion"`
	OK                               bool                       `json:"ok"`
	GeneratedAt                      time.Time                  `json:"generatedAt"`
	DataRoot                         string                     `json:"dataRoot"`
	EvidencePath                     string                     `json:"evidencePath"`
	SettingsPath                     string                     `json:"settingsPath"`
	SettingsJSONContainsRawKey       bool                       `json:"settingsJsonContainsRawKey"`
	SecretBackend                    string                     `json:"secretBackend"`
	SecretStatus                     string                     `json:"secretStatus"`
	ProviderMode                     string                     `json:"providerMode"`
	Provider                         string                     `json:"provider"`
	ProviderBaseURL                  string                     `json:"providerBaseUrl"`
	Model                            string                     `json:"model"`
	APIKeyEnv                        string                     `json:"apiKeyEnv"`
	APIKeyProvided                   bool                       `json:"apiKeyProvided"`
	SourceLanguage                   string                     `json:"sourceLanguage"`
	TargetLanguage                   string                     `json:"targetLanguage"`
	OcrResultID                      string                     `json:"ocrResultId"`
	BlockCount                       int                        `json:"blockCount"`
	FirstRequestForced               bool                       `json:"firstRequestForced"`
	ExternalProviderRequestForced    bool                       `json:"externalProviderRequestForced"`
	ExternalProviderCacheHitNoAPIKey bool                       `json:"externalProviderCacheHitWithoutApiKey"`
	ProviderRequestCountKnown        bool                       `json:"providerRequestCountKnown"`
	ProviderRequestCount             int                        `json:"providerRequestCount"`
	ProviderRequestPath              string                     `json:"providerRequestPath"`
	ProviderAuthHeaderOK             bool                       `json:"providerAuthHeaderOk"`
	ProviderRequestVerified          bool                       `json:"providerRequestVerified"`
	CacheHitAfterProviderDown        bool                       `json:"cacheHitAfterProviderDown"`
	TranslationFiles                 []string                   `json:"translationFiles"`
	TranslatedBlocks                 []translationBlockEvidence `json:"translatedBlocks"`
}

type translationBlockEvidence struct {
	BlockID          string `json:"blockId"`
	SourceLength     int    `json:"sourceLength"`
	TranslatedLength int    `json:"translatedLength"`
	HasTranslated    bool   `json:"hasTranslated"`
}

type providerRecorder struct {
	mu              sync.Mutex
	apiKey          string
	model           string
	targetLanguage  string
	count           int
	path            string
	authHeaderOK    bool
	requestVerified bool
	errs            []string
}

type providerSnapshot struct {
	count           int
	path            string
	authHeaderOK    bool
	requestVerified bool
	errs            []string
}

func main() {
	opts := runOptions{
		EvidenceDir:    filepath.Join("..", "release-out", "ocr-translation-smoke"),
		ProviderMode:   providerModeLocal,
		BaseURL:        strings.TrimSpace(os.Getenv(defaultTranslationBaseURLEnv)),
		Model:          envOrDefault(defaultTranslationModelEnv, smokeModel),
		APIKeyEnv:      defaultTranslationAPIKeyEnv,
		SourceLanguage: smokeSourceLanguage,
		TargetLanguage: smokeTargetLanguage,
		ForceProvider:  true,
	}
	flag.StringVar(&opts.DataRoot, "data-dir", "", "data root for the smoke run; defaults to a temp directory inside the evidence directory")
	flag.StringVar(&opts.EvidenceDir, "evidence-dir", opts.EvidenceDir, "directory for translation smoke evidence")
	flag.StringVar(&opts.ProviderMode, "provider-mode", opts.ProviderMode, "provider mode: local or external-openai-compatible")
	flag.StringVar(&opts.BaseURL, "base-url", opts.BaseURL, "OpenAI-compatible provider base URL; defaults to RF_OCR_TRANSLATION_BASE_URL in external mode")
	flag.StringVar(&opts.Model, "model", opts.Model, "OpenAI-compatible model; defaults to RF_OCR_TRANSLATION_MODEL or rf-translation-smoke")
	flag.StringVar(&opts.APIKeyEnv, "api-key-env", opts.APIKeyEnv, "environment variable that contains the provider API key")
	flag.StringVar(&opts.SourceLanguage, "source-language", opts.SourceLanguage, "source language passed to the translation provider")
	flag.StringVar(&opts.TargetLanguage, "target-language", opts.TargetLanguage, "target language passed to the translation provider")
	flag.BoolVar(&opts.ForceProvider, "force-provider", opts.ForceProvider, "force the first provider request so reused data roots cannot pass only from cache")
	flag.Parse()

	report, err := runSmoke(opts)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode OCR translation smoke report: %v\n", err)
		os.Exit(1)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func run(dataRoot string, evidenceDir string) (smokeReport, error) {
	return runSmoke(runOptions{
		DataRoot:       dataRoot,
		EvidenceDir:    evidenceDir,
		ProviderMode:   providerModeLocal,
		Model:          smokeModel,
		APIKeyEnv:      defaultTranslationAPIKeyEnv,
		SourceLanguage: smokeSourceLanguage,
		TargetLanguage: smokeTargetLanguage,
		ForceProvider:  true,
	})
}

func runSmoke(opts runOptions) (smokeReport, error) {
	opts = normalizeRunOptions(opts)
	if opts.EvidenceDir == "" {
		return smokeReport{}, errors.New("evidence dir is required")
	}
	evidenceDir, err := filepath.Abs(opts.EvidenceDir)
	if err != nil {
		return smokeReport{}, err
	}
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return smokeReport{}, err
	}
	dataRoot := opts.DataRoot
	if dataRoot == "" {
		dataRoot, err = os.MkdirTemp(evidenceDir, "data-root-*")
		if err != nil {
			return smokeReport{}, err
		}
	}

	appData := appdata.NewService(dataRoot)
	info, err := appData.Info()
	if err != nil {
		return smokeReport{}, fmt.Errorf("app data info: %w", err)
	}
	settingsService := settings.NewService(appData)
	secretStore := secretstore.NewStore(appData)
	ocrService := ocr.NewService(appData)

	input, recorder, cleanup, err := prepareProvider(opts)
	if err != nil {
		return smokeReport{}, err
	}
	if cleanup != nil {
		defer func() {
			if cleanup != nil {
				cleanup()
			}
		}()
	}

	if err := writeTranslationSettings(settingsService, secretStore, input); err != nil {
		return smokeReport{}, err
	}
	settingsPath, err := settingsService.Path()
	if err != nil {
		return smokeReport{}, err
	}
	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		return smokeReport{}, err
	}
	if strings.Contains(string(settingsData), input.APIKey) {
		return smokeReport{}, errors.New("settings.json contains raw OCR translation API key")
	}
	loadedKey, ok, err := secretStore.Load(ocrTranslationSecretName)
	if err != nil {
		return smokeReport{}, fmt.Errorf("load translation secret: %w", err)
	}
	if !ok || loadedKey != input.APIKey {
		return smokeReport{}, errors.New("translation secret store did not return the saved API key")
	}
	secretStatus, err := secretStore.Status()
	if err != nil {
		return smokeReport{}, fmt.Errorf("secret store status: %w", err)
	}

	result := smokeOCRResult()
	if err := ocrService.WriteResult(result); err != nil {
		return smokeReport{}, fmt.Errorf("write OCR result: %w", err)
	}
	currentSettings, err := settingsService.Load()
	if err != nil {
		return smokeReport{}, fmt.Errorf("load settings: %w", err)
	}
	req := ocr.TranslateRequest{
		OcrResultID:    result.ID,
		Provider:       currentSettings.OCR.Translation.Provider,
		BaseURL:        currentSettings.OCR.Translation.BaseURL,
		APIKey:         loadedKey,
		Model:          currentSettings.OCR.Translation.Model,
		SourceLanguage: currentSettings.OCR.Translation.SourceLanguage,
		TargetLanguage: currentSettings.OCR.Translation.TargetLanguage,
		Force:          opts.ForceProvider,
	}
	first, err := ocrService.Translate(req)
	if err != nil {
		return smokeReport{}, fmt.Errorf("translate first request: %w", err)
	}
	if err := validateTranslatedBlocks(opts.ProviderMode, first.Blocks); err != nil {
		return smokeReport{}, err
	}
	if recorder != nil && cleanup != nil {
		cleanup()
		cleanup = nil
	}
	cacheReq := req
	cacheReq.Force = false
	cacheReq.APIKey = ""
	second, err := ocrService.Translate(cacheReq)
	if err != nil {
		return smokeReport{}, fmt.Errorf("translate cache request without API key: %w", err)
	}
	if err := validateTranslatedBlocks(opts.ProviderMode, second.Blocks); err != nil {
		return smokeReport{}, err
	}

	var snapshot providerSnapshot
	providerRequestCountKnown := false
	cacheHitAfterProviderDown := false
	if recorder != nil {
		providerRequestCountKnown = true
		cacheHitAfterProviderDown = true
		snapshot = recorder.snapshot()
		if snapshot.count != 1 {
			return smokeReport{}, fmt.Errorf("provider request count after cache request = %d, want 1", snapshot.count)
		}
		if !snapshot.authHeaderOK {
			return smokeReport{}, errors.New("provider did not receive the API key through the Authorization header")
		}
		if !snapshot.requestVerified {
			return smokeReport{}, fmt.Errorf("provider request did not contain required OCR prompt data: %s", strings.Join(snapshot.errs, "; "))
		}
	}
	translationFiles, err := translationEvidenceFiles(info.RootDir)
	if err != nil {
		return smokeReport{}, err
	}
	if len(translationFiles) == 0 {
		return smokeReport{}, errors.New("translation cache file was not written")
	}

	report := smokeReport{
		SchemaVersion:                    2,
		OK:                               true,
		GeneratedAt:                      time.Now().UTC(),
		DataRoot:                         info.RootDir,
		EvidencePath:                     filepath.Join(evidenceDir, evidenceFileName),
		SettingsPath:                     settingsPath,
		SettingsJSONContainsRawKey:       false,
		SecretBackend:                    secretStatus.Backend,
		SecretStatus:                     secretStatus.Dir,
		ProviderMode:                     opts.ProviderMode,
		Provider:                         req.Provider,
		ProviderBaseURL:                  sanitizeBaseURLForReport(req.BaseURL),
		Model:                            req.Model,
		APIKeyEnv:                        opts.APIKeyEnv,
		APIKeyProvided:                   strings.TrimSpace(input.APIKey) != "",
		SourceLanguage:                   req.SourceLanguage,
		TargetLanguage:                   req.TargetLanguage,
		OcrResultID:                      result.ID,
		BlockCount:                       len(result.Blocks),
		FirstRequestForced:               opts.ForceProvider,
		ExternalProviderRequestForced:    opts.ProviderMode == providerModeExternalOpenAI && opts.ForceProvider,
		ExternalProviderCacheHitNoAPIKey: opts.ProviderMode == providerModeExternalOpenAI,
		ProviderRequestCountKnown:        providerRequestCountKnown,
		ProviderRequestCount:             snapshot.count,
		ProviderRequestPath:              snapshot.path,
		ProviderAuthHeaderOK:             snapshot.authHeaderOK,
		ProviderRequestVerified:          snapshot.requestVerified,
		CacheHitAfterProviderDown:        cacheHitAfterProviderDown,
		TranslationFiles:                 translationFiles,
		TranslatedBlocks:                 translationBlockEvidenceFrom(second.Blocks),
	}
	if err := writeEvidence(report, input.APIKey); err != nil {
		return smokeReport{}, err
	}
	return report, nil
}

func normalizeRunOptions(opts runOptions) runOptions {
	opts.DataRoot = strings.TrimSpace(opts.DataRoot)
	opts.EvidenceDir = strings.TrimSpace(opts.EvidenceDir)
	opts.ProviderMode = strings.TrimSpace(opts.ProviderMode)
	opts.BaseURL = strings.TrimSpace(opts.BaseURL)
	opts.Model = strings.TrimSpace(opts.Model)
	opts.APIKeyEnv = strings.TrimSpace(opts.APIKeyEnv)
	opts.SourceLanguage = strings.TrimSpace(opts.SourceLanguage)
	opts.TargetLanguage = strings.TrimSpace(opts.TargetLanguage)
	if opts.ProviderMode == "" {
		opts.ProviderMode = providerModeLocal
	}
	if opts.Model == "" {
		opts.Model = smokeModel
	}
	if opts.APIKeyEnv == "" {
		opts.APIKeyEnv = defaultTranslationAPIKeyEnv
	}
	if opts.SourceLanguage == "" {
		opts.SourceLanguage = smokeSourceLanguage
	}
	if opts.TargetLanguage == "" {
		opts.TargetLanguage = smokeTargetLanguage
	}
	return opts
}

func prepareProvider(opts runOptions) (translationSettingsInput, *providerRecorder, func(), error) {
	switch opts.ProviderMode {
	case providerModeLocal:
		recorder := newProviderRecorder(localSmokeAPIKey, opts.Model, opts.TargetLanguage)
		server := httptest.NewServer(recorder.handler())
		input := translationSettingsInput{
			BaseURL:        server.URL,
			APIKey:         localSmokeAPIKey,
			Model:          opts.Model,
			SourceLanguage: opts.SourceLanguage,
			TargetLanguage: opts.TargetLanguage,
		}
		return input, recorder, server.Close, nil
	case providerModeExternalOpenAI:
		if opts.BaseURL == "" {
			return translationSettingsInput{}, nil, nil, fmt.Errorf("external OpenAI-compatible base URL is required; set -base-url or %s", defaultTranslationBaseURLEnv)
		}
		apiKey := strings.TrimSpace(os.Getenv(opts.APIKeyEnv))
		if apiKey == "" {
			return translationSettingsInput{}, nil, nil, fmt.Errorf("external OpenAI-compatible API key is required in environment variable %s", opts.APIKeyEnv)
		}
		input := translationSettingsInput{
			BaseURL:        opts.BaseURL,
			APIKey:         apiKey,
			Model:          opts.Model,
			SourceLanguage: opts.SourceLanguage,
			TargetLanguage: opts.TargetLanguage,
		}
		return input, nil, nil, nil
	default:
		return translationSettingsInput{}, nil, nil, fmt.Errorf("unsupported provider mode %q; expected %q or %q", opts.ProviderMode, providerModeLocal, providerModeExternalOpenAI)
	}
}

func writeTranslationSettings(service *settings.Service, store *secretstore.Store, input translationSettingsInput) error {
	next := settings.Default()
	next.OCR.Translation.Provider = smokeProvider
	next.OCR.Translation.BaseURL = input.BaseURL
	next.OCR.Translation.APIKey = ""
	next.OCR.Translation.APIKeySet = true
	next.OCR.Translation.Model = input.Model
	next.OCR.Translation.SourceLanguage = input.SourceLanguage
	next.OCR.Translation.TargetLanguage = input.TargetLanguage
	next.OCR.Translation.PrivacyConfirmed = true
	next.OCR.Translation.PrivacyConfirmedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := service.Save(next); err != nil {
		return fmt.Errorf("save translation settings: %w", err)
	}
	if err := store.Save(ocrTranslationSecretName, input.APIKey); err != nil {
		return fmt.Errorf("save translation secret: %w", err)
	}
	return nil
}

func smokeOCRResult() ocr.Result {
	return ocr.Result{
		ID:          "ocr_translation_smoke_result",
		SourceKind:  ocr.SourceRegionScreenshot,
		SourceID:    "translation-smoke-shot",
		ImagePath:   "translation-smoke.png",
		ImageSHA256: "sha256-translation-smoke",
		ModelID:     "ppocrv5-mobile-zh-en",
		Language:    "zh-en",
		Width:       900,
		Height:      280,
		Blocks: []ocr.Block{
			{
				ID:         "b1",
				Text:       "RecordingFreedom",
				Confidence: 0.99,
				Box: []ocr.Point{
					{X: 30.13, Y: 32.08},
					{X: 669.97, Y: 32.08},
					{X: 669.97, Y: 109.86},
					{X: 30.13, Y: 109.86},
				},
			},
			{
				ID:         "b2",
				Text:       "文字识别",
				Confidence: 0.98,
				Box: []ocr.Point{
					{X: 41.34, Y: 137.08},
					{X: 330.30, Y: 137.08},
					{X: 330.30, Y: 234.30},
					{X: 41.34, Y: 234.30},
				},
				LineIndex: 1,
			},
		},
		PlainText:  "RecordingFreedom\n文字识别",
		CreatedAt:  time.Now().UTC(),
		DurationMS: 16,
	}
}

func newProviderRecorder(apiKey string, model string, targetLanguage string) *providerRecorder {
	return &providerRecorder{
		apiKey:          apiKey,
		model:           model,
		targetLanguage:  targetLanguage,
		authHeaderOK:    false,
		requestVerified: false,
		errs:            []string{},
	}
}

func (p *providerRecorder) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
		p.mu.Lock()
		p.count++
		p.path = r.URL.Path
		p.authHeaderOK = r.Header.Get("Authorization") == "Bearer "+p.apiKey
		p.requestVerified, p.errs = verifyProviderRequest(r, string(body), p.model, p.targetLanguage)
		p.mu.Unlock()
		response := map[string]any{
			"choices": []map[string]any{{
				"message": map[string]string{
					"role":    "assistant",
					"content": `[{"blockId":"b1","translated":"RecordingFreedom 已翻译"},{"blockId":"b2","translated":"文字识别 已翻译"}]`,
				},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}

func (p *providerRecorder) snapshot() providerSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	errs := make([]string, len(p.errs))
	copy(errs, p.errs)
	return providerSnapshot{
		count:           p.count,
		path:            p.path,
		authHeaderOK:    p.authHeaderOK,
		requestVerified: p.requestVerified,
		errs:            errs,
	}
}

func verifyProviderRequest(req *http.Request, body string, model string, targetLanguage string) (bool, []string) {
	errs := []string{}
	if req.Method != http.MethodPost {
		errs = append(errs, "method is not POST")
	}
	if req.URL.Path != "/chat/completions" {
		errs = append(errs, "path is not /chat/completions")
	}
	for _, needle := range []string{model, "RecordingFreedom", "文字识别", targetLanguage, "blockId", "b1", "b2"} {
		if !strings.Contains(body, needle) {
			errs = append(errs, "missing "+needle)
		}
	}
	return len(errs) == 0, errs
}

func validateTranslatedBlocks(providerMode string, blocks []ocr.TranslationBlock) error {
	if providerMode == providerModeLocal {
		if err := validateLocalTranslatedBlocks(blocks); err != nil {
			return err
		}
	}
	return validateTranslationCoverage(blocks)
}

func validateLocalTranslatedBlocks(blocks []ocr.TranslationBlock) error {
	if len(blocks) != 2 {
		return fmt.Errorf("translated blocks = %d, want 2", len(blocks))
	}
	want := map[string]string{
		"b1": "RecordingFreedom 已翻译",
		"b2": "文字识别 已翻译",
	}
	for _, block := range blocks {
		if want[block.BlockID] != block.Translated {
			return fmt.Errorf("translated block %q = %q, want %q", block.BlockID, block.Translated, want[block.BlockID])
		}
	}
	return nil
}

func validateTranslationCoverage(blocks []ocr.TranslationBlock) error {
	if len(blocks) != 2 {
		return fmt.Errorf("translated blocks = %d, want 2", len(blocks))
	}
	wantIDs := map[string]bool{"b1": false, "b2": false}
	for _, block := range blocks {
		if _, ok := wantIDs[block.BlockID]; !ok {
			return fmt.Errorf("translated block id %q is not in smoke OCR result", block.BlockID)
		}
		if strings.TrimSpace(block.Translated) == "" {
			return fmt.Errorf("translated block %q is empty", block.BlockID)
		}
		wantIDs[block.BlockID] = true
	}
	for id, seen := range wantIDs {
		if !seen {
			return fmt.Errorf("translated block %q is missing", id)
		}
	}
	return nil
}

func translationEvidenceFiles(root string) ([]string, error) {
	dir := filepath.Join(root, "data", "ocr", "translations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	return files, nil
}

func translationBlockEvidenceFrom(blocks []ocr.TranslationBlock) []translationBlockEvidence {
	out := make([]translationBlockEvidence, 0, len(blocks))
	for _, block := range blocks {
		translated := strings.TrimSpace(block.Translated)
		out = append(out, translationBlockEvidence{
			BlockID:          block.BlockID,
			SourceLength:     len([]rune(block.Source)),
			TranslatedLength: len([]rune(translated)),
			HasTranslated:    translated != "",
		})
	}
	return out
}

func sanitizeBaseURLForReport(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return raw
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func writeEvidence(report smokeReport, secret string) error {
	if strings.TrimSpace(report.EvidencePath) == "" {
		return errors.New("evidence path is required")
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(secret) != "" && strings.Contains(string(data), secret) {
		return errors.New("translation smoke evidence contains raw API key")
	}
	if err := os.MkdirAll(filepath.Dir(report.EvidencePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(report.EvidencePath, append(data, '\n'), 0o644)
}
