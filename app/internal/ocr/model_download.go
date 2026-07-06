package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const modelDownloadStagingPrefix = ".download-"

type modelDownloadState struct {
	snapshot ModelDownloadSnapshot
}

func (s *Service) StartModelDownload(modelID string) (ModelDownloadSnapshot, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ModelDownloadSnapshot{}, errors.New("OCR model id is required")
	}
	model, ok := s.findModel(modelID)
	if !ok {
		return ModelDownloadSnapshot{}, fmt.Errorf("unknown OCR model %q", modelID)
	}
	if !modelPackageDownloadAvailable(model.Package) {
		if !modelSourceDownloadAvailable(model) {
			return ModelDownloadSnapshot{}, fmt.Errorf("OCR model %q does not have a verified RecordingFreedom package or pinned source file download", modelID)
		}
	}
	root, err := s.modelRoot()
	if err != nil {
		return ModelDownloadSnapshot{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return ModelDownloadSnapshot{}, err
	}

	s.modelDownloadMu.Lock()
	if existing := s.modelDownloads[modelID]; existing != nil && (existing.snapshot.Status == ModelDownloadQueued || existing.snapshot.Status == ModelDownloadRunning) {
		snapshot := existing.snapshot
		s.modelDownloadMu.Unlock()
		return snapshot, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	now := s.now().UTC()
	snapshot := ModelDownloadSnapshot{
		ID:         fmt.Sprintf("ocr-model-download-%s-%d", modelID, now.UnixNano()),
		ModelID:    modelID,
		Status:     ModelDownloadQueued,
		TotalBytes: modelDownloadBytes(model),
		StartedAt:  now,
		UpdatedAt:  now,
	}
	s.modelDownloads[modelID] = &modelDownloadState{snapshot: snapshot}
	s.modelDownloadCancels[modelID] = cancel
	s.modelDownloadMu.Unlock()
	s.emitModelDownload(snapshot)

	go s.runModelDownload(ctx, root, model)
	return snapshot, nil
}

func (s *Service) CancelModelDownload(modelID string) (ModelDownloadSnapshot, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ModelDownloadSnapshot{}, errors.New("OCR model id is required")
	}
	s.modelDownloadMu.Lock()
	cancel := s.modelDownloadCancels[modelID]
	state := s.modelDownloads[modelID]
	if cancel == nil || state == nil || (state.snapshot.Status != ModelDownloadQueued && state.snapshot.Status != ModelDownloadRunning) {
		snapshot := ModelDownloadSnapshot{
			ModelID:   modelID,
			Status:    ModelDownloadCancelled,
			UpdatedAt: s.now().UTC(),
		}
		if state != nil {
			snapshot = state.snapshot
		}
		s.modelDownloadMu.Unlock()
		return snapshot, nil
	}
	cancel()
	snapshot := state.snapshot
	s.modelDownloadMu.Unlock()
	return snapshot, nil
}

func (s *Service) ModelDownloads() []ModelDownloadSnapshot {
	s.modelDownloadMu.Lock()
	defer s.modelDownloadMu.Unlock()
	result := make([]ModelDownloadSnapshot, 0, len(s.modelDownloads))
	for _, state := range s.modelDownloads {
		result = append(result, state.snapshot)
	}
	return result
}

func (s *Service) ModelDownloadEvents() <-chan ModelDownloadEvent {
	return s.modelDownloadEvents
}

func (s *Service) runModelDownload(ctx context.Context, root string, model ModelManifest) {
	modelID := model.ID
	zipPath := ""
	sourceDir := ""
	defer func() {
		if zipPath != "" {
			_ = os.Remove(zipPath)
		}
		if sourceDir != "" {
			_ = os.RemoveAll(sourceDir)
		}
		s.modelDownloadMu.Lock()
		delete(s.modelDownloadCancels, modelID)
		s.modelDownloadMu.Unlock()
	}()

	if err := s.setModelDownloadStatus(modelID, ModelDownloadRunning, "", nil); err != nil {
		return
	}
	var info ModelInfo
	var err error
	if modelPackageDownloadAvailable(model.Package) {
		zipPath = filepath.Join(root, fmt.Sprintf("%s%s-%d.zip", modelDownloadStagingPrefix, modelID, s.now().UnixNano()))
		if err := s.downloadModelPackageZip(ctx, model, zipPath); err != nil {
			if errors.Is(err, context.Canceled) {
				_ = s.setModelDownloadStatus(modelID, ModelDownloadCancelled, "", nil)
				return
			}
			_ = s.setModelDownloadStatus(modelID, ModelDownloadFailed, err.Error(), nil)
			return
		}
		info, err = s.InstallModelPackage(zipPath)
	} else {
		sourceDir = filepath.Join(root, fmt.Sprintf("%s%s-%d", modelDownloadStagingPrefix, modelID, s.now().UnixNano()))
		info, err = s.downloadAndInstallModelSourceFiles(ctx, model, sourceDir)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			_ = s.setModelDownloadStatus(modelID, ModelDownloadCancelled, "", nil)
			return
		}
		_ = s.setModelDownloadStatus(modelID, ModelDownloadFailed, err.Error(), nil)
		return
	}
	_ = s.setModelDownloadStatus(modelID, ModelDownloadInstalled, "", &info)
}

