package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

func TestCreateModelZipCanBeImportedByOCRService(t *testing.T) {
	workDir := t.TempDir()
	files := map[string][]byte{
		"det.onnx": []byte("det-model"),
		"cls.onnx": []byte("cls-model"),
		"rec.onnx": []byte("rec-model"),
		"keys.txt": []byte("a\nb\nc\n"),
	}
	model := modelPackage{
		SchemaVersion: 1,
		ID:            "ppocrv5-mobile-zh-en",
		Name:          "PP-OCRv5 Mobile Chinese/English",
		Channel:       "stable",
		Engine:        "onnxruntime",
		Language:      []string{"zh", "en"},
		Version:       "test",
		Source:        ocr.ModelSource{URL: "https://example.invalid/model", License: "Apache-2.0"},
		Smoke: ocr.ModelSmoke{
			Image:       "smoke.png",
			Expected:    "smoke.expected.json",
			MustContain: []string{"RecordingFreedom", "文字识别"},
		},
	}
	downloaded := map[string]string{}
	for name, data := range files {
		path := filepath.Join(workDir, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
		model.Files = append(model.Files, sourceFile{
			Name:        name,
			DownloadURL: "https://example.invalid/" + name,
			Bytes:       int64(len(data)),
			SHA256:      sha256Hex(data),
		})
		downloaded[name] = path
	}

	smokePNG, err := decodeSmokePNG()
	if err != nil {
		t.Fatalf("decodeSmokePNG() error = %v", err)
	}
	smokeExpected := []byte(`{"schemaVersion":1,"mustContain":["RecordingFreedom","文字识别"]}` + "\n")
	manifest, err := buildPackageManifest(model, smokePNG, smokeExpected)
	if err != nil {
		t.Fatalf("buildPackageManifest() error = %v", err)
	}
	zipBytes, err := createModelZip(model, manifest, downloaded, smokePNG, smokeExpected)
	if err != nil {
		t.Fatalf("createModelZip() error = %v", err)
	}
	zipPath := filepath.Join(workDir, "model.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0o644); err != nil {
		t.Fatalf("WriteFile(zip) error = %v", err)
	}

	dataRoot := filepath.Join(workDir, "data-root")
	service := ocr.NewService(appdata.NewService(dataRoot))
	info, err := service.InstallModelPackage(zipPath)
	if err != nil {
		t.Fatalf("InstallModelPackage() error = %v", err)
	}
	if !info.Installed || !info.Verified || !info.SmokeAssetReady {
		t.Fatalf("model info = %#v, want installed, verified and smoke-ready", info)
	}
}

func TestCatalogManifestDeclaresStablePackage(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot() error = %v", err)
	}
	catalog, err := readCatalog(filepath.Join(root, "third_party", "ocr-models", "manifest.json"))
	if err != nil {
		t.Fatalf("readCatalog() error = %v", err)
	}
	model, ok := catalog.Models["ppocrv5-mobile-zh-en"]
	if !ok {
		t.Fatal("catalog missing ppocrv5-mobile-zh-en")
	}
	if err := validateModelPackageSpec(model); err != nil {
		t.Fatalf("validateModelPackageSpec() error = %v", err)
	}
	if model.Channel != "stable" {
		t.Fatalf("channel = %q, want stable", model.Channel)
	}
}

