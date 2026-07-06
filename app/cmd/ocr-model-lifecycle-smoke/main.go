package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

const (
	evidenceFileName = "ocr-model-lifecycle-smoke.json"
	stableModelID    = "ppocrv5-mobile-zh-en"
	smokeLanguage    = "zh-en"
)

var expectedSmokeTexts = []string{"RecordingFreedom", "文字识别"}

type multiStringFlag []string

func (f *multiStringFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *multiStringFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value is required")
	}
	*f = append(*f, value)
	return nil
}

type lifecycleReport struct {
	SchemaVersion     int             `json:"schemaVersion"`
	OK                bool            `json:"ok"`
	GeneratedAt       time.Time       `json:"generatedAt"`
	GOOS              string          `json:"goos"`
	GOARCH            string          `json:"goarch"`
	DataRoot          string          `json:"dataRoot"`
	EvidencePath      string          `json:"evidencePath"`
	WorkerPath        string          `json:"workerPath"`
	RuntimeDir        string          `json:"runtimeDir"`
	StablePackage     string          `json:"stablePackage"`
	CandidatePackages []string        `json:"candidatePackages"`
	FinalActiveModel  string          `json:"finalActiveModel"`
	Steps             []lifecycleStep `json:"steps"`
}

type lifecycleStep struct {
	Name                 string   `json:"name"`
	ModelID              string   `json:"modelId,omitempty"`
	PackagePath          string   `json:"packagePath,omitempty"`
	ActiveModelID        string   `json:"activeModelId,omitempty"`
	Status               string   `json:"status,omitempty"`
	Installed            bool     `json:"installed,omitempty"`
	Verified             bool     `json:"verified,omitempty"`
	DownloadAvailable    bool     `json:"downloadAvailable,omitempty"`
	SmokeImage           string   `json:"smokeImage,omitempty"`
	SmokeAssetReady      bool     `json:"smokeAssetReady,omitempty"`
	ResultID             string   `json:"resultId,omitempty"`
	ResultImage          string   `json:"resultImage,omitempty"`
	PlainText            string   `json:"plainText,omitempty"`
	BlockCount           int      `json:"blockCount,omitempty"`
	ExpectedTextsPresent []string `json:"expectedTextsPresent,omitempty"`
	ForceWorker          bool     `json:"forceWorker,omitempty"`
}

