package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
)

const (
	stateSchemaVersion = 1
	modelRootDir       = "models"
	ocrModelDir        = "ocr"
	stateFileName      = "state.json"
	cacheRootDir       = "ocr"
	cacheDirName       = "cache"
	resultsDirName     = "results"
	translationsDir    = "translations"
	defaultLanguage    = "zh-en"
)

type Service struct {
	appData *appdata.Service
	now     func() time.Time

	workerPathOverride     string
	runtimeDirOverride     string
	workerArgs             []string
	workerCapabilitiesArgs []string
	workerEnv              []string
	workerTimeout          time.Duration
	modelRegistryOverride  []ModelManifest

	jobMu      sync.Mutex
	jobStarted bool
	jobNotify  chan struct{}
	jobEvents  chan JobEvent
	jobQueue   []*jobState
	activeJob  *jobState
	jobsByID   map[string]*jobState
	jobsByKey  map[string]*jobState

	modelDownloadMu      sync.Mutex
	modelDownloads       map[string]*modelDownloadState
	modelDownloadCancels map[string]context.CancelFunc
	modelDownloadEvents  chan ModelDownloadEvent
}

type ServiceOptions struct {
	WorkerPath             string
	RuntimeDir             string
	WorkerArgs             []string
	WorkerCapabilitiesArgs []string
	WorkerEnv              []string
	WorkerTimeout          time.Duration
	ModelRegistry          []ModelManifest
}

func NewService(appData *appdata.Service) *Service {
	return NewServiceWithOptions(appData, ServiceOptions{})
}

func NewServiceWithOptions(appData *appdata.Service, options ServiceOptions) *Service {
	service := &Service{
		appData:              appData,
		now:                  time.Now,
		jobNotify:            make(chan struct{}, 1),
		jobEvents:            make(chan JobEvent, 128),
		jobsByID:             map[string]*jobState{},
		jobsByKey:            map[string]*jobState{},
		modelDownloads:       map[string]*modelDownloadState{},
		modelDownloadCancels: map[string]context.CancelFunc{},
		modelDownloadEvents:  make(chan ModelDownloadEvent, 128),
	}
	service.ApplyOptions(options)
	return service
}

func (s *Service) ApplyOptions(options ServiceOptions) {
	if s == nil {
		return
	}
	s.workerPathOverride = strings.TrimSpace(options.WorkerPath)
	s.runtimeDirOverride = strings.TrimSpace(options.RuntimeDir)
	s.workerArgs = append([]string(nil), options.WorkerArgs...)
	s.workerCapabilitiesArgs = append([]string(nil), options.WorkerCapabilitiesArgs...)
	s.workerEnv = append([]string(nil), options.WorkerEnv...)
	s.workerTimeout = options.WorkerTimeout
	s.modelRegistryOverride = append([]ModelManifest(nil), options.ModelRegistry...)
}

func (s *Service) ListModels() ([]ModelInfo, error) {
	state, err := s.LoadState()
	if err != nil {
		return nil, err
	}
	models := s.modelRegistry()
	result := make([]ModelInfo, 0, len(models))
	for _, model := range models {
		info, err := s.modelInfo(model, state.ActiveModelID)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return modelSortKey(result[i]) < modelSortKey(result[j])
	})
	return result, nil
}

