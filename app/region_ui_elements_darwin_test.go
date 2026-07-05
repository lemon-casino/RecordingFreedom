//go:build darwin && cgo

package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeRegionAXElementRectsPrefersSmallestUniqueContainingRects(t *testing.T) {
	point := image.Point{X: 120, Y: 140}
	rects := normalizeRegionAXElementRects([]regionAXElementRect{
		{Rect: image.Rect(0, 0, 400, 400), Label: "window", Source: "accessibility:ancestor"},
		{Rect: image.Rect(80, 100, 220, 240), Label: "pane", Source: "accessibility:ancestor"},
		{Rect: image.Rect(80, 100, 220, 240), Label: "duplicate", Source: "accessibility:ancestor"},
		{Rect: image.Rect(100, 120, 150, 170), Label: "control", Source: "accessibility:point"},
		{Rect: image.Rect(260, 260, 360, 360), Label: "outside", Source: "accessibility:ancestor"},
		{Rect: image.Rect(0, 0, 300000, 300000), Label: "invalid", Source: "accessibility:ancestor"},
	}, point)

	if len(rects) != 3 {
		t.Fatalf("rect count = %d, want 3: %#v", len(rects), rects)
	}
	if got := rects[0].Label; got != "control" {
		t.Fatalf("first candidate = %q, want smallest containing rect", got)
	}
	if got := rects[1].Label; got != "pane" {
		t.Fatalf("second candidate = %q, want deduped parent pane", got)
	}
	if got := rects[2].Label; got != "window" {
		t.Fatalf("third candidate = %q, want largest parent window", got)
	}
}

func TestRegionAccessibilityProbeFromEnv(t *testing.T) {
	value := strings.TrimSpace(os.Getenv("RECORDINGFREEDOM_REGION_PROBE"))
	if value == "" {
		t.Skip("set RECORDINGFREEDOM_REGION_PROBE=cursor or x,y to print the real desktop Accessibility candidate chain")
	}
	point, err := parseRegionAXProbePoint(value)
	if err != nil {
		t.Fatal(err)
	}
	rects, err := collectRegionAXElementRects(point, os.Getpid())
	if err != nil {
		t.Fatalf("collect Accessibility elements at %v: %v", point, err)
	}
	if len(rects) == 0 {
		t.Fatalf("no Accessibility element candidates at %v; check macOS Accessibility permission", point)
	}
	for index, rect := range rects {
		t.Logf(
			"candidate[%02d] source=%s role=%s label=%q pid=%d rect=%d,%d %dx%d",
			index,
			rect.Source,
			rect.Role,
			safeRegionElementLabel(rect.Label, strings.TrimPrefix(strings.TrimSpace(rect.Role), "AX")),
			rect.PID,
			rect.Rect.Min.X,
			rect.Rect.Min.Y,
			rect.Rect.Dx(),
			rect.Rect.Dy(),
		)
	}
}

func parseRegionAXProbePoint(value string) (image.Point, error) {
	if strings.EqualFold(strings.TrimSpace(value), "cursor") {
		return darwinAXCursorPosition()
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
