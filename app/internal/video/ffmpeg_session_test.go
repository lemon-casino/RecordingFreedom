package video

import (
	"path/filepath"
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

func TestFFmpegSegmentDirIsUniquePerOutputFile(t *testing.T) {
	root := t.TempDir()
	screen := &ffmpegDesktopSession{config: CaptureConfig{OutputPath: filepath.Join(root, "screen.mp4")}}
	webcam := &ffmpegDesktopSession{config: CaptureConfig{OutputPath: filepath.Join(root, "webcam.mp4")}}
	if screen.segmentDir() == webcam.segmentDir() {
		t.Fatalf("screen and webcam segment dirs both = %q, want isolated writers", screen.segmentDir())
	}
	if got := filepath.Base(screen.segmentDir()); got != "screen" {
		t.Fatalf("screen segment dir leaf = %q, want screen", got)
	}
	if got := filepath.Base(webcam.segmentDir()); got != "webcam" {
		t.Fatalf("webcam segment dir leaf = %q, want webcam", got)
	}
}
