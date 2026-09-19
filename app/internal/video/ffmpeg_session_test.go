package video

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
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
	if slices.Contains(args, "-segment_format_options") || argsContainFaststart(args) {
		t.Fatalf("encoding args = %#v, want no faststart on cached segments", args)
	}
}

func TestFFmpegConcatArgsOmitFaststart(t *testing.T) {
	args := ffmpegConcatArgs("cache/ffmpeg-video/screen/segments.txt", "screen.mp4")
	if ffmpegTestFlagValue(args, "-f") != "concat" || !slices.Contains(args, "-c") {
		t.Fatalf("concat args = %#v, want concat demuxer with stream copy", args)
	}
	if argsContainFaststart(args) {
		t.Fatalf("concat args = %#v, want no faststart on internal finalize output", args)
	}
}

// argsContainFaststart reports whether an FFmpeg arg vector enables the
// faststart second pass, in either the flag form (-movflags +faststart) or
// the per-format option form (movflags=+faststart).
func argsContainFaststart(args []string) bool {
	return slices.Contains(args, "+faststart") || slices.ContainsFunc(args, func(arg string) bool {
		return strings.Contains(arg, "movflags=+faststart")
	})
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
	if slices.Contains(args, "-segment_format_options") || argsContainFaststart(args) {
		t.Fatalf("encoding args = %#v, want no faststart on cached segments", args)
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

// TestFFmpegPreviewImageFilterDefaultsToFourFPS locks the default preview
// cadence the camera builders rely on: both preview encoding paths share
// ffmpegPreviewImageFilter, so this default halves the preview JPEG overwrite
// rate for every platform input.
func TestFFmpegPreviewImageFilterDefaultsToFourFPS(t *testing.T) {
	if got := ffmpegPreviewImageFilter(ffmpegInputSpec{}); got != "fps=4,scale=360:-2" {
		t.Fatalf("ffmpegPreviewImageFilter() = %q, want default fps=4,scale=360:-2", got)
	}
	t.Setenv(EnvFFmpegSegmentSeconds, "12")
	session := &ffmpegDesktopSession{}
	args := session.encodingArgs("segment-%03d.mp4", ffmpegInputSpec{
		PreviewImagePath: filepath.Join("cache", "pip-camera-preview.jpg"),
	})
	if filter := ffmpegTestFlagValue(args, "-filter_complex"); filter != "[0:v]split=2[rf_record_src][rf_preview_src];[rf_record_src]pad=ceil(iw/2)*2:ceil(ih/2)*2[rf_record];[rf_preview_src]fps=4,scale=360:-2[rf_preview]" {
		t.Fatalf("filter_complex = %q, want split graph with default 4fps preview", filter)
	}
}

// requireBundledFFmpeg resolves a real ffmpeg binary for integration tests,
// preferring the repo-bundled tools/ffmpeg. It skips when no real binary is
// available so the pure unit suite stays green on machines without ffmpeg.
func requireBundledFFmpeg(t *testing.T) string {
	t.Helper()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	if bundled, err := filepath.Abs(filepath.Join("..", "..", "tools", name)); err == nil {
		if _, statErr := os.Stat(bundled); statErr == nil {
			t.Setenv(EnvFFmpegPath, bundled)
			return bundled
		}
	}
	resolved, err := ResolveFFmpegPath()
	if err != nil {
		t.Skipf("real ffmpeg unavailable: %v", err)
	}
	return resolved
}

func stubFFmpegInputArgs(CaptureConfig) (ffmpegInputSpec, error) {
	return ffmpegInputSpec{Args: []string{"-f", "lavfi", "-i", "testsrc=size=320x240:rate=30"}}, nil
}

// createFFmpegTestSegment encodes a tiny real segment with the same encoder
// settings the desktop session uses, so concat -c copy works on it.
func createFFmpegTestSegment(t *testing.T, ffmpegPath string, dir string, name string, seconds int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", fmt.Sprintf("testsrc=size=320x240:rate=30:duration=%d", seconds),
		"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
		path,
	}
	if out, err := exec.Command(ffmpegPath, args...).CombinedOutput(); err != nil {
		t.Fatalf("create test segment %s: %v: %s", name, err, out)
	}
	return path
}

// createFFmpegTestWAV writes a minimal float32 WAV the mux paths accept,
// mirroring the layout audio.WAVSink produces.
func createFFmpegTestWAV(t *testing.T, dir string, name string, sampleRate int, channels int, samples []float32) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := make([]byte, 44+len(samples)*4)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], uint32(36+len(samples)*4))
	copy(data[8:12], "WAVE")
	copy(data[12:16], "fmt ")
	binary.LittleEndian.PutUint32(data[16:20], 16)
	binary.LittleEndian.PutUint16(data[20:22], 3)
	binary.LittleEndian.PutUint16(data[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(data[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(data[28:32], uint32(sampleRate*channels*4))
	binary.LittleEndian.PutUint16(data[32:34], uint16(channels*4))
	binary.LittleEndian.PutUint16(data[34:36], 32)
	copy(data[36:40], "data")
	binary.LittleEndian.PutUint32(data[40:44], uint32(len(samples)*4))
	for index, sample := range samples {
		binary.LittleEndian.PutUint32(data[44+index*4:], math.Float32bits(sample))
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write test wav %s: %v", name, err)
	}
	return path
}

func newMergedFinalizeTestSession(t *testing.T, root string, segments []ffmpegSegment) *ffmpegDesktopSession {
	t.Helper()
	session, err := newFFmpegDesktopSession(CaptureConfig{
		OutputPath:      filepath.Join(root, "screen.mp4"),
		DiagnosticsPath: filepath.Join(root, "video-diagnostics.json"),
	}, stubFFmpegInputArgs)
	if err != nil {
		t.Fatalf("newFFmpegDesktopSession() error = %v", err)
	}
	session.segments = segments
	if err := os.MkdirAll(session.segmentDir(), 0o755); err != nil {
		t.Fatalf("MkdirAll(segmentDir) error = %v", err)
	}
	return session
}

// TestFFmpegMergedFinalizeMixesSidecarsInSinglePass runs the armed finalize
// for real: two segments and both sidecars go through one ffmpeg invocation,
// the output carries video plus mixed audio, and no fallback mux timing or
// fallback message is recorded.
func TestFFmpegMergedFinalizeMixesSidecarsInSinglePass(t *testing.T) {
	ffmpegPath := requireBundledFFmpeg(t)
	root := t.TempDir()
	segments := []ffmpegSegment{
		{Path: createFFmpegTestSegment(t, ffmpegPath, root, "segment-000-000.mp4", 1)},
		{Path: createFFmpegTestSegment(t, ffmpegPath, root, "segment-001-000.mp4", 1)},
	}
	systemWAV := createFFmpegTestWAV(t, root, "system.wav", 48000, 2, make([]float32, 48000))
	micWAV := createFFmpegTestWAV(t, root, "microphone.wav", 48000, 2, make([]float32, 24000))

	session := newMergedFinalizeTestSession(t, root, segments)
	session.SetFinalizeAudioInputs([]AudioMuxInput{
		{Path: systemWAV, Label: "system"},
		{Path: micWAV, Label: "microphone"},
	})
	if err := session.Stop(); err != nil {
		t.Fatalf("Stop() error = %v, want single-pass merged finalize", err)
	}

	messages := strings.Join(session.Diagnostics().Messages, " ")
	if !strings.Contains(messages, "Merged 2 FFmpeg segment(s) and 2 audio sidecar(s) into screen.mp4 in a single FFmpeg pass.") {
		t.Fatalf("messages = %q, want single-pass merged finalize message", messages)
	}
	if _, ok := session.Diagnostics().Timings["audio_mux"]; ok {
		t.Fatalf("timings = %#v, want no audio_mux entry when the merged pass succeeds", session.Diagnostics().Timings)
	}
	probe, err := recpackage.ProbeMP4(filepath.Join(root, "screen.mp4"))
	if err != nil {
		t.Fatalf("ProbeMP4() error = %v", err)
	}
	if !probe.HasVideoTrack || !probe.HasAudioTrack {
		t.Fatalf("probe = %#v, want video and mixed audio tracks in the merged output", probe)
	}
}

// TestFFmpegMergedFinalizeSingleSegmentUsesDirectVideoInput locks the
// single-segment armed shape: the segment feeds the pass directly instead of
// through the concat demuxer, and the sidecar maps as 1:a:0.
func TestFFmpegMergedFinalizeSingleSegmentUsesDirectVideoInput(t *testing.T) {
	ffmpegPath := requireBundledFFmpeg(t)
	root := t.TempDir()
	segment := createFFmpegTestSegment(t, ffmpegPath, root, "segment-000-000.mp4", 1)
	micWAV := createFFmpegTestWAV(t, root, "microphone.wav", 48000, 1, make([]float32, 48000))

	session := newMergedFinalizeTestSession(t, root, []ffmpegSegment{{Path: segment}})
	session.SetFinalizeAudioInputs([]AudioMuxInput{{Path: micWAV, Label: "microphone"}})

	var mergedArgs []string
	original := ffmpegProcessRun
	ffmpegProcessRun = func(cmd *exec.Cmd) (bool, error) {
		mergedArgs = append([]string(nil), cmd.Args...)
		return original(cmd)
	}
	t.Cleanup(func() { ffmpegProcessRun = original })

	if err := session.Stop(); err != nil {
		t.Fatalf("Stop() error = %v, want merged finalize from a direct segment input", err)
	}
	if ffmpegTestFlagValue(mergedArgs, "-f") == "concat" {
		t.Fatalf("merged args = %v, single segment must not go through the concat demuxer", mergedArgs)
	}
	if !slices.Contains(mergedArgs, segment) || !slices.Contains(mergedArgs, micWAV) {
		t.Fatalf("merged args = %v, want the segment and sidecar as direct inputs", mergedArgs)
	}
	if !slices.Contains(mergedArgs, "1:a:0") || slices.Contains(mergedArgs, "-filter_complex") {
		t.Fatalf("merged args = %v, want single sidecar mapped without amix", mergedArgs)
	}
	probe, err := recpackage.ProbeMP4(filepath.Join(root, "screen.mp4"))
	if err != nil {
		t.Fatalf("ProbeMP4() error = %v", err)
	}
	if !probe.HasVideoTrack || !probe.HasAudioTrack {
		t.Fatalf("probe = %#v, want video and audio tracks", probe)
	}
}

// TestFFmpegMergedFinalizeFallsBackToTwoStepAndMux fails the merged pass via
// the process runner seam and verifies the full fallback chain: the recorded
// merged command shape, the two-step finalize, the separate MuxAudioIntoMP4
// pass, the "fell back" diagnostic, and the extra audio_mux timing.
func TestFFmpegMergedFinalizeFallsBackToTwoStepAndMux(t *testing.T) {
	ffmpegPath := requireBundledFFmpeg(t)
	root := t.TempDir()
	segments := []ffmpegSegment{
		{Path: createFFmpegTestSegment(t, ffmpegPath, root, "segment-000-000.mp4", 1)},
		{Path: createFFmpegTestSegment(t, ffmpegPath, root, "segment-001-000.mp4", 1)},
	}
	micWAV := createFFmpegTestWAV(t, root, "microphone.wav", 48000, 1, make([]float32, 48000))
	outputPath := filepath.Join(root, "screen.mp4")

	session := newMergedFinalizeTestSession(t, root, segments)
	session.SetFinalizeAudioInputs([]AudioMuxInput{{Path: micWAV, Label: "microphone"}})

	var mergedArgs []string
	original := ffmpegProcessRun
	invocations := 0
	ffmpegProcessRun = func(cmd *exec.Cmd) (bool, error) {
		invocations++
		if invocations == 1 {
			mergedArgs = append([]string(nil), cmd.Args...)
			return true, errors.New("stubbed merged finalize failure")
		}
		return original(cmd)
	}
	t.Cleanup(func() { ffmpegProcessRun = original })

	if err := session.Stop(); err != nil {
		t.Fatalf("Stop() error = %v, want fallback finalize to succeed", err)
	}

	if ffmpegTestFlagValue(mergedArgs, "-f") != "concat" || !slices.Contains(mergedArgs, micWAV) || !slices.Contains(mergedArgs, "-shortest") {
		t.Fatalf("merged args = %v, want concat video input plus sidecar and -shortest", mergedArgs)
	}
	if last := mergedArgs[len(mergedArgs)-1]; last != finalizeStagingPath(outputPath) {
		t.Fatalf("merged output = %q, want staging path %q", last, finalizeStagingPath(outputPath))
	}

	messages := strings.Join(session.Diagnostics().Messages, " ")
	if !strings.Contains(messages, "Merged finalize failed: stubbed merged finalize failure") || !strings.Contains(messages, "fell back to two-step finalize + audio mux.") {
		t.Fatalf("messages = %q, want merged failure and fallback diagnostics", messages)
	}
	timings := session.Diagnostics().Timings
	if _, ok := timings["audio_mux"]; !ok {
		t.Fatalf("timings = %#v, want audio_mux entry recorded by the fallback mux", timings)
	}
	if _, ok := timings["finalize"]; !ok {
		t.Fatalf("timings = %#v, want finalize entry covering the whole stage", timings)
	}
	probe, err := recpackage.ProbeMP4(outputPath)
	if err != nil {
		t.Fatalf("ProbeMP4() error = %v", err)
	}
	if !probe.HasVideoTrack || !probe.HasAudioTrack {
		t.Fatalf("probe = %#v, want video and audio tracks after fallback mux", probe)
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
