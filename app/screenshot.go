package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	desktopscreenshot "github.com/kbinani/screenshot"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/image/draw"
)

const (
	screenshotDirName                = "screenshots"
	screenshotHistoryFileName        = "history.json"
	screenshotMaxPreviewBytes        = 16 * 1024 * 1024
	screenshotThumbnailMaxSide       = 320
	screenshotMinRegionSize          = 12
	regionSelectionPurposeScreenshot = "screenshot"
	regionSelectionPurposeScrolling  = "scrolling-screenshot"
	annotationOverlayModeScreenshot  = "screenshot"
	screenshotAnnotationTargetType   = "screenshot-region"
	screenshotAnnotationTargetID     = "screenshot:region"
	scrollingScreenshotMode          = "scrolling"
	scrollingScreenshotMaxFrames     = 24
	scrollingScreenshotMaxHeight     = 32000
	scrollingScreenshotMinAppend     = 8
	scrollingScreenshotScrollDelay   = 180 * time.Millisecond
)

type screenshotCaptureFunc func(image.Rectangle) (*image.RGBA, error)
type screenshotScrollFunc func(image.Rectangle) error
type screenshotSleepFunc func(time.Duration)

type ScreenshotCaptureRequest struct {
	Mode   string      `json:"mode,omitempty"`
	Region *RegionRect `json:"region,omitempty"`
}

type ScreenshotHistoryResult struct {
	Items []ScreenshotItem `json:"items"`
}

type ScreenshotCaptureResult struct {
	Item ScreenshotItem `json:"item"`
}

type ScreenshotItem struct {
	ID            string      `json:"id"`
	Path          string      `json:"path"`
	ThumbnailPath string      `json:"thumbnailPath,omitempty"`
	CreatedAt     string      `json:"createdAt"`
	Width         int         `json:"width"`
	Height        int         `json:"height"`
	Mode          string      `json:"mode"`
	Region        *RegionRect `json:"region,omitempty"`
	Pinned        bool        `json:"pinned"`
	Fixed         bool        `json:"fixed"`
	OCRStatus     string      `json:"ocrStatus"`
	OCRResultID   string      `json:"ocrResultId,omitempty"`
	OCRModelID    string      `json:"ocrModelId,omitempty"`
	OCRLanguage   string      `json:"ocrLanguage,omitempty"`
	OCRUpdatedAt  string      `json:"ocrUpdatedAt,omitempty"`
	OCRError      string      `json:"ocrError,omitempty"`
}

type ScreenshotItemPatchRequest struct {
	ID     string `json:"id"`
	Pinned *bool  `json:"pinned,omitempty"`
	Fixed  *bool  `json:"fixed,omitempty"`
}

type ScreenshotImageRequest struct {
	ID        string `json:"id,omitempty"`
	Path      string `json:"path,omitempty"`
	Thumbnail bool   `json:"thumbnail,omitempty"`
}

type ScreenshotImageResult struct {
	Available bool   `json:"available"`
	DataURL   string `json:"dataUrl,omitempty"`
	Path      string `json:"path,omitempty"`
	Bytes     int64  `json:"bytes,omitempty"`
}

type ScreenshotPinState struct {
	Visible bool           `json:"visible"`
	Item    ScreenshotItem `json:"item,omitempty"`
	DataURL string         `json:"dataUrl,omitempty"`
	Fixed   bool           `json:"fixed"`
}

type ScreenshotWhiteboardContext struct {
	Available bool           `json:"available"`
	Item      ScreenshotItem `json:"item,omitempty"`
	DataURL   string         `json:"dataUrl,omitempty"`
}

type ScreenshotCapturedEvent struct {
	Item ScreenshotItem `json:"item"`
}

type ScreenshotPinEvent struct {
	Visible bool           `json:"visible"`
	Item    ScreenshotItem `json:"item,omitempty"`
	DataURL string         `json:"dataUrl,omitempty"`
	Fixed   bool           `json:"fixed"`
}

type screenshotHistoryStore struct {
	SchemaVersion int              `json:"schemaVersion"`
	Items         []ScreenshotItem `json:"items"`
}

func (s *RecordingFreedomService) ListScreenshots() (ScreenshotHistoryResult, error) {
	items, err := s.loadScreenshotHistory()
	if err != nil {
		return ScreenshotHistoryResult{}, err
	}
	return ScreenshotHistoryResult{Items: items}, nil
}

func (s *RecordingFreedomService) CaptureScreenshot(req ScreenshotCaptureRequest) (ScreenshotCaptureResult, error) {
	rect, region, err := s.captureScreenshotRect(req)
	if err != nil {
		return ScreenshotCaptureResult{}, err
	}
	item, err := s.captureScreenshot(rect, strings.TrimSpace(req.Mode), region)
	if err != nil {
		return ScreenshotCaptureResult{}, err
	}
	s.emitScreenshotCaptured(item)
	return ScreenshotCaptureResult{Item: item}, nil
}

func (s *RecordingFreedomService) ShowScreenshotRegionSelector() (RegionSelectionSession, error) {
	if recorderIsActive(s.recorder.State()) {
		return RegionSelectionSession{}, errors.New("cannot take a screenshot region while recording is active")
	}
	if s.app == nil || s.regionOverlay == nil {
		return RegionSelectionSession{}, errors.New("region overlay window is not configured")
	}
	bounds, displayCount := regionOverlayBounds(s.app.Screen.GetAll())
	captureBounds := screenshotCaptureUnionBounds()
	session := RegionSelectionSession{
		ID:            fmt.Sprintf("screenshot-%d", time.Now().UnixNano()),
		Bounds:        regionRectFromAppRect(bounds),
		CaptureBounds: &captureBounds,
		DisplayBounds: regionDisplayBoundsForScreens(s.app.Screen.GetAll(), screenshotCaptureDisplayBounds()),
		MinimumWidth:  screenshotMinRegionSize,
		MinimumHeight: screenshotMinRegionSize,
		DisplayCount:  displayCount,
		Purpose:       regionSelectionPurposeScreenshot,
	}
	session, err := s.showRegionSelectionSession(session, bounds, regionSelectionSessionReset{ClearScreenshot: true})
	if err != nil {
		return RegionSelectionSession{}, err
	}
	s.logEvent("screenshot", "region-selector-show", map[string]string{
		"sessionId": session.ID,
		"bounds":    fmt.Sprintf("%d,%d %dx%d", bounds.X, bounds.Y, bounds.Width, bounds.Height),
	})
	return session, nil
}

