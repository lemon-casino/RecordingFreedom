package video

import (
	"slices"
	"testing"
)

func TestFFmpegOutputDimensionsAreEven(t *testing.T) {
	config := CaptureConfig{
		SourceGeometry: &SourceGeometry{
			Width:  1707,
			Height: 959,
		},
	}
	if got := captureWidth(config); got != 1708 {
		t.Fatalf("captureWidth() = %d, want padded even width 1708", got)
	}
	if got := captureHeight(config); got != 960 {
		t.Fatalf("captureHeight() = %d, want padded even height 960", got)
	}
}

func TestFFmpegEncodingArgsPadOddDesktopDimensions(t *testing.T) {
	session := &ffmpegDesktopSession{}
	args := session.encodingArgs("screen.mp4")
	if !slices.Contains(args, "-vf") || !slices.Contains(args, "pad=ceil(iw/2)*2:ceil(ih/2)*2") {
		t.Fatalf("encoding args = %#v, want even-dimension padding filter", args)
	}
}
