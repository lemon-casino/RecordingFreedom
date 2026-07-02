//go:build windows

package video

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestWindowsGraphicsCapturePlatformSessionConstructsForScreenSource(t *testing.T) {
	withFakeFFmpeg(t)
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "screen:--display1",
		SourceType:      devices.SourceScreen,
		SourceGeometry:  &SourceGeometry{Width: 1920, Height: 1080},
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}

func TestWindowsGraphicsCapturePlatformSessionConstructsForWindowSource(t *testing.T) {
	withFakeFFmpeg(t)
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "window:1f04aa",
		SourceType:      devices.SourceWindow,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}

func TestWindowsGraphicsCapturePlatformSessionConstructsForApplicationSource(t *testing.T) {
	withFakeFFmpeg(t)
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "application:420",
		SourceType:      devices.SourceApplication,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformSession() returned nil session")
	}
}

func TestWindowsGraphicsCapturePlatformSessionRequiresFFmpeg(t *testing.T) {
	t.Setenv(EnvFFmpegPath, filepath.Join(t.TempDir(), "missing-ffmpeg.exe"))
	_, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "screen:--display1",
		SourceType:      devices.SourceScreen,
		SourceGeometry:  &SourceGeometry{Width: 1920, Height: 1080},
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "FFmpeg") {
		t.Fatalf("NewPlatformSession() error = %v, want FFmpeg dependency error", err)
	}
}

func TestWindowsGraphicsCaptureStartFailsForProgramCapture(t *testing.T) {
	withFakeFFmpeg(t)
	session, err := NewPlatformSession(CaptureConfig{
		Backend:         "windows-graphics-capture",
		SourceID:        "application:420",
		SourceType:      devices.SourceApplication,
		OutputPath:      filepath.Join(t.TempDir(), "screen.mp4"),
		DiagnosticsPath: filepath.Join(t.TempDir(), "video-diagnostics.json"),
	})
	if err != nil {
		t.Fatalf("NewPlatformSession() error = %v", err)
	}
	err = session.Start(nil)
	if err == nil || !strings.Contains(err.Error(), "program capture is queued") {
		t.Fatalf("Start() error = %v, want program capture queued", err)
	}
}

func TestWindowsFFmpegInputArgsPreservesCursorCaptureSetting(t *testing.T) {
	tests := []struct {
		name          string
		captureCursor bool
		wantValue     string
	}{
		{name: "enabled", captureCursor: true, wantValue: "1"},
		{name: "disabled", captureCursor: false, wantValue: "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := windowsFFmpegInputArgs(windowsGraphicsCaptureTarget{Kind: windowsTargetRegion})(CaptureConfig{
				SourceGeometry: &SourceGeometry{X: 12, Y: 24, Width: 640, Height: 360},
				Profile: recordingprofile.Profile{
					Quality:       recordingprofile.QualityBalanced,
					FPS:           30,
					CaptureCursor: tt.captureCursor,
				},
			})
			if err != nil {
				t.Fatalf("windowsFFmpegInputArgs() error = %v", err)
			}
			if got := flagValue(args.Args, "-draw_mouse"); got != tt.wantValue {
				t.Fatalf("-draw_mouse = %q, want %q in args %v", got, tt.wantValue, args.Args)
			}
		})
	}
}

func TestWindowsFFmpegInputArgsUsesDDAGrabForDisplayBoundRegion(t *testing.T) {
	input, err := windowsFFmpegInputArgs(windowsGraphicsCaptureTarget{Kind: windowsTargetRegion})(CaptureConfig{
		SourceGeometry: &SourceGeometry{X: 12, Y: 24, Width: 640, Height: 360, DisplayIndex: 1, NativeID: `\\.\DISPLAY1`},
		Profile: recordingprofile.Profile{
			Quality:       recordingprofile.QualityBalanced,
			FPS:           30,
			CaptureCursor: true,
		},
	})
	if err != nil {
		t.Fatalf("windowsFFmpegInputArgs() error = %v", err)
	}
	if input.Engine != "windows-dda" {
		t.Fatalf("input engine = %q, want windows-dda; args = %v", input.Engine, input.Args)
	}
	if got := flagValue(input.Args, "-f"); got != "lavfi" {
		t.Fatalf("-f = %q, want lavfi in args %v", got, input.Args)
	}
	if !strings.Contains(input.Args[len(input.Args)-1], "ddagrab=") || !strings.Contains(input.Args[len(input.Args)-1], "draw_mouse=1") {
		t.Fatalf("input args = %v, want ddagrab cursor capture", input.Args)
	}
	if !strings.Contains(input.VideoFilter, "hwdownload") {
		t.Fatalf("video filter = %q, want hardware download for ddagrab", input.VideoFilter)
	}
}

func TestWindowsDDAGrabAllScreensBuildsStackedFilter(t *testing.T) {
	input := windowsDDAGrabAllScreensInputSpec(CaptureConfig{
		Profile: recordingprofile.Profile{
			Quality:       recordingprofile.QualityBalanced,
			FPS:           30,
			CaptureCursor: true,
		},
	}, []windowsMonitorBounds{
		{Index: 1, X: -1920, Y: 0, Width: 1920, Height: 1080},
		{Index: 2, X: 0, Y: 0, Width: 2560, Height: 1440},
	})
	graph := strings.Join(input.Args, " ")
	if input.Engine != "windows-dda" || !input.VideoPreFiltered {
		t.Fatalf("input = %#v, want prefiltered windows-dda", input)
	}
	if !strings.Contains(graph, "output_idx=0") || !strings.Contains(graph, "output_idx=1") {
		t.Fatalf("filter graph = %q, want both DDA outputs", graph)
	}
	if !strings.Contains(graph, "xstack=inputs=2:layout=0_0|1920_0") {
		t.Fatalf("filter graph = %q, want virtual desktop xstack layout", graph)
	}
	if !strings.Contains(graph, "draw_mouse=1") {
		t.Fatalf("filter graph = %q, want cursor capture in each DDA input", graph)
	}
}

func flagValue(args []string, flag string) string {
	for index, value := range args {
		if value == flag && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}

func withFakeFFmpeg(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffmpeg.exe")
	if err := os.WriteFile(path, []byte("fake ffmpeg"), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
	t.Setenv(EnvFFmpegPath, path)
}
