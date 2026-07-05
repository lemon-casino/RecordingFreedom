package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"math"
	"sort"
	"strings"
	"time"

	desktopscreenshot "github.com/kbinani/screenshot"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	regionSmartKindScreen  = "screen"
	regionSmartKindWindow  = "window"
	regionSmartKindElement = "element"
	regionSmartKindEdge    = "edge"

	regionSmartEdgeSearchPx  = 56
	regionSmartEdgePaddingPx = 96
	regionSmartEdgeThreshold = 16
	regionSmartWeakEdgePx    = 6
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
	SessionID      string      `json:"sessionId,omitempty"`
	Purpose        string      `json:"purpose,omitempty"`
	PointerX       int         `json:"pointerX,omitempty"`
	PointerY       int         `json:"pointerY,omitempty"`
	Selection      *RegionRect `json:"selection,omitempty"`
	CandidateLevel int         `json:"candidateLevel,omitempty"`
}

type RegionAssistResult struct {
	Candidates []RegionSmartCandidate `json:"candidates"`
	Best       *RegionSmartCandidate  `json:"best,omitempty"`
	Source     string                 `json:"source,omitempty"`
}

type regionElementCandidateCache struct {
	SessionID  string
	UpdatedAt  time.Time
	Candidates []RegionSmartCandidate
}

type regionAssistSnapshotCache struct {
	SessionID string
	Bounds    image.Rectangle
	Image     image.Image
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
	} else {
		point := image.Point{X: req.PointerX, Y: req.PointerY}
		elementCandidates := s.regionCachedElementCandidatesAtPoint(session, point)
		windowCandidates := s.regionWindowCandidatesAtPoint(session, point)
		imageCandidates := s.regionImagePointCandidates(session, point, windowCandidates)
		hoverCandidates := mergeRegionSmartCandidates(
			elementCandidates,
			mergeRegionSmartCandidates(imageCandidates, windowCandidates, session),
			session,
		)
		if len(hoverCandidates) > 0 {
			candidates = mergeRegionSmartCandidates(hoverCandidates, candidates, session)
			best := regionCandidateAtLevel(hoverCandidates, req.CandidateLevel)
			source := regionAssistSourceForCandidate(best)
			s.logRegionAssistResult(source, req, hoverCandidates, best)
			return RegionAssistResult{Candidates: candidates, Best: best, Source: source}, nil
		}
	}
	source := "static"
	if req.Selection != nil {
		source = "selection"
	}
	best := bestRegionSmartCandidate(candidates, req)
	s.logRegionAssistResult(source, req, candidates, best)
	return RegionAssistResult{Candidates: candidates, Best: best, Source: source}, nil
}

func regionAssistSourceForCandidate(candidate *RegionSmartCandidate) string {
	if candidate == nil {
		return "static"
	}
	switch candidate.Kind {
	case regionSmartKindElement:
		return "element"
	case regionSmartKindEdge:
		return "image-hover"
	default:
		return "static"
	}
}