func (s *Service) downloadModelPackageZip(ctx context.Context, model ModelManifest, target string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, model.Package.URL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "RecordingFreedom-ocr-model-downloader/1")
	client := &http.Client{Timeout: 15 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("download %s failed with HTTP %s", model.Package.URL, response.Status)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	hash := sha256.New()
	buffer := make([]byte, 128*1024)
	var downloaded int64
	for {
		if err := ctx.Err(); err != nil {
			_ = out.Close()
			return err
		}
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if _, err := out.Write(chunk); err != nil {
				_ = out.Close()
				return err
			}
			if _, err := hash.Write(chunk); err != nil {
				_ = out.Close()
				return err
			}
			downloaded += int64(n)
			s.updateModelDownloadProgress(model.ID, downloaded, model.Package.Bytes)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			_ = out.Close()
			return readErr
		}
	}
	if err := out.Close(); err != nil {
		return err
	}
	if downloaded != model.Package.Bytes {
		return fmt.Errorf("downloaded OCR model %s bytes = %d, want %d", model.ID, downloaded, model.Package.Bytes)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, model.Package.SHA256) {
		return fmt.Errorf("downloaded OCR model %s sha256 = %s, want %s", model.ID, actual, model.Package.SHA256)
	}
	return nil
}

func (s *Service) downloadAndInstallModelSourceFiles(ctx context.Context, model ModelManifest, stagingDir string) (ModelInfo, error) {
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return ModelInfo{}, err
	}
	downloaded := int64(0)
	for _, file := range model.Files {
		if !isRequiredModelFile(model, file.Name) {
			continue
		}
		if file.Generate != nil {
			n, err := s.downloadGenerateAndVerifyModelFile(ctx, stagingDir, model.ID, file, downloaded, modelDownloadBytes(model))
			downloaded += n
			if err != nil {
				return ModelInfo{}, err
			}
			continue
		}
		n, err := s.downloadAndVerifyModelFile(ctx, stagingDir, model.ID, file, file.Bytes, file.SHA256, downloaded, modelDownloadBytes(model))
		downloaded += n
		if err != nil {
			return ModelInfo{}, err
		}
	}
	installManifest := cloneModelManifest(model)
	installManifest.Package = ModelPackageSource{}
	installManifest.Smoke = ModelSmoke{}
	data, err := json.MarshalIndent(installManifest, "", "  ")
	if err != nil {
		return ModelInfo{}, err
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "manifest.json"), append(data, '\n'), 0o644); err != nil {
		return ModelInfo{}, err
	}
	return s.InstallModelPackage(stagingDir)
}

func (s *Service) downloadGenerateAndVerifyModelFile(ctx context.Context, stagingDir string, modelID string, file ModelFile, downloadedBefore int64, total int64) (int64, error) {
	if file.Generate == nil {
		return 0, errors.New("generated OCR model file source is required")
	}
	if file.Generate.Type != GeneratedPaddleOCRCharacterDictKeys {
		return 0, fmt.Errorf("unsupported generated OCR model file type %q", file.Generate.Type)
	}
	sourceName := file.Name + ".source"
	source := file
	source.Name = sourceName
	source.Bytes = file.Generate.SourceBytes
	source.SHA256 = file.Generate.SourceSHA256
	source.Generate = nil
	n, err := s.downloadAndVerifyModelFile(ctx, stagingDir, modelID, source, source.Bytes, source.SHA256, downloadedBefore, total)
	if err != nil {
		return n, err
	}
	sourceData, err := os.ReadFile(filepath.Join(stagingDir, sourceName))
	if err != nil {
		return n, err
	}
	generated, err := generatePaddleOCRCharacterDictKeys(sourceData)
	if err != nil {
		return n, err
	}
	if int64(len(generated)) != file.Bytes {
		return n, fmt.Errorf("generated OCR model file %s bytes = %d, want %d", file.Name, len(generated), file.Bytes)
	}
	actual := sha256.Sum256(generated)
	if !strings.EqualFold(hex.EncodeToString(actual[:]), file.SHA256) {
		return n, fmt.Errorf("generated OCR model file %s sha256 = %s, want %s", file.Name, hex.EncodeToString(actual[:]), file.SHA256)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, file.Name), generated, 0o644); err != nil {
		return n, err
	}
	_ = os.Remove(filepath.Join(stagingDir, sourceName))
	return n, nil
}

