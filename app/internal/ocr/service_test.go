package ocr

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
)

func TestStatusDefaultsToNoModel(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))

	status, err := service.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Status != StatusNoModel {
		t.Fatalf("status = %q, want %q", status.Status, StatusNoModel)
	}
	if status.ActiveModelID != defaultActiveModelID() {
		t.Fatalf("active model = %q, want default", status.ActiveModelID)
	}
	if len(status.Models) != 3 {
		t.Fatalf("models len = %d, want 3", len(status.Models))
	}
	if status.Models[0].ID != defaultActiveModelID() || status.Models[0].Installed {
		t.Fatalf("first model = %#v, want default stable missing", status.Models[0])
	}
}

func TestNewServiceWithOptionsAppliesRuntimeOverrides(t *testing.T) {
	root := t.TempDir()
	service := NewServiceWithOptions(appdata.NewService(root), ServiceOptions{
		WorkerPath:             filepath.Join(root, "rf-ocr-worker-test"),
		RuntimeDir:             filepath.Join(root, "onnxruntime-test"),
		WorkerArgs:             []string{"--worker-arg"},
		WorkerCapabilitiesArgs: []string{"--capabilities-arg"},
		WorkerEnv:              []string{"RF_OCR_TEST=1"},
		WorkerTimeout:          3 * time.Second,
		ModelRegistry: []ModelManifest{
			{
				SchemaVersion: 1,
				ID:            "custom-model",
				Name:          "Custom Model",
				Channel:       "stable",
				Engine:        "onnxruntime",
				Language:      []string{"zh", "en"},
				Version:       "test",
				Files:         []ModelFile{{Name: "det.onnx"}, {Name: "rec.onnx"}, {Name: "keys.txt"}},
				TextlineOrientation: &ModelTextlineOrientation{
					Mode: TextlineOrientationNone,
				},
			},
		},
	})

	if service.workerPath() != filepath.Join(root, "rf-ocr-worker-test") {
		t.Fatalf("workerPath() = %q, want override", service.workerPath())
	}
	if service.runtimeDir() != filepath.Join(root, "onnxruntime-test") {
		t.Fatalf("runtimeDir() = %q, want override", service.runtimeDir())
	}
	if service.workerTimeout != 3*time.Second {
		t.Fatalf("workerTimeout = %s, want 3s", service.workerTimeout)
	}
	if got := strings.Join(service.workerArgs, " "); got != "--worker-arg" {
		t.Fatalf("workerArgs = %q, want override", got)
	}
	if got := strings.Join(service.workerCapabilitiesArgs, " "); got != "--capabilities-arg" {
		t.Fatalf("workerCapabilitiesArgs = %q, want override", got)
	}
	if got := strings.Join(service.workerEnv, " "); got != "RF_OCR_TEST=1" {
		t.Fatalf("workerEnv = %q, want override", got)
	}
	models := service.modelRegistry()
	if len(models) != 1 || models[0].ID != "custom-model" {
		t.Fatalf("modelRegistry() = %#v, want custom override", models)
	}
}

func TestSetActiveModelRequiresVerifiedModel(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))

	if _, err := service.SetActiveModel("ppocrv6-mobile-zh-en"); err == nil {
		t.Fatal("SetActiveModel() succeeded for missing model, want error")
	}
}

