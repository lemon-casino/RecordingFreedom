package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	desktopscreenshot "github.com/kbinani/screenshot"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	regionSmartKindScreen = "screen"
	regionSmartKindWindow = "window"
	regionSmartKindEdge   = "edge"

	regionSmartEdgeSearchPx  = 56
	regionSmartEdgePaddingPx = 96
	regionSmartEdgeThreshold = 16
)

type RegionSmartCandidate struct {
	ID       string     `json:"id"`
	Kind     string     `json:"kind"`
	Label    string     `json:"label,omitempty"`
	Bounds   RegionRect `json:"bounds"`
	SourceID string     `json:"sourceId,omitempty"`
	Score    float64    `json:"score,omitempty"`
}

type RegionAssistRequest struct {
	SessionID string      `json:"sessionId,omitempty"`
	Purpose   string      `json:"purpose,omitempty"`
	PointerX  int         `json:"pointerX,omitempty"`
	PointerY  int         `json:"pointerY,omitempty"`
	Selection *RegionRect `json:"selection,omitempty"`
}

type RegionAssistResult struct {
	Candidates []RegionSmartCandidate `json:"candidates"`
	Best       *RegionSmartCandidate  `json:"best,omitempty"`
}

func (s *RecordingFreedomService) AssistRegionSelection(req RegionAssistRequest) (RegionAssistResult, error) {
	session, err := s.activeRegionAssistSession(req.SessionID)
	if err != nil {
		return RegionAssistResult{}, err
	}
	candidates := append([]RegionSmartCandidate(nil), session.Candidates...)
	if len(candidates) == 0 {
		candidates = s.regionSmartCandidates(session)
	}
	if req.Selection != nil {
		selection := normalizeRegionSelection(RegionSelectionRequest{
			X:      req.Selection.X,
			Y:      req.Selection.Y,
			Width:  req.Selection.Width,
			Height: req.Selection.Height,
		})
		selectionRect := regionRectFromAppRect(selection)
		if edge, ok := s.regionImageEdgeCandidate(session, selectionRect); ok {
			candidates = append([]RegionSmartCandidate{edge}, candidates...)
		}
	}
	best := bestRegionSmartCandidate(candidates, req)
	return RegionAssistResult{Candidates: candidates, Best: best}, nil
}

func (s *RecordingFreedomService) activeRegionAssistSession(sessionID string) (RegionSelectionSession, error) {
	s.regionMu.Lock()
	defer s.regionMu.Unlock()
	if s.regionSession == nil {
		return RegionSelectionSession{}, errors.New("no active region selection session")
	}
	if strings.TrimSpace(sessionID) != "" && s.regionSession.ID != sessionID {
		return RegionSelectionSession{}, fmt.Errorf("active region selection session is %q, not %q", s.regionSession.ID, sessionID)
	}
	return *s.regionSession, nil
}

func (s *RecordingFreedomService) regionSmartCandidates(session RegionSelectionSession) []RegionSmartCandidate {
	candidates := make([]RegionSmartCandidate, 0, 8)
	seen := map[string]bool{}
	add := func(candidate RegionSmartCandidate) {
		candidate.Bounds = clampRegionCandidateBounds(candidate.Bounds, session.Bounds)
		if candidate.Bounds.Width < session.MinimumWidth || candidate.Bounds.Height < session.MinimumHeight {
			return
		}
		key := fmt.Sprintf("%s:%d:%d:%d:%d", candidate.Kind, candidate.Bounds.X, candidate.Bounds.Y, candidate.Bounds.Width, candidate.Bounds.Height)
		if seen[key] {
			return
		}
		seen[key] = true
		candidates = append(candidates, candidate)
	}
	if s.app != nil {
		for index, screen := range s.app.Screen.GetAll() {
			if screen == nil || screen.Bounds.Width <= 0 || screen.Bounds.Height <= 0 {
				continue
			}
			add(candidateFromAbsoluteRect(
				fmt.Sprintf("screen:%d", index+1),
				regionSmartKindScreen,
				fmt.Sprintf("Screen %d", index+1),
				"",
				screen.Bounds,
				session.Bounds,
				0.62,
			))
		}
	}
	if s.devices != nil {
		for _, source := range s.devices.ListSources() {
			if source.Width <= 0 || source.Height <= 0 {
				continue
			}
			if source.Type != devices.SourceScreen && source.Type != devices.SourceWindow {
				continue
			}
			kind := regionSmartKindScreen
			if source.Type == devices.SourceWindow {
				kind = regionSmartKindWindow
			}
			add(candidateFromAbsoluteRect(
				source.ID,
				kind,
				source.Name,
				source.ID,
				application.Rect{X: source.X, Y: source.Y, Width: source.Width, Height: source.Height},
				session.Bounds,
				0.7,
			))
		}
	}
	sortRegionSmartCandidates(candidates)
	return candidates
}

