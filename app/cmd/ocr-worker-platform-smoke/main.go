package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

const (
	evidenceFileName = "ocr-worker-platform-smoke.json"
	defaultModelID   = "ppocrv5-mobile-zh-en"
)

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

type smokeReport struct {
	SchemaVersion     int                    `json:"schemaVersion"`
	OK                bool                   `json:"ok"`
	GeneratedAt       time.Time              `json:"generatedAt"`
	GOOS              string                 `json:"goos"`
	GOARCH            string                 `json:"goarch"`
	WorkerPath        string                 `json:"workerPath"`
	RuntimeDir        string                 `json:"runtimeDir"`
	ModelPackage      string                 `json:"modelPackage"`
	ModelID           string                 `json:"modelId"`
	ModelDir          string                 `json:"modelDir"`
	EvidencePath      string                 `json:"evidencePath"`
	Capabilities      ocr.WorkerCapabilities `json:"capabilities"`
	Smoke             workerSmokeResult      `json:"smoke"`
	ExpectedTexts     []string               `json:"expectedTexts"`
	ExpectedTextsSeen []string               `json:"expectedTextsSeen"`
}

type workerSmokeResult struct {
	OK             bool     `json:"ok"`
	RuntimeDir     string   `json:"runtimeDir,omitempty"`
	ModelDir       string   `json:"modelDir,omitempty"`
	ImagePath      string   `json:"imagePath,omitempty"`
	MustContain    []string `json:"mustContain,omitempty"`
	PlainText      string   `json:"plainText,omitempty"`
	Blocks         int      `json:"blocks"`
	CandidateCount int      `json:"candidateCount"`
	Error          string   `json:"error,omitempty"`
}