func TestCatalogManifestKeepsPPOCRv6CandidatesOutOfReleaseAll(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot() error = %v", err)
	}
	catalog, err := readCatalog(filepath.Join(root, "third_party", "ocr-models", "manifest.json"))
	if err != nil {
		t.Fatalf("readCatalog() error = %v", err)
	}
	for _, id := range []string{"ppocrv6-mobile-zh-en", "ppocrv6-medium-zh-en"} {
		model, ok := catalog.Models[id]
		if !ok {
			t.Fatalf("catalog missing %s candidate", id)
		}
		if status := modelReleaseStatus(model); status != modelReleaseStatusCandidate {
			t.Fatalf("%s releaseStatus = %q, want %q", id, status, modelReleaseStatusCandidate)
		}
	}
	ids, err := selectedModelIDs(catalog, "all", false)
	if err != nil {
		t.Fatalf("selectedModelIDs(all) error = %v", err)
	}
	if len(ids) != 1 || ids[0] != "ppocrv5-mobile-zh-en" {
		t.Fatalf("selectedModelIDs(all) = %#v, want only stable model", ids)
	}
	candidateIDs, err := selectedModelIDs(catalog, "all", true)
	if err != nil {
		t.Fatalf("selectedModelIDs(all includeCandidates) error = %v", err)
	}
	if strings.Join(candidateIDs, ",") != "ppocrv5-mobile-zh-en,ppocrv6-medium-zh-en,ppocrv6-mobile-zh-en" {
		t.Fatalf("selectedModelIDs(all includeCandidates) = %#v, want stable plus candidates", candidateIDs)
	}
}

func TestSelectedModelIDsRejectsExplicitCandidate(t *testing.T) {
	catalog := catalog{
		SchemaVersion: 1,
		Models: map[string]modelPackage{
			"stable":    {ID: "stable"},
			"candidate": {ID: "candidate", ReleaseStatus: modelReleaseStatusCandidate},
		},
	}
	if ids, err := selectedModelIDs(catalog, "all", false); err != nil || len(ids) != 1 || ids[0] != "stable" {
		t.Fatalf("selectedModelIDs(all) = %#v, %v; want only stable", ids, err)
	}
	if _, err := selectedModelIDs(catalog, "candidate", false); err == nil || !strings.Contains(err.Error(), "cannot be packaged for release") {
		t.Fatalf("selectedModelIDs(candidate) error = %v, want release candidate rejection", err)
	}
	if ids, err := selectedModelIDs(catalog, "candidate", true); err != nil || len(ids) != 1 || ids[0] != "candidate" {
		t.Fatalf("selectedModelIDs(candidate includeCandidates) = %#v, %v; want candidate selected for local smoke", ids, err)
	}
}

func TestPackageModelRejectsCandidateModel(t *testing.T) {
	model := modelPackage{ID: "ppocrv6-mobile-zh-en", ReleaseStatus: modelReleaseStatusCandidate}
	if _, err := packageModel(t.TempDir(), t.TempDir(), model, []byte("png"), true); err == nil || !strings.Contains(err.Error(), "cannot be packaged for release") {
		t.Fatalf("packageModel(candidate) error = %v, want release candidate rejection", err)
	}
}

