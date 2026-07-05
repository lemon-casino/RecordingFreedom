package main

import (
	"fmt"
	"image"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

func regionWindowCandidatesFromSources(s *RecordingFreedomService, session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	if s == nil || s.devices == nil {
		return nil
	}
	return regionWindowCandidatesFromCaptureSources(s.devices.ListSources(), session, point)
}

func regionWindowCandidatesFromCaptureSources(sources []devices.CaptureSource, session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	absolutePoint := mapRegionPointToAbsolutePoint(session, point)
	candidates := make([]RegionSmartCandidate, 0, 4)
	seen := map[string]bool{}
	minWidth, minHeight := regionSmartCandidateMinimumSize(session)
	for index, source := range sources {
		if source.Type != devices.SourceWindow || source.Width <= 0 || source.Height <= 0 {
			continue
		}
		absolute := RegionRect{X: source.X, Y: source.Y, Width: source.Width, Height: source.Height}
		if !regionRectContainsPoint(absolute, absolutePoint) {
			continue
		}
		relative := clampRegionCandidateBounds(regionRelativeRectFromAbsolute(session, absolute), session.Bounds)
		if relative.Width < minWidth || relative.Height < minHeight {
			continue
		}
		key := fmt.Sprintf("%d:%d:%d:%d", relative.X, relative.Y, relative.Width, relative.Height)
		if seen[key] {
			continue
		}
		seen[key] = true
		candidates = append(candidates, RegionSmartCandidate{
			ID:       source.ID,
			Kind:     regionSmartKindWindow,
			Label:    safeRegionElementLabel(source.Name, "Window"),
			SourceID: source.ID,
			Bounds:   relative,
			Score:    0.72 - float64(index)*0.005,
		})
	}
	return candidates
}