func (s *RecordingFreedomService) CompleteScreenshotRegionSelection(req RegionSelectionRequest) (ScreenshotCaptureResult, error) {
	s.regionMu.Lock()
	session := s.regionSession
	absoluteSelection := s.screenshotRegionDIP
	s.regionSession = nil
	s.screenshotRegionDIP = application.Rect{}
	s.regionMu.Unlock()
	if session == nil {
		return ScreenshotCaptureResult{}, errors.New("no active screenshot selection session")
	}
	s.clearRegionElementCache(session.ID)
	s.clearRegionAssistSnapshot(session.ID)
	if session.Purpose != regionSelectionPurposeScreenshot {
		return ScreenshotCaptureResult{}, errors.New("active region selection session is not for screenshots")
	}
	relative := normalizeRegionSelection(req)
	if absoluteSelection.Width > 0 && absoluteSelection.Height > 0 {
		relative = application.Rect{
			X:      absoluteSelection.X - session.Bounds.X,
			Y:      absoluteSelection.Y - session.Bounds.Y,
			Width:  absoluteSelection.Width,
			Height: absoluteSelection.Height,
		}
	}
	if relative.Width < session.MinimumWidth || relative.Height < session.MinimumHeight {
		return ScreenshotCaptureResult{}, fmt.Errorf("selected screenshot region must be at least %d x %d", session.MinimumWidth, session.MinimumHeight)
	}
	if s.regionOverlay != nil {
		s.regionOverlay.Hide()
	}
	capsuleHidden := false
	if s.capsuleWindow != nil {
		s.capsuleWindow.Hide()
		capsuleHidden = true
	}
	defer func() {
		if capsuleHidden {
			s.restoreCapsuleWindow()
		}
	}()
	time.Sleep(140 * time.Millisecond)
	captureRect := mapRegionSelectionToCaptureRect(*session, regionRectFromAppRect(relative))
	item, err := s.captureScreenshot(captureRect, "region", &RegionRect{
		X:      captureRect.Min.X,
		Y:      captureRect.Min.Y,
		Width:  captureRect.Dx(),
		Height: captureRect.Dy(),
	})
	if err != nil {
		return ScreenshotCaptureResult{}, err
	}
	s.emitScreenshotCaptured(item)
	return ScreenshotCaptureResult{Item: item}, nil
}

func (s *RecordingFreedomService) BeginScreenshotRegionEdit(req RegionSelectionRequest) (RegionFrameState, error) {
	session, err := s.activeScreenshotRegionSession()
	if err != nil {
		return RegionFrameState{}, err
	}
	relative := normalizeRegionSelection(req)
	if relative.Width < session.MinimumWidth || relative.Height < session.MinimumHeight {
		return RegionFrameState{}, fmt.Errorf("selected screenshot region must be at least %d x %d", session.MinimumWidth, session.MinimumHeight)
	}
	absolute := application.Rect{
		X:      session.Bounds.X + relative.X,
		Y:      session.Bounds.Y + relative.Y,
		Width:  relative.Width,
		Height: relative.Height,
	}
	return s.setScreenshotRegionEditor(absolute)
}

func (s *RecordingFreedomService) UpdateScreenshotRegionSelection(req RegionSelectionRequest) (RegionFrameState, error) {
	if _, err := s.activeScreenshotRegionSession(); err != nil {
		return RegionFrameState{}, err
	}
	absolute := normalizeRegionSelection(req)
	if absolute.Width < minRegionWidth || absolute.Height < minRegionHeight {
		return RegionFrameState{}, fmt.Errorf("selected screenshot region must be at least %d x %d", minRegionWidth, minRegionHeight)
	}
	return s.setScreenshotRegionEditor(absolute)
}

func (s *RecordingFreedomService) BeginScreenshotAnnotationOverlay(req RegionSelectionRequest) (AnnotationOverlayState, error) {
	if recorderIsActive(s.recorder.State()) {
		return AnnotationOverlayState{}, errors.New("cannot annotate a screenshot while recording is active")
	}
	session, err := s.activeScreenshotRegionSession()
	if err != nil {
		return AnnotationOverlayState{}, err
	}
	relative := normalizeRegionSelection(req)
	if relative.Width < session.MinimumWidth || relative.Height < session.MinimumHeight {
		return AnnotationOverlayState{}, fmt.Errorf("selected screenshot region must be at least %d x %d", session.MinimumWidth, session.MinimumHeight)
	}
	absoluteDIP := application.Rect{
		X:      session.Bounds.X + relative.X,
		Y:      session.Bounds.Y + relative.Y,
		Width:  relative.Width,
		Height: relative.Height,
	}
	captureRect := mapRegionSelectionToCaptureRect(session, regionRectFromAppRect(relative))

	s.regionMu.Lock()
	s.regionSession = nil
	s.screenshotRegionDIP = application.Rect{}
	s.regionMu.Unlock()
	s.clearRegionElementCache(session.ID)
	s.clearRegionAssistSnapshot(session.ID)
	if s.regionOverlay != nil {
		s.regionOverlay.Hide()
	}
	capsuleHidden := false
	if s.capsuleWindow != nil {
		s.capsuleWindow.Hide()
		capsuleHidden = true
	}
	defer func() {
		if capsuleHidden {
			s.restoreCapsuleWindow()
		}
	}()
	time.Sleep(140 * time.Millisecond)

	img, err := desktopscreenshot.CaptureRect(captureRect)
	if err != nil {
		return AnnotationOverlayState{}, err
	}
	if img == nil || img.Bounds().Empty() {
		return AnnotationOverlayState{}, errors.New("captured screenshot is empty")
	}
	dataURL, err := screenshotImageDataURL(img)
	if err != nil {
		return AnnotationOverlayState{}, err
	}
	now := time.Now().UTC()
	region := RegionRect{
		X:      captureRect.Min.X,
		Y:      captureRect.Min.Y,
		Width:  captureRect.Dx(),
		Height: captureRect.Dy(),
	}
	context := ScreenshotWhiteboardContext{
		Available: true,
		Item: ScreenshotItem{
			ID:        "screenshot-draft-" + now.Format("20060102-150405.000000000"),
			CreatedAt: now.Format(time.RFC3339Nano),
			Width:     img.Bounds().Dx(),
			Height:    img.Bounds().Dy(),
			Mode:      "region",
			Region:    &region,
		},
		DataURL: dataURL,
	}
	s.screenshotMu.Lock()
	s.screenshotAnnotation = context
	s.screenshotMu.Unlock()
	if capsuleHidden {
		s.restoreCapsuleWindow()
		capsuleHidden = false
	}
	return s.showScreenshotAnnotationOverlay(absoluteDIP, context.Item)
}

func (s *RecordingFreedomService) LoadScreenshotAnnotationCapture() (ScreenshotWhiteboardContext, error) {
	s.screenshotMu.Lock()
	defer s.screenshotMu.Unlock()
	if !s.screenshotAnnotation.Available || strings.TrimSpace(s.screenshotAnnotation.DataURL) == "" {
		return ScreenshotWhiteboardContext{}, errors.New("screenshot annotation requires an active screenshot draft")
	}
	return s.screenshotAnnotation, nil
}