func main() {
	var workerPath string
	var runtimeDir string
	var modelPackage string
	var modelID string
	var evidenceDir string
	var expectedTexts multiStringFlag

	flag.StringVar(&workerPath, "worker-path", "", "path to rf-ocr-worker executable")
	flag.StringVar(&runtimeDir, "runtime-dir", "", "path to ONNX Runtime directory")
	flag.StringVar(&modelPackage, "model-package", "", "path to stable OCR model package zip or extracted directory")
	flag.StringVar(&modelID, "model-id", defaultModelID, "model id inside the package")
	flag.StringVar(&evidenceDir, "evidence-dir", filepath.Join("..", "release-out", "ocr-worker-platform-smoke"), "directory for platform worker smoke evidence")
	flag.Var(&expectedTexts, "must-contain", "required text in OCR smoke output; may be repeated")
	flag.Parse()

	report, err := run(workerPath, runtimeDir, modelPackage, modelID, evidenceDir, []string(expectedTexts))
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
		fmt.Fprintf(os.Stderr, "encode OCR worker platform smoke report: %v\n", err)
		os.Exit(1)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func run(workerPath string, runtimeDir string, modelPackage string, modelID string, evidenceDir string, expectedTexts []string) (smokeReport, error) {
	workerPath, err := requireFile(workerPath, "-worker-path")
	if err != nil {
		return smokeReport{}, err
	}
	runtimeDir, err = requireDir(runtimeDir, "-runtime-dir")
	if err != nil {
		return smokeReport{}, err
	}
	modelPackage, err = requirePackagePath(modelPackage, "-model-package")
	if err != nil {
		return smokeReport{}, err
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return smokeReport{}, errors.New("-model-id is required")
	}
	if len(expectedTexts) == 0 {
		expectedTexts = []string{"RecordingFreedom", "文字识别"}
	}
	evidenceDir, err = prepareEvidenceDir(evidenceDir)
	if err != nil {
		return smokeReport{}, err
	}

	extractDir, err := os.MkdirTemp(evidenceDir, "model-*")
	if err != nil {
		return smokeReport{}, err
	}
	modelDir, err := prepareModelDir(modelPackage, extractDir, modelID)
	if err != nil {
		return smokeReport{}, err
	}

	capabilities, err := runCapabilities(workerPath, runtimeDir)
	if err != nil {
		return smokeReport{}, err
	}
	if !capabilities.RuntimeAvailable {
		return smokeReport{}, fmt.Errorf("OCR worker runtime is unavailable: %s", capabilities.RuntimeError)
	}
	if !capabilities.SupportsRecognize {
		return smokeReport{}, fmt.Errorf("OCR worker does not support recognition: %s", capabilities.Message)
	}

	smoke, err := runSmoke(workerPath, runtimeDir, modelDir, expectedTexts)
	if err != nil {
		return smokeReport{}, err
	}
	if !smoke.OK {
		return smokeReport{}, fmt.Errorf("OCR worker smoke returned ok=false: %s", smoke.Error)
	}
	seen := presentExpectedTexts(smoke.PlainText, expectedTexts)
	if len(seen) != len(expectedTexts) {
		return smokeReport{}, fmt.Errorf("OCR worker smoke plain text %q is missing expected texts %v", smoke.PlainText, missingExpectedTexts(expectedTexts, seen))
	}
	if smoke.Blocks <= 0 {
		return smokeReport{}, errors.New("OCR worker smoke returned no text blocks")
	}

	report := smokeReport{
		SchemaVersion:     1,
		OK:                true,
		GeneratedAt:       time.Now().UTC(),
		GOOS:              runtime.GOOS,
		GOARCH:            runtime.GOARCH,
		WorkerPath:        workerPath,
		RuntimeDir:        runtimeDir,
		ModelPackage:      modelPackage,
		ModelID:           modelID,
		ModelDir:          modelDir,
		EvidencePath:      filepath.Join(evidenceDir, evidenceFileName),
		Capabilities:      capabilities,
		Smoke:             smoke,
		ExpectedTexts:     expectedTexts,
		ExpectedTextsSeen: seen,
	}
	if err := writeEvidence(report); err != nil {
		return smokeReport{}, err
	}
	return report, nil
}

func runCapabilities(workerPath string, runtimeDir string) (ocr.WorkerCapabilities, error) {
	stdout, stderr, err := runCommand(workerPath, "--capabilities", "--runtime-dir", runtimeDir)
	if err != nil {
		return ocr.WorkerCapabilities{}, fmt.Errorf("run OCR worker capabilities: %w; stderr: %s", err, strings.TrimSpace(stderr))
	}
	var capabilities ocr.WorkerCapabilities
	if err := json.Unmarshal(stdout, &capabilities); err != nil {
		return ocr.WorkerCapabilities{}, fmt.Errorf("decode OCR worker capabilities: %w; output: %s", err, strings.TrimSpace(string(stdout)))
	}
	if capabilities.SchemaVersion != 1 {
		return ocr.WorkerCapabilities{}, fmt.Errorf("capabilities schema = %d, want 1", capabilities.SchemaVersion)
	}
	return capabilities, nil
}

func runSmoke(workerPath string, runtimeDir string, modelDir string, expectedTexts []string) (workerSmokeResult, error) {
	args := []string{"--smoke", "--runtime-dir", runtimeDir, "--model-dir", modelDir}
	for _, text := range expectedTexts {
		args = append(args, "--must-contain", text)
	}
	stdout, stderr, err := runCommand(workerPath, args...)
	if err != nil {
		return workerSmokeResult{}, fmt.Errorf("run OCR worker smoke: %w; stderr: %s", err, strings.TrimSpace(stderr))
	}
	var smoke workerSmokeResult
	if err := json.Unmarshal(stdout, &smoke); err != nil {
		return workerSmokeResult{}, fmt.Errorf("decode OCR worker smoke: %w; output: %s", err, strings.TrimSpace(string(stdout)))
	}
	return smoke, nil
}

func runCommand(name string, args ...string) ([]byte, string, error) {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.String(), err
}

func prepareModelDir(source string, extractDir string, modelID string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		modelDir := filepath.Join(source, modelID)
		if _, err := os.Stat(filepath.Join(modelDir, "manifest.json")); err == nil {
			return modelDir, nil
		}
		if _, err := os.Stat(filepath.Join(source, "manifest.json")); err == nil {
			return source, nil
		}
		return "", fmt.Errorf("model directory %q does not contain %s/manifest.json or manifest.json", source, modelID)
	}
	if err := extractZip(source, extractDir); err != nil {
		return "", err
	}
	modelDir := filepath.Join(extractDir, modelID)
	if _, err := os.Stat(filepath.Join(modelDir, "manifest.json")); err == nil {
		return modelDir, nil
	}
	if _, err := os.Stat(filepath.Join(extractDir, "manifest.json")); err == nil {
		return extractDir, nil
	}
	return "", fmt.Errorf("model package %q does not contain %s/manifest.json or manifest.json", source, modelID)
}

func extractZip(source string, target string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		name := filepath.FromSlash(strings.TrimSpace(file.Name))
		if name == "" {
			continue
		}
		if !safeRelativePath(name) {
			return fmt.Errorf("unsafe zip path %q", file.Name)
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("zip path %q is a symlink", file.Name)
		}
		destination := filepath.Join(target, name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		err = writeStreamToFile(src, destination, file.FileInfo().Mode().Perm())
		closeErr := src.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func writeStreamToFile(reader io.Reader, target string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o644
	}
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, reader)
	return err
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
	return filepath.Abs(path)
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

func safeRelativePath(path string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." || filepath.IsAbs(path) {
		return false
	}
	if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func presentExpectedTexts(text string, expectedTexts []string) []string {
	present := make([]string, 0, len(expectedTexts))
	for _, expected := range expectedTexts {
		if strings.Contains(text, expected) {
			present = append(present, expected)
		}
	}
	return present
}

func missingExpectedTexts(expectedTexts []string, present []string) []string {
	seen := map[string]bool{}
	for _, text := range present {
		seen[text] = true
	}
	missing := make([]string, 0)
	for _, expected := range expectedTexts {
		if !seen[expected] {
			missing = append(missing, expected)
		}
	}
	return missing
}

func writeEvidence(report smokeReport) error {
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