func candidateFromAbsoluteRect(id string, kind string, label string, sourceID string, absolute application.Rect, overlay RegionRect, score float64) RegionSmartCandidate {
	return RegionSmartCandidate{
		ID:       strings.TrimSpace(id),
		Kind:     kind,
		Label:    strings.TrimSpace(label),
		SourceID: strings.TrimSpace(sourceID),
		Bounds: RegionRect{
			X:      absolute.X - overlay.X,
			Y:      absolute.Y - overlay.Y,
			Width:  absolute.Width,
			Height: absolute.Height,
		},
		Score: score,
	}
}

func clampRegionCandidateBounds(bounds RegionRect, overlay RegionRect) RegionRect {
	left := maxInt(0, bounds.X)
	top := maxInt(0, bounds.Y)
	right := minInt(overlay.Width, bounds.X+bounds.Width)
	bottom := minInt(overlay.Height, bounds.Y+bounds.Height)
	if right <= left || bottom <= top {
		return RegionRect{}
	}
	return RegionRect{X: left, Y: top, Width: right - left, Height: bottom - top}
}

func bestRegionSmartCandidate(candidates []RegionSmartCandidate, req RegionAssistRequest) *RegionSmartCandidate {
	if len(candidates) == 0 {
		return nil
	}
	var best *RegionSmartCandidate
	bestScore := -1.0
	if req.Selection != nil && req.Selection.Width > 0 && req.Selection.Height > 0 {
		selection := *req.Selection
		for _, candidate := range candidates {
			score := smartCandidateSelectionScore(candidate, selection)
			if score > bestScore {
				candidate.Score = math.Max(candidate.Score, score)
				next := candidate
				best = &next
				bestScore = score
			}
		}
		if bestScore >= 0.5 {
			return best
		}
		return nil
	}
	point := image.Point{X: req.PointerX, Y: req.PointerY}
	for _, candidate := range candidates {
		if !regionRectContainsPoint(candidate.Bounds, point) {
			continue
		}
		area := maxInt(1, candidate.Bounds.Width*candidate.Bounds.Height)
		score := candidate.Score + 1000000.0/float64(area)
		if score > bestScore {
			next := candidate
			next.Score = score
			best = &next
			bestScore = score
		}
	}
	return best
}

func smartCandidateSelectionScore(candidate RegionSmartCandidate, selection RegionRect) float64 {
	intersection := intersectRegionRects(candidate.Bounds, selection)
	if intersection.Width <= 0 || intersection.Height <= 0 {
		return -1
	}
	selectionArea := float64(maxInt(1, selection.Width*selection.Height))
	candidateArea := float64(maxInt(1, candidate.Bounds.Width*candidate.Bounds.Height))
	intersectionArea := float64(intersection.Width * intersection.Height)
	overlap := intersectionArea / math.Min(selectionArea, candidateArea)
	edgeDistance := float64(absInt(candidate.Bounds.X-selection.X) +
		absInt(candidate.Bounds.Y-selection.Y) +
		absInt(candidate.Bounds.X+candidate.Bounds.Width-selection.X-selection.Width) +
		absInt(candidate.Bounds.Y+candidate.Bounds.Height-selection.Y-selection.Height))
	closeEdges := math.Max(0, 1-edgeDistance/280)
	areaRatio := math.Min(selectionArea, candidateArea) / math.Max(selectionArea, candidateArea)
	score := overlap*0.54 + closeEdges*0.34 + areaRatio*0.12
	if candidate.Kind == regionSmartKindEdge {
		score += 0.16
	}
	return score
}

func intersectRegionRects(a RegionRect, b RegionRect) RegionRect {
	left := maxInt(a.X, b.X)
	top := maxInt(a.Y, b.Y)
	right := minInt(a.X+a.Width, b.X+b.Width)
	bottom := minInt(a.Y+a.Height, b.Y+b.Height)
	if right <= left || bottom <= top {
		return RegionRect{}
	}
	return RegionRect{X: left, Y: top, Width: right - left, Height: bottom - top}
}

