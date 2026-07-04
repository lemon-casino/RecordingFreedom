package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	minRegionWidth                   = 64
	minRegionHeight                  = 64
	regionSelectionPurposeCapture    = "capture"
	regionSelectionPurposeAnnotation = "annotation"
)

type RegionRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type RegionSelectionSession struct {
	ID            string     `json:"id"`
	Bounds        RegionRect `json:"bounds"`
	MinimumWidth  int        `json:"minimumWidth"`
	MinimumHeight int        `json:"minimumHeight"`
	DisplayCount  int        `json:"displayCount"`
	Purpose       string     `json:"purpose,omitempty"`
}

type RegionSelectionRequest struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type RegionSelectionResult struct {
	SessionID string                `json:"sessionId,omitempty"`
	Source    devices.CaptureSource `json:"source,omitempty"`
	Geometry  RegionRect            `json:"geometry,omitempty"`
	Cancelled bool                  `json:"cancelled"`
	Error     string                `json:"error,omitempty"`
}

type RegionFrameState struct {
	Bounds        RegionRect `json:"bounds"`
	OverlayBounds RegionRect `json:"overlayBounds,omitempty"`
	Mode          string     `json:"mode"`
}

func (s *RecordingFreedomService) ShowRegionSelector() (RegionSelectionSession, error) {
	if recorderIsActive(s.recorder.State()) {
		return RegionSelectionSession{}, errors.New("cannot select a region while recording is active")
	}
	if s.app == nil || s.regionOverlay == nil {
		return RegionSelectionSession{}, errors.New("region overlay window is not configured")
	}

	bounds, displayCount := regionOverlayBounds(s.app.Screen.GetAll())
	session := RegionSelectionSession{
		ID:            fmt.Sprintf("region-%d", time.Now().UnixNano()),
		Bounds:        regionRectFromAppRect(bounds),
		MinimumWidth:  minRegionWidth,
		MinimumHeight: minRegionHeight,
		DisplayCount:  displayCount,
		Purpose:       regionSelectionPurposeCapture,
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
	return session, nil
}

func (s *RecordingFreedomService) CompleteRegionSelection(req RegionSelectionRequest) (RegionSelectionResult, error) {
	s.regionMu.Lock()
	session := s.regionSession
	s.regionSession = nil
	s.regionMu.Unlock()
	if session == nil {
		return RegionSelectionResult{}, errors.New("no active region selection session")
	}
	if session.Purpose != "" && session.Purpose != regionSelectionPurposeCapture {
		return RegionSelectionResult{}, errors.New("active region selection session is not for capture")
	}

	relative := normalizeRegionSelection(req)
	if relative.Width < session.MinimumWidth || relative.Height < session.MinimumHeight {
		result := RegionSelectionResult{
			SessionID: session.ID,
			Cancelled: true,
			Error:     fmt.Sprintf("selected region must be at least %d x %d", session.MinimumWidth, session.MinimumHeight),
		}
		s.emitRegionSelection(result)
		return result, errors.New(result.Error)
	}

	absoluteDIP := application.Rect{
		X:      session.Bounds.X + relative.X,
		Y:      session.Bounds.Y + relative.Y,
		Width:  relative.Width,
		Height: relative.Height,
	}
	s.setSelectedRegionDIP(absoluteDIP)
	displayMatchRect := application.DipToPhysicalRect(absoluteDIP)
	if displayMatchRect.Width <= 0 || displayMatchRect.Height <= 0 {
		displayMatchRect = absoluteDIP
	}
	captureRect := absoluteDIP
	if runtime.GOOS == "windows" {
		captureRect = displayMatchRect
	}

	source := regionCaptureSource(captureRect, regionDisplayForRect(displayMatchRect, s.devices.ListSources(), s.app.Screen.GetAll()))
	result := RegionSelectionResult{
		SessionID: session.ID,
		Source:    source,
		Geometry:  regionRectFromAppRect(captureRect),
	}
	_ = s.showRegionEditor(absoluteDIP)
	s.emitRegionSelection(result)
	return result, nil
}

func (s *RecordingFreedomService) CancelRegionSelection() RegionSelectionResult {
	s.regionMu.Lock()
	session := s.regionSession
	s.regionSession = nil
	s.regionMu.Unlock()

	result := RegionSelectionResult{Cancelled: true}
	if session != nil {
		result.SessionID = session.ID
	}
	if session == nil || session.Purpose == "" || session.Purpose == regionSelectionPurposeCapture {
		s.clearSelectedRegionDIP()
		s.emitRegionSelection(result)
	}
	if s.regionOverlay != nil {
		s.regionOverlay.Hide()
	}
	return result
}

func (s *RecordingFreedomService) UpdateSelectedRegion(req RegionSelectionRequest) (RegionSelectionResult, error) {
	if recorderIsActive(s.recorder.State()) {
		return RegionSelectionResult{}, errors.New("cannot edit a region while recording is active")
	}
	absoluteDIP := normalizeRegionSelection(req)
	if absoluteDIP.Width < minRegionWidth || absoluteDIP.Height < minRegionHeight {
		return RegionSelectionResult{}, fmt.Errorf("selected region must be at least %d x %d", minRegionWidth, minRegionHeight)
	}
	s.setSelectedRegionDIP(absoluteDIP)
	result := s.regionResultFromAbsoluteDIP(absoluteDIP)
	_ = s.showRegionEditor(absoluteDIP)
	s.emitRegionSelection(result)
	return result, nil
}

func (s *RecordingFreedomService) CancelSelectedRegion() RegionSelectionResult {
	result := RegionSelectionResult{Cancelled: true}
	s.clearSelectedRegionDIP()
	_ = s.HideRegionFrame()
	if s.regionOverlay != nil {
		s.regionOverlay.Hide()
	}
	s.emitRegionSelection(result)
	return result
}

func (s *RecordingFreedomService) setSelectedRegionDIP(bounds application.Rect) {
	s.regionMu.Lock()
	defer s.regionMu.Unlock()
	s.selectedRegionDIP = bounds
}

func (s *RecordingFreedomService) clearSelectedRegionDIP() {
	s.regionMu.Lock()
	defer s.regionMu.Unlock()
	s.selectedRegionDIP = application.Rect{}
}

func (s *RecordingFreedomService) selectedRegionDisplayBounds() application.Rect {
	s.regionMu.Lock()
	defer s.regionMu.Unlock()
	return s.selectedRegionDIP
}

func (s *RecordingFreedomService) HideRegionFrame() error {
	if s.regionOverlay != nil {
		s.regionOverlay.Hide()
	}
	return nil
}

func (s *RecordingFreedomService) showRegionFrame(bounds application.Rect) error {
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return nil
	}
	overlayBounds := application.Rect{}
	if s.regionOverlay != nil && s.app != nil {
		overlayBounds, _ = regionOverlayBounds(s.app.Screen.GetAll())
		s.regionOverlay.SetIgnoreMouseEvents(true)
		s.regionOverlay.SetAlwaysOnTop(true)
		s.regionOverlay.SetBounds(overlayBounds)
		s.regionOverlay.Show()
		s.regionOverlay.SetBounds(overlayBounds)
	}
	state := RegionFrameState{
		Bounds:        regionRectFromAppRect(bounds),
		OverlayBounds: regionRectFromAppRect(overlayBounds),
		Mode:          "recording",
	}
	s.broadcastRegionFrameState(state)
	return nil
}