func (s *Service) Status() (Status, error) {
	models, err := s.ListModels()
	if err != nil {
		return Status{}, err
	}
	state, err := s.LoadState()
	if err != nil {
		return Status{}, err
	}
	status := Status{
		Status:        StatusNoModel,
		ActiveModelID: state.ActiveModelID,
		Models:        models,
		WorkerPath:    s.workerPath(),
		RuntimeDir:    s.runtimeDir(),
		Message:       "No verified OCR model is installed.",
	}
	for _, model := range models {
		if model.Active && model.Installed && model.Verified {
			status.Status = StatusWorkerAbsent
			status.Message = "A verified OCR model is installed, but the OCR worker has not been connected yet."
			if !fileExists(status.WorkerPath) {
				return status, nil
			}
			capabilities, err := s.queryWorkerCapabilities()
			if err != nil {
				status.Status = StatusWorkerUnavailable
				status.Message = "OCR worker exists but failed capability probe: " + err.Error()
				return status, nil
			}
			status.WorkerCapabilities = &capabilities
			if !capabilities.SupportsRecognize {
				status.Status = StatusWorkerUnavailable
				status.Message = strings.TrimSpace(capabilities.Message)
				if status.Message == "" {
					status.Message = "OCR worker is present but does not support image recognition."
				}
				return status, nil
			}
			status.Status = StatusReady
			status.Message = "OCR model and worker are available."
			return status, nil
		}
	}
	for _, model := range models {
		if model.Active && model.Installed && !model.Verified {
			status.Status = StatusModelInvalid
			status.Message = model.VerificationError
			return status, nil
		}
	}
	return status, nil
}

func (s *Service) SetActiveModel(modelID string) (Status, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return Status{}, errors.New("OCR model id is required")
	}
	model, ok := s.findModel(modelID)
	if !ok {
		return Status{}, fmt.Errorf("unknown OCR model %q", modelID)
	}
	info, err := s.modelInfo(model, modelID)
	if err != nil {
		return Status{}, err
	}
	if !info.Installed || !info.Verified {
		return Status{}, fmt.Errorf("OCR model %q is not installed or failed verification", modelID)
	}
	state := State{SchemaVersion: stateSchemaVersion, ActiveModelID: modelID, UpdatedAt: s.now().UTC()}
	if err := s.saveState(state); err != nil {
		return Status{}, err
	}
	return s.Status()
}

func (s *Service) InstallModel(modelID string) (ModelInfo, error) {
	modelID = strings.TrimSpace(modelID)
	model, ok := s.findModel(modelID)
	if !ok {
		return ModelInfo{}, fmt.Errorf("unknown OCR model %q", modelID)
	}
	info, err := s.modelInfo(model, modelID)
	if err != nil {
		return ModelInfo{}, err
	}
	if !info.Installed {
		return info, fmt.Errorf("OCR model %q is not present under %s", modelID, info.ModelDir)
	}
	if !info.Verified {
		return info, fmt.Errorf("OCR model %q failed verification: %s", modelID, info.VerificationError)
	}
	return info, nil
}

func (s *Service) RemoveModel(modelID string) (Status, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return Status{}, errors.New("OCR model id is required")
	}
	_, ok := s.findModel(modelID)
	if !ok {
		return Status{}, fmt.Errorf("unknown OCR model %q", modelID)
	}
	dir, err := s.modelDir(modelID)
	if err != nil {
		return Status{}, err
	}
	if err := os.RemoveAll(dir); err != nil {
		return Status{}, err
	}
	state, err := s.LoadState()
	if err != nil {
		return Status{}, err
	}
	if state.ActiveModelID == modelID {
		state.ActiveModelID = defaultActiveModelID()
		state.UpdatedAt = s.now().UTC()
		if err := s.saveState(state); err != nil {
			return Status{}, err
		}
	}
	return s.Status()
}

func (s *Service) LoadState() (State, error) {
	path, err := s.statePath()
	if err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{SchemaVersion: stateSchemaVersion, ActiveModelID: defaultActiveModelID()}, nil
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	if state.SchemaVersion != stateSchemaVersion {
		state.SchemaVersion = stateSchemaVersion
	}
	if strings.TrimSpace(state.ActiveModelID) == "" {
		state.ActiveModelID = defaultActiveModelID()
	}
	return state, nil
}