func regionRectContainsPoint(rect RegionRect, point image.Point) bool {
	return point.X >= rect.X && point.X <= rect.X+rect.Width && point.Y >= rect.Y && point.Y <= rect.Y+rect.Height
}

func sortRegionSmartCandidates(candidates []RegionSmartCandidate) {
	sort.SliceStable(candidates, func(left, right int) bool {
		leftArea := candidates[left].Bounds.Width * candidates[left].Bounds.Height
		rightArea := candidates[right].Bounds.Width * candidates[right].Bounds.Height
		if candidates[left].Kind != candidates[right].Kind {
			return candidates[left].Kind == regionSmartKindWindow
		}
		return leftArea < rightArea
	})
}

func (s *RecordingFreedomService) regionImageEdgeCandidate(session RegionSelectionSession, selection RegionRect) (RegionSmartCandidate, bool) {
	if selection.Width < session.MinimumWidth || selection.Height < session.MinimumHeight {
		return RegionSmartCandidate{}, false
	}
	captureRect := mapRegionSelectionToCaptureRect(session, selection)
	if captureRect.Empty() {
		return RegionSmartCandidate{}, false
	}
	captureBounds := captureBoundsForRegionSession(session)
	if captureBounds.Empty() {
		captureBounds = captureRect
	}
	expanded := expandImageRect(captureRect, regionSmartEdgePaddingPx).Intersect(captureBounds)
	if expanded.Empty() {
		return RegionSmartCandidate{}, false
	}
	img, err := desktopscreenshot.CaptureRect(expanded)
	if err != nil || img == nil || img.Bounds().Empty() {
		return RegionSmartCandidate{}, false
	}
	localSelection := image.Rect(
		captureRect.Min.X-expanded.Min.X,
		captureRect.Min.Y-expanded.Min.Y,
		captureRect.Max.X-expanded.Min.X,
		captureRect.Max.Y-expanded.Min.Y,
	)
	snapped, confidence := snapRectToImageEdges(img, localSelection, regionSmartEdgeSearchPx)
	if confidence < 0.5 || snapped.Empty() {
		return RegionSmartCandidate{}, false
	}
	absolute := image.Rect(
		snapped.Min.X+expanded.Min.X,
		snapped.Min.Y+expanded.Min.Y,
		snapped.Max.X+expanded.Min.X,
		snapped.Max.Y+expanded.Min.Y,
	)
	relative := mapCaptureRectToRegionSelection(session, absolute)
	if relative.Width < session.MinimumWidth || relative.Height < session.MinimumHeight {
		return RegionSmartCandidate{}, false
	}
	return RegionSmartCandidate{
		ID:     "edge:snap",
		Kind:   regionSmartKindEdge,
		Label:  "Smart edge",
		Bounds: clampRegionCandidateBounds(relative, session.Bounds),
		Score:  0.86 + confidence*0.1,
	}, true
}

func captureBoundsForRegionSession(session RegionSelectionSession) image.Rectangle {
	if session.CaptureBounds != nil && session.CaptureBounds.Width > 0 && session.CaptureBounds.Height > 0 {
		return image.Rect(
			session.CaptureBounds.X,
			session.CaptureBounds.Y,
			session.CaptureBounds.X+session.CaptureBounds.Width,
			session.CaptureBounds.Y+session.CaptureBounds.Height,
		)
	}
	return image.Rect(session.Bounds.X, session.Bounds.Y, session.Bounds.X+session.Bounds.Width, session.Bounds.Y+session.Bounds.Height)
}

func expandImageRect(rect image.Rectangle, padding int) image.Rectangle {
	return image.Rect(rect.Min.X-padding, rect.Min.Y-padding, rect.Max.X+padding, rect.Max.Y+padding)
}

func mapCaptureRectToRegionSelection(session RegionSelectionSession, rect image.Rectangle) RegionRect {
	capture := session.CaptureBounds
	if capture == nil || capture.Width <= 0 || capture.Height <= 0 || session.Bounds.Width <= 0 || session.Bounds.Height <= 0 {
		return RegionRect{
			X:      rect.Min.X - session.Bounds.X,
			Y:      rect.Min.Y - session.Bounds.Y,
			Width:  rect.Dx(),
			Height: rect.Dy(),
		}
	}
	x := scaleRegionValue(rect.Min.X-capture.X, capture.Width, session.Bounds.Width)
	y := scaleRegionValue(rect.Min.Y-capture.Y, capture.Height, session.Bounds.Height)
	width := scaleRegionValue(rect.Dx(), capture.Width, session.Bounds.Width)
	height := scaleRegionValue(rect.Dy(), capture.Height, session.Bounds.Height)
	return RegionRect{X: x, Y: y, Width: maxInt(1, width), Height: maxInt(1, height)}
}

