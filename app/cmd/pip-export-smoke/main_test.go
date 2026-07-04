package main

import (
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
)

func TestRunRejectsNegativeCanvas(t *testing.T) {
	if _, err := run(options{canvasWidth: -1, timeout: time.Second}); err == nil {
		t.Fatal("run() error = nil, want negative canvas rejection")
	}
}

func TestVerifyPIPPixelsFromFrameAcceptsVisibleCameraRegion(t *testing.T) {
	width := 160
	height := 90
	frame := make([]byte, width*height*3)
	layout := pip.Placement{
		Visible: true,
		Rect:    pip.Rect{X: 120, Y: 50, Width: 24, Height: 24, Visible: true},
	}
	for y := layout.Rect.Y; y < layout.Rect.Y+layout.Rect.Height; y++ {
		for x := layout.Rect.X; x < layout.Rect.X+layout.Rect.Width; x++ {
			offset := (y*width + x) * 3
			frame[offset] = syntheticPIPColor.R
			frame[offset+1] = syntheticPIPColor.G
			frame[offset+2] = syntheticPIPColor.B
		}
	}

	pixel, background, err := verifyPIPPixelsFromFrame(frame, width, height, layout, syntheticPIPColor)
	if err != nil {
		t.Fatalf("verifyPIPPixelsFromFrame() error = %v", err)
	}
	if pixel.R < 200 || background.R != 0 || background.G != 0 || background.B != 0 {
		t.Fatalf("pixels = %#v background=%#v, want red PIP and dark background", pixel, background)
	}
}

func TestVerifyPIPPixelsFromFrameRejectsMissingCameraRegion(t *testing.T) {
	width := 160
	height := 90
	frame := make([]byte, width*height*3)
	layout := pip.Placement{
		Visible: true,
		Rect:    pip.Rect{X: 120, Y: 50, Width: 24, Height: 24, Visible: true},
	}

	if _, _, err := verifyPIPPixelsFromFrame(frame, width, height, layout, syntheticPIPColor); err == nil {
		t.Fatal("verifyPIPPixelsFromFrame() error = nil, want missing PIP pixel failure")
	}
}
