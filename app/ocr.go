package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

type OcrJobEvent struct {
	Event      string         `json:"event,omitempty"`
	JobID      string         `json:"jobId"`
	SourceKind ocr.SourceKind `json:"sourceKind"`
	SourceID   string         `json:"sourceId"`
	Status     string         `json:"status"`
	CacheKey   string         `json:"cacheKey,omitempty"`
	Merged     bool           `json:"merged,omitempty"`
	Error      string         `json:"error,omitempty"`
	Result     *ocr.Result    `json:"result,omitempty"`
}

type OcrModelEvent struct {
	ModelID string         `json:"modelId"`
	Status  string         `json:"status"`
	Error   string         `json:"error,omitempty"`
	Model   *ocr.ModelInfo `json:"model,omitempty"`
}

func (s *RecordingFreedomService) ListOcrModels() ([]ocr.ModelInfo, error) {
	if s.ocr == nil {
		return nil, errors.New("OCR service is not initialized")
	}
	return s.ocr.ListModels()
}

func (s *RecordingFreedomService) InstallOcrModel(modelID string) (ocr.ModelInfo, error) {
	if s.ocr == nil {
		return ocr.ModelInfo{}, errors.New("OCR service is not initialized")
	}
	info, err := s.ocr.InstallModel(modelID)
	if err != nil {
		s.emitOCRModelEvent("ocr.model.failed", OcrModelEvent{ModelID: modelID, Status: "failed", Error: err.Error()})
	} else {
		s.emitOCRModelEvent("ocr.model.installed", OcrModelEvent{ModelID: info.ID, Status: "installed", Model: &info})
	}
	s.emitOCRStatus()
	return info, err
}

func (s *RecordingFreedomService) InstallOcrModelPackage(packagePath string) (ocr.ModelInfo, error) {
	if s.ocr == nil {
		return ocr.ModelInfo{}, errors.New("OCR service is not initialized")
	}
	info, err := s.ocr.InstallModelPackage(packagePath)
	if err != nil {
		s.emitOCRModelEvent("ocr.model.failed", OcrModelEvent{Status: "failed", Error: err.Error()})
		s.emitOCRStatus()
		return ocr.ModelInfo{}, err
	}
	s.emitOCRModelEvent("ocr.model.installed", OcrModelEvent{ModelID: info.ID, Status: "installed", Model: &info})
	s.emitOCRStatus()
	return info, nil
}

func (s *RecordingFreedomService) StartOcrModelDownload(modelID string) (ocr.ModelDownloadSnapshot, error) {
	if s.ocr == nil {
		return ocr.ModelDownloadSnapshot{}, errors.New("OCR service is not initialized")
	}
	snapshot, err := s.ocr.StartModelDownload(modelID)
	if err != nil {
		s.emitOCRModelEvent("ocr.model.failed", OcrModelEvent{ModelID: modelID, Status: "failed", Error: err.Error()})
		return ocr.ModelDownloadSnapshot{}, err
	}
	return snapshot, nil
}

func (s *RecordingFreedomService) CancelOcrModelDownload(modelID string) (ocr.ModelDownloadSnapshot, error) {
	if s.ocr == nil {
		return ocr.ModelDownloadSnapshot{}, errors.New("OCR service is not initialized")
	}
	return s.ocr.CancelModelDownload(modelID)
}

func (s *RecordingFreedomService) GetOcrModelDownloads() ([]ocr.ModelDownloadSnapshot, error) {
	if s.ocr == nil {
		return nil, errors.New("OCR service is not initialized")
	}
	return s.ocr.ModelDownloads(), nil
}

func (s *RecordingFreedomService) RefreshOcrModelCatalog(catalogURL string) (ocr.Status, error) {
	if s.ocr == nil {
		return ocr.Status{}, errors.New("OCR service is not initialized")
	}
	status, err := s.ocr.RefreshModelCatalog(context.Background(), catalogURL)
	s.emitOCRStatus()
	return status, err
}