func TestInstallModelVerifiesManifestAndFiles(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := defaultActiveModelID()
	dir := filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	files := map[string]string{
		"det.onnx": "det",
		"cls.onnx": "cls",
		"rec.onnx": "rec",
		"keys.txt": "keys",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	manifest := `{
  "schemaVersion": 1,
  "id": "ppocrv5-mobile-zh-en",
  "name": "PP-OCRv5 Mobile Chinese/English",
  "channel": "stable",
  "engine": "onnxruntime",
  "language": ["zh", "en"],
  "version": "test",
  "files": [
    {"name": "det.onnx"},
    {"name": "cls.onnx"},
    {"name": "rec.onnx"},
    {"name": "keys.txt"}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}

	info, err := service.InstallModel(modelID)
	if err != nil {
		t.Fatalf("InstallModel() error = %v", err)
	}
	if !info.Installed || !info.Verified {
		t.Fatalf("installed model = %#v, want installed and verified", info)
	}
	status, err := service.SetActiveModel(modelID)
	if err != nil {
		t.Fatalf("SetActiveModel() error = %v", err)
	}
	if status.Status != StatusWorkerAbsent {
		t.Fatalf("status = %q, want worker absent after verified model", status.Status)
	}
}

func TestInstallModelPackageFromZipVerifiesAndDoesNotActivate(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := "ppocrv6-mobile-zh-en"
	packagePath := filepath.Join(root, "ppocrv6-mobile-zh-en.zip")
	writeModelPackageZip(t, packagePath, "bundle", modelID, map[string]string{
		"det.onnx": "det-v6",
		"cls.onnx": "cls-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	})

	info, err := service.InstallModelPackage(packagePath)
	if err != nil {
		t.Fatalf("InstallModelPackage() error = %v", err)
	}
	if info.ID != modelID || !info.Installed || !info.Verified {
		t.Fatalf("installed package info = %#v, want verified %s", info, modelID)
	}
	if info.Active {
		t.Fatalf("installed latest model became active; installs must not auto switch active model")
	}
	if info.SourceURL != "https://example.invalid/"+modelID || info.License != "Apache-2.0" {
		t.Fatalf("model source/license = %q/%q, want manifest metadata", info.SourceURL, info.License)
	}
	if info.SmokeImage != "smoke.png" || info.SmokeExpected != "smoke.expected.json" || !info.SmokeAssetReady || info.SmokeError != "" {
		t.Fatalf("smoke metadata = image:%q expected:%q ready:%v error:%q, want ready smoke assets", info.SmokeImage, info.SmokeExpected, info.SmokeAssetReady, info.SmokeError)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID, "rec.onnx")); err != nil {
		t.Fatalf("installed rec.onnx stat error = %v", err)
	}
	state, err := service.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.ActiveModelID != defaultActiveModelID() {
		t.Fatalf("active model = %q, want unchanged default", state.ActiveModelID)
	}
}

func TestInstallModelPackageAllowsMissingClsWhenTextlineOrientationDisabled(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := "ppocrv6-mobile-zh-en"
	packagePath := filepath.Join(root, "ppocrv6-mobile-zh-en-no-cls.zip")
	writeModelPackageZipWithOrientation(t, packagePath, "bundle", modelID, map[string]string{
		"det.onnx": "det-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	}, TextlineOrientationNone)

	info, err := service.InstallModelPackage(packagePath)
	if err != nil {
		t.Fatalf("InstallModelPackage(no cls, orientation none) error = %v", err)
	}
	if info.ID != modelID || !info.Installed || !info.Verified {
		t.Fatalf("installed package info = %#v, want verified no-cls %s", info, modelID)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID, "cls.onnx")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cls.onnx stat error = %v, want not installed for orientation none", err)
	}
}

func TestInstallModelPackageRejectsUnsafeZipPath(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	packagePath := filepath.Join(root, "unsafe.zip")
	file, err := os.Create(packagePath)
	if err != nil {
		t.Fatalf("Create(zip) error = %v", err)
	}
	zipWriter := zip.NewWriter(file)
	writer, err := zipWriter.Create("../manifest.json")
	if err != nil {
		t.Fatalf("Create(zip entry) error = %v", err)
	}
	if _, err := writer.Write([]byte("{}")); err != nil {
		t.Fatalf("Write(zip entry) error = %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zip Close() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file Close() error = %v", err)
	}

	if _, err := service.InstallModelPackage(packagePath); err == nil {
		t.Fatal("InstallModelPackage() succeeded for unsafe zip path, want error")
	}
	modelRoot := filepath.Join(root, "data", modelRootDir, ocrModelDir)
	if matches, _ := filepath.Glob(filepath.Join(modelRoot, modelInstallStagingPrefix+"*")); len(matches) != 0 {
		t.Fatalf("staging directories were not cleaned after failed install: %#v", matches)
	}
}

func TestStartModelDownloadInstallsVerifiedPackageAndDoesNotActivate(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := "ppocrv6-mobile-zh-en"
	packagePath := filepath.Join(root, "source", modelID+".zip")
	writeModelPackageZip(t, packagePath, "bundle", modelID, map[string]string{
		"det.onnx": "det-v6",
		"cls.onnx": "cls-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	})
	packageBytes, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("ReadFile(package) error = %v", err)
	}
	served := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(packageBytes)
	}))
	defer server.Close()
	service.modelRegistryOverride = []ModelManifest{downloadableTestModel(modelID, server.URL, packageBytes, "")}

	snapshot, err := service.StartModelDownload(modelID)
	if err != nil {
		t.Fatalf("StartModelDownload() error = %v", err)
	}
	if snapshot.Status != ModelDownloadQueued || snapshot.TotalBytes != int64(len(packageBytes)) {
		t.Fatalf("initial snapshot = %#v, want queued with total bytes", snapshot)
	}
	installed := waitModelDownloadStatus(t, service, modelID, ModelDownloadInstalled)
	if !served {
		t.Fatal("test server was not called")
	}
	if installed.Model == nil || installed.Model.ID != modelID || !installed.Model.Installed || !installed.Model.Verified {
		t.Fatalf("installed snapshot model = %#v, want verified %s", installed.Model, modelID)
	}
	if installed.DownloadedBytes != int64(len(packageBytes)) || installed.Percent != 100 {
		t.Fatalf("installed progress = %d %.2f, want complete", installed.DownloadedBytes, installed.Percent)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID, "rec.onnx")); err != nil {
		t.Fatalf("installed rec.onnx stat error = %v", err)
	}
	state, err := service.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.ActiveModelID != defaultActiveModelID() {
		t.Fatalf("active model = %q, want unchanged default", state.ActiveModelID)
	}
}

func TestStartModelDownloadRejectsChecksumMismatchWithoutInstall(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := "ppocrv6-mobile-zh-en"
	packagePath := filepath.Join(root, "source", modelID+".zip")
	writeModelPackageZip(t, packagePath, "bundle", modelID, map[string]string{
		"det.onnx": "det-v6",
		"cls.onnx": "cls-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	})
	packageBytes, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("ReadFile(package) error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(packageBytes)
	}))
	defer server.Close()
	service.modelRegistryOverride = []ModelManifest{downloadableTestModel(modelID, server.URL, packageBytes, strings.Repeat("0", 64))}

	if _, err := service.StartModelDownload(modelID); err != nil {
		t.Fatalf("StartModelDownload() error = %v", err)
	}
	failed := waitModelDownloadStatus(t, service, modelID, ModelDownloadFailed)
	if !strings.Contains(failed.Error, "sha256") {
		t.Fatalf("failed error = %q, want sha256 mismatch", failed.Error)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("download mismatch installed model dir unexpectedly; stat err = %v", err)
	}
}

func TestCancelModelDownloadStopsTransferWithoutInstall(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := "ppocrv6-mobile-zh-en"
	packagePath := filepath.Join(root, "source", modelID+".zip")
	writeModelPackageZip(t, packagePath, "bundle", modelID, map[string]string{
		"det.onnx": "det-v6",
		"cls.onnx": "cls-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	})
	packageBytes, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("ReadFile(package) error = %v", err)
	}
	requestStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		chunkEnd := 64
		if len(packageBytes) < chunkEnd {
			chunkEnd = len(packageBytes)
		}
		_, _ = w.Write(packageBytes[:chunkEnd])
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		close(requestStarted)
		select {
		case <-r.Context().Done():
		case <-releaseHandler:
		}
	}))
	defer server.Close()
	defer close(releaseHandler)
	service.modelRegistryOverride = []ModelManifest{downloadableTestModel(modelID, server.URL, packageBytes, "")}

	if _, err := service.StartModelDownload(modelID); err != nil {
		t.Fatalf("StartModelDownload() error = %v", err)
	}
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for test download request")
	}
	if _, err := service.CancelModelDownload(modelID); err != nil {
		t.Fatalf("CancelModelDownload() error = %v", err)
	}
	cancelled := waitModelDownloadStatus(t, service, modelID, ModelDownloadCancelled)
	if cancelled.Status != ModelDownloadCancelled {
		t.Fatalf("cancelled snapshot = %#v, want cancelled", cancelled)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled download installed model dir unexpectedly; stat err = %v", err)
	}
	waitNoModelDownloadStaging(t, service)
}

func TestRetryModelDownloadAfterCancelInstallsWithoutAutoSwitch(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := "ppocrv6-mobile-zh-en"
	packagePath := filepath.Join(root, "source", modelID+".zip")
	writeModelPackageZip(t, packagePath, "bundle", modelID, map[string]string{
		"det.onnx": "det-v6",
		"cls.onnx": "cls-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	})
	packageBytes, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("ReadFile(package) error = %v", err)
	}
	firstRequestStarted := make(chan struct{})
	releaseFirstHandler := make(chan struct{})
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestNumber := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		if requestNumber == 1 {
			chunkEnd := 64
			if len(packageBytes) < chunkEnd {
				chunkEnd = len(packageBytes)
			}
			_, _ = w.Write(packageBytes[:chunkEnd])
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			close(firstRequestStarted)
			select {
			case <-r.Context().Done():
			case <-releaseFirstHandler:
			}
			return
		}
		_, _ = w.Write(packageBytes)
	}))
	defer server.Close()
	defer close(releaseFirstHandler)
	service.modelRegistryOverride = []ModelManifest{downloadableTestModel(modelID, server.URL, packageBytes, "")}

	first, err := service.StartModelDownload(modelID)
	if err != nil {
		t.Fatalf("first StartModelDownload() error = %v", err)
	}
	select {
	case <-firstRequestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first download request")
	}
	if _, err := service.CancelModelDownload(modelID); err != nil {
		t.Fatalf("CancelModelDownload() error = %v", err)
	}
	cancelled := waitModelDownloadStatus(t, service, modelID, ModelDownloadCancelled)
	if cancelled.Status != ModelDownloadCancelled {
		t.Fatalf("cancelled snapshot = %#v, want cancelled", cancelled)
	}
	waitNoModelDownloadStaging(t, service)

	second, err := service.StartModelDownload(modelID)
	if err != nil {
		t.Fatalf("retry StartModelDownload() error = %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("retry reused cancelled download id %q", second.ID)
	}
	installed := waitModelDownloadStatus(t, service, modelID, ModelDownloadInstalled)
	if atomic.LoadInt32(&requestCount) != 2 {
		t.Fatalf("request count = %d, want exactly two requests", requestCount)
	}
	if installed.Model == nil || installed.Model.ID != modelID || !installed.Model.Installed || !installed.Model.Verified {
		t.Fatalf("retry installed snapshot model = %#v, want verified %s", installed.Model, modelID)
	}
	if installed.Model.Active {
		t.Fatalf("retry download auto-switched active model: %#v", installed.Model)
	}
	state, err := service.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.ActiveModelID != defaultActiveModelID() {
		t.Fatalf("active model = %q, want unchanged default after retry", state.ActiveModelID)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID, "rec.onnx")); err != nil {
		t.Fatalf("retry installed rec.onnx stat error = %v", err)
	}
	waitNoModelDownloadStaging(t, service)
}

func TestRefreshModelCatalogSavesVerifiedPackageRegistry(t *testing.T) {
	root := t.TempDir()
	packageBytes := []byte("verified-model-package")
	model := downloadableTestModel(defaultActiveModelID(), "http://127.0.0.1:1/ppocrv5-mobile-zh-en-test.zip", packageBytes, "")
	catalog := modelCatalogFile{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC(),
		Models:        []ModelManifest{model},
	}
	catalogBytes, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("Marshal(catalog) error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/ocr-model-catalog.json" {
			t.Fatalf("unexpected catalog path %s", request.URL.Path)
		}
		_, _ = writer.Write(catalogBytes)
	}))
	defer server.Close()
	model.Package.URL = server.URL + "/ppocrv5-mobile-zh-en-test.zip"
	catalog.Models[0] = model
	catalogBytes, err = json.Marshal(catalog)
	if err != nil {
		t.Fatalf("Marshal(catalog updated URL) error = %v", err)
	}

	service := NewService(appdata.NewService(root))
	status, err := service.RefreshModelCatalog(context.Background(), server.URL+"/ocr-model-catalog.json")
	if err != nil {
		t.Fatalf("RefreshModelCatalog() error = %v", err)
	}
	found := findModelInfo(status.Models, defaultActiveModelID())
	if found == nil || !found.DownloadAvailable || found.DownloadBytes != int64(len(packageBytes)) {
		t.Fatalf("refreshed model info = %#v, want downloadable model", found)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, modelRegistryFileName)); err != nil {
		t.Fatalf("saved registry was not written: %v", err)
	}

	restarted := NewService(appdata.NewService(root))
	models, err := restarted.ListModels()
	if err != nil {
		t.Fatalf("restarted ListModels() error = %v", err)
	}
	found = findModelInfo(models, defaultActiveModelID())
	if found == nil || !found.DownloadAvailable {
		t.Fatalf("restarted model info = %#v, want saved downloadable model", found)
	}
}

func TestParseModelCatalogAllowsMissingClsWhenTextlineOrientationDisabled(t *testing.T) {
	catalogBytes := []byte(`{"schemaVersion":1,"models":[{"schemaVersion":1,"id":"ppocrv6-mobile-zh-en","name":"PP-OCRv6 Mobile Chinese/English","channel":"latest","engine":"onnxruntime","language":["zh","en"],"version":"test","package":{"url":"https://example.invalid/ppocrv6.zip","sha256":"` + strings.Repeat("a", 64) + `","bytes":1},"textlineOrientation":{"mode":"none"},"files":[{"name":"det.onnx"},{"name":"rec.onnx"},{"name":"keys.txt"}]}]}`)
	models, err := parseModelCatalog(catalogBytes)
	if err != nil {
		t.Fatalf("parseModelCatalog(no cls, orientation none) error = %v", err)
	}
	if len(models) != 1 || models[0].TextlineOrientation == nil || models[0].TextlineOrientation.Mode != TextlineOrientationNone || len(models[0].Files) != 3 {
		t.Fatalf("models = %#v, want no-cls v6 catalog model", models)
	}
}

func TestParseModelCatalogRejectsMissingClsByDefault(t *testing.T) {
	catalogBytes := []byte(`{"schemaVersion":1,"models":[{"schemaVersion":1,"id":"ppocrv6-mobile-zh-en","name":"PP-OCRv6 Mobile Chinese/English","channel":"latest","engine":"onnxruntime","language":["zh","en"],"version":"test","package":{"url":"https://example.invalid/ppocrv6.zip","sha256":"` + strings.Repeat("a", 64) + `","bytes":1},"files":[{"name":"det.onnx"},{"name":"rec.onnx"},{"name":"keys.txt"}]}]}`)
	if _, err := parseModelCatalog(catalogBytes); err == nil || !strings.Contains(err.Error(), "missing required file cls.onnx") {
		t.Fatalf("parseModelCatalog(default missing cls) error = %v, want cls required", err)
	}
}

func TestRefreshModelCatalogRejectsUnsupportedModel(t *testing.T) {
	catalogBytes := []byte(`{"schemaVersion":1,"models":[{"schemaVersion":1,"id":"unknown-model","package":{"url":"https://example.invalid/model.zip","sha256":"` + strings.Repeat("a", 64) + `","bytes":1},"files":[{"name":"det.onnx"},{"name":"cls.onnx"},{"name":"rec.onnx"},{"name":"keys.txt"}]}]}`)
	if _, err := parseModelCatalog(catalogBytes); err == nil || !strings.Contains(err.Error(), "unsupported model") {
		t.Fatalf("parseModelCatalog(unknown) error = %v, want unsupported model", err)
	}
}

func TestInstallModelPackageKeepsExistingModelOnInvalidReplacement(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := defaultActiveModelID()
	validPath := filepath.Join(root, "valid.zip")
	writeModelPackageZip(t, validPath, "", modelID, map[string]string{
		"det.onnx": "det-valid",
		"cls.onnx": "cls-valid",
		"rec.onnx": "rec-valid",
		"keys.txt": "keys-valid",
	})
	if _, err := service.InstallModelPackage(validPath); err != nil {
		t.Fatalf("InstallModelPackage(valid) error = %v", err)
	}
	installedRecPath := filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID, "rec.onnx")
	before, err := os.ReadFile(installedRecPath)
	if err != nil {
		t.Fatalf("ReadFile(installed rec) error = %v", err)
	}
	invalidPath := filepath.Join(root, "invalid.zip")
	writeModelPackageZipWithBadSHA(t, invalidPath, "", modelID, map[string]string{
		"det.onnx": "det-invalid",
		"cls.onnx": "cls-invalid",
		"rec.onnx": "rec-invalid",
		"keys.txt": "keys-invalid",
	}, "definitely-wrong")
	if _, err := service.InstallModelPackage(invalidPath); err == nil {
		t.Fatal("InstallModelPackage(invalid) succeeded, want verification error")
	}
	after, err := os.ReadFile(installedRecPath)
	if err != nil {
		t.Fatalf("ReadFile(installed rec after invalid) error = %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("installed model was changed after invalid replacement: %q -> %q", before, after)
	}
}

func TestInstallModelPackageRejectsMissingSmokeAssets(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	modelID := defaultActiveModelID()
	packagePath := filepath.Join(root, "missing-smoke.zip")
	if err := os.MkdirAll(filepath.Dir(packagePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(zip dir) error = %v", err)
	}
	file, err := os.Create(packagePath)
	if err != nil {
		t.Fatalf("Create(zip) error = %v", err)
	}
	zipWriter := zip.NewWriter(file)
	files := map[string]string{
		"det.onnx": "det",
		"cls.onnx": "cls",
		"rec.onnx": "rec",
		"keys.txt": "keys",
	}
	writeZipEntry(t, zipWriter, "manifest.json", modelPackageManifestJSON(modelID, files, TextlineOrientationCLS, ""))
	for name, content := range files {
		writeZipEntry(t, zipWriter, name, content)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zip Close() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file Close() error = %v", err)
	}

	if _, err := service.InstallModelPackage(packagePath); err == nil || !strings.Contains(err.Error(), "smoke image") {
		t.Fatalf("InstallModelPackage(missing smoke) error = %v, want smoke image error", err)
	}
}

func TestRemoveInactiveModelKeepsActiveModel(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	activeModelID := defaultActiveModelID()
	inactiveModelID := "ppocrv6-mobile-zh-en"
	writeVerifiedTestModel(t, root, activeModelID)
	packagePath := filepath.Join(root, inactiveModelID+".zip")
	writeModelPackageZip(t, packagePath, "bundle", inactiveModelID, map[string]string{
		"det.onnx": "det-v6",
		"cls.onnx": "cls-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	})
	if _, err := service.InstallModelPackage(packagePath); err != nil {
		t.Fatalf("InstallModelPackage(inactive) error = %v", err)
	}
	if _, err := service.SetActiveModel(activeModelID); err != nil {
		t.Fatalf("SetActiveModel(default) error = %v", err)
	}

	status, err := service.RemoveModel(inactiveModelID)
	if err != nil {
		t.Fatalf("RemoveModel(inactive) error = %v", err)
	}
	if status.ActiveModelID != activeModelID {
		t.Fatalf("active model = %q, want unchanged %q", status.ActiveModelID, activeModelID)
	}
	active := findModelInfo(status.Models, activeModelID)
	if active == nil || !active.Active || !active.Installed || !active.Verified {
		t.Fatalf("active model info after inactive remove = %#v, want installed verified active", active)
	}
	removed := findModelInfo(status.Models, inactiveModelID)
	if removed == nil || removed.Installed || removed.Verified {
		t.Fatalf("removed inactive model info = %#v, want not installed", removed)
	}
	state, err := service.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.ActiveModelID != activeModelID {
		t.Fatalf("persisted active model = %q, want unchanged %q", state.ActiveModelID, activeModelID)
	}
}

func TestRemoveActiveModelFallsBackToDefaultModel(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	defaultModelID := defaultActiveModelID()
	activeModelID := "ppocrv6-mobile-zh-en"
	writeVerifiedTestModel(t, root, defaultModelID)
	packagePath := filepath.Join(root, activeModelID+".zip")
	writeModelPackageZip(t, packagePath, "bundle", activeModelID, map[string]string{
		"det.onnx": "det-v6",
		"cls.onnx": "cls-v6",
		"rec.onnx": "rec-v6",
		"keys.txt": "keys-v6",
	})
	if _, err := service.InstallModelPackage(packagePath); err != nil {
		t.Fatalf("InstallModelPackage(active) error = %v", err)
	}
	if _, err := service.SetActiveModel(activeModelID); err != nil {
		t.Fatalf("SetActiveModel(active) error = %v", err)
	}

	status, err := service.RemoveModel(activeModelID)
	if err != nil {
		t.Fatalf("RemoveModel(active) error = %v", err)
	}
	if status.ActiveModelID != defaultModelID {
		t.Fatalf("active model = %q, want fallback %q", status.ActiveModelID, defaultModelID)
	}
	fallback := findModelInfo(status.Models, defaultModelID)
	if fallback == nil || !fallback.Active || !fallback.Installed || !fallback.Verified {
		t.Fatalf("fallback model info = %#v, want installed verified active", fallback)
	}
	removed := findModelInfo(status.Models, activeModelID)
	if removed == nil || removed.Active || removed.Installed || removed.Verified {
		t.Fatalf("removed active model info = %#v, want inactive and not installed", removed)
	}
	state, err := service.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.ActiveModelID != defaultModelID {
		t.Fatalf("persisted active model = %q, want fallback %q", state.ActiveModelID, defaultModelID)
	}
	if _, err := os.Stat(filepath.Join(root, "data", modelRootDir, ocrModelDir, activeModelID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed active model dir still exists; stat err = %v", err)
	}
}

func TestStatusRequiresWorkerCapabilities(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	writeVerifiedTestModel(t, root, defaultActiveModelID())
	service.workerPathOverride = os.Args[0]
	service.workerCapabilitiesArgs = []string{"-test.run=TestOCRWorkerHelperProcess", "--", "--capabilities"}
	service.workerEnv = []string{"RF_OCR_WORKER_HELPER=1", "RF_OCR_WORKER_RECOGNIZE=0"}

	unavailable, err := service.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if unavailable.Status != StatusWorkerUnavailable {
		t.Fatalf("status = %q, want %q", unavailable.Status, StatusWorkerUnavailable)
	}
	if unavailable.WorkerCapabilities == nil || unavailable.WorkerCapabilities.SupportsRecognize {
		t.Fatalf("capabilities = %#v, want non-recognition worker", unavailable.WorkerCapabilities)
	}

	service.workerEnv = []string{"RF_OCR_WORKER_HELPER=1", "RF_OCR_WORKER_RECOGNIZE=1"}
	ready, err := service.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if ready.Status != StatusReady {
		t.Fatalf("status = %q, want %q; message=%s", ready.Status, StatusReady, ready.Message)
	}
	if ready.WorkerCapabilities == nil || !ready.WorkerCapabilities.SupportsRecognize {
		t.Fatalf("capabilities = %#v, want recognition worker", ready.WorkerCapabilities)
	}
}

func TestWriteResultCreatesResultAndCacheEntries(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	result := Result{
		ID:          "ocr_1",
		SourceKind:  SourceRegionScreenshot,
		SourceID:    "shot",
		ImagePath:   "/tmp/shot.png",
		ImageSHA256: "abc123",
		ModelID:     defaultActiveModelID(),
		Language:    defaultLanguage,
		Width:       120,
		Height:      80,
		Blocks:      []Block{{ID: "b1", Text: "RecordingFreedom", Confidence: 0.98}},
		PlainText:   "RecordingFreedom",
		CreatedAt:   now,
	}

	if err := service.WriteResult(result); err != nil {
		t.Fatalf("WriteResult() error = %v", err)
	}
	loaded, err := service.ReadResult("ocr_1")
	if err != nil {
		t.Fatalf("ReadResult() error = %v", err)
	}
	if loaded.PlainText != result.PlainText {
		t.Fatalf("loaded plain text = %q, want %q", loaded.PlainText, result.PlainText)
	}
	cached, ok, err := service.readCachedResult(result.ImageSHA256, result.ModelID, result.Language)
	if err != nil {
		t.Fatalf("readCachedResult() error = %v", err)
	}
	if !ok || cached.ID != result.ID {
		t.Fatalf("cached result = %#v ok=%v, want result", cached, ok)
	}
}

func TestTranslateDisabledProviderDoesNotCallNetwork(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))
	writeTranslationTestResult(t, service)
	var hitCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	_, err := service.Translate(TranslateRequest{
		OcrResultID:    "ocr_translate_fixture",
		Provider:       "disabled",
		BaseURL:        server.URL,
		SourceLanguage: "en",
		TargetLanguage: "zh-CN",
	})
	if err == nil || !strings.Contains(err.Error(), "provider is disabled") {
		t.Fatalf("Translate(disabled) error = %v, want disabled provider error", err)
	}
	if hitCount != 0 {
		t.Fatalf("disabled translation called provider %d times, want 0", hitCount)
	}
}

func TestTranslateOpenAICompatibleWritesAndReadsCache(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))
	writeTranslationTestResult(t, service)
	var hitCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		if hitCount > 1 {
			t.Errorf("OpenAI-compatible provider called after cache was written")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("request path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want bearer key", got)
		}
		var request openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Decode(request) error = %v", err)
		}
		if request.Model != "rf-translator" {
			t.Errorf("model = %q, want rf-translator", request.Model)
		}
		if len(request.Messages) != 2 || !strings.Contains(request.Messages[1].Content, "RecordingFreedom") || !strings.Contains(request.Messages[1].Content, "文字识别") {
			t.Errorf("prompt messages = %#v, want OCR block text", request.Messages)
		}
		response := openAIChatResponse{}
		response.Choices = append(response.Choices, struct {
			Message openAIChatMessage `json:"message"`
		}{
			Message: openAIChatMessage{
				Role:    "assistant",
				Content: `[{"blockId":"b1","translated":"RecordingFreedom translated"},{"blockId":"b2","translated":"Text translated"}]`,
			},
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	request := TranslateRequest{
		OcrResultID:    "ocr_translate_fixture",
		Provider:       "openai-compatible",
		SourceLanguage: "auto",
		TargetLanguage: "zh-CN",
		BaseURL:        server.URL,
		APIKey:         "test-key",
		Model:          "rf-translator",
	}

	first, err := service.Translate(request)
	if err != nil {
		t.Fatalf("Translate(openai first) error = %v", err)
	}
	if first.Provider != "openai-compatible" || first.Model != "rf-translator" || first.PromptVersion != translationPromptVersion {
		t.Fatalf("translation metadata = %#v, want provider/model/prompt version", first)
	}
	if got := translationBlockTexts(first.Blocks); got != "b1=RecordingFreedom translated;b2=Text translated" {
		t.Fatalf("translated blocks = %s, want translated text in OCR block order", got)
	}

	second, err := service.Translate(request)
	if err != nil {
		t.Fatalf("Translate(openai cached) error = %v", err)
	}
	if hitCount != 1 {
		t.Fatalf("provider hit count = %d, want 1 after cache hit", hitCount)
	}
	if got := translationBlockTexts(second.Blocks); got != translationBlockTexts(first.Blocks) {
		t.Fatalf("cached blocks = %s, want %s", got, translationBlockTexts(first.Blocks))
	}
}

func TestTranslateDeepLSegments(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))
	writeTranslationTestResult(t, service)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("auth_key"); got != "deepl-key" {
			t.Errorf("auth_key = %q, want deepl-key", got)
		}
		if got := r.Form.Get("source_lang"); got != "EN" {
			t.Errorf("source_lang = %q, want EN", got)
		}
		if got := r.Form.Get("target_lang"); got != "ZH" {
			t.Errorf("target_lang = %q, want ZH", got)
		}
		if got := strings.Join(r.Form["text"], "|"); got != "RecordingFreedom|文字识别" {
			t.Errorf("text form values = %q, want OCR blocks", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translations":[{"text":"录制自由"},{"text":"文字识别"}]}`))
	}))
	defer server.Close()

	result, err := service.Translate(TranslateRequest{
		OcrResultID:    "ocr_translate_fixture",
		Provider:       "deepl",
		SourceLanguage: "en",
		TargetLanguage: "zh-CN",
		BaseURL:        server.URL,
		APIKey:         "deepl-key",
		Force:          true,
	})
	if err != nil {
		t.Fatalf("Translate(deepl) error = %v", err)
	}
	if got := translationBlockTexts(result.Blocks); got != "b1=录制自由;b2=文字识别" {
		t.Fatalf("translated blocks = %s, want DeepL mapped blocks", got)
	}
}

