package ocr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultModelCatalogURL = "https://github.com/lemon-casino/RecordingFreedom/releases/latest/download/ocr-model-catalog.json"
	modelRegistryFileName  = "registry.json"
	maxModelCatalogBytes   = 2 << 20
)

type modelCatalogFile struct {
	SchemaVersion int             `json:"schemaVersion"`
	GeneratedAt   time.Time       `json:"generatedAt,omitempty"`
	Models        []ModelManifest `json:"models"`
}

func (s *Service) RefreshModelCatalog(ctx context.Context, catalogURL string) (Status, error) {
	models, err := s.downloadModelCatalog(ctx, catalogURL)
	if err != nil {
		return Status{}, err
	}
	if err := s.saveModelCatalog(models); err != nil {
		return Status{}, err
	}
	return s.Status()
}

func (s *Service) downloadModelCatalog(ctx context.Context, catalogURL string) ([]ModelManifest, error) {
	catalogURL = strings.TrimSpace(catalogURL)
	if catalogURL == "" {
		catalogURL = DefaultModelCatalogURL
	}
	if err := validateModelCatalogURL(catalogURL); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "RecordingFreedom-ocr-model-catalog/1")
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download OCR model catalog failed with HTTP %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxModelCatalogBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxModelCatalogBytes {
		return nil, fmt.Errorf("OCR model catalog exceeds %d bytes", maxModelCatalogBytes)
	}
	return parseModelCatalog(data)
}

func (s *Service) savedModelCatalogPath() (string, error) {
	root, err := s.modelRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, modelRegistryFileName), nil
}

func (s *Service) saveModelCatalog(models []ModelManifest) error {
	path, err := s.savedModelCatalogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(modelCatalogFile{
		SchemaVersion: 1,
		GeneratedAt:   s.now().UTC(),
		Models:        models,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func (s *Service) loadSavedModelCatalog() ([]ModelManifest, error) {
	path, err := s.savedModelCatalogPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseModelCatalog(data)
}

func parseModelCatalog(data []byte) ([]ModelManifest, error) {
	var catalog modelCatalogFile
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	if catalog.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported OCR model catalog schema %d", catalog.SchemaVersion)
	}
	if len(catalog.Models) == 0 {
		return nil, errors.New("OCR model catalog is empty")
	}
	defaults := defaultModelRegistry()
	known := make(map[string]ModelManifest, len(defaults))
	for _, model := range defaults {
		known[model.ID] = model
	}
	seen := map[string]bool{}
	result := make([]ModelManifest, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		model.ID = strings.TrimSpace(model.ID)
		if model.ID == "" {
			return nil, errors.New("OCR model catalog contains an empty model id")
		}
		base, ok := known[model.ID]
		if !ok {
			return nil, fmt.Errorf("OCR model catalog contains unsupported model %q", model.ID)
		}
		if seen[model.ID] {
			return nil, fmt.Errorf("OCR model catalog contains duplicate model %q", model.ID)
		}
		seen[model.ID] = true
		merged := mergeCatalogModel(base, model)
		if err := validateCatalogModel(merged); err != nil {
			return nil, err
		}
		result = append(result, merged)
	}
	return result, nil
}

func mergeCatalogModel(base ModelManifest, catalog ModelManifest) ModelManifest {
	if strings.TrimSpace(catalog.Name) == "" {
		catalog.Name = base.Name
	}
	if strings.TrimSpace(catalog.Channel) == "" {
		catalog.Channel = base.Channel
	}
	if strings.TrimSpace(catalog.Engine) == "" {
		catalog.Engine = base.Engine
	}
	if len(catalog.Language) == 0 {
		catalog.Language = append([]string(nil), base.Language...)
	}
	if strings.TrimSpace(catalog.Version) == "" {
		catalog.Version = base.Version
	}
	if catalog.TextlineOrientation == nil {
		catalog.TextlineOrientation = base.TextlineOrientation
	}
	if len(catalog.Files) == 0 {
		catalog.Files = append([]ModelFile(nil), base.Files...)
	}
	return catalog
}

func validateCatalogModel(model ModelManifest) error {
	if model.SchemaVersion != 1 {
		return fmt.Errorf("OCR model %q has unsupported schema %d", model.ID, model.SchemaVersion)
	}
	if strings.TrimSpace(model.Name) == "" || strings.TrimSpace(model.Channel) == "" || strings.TrimSpace(model.Engine) == "" {
		return fmt.Errorf("OCR model %q is missing name, channel, or engine", model.ID)
	}
	if !modelPackageDownloadAvailable(model.Package) {
		return fmt.Errorf("OCR model %q does not declare a verified package download", model.ID)
	}
	if err := validateModelCatalogURL(model.Package.URL); err != nil {
		return fmt.Errorf("OCR model %q package URL is invalid: %w", model.ID, err)
	}
	if err := ValidateTextlineOrientationMode(model); err != nil {
		return fmt.Errorf("OCR model %q catalog is invalid: %w", model.ID, err)
	}
	required := make(map[string]bool)
	for _, name := range RequiredModelFileNames(model) {
		required[name] = false
	}
	for _, file := range model.Files {
		if _, ok := required[file.Name]; ok {
			required[file.Name] = true
		}
	}
	for name, found := range required {
		if !found {
			return fmt.Errorf("OCR model %q catalog is missing required file %s", model.ID, name)
		}
	}
	return nil
}

func validateModelCatalogURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if parsed.Scheme == "https" && parsed.Host != "" {
		return nil
	}
	if parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("only https URLs are allowed, except loopback http test URLs")
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
