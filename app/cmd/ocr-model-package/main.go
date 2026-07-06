package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"gopkg.in/yaml.v3"
)

const smokePNGBase64 = "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"

const generatedPaddleOCRCharacterDictKeys = "paddleocr-character-dict-keys"
const modelReleaseStatusReady = "ready"
const modelReleaseStatusCandidate = "candidate"

type catalog struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Models        map[string]modelPackage `json:"models"`
}

type modelPackage struct {
	SchemaVersion       int                           `json:"schemaVersion"`
	ID                  string                        `json:"id"`
	Name                string                        `json:"name"`
	Channel             string                        `json:"channel"`
	Engine              string                        `json:"engine"`
	Language            []string                      `json:"language"`
	Version             string                        `json:"version"`
	Source              ocr.ModelSource               `json:"source"`
	ReleaseStatus       string                        `json:"releaseStatus,omitempty"`
	TextlineOrientation *ocr.ModelTextlineOrientation `json:"textlineOrientation,omitempty"`
	Files               []sourceFile                  `json:"files"`
	Smoke               ocr.ModelSmoke                `json:"smoke"`
}

type sourceFile struct {
	Name        string               `json:"name"`
	SourcePath  string               `json:"sourcePath,omitempty"`
	DownloadURL string               `json:"downloadUrl"`
	Bytes       int64                `json:"bytes"`
	SHA256      string               `json:"sha256"`
	Generate    *generatedFileSource `json:"generate,omitempty"`
}

type generatedFileSource struct {
	Type         string `json:"type"`
	SourceBytes  int64  `json:"sourceBytes"`
	SourceSHA256 string `json:"sourceSha256"`
}

type smokeExpected struct {
	SchemaVersion int      `json:"schemaVersion"`
	MustContain   []string `json:"mustContain"`
	Notes         string   `json:"notes"`
}

type downloadCatalog struct {
	SchemaVersion int                 `json:"schemaVersion"`
	GeneratedAt   time.Time           `json:"generatedAt"`
	Models        []ocr.ModelManifest `json:"models"`
}