func (s *RecordingFreedomService) RemoveOcrModel(modelID string) (ocr.Status, error) {
	if s.ocr == nil {
		return ocr.Status{}, errors.New("OCR service is not initialized")
	}
	status, err := s.ocr.RemoveModel(modelID)
	s.emitOCRStatus()
	return status, err
}

func (s *RecordingFreedomService) SetActiveOcrModel(modelID string) (ocr.Status, error) {
	if s.ocr == nil {
		return ocr.Status{}, errors.New("OCR service is not initialized")
	}
	status, err := s.ocr.SetActiveModel(modelID)
	s.emitOCRStatus()
	return status, err
}

func (s *RecordingFreedomService) GetOcrStatus() (ocr.Status, error) {
	if s.ocr == nil {
		return ocr.Status{}, errors.New("OCR service is not initialized")
	}
	return s.ocr.Status()
}

func (s *RecordingFreedomService) RecognizeImage(req ocr.RecognizeRequest) (ocr.Result, error) {
	if s.ocr == nil {
		return ocr.Result{}, errors.New("OCR service is not initialized")
	}
	s.logOCRRecognizeRequest("recognize-request", req)
	result, err := s.ocr.RecognizeImage(req)
	if err != nil {
		s.logEvent("ocr", "recognize-failed", map[string]string{
			"sourceKind": string(req.SourceKind),
			"sourceId":   req.SourceID,
			"language":   req.Language,
			"modelId":    req.ModelID,
		})
		s.emitOCRJobEvent("ocr.job.failed", OcrJobEvent{
			JobID:      "ocr-" + fmt.Sprint(time.Now().UnixNano()),
			SourceKind: req.SourceKind,
			SourceID:   req.SourceID,
			Status:     ocr.ResultStatusFailed,
			Error:      err.Error(),
		})
		return ocr.Result{}, err
	}
	s.logEvent("ocr", "recognize-ready", map[string]string{
		"resultId":   result.ID,
		"sourceKind": string(result.SourceKind),
		"sourceId":   result.SourceID,
		"language":   result.Language,
		"modelId":    result.ModelID,
		"blockCount": fmt.Sprint(len(result.Blocks)),
	})
	s.emitOCRJobEvent("ocr.job.finished", OcrJobEvent{
		JobID:      result.ID,
		SourceKind: result.SourceKind,
		SourceID:   result.SourceID,
		Status:     ocr.ResultStatusReady,
		Result:     &result,
	})
	return result, nil
}

func (s *RecordingFreedomService) QueueRecognizeImage(req ocr.RecognizeRequest) (ocr.JobSnapshot, error) {
	if s.ocr == nil {
		return ocr.JobSnapshot{}, errors.New("OCR service is not initialized")
	}
	s.logOCRRecognizeRequest("queue-request", req)
	snapshot, err := s.ocr.EnqueueRecognize(req)
	if err != nil {
		s.logEvent("ocr", "queue-failed", map[string]string{
			"sourceKind": string(req.SourceKind),
			"sourceId":   req.SourceID,
			"language":   req.Language,
			"modelId":    req.ModelID,
			"priority":   req.Priority,
		})
		return ocr.JobSnapshot{}, err
	}
	s.logEvent("ocr", "queue-accepted", map[string]string{
		"jobId":      snapshot.JobID,
		"sourceKind": string(snapshot.Request.SourceKind),
		"sourceId":   snapshot.Request.SourceID,
		"language":   snapshot.Request.Language,
		"modelId":    snapshot.Request.ModelID,
		"priority":   snapshot.Request.Priority,
		"status":     snapshot.Status,
		"merged":     fmt.Sprint(snapshot.Merged),
	})
	return snapshot, nil
}

func (s *RecordingFreedomService) QueueRecognizeScreenshot(itemID string) (ocr.JobSnapshot, error) {
	item, err := s.screenshotItemByID(itemID)
	if err != nil {
		return ocr.JobSnapshot{}, err
	}
	req := screenshotOCRRequest(item, ocr.JobPriorityInteractive)
	_ = s.patchScreenshotOCRState(item.ID, ocr.ResultStatusQueued, "", "", req.Language, "")
	snapshot, err := s.QueueRecognizeImage(req)
	if err != nil {
		_ = s.patchScreenshotOCRState(item.ID, ocr.ResultStatusFailed, "", "", req.Language, err.Error())
		return ocr.JobSnapshot{}, err
	}
	return snapshot, nil
}