func (s *RecordingFreedomService) SaveScreenshotAnnotationCapture(req AnnotationCaptureRequest) (ScreenshotCaptureResult, error) {
	s.screenshotMu.Lock()
	context := s.screenshotAnnotation
	s.screenshotMu.Unlock()
	if !context.Available {
		return ScreenshotCaptureResult{}, errors.New("screenshot annotation requires an active screenshot draft")
	}
	data, err := decodeAnnotationSnapshot(req.SnapshotDataURL)
	if err != nil {
		return ScreenshotCaptureResult{}, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return ScreenshotCaptureResult{}, fmt.Errorf("screenshot annotation PNG is invalid: %w", err)
	}
	item, err := s.saveScreenshotImage(img, "region", context.Item.Region)
	if err != nil {
		return ScreenshotCaptureResult{}, err
	}
	s.screenshotMu.Lock()
	s.screenshotAnnotation.Item = item
	s.screenshotMu.Unlock()
	s.emitScreenshotCaptured(item)
	s.logEvent("screenshot", "annotation-save", map[string]string{
		"id":     item.ID,
		"path":   item.Path,
		"width":  fmt.Sprint(item.Width),
		"height": fmt.Sprint(item.Height),
	})
	return ScreenshotCaptureResult{Item: item}, nil
}

func (s *RecordingFreedomService) ReselectScreenshotAnnotationRegion() (RegionSelectionSession, error) {
	if recorderIsActive(s.recorder.State()) {
		return RegionSelectionSession{}, errors.New("cannot reselect screenshot region while recording is active")
	}
	s.screenshotMu.Lock()
	s.screenshotAnnotation = ScreenshotWhiteboardContext{}
	s.screenshotMu.Unlock()
	if err := s.HideAnnotationOverlay(); err != nil {
		return RegionSelectionSession{}, err
	}
	return s.ShowScreenshotRegionSelector()
}

func (s *RecordingFreedomService) HideScreenshotAnnotationOverlay() error {
	s.screenshotMu.Lock()
	s.screenshotAnnotation = ScreenshotWhiteboardContext{}
	s.screenshotMu.Unlock()
	return s.HideAnnotationOverlay()
}

func (s *RecordingFreedomService) activeScreenshotRegionSession() (RegionSelectionSession, error) {
	s.regionMu.Lock()
	defer s.regionMu.Unlock()
	if s.regionSession == nil {
		return RegionSelectionSession{}, errors.New("no active screenshot selection session")
	}
	if s.regionSession.Purpose != regionSelectionPurposeScreenshot {
		return RegionSelectionSession{}, errors.New("active region selection session is not for screenshots")
	}
	return *s.regionSession, nil
}

func (s *RecordingFreedomService) setScreenshotRegionEditor(bounds application.Rect) (RegionFrameState, error) {
	s.regionMu.Lock()
	s.screenshotRegionDIP = bounds
	s.regionMu.Unlock()
	return s.showRegionEditorWithPurpose(bounds, regionSelectionPurposeScreenshot)
}

func (s *RecordingFreedomService) showScreenshotAnnotationOverlay(canvasBounds application.Rect, item ScreenshotItem) (AnnotationOverlayState, error) {
	if s.app == nil || s.annotationOverlay == nil {
		return AnnotationOverlayState{}, errors.New("annotation overlay window is not configured")
	}
	if canvasBounds.Width <= 0 || canvasBounds.Height <= 0 {
		return AnnotationOverlayState{}, errors.New("screenshot annotation bounds are empty")
	}
	windowBounds := annotationOverlayWindowBounds(canvasBounds)
	state := AnnotationOverlayState{
		Mode:         annotationOverlayModeScreenshot,
		WindowBounds: regionRectFromAppRect(windowBounds),
		CanvasBounds: RegionRect{
			X:      annotationOverlayFrameInset,
			Y:      annotationOverlayFrameInset,
			Width:  canvasBounds.Width,
			Height: canvasBounds.Height,
		},
		Target:          screenshotAnnotationTargetFromRect(canvasBounds, item),
		CaptureExcluded: false,
	}
	s.annotationOverlay.SetIgnoreMouseEvents(false)
	s.annotationOverlay.SetAlwaysOnTop(true)
	s.annotationOverlay.SetBounds(windowBounds)
	s.annotationOverlay.Show()
	s.annotationOverlay.SetBounds(windowBounds)
	s.annotationOverlay.Focus()
	s.broadcastAnnotationOverlayState(state)
	go s.rebroadcastAnnotationOverlayState(state, s.nextAnnotationToken())
	s.logEvent("screenshot", "annotation-overlay-show", map[string]string{
		"id":     item.ID,
		"bounds": fmt.Sprintf("%d,%d %dx%d", canvasBounds.X, canvasBounds.Y, canvasBounds.Width, canvasBounds.Height),
	})
	s.emitWhiteboardVisibility(true, "annotation")
	return state, nil
}

func (s *RecordingFreedomService) PatchScreenshotItem(req ScreenshotItemPatchRequest) (ScreenshotHistoryResult, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return ScreenshotHistoryResult{}, errors.New("screenshot id is required")
	}
	items, err := s.loadScreenshotHistory()
	if err != nil {
		return ScreenshotHistoryResult{}, err
	}
	changed := false
	for index := range items {
		if items[index].ID != id {
			continue
		}
		if req.Pinned != nil {
			changed = true
			if !*req.Pinned {
				items[index].Fixed = false
			}
		}
		if req.Fixed != nil {
			items[index].Fixed = *req.Fixed
			changed = true
		}
		items[index].Pinned = false
		break
	}
	if !changed {
		return ScreenshotHistoryResult{}, fmt.Errorf("screenshot %q was not found", id)
	}
	if err := s.saveScreenshotHistory(items); err != nil {
		return ScreenshotHistoryResult{}, err
	}
	s.emitScreenshotHistoryChanged(items)
	var pinState ScreenshotPinState
	for _, item := range items {
		if item.ID != id {
			continue
		}
		s.screenshotMu.Lock()
		if s.screenshotPinState.Visible && s.screenshotPinState.Item.ID == id {
			s.screenshotPinState.Item = item
			s.screenshotPinState.Fixed = item.Fixed
			pinState = s.screenshotPinState
		}
		s.screenshotMu.Unlock()
		break
	}
	if pinState.Visible {
		s.broadcastScreenshotPinState(pinState)
		s.emitScreenshotPin(pinState)
	}
	return ScreenshotHistoryResult{Items: items}, nil
}

func (s *RecordingFreedomService) ReadScreenshotImage(req ScreenshotImageRequest) (ScreenshotImageResult, error) {
	path, err := s.screenshotImagePath(req)
	if err != nil {
		return ScreenshotImageResult{}, err
	}
	image, err := readPreviewImageDataURL(path, "screenshot image", "image/png", screenshotMaxPreviewBytes, 0)
	if err != nil {
		return ScreenshotImageResult{}, err
	}
	return ScreenshotImageResult{
		Available: image.Available,
		DataURL:   image.DataURL,
		Path:      path,
		Bytes:     image.Bytes,
	}, nil
}