type packagedModel struct {
	Manifest ocr.ModelManifest
	FileName string
	Bytes    int64
	SHA256   string
	Path     string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var modelID string
	var manifestPath string
	var outputDir string
	var catalogOutput string
	var releaseBaseURL string
	var force bool
	var includeCandidates bool
	flagSet := flag.NewFlagSet("ocr-model-package", flag.ContinueOnError)
	flagSet.StringVar(&modelID, "model", "ppocrv5-mobile-zh-en", "OCR model id to package")
	flagSet.StringVar(&manifestPath, "manifest", "", "OCR model package source manifest")
	flagSet.StringVar(&outputDir, "output", "", "output directory for the generated model zip")
	flagSet.StringVar(&catalogOutput, "catalog-output", "", "optional output path for a RecordingFreedom model download catalog")
	flagSet.StringVar(&releaseBaseURL, "release-base-url", "", "base release asset URL used in the generated model download catalog")
	flagSet.BoolVar(&force, "force", false, "overwrite an existing model zip")
	flagSet.BoolVar(&includeCandidates, "include-candidates", false, "allow releaseStatus=candidate models for local smoke packaging; cannot be used with -catalog-output")
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	root, err := repoRoot()
	if err != nil {
		return err
	}
	if strings.TrimSpace(manifestPath) == "" {
		manifestPath = filepath.Join(root, "third_party", "ocr-models", "manifest.json")
	}
	if strings.TrimSpace(outputDir) == "" {
		outputDir = filepath.Join(root, "dist", "ocr-models")
	}
	manifestPath, err = filepath.Abs(manifestPath)
	if err != nil {
		return err
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	if strings.TrimSpace(catalogOutput) != "" {
		catalogOutput, err = filepath.Abs(catalogOutput)
		if err != nil {
			return err
		}
	}

	catalog, err := readCatalog(manifestPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	workDir, err := os.MkdirTemp("", "recordingfreedom-ocr-model-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	smokePNG, err := base64.StdEncoding.DecodeString(smokePNGBase64)
	if err != nil {
		return err
	}

	modelIDs, err := selectedModelIDs(catalog, modelID, includeCandidates)
	if err != nil {
		return err
	}
	if includeCandidates && strings.TrimSpace(catalogOutput) != "" {
		for _, id := range modelIDs {
			if !modelIsReleaseReady(catalog.Models[id]) {
				return fmt.Errorf("-include-candidates cannot be used with -catalog-output because OCR model package %q is %s", id, modelReleaseStatus(catalog.Models[id]))
			}
		}
	}

	packages := make([]packagedModel, 0, len(modelIDs))
	for _, id := range modelIDs {
		model := catalog.Models[id]
		result, err := packageModelWithOptions(workDir, outputDir, model, smokePNG, force, packageOptions{AllowCandidate: includeCandidates})
		if err != nil {
			return err
		}
		packages = append(packages, result)
	}
	if strings.TrimSpace(catalogOutput) != "" {
		if err := writeDownloadCatalog(catalogOutput, strings.TrimSpace(releaseBaseURL), packages); err != nil {
			return err
		}
	}
	for _, result := range packages {
		fmt.Printf("OCR model package ready: %s\n", result.Path)
		fmt.Printf("SHA256: %s\n", result.SHA256)
	}
	return nil
}

func selectedModelIDs(catalog catalog, modelID string, includeCandidates bool) ([]string, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, errors.New("-model is required")
	}
	if strings.EqualFold(modelID, "all") {
		ids := make([]string, 0, len(catalog.Models))
		for id, model := range catalog.Models {
			if includeCandidates || modelIsReleaseReady(model) {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		if len(ids) == 0 {
			return nil, errors.New("-model all did not find any release-ready OCR models")
		}
		return ids, nil
	}
	seen := map[string]bool{}
	ids := []string{}
	for _, raw := range strings.Split(modelID, ",") {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate OCR model id %q", id)
		}
		model, ok := catalog.Models[id]
		if !ok {
			return nil, fmt.Errorf("OCR model package catalog does not define %q", id)
		}
		if !includeCandidates && !modelIsReleaseReady(model) {
			return nil, fmt.Errorf("OCR model package %q is %s and cannot be packaged for release", id, modelReleaseStatus(model))
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errors.New("-model did not select any OCR model")
	}
	return ids, nil
}

type packageOptions struct {
	AllowCandidate bool
}

func packageModel(workDir string, outputDir string, model modelPackage, smokePNG []byte, force bool) (packagedModel, error) {
	return packageModelWithOptions(workDir, outputDir, model, smokePNG, force, packageOptions{})
}

func packageModelWithOptions(workDir string, outputDir string, model modelPackage, smokePNG []byte, force bool, opts packageOptions) (packagedModel, error) {
	if !opts.AllowCandidate && !modelIsReleaseReady(model) {
		return packagedModel{}, fmt.Errorf("OCR model package %q is %s and cannot be packaged for release", model.ID, modelReleaseStatus(model))
	}
	if err := validateModelPackageSpec(model); err != nil {
		return packagedModel{}, err
	}
	outputPath := filepath.Join(outputDir, model.ID+"-"+model.Version+".zip")
	if _, err := os.Stat(outputPath); err == nil && !force {
		return packagedModel{}, fmt.Errorf("OCR model package already exists: %s", outputPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return packagedModel{}, err
	}

	modelWorkDir := filepath.Join(workDir, model.ID)
	if err := os.MkdirAll(modelWorkDir, 0o755); err != nil {
		return packagedModel{}, err
	}
	downloaded := make(map[string]string, len(model.Files))
	for _, file := range model.Files {
		path, err := materializePackageFile(modelWorkDir, file)
		if err != nil {
			return packagedModel{}, err
		}
		downloaded[file.Name] = path
	}

	smokeExpectedJSON, err := json.MarshalIndent(smokeExpected{
		SchemaVersion: 1,
		MustContain:   append([]string(nil), model.Smoke.MustContain...),
		Notes:         "G5 validates this asset as package data. G6 native worker smoke must verify OCR output against mustContain.",
	}, "", "  ")
	if err != nil {
		return packagedModel{}, err
	}
	smokeExpectedJSON = append(smokeExpectedJSON, '\n')

	packageManifest, err := buildPackageManifest(model, smokePNG, smokeExpectedJSON)
	if err != nil {
		return packagedModel{}, err
	}
	zipBytes, err := createModelZip(model, packageManifest, downloaded, smokePNG, smokeExpectedJSON)
	if err != nil {
		return packagedModel{}, err
	}
	if err := os.WriteFile(outputPath, zipBytes, 0o644); err != nil {
		return packagedModel{}, err
	}
	sum := sha256.Sum256(zipBytes)
	return packagedModel{
		Manifest: packageManifest,
		FileName: filepath.Base(outputPath),
		Bytes:    int64(len(zipBytes)),
		SHA256:   hex.EncodeToString(sum[:]),
		Path:     outputPath,
	}, nil
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "third_party", "ocr-models", "manifest.json")); err == nil {
			return wd, nil
		}
		if _, err := os.Stat(filepath.Join(wd, "..", "third_party", "ocr-models", "manifest.json")); err == nil {
			return filepath.Clean(filepath.Join(wd, "..")), nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", errors.New("could not find RecordingFreedom repository root")
		}
		wd = parent
	}
}

func readCatalog(path string) (catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return catalog{}, err
	}
	var result catalog
	if err := json.Unmarshal(data, &result); err != nil {
		return catalog{}, err
	}
	if result.SchemaVersion != 1 {
		return catalog{}, fmt.Errorf("unsupported OCR model catalog schema %d", result.SchemaVersion)
	}
	if len(result.Models) == 0 {
		return catalog{}, errors.New("OCR model catalog is empty")
	}
	for id, model := range result.Models {
		if status := modelReleaseStatus(model); status != modelReleaseStatusReady && status != modelReleaseStatusCandidate {
			return catalog{}, fmt.Errorf("OCR model package %q has unsupported releaseStatus %q", id, model.ReleaseStatus)
		}
	}
	return result, nil
}

func modelReleaseStatus(model modelPackage) string {
	status := strings.TrimSpace(model.ReleaseStatus)
	if status == "" {
		return modelReleaseStatusReady
	}
	return status
}

func modelIsReleaseReady(model modelPackage) bool {
	return modelReleaseStatus(model) == modelReleaseStatusReady
}

func validateModelPackageSpec(model modelPackage) error {
	if strings.TrimSpace(model.ID) == "" {
		return errors.New("OCR model package id is required")
	}
	if err := ocr.ValidateTextlineOrientationMode(ocr.ModelManifest{TextlineOrientation: model.TextlineOrientation}); err != nil {
		return fmt.Errorf("OCR model package %q is invalid: %w", model.ID, err)
	}
	required := make(map[string]bool)
	for _, name := range ocr.RequiredModelFileNames(ocr.ModelManifest{TextlineOrientation: model.TextlineOrientation}) {
		required[name] = false
	}
	for _, file := range model.Files {
		if !safePackageName(file.Name) {
			return fmt.Errorf("unsafe OCR model package file name %q", file.Name)
		}
		if strings.TrimSpace(file.DownloadURL) == "" {
			return fmt.Errorf("OCR model file %q missing downloadUrl", file.Name)
		}
		if file.Bytes <= 0 {
			return fmt.Errorf("OCR model file %q missing bytes", file.Name)
		}
		if len(file.SHA256) != 64 {
			return fmt.Errorf("OCR model file %q has invalid sha256", file.Name)
		}
		if file.Generate != nil {
			if file.Name != "keys.txt" {
				return fmt.Errorf("OCR model file %q cannot use generated source %q", file.Name, file.Generate.Type)
			}
			if file.Generate.Type != generatedPaddleOCRCharacterDictKeys {
				return fmt.Errorf("OCR model file %q has unsupported generate type %q", file.Name, file.Generate.Type)
			}
			if file.Generate.SourceBytes <= 0 {
				return fmt.Errorf("OCR model file %q missing generated source bytes", file.Name)
			}
			if len(file.Generate.SourceSHA256) != 64 {
				return fmt.Errorf("OCR model file %q has invalid generated source sha256", file.Name)
			}
		}
		if _, ok := required[file.Name]; ok {
			required[file.Name] = true
		}
	}
	for name, found := range required {
		if !found {
			return fmt.Errorf("OCR model %q is missing required file %s", model.ID, name)
		}
	}
	if strings.TrimSpace(model.Smoke.Image) == "" || strings.TrimSpace(model.Smoke.Expected) == "" {
		return fmt.Errorf("OCR model %q must declare smoke image and expected JSON", model.ID)
	}
	return nil
}

func downloadAndVerifyFile(workDir string, file sourceFile) (string, error) {
	target := filepath.Join(workDir, file.Name)
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt-1) * time.Second)
		}
		path, err := downloadAndVerifyFileOnce(target, file)
		if err == nil {
			return path, nil
		}
		lastErr = err
		_ = os.Remove(target)
	}
	return "", fmt.Errorf("download %s failed after retries: %w", file.Name, lastErr)
}