func (s *RecordingFreedomService) queueScreenshotOCRAfterSave(item ScreenshotItem) {
	if s == nil || s.ocr == nil || s.settings == nil || strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Path) == "" {
		return
	}
	go func() {
		current, err := s.loadSettingsForMutation()
		if err != nil {
			s.logEvent("ocr", "auto-queue-settings-failed", map[string]string{
				"screenshotId": item.ID,
				"error":        err.Error(),
			})
			return
		}
		if !current.OCR.AutoRecognizeScreenshots {
			return
		}
		req := screenshotOCRRequest(item, ocr.JobPriorityNormal)
		if err := s.patchScreenshotOCRState(item.ID, ocr.ResultStatusQueued, "", "", req.Language, ""); err != nil {
			s.logEvent("ocr", "auto-queue-state-failed", map[string]string{
				"screenshotId": item.ID,
				"sourceKind":   string(req.SourceKind),
				"error":        err.Error(),
			})
		}
		if _, err := s.QueueRecognizeImage(req); err != nil {
			_ = s.patchScreenshotOCRState(item.ID, ocr.ResultStatusFailed, "", "", req.Language, err.Error())
			s.logEvent("ocr", "auto-queue-failed", map[string]string{
				"screenshotId": item.ID,
				"sourceKind":   string(req.SourceKind),
				"error":        err.Error(),
			})
		}
	}()
}

func (s *RecordingFreedomService) QueueRecognizePinnedScreenshot(itemID string) (ocr.JobSnapshot, error) {
	item, err := s.screenshotItemByID(itemID)
	if err != nil {
		return ocr.JobSnapshot{}, err
	}
	req := ocr.RecognizeRequest{
		ImagePath:  item.Path,
		SourceKind: ocr.SourcePinnedScreenshot,
		SourceID:   item.ID,
		Language:   "zh-en",
		Priority:   ocr.JobPriorityInteractive,
	}
	_ = s.patchScreenshotOCRState(item.ID, ocr.ResultStatusQueued, "", "", req.Language, "")
	snapshot, err := s.QueueRecognizeImage(req)
	if err != nil {
		_ = s.patchScreenshotOCRState(item.ID, ocr.ResultStatusFailed, "", "", req.Language, err.Error())
		return ocr.JobSnapshot{}, err
	}
	return snapshot, nil
}

func (s *RecordingFreedomService) QueueRecognizeWhiteboard(req ocr.WhiteboardRequest) (ocr.JobSnapshot, error) {
	recognizeReq, err := whiteboardOCRRequest(req)
	if err != nil {
		return ocr.JobSnapshot{}, err
	}
	_ = s.patchScreenshotOCRState(recognizeReq.SourceID, ocr.ResultStatusQueued, "", "", recognizeReq.Language, "")
	snapshot, err := s.QueueRecognizeImage(recognizeReq)
	if err != nil {
		_ = s.patchScreenshotOCRState(recognizeReq.SourceID, ocr.ResultStatusFailed, "", "", recognizeReq.Language, err.Error())
		return ocr.JobSnapshot{}, err
	}
	return snapshot, nil
}

func (s *RecordingFreedomService) RecognizeScreenshot(itemID string) (ocr.Result, error) {
	item, err := s.screenshotItemByID(itemID)
	if err != nil {
		return ocr.Result{}, err
	}
	req := screenshotOCRRequest(item, ocr.JobPriorityInteractive)
	result, err := s.RecognizeImage(req)
	if err != nil {
		_ = s.patchScreenshotOCRState(item.ID, "failed", "", "", req.Language, err.Error())
		return ocr.Result{}, err
	}
	_ = s.patchScreenshotOCRState(item.ID, "ready", result.ID, result.ModelID, result.Language, "")
	return result, nil
}