func (s *RecordingFreedomService) OpenScreenshot(req ScreenshotImageRequest) (ScreenshotItem, error) {
	item, err := s.screenshotItemForRequest(req)
	if err != nil {
		return ScreenshotItem{}, err
	}
	if err := openPath(item.Path); err != nil {
		return ScreenshotItem{}, err
	}
	return item, nil
}

func (s *RecordingFreedomService) OpenScreenshotDirectory(req ScreenshotImageRequest) (ScreenshotItem, error) {
	item, err := s.screenshotItemForRequest(req)
	if err != nil {
		return ScreenshotItem{}, err
	}
	path, err := managedScreenshotPath(s, item.Path)
	if err != nil {
		return ScreenshotItem{}, err
	}
	if err := openPath(filepath.Dir(path)); err != nil {
		return ScreenshotItem{}, err
	}
	return item, nil
}

func (s *RecordingFreedomService) DeleteScreenshotItem(id string) (ScreenshotHistoryResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ScreenshotHistoryResult{}, errors.New("screenshot id is required")
	}
	items, err := s.loadScreenshotHistory()
	if err != nil {
		return ScreenshotHistoryResult{}, err
	}
	remaining := make([]ScreenshotItem, 0, len(items))
	var removed ScreenshotItem
	for _, item := range items {
		if item.ID == id {
			removed = item
			continue
		}
		remaining = append(remaining, item)
	}
	if removed.ID == "" {
		return ScreenshotHistoryResult{}, fmt.Errorf("screenshot %q was not found", id)
	}
	for _, path := range uniqueScreenshotFilePaths(removed) {
		if err := removeManagedScreenshotFile(s, path); err != nil {
			return ScreenshotHistoryResult{}, err
		}
	}
	if err := s.saveScreenshotHistory(remaining); err != nil {
		return ScreenshotHistoryResult{}, err
	}
	s.emitScreenshotHistoryChanged(remaining)
	s.screenshotMu.Lock()
	clearWhiteboardContext := s.whiteboardScreenshot.Item.ID == id
	pinnedDeleted := s.screenshotPinState.Visible && s.screenshotPinState.Item.ID == id
	if clearWhiteboardContext {
		s.whiteboardScreenshot = ScreenshotWhiteboardContext{}
	}
	s.screenshotMu.Unlock()
	if pinnedDeleted {
		_ = s.HidePinnedScreenshot()
	}
	s.logEvent("screenshot", "delete", map[string]string{"id": id, "path": removed.Path})
	return ScreenshotHistoryResult{Items: remaining}, nil
}

func (s *RecordingFreedomService) ShowPinnedScreenshot(id string) (ScreenshotPinState, error) {
	if s.screenshotPinWindow == nil {
		return ScreenshotPinState{}, errors.New("screenshot pin window is not configured")
	}
	item, err := s.screenshotItemByID(id)
	if err != nil {
		return ScreenshotPinState{}, err
	}
	imageResult, err := s.ReadScreenshotImage(ScreenshotImageRequest{ID: item.ID})
	if err != nil {
		return ScreenshotPinState{}, err
	}
	if !imageResult.Available {
		return ScreenshotPinState{}, fmt.Errorf("screenshot %q image is unavailable", item.ID)
	}
	state := ScreenshotPinState{
		Visible: true,
		Item:    item,
		DataURL: imageResult.DataURL,
		Fixed:   item.Fixed,
	}
	s.screenshotMu.Lock()
	s.screenshotPinState = state
	s.screenshotMu.Unlock()
	s.screenshotPinWindow.SetAlwaysOnTop(true)
	s.screenshotPinWindow.SetBounds(screenshotPinWindowBounds(item))
	s.screenshotPinWindow.Show()
	s.screenshotPinWindow.SetBounds(screenshotPinWindowBounds(item))
	s.broadcastScreenshotPinState(state)
	s.emitScreenshotPin(state)
	return state, nil
}

func (s *RecordingFreedomService) HidePinnedScreenshot() error {
	state := ScreenshotPinState{Visible: false}
	s.screenshotMu.Lock()
	s.screenshotPinState = state
	s.screenshotMu.Unlock()
	if s.screenshotPinWindow != nil {
		s.screenshotPinWindow.Hide()
	}
	s.broadcastScreenshotPinState(state)
	s.emitScreenshotPin(state)
	return nil
}

func (s *RecordingFreedomService) LoadPinnedScreenshot() ScreenshotPinState {
	s.screenshotMu.Lock()
	defer s.screenshotMu.Unlock()
	return s.screenshotPinState
}

func (s *RecordingFreedomService) OpenScreenshotInWhiteboard(id string) (ScreenshotWhiteboardContext, error) {
	item, err := s.screenshotItemByID(id)
	if err != nil {
		return ScreenshotWhiteboardContext{}, err
	}
	imageResult, err := s.ReadScreenshotImage(ScreenshotImageRequest{ID: item.ID})
	if err != nil {
		return ScreenshotWhiteboardContext{}, err
	}
	if !imageResult.Available {
		return ScreenshotWhiteboardContext{}, fmt.Errorf("screenshot %q image is unavailable", item.ID)
	}
	context := ScreenshotWhiteboardContext{
		Available: true,
		Item:      item,
		DataURL:   imageResult.DataURL,
	}
	s.screenshotMu.Lock()
	s.whiteboardScreenshot = context
	s.screenshotMu.Unlock()
	if err := s.ShowWhiteboardWindow(); err != nil {
		return ScreenshotWhiteboardContext{}, err
	}
	s.broadcastScreenshotWhiteboardContext(context)
	s.emitScreenshotWhiteboardContext(context)
	return context, nil
}

func (s *RecordingFreedomService) ConsumeScreenshotWhiteboardContext() ScreenshotWhiteboardContext {
	s.screenshotMu.Lock()
	defer s.screenshotMu.Unlock()
	context := s.whiteboardScreenshot
	s.whiteboardScreenshot = ScreenshotWhiteboardContext{}
	return context
}

func (s *RecordingFreedomService) StartScrollingScreenshot() (RegionSelectionSession, error) {
	if recorderIsActive(s.recorder.State()) {
		return RegionSelectionSession{}, errors.New("cannot take a scrolling screenshot while recording is active")
	}
	if s.app == nil || s.regionOverlay == nil {
		return RegionSelectionSession{}, errors.New("region overlay window is not configured")
	}
	bounds, displayCount := regionOverlayBounds(s.app.Screen.GetAll())
	captureBounds := screenshotCaptureUnionBounds()
	session := RegionSelectionSession{
		ID:            fmt.Sprintf("scrolling-screenshot-%d", time.Now().UnixNano()),
		Bounds:        regionRectFromAppRect(bounds),
		CaptureBounds: &captureBounds,
		DisplayBounds: regionDisplayBoundsForScreens(s.app.Screen.GetAll(), screenshotCaptureDisplayBounds()),
		MinimumWidth:  minRegionWidth,
		MinimumHeight: minRegionHeight,
		DisplayCount:  displayCount,
		Purpose:       regionSelectionPurposeScrolling,
	}
	session, err := s.showRegionSelectionSession(session, bounds, regionSelectionSessionReset{ClearScreenshot: true})
	if err != nil {
		return RegionSelectionSession{}, err
	}
	s.logEvent("screenshot", "scrolling-selector-show", map[string]string{
		"sessionId": session.ID,
		"bounds":    fmt.Sprintf("%d,%d %dx%d", bounds.X, bounds.Y, bounds.Width, bounds.Height),
	})
	return session, nil
}

