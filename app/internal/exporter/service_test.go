package exporter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
)

func TestFFmpegArgsScreenOnlyCopyExport(t *testing.T) {
	plan := testPlan(t, false)

	args, err := FFmpegArgs(plan, filepath.Join(plan.PackageDir, "exports", "tmp.mp4"), Options{})
	if err != nil {
		t.Fatalf("FFmpegArgs() error = %v", err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"-i " + plan.ScreenInputPath, "-map 0:v:0", "-map 0:a?", "-c copy"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args = %q, want %q", joined, want)
		}
	}
	if strings.Contains(joined, "filter_complex") {
		t.Fatalf("screen-only args unexpectedly contain filter_complex: %q", joined)
	}
}

func TestFFmpegArgsPIPComposesMirrorOffsetAndSquareFeather(t *testing.T) {
	plan := testPlan(t, true)
	plan.WebcamStartOffsetMs = 120
	plan.PIPLayout.Shape = pip.ShapeSquare
	plan.PIPLayout.Mirror = true
	plan.PIPLayout.EdgeFeather = 0.18

	args, err := FFmpegArgs(plan, filepath.Join(plan.PackageDir, "exports", "tmp.mp4"), Options{VideoPreset: "medium", CRF: "23"})
	if err != nil {
		t.Fatalf("FFmpegArgs() error = %v", err)
	}
	filter := argAfter(args, "-filter_complex")
	for _, want := range []string{
		"setpts=PTS-STARTPTS+0.120/TB",
		"scale=320:180",
		"crop=320:180",
		"hflip",
		"overlay=32:48",
		"geq=r='r(X,Y)'",
	} {
		if !strings.Contains(filter, want) {
			t.Fatalf("filter = %q, want %q", filter, want)
		}
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"-c:v libx264", "-preset medium", "-crf 23", "-map [vout]", "-map 0:a?"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args = %q, want %q", joined, want)
		}
	}
}

func TestFFmpegArgsPIPCircleUsesAlphaMask(t *testing.T) {
	plan := testPlan(t, true)
	plan.PIPLayout.Shape = pip.ShapeCircle
	plan.PIPLayout.EdgeFeather = 0.2

	args, err := FFmpegArgs(plan, filepath.Join(plan.PackageDir, "exports", "tmp.mp4"), Options{})
	if err != nil {
		t.Fatalf("FFmpegArgs() error = %v", err)
	}
	filter := argAfter(args, "-filter_complex")
	for _, want := range []string{"sqrt(pow(X-W/2,2)+pow(Y-H/2,2))", "min(W,H)/2", "[pip]"} {
		if !strings.Contains(filter, want) {
			t.Fatalf("filter = %q, want %q", filter, want)
		}
	}
}

func TestExportRunsFFmpegIntoTempThenInstallsOutput(t *testing.T) {
	plan := testPlan(t, true)
	outputPath := plan.OutputPath
	var capturedArgs []string
	service := NewServiceWithRunner(CommandRunnerFunc(func(ctx context.Context, executable string, args []string) error {
		capturedArgs = append([]string(nil), args...)
		return os.WriteFile(args[len(args)-1], make([]byte, minReadableMP4Bytes+1), 0o644)
	}))

	result, err := service.Export(context.Background(), plan, Options{FFmpegPath: "ffmpeg-test", SkipOutputVerification: true})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if result.OutputPath != outputPath || result.Bytes <= minReadableMP4Bytes || !result.PIPVisible {
		t.Fatalf("result = %#v, want installed visible PIP export", result)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("export output was not installed: %v", err)
	}
	if len(capturedArgs) == 0 || capturedArgs[len(capturedArgs)-1] == outputPath {
		t.Fatalf("runner args = %#v, want temporary output path", capturedArgs)
	}
}

func TestExportVerifiesVideoTrackBeforeInstallingOutput(t *testing.T) {
	plan := testPlan(t, true)
	var calls [][]string
	service := NewServiceWithRunner(CommandRunnerFunc(func(ctx context.Context, executable string, args []string) error {
		calls = append(calls, append([]string(nil), args...))
		if len(calls) == 1 {
			return os.WriteFile(args[len(args)-1], make([]byte, minReadableMP4Bytes+1), 0o644)
		}
		joined := strings.Join(args, " ")
		for _, want := range []string{"-map 0:v:0", "-frames:v 1", "-f null -"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("verification args = %q, want %q", joined, want)
			}
		}
		if !strings.Contains(joined, ".recording-export-") {
			t.Fatalf("verification args = %#v, want video decode check", args)
		}
		return nil
	}))

	result, err := service.Export(context.Background(), plan, Options{FFmpegPath: "ffmpeg-test"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if !result.OutputVerified {
		t.Fatalf("OutputVerified = false, want true")
	}
	if len(calls) != 2 {
		t.Fatalf("runner calls = %d, want export + verification", len(calls))
	}
	if _, err := os.Stat(plan.OutputPath); err != nil {
		t.Fatalf("verified export output was not installed: %v", err)
	}
}

func TestExportRejectsUndecodableOutput(t *testing.T) {
	plan := testPlan(t, true)
	service := NewServiceWithRunner(CommandRunnerFunc(func(ctx context.Context, executable string, args []string) error {
		if args[len(args)-1] == "-" {
			return errors.New("missing video stream")
		}
		return os.WriteFile(args[len(args)-1], make([]byte, minReadableMP4Bytes+1), 0o644)
	}))

	if _, err := service.Export(context.Background(), plan, Options{FFmpegPath: "ffmpeg-test"}); err == nil || !strings.Contains(err.Error(), "verify export output video track") {
		t.Fatalf("Export() error = %v, want verification failure", err)
	}
	if _, err := os.Stat(plan.OutputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output installed after failed verification: %v", err)
	}
}

func TestExportRejectsOverwritingRawScreenVideo(t *testing.T) {
	plan := testPlan(t, false)
	plan.OutputPath = plan.ScreenInputPath

	if _, err := NewServiceWithRunner(CommandRunnerFunc(func(context.Context, string, []string) error {
		t.Fatal("runner should not be called")
		return nil
	})).Export(context.Background(), plan, Options{FFmpegPath: "ffmpeg-test"}); err == nil {
		t.Fatal("Export() error = nil, want raw screen overwrite rejection")
	}
}

func testPlan(t *testing.T, withPIP bool) exportplan.Plan {
	t.Helper()
	packageDir := t.TempDir()
	screenPath := filepath.Join(packageDir, "screen.mp4")
	if err := os.WriteFile(screenPath, make([]byte, 128), 0o644); err != nil {
		t.Fatalf("write screen: %v", err)
	}
	plan := exportplan.Plan{
		PackageDir:      packageDir,
		OutputPath:      filepath.Join(packageDir, "exports", "recording.mp4"),
		ScreenInputPath: screenPath,
		PIPLayout:       pip.Placement{Visible: false, Rect: pip.Rect{Visible: false}},
	}
	if withPIP {
		webcamPath := filepath.Join(packageDir, "webcam.mov")
		if err := os.WriteFile(webcamPath, make([]byte, 128), 0o644); err != nil {
			t.Fatalf("write webcam: %v", err)
		}
		plan.WebcamInputPath = webcamPath
		plan.PIPLayout = pip.Placement{
			Visible:     true,
			Rect:        pip.Rect{X: 32, Y: 48, Width: 319, Height: 179, Visible: true},
			Shape:       pip.ShapeCircle,
			Mirror:      false,
			EdgeFeather: 0.16,
		}
	}
	return plan
}

func argAfter(args []string, flag string) string {
	for index, arg := range args {
		if arg == flag && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}
