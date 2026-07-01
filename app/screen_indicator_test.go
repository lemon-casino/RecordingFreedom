package main

import (
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestScreenIndicatorBoundsCenterCardOnDisplay(t *testing.T) {
	display := application.Rect{X: 1920, Y: 0, Width: 2560, Height: 1440}
	got := screenIndicatorBounds(display)

	if got.Width != screenIndicatorMaxWidth || got.Height != screenIndicatorMaxHeight {
		t.Fatalf("indicator size = %dx%d, want capped %dx%d", got.Width, got.Height, screenIndicatorMaxWidth, screenIndicatorMaxHeight)
	}
	wantX := display.X + (display.Width-got.Width)/2
	wantY := display.Y + (display.Height-got.Height)/2
	if got.X != wantX || got.Y != wantY {
		t.Fatalf("indicator origin = %d,%d, want centered on second display", got.X, got.Y)
	}
}

func TestScreenSourceDisplayBoundsPrefersWailsScreenByDisplayIndex(t *testing.T) {
	source := devices.CaptureSource{
		ID:           "screen:display-2",
		Type:         devices.SourceScreen,
		X:            9999,
		Y:            9999,
		Width:        1,
		Height:       1,
		DisplayIndex: 2,
	}
	got := screenSourceDisplayBounds(source, []*application.Screen{
		{Bounds: application.Rect{X: 0, Y: 0, Width: 1280, Height: 720}},
		{Bounds: application.Rect{X: 1280, Y: 0, Width: 1920, Height: 1080}},
	})

	if got.X != 1280 || got.Y != 0 || got.Width != 1920 || got.Height != 1080 {
		t.Fatalf("display bounds = %#v, want Wails screen 2 bounds", got)
	}
}

func TestScreenSourceDisplayBoundsMatchesPhysicalBoundsBeforeDisplayIndex(t *testing.T) {
	source := devices.CaptureSource{
		ID:           "screen:display-2",
		Type:         devices.SourceScreen,
		X:            2560,
		Y:            0,
		Width:        3840,
		Height:       2160,
		DisplayIndex: 1,
	}
	got := screenSourceDisplayBounds(source, []*application.Screen{
		{
			Bounds:         application.Rect{X: 0, Y: 0, Width: 1280, Height: 720},
			PhysicalBounds: application.Rect{X: 0, Y: 0, Width: 2560, Height: 1440},
		},
		{
			Bounds:         application.Rect{X: 1280, Y: 0, Width: 1920, Height: 1080},
			PhysicalBounds: application.Rect{X: 2560, Y: 0, Width: 3840, Height: 2160},
		},
	})

	if got.X != 1280 || got.Y != 0 || got.Width != 1920 || got.Height != 1080 {
		t.Fatalf("display bounds = %#v, want screen matched by physical bounds", got)
	}
}

func TestFindScreenSourceOnlyReturnsScreenType(t *testing.T) {
	sources := []devices.CaptureSource{
		{ID: "application:1", Type: devices.SourceApplication},
		{ID: "screen:display-1", Type: devices.SourceScreen, DisplayIndex: 1},
	}

	got, ok := findScreenSource(sources, "screen:display-1")
	if !ok || got.DisplayIndex != 1 {
		t.Fatalf("findScreenSource() = %#v, %v; want display 1", got, ok)
	}
	if _, ok := findScreenSource(sources, "application:1"); ok {
		t.Fatal("findScreenSource returned an application source")
	}
}
