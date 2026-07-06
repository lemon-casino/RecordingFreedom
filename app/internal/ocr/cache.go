package ocr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var cachePartPattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (s *Service) RecognizeImage(req RecognizeRequest) (Result, error) {
	req = normalizeRecognizeRequest(req)
	if strings.TrimSpace(req.ImagePath) == "" {
		return Result{}, errors.New("OCR image path is required")
	}
	imagePath, err := filepath.Abs(req.ImagePath)
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(imagePath)
	if err != nil {
		return Result{}, err
	}
	if info.IsDir() {
		return Result{}, fmt.Errorf("OCR image path %q is a directory", imagePath)
	}
	sum, err := fileSHA256(imagePath)
	if err != nil {
		return Result{}, err
	}
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		state, err := s.LoadState()
		if err != nil {
			return Result{}, err
		}
		modelID = state.ActiveModelID
	}
	if !req.Force {
		cached, ok, err := s.readCachedResult(sum, modelID, req.Language)
		if err != nil {
			return Result{}, err
		}
		if ok {
			req.ImagePath = imagePath
			req.ModelID = modelID
			target := cloneResultForRequest(&cached, req)
			target.ImageSHA256 = sum
			if strings.TrimSpace(target.ModelID) == "" {
				target.ModelID = modelID
			}
			if strings.TrimSpace(target.Language) == "" {
				target.Language = req.Language
			}
			if err := s.WriteResult(target); err != nil {
				return Result{}, err
			}
			return target, nil
		}
	}
	models, err := s.ListModels()
	if err != nil {
		return Result{}, err
	}
	var active ModelInfo
	for _, model := range models {
		if model.ID == modelID {
			active = model
			break
		}
	}
	if active.ID == "" {
		return Result{}, fmt.Errorf("unknown OCR model %q", modelID)
	}
	if !active.Installed || !active.Verified {
		return Result{}, fmt.Errorf("OCR model %q is not installed or verified", modelID)
	}
	if _, err := s.requireRecognizeWorker(); err != nil {
		return Result{}, err
	}
	result, err := s.runRecognizePipeline(req, imagePath, sum, active)
	if err != nil {
		return Result{}, err
	}
	result = normalizeWorkerResult(result, req, imagePath, sum, active.ID)
	if err := s.WriteResult(result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *Service) runRecognizePipeline(req RecognizeRequest, imagePath string, imageSHA256 string, model ModelInfo) (Result, error) {
	if shouldTileScrollingImage(req, imagePath) {
		return s.recognizeScrollingImageTiles(req, imagePath, imageSHA256, model)
	}
	return s.runWorkerRecognize(req, imageSHA256, model)
}

func (s *Service) ReadResult(resultID string) (Result, error) {
	resultID = strings.TrimSpace(resultID)
	if resultID == "" {
		return Result{}, errors.New("OCR result id is required")
	}
	dir, err := s.resultsDir()
	if err != nil {
		return Result{}, err
	}
	path := filepath.Join(dir, safeCachePart(resultID)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *Service) WriteResult(result Result) error {
	if strings.TrimSpace(result.ID) == "" {
		return errors.New("OCR result id is required")
	}
	if strings.TrimSpace(result.ImageSHA256) == "" || strings.TrimSpace(result.ModelID) == "" || strings.TrimSpace(result.Language) == "" {
		return errors.New("OCR result cache key is incomplete")
	}
	dir, err := s.resultsDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, safeCachePart(result.ID)+".json")
	if err := writeJSONFile(path, result); err != nil {
		return err
	}
	cachePath, err := s.cachePath(result.ImageSHA256, result.ModelID, result.Language)
	if err != nil {
		return err
	}
	return writeJSONFile(cachePath, result)
}

func normalizeRecognizeRequest(req RecognizeRequest) RecognizeRequest {
	if req.SourceKind == "" {
		req.SourceKind = SourceImage
	}
	if strings.TrimSpace(req.Language) == "" {
		req.Language = defaultLanguage
	}
	if strings.TrimSpace(req.Priority) == "" {
		req.Priority = JobPriorityNormal
	}
	return req
}

func normalizeWorkerResult(result Result, req RecognizeRequest, imagePath string, imageSHA256 string, modelID string) Result {
	if strings.TrimSpace(result.ID) == "" {
		result.ID = "ocr_" + requestIDNow()
	}
	result.SourceKind = req.SourceKind
	result.SourceID = req.SourceID
	result.ImagePath = imagePath
	result.ImageSHA256 = imageSHA256
	result.ModelID = modelID
	result.Language = req.Language
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	}
	if result.Width <= 0 || result.Height <= 0 {
		if width, height, err := ImageDimensions(imagePath); err == nil {
			result.Width = width
			result.Height = height
		}
	}
	if result.PlainText == "" && len(result.Blocks) > 0 {
		parts := make([]string, 0, len(result.Blocks))
		for _, block := range result.Blocks {
			text := strings.TrimSpace(block.Text)
			if text != "" {
				parts = append(parts, text)
			}
		}
		result.PlainText = strings.Join(parts, "\n")
	}
	return result
}

func (s *Service) readCachedResult(imageSHA256 string, modelID string, language string) (Result, bool, error) {
	path, err := s.cachePath(imageSHA256, modelID, language)
	if err != nil {
		return Result{}, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, false, nil
		}
		return Result{}, false, err
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return Result{}, false, err
	}
	return result, true, nil
}

func (s *Service) cachePath(imageSHA256 string, modelID string, language string) (string, error) {
	dir, err := s.cacheDir()
	if err != nil {
		return "", err
	}
	name := safeCachePart(imageSHA256) + "." + safeCachePart(modelID) + "." + safeCachePart(language) + ".json"
	return filepath.Join(dir, name), nil
}

func (s *Service) cacheDir() (string, error) {
	root, err := s.ocrDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, cacheDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *Service) resultsDir() (string, error) {
	root, err := s.ocrDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, resultsDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *Service) translationsDir() (string, error) {
	root, err := s.ocrDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, translationsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func safeCachePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty"
	}
	return cachePartPattern.ReplaceAllString(value, "_")
}

func ImageDimensions(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func HashReader(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