func TestTranslateRejectsSegmentMismatchWithoutCache(t *testing.T) {
	service := NewService(appdata.NewService(t.TempDir()))
	writeTranslationTestResult(t, service)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translations":[{"text":"only one"}]}`))
	}))
	defer server.Close()

	_, err := service.Translate(TranslateRequest{
		OcrResultID:    "ocr_translate_fixture",
		Provider:       "deepl",
		SourceLanguage: "en",
		TargetLanguage: "zh-CN",
		BaseURL:        server.URL,
		APIKey:         "deepl-key",
		Force:          true,
	})
	if err == nil || !strings.Contains(err.Error(), "returned 1 segments for 2 OCR blocks") {
		t.Fatalf("Translate(mismatch) error = %v, want segment mismatch", err)
	}
	dir, err := service.translationsDir()
	if err != nil {
		t.Fatalf("translationsDir() error = %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(translations) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("translation cache entries = %d, want 0 after failed provider response", len(entries))
	}
}

func TestRecognizeImageRunsWorkerAndWritesCache(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	service.workerPathOverride = os.Args[0]
	service.workerArgs = []string{"-test.run=TestOCRWorkerHelperProcess"}
	service.workerCapabilitiesArgs = []string{"-test.run=TestOCRWorkerHelperProcess", "--", "--capabilities"}
	service.workerEnv = []string{"RF_OCR_WORKER_HELPER=1", "RF_OCR_WORKER_RECOGNIZE=1"}
	service.workerTimeout = 5 * time.Second
	writeVerifiedTestModel(t, root, defaultActiveModelID())
	imagePath := writeTestPNG(t, root, 80, 60)

	req := RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: SourceRegionScreenshot,
		SourceID:   "shot",
		Language:   defaultLanguage,
		Priority:   JobPriorityInteractive,
	}
	result, err := service.RecognizeImage(req)
	if err != nil {
		t.Fatalf("RecognizeImage() error = %v", err)
	}
	if result.ID != "ocr_worker_result" || result.PlainText != "RecordingFreedom\n文字识别" {
		t.Fatalf("OCR result = %#v, want test helper result", result)
	}
	if result.ImageSHA256 == "" || result.ModelID != defaultActiveModelID() || result.Width != 80 || result.Height != 60 {
		t.Fatalf("normalized OCR result = %#v, want image/model metadata", result)
	}

	service.workerPathOverride = filepath.Join(root, "missing-worker")
	cached, err := service.RecognizeImage(req)
	if err != nil {
		t.Fatalf("RecognizeImage(cached) error = %v", err)
	}
	if cached.ID != result.ID || cached.PlainText != result.PlainText {
		t.Fatalf("cached result = %#v, want original result %#v", cached, result)
	}

	secondReq := req
	secondReq.SourceKind = SourceFullScreenshot
	secondReq.SourceID = "shot-cache-2"
	secondCached, err := service.RecognizeImage(secondReq)
	if err != nil {
		t.Fatalf("RecognizeImage(cached second source) error = %v", err)
	}
	if secondCached.ID == result.ID {
		t.Fatalf("second source cached result reused original result id %q", secondCached.ID)
	}
	if secondCached.SourceKind != SourceFullScreenshot || secondCached.SourceID != "shot-cache-2" || secondCached.ImagePath != imagePath {
		t.Fatalf("second source cached result = %#v, want full screenshot source", secondCached)
	}
	persistedSecond, err := service.ReadResult(secondCached.ID)
	if err != nil {
		t.Fatalf("ReadResult(second cached result) error = %v", err)
	}
	if persistedSecond.SourceKind != secondCached.SourceKind || persistedSecond.SourceID != secondCached.SourceID || persistedSecond.ImagePath != imagePath {
		t.Fatalf("persisted second cached result = %#v, want current source metadata", persistedSecond)
	}
}

