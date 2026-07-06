package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type FloatingPanelKind string

const (
	FloatingPanelSource    FloatingPanelKind = "source"
	FloatingPanelAudio     FloatingPanelKind = "audio"
	FloatingPanelCamera    FloatingPanelKind = "camera"
	FloatingPanelBoard     FloatingPanelKind = "board"
	FloatingPanelLanguage  FloatingPanelKind = "language"
	FloatingPanelSettings  FloatingPanelKind = "settings"
	FloatingPanelClose     FloatingPanelKind = "close"
	FloatingPanelOCRResult FloatingPanelKind = "ocr-result"
)

type FloatingRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FloatingPanelRequest struct {
	Kind      FloatingPanelKind `json:"kind"`
	Anchor    FloatingRect      `json:"anchor"`
	Bounds    FloatingRect      `json:"bounds"`
	DockSide  string            `json:"dockSide,omitempty"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	MinWidth  int               `json:"minWidth,omitempty"`
	MaxHeight int               `json:"maxHeight,omitempty"`
	Token     uint64            `json:"token"`
	ScreenID  string            `json:"screenId,omitempty"`
	Direction string            `json:"direction,omitempty"`
	ContextID string            `json:"contextId,omitempty"`
}

type FloatingPanelState struct {
	Visible   bool              `json:"visible"`
	Kind      FloatingPanelKind `json:"kind,omitempty"`
	Anchor    FloatingRect      `json:"anchor"`
	Bounds    FloatingRect      `json:"bounds"`
	DockSide  string            `json:"dockSide,omitempty"`
	Token     uint64            `json:"token"`
	ScreenID  string            `json:"screenId,omitempty"`
	Direction string            `json:"direction,omitempty"`
	ContextID string            `json:"contextId,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type FloatingSelectOption struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled,omitempty"`
	Swatch   string `json:"swatch,omitempty"`
}

type FloatingSelectRequest struct {
	ID         string                 `json:"id"`
	Anchor     FloatingRect           `json:"anchor"`
	Bounds     FloatingRect           `json:"bounds"`
	Value      string                 `json:"value"`
	Options    []FloatingSelectOption `json:"options"`
	Token      uint64                 `json:"token"`
	PanelToken uint64                 `json:"panelToken,omitempty"`
	Width      int                    `json:"width,omitempty"`
	MaxHeight  int                    `json:"maxHeight,omitempty"`
	ScreenID   string                 `json:"screenId,omitempty"`
	Direction  string                 `json:"direction,omitempty"`
}

type FloatingSelectState struct {
	Visible    bool                   `json:"visible"`
	ID         string                 `json:"id,omitempty"`
	Anchor     FloatingRect           `json:"anchor"`
	Bounds     FloatingRect           `json:"bounds"`
	Value      string                 `json:"value,omitempty"`
	Options    []FloatingSelectOption `json:"options,omitempty"`
	Token      uint64                 `json:"token"`
	PanelToken uint64                 `json:"panelToken,omitempty"`
	ScreenID   string                 `json:"screenId,omitempty"`
	Direction  string                 `json:"direction,omitempty"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type FloatingSelectChosenEvent struct {
	ID         string `json:"id"`
	Value      string `json:"value"`
	Token      uint64 `json:"token"`
	PanelToken uint64 `json:"panelToken,omitempty"`
}

type SourceGeometry struct {
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	DisplayIndex int    `json:"displayIndex,omitempty"`
	NativeID     string `json:"nativeId,omitempty"`
}

type SourceControlState struct {
	RecordingMode  string          `json:"recordingMode"`
	SourceID       string          `json:"sourceId,omitempty"`
	SourceType     string          `json:"sourceType,omitempty"`
	SourceGeometry *SourceGeometry `json:"sourceGeometry,omitempty"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type SourceStatePatchRequest struct {
	RecordingMode  string          `json:"recordingMode,omitempty"`
	SourceID       string          `json:"sourceId,omitempty"`
	SourceType     string          `json:"sourceType,omitempty"`
	SourceGeometry *SourceGeometry `json:"sourceGeometry,omitempty"`
	ClearGeometry  bool            `json:"clearGeometry,omitempty"`
}

func (s *RecordingFreedomService) ShowFloatingPanel(req FloatingPanelRequest) (FloatingPanelState, error) {
	return s.showFloatingPanel(req)
}

func (s *RecordingFreedomService) UpdateFloatingPanel(req FloatingPanelRequest) (FloatingPanelState, error) {
	return s.showFloatingPanel(req)
}

func (s *RecordingFreedomService) showFloatingPanel(req FloatingPanelRequest) (FloatingPanelState, error) {
	if s == nil || s.floatingPanelWindow == nil {
		return FloatingPanelState{}, errors.New("floating panel window is not configured")
	}
	if !validFloatingPanelKind(req.Kind) {
		return FloatingPanelState{}, fmt.Errorf("unsupported floating panel kind %q", req.Kind)
	}
	bounds := normalizedFloatingBounds(req.Bounds, req.Width, req.Height, 320, 180)
	state := FloatingPanelState{
		Visible:   true,
		Kind:      req.Kind,
		Anchor:    req.Anchor,
		Bounds:    bounds,
		DockSide:  strings.TrimSpace(req.DockSide),
		Token:     req.Token,
		ScreenID:  strings.TrimSpace(req.ScreenID),
		Direction: strings.TrimSpace(req.Direction),
		ContextID: strings.TrimSpace(req.ContextID),
		UpdatedAt: time.Now(),
	}
	s.floatingMu.Lock()
	s.floatingPanelState = state
	s.floatingMu.Unlock()

	regionState := s.floatingPanelRegions.Set(fullFloatingWindowHitRegionRequest(bounds, 22))
	if err := s.applyFloatingWindowBounds(s.floatingPanelWindow, bounds, regionState, false); err != nil {
		return FloatingPanelState{}, err
	}
	s.updateFloatingOutsideClickWatcher()
	s.emitFloatingPanelChanged(state)
	s.logEvent("floating-panel", "show", map[string]string{
		"kind":      string(req.Kind),
		"token":     fmt.Sprint(req.Token),
		"bounds":    formatFloatingRect(bounds),
		"dockSide":  state.DockSide,
		"direction": state.Direction,
	})
	return state, nil
}

func (s *RecordingFreedomService) HideFloatingPanel(token uint64) error {
	if s == nil {
		return nil
	}
	_ = s.HideFloatingSelect(0)
	s.floatingMu.Lock()
	current := s.floatingPanelState
	if token != 0 && current.Token != 0 && token != current.Token {
		s.floatingMu.Unlock()
		return nil
	}
	current.Visible = false
	current.UpdatedAt = time.Now()
	s.floatingPanelState = current
	s.floatingMu.Unlock()
	if s.floatingPanelWindow != nil {
		s.floatingPanelWindow.Hide()
	}
	s.updateFloatingOutsideClickWatcher()
	s.emitFloatingPanelChanged(current)
	s.logEvent("floating-panel", "hide", map[string]string{"token": fmt.Sprint(token)})
	return nil
}

func (s *RecordingFreedomService) GetFloatingPanelState() FloatingPanelState {
	if s == nil {
		return FloatingPanelState{}
	}
	s.floatingMu.Lock()
	defer s.floatingMu.Unlock()
	return s.floatingPanelState
}

func (s *RecordingFreedomService) ShowFloatingSelect(req FloatingSelectRequest) (FloatingSelectState, error) {
	if s == nil || s.floatingSelectWindow == nil {
		return FloatingSelectState{}, errors.New("floating select window is not configured")
	}
	bounds := normalizedFloatingBounds(req.Bounds, req.Width, req.MaxHeight, 180, 44)
	options := make([]FloatingSelectOption, 0, len(req.Options))
	for _, option := range req.Options {
		if strings.TrimSpace(option.Value) == "" && strings.TrimSpace(option.Label) == "" {
			continue
		}
		options = append(options, option)
	}
	state := FloatingSelectState{
		Visible:    true,
		ID:         strings.TrimSpace(req.ID),
		Anchor:     req.Anchor,
		Bounds:     bounds,
		Value:      req.Value,
		Options:    options,
		Token:      req.Token,
		PanelToken: req.PanelToken,
		ScreenID:   strings.TrimSpace(req.ScreenID),
		Direction:  strings.TrimSpace(req.Direction),
		UpdatedAt:  time.Now(),
	}
	s.floatingMu.Lock()
	s.floatingSelectState = state
	s.floatingMu.Unlock()

	regionState := s.floatingSelectRegions.Set(fullFloatingWindowHitRegionRequest(bounds, 16))
	if err := s.applyFloatingWindowBounds(s.floatingSelectWindow, bounds, regionState, false); err != nil {
		return FloatingSelectState{}, err
	}
	s.updateFloatingOutsideClickWatcher()
	s.emitFloatingSelectChanged(state)
	s.logEvent("floating-select", "show", map[string]string{
		"id":        state.ID,
		"token":     fmt.Sprint(req.Token),
		"bounds":    formatFloatingRect(bounds),
		"direction": state.Direction,
	})
	return state, nil
}

func (s *RecordingFreedomService) HideFloatingSelect(token uint64) error {
	if s == nil {
		return nil
	}
	s.floatingMu.Lock()
	current := s.floatingSelectState
	if token != 0 && current.Token != 0 && token != current.Token {
		s.floatingMu.Unlock()
		return nil
	}
	current.Visible = false
	current.UpdatedAt = time.Now()
	s.floatingSelectState = current
	s.floatingMu.Unlock()
	if s.floatingSelectWindow != nil {
		s.floatingSelectWindow.Hide()
	}
	s.updateFloatingOutsideClickWatcher()
	s.emitFloatingSelectChanged(current)
	s.logEvent("floating-select", "hide", map[string]string{"token": fmt.Sprint(token)})
	return nil
}

func (s *RecordingFreedomService) CompleteFloatingSelect(event FloatingSelectChosenEvent) error {
	if s == nil {
		return nil
	}
	s.floatingMu.Lock()
	current := s.floatingSelectState
	if event.Token == 0 {
		event.Token = current.Token
	}
	if event.ID == "" {
		event.ID = current.ID
	}
	if event.PanelToken == 0 {
		event.PanelToken = current.PanelToken
	}
	if current.Token != 0 && event.Token != current.Token {
		s.floatingMu.Unlock()
		return nil
	}
	s.floatingMu.Unlock()
	if s.app != nil {
		s.app.Event.Emit("floating.select.chosen", event)
	}
	return s.HideFloatingSelect(event.Token)
}

func (s *RecordingFreedomService) GetFloatingSelectState() FloatingSelectState {
	if s == nil {
		return FloatingSelectState{}
	}
	s.floatingMu.Lock()
	defer s.floatingMu.Unlock()
	return s.floatingSelectState
}

func (s *RecordingFreedomService) SetFloatingPanelHitRegions(req CapsuleWindowHitRegionsRequest) error {
	state, changed := s.floatingPanelRegions.Update(req)
	if !changed && !req.Force {
		return nil
	}
	return s.applyFloatingPanelWindowRegion(state)
}

func (s *RecordingFreedomService) SetFloatingSelectHitRegions(req CapsuleWindowHitRegionsRequest) error {
	state, changed := s.floatingSelectRegions.Update(req)
	if !changed && !req.Force {
		return nil
	}
	return s.applyFloatingSelectWindowRegion(state)
}

func (s *RecordingFreedomService) GetSourceState() (SourceControlState, error) {
	if s == nil {
		return SourceControlState{}, nil
	}
	s.sourceMu.Lock()
	defer s.sourceMu.Unlock()
	if s.sourceState.RecordingMode == "" && s.sourceState.SourceID == "" && s.settings != nil {
		current, err := s.loadSettingsForMutation()
		if err == nil {
			s.sourceState = sourceStateFromSettings(current)
		}
	}
	if s.sourceState.RecordingMode == "" {
		s.sourceState.RecordingMode = "video"
	}
	return s.sourceState, nil
}

func (s *RecordingFreedomService) PatchSourceState(req SourceStatePatchRequest) (SourceControlState, error) {
	if s == nil {
		return SourceControlState{}, nil
	}
	s.sourceMu.Lock()
	current := s.sourceState
	if current.RecordingMode == "" {
		current = sourceStateFromSettings(settings.Settings{})
		if s.settings != nil {
			if saved, err := s.loadSettingsForMutation(); err == nil {
				current = sourceStateFromSettings(saved)
			}
		}
	}
	if mode := strings.TrimSpace(req.RecordingMode); mode == "video" || mode == "audio" {
		current.RecordingMode = mode
	}
	if sourceID := strings.TrimSpace(req.SourceID); sourceID != "" {
		current.SourceID = sourceID
	}
	if sourceType := strings.TrimSpace(req.SourceType); sourceType != "" {
		current.SourceType = sourceType
	}
	if req.SourceGeometry != nil {
		geometry := *req.SourceGeometry
		current.SourceGeometry = &geometry
	}
	if req.ClearGeometry || (strings.TrimSpace(req.SourceType) != "" && strings.TrimSpace(req.SourceType) != "region") {
		current.SourceGeometry = nil
	}
	current.UpdatedAt = time.Now()
	s.sourceState = current
	s.sourceMu.Unlock()

	if s.settings != nil && (strings.TrimSpace(req.SourceID) != "" || strings.TrimSpace(req.SourceType) != "") {
		s.settingsMu.Lock()
		saved, err := s.loadSettingsForMutation()
		if err == nil {
			if current.SourceID != "" {
				saved.Source.LastSourceID = current.SourceID
			}
			if current.SourceType != "" {
				saved.Source.LastSourceType = current.SourceType
			}
			saved, err = s.settings.Save(saved)
		}
		s.settingsMu.Unlock()
		if err != nil {
			return SourceControlState{}, err
		}
		s.emitSettingsChanged(saved)
	}
	s.emitSourceStateChanged(current)
	return current, nil
}

func sourceStateFromSettings(current settings.Settings) SourceControlState {
	sourceType := strings.TrimSpace(current.Source.LastSourceType)
	if sourceType == "" {
		sourceType = "screen"
	}
	return SourceControlState{
		RecordingMode: "video",
		SourceID:      strings.TrimSpace(current.Source.LastSourceID),
		SourceType:    sourceType,
		UpdatedAt:     time.Now(),
	}
}

func validFloatingPanelKind(kind FloatingPanelKind) bool {
	switch kind {
	case FloatingPanelSource, FloatingPanelAudio, FloatingPanelCamera, FloatingPanelBoard, FloatingPanelLanguage, FloatingPanelSettings, FloatingPanelClose, FloatingPanelOCRResult:
		return true
	default:
		return false
	}
}

func normalizedFloatingBounds(bounds FloatingRect, fallbackWidth int, fallbackHeight int, minWidth int, minHeight int) FloatingRect {
	width := bounds.Width
	height := bounds.Height
	if width <= 0 {
		width = fallbackWidth
	}
	if height <= 0 {
		height = fallbackHeight
	}
	if width < minWidth {
		width = minWidth
	}
	if height < minHeight {
		height = minHeight
	}
	return FloatingRect{
		X:      bounds.X,
		Y:      bounds.Y,
		Width:  width,
		Height: height,
	}
}

func (s *RecordingFreedomService) SetCapsuleWindowBounds(bounds FloatingRect) error {
	if s == nil || s.capsuleWindow == nil {
		return nil
	}
	bounds = normalizedFloatingBounds(bounds, bounds.Width, bounds.Height, 1, 1)
	return application.InvokeSyncWithError(func() error {
		s.capsuleWindow.SetBounds(application.Rect{
			X:      bounds.X,
			Y:      bounds.Y,
			Width:  bounds.Width,
			Height: bounds.Height,
		})
		return nil
	})
}

func (s *RecordingFreedomService) applyFloatingWindowBounds(window *application.WebviewWindow, bounds FloatingRect, regionState capsuleWindowHitRegionState, focus bool) error {
	if window == nil {
		return nil
	}
	rect := application.Rect{
		X:      bounds.X,
		Y:      bounds.Y,
		Width:  bounds.Width,
		Height: bounds.Height,
	}
	window.SetAlwaysOnTop(true)
	window.SetBounds(rect)
	if err := s.applyWindowRegion(window, regionState); err != nil {
		return err
	}
	window.Show()
	window.SetBounds(rect)
	if focus {
		window.Focus()
	}
	return nil
}

func fullFloatingWindowHitRegionRequest(bounds FloatingRect, radius float64) CapsuleWindowHitRegionsRequest {
	width := bounds.Width
	height := bounds.Height
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	return CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		Force:          true,
		ViewportWidth:  float64(width),
		ViewportHeight: float64(height),
		Regions: []CapsuleWindowHitRegion{
			{
				X:      0,
				Y:      0,
				Width:  float64(width),
				Height: float64(height),
				Kind:   "round-rect",
				Radius: radius,
			},
		},
	}
}

func (s *RecordingFreedomService) emitFloatingPanelChanged(state FloatingPanelState) {
	if s != nil && s.app != nil {
		s.app.Event.Emit("floating.panel.changed", state)
	}
}

func (s *RecordingFreedomService) emitFloatingSelectChanged(state FloatingSelectState) {
	if s != nil && s.app != nil {
		s.app.Event.Emit("floating.select.changed", state)
	}
}

func (s *RecordingFreedomService) emitSourceStateChanged(state SourceControlState) {
	if s != nil && s.app != nil {
		s.app.Event.Emit("source.state.changed", state)
	}
}

func formatFloatingRect(rect FloatingRect) string {
	return fmt.Sprintf("%d,%d %dx%d", rect.X, rect.Y, rect.Width, rect.Height)
}
