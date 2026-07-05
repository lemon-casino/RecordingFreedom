package main

import (
	"image"
	"testing"
)

const regionAssistProbeOverlayRadius = 4096

func runRegionAssistDesktopProbe(t *testing.T, point image.Point) {
	t.Helper()
	session := RegionSelectionSession{
		ID: "desktop-region-assist-probe",
		Bounds: RegionRect{
			X:      point.X - regionAssistProbeOverlayRadius,
			Y:      point.Y - regionAssistProbeOverlayRadius,
			Width:  regionAssistProbeOverlayRadius * 2,
			Height: regionAssistProbeOverlayRadius * 2,
		},
		MinimumWidth:  minRegionWidth,
		MinimumHeight: minRegionHeight,
		DisplayCount:  1,
		Purpose:       regionSelectionPurposeScreenshot,
	}
	relativePoint := image.Point{
		X: point.X - session.Bounds.X,
		Y: point.Y - session.Bounds.Y,
	}
	service := &RecordingFreedomService{}
	service.regionSession = &session
	service.resetRegionElementCache(session.ID)
	if candidates := service.regionElementCandidatesAtPoint(session, relativePoint); len(candidates) == 0 {
		t.Fatalf("no native element candidates at %v; check UI Automation/Accessibility permission and hover a visible UI control", point)
	}
	result, err := service.AssistRegionSelection(RegionAssistRequest{
		SessionID: session.ID,
		Purpose:   session.Purpose,
		PointerX:  relativePoint.X,
		PointerY:  relativePoint.Y,
	})
	if err != nil {
		t.Fatalf("assist region selection at %v: %v", point, err)
	}
	if result.Source != "element" {
		t.Fatalf("assist source = %q, want element; best=%#v candidates=%#v", result.Source, result.Best, result.Candidates)
	}
	if result.Best == nil || result.Best.Kind != regionSmartKindElement {
		t.Fatalf("assist best = %#v, want element candidate", result.Best)
	}
	if !regionRectContainsPoint(result.Best.Bounds, relativePoint) {
		t.Fatalf("assist best bounds %#v do not contain relative point %v", result.Best.Bounds, relativePoint)
	}
	t.Logf(
		"region-assist source=%s best=%s label=%q sourceId=%s bounds=%d,%d %dx%d candidates=%d",
		result.Source,
		result.Best.ID,
		result.Best.Label,
		result.Best.SourceID,
		result.Best.Bounds.X,
		result.Best.Bounds.Y,
		result.Best.Bounds.Width,
		result.Best.Bounds.Height,
		len(result.Candidates),
	)
}