func TestRecognizeScrollingScreenshotUsesTilesAndDedupesOverlap(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "tile-worker.log")
	service := NewService(appdata.NewService(root))
	service.workerPathOverride = os.Args[0]
	service.workerArgs = []string{"-test.run=TestOCRWorkerHelperProcess"}
	service.workerCapabilitiesArgs = []string{"-test.run=TestOCRWorkerHelperProcess", "--", "--capabilities"}
	service.workerEnv = []string{
		"RF_OCR_WORKER_HELPER=1",
		"RF_OCR_WORKER_RECOGNIZE=1",
		"RF_OCR_WORKER_TILE=1",
		"RF_OCR_WORKER_LOG=" + logPath,
	}
	service.workerTimeout = 5 * time.Second
	writeVerifiedTestModel(t, root, defaultActiveModelID())
	imagePath := writeTestPNG(t, root, 320, 4520)

	result, err := service.RecognizeImage(RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: SourceScrollingScreenshot,
		SourceID:   "scroll-shot",
		Language:   defaultLanguage,
		Priority:   JobPriorityBackground,
	})
	if err != nil {
		t.Fatalf("RecognizeImage(scrolling tiled) error = %v", err)
	}
	if result.SourceKind != SourceScrollingScreenshot || result.SourceID != "scroll-shot" {
		t.Fatalf("result source = %s/%s, want scrolling source", result.SourceKind, result.SourceID)
	}
	if result.Width != 320 || result.Height != 4520 {
		t.Fatalf("result dimensions = %dx%d, want original long image dimensions", result.Width, result.Height)
	}
	wantText := "tile-0\noverlap-2080\ntile-2080\noverlap-4160\ntile-4160"
	if result.PlainText != wantText {
		t.Fatalf("plain text = %q, want %q", result.PlainText, wantText)
	}
	if len(result.Blocks) != 5 {
		t.Fatalf("blocks len = %d, want 5 after overlap dedupe: %#v", len(result.Blocks), result.Blocks)
	}
	for index, block := range result.Blocks {
		if block.LineIndex != index {
			t.Fatalf("block %d line index = %d, want %d", index, block.LineIndex, index)
		}
	}
	secondTileBounds, ok := ocrBlockBounds(result.Blocks[2])
	if !ok || math.Round(secondTileBounds.top) != 2120 {
		t.Fatalf("second tile block bounds = %#v ok=%v, want y mapped to 2120", secondTileBounds, ok)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(tile log) error = %v", err)
	}
	if got := strings.Count(string(logData), "recognize "); got != 3 {
		t.Fatalf("tile recognize calls = %d, want 3; log=%s", got, string(logData))
	}

	service.workerPathOverride = filepath.Join(root, "missing-worker")
	cached, err := service.RecognizeImage(RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: SourceScrollingScreenshot,
		SourceID:   "scroll-shot-cache",
		Language:   defaultLanguage,
	})
	if err != nil {
		t.Fatalf("RecognizeImage(scrolling cached) error = %v", err)
	}
	if cached.SourceID != "scroll-shot-cache" || cached.PlainText != result.PlainText {
		t.Fatalf("cached scrolling result = %#v, want current source with cached tiled text", cached)
	}
}