func (s *RecordingFreedomService) logRegionAssistResult(source string, req RegionAssistRequest, candidates []RegionSmartCandidate, best *RegionSmartCandidate) {
	if s == nil {
		return
	}
	bestID := ""
	bestKind := ""
	bestLabel := ""
	bestSource := ""
	bestBounds := ""
	if best != nil {
		bestID = best.ID
		bestKind = best.Kind
		bestLabel = best.Label
		bestSource = best.SourceID
		bestBounds = fmt.Sprintf("%d,%d %dx%d", best.Bounds.X, best.Bounds.Y, best.Bounds.Width, best.Bounds.Height)
	}
	key := fmt.Sprintf("%s|%s|%s|%d|%s|%s", req.SessionID, source, bestID, req.CandidateLevel, bestKind, bestBounds)
	now := time.Now()
	s.regionAssistLogMu.Lock()
	if key == s.regionAssistLogKey && now.Sub(s.regionAssistLogAt) < 1500*time.Millisecond {
		s.regionAssistLogMu.Unlock()
		return
	}
	s.regionAssistLogKey = key
	s.regionAssistLogAt = now
	s.regionAssistLogMu.Unlock()
	s.logEvent("region-assist", "candidate", map[string]string{
		"sessionId":      req.SessionID,
		"purpose":        req.Purpose,
		"source":         source,
		"pointer":        fmt.Sprintf("%d,%d", req.PointerX, req.PointerY),
		"candidateLevel": fmt.Sprintf("%d", req.CandidateLevel),
		"candidateCount": fmt.Sprintf("%d", len(candidates)),
		"bestId":         bestID,
		"bestKind":       bestKind,
		"bestLabel":      bestLabel,
		"bestSource":     bestSource,
		"bestBounds":     bestBounds,
	})
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

func (s *RecordingFreedomService) resetRegionElementCache(sessionID string) {
	if s == nil {
		return
	}
	s.regionElementCacheMu.Lock()
	defer s.regionElementCacheMu.Unlock()
	s.regionElementCache = regionElementCandidateCache{
		SessionID: strings.TrimSpace(sessionID),
		UpdatedAt: time.Now(),
	}
}

func (s *RecordingFreedomService) clearRegionElementCache(sessionID string) {
	if s == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	s.regionElementCacheMu.Lock()
	defer s.regionElementCacheMu.Unlock()
	if sessionID == "" || s.regionElementCache.SessionID == "" || s.regionElementCache.SessionID == sessionID {
		s.regionElementCache = regionElementCandidateCache{}
	}
}

func (s *RecordingFreedomService) prepareRegionAssistSnapshot(session RegionSelectionSession) {
	if s == nil {
		return
	}
	bounds := regionAssistSnapshotBounds(session)
	if bounds.Empty() {
		s.clearRegionAssistSnapshot(session.ID)
		return
	}
	img, err := desktopscreenshot.CaptureRect(bounds)
	if err != nil || img == nil || img.Bounds().Empty() {
		s.clearRegionAssistSnapshot(session.ID)
		if err != nil {
			s.logEvent("region-assist", "snapshot-failed", map[string]string{
				"sessionId": session.ID,
				"bounds":    fmt.Sprintf("%d,%d %dx%d", bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()),
				"error":     err.Error(),
			})
		}
		return
	}
	s.regionSnapshotMu.Lock()
	s.regionSnapshotCache = regionAssistSnapshotCache{
		SessionID: strings.TrimSpace(session.ID),
		Bounds:    bounds,
		Image:     img,
	}
	s.regionSnapshotMu.Unlock()
}

func (s *RecordingFreedomService) clearRegionAssistSnapshot(sessionID string) {
	if s == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	s.regionSnapshotMu.Lock()
	defer s.regionSnapshotMu.Unlock()
	if sessionID == "" || s.regionSnapshotCache.SessionID == "" || s.regionSnapshotCache.SessionID == sessionID {
		s.regionSnapshotCache = regionAssistSnapshotCache{}
	}
}

func (s *RecordingFreedomService) regionAssistImage(session RegionSelectionSession, rect image.Rectangle) (image.Image, bool) {
	if s == nil || rect.Empty() {
		return nil, false
	}
	sessionID := strings.TrimSpace(session.ID)
	s.regionSnapshotMu.Lock()
	cache := s.regionSnapshotCache
	s.regionSnapshotMu.Unlock()
	if cache.SessionID == sessionID && cache.Image != nil && rect.In(cache.Bounds) {
		dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		srcPoint := image.Point{
			X: rect.Min.X - cache.Bounds.Min.X + cache.Image.Bounds().Min.X,
			Y: rect.Min.Y - cache.Bounds.Min.Y + cache.Image.Bounds().Min.Y,
		}
		imagedraw.Draw(dst, dst.Bounds(), cache.Image, srcPoint, imagedraw.Src)
		return dst, true
	}
	return nil, false
}

func regionAssistSnapshotBounds(session RegionSelectionSession) image.Rectangle {
	if session.CaptureBounds != nil && session.CaptureBounds.Width > 0 && session.CaptureBounds.Height > 0 {
		return captureBoundsForRegionSession(session)
	}
	union := image.Rectangle{}
	for _, display := range session.DisplayBounds {
		rect := regionRectToImage(display.CaptureBounds)
		if rect.Empty() {
			continue
		}
		if union.Empty() {
			union = rect
		} else {
			union = union.Union(rect)
		}
	}
	if !union.Empty() {
		return union
	}
	return captureBoundsForRegionSession(session)
}

func (s *RecordingFreedomService) regionCachedElementCandidatesAtPoint(session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	cached := s.regionElementCacheCandidatesAtPoint(session.ID, point)
	native := s.regionElementCandidatesAtPoint(session, point)
	if len(native) > 0 {
		s.rememberRegionElementCandidates(session.ID, native, session)
		return mergeRegionSmartCandidates(native, cached, session)
	}
	return cached
}

func (s *RecordingFreedomService) regionElementCacheCandidatesAtPoint(sessionID string, point image.Point) []RegionSmartCandidate {
	if s == nil {
		return nil
	}
	s.regionElementCacheMu.Lock()
	defer s.regionElementCacheMu.Unlock()
	if s.regionElementCache.SessionID != sessionID {
		return nil
	}
	candidates := make([]RegionSmartCandidate, 0, len(s.regionElementCache.Candidates))
	for _, candidate := range s.regionElementCache.Candidates {
		if candidate.Kind == regionSmartKindElement && regionRectContainsPoint(candidate.Bounds, point) {
			candidates = append(candidates, candidate)
		}
	}
	sortRegionSmartCandidates(candidates)
	return candidates
}

func (s *RecordingFreedomService) rememberRegionElementCandidates(sessionID string, candidates []RegionSmartCandidate, session RegionSelectionSession) {
	if s == nil || strings.TrimSpace(sessionID) == "" || len(candidates) == 0 {
		return
	}
	s.regionElementCacheMu.Lock()
	defer s.regionElementCacheMu.Unlock()
	if s.regionElementCache.SessionID != sessionID {
		s.regionElementCache = regionElementCandidateCache{SessionID: sessionID}
	}
	merged := mergeRegionSmartCandidates(candidates, s.regionElementCache.Candidates, session)
	elements := merged[:0]
	for _, candidate := range merged {
		if candidate.Kind == regionSmartKindElement {
			elements = append(elements, candidate)
		}
	}
	sortRegionSmartCandidates(elements)
	s.regionElementCache.Candidates = append([]RegionSmartCandidate(nil), elements...)
	s.regionElementCache.UpdatedAt = time.Now()
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

func mergeRegionSmartCandidates(primary []RegionSmartCandidate, fallback []RegionSmartCandidate, session RegionSelectionSession) []RegionSmartCandidate {
	merged := make([]RegionSmartCandidate, 0, len(primary)+len(fallback))
	seen := map[string]bool{}
	minWidth, minHeight := regionSmartCandidateMinimumSize(session)
	add := func(candidate RegionSmartCandidate) {
		candidate.Bounds = clampRegionCandidateBounds(candidate.Bounds, session.Bounds)
		if candidate.Bounds.Width < minWidth || candidate.Bounds.Height < minHeight {
			return
		}
		key := fmt.Sprintf("%s:%d:%d:%d:%d", candidate.Kind, candidate.Bounds.X, candidate.Bounds.Y, candidate.Bounds.Width, candidate.Bounds.Height)
		if seen[key] {
			return
		}
		seen[key] = true
		merged = append(merged, candidate)
	}
	for _, candidate := range primary {
		add(candidate)
	}
	for _, candidate := range fallback {
		add(candidate)
	}
	return merged
}

func regionSmartCandidateMinimumSize(session RegionSelectionSession) (int, int) {
	minWidth := session.MinimumWidth
	minHeight := session.MinimumHeight
	if minWidth <= 0 {
		minWidth = minRegionWidth
	}
	if minHeight <= 0 {
		minHeight = minRegionHeight
	}
	if session.Purpose == regionSelectionPurposeScreenshot {
		return minInt(minWidth, 12), minInt(minHeight, 12)
	}
	return minInt(minWidth, 24), minInt(minHeight, 24)
}

func regionCandidateAtLevel(candidates []RegionSmartCandidate, level int) *RegionSmartCandidate {
	if len(candidates) == 0 {
		return nil
	}
	if level < 0 {
		level = 0
	}
	if level >= len(candidates) {
		level = len(candidates) - 1
	}
	next := candidates[level]
	return &next
}

func safeRegionElementLabel(raw string, fallback string) string {
	label := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "UI element"
	}
	if label == "" {
		return fallback
	}
	lower := strings.ToLower(label)
	if len(label) > 80 ||
		strings.HasPrefix(label, "{") ||
		strings.HasPrefix(label, "[") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "authorization") ||
		strings.Contains(lower, "password") {
		return fallback
	}
	return label
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
	return rect.Width > 0 &&
		rect.Height > 0 &&
		point.X >= rect.X &&
		point.X < rect.X+rect.Width &&
		point.Y >= rect.Y &&
		point.Y < rect.Y+rect.Height
}