func materializePackageFile(workDir string, file sourceFile) (string, error) {
	if file.Generate == nil {
		return downloadAndVerifyFile(workDir, file)
	}
	switch file.Generate.Type {
	case generatedPaddleOCRCharacterDictKeys:
		return downloadGenerateAndVerifyFile(workDir, file, generatePaddleOCRCharacterDictKeys)
	default:
		return "", fmt.Errorf("unsupported generated OCR model file type %q", file.Generate.Type)
	}
}

func downloadGenerateAndVerifyFile(workDir string, file sourceFile, generate func([]byte) ([]byte, error)) (string, error) {
	sourcePath, err := downloadAndVerifyFile(workDir, sourceFile{
		Name:        file.Name + ".source",
		DownloadURL: file.DownloadURL,
		Bytes:       file.Generate.SourceBytes,
		SHA256:      file.Generate.SourceSHA256,
	})
	if err != nil {
		return "", err
	}
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	generated, err := generate(sourceData)
	if err != nil {
		return "", err
	}
	if int64(len(generated)) != file.Bytes {
		return "", fmt.Errorf("generated %s bytes = %d, want %d", file.Name, len(generated), file.Bytes)
	}
	sum := sha256.Sum256(generated)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actual, file.SHA256) {
		return "", fmt.Errorf("generated %s sha256 = %s, want %s", file.Name, actual, file.SHA256)
	}
	target := filepath.Join(workDir, file.Name)
	if err := os.WriteFile(target, generated, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func generatePaddleOCRCharacterDictKeys(data []byte) ([]byte, error) {
	characters, err := extractPaddleOCRCharacterDict(data)
	if err != nil {
		return nil, err
	}
	if len(characters) == 0 {
		return nil, errors.New("PaddleOCR inference.yml character_dict is empty")
	}
	for _, character := range characters {
		if strings.ContainsAny(character, "\r\n") {
			return nil, fmt.Errorf("PaddleOCR character_dict item contains newline: %q", character)
		}
	}
	return []byte(strings.Join(characters, "\n") + "\n"), nil
}

func extractPaddleOCRCharacterDict(data []byte) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse PaddleOCR inference.yml: %w", err)
	}
	postProcess := yamlMappingValue(&root, "PostProcess")
	if postProcess == nil {
		return nil, errors.New("PaddleOCR inference.yml missing PostProcess")
	}
	dict := yamlMappingValue(postProcess, "character_dict")
	if dict == nil {
		return nil, errors.New("PaddleOCR inference.yml missing PostProcess.character_dict")
	}
	if dict.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("PaddleOCR PostProcess.character_dict kind = %v, want sequence", dict.Kind)
	}
	characters := make([]string, 0, len(dict.Content))
	for _, item := range dict.Content {
		if item.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("PaddleOCR character_dict item kind = %v, want scalar", item.Kind)
		}
		characters = append(characters, item.Value)
	}
	return characters, nil
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return yamlMappingValue(node.Content[0], key)
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func downloadAndVerifyFileOnce(target string, file sourceFile) (string, error) {
	request, err := http.NewRequest(http.MethodGet, file.DownloadURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "RecordingFreedom-ocr-model-package/1")
	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("download %s failed with HTTP %s", file.DownloadURL, response.Status)
	}
	out, err := os.Create(target)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, hash), response.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written != file.Bytes {
		return "", fmt.Errorf("downloaded %s bytes = %d, want %d", file.Name, written, file.Bytes)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, file.SHA256) {
		return "", fmt.Errorf("downloaded %s sha256 = %s, want %s", file.Name, actual, file.SHA256)
	}
	return target, nil
}