func main() {
	var dataRoot string
	var evidenceDir string
	var workerPath string
	var runtimeDir string
	var stablePackage string
	var candidatePackages multiStringFlag

	flag.StringVar(&dataRoot, "data-dir", "", "data root for the smoke run; defaults to a temp directory inside the evidence directory")
	flag.StringVar(&evidenceDir, "evidence-dir", filepath.Join("..", "release-out", "ocr-model-lifecycle-smoke"), "directory for OCR model lifecycle evidence")
	flag.StringVar(&workerPath, "worker-path", "", "path to rf-ocr-worker executable")
	flag.StringVar(&runtimeDir, "runtime-dir", "", "path to ONNX Runtime directory")
	flag.StringVar(&stablePackage, "stable-package", "", "path to the stable OCR model package zip or extracted directory")
	flag.Var(&candidatePackages, "candidate-package", "path to a candidate OCR model package zip or extracted directory; may be repeated")
	flag.Parse()

	report, err := run(dataRoot, evidenceDir, workerPath, runtimeDir, stablePackage, []string(candidatePackages))
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
		fmt.Fprintf(os.Stderr, "encode OCR model lifecycle smoke report: %v\n", err)
		os.Exit(1)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func run(dataRoot string, evidenceDir string, workerPath string, runtimeDir string, stablePackage string, candidatePackages []string) (lifecycleReport, error) {
	evidenceDir, err := prepareEvidenceDir(evidenceDir)
	if err != nil {
		return lifecycleReport{}, err
	}
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		dataRoot, err = os.MkdirTemp(evidenceDir, "data-root-*")
		if err != nil {
			return lifecycleReport{}, err
		}
	}
	workerPath, err = requireFile(workerPath, "-worker-path")
	if err != nil {
		return lifecycleReport{}, err
	}
	runtimeDir, err = requireDir(runtimeDir, "-runtime-dir")
	if err != nil {
		return lifecycleReport{}, err
	}
	stablePackage, err = requirePackagePath(stablePackage, "-stable-package")
	if err != nil {
		return lifecycleReport{}, err
	}
	if len(candidatePackages) == 0 {
		return lifecycleReport{}, errors.New("at least one -candidate-package is required")
	}
	resolvedCandidates := make([]string, 0, len(candidatePackages))
	for _, candidate := range candidatePackages {
		resolved, err := requirePackagePath(candidate, "-candidate-package")
		if err != nil {
			return lifecycleReport{}, err
		}
		resolvedCandidates = append(resolvedCandidates, resolved)
	}

	data := appdata.NewService(dataRoot)
	info, err := data.Info()
	if err != nil {
		return lifecycleReport{}, fmt.Errorf("app data info: %w", err)
	}
	service := ocr.NewServiceWithOptions(data, ocr.ServiceOptions{
		WorkerPath: workerPath,
		RuntimeDir: runtimeDir,
	})

	report := lifecycleReport{
		SchemaVersion:     1,
		OK:                true,
		GeneratedAt:       time.Now().UTC(),
		GOOS:              runtime.GOOS,
		GOARCH:            runtime.GOARCH,
		DataRoot:          info.RootDir,
		EvidencePath:      filepath.Join(evidenceDir, evidenceFileName),
		WorkerPath:        workerPath,
		RuntimeDir:        runtimeDir,
		StablePackage:     stablePackage,
		CandidatePackages: resolvedCandidates,
	}

	stableInfo, err := service.InstallModelPackage(stablePackage)
	if err != nil {
		return lifecycleReport{}, fmt.Errorf("install stable package: %w", err)
	}
	if stableInfo.ID != stableModelID {
		return lifecycleReport{}, fmt.Errorf("stable package installed model %q, want %q", stableInfo.ID, stableModelID)
	}
	status, err := service.Status()
	if err != nil {
		return lifecycleReport{}, fmt.Errorf("status after stable install: %w", err)
	}
	if err := requireActiveReady(status, stableModelID); err != nil {
		return lifecycleReport{}, fmt.Errorf("stable status after install: %w", err)
	}
	report.Steps = append(report.Steps, stepFromModel("install-stable", stableInfo, stablePackage, status))

	stableResult, err := recognizeModelSmoke(service, stableModelID, "recognize-stable-initial")
	if err != nil {
		return lifecycleReport{}, err
	}
	report.Steps = append(report.Steps, stableResult)

	for _, candidatePackage := range resolvedCandidates {
		candidateInfo, err := service.InstallModelPackage(candidatePackage)
		if err != nil {
			return lifecycleReport{}, fmt.Errorf("install candidate package %s: %w", candidatePackage, err)
		}
		if candidateInfo.ID == stableModelID {
			return lifecycleReport{}, fmt.Errorf("candidate package %s installed stable model id", candidatePackage)
		}
		status, err = service.Status()
		if err != nil {
			return lifecycleReport{}, fmt.Errorf("status after candidate install: %w", err)
		}
		if status.ActiveModelID != stableModelID {
			return lifecycleReport{}, fmt.Errorf("candidate install auto-activated %q; want active model to remain %q", status.ActiveModelID, stableModelID)
		}
		report.Steps = append(report.Steps, stepFromModel("install-candidate-no-auto-activate", candidateInfo, candidatePackage, status))

		status, err = service.SetActiveModel(candidateInfo.ID)
		if err != nil {
			return lifecycleReport{}, fmt.Errorf("set active candidate %s: %w", candidateInfo.ID, err)
		}
		if err := requireActiveReady(status, candidateInfo.ID); err != nil {
			return lifecycleReport{}, fmt.Errorf("candidate status after switch: %w", err)
		}
		report.Steps = append(report.Steps, stepFromStatus("activate-candidate", candidateInfo.ID, status))

		candidateResult, err := recognizeModelSmoke(service, candidateInfo.ID, "recognize-candidate")
		if err != nil {
			return lifecycleReport{}, err
		}
		report.Steps = append(report.Steps, candidateResult)

		status, err = service.RemoveModel(candidateInfo.ID)
		if err != nil {
			return lifecycleReport{}, fmt.Errorf("remove active candidate %s: %w", candidateInfo.ID, err)
		}
		if err := requireActiveReady(status, stableModelID); err != nil {
			return lifecycleReport{}, fmt.Errorf("stable fallback after removing %s: %w", candidateInfo.ID, err)
		}
		if removed := findModel(status.Models, candidateInfo.ID); removed != nil && (removed.Installed || removed.Verified || removed.Active) {
			return lifecycleReport{}, fmt.Errorf("removed candidate state = %#v, want not installed, not verified, not active", *removed)
		}
		report.Steps = append(report.Steps, stepFromStatus("remove-candidate-fallback-stable", candidateInfo.ID, status))

		fallbackResult, err := recognizeModelSmoke(service, stableModelID, "recognize-stable-after-candidate-remove")
		if err != nil {
			return lifecycleReport{}, err
		}
		report.Steps = append(report.Steps, fallbackResult)
	}

	finalStatus, err := service.Status()
	if err != nil {
		return lifecycleReport{}, fmt.Errorf("final status: %w", err)
	}
	if err := requireActiveReady(finalStatus, stableModelID); err != nil {
		return lifecycleReport{}, fmt.Errorf("final stable status: %w", err)
	}
	report.FinalActiveModel = finalStatus.ActiveModelID
	if err := writeEvidence(report); err != nil {
		return lifecycleReport{}, err
	}
	return report, nil
}

func prepareEvidenceDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("evidence dir is required")
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return "", err
	}
	return resolved, nil
}

func requireFile(path string, flagName string) (string, error) {
	resolved, err := requirePath(path, flagName)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s must be a file: %s", flagName, resolved)
	}
	return resolved, nil
}

func requireDir(path string, flagName string) (string, error) {
	resolved, err := requirePath(path, flagName)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s must be a directory: %s", flagName, resolved)
	}
	return resolved, nil
}

func requirePackagePath(path string, flagName string) (string, error) {
	resolved, err := requirePath(path, flagName)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return resolved, nil
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".zip") {
		return "", fmt.Errorf("%s must be a .zip file or directory: %s", flagName, resolved)
	}
	return resolved, nil
}

func requirePath(path string, flagName string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is required", flagName)
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func requireActiveReady(status ocr.Status, modelID string) error {
	if status.ActiveModelID != modelID {
		return fmt.Errorf("active model = %q, want %q", status.ActiveModelID, modelID)
	}
	if status.Status != ocr.StatusReady {
		return fmt.Errorf("status = %q, want %q: %s", status.Status, ocr.StatusReady, status.Message)
	}
	model := findModel(status.Models, modelID)
	if model == nil {
		return fmt.Errorf("model %q is missing from status", modelID)
	}
	if !model.Active || !model.Installed || !model.Verified {
		return fmt.Errorf("model %q active/installed/verified = %t/%t/%t, want true/true/true", modelID, model.Active, model.Installed, model.Verified)
	}
	return nil
}

func recognizeModelSmoke(service *ocr.Service, modelID string, stepName string) (lifecycleStep, error) {
	status, err := service.Status()
	if err != nil {
		return lifecycleStep{}, fmt.Errorf("%s status: %w", stepName, err)
	}
	if err := requireActiveReady(status, modelID); err != nil {
		return lifecycleStep{}, fmt.Errorf("%s requires active model %s: %w", stepName, modelID, err)
	}
	model := findModel(status.Models, modelID)
	if model == nil {
		return lifecycleStep{}, fmt.Errorf("%s model %q not found", stepName, modelID)
	}
	smokeImage := strings.TrimSpace(model.SmokeImage)
	if smokeImage == "" {
		smokeImage = "smoke.png"
	}
	imagePath := filepath.Join(model.ModelDir, smokeImage)
	result, err := service.RecognizeImage(ocr.RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: ocr.SourceImage,
		SourceID:   stepName + "-" + modelID,
		Language:   smokeLanguage,
		ModelID:    modelID,
		Force:      true,
		Priority:   ocr.JobPriorityInteractive,
	})
	if err != nil {
		return lifecycleStep{}, fmt.Errorf("%s recognize %s: %w", stepName, modelID, err)
	}
	present := presentExpectedTexts(result.PlainText)
	if len(present) != len(expectedSmokeTexts) {
		return lifecycleStep{}, fmt.Errorf("%s plain text %q is missing expected texts %v", stepName, result.PlainText, missingExpectedTexts(present))
	}
	if len(result.Blocks) == 0 {
		return lifecycleStep{}, fmt.Errorf("%s returned no OCR blocks", stepName)
	}
	step := stepFromStatus(stepName, modelID, status)
	step.SmokeImage = imagePath
	step.ResultID = result.ID
	step.ResultImage = result.ImagePath
	step.PlainText = result.PlainText
	step.BlockCount = len(result.Blocks)
	step.ExpectedTextsPresent = present
	step.ForceWorker = true
	return step, nil
}