func (s *RecordingFreedomService) RecognizePinnedScreenshot(itemID string) (ocr.Result, error) {
	item, err := s.screenshotItemByID(itemID)
	if err != nil {
		return ocr.Result{}, err
	}
	return s.RecognizeImage(ocr.RecognizeRequest{
		ImagePath:  item.Path,
		SourceKind: ocr.SourcePinnedScreenshot,
		SourceID:   item.ID,
		Language:   "zh-en",
		Priority:   ocr.JobPriorityInteractive,
	})
}

func (s *RecordingFreedomService) RecognizeWhiteboard(req ocr.WhiteboardRequest) (ocr.Result, error) {
	recognizeReq, err := whiteboardOCRRequest(req)
	if err != nil {
		return ocr.Result{}, err
	}
	result, err := s.RecognizeImage(recognizeReq)
	if err != nil {
		_ = s.patchScreenshotOCRState(recognizeReq.SourceID, ocr.ResultStatusFailed, "", "", recognizeReq.Language, err.Error())
		return ocr.Result{}, err
	}
	_ = s.patchScreenshotOCRState(recognizeReq.SourceID, ocr.ResultStatusReady, result.ID, result.ModelID, result.Language, "")
	return result, nil
}

func screenshotOCRRequest(item ScreenshotItem, priority string) ocr.RecognizeRequest {
	return ocr.RecognizeRequest{
		ImagePath:  item.Path,
		SourceKind: screenshotOCRSourceKind(item),
		SourceID:   item.ID,
		Language:   "zh-en",
		Priority:   priority,
	}
}

func whiteboardOCRRequest(req ocr.WhiteboardRequest) (ocr.RecognizeRequest, error) {
	if strings.TrimSpace(req.ImagePath) == "" {
		return ocr.RecognizeRequest{}, errors.New("whiteboard OCR image path is required")
	}
	language := strings.TrimSpace(req.Language)
	if language == "" {
		language = "zh-en"
	}
	sourceID := strings.TrimSpace(req.SceneID)
	if sourceID == "" {
		sourceID = strings.TrimSpace(req.ElementID)
	}
	sourceKind := ocr.SourceWhiteboard
	if strings.TrimSpace(req.ElementID) != "" {
		sourceKind = ocr.SourceWhiteboardSelection
	}
	priority := strings.TrimSpace(req.Priority)
	switch priority {
	case ocr.JobPriorityInteractive, ocr.JobPriorityNormal, ocr.JobPriorityBackground:
	default:
		priority = ocr.JobPriorityInteractive
	}
	return ocr.RecognizeRequest{
		ImagePath:  req.ImagePath,
		SourceKind: sourceKind,
		SourceID:   sourceID,
		Language:   language,
		Force:      req.Force,
		Priority:   priority,
	}, nil
}

func (s *RecordingFreedomService) TranslateOcr(req ocr.TranslateRequest) (ocr.TranslationResult, error) {
	if s.ocr == nil {
		return ocr.TranslationResult{}, errors.New("OCR service is not initialized")
	}
	resolved, err := s.resolveOCRTranslateRequest(req)
	if err != nil {
		return ocr.TranslationResult{}, err
	}
	s.logOCRTranslateRequest("translate-request", resolved)
	result, err := s.ocr.Translate(resolved)
	if err != nil {
		s.logOCRTranslateRequest("translate-failed", resolved)
		return ocr.TranslationResult{}, err
	}
	s.logEvent("ocr", "translate-ready", map[string]string{
		"ocrResultId":    result.OcrResultID,
		"provider":       result.Provider,
		"sourceLanguage": result.SourceLanguage,
		"targetLanguage": result.TargetLanguage,
		"model":          result.Model,
		"blockCount":     fmt.Sprint(len(result.Blocks)),
	})
	return result, nil
}

func (s *RecordingFreedomService) CancelOcrJob(jobID string) error {
	if s.ocr == nil {
		return errors.New("OCR service is not initialized")
	}
	if err := s.ocr.CancelJob(jobID); err != nil {
		return err
	}
	return nil
}