func snapRectToImageEdges(img image.Image, selection image.Rectangle, search int) (image.Rectangle, float64) {
	if img == nil || img.Bounds().Empty() || selection.Empty() {
		return selection, 0
	}
	bounds := img.Bounds()
	selection = selection.Intersect(bounds)
	if selection.Empty() {
		return selection, 0
	}
	left, leftScore := strongestVerticalEdge(img, selection.Min.X-search, selection.Min.X+search, selection.Min.Y, selection.Max.Y)
	right, rightScore := strongestVerticalEdge(img, selection.Max.X-search, selection.Max.X+search, selection.Min.Y, selection.Max.Y)
	top, topScore := strongestHorizontalEdge(img, selection.Min.Y-search, selection.Min.Y+search, selection.Min.X, selection.Max.X)
	bottom, bottomScore := strongestHorizontalEdge(img, selection.Max.Y-search, selection.Max.Y+search, selection.Min.X, selection.Max.X)

	next := selection
	snapped := 0
	total := 0
	if leftScore >= regionSmartEdgeThreshold {
		next.Min.X = left
		snapped++
		total += leftScore
	}
	if rightScore >= regionSmartEdgeThreshold {
		next.Max.X = right
		snapped++
		total += rightScore
	}
	if topScore >= regionSmartEdgeThreshold {
		next.Min.Y = top
		snapped++
		total += topScore
	}
	if bottomScore >= regionSmartEdgeThreshold {
		next.Max.Y = bottom
		snapped++
		total += bottomScore
	}
	next = next.Intersect(bounds)
	if next.Dx() < minRegionWidth || next.Dy() < minRegionHeight || snapped < 2 {
		return selection, 0
	}
	confidence := math.Min(1, float64(total)/float64(snapped*48))
	return next, confidence
}

func strongestVerticalEdge(img image.Image, startX int, endX int, startY int, endY int) (int, int) {
	bounds := img.Bounds()
	startX = maxInt(bounds.Min.X+1, startX)
	endX = minInt(bounds.Max.X-1, endX)
	startY = maxInt(bounds.Min.Y, startY)
	endY = minInt(bounds.Max.Y, endY)
	bestX := startX
	bestScore := -1
	for x := startX; x <= endX; x++ {
		score := averageVerticalEdgeScore(img, x, startY, endY)
		if score > bestScore {
			bestScore = score
			bestX = x
		}
	}
	return bestX, bestScore
}

func strongestHorizontalEdge(img image.Image, startY int, endY int, startX int, endX int) (int, int) {
	bounds := img.Bounds()
	startY = maxInt(bounds.Min.Y+1, startY)
	endY = minInt(bounds.Max.Y-1, endY)
	startX = maxInt(bounds.Min.X, startX)
	endX = minInt(bounds.Max.X, endX)
	bestY := startY
	bestScore := -1
	for y := startY; y <= endY; y++ {
		score := averageHorizontalEdgeScore(img, y, startX, endX)
		if score > bestScore {
			bestScore = score
			bestY = y
		}
	}
	return bestY, bestScore
}

func averageVerticalEdgeScore(img image.Image, x int, startY int, endY int) int {
	if endY <= startY {
		return 0
	}
	step := maxInt(1, (endY-startY)/96)
	total := 0
	samples := 0
	for y := startY; y < endY; y += step {
		total += regionEdgeColorDistance(img.At(x-1, y), img.At(x, y))
		samples++
	}
	if samples == 0 {
		return 0
	}
	return total / samples
}

func averageHorizontalEdgeScore(img image.Image, y int, startX int, endX int) int {
	if endX <= startX {
		return 0
	}
	step := maxInt(1, (endX-startX)/96)
	total := 0
	samples := 0
	for x := startX; x < endX; x += step {
		total += regionEdgeColorDistance(img.At(x, y-1), img.At(x, y))
		samples++
	}
	if samples == 0 {
		return 0
	}
	return total / samples
}

func regionEdgeColorDistance(a color.Color, b color.Color) int {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return (absInt(int(ar>>8)-int(br>>8)) +
		absInt(int(ag>>8)-int(bg>>8)) +
		absInt(int(ab>>8)-int(bb>>8)) +
		absInt(int(aa>>8)-int(ba>>8))) / 3
}