func (s *RecordingFreedomService) showRegionEditor(bounds application.Rect) error {
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return nil
	}
	overlayBounds := application.Rect{}
	if s.regionOverlay != nil && s.app != nil {
		overlayBounds, _ = regionOverlayBounds(s.app.Screen.GetAll())
		s.regionOverlay.SetIgnoreMouseEvents(false)
		s.regionOverlay.SetAlwaysOnTop(true)
		s.regionOverlay.SetBounds(overlayBounds)
		s.regionOverlay.Show()
		s.regionOverlay.SetBounds(overlayBounds)
	}
	state := RegionFrameState{
		Bounds:        regionRectFromAppRect(bounds),
		OverlayBounds: regionRectFromAppRect(overlayBounds),
		Mode:          "edit",
	}
	s.broadcastRegionFrameState(state)
	if s.capsuleWindow != nil {
		s.capsuleWindow.SetAlwaysOnTop(true)
		s.capsuleWindow.Show()
	}
	return nil
}

func (s *RecordingFreedomService) broadcastRegionFrameState(state RegionFrameState) {
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	script := fmt.Sprintf(
		"window.__RF_REGION_FRAME__=%s;window.dispatchEvent(new CustomEvent('rf-region-frame',{detail:window.__RF_REGION_FRAME__}));",
		string(payload),
	)
	if s.regionOverlay != nil {
		s.regionOverlay.ExecJS(script)
	}
}

func (s *RecordingFreedomService) regionResultFromAbsoluteDIP(absoluteDIP application.Rect) RegionSelectionResult {
	displayMatchRect := application.DipToPhysicalRect(absoluteDIP)
	if displayMatchRect.Width <= 0 || displayMatchRect.Height <= 0 {
		displayMatchRect = absoluteDIP
	}
	captureRect := absoluteDIP
	if runtime.GOOS == "windows" {
		captureRect = displayMatchRect
	}
	source := regionCaptureSource(captureRect, regionDisplayForRect(displayMatchRect, s.devices.ListSources(), s.app.Screen.GetAll()))
	return RegionSelectionResult{
		Source:   source,
		Geometry: regionRectFromAppRect(captureRect),
	}
}

func (s *RecordingFreedomService) emitRegionSelection(result RegionSelectionResult) {
	if s.app == nil {
		return
	}
	s.app.Event.Emit("capture.region.selected", result)
}

