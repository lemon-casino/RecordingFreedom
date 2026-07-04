package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	desktopscreenshot "github.com/kbinani/screenshot"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/image/draw"
)

const (
	screenshotDirName                = "screenshots"
	screenshotHistoryFileName        = "history.json"
	screenshotMaxPreviewBytes        = 16 * 1024 * 1024
	screenshotThumbnailMaxSide       = 320
	regionSelectionPurposeScreenshot = "screenshot"
	annotationOverlayModeScreenshot  = "screenshot"
	screenshotAnnotationTargetType   = "screenshot-region"
	screenshotAnnotationTargetID     = "screenshot:region"
)

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
		MinimumWidth:  minRegionWidth,
		MinimumHeight: minRegionHeight,
		DisplayCount:  displayCount,
		Purpose:       regionSelectionPurposeScreenshot,
	}
	s.regionMu.Lock()
	s.regionSession = &session
	s.regionMu.Unlock()
	s.regionOverlay.SetBounds(bounds)
	s.regionOverlay.SetAlwaysOnTop(true)
	s.regionOverlay.Show()
	s.regionOverlay.SetBounds(bounds)
	s.regionOverlay.Focus()
	if payload, err := json.Marshal(session); err == nil {
		s.regionOverlay.ExecJS(fmt.Sprintf(
			"window.__RF_REGION_SESSION__=%s;window.dispatchEvent(new CustomEvent('rf-region-session',{detail:window.__RF_REGION_SESSION__}));",
			string(payload),
		))
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
			items[index].Pinned = *req.Pinned
			changed = true
			if !*req.Pinned {
				items[index].Fixed = false
			}
		}
		if req.Fixed != nil {
			items[index].Fixed = *req.Fixed
			if *req.Fixed {
				items[index].Pinned = true
			}
			changed = true
		}
		break
	}
	if !changed {
		return ScreenshotHistoryResult{}, fmt.Errorf("screenshot %q was not found", id)
	}
	if err := s.saveScreenshotHistory(items); err != nil {
		return ScreenshotHistoryResult{}, err
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
	pinned := true
	history, err := s.PatchScreenshotItem(ScreenshotItemPatchRequest{ID: item.ID, Pinned: &pinned})
	if err != nil {
		return ScreenshotPinState{}, err
	}
	for _, candidate := range history.Items {
		if candidate.ID == item.ID {
			item = candidate
			break
		}
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
	return context, nil
}

func (s *RecordingFreedomService) ConsumeScreenshotWhiteboardContext() ScreenshotWhiteboardContext {
	s.screenshotMu.Lock()
	defer s.screenshotMu.Unlock()
	context := s.whiteboardScreenshot
	s.whiteboardScreenshot = ScreenshotWhiteboardContext{}
	return context
}

func (s *RecordingFreedomService) StartScrollingScreenshot() error {
	return errors.New("scrolling screenshot requires platform scroll automation and is not enabled in this build")
}

func (s *RecordingFreedomService) captureScreenshotRect(req ScreenshotCaptureRequest) (image.Rectangle, *RegionRect, error) {
	mode := strings.TrimSpace(req.Mode)
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
	bounds := screenshotCaptureUnionBounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return image.Rectangle{}, nil, errors.New("no active display is available for screenshots")
	}
	region := bounds
	return image.Rect(bounds.X, bounds.Y, bounds.X+bounds.Width, bounds.Y+bounds.Height), &region, nil
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
	s.logEvent("screenshot", "capture", map[string]string{
		"id":     item.ID,
		"path":   item.Path,
		"width":  fmt.Sprint(item.Width),
		"height": fmt.Sprint(item.Height),
		"mode":   item.Mode,
	})
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
	data, err := os.ReadFile(path)
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
		seen[item.ID] = true
		normalized = append(normalized, item)
	}
	sort.SliceStable(normalized, func(left, right int) bool {
		return normalized[left].CreatedAt > normalized[right].CreatedAt
	})
	return normalized
}

func managedScreenshotPath(s *RecordingFreedomService, path string) (string, error) {
	if s == nil {
		return "", errors.New("screenshot service is not initialized")
	}
	dir, err := s.screenshotDir()
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("screenshot path %q must stay inside %q", path, root)
	}
	if strings.ToLower(filepath.Ext(target)) != ".png" {
		return "", fmt.Errorf("screenshot path %q must be a PNG", path)
	}
	return target, nil
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

func mapRegionSelectionToCaptureRect(session RegionSelectionSession, relative RegionRect) image.Rectangle {
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

func (s *RecordingFreedomService) emitScreenshotCaptured(item ScreenshotItem) {
	if s == nil || s.app == nil {
		return
	}
	s.app.Event.Emit("screenshot.captured", ScreenshotCapturedEvent{Item: item})
}

func (s *RecordingFreedomService) emitScreenshotPin(state ScreenshotPinState) {
	if s == nil || s.app == nil {
		return
	}
	s.app.Event.Emit("screenshot.pin", ScreenshotPinEvent{
		Visible: state.Visible,
		Item:    state.Item,
		Fixed:   state.Fixed,
	})
}