func TestRunRejectsCandidateCatalogOutput(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest := `{
  "schemaVersion": 1,
  "models": {
    "ppocrv6-mobile-zh-en": {
      "schemaVersion": 1,
      "id": "ppocrv6-mobile-zh-en",
      "releaseStatus": "candidate"
    }
  }
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	err := run([]string{
		"-manifest", manifestPath,
		"-model", "ppocrv6-mobile-zh-en",
		"-include-candidates",
		"-output", filepath.Join(dir, "out"),
		"-catalog-output", filepath.Join(dir, "catalog.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "-include-candidates cannot be used with -catalog-output") {
		t.Fatalf("run(include candidate catalog) error = %v, want catalog-output rejection", err)
	}
}

func TestGeneratePaddleOCRCharacterDictKeys(t *testing.T) {
	data := []byte("Global:\n  model_name: PP-OCRv6_test_rec\nPostProcess:\n  name: CTCLabelDecode\n  character_dict:\n  - A\n  - 文\n  - \" \"\n")
	keys, err := generatePaddleOCRCharacterDictKeys(data)
	if err != nil {
		t.Fatalf("generatePaddleOCRCharacterDictKeys() error = %v", err)
	}
	if string(keys) != "A\n文\n \n" {
		t.Fatalf("keys = %q, want exact character_dict lines", string(keys))
	}
}

func TestValidateModelPackageSpecAllowsMissingClsWhenTextlineOrientationDisabled(t *testing.T) {
	model := modelPackage{
		SchemaVersion: 1,
		ID:            "ppocrv6-mobile-zh-en",
		Name:          "PP-OCRv6 Mobile Chinese/English",
		Channel:       "latest",
		Engine:        "onnxruntime",
		Language:      []string{"zh", "en"},
		Version:       "test",
		TextlineOrientation: &ocr.ModelTextlineOrientation{
			Mode: ocr.TextlineOrientationNone,
		},
		Files: []sourceFile{
			sourceFileForTest("det.onnx", "det"),
			sourceFileForTest("rec.onnx", "rec"),
			sourceFileForTest("keys.txt", "keys"),
		},
		Smoke: ocr.ModelSmoke{Image: "smoke.png", Expected: "smoke.expected.json", MustContain: []string{"RecordingFreedom"}},
	}
	if err := validateModelPackageSpec(model); err != nil {
		t.Fatalf("validateModelPackageSpec(no cls, orientation none) error = %v", err)
	}
	model.TextlineOrientation = nil
	if err := validateModelPackageSpec(model); err == nil || !strings.Contains(err.Error(), "missing required file cls.onnx") {
		t.Fatalf("validateModelPackageSpec(default no cls) error = %v, want cls required", err)
	}
}

func TestPackageModelGeneratesKeysFromPaddleOCRInferenceYAML(t *testing.T) {
	files := map[string][]byte{
		"/det.onnx":      []byte("det-model"),
		"/cls.onnx":      []byte("cls-model"),
		"/rec.onnx":      []byte("rec-model"),
		"/inference.yml": []byte("PostProcess:\n  name: CTCLabelDecode\n  character_dict:\n  - A\n  - 文\n  - \" \"\n"),
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data, ok := files[request.URL.Path]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write(data)
	}))
	defer server.Close()

	generatedKeys := []byte("A\n文\n \n")
	model := modelPackage{
		SchemaVersion: 1,
		ID:            "ppocrv6-mobile-zh-en",
		Name:          "PP-OCRv6 Mobile Chinese/English",
		Channel:       "latest",
		Engine:        "onnxruntime",
		Language:      []string{"zh", "en"},
		Version:       "test",
		Source:        ocr.ModelSource{URL: "https://huggingface.co/PaddlePaddle/PP-OCRv6_small_rec_onnx", License: "Apache-2.0"},
		Files: []sourceFile{
			{Name: "det.onnx", DownloadURL: server.URL + "/det.onnx", Bytes: int64(len(files["/det.onnx"])), SHA256: sha256Hex(files["/det.onnx"])},
			{Name: "cls.onnx", DownloadURL: server.URL + "/cls.onnx", Bytes: int64(len(files["/cls.onnx"])), SHA256: sha256Hex(files["/cls.onnx"])},
			{Name: "rec.onnx", DownloadURL: server.URL + "/rec.onnx", Bytes: int64(len(files["/rec.onnx"])), SHA256: sha256Hex(files["/rec.onnx"])},
			{
				Name:        "keys.txt",
				DownloadURL: server.URL + "/inference.yml",
				Bytes:       int64(len(generatedKeys)),
				SHA256:      sha256Hex(generatedKeys),
				Generate: &generatedFileSource{
					Type:         generatedPaddleOCRCharacterDictKeys,
					SourceBytes:  int64(len(files["/inference.yml"])),
					SourceSHA256: sha256Hex(files["/inference.yml"]),
				},
			},
		},
		Smoke: ocr.ModelSmoke{
			Image:       "smoke.png",
			Expected:    "smoke.expected.json",
			MustContain: []string{"RecordingFreedom", "文字识别"},
		},
	}
	smokePNG, err := decodeSmokePNG()
	if err != nil {
		t.Fatalf("decodeSmokePNG() error = %v", err)
	}
	result, err := packageModel(t.TempDir(), t.TempDir(), model, smokePNG, true)
	if err != nil {
		t.Fatalf("packageModel() error = %v", err)
	}
	reader, err := zip.OpenReader(result.Path)
	if err != nil {
		t.Fatalf("OpenReader(package) error = %v", err)
	}
	defer reader.Close()
	keys := readZipFile(t, &reader.Reader, model.ID+"/keys.txt")
	if string(keys) != string(generatedKeys) {
		t.Fatalf("generated keys.txt = %q, want %q", string(keys), string(generatedKeys))
	}
}

func TestWriteDownloadCatalogPinsGeneratedPackage(t *testing.T) {
	workDir := t.TempDir()
	manifest := ocr.ModelManifest{
		SchemaVersion: 1,
		ID:            "ppocrv5-mobile-zh-en",
		Name:          "PP-OCRv5 Mobile Chinese/English",
		Channel:       "stable",
		Engine:        "onnxruntime",
		Language:      []string{"zh", "en"},
		Version:       "test",
		Files: []ocr.ModelFile{
			{Name: "det.onnx", SHA256: strings.Repeat("a", 64), Bytes: 1},
			{Name: "cls.onnx", SHA256: strings.Repeat("b", 64), Bytes: 1},
			{Name: "rec.onnx", SHA256: strings.Repeat("c", 64), Bytes: 1},
			{Name: "keys.txt", SHA256: strings.Repeat("d", 64), Bytes: 1},
		},
		Smoke: ocr.ModelSmoke{Image: "smoke.png", Expected: "smoke.expected.json", MustContain: []string{"RecordingFreedom", "文字识别"}},
	}
	catalogPath := filepath.Join(workDir, "ocr-model-catalog.json")
	if err := writeDownloadCatalog(catalogPath, "https://github.com/lemon-casino/RecordingFreedom/releases/download/v0.1.0", []packagedModel{
		{
			Manifest: manifest,
			FileName: "ppocrv5-mobile-zh-en-test.zip",
			Bytes:    1234,
			SHA256:   strings.Repeat("e", 64),
		},
		{
			Manifest: ocr.ModelManifest{
				SchemaVersion: 1,
				ID:            "ppocrv6-mobile-zh-en",
				Name:          "PP-OCRv6 Mobile Chinese/English",
				Channel:       "latest",
				Engine:        "onnxruntime",
				Language:      []string{"zh", "en"},
				Version:       "test",
				Files:         manifest.Files,
				Smoke:         manifest.Smoke,
			},
			FileName: "ppocrv6-mobile-zh-en-test.zip",
			Bytes:    5678,
			SHA256:   strings.Repeat("f", 64),
		},
	}); err != nil {
		t.Fatalf("writeDownloadCatalog() error = %v", err)
	}
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("ReadFile(catalog) error = %v", err)
	}
	var catalog downloadCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatalf("Unmarshal(catalog) error = %v", err)
	}
	if len(catalog.Models) != 2 {
		t.Fatalf("models = %d, want 2", len(catalog.Models))
	}
	if catalog.Models[0].Package.URL != "https://github.com/lemon-casino/RecordingFreedom/releases/download/v0.1.0/ppocrv5-mobile-zh-en-test.zip" {
		t.Fatalf("package URL = %q", catalog.Models[0].Package.URL)
	}
	if catalog.Models[0].Package.Bytes != 1234 || catalog.Models[0].Package.SHA256 != strings.Repeat("e", 64) {
		t.Fatalf("package source = %#v, want pinned bytes and sha", catalog.Models[0].Package)
	}
	if catalog.Models[1].ID != "ppocrv6-mobile-zh-en" || catalog.Models[1].Package.Bytes != 5678 || catalog.Models[1].Package.SHA256 != strings.Repeat("f", 64) {
		t.Fatalf("second package = %#v, want latest package metadata", catalog.Models[1])
	}
}

func TestSelectedModelIDsSupportsAllAndCommaList(t *testing.T) {
	catalog := catalog{
		SchemaVersion: 1,
		Models: map[string]modelPackage{
			"ppocrv6-mobile-zh-en": {ID: "ppocrv6-mobile-zh-en"},
			"ppocrv5-mobile-zh-en": {ID: "ppocrv5-mobile-zh-en"},
		},
	}
	all, err := selectedModelIDs(catalog, "all", false)
	if err != nil {
		t.Fatalf("selectedModelIDs(all) error = %v", err)
	}
	if strings.Join(all, ",") != "ppocrv5-mobile-zh-en,ppocrv6-mobile-zh-en" {
		t.Fatalf("all ids = %v, want deterministic sorted ids", all)
	}
	selected, err := selectedModelIDs(catalog, " ppocrv6-mobile-zh-en, ppocrv5-mobile-zh-en ", false)
	if err != nil {
		t.Fatalf("selectedModelIDs(list) error = %v", err)
	}
	if strings.Join(selected, ",") != "ppocrv6-mobile-zh-en,ppocrv5-mobile-zh-en" {
		t.Fatalf("selected ids = %v, want input order", selected)
	}
	if _, err := selectedModelIDs(catalog, "missing-model", false); err == nil || !strings.Contains(err.Error(), "does not define") {
		t.Fatalf("missing model error = %v, want catalog error", err)
	}
}

func TestEmbeddedSmokePNGIsGoDecodable(t *testing.T) {
	smokePNG, err := decodeSmokePNG()
	if err != nil {
		t.Fatalf("decodeSmokePNG() error = %v", err)
	}
	image, err := png.Decode(bytes.NewReader(smokePNG))
	if err != nil {
		t.Fatalf("png.Decode(smokePNG) error = %v", err)
	}
	bounds := image.Bounds()
	if bounds.Dx() < 600 || bounds.Dy() < 160 {
		t.Fatalf("smoke image size = %dx%d, want a stable OCR-sized image", bounds.Dx(), bounds.Dy())
	}
}

func TestDownloadAndVerifyFileRetriesPartialDownloads(t *testing.T) {
	payload := []byte("verified-model-bytes")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts++
		if attempts == 1 {
			_, _ = writer.Write(payload[:4])
			return
		}
		_, _ = writer.Write(payload)
	}))
	defer server.Close()

	path, err := downloadAndVerifyFile(t.TempDir(), sourceFile{
		Name:        "det.onnx",
		DownloadURL: server.URL + "/det.onnx",
		Bytes:       int64(len(payload)),
		SHA256:      sha256Hex(payload),
	})
	if err != nil {
		t.Fatalf("downloadAndVerifyFile() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want retry after partial download", attempts)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(download) error = %v", err)
	}
	if string(data) != string(payload) {
		t.Fatalf("downloaded payload = %q, want verified full payload", string(data))
	}
}

func readZipFile(t *testing.T, reader *zip.Reader, name string) []byte {
	t.Helper()
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			t.Fatalf("Open(%s) error = %v", name, err)
		}
		defer handle.Close()
		data, err := io.ReadAll(handle)
		if err != nil {
			t.Fatalf("ReadAll(%s) error = %v", name, err)
		}
		return data
	}
	t.Fatalf("zip missing %s", name)
	return nil
}

func decodeSmokePNG() ([]byte, error) {
	return base64Decode(smokePNGBase64)
}

func base64Decode(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(value)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sourceFileForTest(name string, content string) sourceFile {
	data := []byte(content)
	return sourceFile{
		Name:        name,
		DownloadURL: "https://example.invalid/" + name,
		Bytes:       int64(len(data)),
		SHA256:      sha256Hex(data),
	}
}