func (s *Service) saveState(state State) error {
	path, err := s.statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func (s *Service) modelInfo(manifest ModelManifest, activeModelID string) (ModelInfo, error) {
	dir, err := s.modelDir(manifest.ID)
	if err != nil {
		return ModelInfo{}, err
	}
	display := manifest
	if installedManifest, err := readModelManifest(filepath.Join(dir, "manifest.json")); err == nil && installedManifest.ID == manifest.ID {
		display = installedManifest
	}
	info := ModelInfo{
		ID:                display.ID,
		Name:              display.Name,
		Channel:           display.Channel,
		Engine:            display.Engine,
		Language:          append([]string(nil), display.Language...),
		Version:           display.Version,
		SourceURL:         display.Source.URL,
		License:           display.Source.License,
		DownloadAvailable: strings.TrimSpace(display.Package.URL) != "" && display.Package.Bytes > 0 && len(strings.TrimSpace(display.Package.SHA256)) == 64,
		DownloadBytes:     display.Package.Bytes,
		Active:            manifest.ID == activeModelID,
		ModelDir:          dir,
	}
	if modelSmokeDeclared(display.Smoke) {
		info.SmokeImage = strings.TrimSpace(display.Smoke.Image)
		info.SmokeExpected = modelSmokeExpectedPath(display.Smoke)
	}
	installed, missing, verificationErr := verifyModelDir(dir, manifest)
	info.Installed = installed
	info.Verified = installed && verificationErr == ""
	info.MissingFiles = missing
	info.VerificationError = verificationErr
	if installed && modelSmokeDeclared(display.Smoke) {
		if smokeErr := verifyModelSmokeAssets(dir, display); smokeErr != "" {
			info.SmokeError = smokeErr
		} else {
			info.SmokeAssetReady = true
		}
	}
	return info, nil
}

func verifyModelDir(dir string, fallback ModelManifest) (bool, []string, string) {
	manifestPath := filepath.Join(dir, "manifest.json")
	if !fileExists(manifestPath) {
		missing := []string{"manifest.json"}
		for _, name := range RequiredModelFileNames(fallback) {
			if !fileExists(filepath.Join(dir, name)) {
				missing = append(missing, name)
			}
		}
		sort.Strings(missing)
		return false, missing, ""
	}
	manifest, err := readModelManifest(manifestPath)
	if err != nil {
		return true, nil, err.Error()
	}
	if manifest.ID != fallback.ID {
		return true, nil, fmt.Sprintf("manifest id %q does not match expected model %q", manifest.ID, fallback.ID)
	}
	if err := ValidateTextlineOrientationMode(manifest); err != nil {
		return true, nil, err.Error()
	}
	missing := make([]string, 0)
	for _, name := range RequiredModelFileNames(manifest) {
		if !modelManifestDeclaresFile(manifest, name) {
			return true, nil, fmt.Sprintf("manifest is missing required file %q", name)
		}
		if !fileExists(filepath.Join(dir, name)) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return false, missing, ""
	}
	for _, file := range manifest.Files {
		if strings.TrimSpace(file.Name) == "" {
			return true, nil, "manifest contains an empty file name"
		}
		path := filepath.Join(dir, file.Name)
		if !fileExists(path) {
			return true, nil, fmt.Sprintf("manifest file %q is missing", file.Name)
		}
		if file.Bytes > 0 {
			info, err := os.Stat(path)
			if err != nil {
				return true, nil, err.Error()
			}
			if info.Size() != file.Bytes {
				return true, nil, fmt.Sprintf("manifest file %q size = %d, want %d", file.Name, info.Size(), file.Bytes)
			}
		}
		if file.SHA256 != "" {
			sum, err := fileSHA256(path)
			if err != nil {
				return true, nil, err.Error()
			}
			if !strings.EqualFold(sum, file.SHA256) {
				return true, nil, fmt.Sprintf("manifest file %q sha256 = %s, want %s", file.Name, sum, file.SHA256)
			}
		}
	}
	if smokeErr := verifyModelSmokeAssets(dir, manifest); smokeErr != "" {
		return true, nil, smokeErr
	}
	return true, nil, ""
}

func modelManifestDeclaresFile(manifest ModelManifest, name string) bool {
	for _, file := range manifest.Files {
		if file.Name == name {
			return true
		}
	}
	return false
}

func RequiredModelFileNames(manifest ModelManifest) []string {
	required := []string{"det.onnx", "rec.onnx", "keys.txt"}
	if ModelTextlineOrientationMode(manifest) == TextlineOrientationCLS {
		required = append(required, "cls.onnx")
	}
	sort.Strings(required)
	return required
}

func ModelTextlineOrientationMode(manifest ModelManifest) string {
	if manifest.TextlineOrientation == nil {
		return TextlineOrientationCLS
	}
	mode := strings.TrimSpace(manifest.TextlineOrientation.Mode)
	if mode == "" {
		return TextlineOrientationCLS
	}
	return mode
}

func ValidateTextlineOrientationMode(manifest ModelManifest) error {
	switch ModelTextlineOrientationMode(manifest) {
	case TextlineOrientationCLS, TextlineOrientationNone:
		return nil
	default:
		mode := ""
		if manifest.TextlineOrientation != nil {
			mode = manifest.TextlineOrientation.Mode
		}
		return fmt.Errorf("unsupported textlineOrientation.mode %q", mode)
	}
}

func verifyModelSmokeAssets(dir string, manifest ModelManifest) string {
	smoke := manifest.Smoke
	if !modelSmokeDeclared(smoke) {
		return ""
	}
	imageName := strings.TrimSpace(smoke.Image)
	if imageName == "" {
		return "model smoke.image is required when smoke validation is declared"
	}
	if !safeModelPackageRelativePath(imageName) {
		return fmt.Sprintf("model smoke image path %q is unsafe", smoke.Image)
	}
	imagePath := filepath.Join(dir, imageName)
	if !fileExists(imagePath) {
		return fmt.Sprintf("model smoke image %q is missing", imageName)
	}
	if _, _, err := ImageDimensions(imagePath); err != nil {
		return fmt.Sprintf("model smoke image %q is not a supported image: %v", imageName, err)
	}

	expectedName := modelSmokeExpectedPath(smoke)
	if strings.TrimSpace(expectedName) == "" {
		return "model smoke expected JSON is required"
	}
	if !safeModelPackageRelativePath(expectedName) {
		return fmt.Sprintf("model smoke expected path %q is unsafe", expectedName)
	}
	expectedPath := filepath.Join(dir, expectedName)
	if !fileExists(expectedPath) {
		return fmt.Sprintf("model smoke expected JSON %q is missing", expectedName)
	}
	if err := validateSmokeExpectedJSON(expectedPath); err != nil {
		return fmt.Sprintf("model smoke expected JSON %q is invalid: %v", expectedName, err)
	}
	return ""
}

func modelSmokeDeclared(smoke ModelSmoke) bool {
	return strings.TrimSpace(smoke.Image) != "" ||
		strings.TrimSpace(smoke.Expected) != "" ||
		len(smoke.MustContain) > 0 ||
		smoke.MaxDurationMS > 0
}

func modelSmokeExpectedPath(smoke ModelSmoke) string {
	expected := strings.TrimSpace(smoke.Expected)
	if expected != "" {
		return expected
	}
	if modelSmokeDeclared(smoke) {
		return "smoke.expected.json"
	}
	return ""
}

func validateSmokeExpectedJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value == nil {
		return errors.New("expected JSON must not be null")
	}
	return nil
}