func (s *RecordingFreedomService) OpenOcrResult(resultID string) (ocr.Result, error) {
	if s.ocr == nil {
		return ocr.Result{}, errors.New("OCR service is not initialized")
	}
	result, err := s.ocr.ReadResult(resultID)
	if err != nil {
		s.logEvent("ocr", "open-result-failed", map[string]string{
			"resultId": strings.TrimSpace(resultID),
		})
		return ocr.Result{}, err
	}
	s.logEvent("ocr", "open-result", map[string]string{
		"resultId":   result.ID,
		"sourceKind": string(result.SourceKind),
		"sourceId":   result.SourceID,
		"language":   result.Language,
		"modelId":    result.ModelID,
		"blockCount": fmt.Sprint(len(result.Blocks)),
	})
	return result, nil
}

func (s *RecordingFreedomService) ReadOcrResultImage(resultID string) (ScreenshotImageResult, error) {
	if s.ocr == nil {
		return ScreenshotImageResult{}, errors.New("OCR service is not initialized")
	}
	result, err := s.ocr.ReadResult(resultID)
	if err != nil {
		s.logEvent("ocr", "read-result-image-failed", map[string]string{
			"resultId": strings.TrimSpace(resultID),
		})
		return ScreenshotImageResult{}, err
	}
	path, err := s.ocrResultImagePath(result)
	if err != nil {
		s.logEvent("ocr", "read-result-image-failed", map[string]string{
			"resultId":   result.ID,
			"sourceKind": string(result.SourceKind),
			"sourceId":   result.SourceID,
		})
		return ScreenshotImageResult{}, err
	}
	image, err := readPreviewImageDataURL(path, "OCR result image", ocrResultImageMIMEType(path), screenshotMaxPreviewBytes, 0)
	if err != nil {
		s.logEvent("ocr", "read-result-image-failed", map[string]string{
			"resultId":   result.ID,
			"sourceKind": string(result.SourceKind),
			"sourceId":   result.SourceID,
		})
		return ScreenshotImageResult{}, err
	}
	s.logEvent("ocr", "read-result-image", map[string]string{
		"resultId":   result.ID,
		"sourceKind": string(result.SourceKind),
		"sourceId":   result.SourceID,
		"language":   result.Language,
		"modelId":    result.ModelID,
		"bytes":      fmt.Sprint(image.Bytes),
	})
	return ScreenshotImageResult{
		Available: image.Available,
		DataURL:   image.DataURL,
		Path:      path,
		Bytes:     image.Bytes,
	}, nil
}

func (s *RecordingFreedomService) ocrResultImagePath(result ocr.Result) (string, error) {
	if strings.TrimSpace(result.ImagePath) != "" {
		return s.managedOCRResultImagePath(result.ImagePath)
	}
	return s.fallbackOCRResultImagePath(result)
}

func (s *RecordingFreedomService) logOCRRecognizeRequest(event string, req ocr.RecognizeRequest) {
	if s == nil {
		return
	}
	s.logEvent("ocr", event, map[string]string{
		"sourceKind": string(req.SourceKind),
		"sourceId":   req.SourceID,
		"language":   req.Language,
		"modelId":    req.ModelID,
		"priority":   req.Priority,
		"force":      fmt.Sprint(req.Force),
	})
}

func (s *RecordingFreedomService) logOCRTranslateRequest(event string, req ocr.TranslateRequest) {
	if s == nil {
		return
	}
	s.logEvent("ocr", event, map[string]string{
		"ocrResultId":    req.OcrResultID,
		"provider":       req.Provider,
		"sourceLanguage": req.SourceLanguage,
		"targetLanguage": req.TargetLanguage,
		"model":          req.Model,
		"blockCount":     fmt.Sprint(len(req.BlockIDs)),
		"force":          fmt.Sprint(req.Force),
	})
}