func (s *RecordingFreedomService) CompleteScrollingScreenshotSelection(req RegionSelectionRequest) (ScreenshotCaptureResult, error) {
	s.regionMu.Lock()
	session := s.regionSession
	s.regionSession = nil
	s.screenshotRegionDIP = application.Rect{}
	s.regionMu.Unlock()
	if session == nil {
		return ScreenshotCaptureResult{}, errors.New("no active scrolling screenshot selection session")
	}
	s.clearRegionElementCache(session.ID)
	s.clearRegionAssistSnapshot(session.ID)
	if session.Purpose != regionSelectionPurposeScrolling {
		return ScreenshotCaptureResult{}, errors.New("active region selection session is not for scrolling screenshots")
	}
	relative := normalizeRegionSelection(req)
	if relative.Width < session.MinimumWidth || relative.Height < session.MinimumHeight {
		return ScreenshotCaptureResult{}, fmt.Errorf("selected scrolling screenshot region must be at least %d x %d", session.MinimumWidth, session.MinimumHeight)
	}
	if s.regionOverlay != nil {
		s.regionOverlay.Hide()
	}
	capsuleHidden := false
	if s.capsuleWindow != nil {
		s.capsuleWindow.Hide()
		capsuleHidden = true
	}
	defer func() {
		if capsuleHidden {
			s.restoreCapsuleWindow()
		}
	}()
	time.Sleep(160 * time.Millisecond)

	captureRect := mapRegionSelectionToCaptureRect(*session, regionRectFromAppRect(relative))
	region := RegionRect{
		X:      captureRect.Min.X,
		Y:      captureRect.Min.Y,
		Width:  captureRect.Dx(),
		Height: captureRect.Dy(),
	}
	item, err := s.captureScrollingScreenshot(captureRect, &region)
	if err != nil {
		return ScreenshotCaptureResult{}, err
	}
	s.emitScreenshotCaptured(item)
	return ScreenshotCaptureResult{Item: item}, nil
}

func (s *RecordingFreedomService) captureScreenshotRect(req ScreenshotCaptureRequest) (image.Rectangle, *RegionRect, error) {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "full"
	}
	if req.Region != nil {
		region := *req.Region
		if region.Width < minRegionWidth || region.Height < minRegionHeight {
			return image.Rectangle{}, nil, fmt.Errorf("screenshot region must be at least %d x %d", minRegionWidth, minRegionHeight)
		}
		return image.Rect(region.X, region.Y, region.X+region.Width, region.Y+region.Height), &region, nil
	}
	switch mode {
	case "screen":
		rect, ok := firstDisplayScreenshotRect()
		if ok {
			region := RegionRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()}
			return rect, &region, nil
		}
	case "window":
		rect, ok := s.firstWindowScreenshotRect()
		if ok {
			region := RegionRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()}
			return rect, &region, nil
		}
		return image.Rectangle{}, nil, errors.New("no visible window is available for window screenshot")
	case "focused-window":
		rect, ok := s.focusedWindowScreenshotRect()
		if ok {
			region := RegionRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()}
			return rect, &region, nil
		}
		rect, ok = s.firstWindowScreenshotRect()
		if ok {
			region := RegionRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()}
			return rect, &region, nil
		}
		return image.Rectangle{}, nil, errors.New("no focused or visible window is available for focused window screenshot")
	}
	bounds := screenshotCaptureUnionBounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return image.Rectangle{}, nil, errors.New("no active display is available for screenshots")
	}
	region := bounds
	return image.Rect(bounds.X, bounds.Y, bounds.X+bounds.Width, bounds.Y+bounds.Height), &region, nil
}

var focusedWindowScreenshotRectProvider = detectFocusedWindowScreenshotRect

func (s *RecordingFreedomService) focusedWindowScreenshotRect() (image.Rectangle, bool) {
	return focusedWindowScreenshotRectProvider()
}

func firstDisplayScreenshotRect() (image.Rectangle, bool) {
	if desktopscreenshot.NumActiveDisplays() <= 0 {
		return image.Rectangle{}, false
	}
	rect := desktopscreenshot.GetDisplayBounds(0)
	return rect, !rect.Empty()
}

func (s *RecordingFreedomService) firstWindowScreenshotRect() (image.Rectangle, bool) {
	if s == nil || s.devices == nil {
		return image.Rectangle{}, false
	}
	for _, source := range s.devices.ListSources() {
		if source.Type != devices.SourceWindow || source.Width <= 0 || source.Height <= 0 {
			continue
		}
		rect := image.Rect(source.X, source.Y, source.X+source.Width, source.Y+source.Height)
		if !rect.Empty() {
			return rect, true
		}
	}
	return image.Rectangle{}, false
}

func (s *RecordingFreedomService) captureScreenshot(rect image.Rectangle, mode string, region *RegionRect) (ScreenshotItem, error) {
	if rect.Empty() {
		return ScreenshotItem{}, errors.New("screenshot rectangle is empty")
	}
	img, err := desktopscreenshot.CaptureRect(rect)
	if err != nil {
		return ScreenshotItem{}, err
	}
	if img == nil || img.Bounds().Empty() {
		return ScreenshotItem{}, errors.New("captured screenshot is empty")
	}
	return s.saveScreenshotImage(img, mode, region)
}

func (s *RecordingFreedomService) captureScrollingScreenshot(rect image.Rectangle, region *RegionRect) (ScreenshotItem, error) {
	return s.captureScrollingScreenshotWith(rect, region, desktopscreenshot.CaptureRect, scrollDownAtRect, time.Sleep)
}

func (s *RecordingFreedomService) captureScrollingScreenshotWith(rect image.Rectangle, region *RegionRect, capture screenshotCaptureFunc, scroll screenshotScrollFunc, sleep screenshotSleepFunc) (ScreenshotItem, error) {
	img, frames, scrolled, err := captureScrollingScreenshotImage(rect, capture, scroll, sleep)
	if err != nil {
		return ScreenshotItem{}, err
	}
	mode := scrollingScreenshotMode
	event := "scrolling-capture"
	if !scrolled {
		mode = "region"
		event = "scrolling-fallback-region"
	}
	item, err := s.saveScreenshotImage(img, mode, region)
	if err != nil {
		return ScreenshotItem{}, err
	}
	s.logEvent("screenshot", event, map[string]string{
		"id":     item.ID,
		"path":   item.Path,
		"width":  fmt.Sprint(item.Width),
		"height": fmt.Sprint(item.Height),
		"frames": fmt.Sprint(frames),
		"mode":   item.Mode,
	})
	return item, nil
}

