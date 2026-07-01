package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	screenIndicatorMinWidth  = 240
	screenIndicatorMaxWidth  = 520
	screenIndicatorMinHeight = 180
	screenIndicatorMaxHeight = 380
)

type ScreenIndicatorRequest struct {
	SourceID string `json:"sourceId"`
}

type ScreenIndicatorResult struct {
	SourceID     string     `json:"sourceId"`
	DisplayIndex int        `json:"displayIndex"`
	Label        string     `json:"label"`
	SourceBounds RegionRect `json:"sourceBounds"`
	WindowBounds RegionRect `json:"windowBounds"`
}

func (s *RecordingFreedomService) ShowScreenIndicator(req ScreenIndicatorRequest) (ScreenIndicatorResult, error) {
	if s.app == nil || s.screenIndicator == nil {
		return ScreenIndicatorResult{}, errors.New("screen indicator window is not configured")
	}
	source, ok := findScreenSource(s.devices.ListSources(), req.SourceID)
	if !ok {
		return ScreenIndicatorResult{}, fmt.Errorf("screen source %q was not returned by DeviceService", req.SourceID)
	}

	displayBounds := screenSourceDisplayBounds(source, s.app.Screen.GetAll())
	windowBounds := screenIndicatorBounds(displayBounds)
	result := ScreenIndicatorResult{
		SourceID:     source.ID,
		DisplayIndex: screenDisplayIndex(source),
		Label:        fmt.Sprintf("%d", screenDisplayIndex(source)),
		SourceBounds: regionRectFromAppRect(displayBounds),
		WindowBounds: regionRectFromAppRect(windowBounds),
	}

	s.screenIndicator.SetBounds(windowBounds)
	s.screenIndicator.SetAlwaysOnTop(true)
	s.screenIndicator.Show()
	if payload, err := json.Marshal(result); err == nil {
		s.screenIndicator.ExecJS(fmt.Sprintf(
			"window.__RF_SCREEN_INDICATOR__=%s;window.dispatchEvent(new CustomEvent('rf-screen-indicator',{detail:window.__RF_SCREEN_INDICATOR__}));",
			string(payload),
		))
	}
	return result, nil
}

func (s *RecordingFreedomService) HideScreenIndicator() error {
	if s.screenIndicator == nil {
		return nil
	}
	s.screenIndicator.Hide()
	return nil
}

func findScreenSource(sources []devices.CaptureSource, sourceID string) (devices.CaptureSource, bool) {
	for _, source := range sources {
		if source.ID == sourceID && source.Type == devices.SourceScreen {
			return source, true
		}
	}
	return devices.CaptureSource{}, false
}

func screenDisplayIndex(source devices.CaptureSource) int {
	if source.DisplayIndex > 0 {
		return source.DisplayIndex
	}
	return 1
}

func screenSourceDisplayBounds(source devices.CaptureSource, screens []*application.Screen) application.Rect {
	physical := screenSourcePhysicalBounds(source)
	if bounds, ok := screenBoundsMatchingPhysicalSource(physical, screens); ok {
		return bounds
	}

	if source.DisplayIndex > 0 && source.DisplayIndex <= len(screens) {
		bounds := screens[source.DisplayIndex-1].Bounds
		if bounds.Width > 0 && bounds.Height > 0 {
			return bounds
		}
	}

	return physical
}

func screenSourcePhysicalBounds(source devices.CaptureSource) application.Rect {
	return application.Rect{
		X:      source.X,
		Y:      source.Y,
		Width:  source.Width,
		Height: source.Height,
	}
}

func screenBoundsMatchingPhysicalSource(physical application.Rect, screens []*application.Screen) (application.Rect, bool) {
	if physical.Width <= 0 || physical.Height <= 0 {
		return application.Rect{}, false
	}

	var best application.Rect
	bestArea := 0
	for _, screen := range screens {
		if screen == nil || screen.PhysicalBounds.Width <= 0 || screen.PhysicalBounds.Height <= 0 {
			continue
		}
		area := rectIntersectionArea(physical, screen.PhysicalBounds)
		if area > bestArea {
			bestArea = area
			best = screen.Bounds
		}
	}
	return best, bestArea > 0 && best.Width > 0 && best.Height > 0
}

func rectIntersectionArea(a application.Rect, b application.Rect) int {
	left := maxInt(a.X, b.X)
	top := maxInt(a.Y, b.Y)
	right := minInt(a.X+a.Width, b.X+b.Width)
	bottom := minInt(a.Y+a.Height, b.Y+b.Height)
	if right <= left || bottom <= top {
		return 0
	}
	return (right - left) * (bottom - top)
}

func screenIndicatorBounds(display application.Rect) application.Rect {
	if display.Width <= 0 || display.Height <= 0 {
		display.Width = 1280
		display.Height = 720
	}

	width := clampInt(display.Width*52/100, screenIndicatorMinWidth, screenIndicatorMaxWidth)
	height := clampInt(display.Height*42/100, screenIndicatorMinHeight, screenIndicatorMaxHeight)
	if maxWidth := display.Width - 64; maxWidth > 0 && width > maxWidth {
		width = clampInt(maxWidth, 160, screenIndicatorMaxWidth)
	}
	if maxHeight := display.Height - 64; maxHeight > 0 && height > maxHeight {
		height = clampInt(maxHeight, 120, screenIndicatorMaxHeight)
	}

	return application.Rect{
		X:      display.X + (display.Width-width)/2,
		Y:      display.Y + (display.Height-height)/2,
		Width:  width,
		Height: height,
	}
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