func (s *RecordingFreedomService) fallbackOCRResultImagePath(result ocr.Result) (string, error) {
	sourceID := strings.TrimSpace(result.SourceID)
	if sourceID == "" {
		return "", errors.New("OCR result image path is empty")
	}
	if !ocrResultSourceCanFallbackToScreenshotHistory(result.SourceKind) {
		return "", errors.New("OCR result image path is empty")
	}
	item, err := s.screenshotItemByID(sourceID)
	if err != nil {
		return "", err
	}
	return managedScreenshotPath(s, item.Path)
}

func (s *RecordingFreedomService) managedOCRResultImagePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("OCR result image path is empty")
	}
	if s == nil || s.appData == nil {
		return "", errors.New("app data service is not initialized")
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	root, err := s.appData.RootDir()
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("OCR result image %q must stay inside %q", path, root)
	}
	switch strings.ToLower(filepath.Ext(target)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return target, nil
	default:
		return "", fmt.Errorf("OCR result image %q must be PNG, JPEG, or WebP", path)
	}
}

func ocrResultSourceCanFallbackToScreenshotHistory(kind ocr.SourceKind) bool {
	switch kind {
	case ocr.SourceRegionScreenshot,
		ocr.SourceFullScreenshot,
		ocr.SourceWindowScreenshot,
		ocr.SourceFocusedWindowScreenshot,
		ocr.SourceScrollingScreenshot,
		ocr.SourcePinnedScreenshot,
		ocr.SourceWhiteboard:
		return true
	default:
		return false
	}
}

func ocrResultImageMIMEType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func screenshotOCRSourceKind(item ScreenshotItem) ocr.SourceKind {
	switch strings.TrimSpace(item.Mode) {
	case "full", "screen":
		return ocr.SourceFullScreenshot
	case "window":
		return ocr.SourceWindowScreenshot
	case "focused-window":
		return ocr.SourceFocusedWindowScreenshot
	case "scrolling":
		return ocr.SourceScrollingScreenshot
	case "whiteboard":
		return ocr.SourceWhiteboard
	default:
		return ocr.SourceRegionScreenshot
	}
}

func (s *RecordingFreedomService) patchScreenshotOCRState(itemID string, status string, resultID string, modelID string, language string, message string) error {
	items, err := s.loadScreenshotHistory()
	if err != nil {
		return err
	}
	updated := false
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for index := range items {
		if items[index].ID != itemID {
			continue
		}
		items[index].OCRStatus = normalizeScreenshotOCRStatus(status)
		items[index].OCRResultID = strings.TrimSpace(resultID)
		items[index].OCRModelID = strings.TrimSpace(modelID)
		items[index].OCRLanguage = strings.TrimSpace(language)
		items[index].OCRUpdatedAt = now
		items[index].OCRError = strings.TrimSpace(message)
		updated = true
		break
	}
	if !updated {
		item, itemErr := s.screenshotItemByID(itemID)
		if itemErr != nil || strings.TrimSpace(item.Path) == "" {
			return fmt.Errorf("screenshot %q was not found", itemID)
		}
		item.OCRStatus = normalizeScreenshotOCRStatus(status)
		item.OCRResultID = strings.TrimSpace(resultID)
		item.OCRModelID = strings.TrimSpace(modelID)
		item.OCRLanguage = strings.TrimSpace(language)
		item.OCRUpdatedAt = now
		item.OCRError = strings.TrimSpace(message)
		items = append([]ScreenshotItem{item}, items...)
	}
	if err := s.saveScreenshotHistory(items); err != nil {
		return err
	}
	s.emitScreenshotHistoryChanged(items)
	return nil
}

func (s *RecordingFreedomService) emitOCRStatus() {
	if s.app == nil || s.ocr == nil {
		return
	}
	status, err := s.ocr.Status()
	if err != nil {
		return
	}
	s.app.Event.Emit("ocr.status.changed", status)
}

func (s *RecordingFreedomService) emitOCRJobEvent(name string, event OcrJobEvent) {
	event.Event = strings.TrimSpace(name)
	if event.Event == "" {
		event.Event = ocrJobEventName(event.Status)
	}
	_ = s.writeOCRJobEvidenceEvent(event)
	if s.app == nil {
		return
	}
	s.app.Event.Emit(name, event)
}