func TestEnqueueRecognizeMergesSameImageAndEmitsPerSourceResults(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	service.workerPathOverride = os.Args[0]
	service.workerArgs = []string{"-test.run=TestOCRWorkerHelperProcess"}
	service.workerCapabilitiesArgs = []string{"-test.run=TestOCRWorkerHelperProcess", "--", "--capabilities"}
	service.workerEnv = []string{"RF_OCR_WORKER_HELPER=1", "RF_OCR_WORKER_RECOGNIZE=1"}
	service.workerTimeout = 5 * time.Second
	writeVerifiedTestModel(t, root, defaultActiveModelID())
	imagePath := writeTestPNG(t, root, 80, 60)

	first, err := service.EnqueueRecognize(RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: SourceRegionScreenshot,
		SourceID:   "shot-1",
		Language:   defaultLanguage,
		Priority:   JobPriorityBackground,
	})
	if err != nil {
		t.Fatalf("EnqueueRecognize(first) error = %v", err)
	}
	second, err := service.EnqueueRecognize(RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: SourcePinnedScreenshot,
		SourceID:   "shot-2",
		Language:   defaultLanguage,
		Priority:   JobPriorityInteractive,
	})
	if err != nil {
		t.Fatalf("EnqueueRecognize(second) error = %v", err)
	}
	if first.JobID != second.JobID || !second.Merged {
		t.Fatalf("job snapshots = %#v / %#v, want merged same job", first, second)
	}

	ready := map[string]Result{}
	deadline := time.After(3 * time.Second)
	for len(ready) < 2 {
		select {
		case event := <-service.Events():
			if event.Status == ResultStatusReady && event.Result != nil {
				ready[event.Request.SourceID] = *event.Result
			}
		case <-deadline:
			t.Fatalf("timed out waiting for two ready OCR events; got %#v", ready)
		}
	}
	if ready["shot-1"].SourceKind != SourceRegionScreenshot || ready["shot-1"].SourceID != "shot-1" {
		t.Fatalf("shot-1 result = %#v, want region source", ready["shot-1"])
	}
	if ready["shot-2"].SourceKind != SourcePinnedScreenshot || ready["shot-2"].SourceID != "shot-2" {
		t.Fatalf("shot-2 result = %#v, want pinned source", ready["shot-2"])
	}
	if ready["shot-1"].PlainText != "RecordingFreedom\n文字识别" || ready["shot-2"].PlainText != ready["shot-1"].PlainText {
		t.Fatalf("ready plain text = %#v, want helper text for both sources", ready)
	}
}