func captureScrollingScreenshotImage(rect image.Rectangle, capture screenshotCaptureFunc, scroll screenshotScrollFunc, sleep screenshotSleepFunc) (*image.RGBA, int, bool, error) {
	if rect.Empty() {
		return nil, 0, false, errors.New("scrolling screenshot rectangle is empty")
	}
	first, err := capture(rect)
	if err != nil {
		return nil, 0, false, err
	}
	if first == nil || first.Bounds().Empty() {
		return nil, 0, false, errors.New("captured scrolling screenshot frame is empty")
	}
	stitched := cloneToRGBA(first)
	previous := cloneToRGBA(first)
	frames := 1
	appendedTotal := 0
	stableFrames := 0
	for frames < scrollingScreenshotMaxFrames && stitched.Bounds().Dy() < scrollingScreenshotMaxHeight {
		if err := scroll(rect); err != nil {
			return nil, frames, false, err
		}
		if sleep != nil {
			sleep(scrollingScreenshotScrollDelay)
		}
		next, err := capture(rect)
		if err != nil {
			return nil, frames, false, err
		}
		if next == nil || next.Bounds().Empty() {
			return nil, frames, false, errors.New("captured scrolling screenshot frame is empty")
		}
		frames++
		if previous.Bounds().Dx() != next.Bounds().Dx() || previous.Bounds().Dy() != next.Bounds().Dy() {
			return nil, frames, false, fmt.Errorf("scrolling screenshot frame size changed from %dx%d to %dx%d", previous.Bounds().Dx(), previous.Bounds().Dy(), next.Bounds().Dx(), next.Bounds().Dy())
		}
		nextFrame := cloneToRGBA(next)
		if framesNearlyEqual(previous, nextFrame) {
			break
		}
		var appended int
		stitched, appended = appendScrollingFrame(stitched, previous, nextFrame, scrollingScreenshotMaxHeight)
		if appended > 0 {
			appendedTotal += appended
		}
		if appended <= scrollingScreenshotMinAppend {
			stableFrames++
			if stableFrames >= 2 {
				break
			}
		} else {
			stableFrames = 0
		}
		previous = nextFrame
	}
	if appendedTotal == 0 {
		return stitched, frames, false, nil
	}
	return stitched, frames, true, nil
}

func appendScrollingFrame(stitched *image.RGBA, previous *image.RGBA, next *image.RGBA, maxHeight int) (*image.RGBA, int) {
	if stitched == nil || previous == nil || next == nil {
		return stitched, 0
	}
	width := stitched.Bounds().Dx()
	if width <= 0 || next.Bounds().Dx() != width {
		return stitched, 0
	}
	overlap := detectScrollingOverlap(previous, next)
	startY := overlap
	if startY < 0 {
		startY = 0
	}
	if startY >= next.Bounds().Dy() {
		return stitched, 0
	}
	appendHeight := next.Bounds().Dy() - startY
	currentHeight := stitched.Bounds().Dy()
	if maxHeight > 0 && currentHeight+appendHeight > maxHeight {
		appendHeight = maxHeight - currentHeight
	}
	if appendHeight <= 0 {
		return stitched, 0
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, currentHeight+appendHeight))
	draw.Draw(dst, image.Rect(0, 0, width, currentHeight), stitched, stitched.Bounds().Min, draw.Src)
	draw.Draw(dst, image.Rect(0, currentHeight, width, currentHeight+appendHeight), next, image.Point{X: next.Bounds().Min.X, Y: next.Bounds().Min.Y + startY}, draw.Src)
	return dst, appendHeight
}

func detectScrollingOverlap(previous *image.RGBA, next *image.RGBA) int {
	if previous == nil || next == nil {
		return 0
	}
	width := minInt(previous.Bounds().Dx(), next.Bounds().Dx())
	height := minInt(previous.Bounds().Dy(), next.Bounds().Dy())
	if width <= 0 || height < 24 {
		return 0
	}
	minShift := maxInt(6, height/12)
	maxShift := maxInt(minShift, height*9/10)
	bestShift := 0
	bestScore := int64(1<<62 - 1)
	for shift := minShift; shift <= maxShift; shift += 4 {
		score := overlapAverageDiff(previous, next, shift, width, height)
		if score < bestScore {
			bestScore = score
			bestShift = shift
		}
	}
	if bestShift == 0 {
		return 0
	}
	for shift := maxInt(minShift, bestShift-3); shift <= minInt(maxShift, bestShift+3); shift++ {
		score := overlapAverageDiff(previous, next, shift, width, height)
		if score < bestScore {
			bestScore = score
			bestShift = shift
		}
	}
	if bestScore > 18 {
		return 0
	}
	return height - bestShift
}

func overlapAverageDiff(previous *image.RGBA, next *image.RGBA, shift int, width int, height int) int64 {
	overlap := height - shift
	if overlap <= 0 {
		return 1<<62 - 1
	}
	stepX := maxInt(1, width/80)
	stepY := maxInt(1, overlap/80)
	var total int64
	var samples int64
	prevBounds := previous.Bounds()
	nextBounds := next.Bounds()
	for y := 0; y < overlap; y += stepY {
		for x := 0; x < width; x += stepX {
			total += int64(colorDistance(previous.At(prevBounds.Min.X+x, prevBounds.Min.Y+shift+y), next.At(nextBounds.Min.X+x, nextBounds.Min.Y+y)))
			samples++
		}
	}
	if samples == 0 {
		return 1<<62 - 1
	}
	return total / samples
}

func framesNearlyEqual(previous *image.RGBA, next *image.RGBA) bool {
	if previous == nil || next == nil {
		return false
	}
	if previous.Bounds().Dx() != next.Bounds().Dx() || previous.Bounds().Dy() != next.Bounds().Dy() {
		return false
	}
	score := overlapAverageDiff(previous, next, 0, previous.Bounds().Dx(), previous.Bounds().Dy())
	return score <= 2
}

func colorDistance(a color.Color, b color.Color) int {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return absInt(int(ar>>8)-int(br>>8)) +
		absInt(int(ag>>8)-int(bg>>8)) +
		absInt(int(ab>>8)-int(bb>>8)) +
		absInt(int(aa>>8)-int(ba>>8))
}