func (s *RecordingFreedomService) writeOCRJobEvidenceEvent(event OcrJobEvent) error {
	dir := s.ocrEvidenceDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "ocr-job-events.jsonl")
	s.logMu.Lock()
	defer s.logMu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *RecordingFreedomService) ocrEvidenceDir() string {
	root := ""
	if s != nil && s.appData != nil {
		root, _ = s.appData.RootDir()
	}
	if strings.TrimSpace(root) == "" {
		root = softwareRootFallback()
	}
	return filepath.Join(root, "data", "ocr", "evidence")
}

func (s *RecordingFreedomService) emitOCRModelEvent(name string, event OcrModelEvent) {
	if s.app == nil {
		return
	}
	s.app.Event.Emit(name, event)
}

func (s *RecordingFreedomService) startOCRJobEventPump() {
	if s == nil || s.ocr == nil {
		return
	}
	s.ocrPumpOnce.Do(func() {
		go func() {
			events := s.ocr.Events()
			for event := range events {
				s.handleOCRJobEvent(event)
			}
		}()
	})
}

func (s *RecordingFreedomService) startOCRModelDownloadEventPump() {
	if s == nil || s.ocr == nil {
		return
	}
	s.ocrModelDownloadPumpOnce.Do(func() {
		go func() {
			events := s.ocr.ModelDownloadEvents()
			for event := range events {
				s.handleOCRModelDownloadEvent(event)
			}
		}()
	})
}

func (s *RecordingFreedomService) handleOCRModelDownloadEvent(event ocr.ModelDownloadEvent) {
	if s == nil || s.app == nil {
		return
	}
	s.app.Event.Emit("ocr.model.download.changed", event)
	switch event.Snapshot.Status {
	case ocr.ModelDownloadInstalled:
		if event.Snapshot.Model != nil {
			s.emitOCRModelEvent("ocr.model.installed", OcrModelEvent{ModelID: event.Snapshot.Model.ID, Status: "installed", Model: event.Snapshot.Model})
		}
		s.emitOCRStatus()
	case ocr.ModelDownloadFailed, ocr.ModelDownloadCancelled:
		if event.Snapshot.Status == ocr.ModelDownloadFailed {
			s.emitOCRModelEvent("ocr.model.failed", OcrModelEvent{ModelID: event.Snapshot.ModelID, Status: "failed", Error: event.Snapshot.Error})
		}
		s.emitOCRStatus()
	}
}

func (s *RecordingFreedomService) handleOCRJobEvent(event ocr.JobEvent) {
	if s == nil {
		return
	}
	if isScreenshotOCRSourceKind(event.Request.SourceKind) && strings.TrimSpace(event.Request.SourceID) != "" {
		if s.shouldIgnoreScreenshotOCRJobEvent(event) {
			s.emitOCRJobEvent(ocrJobEventName(event.Status), OcrJobEvent{
				JobID:      event.JobID,
				SourceKind: event.Request.SourceKind,
				SourceID:   event.Request.SourceID,
				Status:     event.Status,
				CacheKey:   event.CacheKey,
				Merged:     event.Merged,
				Error:      event.Error,
				Result:     event.Result,
			})
			return
		}
		switch event.Status {
		case ocr.ResultStatusQueued, ocr.ResultStatusRunning:
			s.clearCancelledScreenshotOCRSource(event)
			_ = s.patchScreenshotOCRState(event.Request.SourceID, event.Status, "", "", event.Request.Language, "")
		case ocr.ResultStatusReady:
			resultID := ""
			modelID := ""
			language := event.Request.Language
			if event.Result != nil {
				resultID = event.Result.ID
				modelID = event.Result.ModelID
				language = event.Result.Language
			}
			_ = s.patchScreenshotOCRState(event.Request.SourceID, ocr.ResultStatusReady, resultID, modelID, language, "")
		case ocr.ResultStatusFailed:
			_ = s.patchScreenshotOCRState(event.Request.SourceID, ocr.ResultStatusFailed, "", "", event.Request.Language, event.Error)
		case ocr.ResultStatusCancelled:
			s.markCancelledScreenshotOCRSource(event)
			_ = s.patchScreenshotOCRState(event.Request.SourceID, ocr.ResultStatusNone, "", "", event.Request.Language, "")
		}
	}
	s.emitOCRJobEvent(ocrJobEventName(event.Status), OcrJobEvent{
		JobID:      event.JobID,
		SourceKind: event.Request.SourceKind,
		SourceID:   event.Request.SourceID,
		Status:     event.Status,
		CacheKey:   event.CacheKey,
		Merged:     event.Merged,
		Error:      event.Error,
		Result:     event.Result,
	})
}