func readModelManifest(path string) (ModelManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ModelManifest{}, err
	}
	var manifest ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ModelManifest{}, err
	}
	if manifest.SchemaVersion != 1 {
		return ModelManifest{}, fmt.Errorf("unsupported OCR model manifest schema %d", manifest.SchemaVersion)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return ModelManifest{}, errors.New("OCR model manifest id is required")
	}
	return manifest, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *Service) statePath() (string, error) {
	dir, err := s.ocrDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

func (s *Service) modelDir(modelID string) (string, error) {
	root, err := s.rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID), nil
}

func (s *Service) ocrDataDir() (string, error) {
	root, err := s.rootDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "data", cacheRootDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *Service) rootDir() (string, error) {
	if s == nil || s.appData == nil {
		return "", errors.New("OCR app data service is not initialized")
	}
	return s.appData.RootDir()
}

func (s *Service) workerPath() string {
	if s != nil && strings.TrimSpace(s.workerPathOverride) != "" {
		return s.workerPathOverride
	}
	root, err := s.rootDir()
	if err != nil {
		return ""
	}
	name := "rf-ocr-worker"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(root, "tools", "ocr-worker", runtime.GOOS+"-"+runtime.GOARCH, name)
}

func (s *Service) runtimeDir() string {
	if s != nil && strings.TrimSpace(s.runtimeDirOverride) != "" {
		return s.runtimeDirOverride
	}
	root, err := s.rootDir()
	if err != nil {
		return ""
	}
	return filepath.Join(root, "tools", "onnxruntime", runtime.GOOS+"-"+runtime.GOARCH)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func modelSortKey(model ModelInfo) string {
	channel := "2"
	switch model.Channel {
	case "stable":
		channel = "0"
	case "latest":
		channel = "1"
	case "quality":
		channel = "3"
	}
	return channel + ":" + model.ID
}

func findDefaultModel(modelID string) (ModelManifest, bool) {
	for _, model := range defaultModelRegistry() {
		if model.ID == modelID {
			return model, true
		}
	}
	return ModelManifest{}, false
}

func (s *Service) findModel(modelID string) (ModelManifest, bool) {
	for _, model := range s.modelRegistry() {
		if model.ID == modelID {
			return model, true
		}
	}
	return ModelManifest{}, false
}

func (s *Service) modelRegistry() []ModelManifest {
	if s != nil && len(s.modelRegistryOverride) > 0 {
		return append([]ModelManifest(nil), s.modelRegistryOverride...)
	}
	models := defaultModelRegistry()
	if s != nil {
		if catalogModels, err := s.loadSavedModelCatalog(); err == nil && len(catalogModels) > 0 {
			byID := make(map[string]ModelManifest, len(catalogModels))
			for _, model := range catalogModels {
				byID[model.ID] = model
			}
			for index, model := range models {
				if catalogModel, ok := byID[model.ID]; ok {
					models[index] = catalogModel
				}
			}
		}
	}
	return models
}

func defaultActiveModelID() string {
	return "ppocrv5-mobile-zh-en"
}

func defaultModelRegistry() []ModelManifest {
	files := []ModelFile{
		{Name: "det.onnx"},
		{Name: "cls.onnx"},
		{Name: "rec.onnx"},
		{Name: "keys.txt"},
	}
	return []ModelManifest{
		{
			SchemaVersion: 1,
			ID:            "ppocrv5-mobile-zh-en",
			Name:          "PP-OCRv5 Mobile Chinese/English",
			Channel:       "stable",
			Engine:        "onnxruntime",
			Language:      []string{"zh", "en"},
			Version:       "stable",
			Files:         files,
		},
		{
			SchemaVersion: 1,
			ID:            "ppocrv6-mobile-zh-en",
			Name:          "PP-OCRv6 Mobile Chinese/English",
			Channel:       "latest",
			Engine:        "onnxruntime",
			Language:      []string{"zh", "en"},
			Version:       "latest",
			Files:         files,
		},
		{
			SchemaVersion: 1,
			ID:            "ppocrv6-medium-zh-en",
			Name:          "PP-OCRv6 Medium Chinese/English",
			Channel:       "quality",
			Engine:        "onnxruntime",
			Language:      []string{"zh", "en"},
			Version:       "latest",
			Files:         files,
		},
	}
}
