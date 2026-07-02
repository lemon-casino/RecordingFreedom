package main

import (
	"math"
	"sync"
)

const capsuleHitRegionPadding = 2

type CapsuleWindowHitRegion struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Kind   string  `json:"kind,omitempty"`
	Radius float64 `json:"radius,omitempty"`
}

type CapsuleWindowHitRegionsRequest struct {
	Enabled          bool                     `json:"enabled"`
	ViewportWidth    float64                  `json:"viewportWidth"`
	ViewportHeight   float64                  `json:"viewportHeight"`
	DevicePixelRatio float64                  `json:"devicePixelRatio"`
	Regions          []CapsuleWindowHitRegion `json:"regions"`
}

type capsuleWindowHitRegionState struct {
	enabled        bool
	viewportWidth  float64
	viewportHeight float64
	regions        []CapsuleWindowHitRegion
}

type capsuleWindowHitRegions struct {
	mu    sync.RWMutex
	state capsuleWindowHitRegionState
}

func (s *RecordingFreedomService) SetCapsuleWindowHitRegions(req CapsuleWindowHitRegionsRequest) error {
	state := s.capsuleHitRegions.Set(req)
	return s.applyCapsuleWindowRegion(state)
}

func (r *capsuleWindowHitRegions) Set(req CapsuleWindowHitRegionsRequest) capsuleWindowHitRegionState {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := capsuleWindowHitRegionState{
		enabled:        req.Enabled,
		viewportWidth:  sanePositive(req.ViewportWidth),
		viewportHeight: sanePositive(req.ViewportHeight),
	}
	for _, region := range req.Regions {
		normalized, ok := normalizeCapsuleHitRegion(region, state.viewportWidth, state.viewportHeight)
		if ok {
			state.regions = append(state.regions, normalized)
		}
	}
	if state.viewportWidth <= 0 || state.viewportHeight <= 0 || len(state.regions) == 0 {
		state.enabled = false
	}
	r.state = state
	return state
}

func (r *capsuleWindowHitRegions) TestClientPoint(clientX int, clientY int, clientWidth int, clientHeight int) (handled bool, hit bool) {
	r.mu.RLock()
	state := r.state
	r.mu.RUnlock()

	if !state.enabled || len(state.regions) == 0 || state.viewportWidth <= 0 || state.viewportHeight <= 0 {
		return false, true
	}
	scaleX := float64(clientWidth) / state.viewportWidth
	scaleY := float64(clientHeight) / state.viewportHeight
	if !isSaneScale(scaleX) {
		scaleX = 1
	}
	if !isSaneScale(scaleY) {
		scaleY = 1
	}
	x := float64(clientX) / scaleX
	y := float64(clientY) / scaleY
	for _, region := range state.regions {
		if pointInCapsuleHitRegion(x, y, region) {
			return true, true
		}
	}
	return true, false
}

func normalizeCapsuleHitRegion(region CapsuleWindowHitRegion, viewportWidth float64, viewportHeight float64) (CapsuleWindowHitRegion, bool) {
	if region.Width <= 0 || region.Height <= 0 || viewportWidth <= 0 || viewportHeight <= 0 {
		return CapsuleWindowHitRegion{}, false
	}
	x := math.Max(0, math.Floor(region.X-float64(capsuleHitRegionPadding)))
	y := math.Max(0, math.Floor(region.Y-float64(capsuleHitRegionPadding)))
	right := math.Min(viewportWidth, math.Ceil(region.X+region.Width+float64(capsuleHitRegionPadding)))
	bottom := math.Min(viewportHeight, math.Ceil(region.Y+region.Height+float64(capsuleHitRegionPadding)))
	if right <= x || bottom <= y {
		return CapsuleWindowHitRegion{}, false
	}
	radius := sanePositive(region.Radius)
	maxRadius := math.Min(right-x, bottom-y) / 2
	if radius > maxRadius {
		radius = maxRadius
	}
	return CapsuleWindowHitRegion{
		X:      x,
		Y:      y,
		Width:  right - x,
		Height: bottom - y,
		Kind:   region.Kind,
		Radius: radius,
	}, true
}

func pointInCapsuleHitRegion(x float64, y float64, region CapsuleWindowHitRegion) bool {
	return x >= region.X && y >= region.Y && x < region.X+region.Width && y < region.Y+region.Height
}

func sanePositive(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	return value
}

func isSaneScale(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0.1 && value < 10
}
