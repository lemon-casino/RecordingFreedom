package video

import (
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
	if !slices.Contains(args, "-shortest") || !slices.Contains(args, "+faststart") {
		t.Fatalf("args = %#v, want bounded mux output flags", args)
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