func cloneToRGBA(src image.Image) *image.RGBA {
	if src == nil || src.Bounds().Empty() {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)
	return dst
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (s *RecordingFreedomService) saveScreenshotImage(img image.Image, mode string, region *RegionRect) (ScreenshotItem, error) {
	if img == nil || img.Bounds().Empty() {
		return ScreenshotItem{}, errors.New("screenshot image is empty")
	}
	now := time.Now().UTC()
	id := "screenshot-" + now.Format("20060102-150405.000000000")
	dir, err := s.screenshotDir()
	if err != nil {
		return ScreenshotItem{}, err
	}
	thumbDir := filepath.Join(dir, "thumbnails")
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return ScreenshotItem{}, err
	}
	path := filepath.Join(dir, id+".png")
	thumbPath := filepath.Join(thumbDir, id+".png")
	if err := writePNG(path, img); err != nil {
		return ScreenshotItem{}, err
	}
	if err := writePNG(thumbPath, screenshotThumbnail(img)); err != nil {
		return ScreenshotItem{}, err
	}
	if mode == "" {
		mode = "full"
	}
	item := ScreenshotItem{
		ID:            id,
		Path:          path,
		ThumbnailPath: thumbPath,
		CreatedAt:     now.Format(time.RFC3339Nano),
		Width:         img.Bounds().Dx(),
		Height:        img.Bounds().Dy(),
		Mode:          mode,
		Region:        region,
		OCRStatus:     "none",
	}
	items, err := s.loadScreenshotHistory()
	if err != nil {
		return ScreenshotItem{}, err
	}
	items = append([]ScreenshotItem{item}, items...)
	if len(items) > 200 {
		items = items[:200]
	}
	if err := s.saveScreenshotHistory(items); err != nil {
		return ScreenshotItem{}, err
	}
	s.emitScreenshotHistoryChanged(items)
	s.logEvent("screenshot", "capture", map[string]string{
		"id":     item.ID,
		"path":   item.Path,
		"width":  fmt.Sprint(item.Width),
		"height": fmt.Sprint(item.Height),
		"mode":   item.Mode,
	})
	s.queueScreenshotOCRAfterSave(item)
	return item, nil
}

func screenshotImageDataURL(img image.Image) (string, error) {
	if img == nil || img.Bounds().Empty() {
		return "", errors.New("screenshot image is empty")
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", err
	}
	return whiteboardPNGContentPrefix + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func screenshotAnnotationTargetFromRect(bounds application.Rect, item ScreenshotItem) recpackage.ManifestAnnotationTarget {
	geometry := recpackage.ManifestSourceGeometry{
		X:      bounds.X,
		Y:      bounds.Y,
		Width:  bounds.Width,
		Height: bounds.Height,
	}
	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = screenshotAnnotationTargetID
	}
	return recpackage.ManifestAnnotationTarget{
		Type:     screenshotAnnotationTargetType,
		ID:       id,
		Geometry: &geometry,
	}
}

func (s *RecordingFreedomService) screenshotImagePath(req ScreenshotImageRequest) (string, error) {
	item, err := s.screenshotItemForRequest(req)
	if err != nil {
		return "", err
	}
	path := item.Path
	if req.Thumbnail && strings.TrimSpace(item.ThumbnailPath) != "" {
		path = item.ThumbnailPath
	}
	return managedScreenshotPath(s, path)
}

func (s *RecordingFreedomService) screenshotItemForRequest(req ScreenshotImageRequest) (ScreenshotItem, error) {
	if strings.TrimSpace(req.ID) != "" {
		return s.screenshotItemByID(req.ID)
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return ScreenshotItem{}, errors.New("screenshot id or path is required")
	}
	managed, err := managedScreenshotPath(s, path)
	if err != nil {
		return ScreenshotItem{}, err
	}
	return ScreenshotItem{
		ID:   strings.TrimSuffix(filepath.Base(managed), filepath.Ext(managed)),
		Path: managed,
	}, nil
}

func (s *RecordingFreedomService) screenshotItemByID(id string) (ScreenshotItem, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ScreenshotItem{}, errors.New("screenshot id is required")
	}
	items, err := s.loadScreenshotHistory()
	if err != nil {
		return ScreenshotItem{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return ScreenshotItem{}, fmt.Errorf("screenshot %q was not found", id)
}

func (s *RecordingFreedomService) loadScreenshotHistory() ([]ScreenshotItem, error) {
	path, err := s.screenshotHistoryPath()
	if err != nil {
		return nil, err
	}
	data, err := readFileWithTransientRetry(path)
	if errors.Is(err, os.ErrNotExist) {
		return []ScreenshotItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	var store screenshotHistoryStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	items := normalizeScreenshotHistory(store.Items)
	return items, nil
}

func (s *RecordingFreedomService) saveScreenshotHistory(items []ScreenshotItem) error {
	path, err := s.screenshotHistoryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	store := screenshotHistoryStore{
		SchemaVersion: 1,
		Items:         normalizeScreenshotHistory(items),
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(data, '\n'), 0o644)
}

func normalizeScreenshotHistory(items []ScreenshotItem) []ScreenshotItem {
	normalized := make([]ScreenshotItem, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Path = strings.TrimSpace(item.Path)
		if item.ID == "" || item.Path == "" || seen[item.ID] {
			continue
		}
		if item.Mode == "" {
			item.Mode = "region"
		}
		item.Pinned = false
		item.OCRStatus = normalizeScreenshotOCRStatus(item.OCRStatus)
		if item.OCRStatus == "none" {
			item.OCRResultID = ""
			item.OCRModelID = ""
			item.OCRUpdatedAt = ""
			item.OCRError = ""
		}
		seen[item.ID] = true
		normalized = append(normalized, item)
	}
	sort.SliceStable(normalized, func(left, right int) bool {
		return normalized[left].CreatedAt > normalized[right].CreatedAt
	})
	return normalized
}

func normalizeScreenshotOCRStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "queued", "running", "ready", "failed":
		return normalized
	default:
		return "none"
	}
}

func managedScreenshotPath(s *RecordingFreedomService, path string) (string, error) {
	if s == nil {
		return "", errors.New("screenshot service is not initialized")
	}
	raw := strings.TrimSpace(path)
	if raw == "" {
		return "", errors.New("screenshot path is required")
	}
	dir, err := s.screenshotDir()
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	candidates := []string{raw}
	if !filepath.IsAbs(raw) {
		clean := filepath.Clean(raw)
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("screenshot path %q must stay inside %q", path, root)
		}
		dataDir := filepath.Dir(root)
		appRoot := filepath.Dir(dataDir)
		candidates = []string{
			filepath.Join(root, clean),
			filepath.Join(root, filepath.Base(clean)),
			filepath.Join(dataDir, clean),
			filepath.Join(appRoot, clean),
		}
	}
	var firstValid string
	var firstExisting string
	var firstExistingDir string
	for _, candidate := range candidates {
		target, ok, err := managedScreenshotCandidate(root, candidate)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		if firstValid == "" {
			firstValid = target
		}
		if _, err := os.Stat(target); err == nil {
			firstExisting = target
			break
		}
		if firstExistingDir == "" {
			if info, err := os.Stat(filepath.Dir(target)); err == nil && info.IsDir() {
				firstExistingDir = target
			}
		}
	}
	if firstExisting != "" {
		return firstExisting, nil
	}
	if firstExistingDir != "" {
		return firstExistingDir, nil
	}
	if firstValid != "" {
		return firstValid, nil
	}
	return "", fmt.Errorf("screenshot path %q must stay inside %q", path, root)
}