func buildPackageManifest(model modelPackage, smokePNG []byte, smokeExpectedJSON []byte) (ocr.ModelManifest, error) {
	files := make([]ocr.ModelFile, 0, len(model.Files)+2)
	for _, file := range model.Files {
		files = append(files, ocr.ModelFile{Name: file.Name, SHA256: file.SHA256, Bytes: file.Bytes})
	}
	smokeHash := sha256.Sum256(smokePNG)
	files = append(files, ocr.ModelFile{Name: model.Smoke.Image, SHA256: hex.EncodeToString(smokeHash[:]), Bytes: int64(len(smokePNG))})
	expectedHash := sha256.Sum256(smokeExpectedJSON)
	files = append(files, ocr.ModelFile{Name: model.Smoke.Expected, SHA256: hex.EncodeToString(expectedHash[:]), Bytes: int64(len(smokeExpectedJSON))})
	return ocr.ModelManifest{
		SchemaVersion:       model.SchemaVersion,
		ID:                  model.ID,
		Name:                model.Name,
		Channel:             model.Channel,
		Engine:              model.Engine,
		Language:            append([]string(nil), model.Language...),
		Version:             model.Version,
		Source:              model.Source,
		TextlineOrientation: model.TextlineOrientation,
		Files:               files,
		Smoke:               model.Smoke,
	}, nil
}