func normalizeRegionSelection(req RegionSelectionRequest) application.Rect {
	x := req.X
	y := req.Y
	width := req.Width
	height := req.Height
	if width < 0 {
		x += width
		width = -width
	}
	if height < 0 {
		y += height
		height = -height
	}
	return application.Rect{X: x, Y: y, Width: width, Height: height}
}

func regionCaptureSource(rect application.Rect, display devices.CaptureSource) devices.CaptureSource {
	source := devices.CaptureSource{
		ID:                "region:custom",
		Type:              devices.SourceRegion,
		Name:              "Custom Region",
		Subtitle:          fmt.Sprintf("%d x %d selected region · %d,%d", rect.Width, rect.Height, rect.X, rect.Y),
		X:                 rect.X,
		Y:                 rect.Y,
		Width:             rect.Width,
		Height:            rect.Height,
		NativeID:          "region:virtual-desktop",
		Capability:        devices.CapabilityNativeQueued,
		UnavailableReason: "region crop writer is queued; the selected geometry is preserved for source.geometry",
	}
	if display.ID == "" {
		source.UnavailableReason = "region spans multiple displays or no display target was found; multi-display region crop writer is queued"
		if runtime.GOOS == "windows" {
			source.Available = true
			source.Capability = devices.CapabilityEnumerated
			source.UnavailableReason = ""
			source.Subtitle = fmt.Sprintf("%d x %d selected region on virtual desktop · %d,%d", rect.Width, rect.Height, rect.X, rect.Y)
		}
		return source
	}
	source.DisplayIndex = display.DisplayIndex
	source.NativeID = display.NativeID
	if source.NativeID == "" {
		source.NativeID = display.ID
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		source.Available = true
		source.Capability = devices.CapabilityEnumerated
		source.UnavailableReason = ""
		if display.ID != "" {
			source.Subtitle = fmt.Sprintf("%d x %d selected region on display %d · %d,%d", rect.Width, rect.Height, display.DisplayIndex, rect.X, rect.Y)
		} else {
			source.Subtitle = fmt.Sprintf("%d x %d selected region on virtual desktop · %d,%d", rect.Width, rect.Height, rect.X, rect.Y)
		}
	}
	return source
}

func regionDisplayForRect(rect application.Rect, sources []devices.CaptureSource, screens []*application.Screen) devices.CaptureSource {
	if rect.Width <= 0 || rect.Height <= 0 {
		return devices.CaptureSource{}
	}
	for index, screen := range screens {
		if screen == nil || screen.PhysicalBounds.Width <= 0 || screen.PhysicalBounds.Height <= 0 {
			continue
		}
		if !rectContains(screen.PhysicalBounds, rect) {
			continue
		}
		if source := sourceDisplayByIndex(sources, index+1); source.ID != "" {
			return source
		}
	}
	for _, source := range sources {
		if source.Type != devices.SourceScreen || source.Width <= 0 || source.Height <= 0 {
			continue
		}
		display := application.Rect{
			X:      source.X,
			Y:      source.Y,
			Width:  source.Width,
			Height: source.Height,
		}
		if rectContains(display, rect) {
			return source
		}
	}
	return devices.CaptureSource{}
}

func sourceDisplayByIndex(sources []devices.CaptureSource, displayIndex int) devices.CaptureSource {
	if displayIndex <= 0 {
		return devices.CaptureSource{}
	}
	for _, source := range sources {
		if source.Type == devices.SourceScreen && source.DisplayIndex == displayIndex {
			return source
		}
	}
	return devices.CaptureSource{}
}

func rectContains(outer application.Rect, inner application.Rect) bool {
	return inner.X >= outer.X &&
		inner.Y >= outer.Y &&
		inner.X+inner.Width <= outer.X+outer.Width &&
		inner.Y+inner.Height <= outer.Y+outer.Height
}

func regionOverlayBounds(screens []*application.Screen) (application.Rect, int) {
	if len(screens) == 0 {
		return application.Rect{Width: 1280, Height: 720}, 0
	}

	minX := screens[0].Bounds.X
	minY := screens[0].Bounds.Y
	maxX := screens[0].Bounds.X + screens[0].Bounds.Width
	maxY := screens[0].Bounds.Y + screens[0].Bounds.Height
	for _, screen := range screens[1:] {
		bounds := screen.Bounds
		if bounds.X < minX {
			minX = bounds.X
		}
		if bounds.Y < minY {
			minY = bounds.Y
		}
		if bounds.X+bounds.Width > maxX {
			maxX = bounds.X + bounds.Width
		}
		if bounds.Y+bounds.Height > maxY {
			maxY = bounds.Y + bounds.Height
		}
	}
	width := maxX - minX
	height := maxY - minY
	if width <= 0 {
		width = 1280
	}
	if height <= 0 {
		height = 720
	}
	return application.Rect{X: minX, Y: minY, Width: width, Height: height}, len(screens)
}

func regionRectFromAppRect(rect application.Rect) RegionRect {
	return RegionRect{
		X:      rect.X,
		Y:      rect.Y,
		Width:  rect.Width,
		Height: rect.Height,
	}
}
