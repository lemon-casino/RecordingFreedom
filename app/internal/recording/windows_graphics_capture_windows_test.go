//go:build windows

package recording

import (
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func TestWindowsDesktopCaptureBackendIsDefaultOnWindows(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "windows", "native")
	runtimeBackend, ok := backend.(*NativeRuntimeBackend)
	if !ok {
		t.Fatalf("SelectBackend(native windows) = %T, want *NativeRuntimeBackend", backend)
	}
	if runtimeBackend.ID() != BackendFFmpegDesktopCapture {
		t.Fatalf("backend id = %q, want %q", runtimeBackend.ID(), BackendFFmpegDesktopCapture)
	}
}

func TestWindowsGraphicsCaptureBackendAliasIsRegistered(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "windows", "wgc")
	runtimeBackend, ok := backend.(*NativeRuntimeBackend)
	if !ok {
		t.Fatalf("SelectBackend(wgc windows) = %T, want *NativeRuntimeBackend", backend)
	}
	if runtimeBackend.ID() != BackendWindowsGraphicsCapture {
		t.Fatalf("backend id = %q, want %q", runtimeBackend.ID(), BackendWindowsGraphicsCapture)
	}
}

func TestWindowsDesktopCaptureRuntimeFailsWithoutFakeMediaWhenFFmpegMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("RECORDINGFREEDOM_FFMPEG_PATH", filepath.Join(root, "missing-ffmpeg.exe"))
	service := NewServiceWithBackend(appdata.NewService(root), SelectBackend(recpackage.NewService(), "windows", "native"))

	_, err := service.StartRecording(StartRequest{
		SourceID:   "screen:--display1",
		SourceType: SourceScreen,
		SourceGeometry: &SourceGeometry{
			Width:  1920,
			Height: 1080,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "FFmpeg") {
		t.Fatalf("StartRecording() error = %v, want FFmpeg dependency error", err)
	}
	if service.State() != StateFailed {
		t.Fatalf("State() = %q, want %q", service.State(), StateFailed)
	}

	matches, err := filepath.Glob(filepath.Join(root, "data", "video", "*.rfrec"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("packages = %#v, want one failed Windows capture package", matches)
	}
	manifest, err := recpackage.NewService().ReadManifest(filepath.Join(matches[0], recpackage.ManifestFile))
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Status != recpackage.StatusFailed {
		t.Fatalf("manifest status = %q, want %q", manifest.Status, recpackage.StatusFailed)
	}
	if exists(filepath.Join(matches[0], recpackage.ScreenVideoFile)) {
		t.Fatal("Windows capture dependency failure created fake screen.mp4")
	}
	if !exists(filepath.Join(matches[0], recpackage.VideoDiagnosticsFile)) {
		t.Fatal("failed Windows capture package did not keep video-diagnostics.json")
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// newWindowsMuxTestRuntime builds a runtime with a real screen-mode package
// and one enabled system audio sidecar path, backed by fake media sessions.
func newWindowsMuxTestRuntime(t *testing.T) *NativeBackendRuntime {
	t.Helper()
	runtime, err := NewNativeBackendRuntime(recpackage.NewService(), BackendFFmpegDesktopCapture, BackendStartRequest{
		VideoDir:  t.TempDir(),
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:primary",
			SourceType: SourceScreen,
			Audio: AudioRequest{
				System:         true,
				SystemDeviceID: "system-audio:default",
			},
		},
	}, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(video.CaptureConfig) (NativeVideoSession, error) {
			return &fakeNativeVideoSession{}, nil
		},
		AudioSessionFactory: func(audio.CaptureConfig, audio.NoiseSuppressor) (NativeAudioSession, error) {
			return &fakeNativeAudioSession{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewNativeBackendRuntime() error = %v", err)
	}
	return runtime
}

// requireBundledFFmpegRecording resolves the repo-bundled ffmpeg for the
// recording package integration tests, skipping without one.
func requireBundledFFmpegRecording(t *testing.T) string {
	t.Helper()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	if bundled, err := filepath.Abs(filepath.Join("..", "..", "tools", name)); err == nil {
		if _, statErr := os.Stat(bundled); statErr == nil {
			t.Setenv(video.EnvFFmpegPath, bundled)
			return bundled
		}
	}
	resolved, err := video.ResolveFFmpegPath()
	if err != nil {
		t.Skipf("real ffmpeg unavailable: %v", err)
	}
	return resolved
}

// TestWindowsMuxAudioSidecarsIntoScreenSkipsMuxWhenFinalizeArmed proves the
// armed plan skips the second mux: screen.mp4 does not exist, so a mux attempt
// would fail, while the skip branch only patches the manifest with the booleans
// recorded at arming time.
func TestWindowsMuxAudioSidecarsIntoScreenSkipsMuxWhenFinalizeArmed(t *testing.T) {
	runtime := newWindowsMuxTestRuntime(t)
	runtime.videoFinalizeAudio = &videoFinalizeAudioPlan{
		Inputs:    []video.AudioMuxInput{{Path: runtime.Plan.SystemAudioPath, Label: "system"}},
		MuxSystem: true,
	}

	if err := windowsMuxAudioSidecarsIntoScreen(runtime); err != nil {
		t.Fatalf("windowsMuxAudioSidecarsIntoScreen() error = %v, want armed plan to skip the second mux", err)
	}

	manifest, err := recpackage.NewService().ReadManifest(runtime.Plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Media.SystemAudioStorage != recpackage.AudioStorageMuxed || manifest.Media.SystemAudioPath != recpackage.ScreenVideoFile {
		t.Fatalf("manifest media = %#v, want system audio muxed into screen.mp4", manifest.Media)
	}
	if manifest.Media.MicrophoneAudioStorage == recpackage.AudioStorageMuxed {
		t.Fatalf("manifest media = %#v, want microphone left as sidecar when it was not armed", manifest.Media)
	}
	if runtime.Plan.SystemAudioPath != runtime.Plan.ScreenVideoPath {
		t.Fatalf("plan system audio path = %q, want screen media path %q", runtime.Plan.SystemAudioPath, runtime.Plan.ScreenVideoPath)
	}
}

// TestWindowsMuxAudioSidecarsIntoScreenNoopsWithoutReadableSidecars keeps the
// un-armed entry condition: no readable sidecar means no mux and no manifest
// change, matching the pre-merge behavior.
func TestWindowsMuxAudioSidecarsIntoScreenNoopsWithoutReadableSidecars(t *testing.T) {
	runtime := newWindowsMuxTestRuntime(t)

	if err := windowsMuxAudioSidecarsIntoScreen(runtime); err != nil {
		t.Fatalf("windowsMuxAudioSidecarsIntoScreen() error = %v, want no-op without sidecars", err)
	}
	manifest, err := recpackage.NewService().ReadManifest(runtime.Plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Media.SystemAudioStorage != recpackage.AudioStorageSidecar {
		t.Fatalf("manifest storage = %q, want unchanged sidecar storage", manifest.Media.SystemAudioStorage)
	}
	if runtime.videoFinalizeAudio != nil {
		t.Fatalf("videoFinalizeAudio plan = %#v, want nil when not armed", runtime.videoFinalizeAudio)
	}
}

// writeFloat32TestWAV writes a minimal float32 WAV a real mux can decode.
func writeFloat32TestWAV(t *testing.T, path string, sampleRate int, channels int, samples []float32) {
	t.Helper()
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
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

// TestWindowsMuxAudioSidecarsIntoScreenMuxesSidecarWhenNotArmed runs the
// un-armed two-step path for real: the sidecar is muxed into screen.mp4 and
// the manifest records the muxed storage.
func TestWindowsMuxAudioSidecarsIntoScreenMuxesSidecarWhenNotArmed(t *testing.T) {
	ffmpegPath := requireBundledFFmpegRecording(t)
	runtime := newWindowsMuxTestRuntime(t)
	if err := exec.Command(ffmpegPath, "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=30:duration=1",
		"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
		runtime.Plan.ScreenVideoPath).Run(); err != nil {
		t.Fatalf("encode test screen.mp4: %v", err)
	}
	writeFloat32TestWAV(t, runtime.Plan.SystemAudioPath, 48000, 2, make([]float32, 48000))

	if err := windowsMuxAudioSidecarsIntoScreen(runtime); err != nil {
		t.Fatalf("windowsMuxAudioSidecarsIntoScreen() error = %v, want two-step mux", err)
	}

	manifest, err := recpackage.NewService().ReadManifest(runtime.Plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Media.SystemAudioStorage != recpackage.AudioStorageMuxed {
		t.Fatalf("manifest storage = %q, want muxed after the two-step mux", manifest.Media.SystemAudioStorage)
	}
	probe, err := recpackage.ProbeMP4(runtime.Plan.ScreenVideoPath)
	if err != nil {
		t.Fatalf("ProbeMP4() error = %v", err)
	}
	if !probe.HasVideoTrack || !probe.HasAudioTrack {
		t.Fatalf("probe = %#v, want muxed audio track in screen.mp4", probe)
	}
}