func TestCancelQueuedJobEmitsCancelledAndRemovesJob(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	service.jobStarted = true
	writeVerifiedTestModel(t, root, defaultActiveModelID())
	imagePath := writeTestPNG(t, root, 80, 60)

	snapshot, err := service.EnqueueRecognize(RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: SourceRegionScreenshot,
		SourceID:   "shot-cancel",
		Language:   defaultLanguage,
		Priority:   JobPriorityNormal,
	})
	if err != nil {
		t.Fatalf("EnqueueRecognize() error = %v", err)
	}
	if err := service.CancelJob(snapshot.JobID); err != nil {
		t.Fatalf("CancelJob() error = %v", err)
	}
	if service.jobsByID[snapshot.JobID] != nil {
		t.Fatalf("cancelled job still exists in jobsByID")
	}
	var cancelled bool
	deadline := time.After(time.Second)
	for !cancelled {
		select {
		case event := <-service.Events():
			cancelled = event.JobID == snapshot.JobID && event.Status == ResultStatusCancelled
		case <-deadline:
			t.Fatal("timed out waiting for cancelled OCR event")
		}
	}
}

func TestQueuedJobsPreferInteractivePriority(t *testing.T) {
	root := t.TempDir()
	service := NewService(appdata.NewService(root))
	service.jobStarted = true
	writeVerifiedTestModel(t, root, defaultActiveModelID())
	imagePath := writeTestPNG(t, root, 80, 60)
	backgroundImagePath := writeTestPNG(t, filepath.Join(root, "background"), 81, 60)

	if _, err := service.EnqueueRecognize(RecognizeRequest{
		ImagePath:  backgroundImagePath,
		SourceKind: SourceImage,
		SourceID:   "background",
		Language:   defaultLanguage,
		Priority:   JobPriorityBackground,
	}); err != nil {
		t.Fatalf("EnqueueRecognize(background) error = %v", err)
	}
	if _, err := service.EnqueueRecognize(RecognizeRequest{
		ImagePath:  imagePath,
		SourceKind: SourceImage,
		SourceID:   "interactive",
		Language:   defaultLanguage,
		Priority:   JobPriorityInteractive,
	}); err != nil {
		t.Fatalf("EnqueueRecognize(interactive) error = %v", err)
	}

	next := service.nextJob()
	if next == nil {
		t.Fatal("nextJob() = nil, want interactive job")
	}
	if len(next.requests) != 1 || next.requests[0].SourceID != "interactive" {
		t.Fatalf("next job requests = %#v, want interactive first", next.requests)
	}
}