func createModelZip(model modelPackage, manifest ocr.ModelManifest, files map[string]string, smokePNG []byte, smokeExpectedJSON []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	prefix := model.ID + "/"
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := writeZipFile(writer, prefix+"manifest.json", append(manifestData, '\n')); err != nil {
		return nil, err
	}
	for _, file := range model.Files {
		data, err := os.ReadFile(files[file.Name])
		if err != nil {
			return nil, err
		}
		if err := writeZipFile(writer, prefix+file.Name, data); err != nil {
			return nil, err
		}
	}
	if err := writeZipFile(writer, prefix+model.Smoke.Image, smokePNG); err != nil {
		return nil, err
	}
	if err := writeZipFile(writer, prefix+model.Smoke.Expected, smokeExpectedJSON); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeDownloadCatalog(path string, releaseBaseURL string, packages []packagedModel) error {
	if strings.TrimSpace(releaseBaseURL) == "" {
		return errors.New("-release-base-url is required when -catalog-output is set")
	}
	releaseBaseURL = strings.TrimRight(strings.TrimSpace(releaseBaseURL), "/")
	if len(packages) == 0 {
		return errors.New("no OCR model packages were generated for catalog output")
	}
	models := make([]ocr.ModelManifest, 0, len(packages))
	for _, result := range packages {
		if result.FileName == "" || strings.ContainsAny(result.FileName, `/\`) {
			return fmt.Errorf("invalid release model package file %q", result.FileName)
		}
		if result.Bytes <= 0 {
			return fmt.Errorf("invalid package size for %q", result.FileName)
		}
		if len(result.SHA256) != 64 {
			return fmt.Errorf("invalid package sha256 for %q", result.FileName)
		}
		manifest := result.Manifest
		manifest.Package = ocr.ModelPackageSource{
			URL:    releaseBaseURL + "/" + result.FileName,
			SHA256: result.SHA256,
			Bytes:  result.Bytes,
		}
		models = append(models, manifest)
	}
	catalog := downloadCatalog{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC(),
		Models:        models,
	}
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeZipFile(writer *zip.Writer, name string, data []byte) error {
	if !safeZipPath(name) {
		return fmt.Errorf("unsafe zip path %q", name)
	}
	header := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate}
	header.SetMode(0o644)
	file, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

func safePackageName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && !strings.ContainsAny(name, `/\`) && name != "." && name != ".."
}

func safeZipPath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "../") || strings.HasPrefix(path, "..") {
		return false
	}
	return true
}
