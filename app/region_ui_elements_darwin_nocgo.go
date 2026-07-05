//go:build darwin && !cgo

package main

import "image"

func (s *RecordingFreedomService) regionElementCandidatesAtPoint(session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	return nil
}

func (s *RecordingFreedomService) regionWindowCandidatesAtPoint(session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	return regionWindowCandidatesFromSources(s, session, point)
}

func currentRegionCursorPoint(session RegionSelectionSession) (image.Point, bool) {
	return image.Point{}, false
}