func TestOCRWorkerHelperProcess(t *testing.T) {
	if os.Getenv("RF_OCR_WORKER_HELPER") != "1" {
		return
	}
	if hasArg("--capabilities") {
		supportsRecognize := os.Getenv("RF_OCR_WORKER_RECOGNIZE") == "1"
		message := "ONNX runtime is not connected in this helper."
		if supportsRecognize {
			message = "helper recognition is enabled"
		}
		_ = json.NewEncoder(os.Stdout).Encode(WorkerCapabilities{
			SchemaVersion:     1,
			Name:              "rf-ocr-worker-test-helper",
			Version:           "test",
			ProtocolVersion:   "ocr-jsonl-v1",
			Engine:            "onnxruntime",
			ModelFormats:      []string{"onnx"},
			SupportsRecognize: supportsRecognize,
			Message:           message,
		})
		os.Exit(0)
	}
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for {
		var req workerRequest
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "decode request: %v\n", err)
			os.Exit(2)
		}
		switch req.Method {
		case workerMethodInit:
			_ = encoder.Encode(workerResponse{ID: req.ID, OK: true})
		case workerMethodRecognize:
			if os.Getenv("RF_OCR_WORKER_TILE") == "1" {
				params := helperRecognizeParams(req.Params)
				appendHelperRecognizeLog(params)
				width, height, err := ImageDimensions(params.ImagePath)
				if err != nil {
					_ = encoder.Encode(workerResponse{
						ID: req.ID,
						OK: false,
						Error: &workerError{
							Code:    "image_preprocess_failed",
							Message: err.Error(),
						},
					})
					continue
				}
				originY := helperTileOrigin(params.SourceID)
				blocks := helperTileBlocks(originY, height)
				_ = encoder.Encode(workerResponse{
					ID: req.ID,
					OK: true,
					Result: &Result{
						ID:        fmt.Sprintf("ocr_tile_%d", originY),
						Width:     width,
						Height:    height,
						Blocks:    blocks,
						PlainText: plainTextFromBlocks(blocks),
					},
				})
				continue
			}
			_ = encoder.Encode(workerResponse{
				ID: req.ID,
				OK: true,
				Result: &Result{
					ID:        "ocr_worker_result",
					Width:     80,
					Height:    60,
					Blocks:    []Block{{ID: "b1", Text: "RecordingFreedom"}, {ID: "b2", Text: "文字识别"}},
					PlainText: "RecordingFreedom\n文字识别",
				},
			})
		case workerMethodRelease:
			_ = encoder.Encode(workerResponse{ID: req.ID, OK: true})
			os.Exit(0)
		default:
			_ = encoder.Encode(workerResponse{
				ID: req.ID,
				OK: false,
				Error: &workerError{
					Code:    "unknown_method",
					Message: "unknown method " + req.Method,
				},
			})
		}
	}
}

func helperRecognizeParams(raw any) workerRecognizeParams {
	data, _ := json.Marshal(raw)
	var params workerRecognizeParams
	_ = json.Unmarshal(data, &params)
	return params
}

func appendHelperRecognizeLog(params workerRecognizeParams) {
	logPath := os.Getenv("RF_OCR_WORKER_LOG")
	if logPath == "" {
		return
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "recognize %s %s\n", params.SourceID, params.ImagePath)
}

func helperTileOrigin(sourceID string) int {
	parts := strings.Split(sourceID, "#tile:")
	if len(parts) != 2 {
		return 0
	}
	var y int
	_, _ = fmt.Sscanf(parts[1], "%d:", &y)
	return y
}

func helperTileBlocks(originY int, height int) []Block {
	blocks := make([]Block, 0, 3)
	if originY > 0 {
		blocks = append(blocks, helperTextBlock(
			fmt.Sprintf("top-overlap-%d", originY),
			fmt.Sprintf("overlap-%d", originY),
			10,
		))
	}
	blocks = append(blocks, helperTextBlock(
		fmt.Sprintf("unique-%d", originY),
		fmt.Sprintf("tile-%d", originY),
		40,
	))
	if height >= scrollingOCRTileHeight {
		nextOrigin := originY + height - scrollingOCRTileOverlap
		blocks = append(blocks, helperTextBlock(
			fmt.Sprintf("bottom-overlap-%d", nextOrigin),
			fmt.Sprintf("overlap-%d", nextOrigin),
			float64(height-scrollingOCRTileOverlap+10),
		))
	}
	return blocks
}

func helperTextBlock(id string, text string, y float64) Block {
	return Block{
		ID:         id,
		Text:       text,
		Confidence: 0.92,
		Box: []Point{
			{X: 10, Y: y},
			{X: 110, Y: y},
			{X: 110, Y: y + 30},
			{X: 10, Y: y + 30},
		},
	}
}

func hasArg(target string) bool {
	for _, arg := range os.Args[1:] {
		if arg == target {
			return true
		}
	}
	return false
}