func managedScreenshotCandidate(root string, candidate string) (string, bool, error) {
	target, err := filepath.Abs(candidate)
	if err != nil {
		return "", false, err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", false, err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false, nil
	}
	if strings.ToLower(filepath.Ext(target)) != ".png" {
		return "", false, fmt.Errorf("screenshot path %q must be a PNG", candidate)
	}
	return target, true, nil
}

func uniqueScreenshotFilePaths(item ScreenshotItem) []string {
	seen := map[string]bool{}
	paths := make([]string, 0, 2)
	for _, path := range []string{item.Path, item.ThumbnailPath} {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths
}

func removeManagedScreenshotFile(s *RecordingFreedomService, path string) error {
	managed, err := managedScreenshotPath(s, path)
	if err != nil {
		return err
	}
	if err := os.Remove(managed); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *RecordingFreedomService) screenshotHistoryPath() (string, error) {
	dir, err := s.screenshotDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, screenshotHistoryFileName), nil
}

func (s *RecordingFreedomService) screenshotDir() (string, error) {
	if s == nil || s.appData == nil {
		return "", errors.New("app data service is not initialized")
	}
	root, err := s.appData.RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "data", screenshotDirName), nil
}

func screenshotCaptureUnionBounds() RegionRect {
	count := desktopscreenshot.NumActiveDisplays()
	if count <= 0 {
		return RegionRect{}
	}
	union := desktopscreenshot.GetDisplayBounds(0)
	for index := 1; index < count; index++ {
		union = union.Union(desktopscreenshot.GetDisplayBounds(index))
	}
	return RegionRect{
		X:      union.Min.X,
		Y:      union.Min.Y,
		Width:  union.Dx(),
		Height: union.Dy(),
	}
}

func screenshotCaptureDisplayBounds() []RegionRect {
	count := desktopscreenshot.NumActiveDisplays()
	if count <= 0 {
		return nil
	}
	displays := make([]RegionRect, 0, count)
	for index := 0; index < count; index++ {
		bounds := desktopscreenshot.GetDisplayBounds(index)
		if bounds.Empty() {
			continue
		}
		displays = append(displays, RegionRect{
			X:      bounds.Min.X,
			Y:      bounds.Min.Y,
			Width:  bounds.Dx(),
			Height: bounds.Dy(),
		})
	}
	return displays
}

func mapRegionSelectionToCaptureRect(session RegionSelectionSession, relative RegionRect) image.Rectangle {
	if rect, ok := mapRegionSelectionToDisplayCaptureRect(session, relative); ok {
		return rect
	}
	capture := session.CaptureBounds
	if capture == nil || capture.Width <= 0 || capture.Height <= 0 || session.Bounds.Width <= 0 || session.Bounds.Height <= 0 {
		absolute := application.Rect{
			X:      session.Bounds.X + relative.X,
			Y:      session.Bounds.Y + relative.Y,
			Width:  relative.Width,
			Height: relative.Height,
		}
		return image.Rect(absolute.X, absolute.Y, absolute.X+absolute.Width, absolute.Y+absolute.Height)
	}
	x := capture.X + scaleRegionValue(relative.X, session.Bounds.Width, capture.Width)
	y := capture.Y + scaleRegionValue(relative.Y, session.Bounds.Height, capture.Height)
	width := scaleRegionValue(relative.Width, session.Bounds.Width, capture.Width)
	height := scaleRegionValue(relative.Height, session.Bounds.Height, capture.Height)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return image.Rect(x, y, x+width, y+height)
}

func scaleRegionValue(value int, sourceSize int, targetSize int) int {
	if sourceSize <= 0 || targetSize <= 0 {
		return value
	}
	return int(float64(value)*float64(targetSize)/float64(sourceSize) + 0.5)
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func screenshotThumbnail(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= screenshotThumbnailMaxSide && height <= screenshotThumbnailMaxSide {
		return img
	}
	scale := float64(screenshotThumbnailMaxSide) / float64(width)
	if height > width {
		scale = float64(screenshotThumbnailMaxSide) / float64(height)
	}
	nextWidth := int(float64(width)*scale + 0.5)
	nextHeight := int(float64(height)*scale + 0.5)
	if nextWidth < 1 {
		nextWidth = 1
	}
	if nextHeight < 1 {
		nextHeight = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nextWidth, nextHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

func screenshotPinWindowBounds(item ScreenshotItem) application.Rect {
	width := item.Width
	height := item.Height
	if width <= 0 {
		width = 480
	}
	if height <= 0 {
		height = 320
	}
	maxWidth := 720
	maxHeight := 520
	scale := 1.0
	if width > maxWidth {
		scale = float64(maxWidth) / float64(width)
	}
	if int(float64(height)*scale) > maxHeight {
		scale = float64(maxHeight) / float64(height)
	}
	return application.Rect{
		X:      120,
		Y:      120,
		Width:  int(float64(width)*scale) + 24,
		Height: int(float64(height)*scale) + 52,
	}
}

func (s *RecordingFreedomService) broadcastScreenshotPinState(state ScreenshotPinState) {
	if s.screenshotPinWindow == nil {
		return
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	s.screenshotPinWindow.ExecJS(fmt.Sprintf(
		"window.__RF_SCREENSHOT_PIN__=%s;window.dispatchEvent(new CustomEvent('rf-screenshot-pin',{detail:window.__RF_SCREENSHOT_PIN__}));",
		string(payload),
	))
}

func (s *RecordingFreedomService) broadcastScreenshotWhiteboardContext(context ScreenshotWhiteboardContext) {
	if s.whiteboardWindow == nil {
		return
	}
	payload, err := json.Marshal(context)
	if err != nil {
		return
	}
	s.whiteboardWindow.ExecJS(fmt.Sprintf(
		"window.__RF_SCREENSHOT_WHITEBOARD__=%s;window.dispatchEvent(new CustomEvent('rf-screenshot-whiteboard',{detail:window.__RF_SCREENSHOT_WHITEBOARD__}));",
		string(payload),
	))
}

func (s *RecordingFreedomService) emitScreenshotCaptured(item ScreenshotItem) {
	if s == nil || s.app == nil {
		return
	}
	s.app.Event.Emit("screenshot.captured", ScreenshotCapturedEvent{Item: item})
}

func (s *RecordingFreedomService) emitScreenshotHistoryChanged(items []ScreenshotItem) {
	if s == nil || s.app == nil {
		return
	}
	s.app.Event.Emit("screenshot.history.changed", ScreenshotHistoryResult{Items: normalizeScreenshotHistory(items)})
}

func (s *RecordingFreedomService) emitScreenshotPin(state ScreenshotPinState) {
	if s == nil || s.app == nil {
		return
	}
	s.app.Event.Emit("screenshot.pin", ScreenshotPinEvent{
		Visible: state.Visible,
		Item:    state.Item,
		DataURL: state.DataURL,
		Fixed:   state.Fixed,
	})
}

func (s *RecordingFreedomService) emitScreenshotWhiteboardContext(context ScreenshotWhiteboardContext) {
	if s == nil || s.app == nil {
		return
	}
	s.app.Event.Emit("screenshot.whiteboard", context)
}