func (s *Service) downloadAndVerifyModelFile(ctx context.Context, stagingDir string, modelID string, file ModelFile, expectedBytes int64, expectedSHA256 string, downloadedBefore int64, total int64) (int64, error) {
	if !safeModelPackageRelativePath(file.Name) || strings.ContainsAny(file.Name, `/\`) {
		return 0, fmt.Errorf("unsafe OCR model file name %q", file.Name)
	}
	if strings.TrimSpace(file.DownloadURL) == "" {
		return 0, fmt.Errorf("OCR model file %s does not declare downloadUrl", file.Name)
	}
	if expectedBytes <= 0 || len(strings.TrimSpace(expectedSHA256)) != 64 {
		return 0, fmt.Errorf("OCR model file %s does not declare verified bytes and sha256", file.Name)
	}
	target := filepath.Join(stagingDir, file.Name)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, file.DownloadURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("User-Agent", "RecordingFreedom-ocr-model-downloader/1")
	client := &http.Client{Timeout: 15 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, fmt.Errorf("download %s failed with HTTP %s", file.DownloadURL, response.Status)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	out, err := os.Create(target)
	if err != nil {
		return 0, err
	}
	hash := sha256.New()
	buffer := make([]byte, 128*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			_ = out.Close()
			return written, err
		}
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if _, err := out.Write(chunk); err != nil {
				_ = out.Close()
				return written, err
			}
			if _, err := hash.Write(chunk); err != nil {
				_ = out.Close()
				return written, err
			}
			written += int64(n)
			s.updateModelDownloadProgress(modelID, downloadedBefore+written, total)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			_ = out.Close()
			return written, readErr
		}
	}
	if err := out.Close(); err != nil {
		return written, err
	}
	if written != expectedBytes {
		return written, fmt.Errorf("downloaded OCR model file %s bytes = %d, want %d", file.Name, written, expectedBytes)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expectedSHA256) {
		return written, fmt.Errorf("downloaded OCR model file %s sha256 = %s, want %s", file.Name, actual, expectedSHA256)
	}
	return written, nil
}

func (s *Service) updateModelDownloadProgress(modelID string, downloaded int64, total int64) {
	s.modelDownloadMu.Lock()
	state := s.modelDownloads[modelID]
	if state == nil {
		s.modelDownloadMu.Unlock()
		return
	}
	state.snapshot.DownloadedBytes = downloaded
	state.snapshot.TotalBytes = total
	state.snapshot.Percent = modelDownloadPercent(downloaded, total)
	state.snapshot.UpdatedAt = s.now().UTC()
	snapshot := state.snapshot
	s.modelDownloadMu.Unlock()
	s.emitModelDownload(snapshot)
}

func (s *Service) setModelDownloadStatus(modelID string, status string, message string, model *ModelInfo) error {
	s.modelDownloadMu.Lock()
	state := s.modelDownloads[modelID]
	if state == nil {
		s.modelDownloadMu.Unlock()
		return fmt.Errorf("OCR model download %q was not found", modelID)
	}
	state.snapshot.Status = status
	state.snapshot.Error = strings.TrimSpace(message)
	if model != nil {
		state.snapshot.Model = model
	}
	if status == ModelDownloadInstalled {
		state.snapshot.DownloadedBytes = state.snapshot.TotalBytes
		state.snapshot.Percent = 100
	}
	state.snapshot.UpdatedAt = s.now().UTC()
	snapshot := state.snapshot
	s.modelDownloadMu.Unlock()
	s.emitModelDownload(snapshot)
	return nil
}

func (s *Service) emitModelDownload(snapshot ModelDownloadSnapshot) {
	if s == nil || s.modelDownloadEvents == nil {
		return
	}
	select {
	case s.modelDownloadEvents <- ModelDownloadEvent{Snapshot: snapshot}:
	default:
	}
}

func modelPackageDownloadAvailable(source ModelPackageSource) bool {
	return strings.TrimSpace(source.URL) != "" && source.Bytes > 0 && len(strings.TrimSpace(source.SHA256)) == 64
}

func modelSourceDownloadAvailable(model ModelManifest) bool {
	required := make(map[string]bool)
	for _, name := range RequiredModelFileNames(model) {
		required[name] = false
	}
	for _, file := range model.Files {
		if _, ok := required[file.Name]; !ok {
			continue
		}
		if strings.TrimSpace(file.DownloadURL) == "" || file.Bytes <= 0 || len(strings.TrimSpace(file.SHA256)) != 64 {
			return false
		}
		if file.Generate != nil {
			if file.Generate.Type != GeneratedPaddleOCRCharacterDictKeys || file.Generate.SourceBytes <= 0 || len(strings.TrimSpace(file.Generate.SourceSHA256)) != 64 {
				return false
			}
		}
		required[file.Name] = true
	}
	for _, found := range required {
		if !found {
			return false
		}
	}
	return len(required) > 0
}

func modelDownloadBytes(model ModelManifest) int64 {
	if modelPackageDownloadAvailable(model.Package) {
		return model.Package.Bytes
	}
	var total int64
	for _, file := range model.Files {
		if !isRequiredModelFile(model, file.Name) {
			continue
		}
		if file.Generate != nil && file.Generate.SourceBytes > 0 {
			total += file.Generate.SourceBytes
			continue
		}
		total += file.Bytes
	}
	return total
}

func isRequiredModelFile(model ModelManifest, name string) bool {
	for _, required := range RequiredModelFileNames(model) {
		if name == required {
			return true
		}
	}
	return false
}

func modelDownloadPercent(downloaded int64, total int64) float64 {
	if total <= 0 || downloaded <= 0 {
		return 0
	}
	percent := (float64(downloaded) / float64(total)) * 100
	if percent > 100 {
		return 100
	}
	return percent
}