func writeVerifiedTestModel(t *testing.T, root string, modelID string) {
	t.Helper()
	dir := filepath.Join(root, "data", modelRootDir, ocrModelDir, modelID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(model) error = %v", err)
	}
	for _, name := range []string{"det.onnx", "cls.onnx", "rec.onnx", "keys.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	manifest := `{
  "schemaVersion": 1,
  "id": "` + modelID + `",
  "name": "Test OCR Model",
  "channel": "stable",
  "engine": "onnxruntime",
  "language": ["zh", "en"],
  "version": "test",
  "files": [
    {"name": "det.onnx"},
    {"name": "cls.onnx"},
    {"name": "rec.onnx"},
    {"name": "keys.txt"}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
}

func writeTranslationTestResult(t *testing.T, service *Service) {
	t.Helper()
	result := Result{
		ID:          "ocr_translate_fixture",
		SourceKind:  SourceRegionScreenshot,
		SourceID:    "shot-translate",
		ImagePath:   "/tmp/shot-translate.png",
		ImageSHA256: "translate-image-sha",
		ModelID:     defaultActiveModelID(),
		Language:    defaultLanguage,
		Width:       320,
		Height:      180,
		Blocks: []Block{
			{
				ID:         "b1",
				Text:       "RecordingFreedom",
				Confidence: 0.98,
				Box:        []Point{{X: 10, Y: 10}, {X: 160, Y: 10}, {X: 160, Y: 40}, {X: 10, Y: 40}},
				LineIndex:  0,
			},
			{
				ID:         "b2",
				Text:       "文字识别",
				Confidence: 0.96,
				Box:        []Point{{X: 10, Y: 50}, {X: 120, Y: 50}, {X: 120, Y: 80}, {X: 10, Y: 80}},
				LineIndex:  1,
			},
		},
		PlainText: "RecordingFreedom\n文字识别",
		CreatedAt: time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC),
	}
	if err := service.WriteResult(result); err != nil {
		t.Fatalf("WriteResult(translation fixture) error = %v", err)
	}
}

func translationBlockTexts(blocks []TranslationBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		parts = append(parts, block.BlockID+"="+block.Translated)
	}
	return strings.Join(parts, ";")
}

func writeModelPackageZip(t *testing.T, path string, topDir string, modelID string, files map[string]string) {
	t.Helper()
	writeModelPackageZipWithBadSHA(t, path, topDir, modelID, files, "")
}

func downloadableTestModel(modelID string, packageURL string, packageBytes []byte, packageSHA string) ModelManifest {
	if packageSHA == "" {
		sum := sha256.Sum256(packageBytes)
		packageSHA = hex.EncodeToString(sum[:])
	}
	return ModelManifest{
		SchemaVersion: 1,
		ID:            modelID,
		Name:          "Downloadable Test OCR Model",
		Channel:       "latest",
		Engine:        "onnxruntime",
		Language:      []string{"zh", "en"},
		Version:       "test-download",
		Source: ModelSource{
			URL:     "https://example.invalid/" + modelID,
			License: "Apache-2.0",
		},
		Package: ModelPackageSource{
			URL:    packageURL,
			Bytes:  int64(len(packageBytes)),
			SHA256: packageSHA,
		},
		Files: []ModelFile{
			{Name: "det.onnx"},
			{Name: "cls.onnx"},
			{Name: "rec.onnx"},
			{Name: "keys.txt"},
		},
	}
}

func waitModelDownloadStatus(t *testing.T, service *Service, modelID string, status string) ModelDownloadSnapshot {
	t.Helper()
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case event := <-service.ModelDownloadEvents():
			if event.Snapshot.ModelID == modelID && event.Snapshot.Status == status {
				return event.Snapshot
			}
		case <-ticker.C:
			for _, snapshot := range service.ModelDownloads() {
				if snapshot.ModelID == modelID && snapshot.Status == status {
					return snapshot
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for model download %s status %s; snapshots=%#v", modelID, status, service.ModelDownloads())
		}
	}
}

func waitNoModelDownloadStaging(t *testing.T, service *Service) {
	t.Helper()
	root, err := service.modelRoot()
	if err != nil {
		t.Fatalf("modelRoot() error = %v", err)
	}
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		matches, err := filepath.Glob(filepath.Join(root, modelDownloadStagingPrefix+"*"))
		if err != nil {
			t.Fatalf("Glob(download staging) error = %v", err)
		}
		if len(matches) == 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("model download staging files were not cleaned: %#v", matches)
		}
	}
}

func findModelInfo(models []ModelInfo, modelID string) *ModelInfo {
	for index := range models {
		if models[index].ID == modelID {
			return &models[index]
		}
	}
	return nil
}

func writeModelPackageZipWithBadSHA(t *testing.T, path string, topDir string, modelID string, files map[string]string, badSHA string) {
	t.Helper()
	writeModelPackageZipWithOrientationAndBadSHA(t, path, topDir, modelID, files, TextlineOrientationCLS, badSHA)
}

func writeModelPackageZipWithOrientation(t *testing.T, path string, topDir string, modelID string, files map[string]string, orientationMode string) {
	t.Helper()
	writeModelPackageZipWithOrientationAndBadSHA(t, path, topDir, modelID, files, orientationMode, "")
}

func writeModelPackageZipWithOrientationAndBadSHA(t *testing.T, path string, topDir string, modelID string, files map[string]string, orientationMode string, badSHA string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(zip dir) error = %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(zip) error = %v", err)
	}
	zipWriter := zip.NewWriter(file)
	base := strings.Trim(strings.ReplaceAll(topDir, "\\", "/"), "/")
	entryName := func(name string) string {
		if base == "" {
			return name
		}
		return base + "/" + name
	}
	manifest := modelPackageManifestJSON(modelID, files, orientationMode, badSHA)
	writeZipEntry(t, zipWriter, entryName("manifest.json"), manifest)
	for name, content := range files {
		writeZipEntry(t, zipWriter, entryName(name), content)
	}
	writeZipEntryBytes(t, zipWriter, entryName("smoke.png"), testPNGBytes(t, 64, 32))
	writeZipEntry(t, zipWriter, entryName("smoke.expected.json"), `{"mustContain":["RecordingFreedom","文字识别"]}`)
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zip Close() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file Close() error = %v", err)
	}
}

func writeZipEntry(t *testing.T, zipWriter *zip.Writer, name string, content string) {
	t.Helper()
	writeZipEntryBytes(t, zipWriter, name, []byte(content))
}

func writeZipEntryBytes(t *testing.T, zipWriter *zip.Writer, name string, content []byte) {
	t.Helper()
	writer, err := zipWriter.Create(name)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", name, err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("Write(%s) error = %v", name, err)
	}
}

func modelPackageManifestJSON(modelID string, files map[string]string, orientationMode string, badSHA string) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]string, 0, len(names))
	for _, name := range names {
		content := files[name]
		sum := sha256String(content)
		if badSHA != "" && name == "rec.onnx" {
			sum = badSHA
		}
		entries = append(entries, fmt.Sprintf(`    {"name": %q, "sha256": %q, "bytes": %d}`, name, sum, len([]byte(content))))
	}
	return fmt.Sprintf(`{
  "schemaVersion": 1,
  "id": %q,
  "name": "Test OCR Model Package",
  "channel": "latest",
  "engine": "onnxruntime",
  "language": ["zh", "en"],
  "version": "test-package",
  "textlineOrientation": {"mode": %q},
  "source": {
    "url": "https://example.invalid/%s",
    "license": "Apache-2.0"
  },
  "files": [
%s
  ],
  "smoke": {
    "image": "smoke.png",
    "expected": "smoke.expected.json",
    "mustContain": ["RecordingFreedom", "文字识别"],
    "maxDurationMs": 3000
  }
}`, modelID, orientationMode, modelID, strings.Join(entries, ",\n"))
}

func sha256String(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func writeTestPNG(t *testing.T, root string, width int, height int) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(test png root) error = %v", err)
	}
	path := filepath.Join(root, "ocr-test.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(test png) error = %v", err)
	}
	defer file.Close()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return path
}

func testPNGBytes(t *testing.T, width int, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode(bytes) error = %v", err)
	}
	return buf.Bytes()
}