func stepFromModel(name string, model ocr.ModelInfo, packagePath string, status ocr.Status) lifecycleStep {
	step := stepFromStatus(name, model.ID, status)
	step.PackagePath = packagePath
	step.Installed = model.Installed
	step.Verified = model.Verified
	step.DownloadAvailable = model.DownloadAvailable
	step.SmokeAssetReady = model.SmokeAssetReady
	if strings.TrimSpace(model.SmokeImage) != "" {
		step.SmokeImage = filepath.Join(model.ModelDir, model.SmokeImage)
	}
	return step
}

func stepFromStatus(name string, modelID string, status ocr.Status) lifecycleStep {
	step := lifecycleStep{
		Name:          name,
		ModelID:       modelID,
		ActiveModelID: status.ActiveModelID,
		Status:        status.Status,
	}
	if model := findModel(status.Models, modelID); model != nil {
		step.Installed = model.Installed
		step.Verified = model.Verified
		step.DownloadAvailable = model.DownloadAvailable
		step.SmokeAssetReady = model.SmokeAssetReady
		if strings.TrimSpace(model.SmokeImage) != "" {
			step.SmokeImage = filepath.Join(model.ModelDir, model.SmokeImage)
		}
	}
	return step
}

func findModel(models []ocr.ModelInfo, modelID string) *ocr.ModelInfo {
	for index := range models {
		if models[index].ID == modelID {
			return &models[index]
		}
	}
	return nil
}

func presentExpectedTexts(text string) []string {
	present := make([]string, 0, len(expectedSmokeTexts))
	for _, expected := range expectedSmokeTexts {
		if strings.Contains(text, expected) {
			present = append(present, expected)
		}
	}
	return present
}

func missingExpectedTexts(present []string) []string {
	seen := map[string]bool{}
	for _, text := range present {
		seen[text] = true
	}
	missing := make([]string, 0)
	for _, expected := range expectedSmokeTexts {
		if !seen[expected] {
			missing = append(missing, expected)
		}
	}
	return missing
}

func writeEvidence(report lifecycleReport) error {
	if strings.TrimSpace(report.EvidencePath) == "" {
		return errors.New("evidence path is required")
	}
	if err := os.MkdirAll(filepath.Dir(report.EvidencePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(report.EvidencePath, append(data, '\n'), 0o644)
}
