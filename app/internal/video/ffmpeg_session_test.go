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
	args := session.encodingArgs("screen.mp4", ffmpegInputSpec{})
	if !slices.Contains(args, "-vf") || !slices.Contains(args, "pad=ceil(iw/2)*2:ceil(ih/2)*2") {
		t.Fatalf("encoding args = %#v, want even-dimension padding filter", args)
	}
}

func TestFFmpegEncodingArgsUsesSegmentMuxer(t *testing.T) {
	t.Setenv(EnvFFmpegSegmentSeconds, "12")
	session := &ffmpegDesktopSession{}
	args := session.encodingArgs("segment-%03d.mp4", ffmpegInputSpec{})
	if ffmpegTestFlagValue(args, "-f") != "segment" {
		t.Fatalf("-f = %q, want segment in args %v", ffmpegTestFlagValue(args, "-f"), args)
	}
	if ffmpegTestFlagValue(args, "-segment_time") != "12" {
		t.Fatalf("-segment_time = %q, want 12 in args %v", ffmpegTestFlagValue(args, "-segment_time"), args)
	}
	if !slices.Contains(args, "-reset_timestamps") {
		t.Fatalf("encoding args = %#v, want reset timestamps for concat-safe chunks", args)
	}
}

func TestFFmpegEncodingArgsWritesPreviewImageAsSecondOutput(t *testing.T) {
	t.Setenv(EnvFFmpegSegmentSeconds, "12")
	previewPath := filepath.Join("cache", "pip-camera-preview.jpg")
	session := &ffmpegDesktopSession{}
	args := session.encodingArgs("segment-%03d.mp4", ffmpegInputSpec{
		PreviewImagePath:  previewPath,
		PreviewImageFPS:   30,
		PreviewImageWidth: 999,
	})

	segmentIndex := slices.Index(args, "segment-%03d.mp4")
	previewIndex := slices.Index(args, previewPath)
	if segmentIndex < 0 || previewIndex < 0 || previewIndex <= segmentIndex {
		t.Fatalf("encoding args = %#v, want segment output followed by preview image output", args)
	}
	if filter := ffmpegTestFlagValue(args, "-filter_complex"); filter != "[0:v]split=2[rf_record_src][rf_preview_src];[rf_record_src]pad=ceil(iw/2)*2:ceil(ih/2)*2[rf_record];[rf_preview_src]fps=15,scale=720:-2[rf_preview]" {
		t.Fatalf("filter_complex = %q, want split recording and preview graph in args %v", filter, args)
	}
	if ffmpegTestFlagValue(args, "-map") != "[rf_record]" {
		t.Fatalf("recording -map = %q, want split recording stream in args %v", ffmpegTestFlagValue(args, "-map"), args)
	}
	if ffmpegTestFlagValueAfter(args, segmentIndex, "-map") != "[rf_preview]" {
		t.Fatalf("preview -map = %q, want split preview stream in args %v", ffmpegTestFlagValueAfter(args, segmentIndex, "-map"), args)
	}
	if ffmpegTestFlagValueAfter(args, segmentIndex, "-f") != "image2" || ffmpegTestFlagValueAfter(args, segmentIndex, "-update") != "1" {
		t.Fatalf("preview output args = %#v, want image2 -update 1", args[segmentIndex+1:])
	}
}

func TestFFmpegEncodingArgsKeepsLegacyPreviewOutputForPreFilteredInput(t *testing.T) {
	previewPath := filepath.Join("cache", "pip-camera-preview.jpg")
	session := &ffmpegDesktopSession{}
	args := session.encodingArgs("segment-%03d.mp4", ffmpegInputSpec{
		VideoPreFiltered: true,
		PreviewImagePath: previewPath,
	})
	segmentIndex := slices.Index(args, "segment-%03d.mp4")
	if segmentIndex < 0 {
		t.Fatalf("encoding args = %#v, want segment output", args)
	}
	if ffmpegTestFlagValueAfter(args, segmentIndex, "-map") != "0:v:0" {
		t.Fatalf("preview -map = %q, want original mapped input for pre-filtered capture", ffmpegTestFlagValueAfter(args, segmentIndex, "-map"))
	}
}

func ffmpegTestFlagValue(args []string, flag string) string {
	for index, value := range args {
		if value == flag && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}

func ffmpegTestFlagValueAfter(args []string, start int, flag string) string {
	for index := start + 1; index < len(args)-1; index++ {
		if args[index] == flag {
			return args[index+1]
		}
	}
	return ""
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