func (s *RecordingFreedomService) shouldIgnoreScreenshotOCRJobEvent(event ocr.JobEvent) bool {
	if s == nil || event.Status == ocr.ResultStatusQueued || event.Status == ocr.ResultStatusCancelled {
		return false
	}
	key := screenshotOCRCancelKey(event.Request.SourceKind, event.Request.SourceID)
	jobID := strings.TrimSpace(event.JobID)
	if key == "" || jobID == "" {
		return false
	}
	s.screenshotMu.Lock()
	defer s.screenshotMu.Unlock()
	jobs := s.ocrCancelledSources[key]
	_, ignored := jobs[jobID]
	if ignored && (event.Status == ocr.ResultStatusReady || event.Status == ocr.ResultStatusFailed) {
		delete(jobs, jobID)
		if len(jobs) == 0 {
			delete(s.ocrCancelledSources, key)
		}
	}
	return ignored
}

func (s *RecordingFreedomService) markCancelledScreenshotOCRSource(event ocr.JobEvent) {
	key := screenshotOCRCancelKey(event.Request.SourceKind, event.Request.SourceID)
	if key == "" {
		return
	}
	s.screenshotMu.Lock()
	defer s.screenshotMu.Unlock()
	if s.ocrCancelledSources == nil {
		s.ocrCancelledSources = map[string]map[string]struct{}{}
	}
	jobID := strings.TrimSpace(event.JobID)
	if jobID == "" {
		return
	}
	if s.ocrCancelledSources[key] == nil {
		s.ocrCancelledSources[key] = map[string]struct{}{}
	}
	s.ocrCancelledSources[key][jobID] = struct{}{}
}

func (s *RecordingFreedomService) clearCancelledScreenshotOCRSource(event ocr.JobEvent) {
	key := screenshotOCRCancelKey(event.Request.SourceKind, event.Request.SourceID)
	jobID := strings.TrimSpace(event.JobID)
	if key == "" || jobID == "" {
		return
	}
	s.screenshotMu.Lock()
	defer s.screenshotMu.Unlock()
	if s.ocrCancelledSources != nil && s.ocrCancelledSources[key] != nil {
		delete(s.ocrCancelledSources[key], jobID)
		if len(s.ocrCancelledSources[key]) == 0 {
			delete(s.ocrCancelledSources, key)
		}
	}
}

func screenshotOCRCancelKey(kind ocr.SourceKind, sourceID string) string {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return ""
	}
	return string(kind) + "\x00" + sourceID
}

func ocrJobEventName(status string) string {
	switch status {
	case ocr.ResultStatusQueued:
		return "ocr.job.queued"
	case ocr.ResultStatusRunning:
		return "ocr.job.started"
	case ocr.ResultStatusReady:
		return "ocr.job.finished"
	case ocr.ResultStatusCancelled:
		return "ocr.job.cancelled"
	default:
		return "ocr.job.failed"
	}
}

func isScreenshotOCRSourceKind(kind ocr.SourceKind) bool {
	switch kind {
	case ocr.SourceRegionScreenshot,
		ocr.SourceFullScreenshot,
		ocr.SourceWindowScreenshot,
		ocr.SourceFocusedWindowScreenshot,
		ocr.SourceScrollingScreenshot,
		ocr.SourcePinnedScreenshot,
		ocr.SourceWhiteboard,
		ocr.SourceWhiteboardSelection:
		return true
	default:
		return false
	}
}
