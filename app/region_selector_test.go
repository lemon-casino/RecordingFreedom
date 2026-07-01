package main

import (
	"runtime"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestNormalizeRegionSelectionAcceptsReverseDrag(t *testing.T) {
	got := normalizeRegionSelection(RegionSelectionRequest{
		X:      400,
		Y:      300,
		Width:  -240,
		Height: -120,
	})

	if got.X != 160 || got.Y != 180 || got.Width != 240 || got.Height != 120 {
		t.Fatalf("normalizeRegionSelection() = %#v, want normalized positive rect", got)
	}
}

func TestRegionCaptureSourcePreservesGeometryAndPlatformAvailability(t *testing.T) {
	got := regionCaptureSource(application.Rect{X: -120, Y: 80, Width: 1728, Height: 906}, devices.CaptureSource{})

	if got.ID != "region:custom" || got.Type != devices.SourceRegion {
		t.Fatalf("region source identity = %#v", got)
	}
	if got.X != -120 || got.Y != 80 || got.Width != 1728 || got.Height != 906 {
		t.Fatalf("region source geometry = (%d,%d %dx%d), want selected geometry", got.X, got.Y, got.Width, got.Height)
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if !got.Available || got.Capability != devices.CapabilityEnumerated || got.UnavailableReason != "" {
			t.Fatalf("region source availability = %#v, want available desktop crop source", got)
		}
	} else if got.Available || got.Capability != devices.CapabilityNativeQueued || got.UnavailableReason == "" {
		t.Fatalf("region source availability = %#v, want queued until native crop writer lands", got)
	}
}

func TestRegionDisplayForRectRequiresSingleDisplayContainment(t *testing.T) {
	sources := []devices.CaptureSource{
		{ID: "screen:left", Type: devices.SourceScreen, X: -1920, Y: 0, Width: 1920, Height: 1080, NativeID: "cgdisplay:1", DisplayIndex: 1},
		{ID: "screen:right", Type: devices.SourceScreen, X: 0, Y: 0, Width: 2560, Height: 1440, NativeID: "cgdisplay:2", DisplayIndex: 2},
	}

	got := regionDisplayForRect(application.Rect{X: 20, Y: 30, Width: 800, Height: 600}, sources, nil)
	if got.ID != "screen:right" || got.NativeID != "cgdisplay:2" {
		t.Fatalf("region display = %#v, want right display", got)
	}

	crossScreen := regionDisplayForRect(application.Rect{X: -100, Y: 30, Width: 300, Height: 600}, sources, nil)
	if crossScreen.ID != "" {
		t.Fatalf("cross-screen region display = %#v, want empty display target", crossScreen)
	}
}

func TestRegionDisplayForRectPrefersWailsPhysicalBounds(t *testing.T) {
	sources := []devices.CaptureSource{
		{ID: "screen:primary", Type: devices.SourceScreen, X: 0, Y: 0, Width: 3024, Height: 1964, NativeID: "cgdisplay:1", DisplayIndex: 1},
		{ID: "screen:external", Type: devices.SourceScreen, X: 9999, Y: 9999, Width: 1, Height: 1, NativeID: "cgdisplay:2", DisplayIndex: 2},
	}
	got := regionDisplayForRect(application.Rect{X: 3024, Y: 0, Width: 1280, Height: 720}, sources, []*application.Screen{
		{PhysicalBounds: application.Rect{X: 0, Y: 0, Width: 3024, Height: 1964}},
		{PhysicalBounds: application.Rect{X: 3024, Y: 0, Width: 2560, Height: 1440}},
	})

	if got.ID != "screen:external" || got.NativeID != "cgdisplay:2" {
		t.Fatalf("region display = %#v, want display matched by Wails physical bounds", got)
	}
}

func TestRegionOverlayBoundsUseVirtualDesktop(t *testing.T) {
	bounds, displayCount := regionOverlayBounds([]*application.Screen{
		{
			ID:        "left",
			Bounds:    application.Rect{X: -1440, Y: 0, Width: 1440, Height: 900},
			IsPrimary: false,
		},
		{
			ID:        "primary",
			Bounds:    application.Rect{X: 0, Y: 0, Width: 2560, Height: 1440},
			IsPrimary: true,
		},
	})

	if displayCount != 2 {
		t.Fatalf("displayCount = %d, want 2", displayCount)
	}
	if bounds.X != -1440 || bounds.Y != 0 || bounds.Width != 4000 || bounds.Height != 1440 {
		t.Fatalf("regionOverlayBounds() = %#v, want virtual desktop bounds", bounds)
	}
}