func sortRegionSmartCandidates(candidates []RegionSmartCandidate) {
	sort.SliceStable(candidates, func(left, right int) bool {
		leftArea := candidates[left].Bounds.Width * candidates[left].Bounds.Height
		rightArea := candidates[right].Bounds.Width * candidates[right].Bounds.Height
		if candidates[left].Kind != candidates[right].Kind {
			if candidates[left].Kind == regionSmartKindElement {
				return true
			}
			if candidates[right].Kind == regionSmartKindElement {
				return false
			}
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
	captureBounds := captureBoundsForRegionSelection(session, selection)
	if captureBounds.Empty() {
		captureBounds = captureRect
	}
	expanded := expandImageRect(captureRect, regionSmartEdgePaddingPx).Intersect(captureBounds)
	if expanded.Empty() {
		return RegionSmartCandidate{}, false
	}
	img, ok := s.regionAssistImage(session, expanded)
	if !ok || img == nil || img.Bounds().Empty() {
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

func (s *RecordingFreedomService) regionImagePointCandidate(session RegionSelectionSession, point image.Point) (RegionSmartCandidate, bool) {
	candidates := s.regionImagePointCandidates(session, point, nil)
	if len(candidates) == 0 {
		return RegionSmartCandidate{}, false
	}
	return candidates[0], true
}

func (s *RecordingFreedomService) regionImagePointCandidates(session RegionSelectionSession, point image.Point, parentCandidates []RegionSmartCandidate) []RegionSmartCandidate {
	if point.X < 0 || point.Y < 0 || point.X > session.Bounds.Width || point.Y > session.Bounds.Height {
		return nil
	}
	captureBounds := captureBoundsForRegionPoint(session, point)
	if captureBounds.Empty() {
		return nil
	}
	capturePoint := mapRegionPointToCapturePoint(session, point)
	if !capturePoint.In(captureBounds) {
		return nil
	}
	img, ok := s.regionAssistImage(session, captureBounds)
	if !ok || img == nil || img.Bounds().Empty() {
		return nil
	}
	localPoint := image.Point{X: capturePoint.X - captureBounds.Min.X, Y: capturePoint.Y - captureBounds.Min.Y}
	localParents := localImageParentRectsForRegionPoint(session, point, captureBounds, capturePoint, img.Bounds(), parentCandidates)
	detected := detectImageRegionsAroundPoint(img, localPoint, localParents)
	if len(detected) == 0 {
		return nil
	}

	candidates := make([]RegionSmartCandidate, 0, len(detected))
	seen := map[string]bool{}
	for index, candidate := range detected {
		absolute := image.Rect(
			candidate.Rect.Min.X+captureBounds.Min.X,
			candidate.Rect.Min.Y+captureBounds.Min.Y,
			candidate.Rect.Max.X+captureBounds.Min.X,
			candidate.Rect.Max.Y+captureBounds.Min.Y,
		)
		relative := mapCaptureRectToRegionSelection(session, absolute)
		relative = clampRegionCandidateBounds(relative, session.Bounds)
		if relative.Width < session.MinimumWidth || relative.Height < session.MinimumHeight {
			continue
		}
		key := fmt.Sprintf("%d:%d:%d:%d", relative.X, relative.Y, relative.Width, relative.Height)
		if seen[key] {
			continue
		}
		seen[key] = true
		candidates = append(candidates, RegionSmartCandidate{
			ID:     candidate.ID,
			Kind:   regionSmartKindEdge,
			Label:  candidate.Label,
			Bounds: relative,
			Score:  0.72 + candidate.Confidence*0.12 - float64(index)*0.01,
		})
	}
	return candidates
}

func localImageParentRectsForRegionPoint(session RegionSelectionSession, point image.Point, captureBounds image.Rectangle, capturePoint image.Point, imageBounds image.Rectangle, parentCandidates []RegionSmartCandidate) []image.Rectangle {
	parents := make([]image.Rectangle, 0, len(parentCandidates)+1)
	seen := map[string]bool{}
	add := func(rect image.Rectangle) {
		rect = rect.Intersect(captureBounds)
		if rect.Empty() || !capturePoint.In(rect) {
			return
		}
		local := image.Rect(
			rect.Min.X-captureBounds.Min.X,
			rect.Min.Y-captureBounds.Min.Y,
			rect.Max.X-captureBounds.Min.X,
			rect.Max.Y-captureBounds.Min.Y,
		).Intersect(imageBounds)
		if local.Dx() < minRegionWidth || local.Dy() < minRegionHeight {
			return
		}
		key := fmt.Sprintf("%d:%d:%d:%d", local.Min.X, local.Min.Y, local.Dx(), local.Dy())
		if seen[key] {
			return
		}
		seen[key] = true
		parents = append(parents, local)
	}
	for _, candidate := range parentCandidates {
		if !regionRectContainsPoint(candidate.Bounds, point) {
			continue
		}
		add(mapRegionSelectionToCaptureRect(session, candidate.Bounds))
	}
	add(captureBounds)
	return parents
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
	if relative, ok := mapCaptureRectToDisplaySelection(session, rect); ok {
		return relative
	}
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

func mapRegionPointToCapturePoint(session RegionSelectionSession, point image.Point) image.Point {
	if capturePoint, ok := mapRegionPointToDisplayCapturePoint(session, point); ok {
		return capturePoint
	}
	capture := session.CaptureBounds
	if capture == nil || capture.Width <= 0 || capture.Height <= 0 || session.Bounds.Width <= 0 || session.Bounds.Height <= 0 {
		return image.Point{X: session.Bounds.X + point.X, Y: session.Bounds.Y + point.Y}
	}
	return image.Point{
		X: capture.X + scaleRegionValue(point.X, session.Bounds.Width, capture.Width),
		Y: capture.Y + scaleRegionValue(point.Y, session.Bounds.Height, capture.Height),
	}
}

type detectedImageRegion struct {
	ID         string
	Label      string
	Rect       image.Rectangle
	Confidence float64
}

func detectImageRegionsAroundPoint(img image.Image, point image.Point, parents []image.Rectangle) []detectedImageRegion {
	if img == nil || img.Bounds().Empty() || !point.In(img.Bounds()) {
		return nil
	}
	detected := make([]detectedImageRegion, 0, 4)
	seen := map[string]bool{}
	add := func(candidate detectedImageRegion) {
		candidate.Rect = candidate.Rect.Intersect(img.Bounds())
		if candidate.Confidence < 0.5 || candidate.Rect.Dx() < minRegionWidth || candidate.Rect.Dy() < minRegionHeight || !point.In(candidate.Rect) {
			return
		}
		key := fmt.Sprintf("%d:%d:%d:%d", candidate.Rect.Min.X, candidate.Rect.Min.Y, candidate.Rect.Dx(), candidate.Rect.Dy())
		if seen[key] {
			return
		}
		seen[key] = true
		detected = append(detected, candidate)
	}

	if rect, confidence := detectImageRegionAroundPoint(img, point); confidence >= 0.55 && !rect.Empty() {
		add(detectedImageRegion{
			ID:         "edge:hover",
			Label:      "Smart region",
			Rect:       rect,
			Confidence: confidence,
		})
	}
	if len(parents) == 0 {
		parents = []image.Rectangle{img.Bounds()}
	}
	for index, parent := range parents {
		for _, candidate := range detectImagePanelRegionsAroundPoint(img, point, parent) {
			candidate.ID = fmt.Sprintf("%s:%d", candidate.ID, index)
			add(candidate)
		}
	}

	sort.SliceStable(detected, func(left, right int) bool {
		leftArea := detected[left].Rect.Dx() * detected[left].Rect.Dy()
		rightArea := detected[right].Rect.Dx() * detected[right].Rect.Dy()
		leftScore := detected[left].Confidence + 160000.0/float64(maxInt(1, leftArea))
		rightScore := detected[right].Confidence + 160000.0/float64(maxInt(1, rightArea))
		if math.Abs(leftScore-rightScore) > 0.015 {
			return leftScore > rightScore
		}
		return leftArea < rightArea
	})
	return detected
}

type imagePanelBoundary struct {
	Position   int
	Score      int
	FromParent bool
}

func detectImagePanelRegionsAroundPoint(img image.Image, point image.Point, parent image.Rectangle) []detectedImageRegion {
	bounds := img.Bounds()
	parent = parent.Intersect(bounds)
	if parent.Dx() < minRegionWidth || parent.Dy() < minRegionHeight || !point.In(parent) {
		return nil
	}
	left := nearestVerticalPanelBoundary(img, point, parent, -1)
	right := nearestVerticalPanelBoundary(img, point, parent, 1)
	top := nearestHorizontalPanelBoundary(img, point, parent, -1)
	bottom := nearestHorizontalPanelBoundary(img, point, parent, 1)
	if right.Position-left.Position < minRegionWidth || bottom.Position-top.Position < minRegionHeight {
		return nil
	}

	candidates := make([]detectedImageRegion, 0, 2)
	verticalStrong := panelBoundaryStrongCount(left, right)
	horizontalStrong := panelBoundaryStrongCount(top, bottom)
	verticalPanel := image.Rect(left.Position, top.Position, right.Position, bottom.Position).Intersect(parent)
	if verticalStrong >= 1 &&
		verticalPanel.Dx() >= minRegionWidth &&
		verticalPanel.Dy() >= minRegionHeight &&
		verticalPanel.Dx() <= maxInt(420, parent.Dx()*58/100) {
		confidence := panelCandidateConfidence(parent, verticalPanel, verticalStrong, horizontalStrong, left, right, top, bottom)
		candidates = append(candidates, detectedImageRegion{
			ID:         "edge:panel-vertical",
			Label:      "Smart panel",
			Rect:       verticalPanel,
			Confidence: confidence,
		})
	}

	horizontalPanel := image.Rect(left.Position, top.Position, right.Position, bottom.Position).Intersect(parent)
	if horizontalStrong >= 1 &&
		horizontalPanel.Dx() >= minRegionWidth &&
		horizontalPanel.Dy() >= minRegionHeight &&
		horizontalPanel.Dy() <= maxInt(320, parent.Dy()*45/100) {
		confidence := panelCandidateConfidence(parent, horizontalPanel, horizontalStrong, verticalStrong, top, bottom, left, right)
		candidates = append(candidates, detectedImageRegion{
			ID:         "edge:panel-horizontal",
			Label:      "Smart panel",
			Rect:       horizontalPanel,
			Confidence: confidence,
		})
	}
	return candidates
}

func nearestVerticalPanelBoundary(img image.Image, point image.Point, parent image.Rectangle, direction int) imagePanelBoundary {
	best := imagePanelBoundary{Position: point.X, Score: -1}
	maxSearch := minInt(1400, maxInt(parent.Dx(), 260))
	startY := maxInt(parent.Min.Y, point.Y-520)
	endY := minInt(parent.Max.Y, point.Y+520)
	if direction < 0 {
		limit := maxInt(parent.Min.X+1, point.X-maxSearch)
		for current := point.X; current >= limit; current-- {
			score := averageVerticalPanelBoundaryScore(img, current, startY, endY)
			if score > best.Score {
				best = imagePanelBoundary{Position: current, Score: score}
			}
			if score >= regionSmartEdgeThreshold && absInt(current-point.X) >= minRegionWidth/2 {
				break
			}
		}
		if best.Score >= regionSmartEdgeThreshold {
			return best
		}
		return imagePanelBoundary{Position: parent.Min.X, Score: regionSmartEdgeThreshold, FromParent: true}
	}
	limit := minInt(parent.Max.X-1, point.X+maxSearch)
	for current := point.X; current <= limit; current++ {
		score := averageVerticalPanelBoundaryScore(img, current, startY, endY)
		if score > best.Score {
			best = imagePanelBoundary{Position: current, Score: score}
		}
		if score >= regionSmartEdgeThreshold && absInt(current-point.X) >= minRegionWidth/2 {
			break
		}
	}
	if best.Score >= regionSmartEdgeThreshold {
		return best
	}
	return imagePanelBoundary{Position: parent.Max.X, Score: regionSmartEdgeThreshold, FromParent: true}
}

func nearestHorizontalPanelBoundary(img image.Image, point image.Point, parent image.Rectangle, direction int) imagePanelBoundary {
	best := imagePanelBoundary{Position: point.Y, Score: -1}
	maxSearch := minInt(1400, maxInt(parent.Dy(), 260))
	startX := maxInt(parent.Min.X, point.X-620)
	endX := minInt(parent.Max.X, point.X+620)
	if direction < 0 {
		limit := maxInt(parent.Min.Y+1, point.Y-maxSearch)
		for current := point.Y; current >= limit; current-- {
			score := averageHorizontalPanelBoundaryScore(img, current, startX, endX)
			if score > best.Score {
				best = imagePanelBoundary{Position: current, Score: score}
			}
			if score >= regionSmartEdgeThreshold && absInt(current-point.Y) >= minRegionHeight/2 {
				break
			}
		}
		if best.Score >= regionSmartEdgeThreshold {
			return best
		}
		return imagePanelBoundary{Position: parent.Min.Y, Score: regionSmartEdgeThreshold, FromParent: true}
	}
	limit := minInt(parent.Max.Y-1, point.Y+maxSearch)
	for current := point.Y; current <= limit; current++ {
		score := averageHorizontalPanelBoundaryScore(img, current, startX, endX)
		if score > best.Score {
			best = imagePanelBoundary{Position: current, Score: score}
		}
		if score >= regionSmartEdgeThreshold && absInt(current-point.Y) >= minRegionHeight/2 {
			break
		}
	}
	if best.Score >= regionSmartEdgeThreshold {
		return best
	}
	return imagePanelBoundary{Position: parent.Max.Y, Score: regionSmartEdgeThreshold, FromParent: true}
}

func panelBoundaryStrongCount(boundaries ...imagePanelBoundary) int {
	count := 0
	for _, boundary := range boundaries {
		if !boundary.FromParent && boundary.Score >= regionSmartEdgeThreshold {
			count++
		}
	}
	return count
}

func panelCandidateConfidence(parent image.Rectangle, rect image.Rectangle, primaryStrong int, secondaryStrong int, boundaries ...imagePanelBoundary) float64 {
	scoreTotal := 0
	parentEdges := 0
	for _, boundary := range boundaries {
		scoreTotal += maxInt(0, boundary.Score)
		if boundary.FromParent {
			parentEdges++
		}
	}
	confidence := 0.52 + math.Min(0.26, float64(scoreTotal)/float64(maxInt(1, len(boundaries))*regionSmartEdgeThreshold*10))
	confidence += float64(primaryStrong) * 0.06
	confidence += float64(secondaryStrong) * 0.025
	confidence -= float64(maxInt(0, parentEdges-2)) * 0.035
	areaRatio := float64(rect.Dx()*rect.Dy()) / float64(maxInt(1, parent.Dx()*parent.Dy()))
	if areaRatio > 0.82 {
		confidence -= 0.18
	} else if areaRatio < 0.72 {
		confidence += 0.04
	}
	return math.Max(0, math.Min(1, confidence))
}

func detectImageRegionAroundPoint(img image.Image, point image.Point) (image.Rectangle, float64) {
	if img == nil {
		return image.Rectangle{}, 0
	}
	bounds := img.Bounds()
	if bounds.Empty() || !point.In(bounds) {
		return image.Rectangle{}, 0
	}
	maxSearchX := minInt(900, maxInt(bounds.Dx()/2, 220))
	maxSearchY := minInt(700, maxInt(bounds.Dy()/2, 180))
	left, leftScore := nearestStrongVerticalBoundary(img, point.X, point.Y, -1, maxSearchX)
	right, rightScore := nearestStrongVerticalBoundary(img, point.X, point.Y, 1, maxSearchX)
	top, topScore := nearestStrongHorizontalBoundary(img, point.X, point.Y, -1, maxSearchY)
	bottom, bottomScore := nearestStrongHorizontalBoundary(img, point.X, point.Y, 1, maxSearchY)
	if right-left < minRegionWidth || bottom-top < minRegionHeight {
		return image.Rectangle{}, 0
	}
	rect := image.Rect(left, top, right, bottom).Intersect(bounds)
	if rect.Dx() < minRegionWidth || rect.Dy() < minRegionHeight {
		return image.Rectangle{}, 0
	}
	scoreTotal := leftScore + rightScore + topScore + bottomScore
	confidence := math.Min(1, float64(scoreTotal)/float64(regionSmartEdgeThreshold*5))
	area := rect.Dx() * rect.Dy()
	if area > bounds.Dx()*bounds.Dy()*3/4 {
		confidence *= 0.72
	}
	return rect, confidence
}

func nearestStrongVerticalBoundary(img image.Image, x int, y int, direction int, maxSearch int) (int, int) {
	bounds := img.Bounds()
	bestX := x
	bestScore := -1
	limit := x + direction*maxSearch
	if direction < 0 {
		limit = maxInt(bounds.Min.X+1, limit)
		for current := x; current >= limit; current-- {
			score := localVerticalBoundaryScore(img, current, y)
			if score > bestScore {
				bestScore = score
				bestX = current
			}
			if bestScore >= regionSmartEdgeThreshold*2 && absInt(current-x) >= minRegionWidth/2 {
				break
			}
		}
		return maxInt(bounds.Min.X, bestX), bestScore
	}
	limit = minInt(bounds.Max.X-1, limit)
	for current := x; current <= limit; current++ {
		score := localVerticalBoundaryScore(img, current, y)
		if score > bestScore {
			bestScore = score
			bestX = current
		}
		if bestScore >= regionSmartEdgeThreshold*2 && absInt(current-x) >= minRegionWidth/2 {
			break
		}
	}
	return minInt(bounds.Max.X, bestX), bestScore
}

func nearestStrongHorizontalBoundary(img image.Image, x int, y int, direction int, maxSearch int) (int, int) {
	bounds := img.Bounds()
	bestY := y
	bestScore := -1
	limit := y + direction*maxSearch
	if direction < 0 {
		limit = maxInt(bounds.Min.Y+1, limit)
		for current := y; current >= limit; current-- {
			score := localHorizontalBoundaryScore(img, x, current)
			if score > bestScore {
				bestScore = score
				bestY = current
			}
			if bestScore >= regionSmartEdgeThreshold*2 && absInt(current-y) >= minRegionHeight/2 {
				break
			}
		}
		return maxInt(bounds.Min.Y, bestY), bestScore
	}
	limit = minInt(bounds.Max.Y-1, limit)
	for current := y; current <= limit; current++ {
		score := localHorizontalBoundaryScore(img, x, current)
		if score > bestScore {
			bestScore = score
			bestY = current
		}
		if bestScore >= regionSmartEdgeThreshold*2 && absInt(current-y) >= minRegionHeight/2 {
			break
		}
	}
	return minInt(bounds.Max.Y, bestY), bestScore
}

func localVerticalBoundaryScore(img image.Image, x int, centerY int) int {
	bounds := img.Bounds()
	if x <= bounds.Min.X || x >= bounds.Max.X {
		return 0
	}
	startY := maxInt(bounds.Min.Y, centerY-420)
	endY := minInt(bounds.Max.Y, centerY+420)
	return averageVerticalEdgeScore(img, x, startY, endY)
}

func localHorizontalBoundaryScore(img image.Image, centerX int, y int) int {
	bounds := img.Bounds()
	if y <= bounds.Min.Y || y >= bounds.Max.Y {
		return 0
	}
	startX := maxInt(bounds.Min.X, centerX-520)
	endX := minInt(bounds.Max.X, centerX+520)
	return averageHorizontalEdgeScore(img, y, startX, endX)
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

func averageVerticalPanelBoundaryScore(img image.Image, x int, startY int, endY int) int {
	if endY <= startY {
		return 0
	}
	step := maxInt(1, (endY-startY)/128)
	total := 0
	weakSamples := 0
	strongSamples := 0
	samples := 0
	for y := startY; y < endY; y += step {
		distance := regionEdgeColorDistance(img.At(x-1, y), img.At(x, y))
		total += distance
		if distance >= regionSmartWeakEdgePx {
			weakSamples++
		}
		if distance >= regionSmartEdgeThreshold {
			strongSamples++
		}
		samples++
	}
	return panelBoundaryContinuityScore(total, weakSamples, strongSamples, samples)
}

func averageHorizontalPanelBoundaryScore(img image.Image, y int, startX int, endX int) int {
	if endX <= startX {
		return 0
	}
	step := maxInt(1, (endX-startX)/128)
	total := 0
	weakSamples := 0
	strongSamples := 0
	samples := 0
	for x := startX; x < endX; x += step {
		distance := regionEdgeColorDistance(img.At(x, y-1), img.At(x, y))
		total += distance
		if distance >= regionSmartWeakEdgePx {
			weakSamples++
		}
		if distance >= regionSmartEdgeThreshold {
			strongSamples++
		}
		samples++
	}
	return panelBoundaryContinuityScore(total, weakSamples, strongSamples, samples)
}

func panelBoundaryContinuityScore(total int, weakSamples int, strongSamples int, samples int) int {
	if samples <= 0 {
		return 0
	}
	average := total / samples
	weakCoverage := weakSamples * 100 / samples
	strongCoverage := strongSamples * 100 / samples
	score := average
	if weakCoverage >= 45 && average >= regionSmartWeakEdgePx {
		score += minInt(14, weakCoverage/6)
	}
	if strongCoverage >= 18 {
		score += minInt(10, strongCoverage/5)
	}
	return score
}

func regionEdgeColorDistance(a color.Color, b color.Color) int {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return (absInt(int(ar>>8)-int(br>>8)) +
		absInt(int(ag>>8)-int(bg>>8)) +
		absInt(int(ab>>8)-int(bb>>8)) +
		absInt(int(aa>>8)-int(ba>>8))) / 3
}
