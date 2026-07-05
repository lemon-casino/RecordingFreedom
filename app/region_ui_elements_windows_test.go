//go:build windows

package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeRegionUIElementRectsPrefersSmallestUniqueContainingRects(t *testing.T) {
	point := image.Point{X: 120, Y: 140}
	rects := normalizeRegionUIElementRects([]regionUIElementRect{
		{Rect: image.Rect(0, 0, 400, 400), Label: "window", Source: "content:window"},
		{Rect: image.Rect(80, 100, 220, 240), Label: "pane", Source: "raw:window"},
		{Rect: image.Rect(80, 100, 220, 240), Label: "duplicate", Source: "control:window"},
		{Rect: image.Rect(100, 120, 150, 170), Label: "too-small", Source: "raw:window"},
		{Rect: image.Rect(260, 260, 360, 360), Label: "outside", Source: "raw:window"},
	}, point)

	if len(rects) != 3 {
		t.Fatalf("rect count = %d, want 3: %#v", len(rects), rects)
	}
	if got := rects[0].Label; got != "too-small" {
		t.Fatalf("first candidate = %q, want smallest containing rect", got)
	}
	if got := rects[1].Label; got != "pane" {
		t.Fatalf("second candidate = %q, want deduped parent pane", got)
	}
	if got := rects[2].Label; got != "window" {
		t.Fatalf("third candidate = %q, want largest parent window", got)
	}
}

func TestRegionUIElementProbeFromEnv(t *testing.T) {
	value := strings.TrimSpace(os.Getenv("RECORDINGFREEDOM_REGION_PROBE"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("RECORDINGFREEDOM_UIA_PROBE"))
	}
	if value == "" {
		t.Skip("set RECORDINGFREEDOM_REGION_PROBE=cursor or x,y to print the real desktop UIA candidate chain")
	}
	point, err := parseRegionUIProbePoint(value)
	if err != nil {
		t.Fatal(err)
	}
	rects, err := collectRegionUIElementRects(point)
	if err != nil {
		t.Fatalf("collect UIA elements at %v: %v", point, err)
	}
	if len(rects) == 0 {
		t.Fatalf("no UIA element candidates at %v", point)
	}
	for index, rect := range rects {
		t.Logf(
			"candidate[%02d] source=%s control=%s label=%q rect=%d,%d %dx%d",
			index,
			rect.Source,
			regionUIControlTypeName(rect.ControlType),
			safeRegionElementLabel(rect.Label, regionUIControlTypeName(rect.ControlType)),
			rect.Rect.Min.X,
			rect.Rect.Min.Y,
			rect.Rect.Dx(),
			rect.Rect.Dy(),
		)
	}
}

func parseRegionUIProbePoint(value string) (image.Point, error) {
	if strings.EqualFold(strings.TrimSpace(value), "cursor") {
		return uiaCursorPosition()
	}
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return image.Point{}, fmt.Errorf("region probe must be cursor or x,y, got %q", value)
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return image.Point{}, fmt.Errorf("parse probe x from %q: %w", value, err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return image.Point{}, fmt.Errorf("parse probe y from %q: %w", value, err)
	}
	return image.Point{X: x, Y: y}, nil
}
