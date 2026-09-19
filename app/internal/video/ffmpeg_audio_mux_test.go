package video

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFFmpegAudioMuxArgsSingleInputMapsAudio(t *testing.T) {
	args := ffmpegAudioMuxArgs("screen.mp4", []AudioMuxInput{{Path: "microphone.wav"}}, "screen.muxed.mp4")
	if !slices.Contains(args, "-map") || !slices.Contains(args, "1:a:0") {
		t.Fatalf("args = %#v, want first audio input mapped", args)
	}
	if slices.Contains(args, "-filter_complex") {
		t.Fatalf("args = %#v, single input should not need amix filter", args)
	}
	if !slices.Contains(args, "-shortest") {
		t.Fatalf("args = %#v, want bounded mux output flags", args)
	}
	if slices.Contains(args, "+faststart") || slices.Contains(args, "movflags=+faststart") {
		t.Fatalf("args = %#v, want no faststart on internal screen mux output", args)
	}
}

func TestFFmpegAudioMuxArgsMultipleInputsMixesAudio(t *testing.T) {
	args := ffmpegAudioMuxArgs("screen.mp4", []AudioMuxInput{{Path: "system.wav"}, {Path: "microphone.wav"}}, "screen.muxed.mp4")
	filterIndex := slices.Index(args, "-filter_complex")
	if filterIndex < 0 || filterIndex+1 >= len(args) {
		t.Fatalf("args = %#v, want filter_complex", args)
	}
	filter := args[filterIndex+1]
	for _, want := range []string{"[1:a]", "[2:a]", "amix=inputs=2", "[mixed_audio]"} {
		if !strings.Contains(filter, want) {
			t.Fatalf("filter = %q, want %q", filter, want)
		}
	}
	if !slices.Contains(args, "[mixed_audio]") {
		t.Fatalf("args = %#v, want mixed audio output mapped", args)
	}
	if slices.Contains(args, "+faststart") || slices.Contains(args, "movflags=+faststart") {
		t.Fatalf("args = %#v, want no faststart on internal screen mux output", args)
	}
}

func TestFFmpegAudioOnlyMuxArgsSingleInputMapsAudio(t *testing.T) {
	args := ffmpegAudioOnlyMuxArgs([]AudioMuxInput{{Path: "audio.wav"}}, "audio.m4a")
	if !slices.Contains(args, "-map") || !slices.Contains(args, "0:a:0") {
		t.Fatalf("args = %#v, want first input audio mapped", args)
	}
	if slices.Contains(args, "-filter_complex") {
		t.Fatalf("args = %#v, single input should not need amix filter", args)
	}
	if !slices.Contains(args, "-vn") || !slices.Contains(args, "+faststart") {
		t.Fatalf("args = %#v, want audio-only faststart output flags", args)
	}
}

func TestFFmpegAudioOnlyMuxArgsMultipleInputsMixesAudio(t *testing.T) {
	args := ffmpegAudioOnlyMuxArgs([]AudioMuxInput{{Path: "system.wav"}, {Path: "microphone.wav"}}, "audio.m4a")
	filterIndex := slices.Index(args, "-filter_complex")
	if filterIndex < 0 || filterIndex+1 >= len(args) {
		t.Fatalf("args = %#v, want filter_complex", args)
	}
	filter := args[filterIndex+1]
	for _, want := range []string{"[0:a]", "[1:a]", "amix=inputs=2", "[mixed_audio]"} {
		if !strings.Contains(filter, want) {
			t.Fatalf("filter = %q, want %q", filter, want)
		}
	}
}

// TestFFmpegMergedFinalizeArgsSingleSegmentMapsSidecar locks the single-pass
// finalize shape for one segment and one sidecar: the segment is fed directly
// and the sidecar maps like ffmpegAudioMuxArgs would map it.
func TestFFmpegMergedFinalizeArgsSingleSegmentMapsSidecar(t *testing.T) {
	args := ffmpegMergedFinalizeArgs(
		[]string{"-i", "segment-000-000.mp4"},
		[]AudioMuxInput{{Path: "microphone.wav", Label: "microphone"}},
		".finalize-screen.mp4",
	)
	if got := ffmpegTestFlagValue(args, "-f"); got != "" {
		t.Fatalf("-f = %q, want no concat demuxer for a single segment in args %v", got, args)
	}
	if args[4] != "-i" || args[5] != "segment-000-000.mp4" || args[6] != "-i" || args[7] != "microphone.wav" {
		t.Fatalf("args = %v, want video segment input followed by the audio sidecar input", args)
	}
	if !slices.Contains(args, "-map") || !slices.Contains(args, "1:a:0") {
		t.Fatalf("args = %#v, want first audio input mapped", args)
	}
	if slices.Contains(args, "-filter_complex") {
		t.Fatalf("args = %#v, single sidecar should not need amix filter", args)
	}
	assertMergedFinalizeTailArgs(t, args)
}

// TestFFmpegMergedFinalizeArgsMultipleSegmentsReuseMixFilter locks the
// multi-segment merged shape and the verbatim reuse of audioMixFilter: the
// merged filter graph must equal the two-step mux filter so both paths stay
// byte-identical.
func TestFFmpegMergedFinalizeArgsMultipleSegmentsReuseMixFilter(t *testing.T) {
	args := ffmpegMergedFinalizeArgs(
		[]string{"-f", "concat", "-safe", "0", "-i", filepath.Join("cache", "ffmpeg-video", "screen", "segments.txt")},
		[]AudioMuxInput{{Path: "system.wav", Label: "system"}, {Path: "microphone.wav", Label: "microphone"}},
		".finalize-screen.mp4",
	)
	if ffmpegTestFlagValue(args, "-f") != "concat" || !slices.Contains(args, "-safe") {
		t.Fatalf("args = %#v, want concat demuxer input for multiple segments", args)
	}
	if got := ffmpegTestFlagValue(args, "-filter_complex"); got != audioMixFilter(2) {
		t.Fatalf("filter_complex = %q, want the shared audioMixFilter(2) graph verbatim", got)
	}
	if !slices.Contains(args, "[mixed_audio]") {
		t.Fatalf("args = %#v, want mixed audio output mapped", args)
	}
	assertMergedFinalizeTailArgs(t, args)
}

func assertMergedFinalizeTailArgs(t *testing.T, args []string) {
	t.Helper()
	if !slices.Contains(args, "-map") || !slices.Contains(args, "0:v:0") || !slices.Contains(args, "-c:v") || !slices.Contains(args, "copy") {
		t.Fatalf("args = %#v, want video stream-copied from the segment input", args)
	}
	for flag, want := range map[string]string{
		"-c:a": "aac",
		"-b:a": "192k",
		"-ar":  "48000",
		"-ac":  "2",
	} {
		if got := ffmpegTestFlagValue(args, flag); got != want {
			t.Fatalf("%s = %q, want %q in args %v", flag, got, want, args)
		}
	}
	if !slices.Contains(args, "-shortest") {
		t.Fatalf("args = %#v, want bounded mux output flag", args)
	}
	if args[len(args)-1] != ".finalize-screen.mp4" {
		t.Fatalf("args = %v, want staging output as the last argument", args)
	}
	if argsContainFaststart(args) {
		t.Fatalf("args = %#v, want no faststart on internal merged output", args)
	}
}
