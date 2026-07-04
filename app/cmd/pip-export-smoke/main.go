package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exporter"
	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

const (
	defaultSyntheticWidth    = 1280
	defaultSyntheticHeight   = 720
	defaultSyntheticDuration = 2 * time.Second
	defaultCommandTimeout    = 2 * time.Minute
	sampleColorTolerance     = 95
)

var syntheticPIPColor = sampleColor{R: 253, G: 0, B: 0}

type report struct {
	OK                  bool            `json:"ok"`
	Synthetic           bool            `json:"synthetic"`
	DataDir             string          `json:"dataDir,omitempty"`
	VideoDir            string          `json:"videoDir"`
	PackageDir          string          `json:"packageDir"`
	ManifestPath        string          `json:"manifestPath,omitempty"`
	OutputPath          string          `json:"outputPath"`
	OutputBytes         int64           `json:"outputBytes"`
	OutputVerified      bool            `json:"outputVerified"`
	PIPPixelVerified    bool            `json:"pipPixelVerified,omitempty"`
	PIPPixel            sampledPixel    `json:"pipPixel,omitempty"`
	BackgroundPixel     sampledPixel    `json:"backgroundPixel,omitempty"`
	ScreenInputPath     string          `json:"screenInputPath"`
	WebcamInputPath     string          `json:"webcamInputPath,omitempty"`
	WebcamStartOffsetMs int             `json:"webcamStartOffsetMs,omitempty"`
	PIPVisible          bool            `json:"pipVisible"`
	PIPLayout           pip.Placement   `json:"pipLayout"`
	Warnings            []string        `json:"warnings,omitempty"`
	DurationMs          int64           `json:"durationMs"`
	FFmpegPath          string          `json:"ffmpegPath"`
	Plan                exportplan.Plan `json:"plan"`
}

type sampledPixel struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type sampleColor struct {
	R uint8
	G uint8
	B uint8
}

type options struct {
	videoDir     string
	packageDir   string
	outputPath   string
	canvasWidth  int
	canvasHeight int
	requireSync  bool
	allowMock    bool
	ffmpegPath   string
	timeout      time.Duration
	synthetic    bool
	dataDir      string
	keep         bool
	width        int
	height       int
	duration     time.Duration
}

func main() {
	var opts options
	flag.StringVar(&opts.videoDir, "video-dir", "", "recording video root that contains the .rfrec package")
	flag.StringVar(&opts.packageDir, "package-dir", "", "recording package directory ending in .rfrec")
	flag.StringVar(&opts.outputPath, "output", exportplan.DefaultOutputPath, "package-relative export path")
	flag.IntVar(&opts.canvasWidth, "canvas-width", 0, "screen canvas width for PIP placement")
	flag.IntVar(&opts.canvasHeight, "canvas-height", 0, "screen canvas height for PIP placement")
	flag.BoolVar(&opts.requireSync, "require-sync", true, "require real sync diagnostics before export")
	flag.BoolVar(&opts.allowMock, "allow-mock", false, "allow mock packages for command plumbing tests")
	flag.StringVar(&opts.ffmpegPath, "ffmpeg", "", "optional FFmpeg executable path")
	flag.DurationVar(&opts.timeout, "timeout", 10*time.Minute, "FFmpeg export timeout")
	flag.BoolVar(&opts.synthetic, "synthetic", false, "generate a black screen + red camera .rfrec package and verify PIP pixels in the exported MP4")
	flag.StringVar(&opts.dataDir, "data-dir", "", "synthetic data root; defaults to a temporary directory when -synthetic is used")
	flag.BoolVar(&opts.keep, "keep", false, "keep the generated synthetic data root for manual inspection")
	flag.IntVar(&opts.width, "width", defaultSyntheticWidth, "synthetic screen width when -synthetic is used")
	flag.IntVar(&opts.height, "height", defaultSyntheticHeight, "synthetic screen height when -synthetic is used")
	flag.DurationVar(&opts.duration, "duration", defaultSyntheticDuration, "synthetic media duration when -synthetic is used")
	flag.Parse()

	result, err := run(opts)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err != nil {
		_ = encoder.Encode(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	if err := encoder.Encode(result); err != nil {
		fail(err)
	}
}

func run(opts options) (report, error) {
	if opts.canvasWidth < 0 || opts.canvasHeight < 0 {
		return report{}, errors.New("canvas dimensions cannot be negative")
	}
	if opts.timeout <= 0 {
		opts.timeout = defaultCommandTimeout
	}
	synthetic := syntheticPackage{}
	cleanup := func() {}
	if opts.synthetic {
		var err error
		synthetic, cleanup, err = prepareSyntheticPackage(opts)
		if err != nil {
			return report{}, err
		}
		defer cleanup()
		opts.videoDir = synthetic.videoDir
		opts.packageDir = synthetic.packageDir
		opts.canvasWidth = synthetic.width
		opts.canvasHeight = synthetic.height
		opts.ffmpegPath = synthetic.ffmpegPath
	}
	startedAt := time.Now()
	plan, err := exportplan.NewService(nil).Plan(exportplan.Request{
		VideoDir:    opts.videoDir,
		PackageDir:  opts.packageDir,
		OutputPath:  opts.outputPath,
		Canvas:      pip.Size{Width: opts.canvasWidth, Height: opts.canvasHeight},
		RequireSync: opts.requireSync,
		AllowMock:   opts.allowMock,
	})
	if err != nil {
		return report{}, fmt.Errorf("create export plan: %w", err)
	}
	exportResult, err := exporter.NewService().Export(nil, plan, exporter.Options{
		FFmpegPath: opts.ffmpegPath,
		Timeout:    opts.timeout,
	})
	if err != nil {
		return report{}, fmt.Errorf("export PIP recording: %w", err)
	}
	var pipPixel sampledPixel
	var background sampledPixel
	pipPixelVerified := false
	if opts.synthetic {
		pipPixel, background, err = verifySyntheticPIPPixels(opts.ffmpegPath, exportResult.OutputPath, opts.width, opts.height, plan.PIPLayout, opts.duration, opts.timeout)
		if err != nil {
			return report{}, err
		}
		pipPixelVerified = true
	}
	return report{
		OK:                  true,
		Synthetic:           opts.synthetic,
		DataDir:             synthetic.dataDir,
		VideoDir:            opts.videoDir,
		PackageDir:          opts.packageDir,
		ManifestPath:        synthetic.manifestPath,
		OutputPath:          exportResult.OutputPath,
		OutputBytes:         exportResult.Bytes,
		OutputVerified:      exportResult.OutputVerified,
		PIPPixelVerified:    pipPixelVerified,
		PIPPixel:            pipPixel,
		BackgroundPixel:     background,
		ScreenInputPath:     plan.ScreenInputPath,
		WebcamInputPath:     plan.WebcamInputPath,
		WebcamStartOffsetMs: plan.WebcamStartOffsetMs,
		PIPVisible:          plan.PIPLayout.Visible,
		PIPLayout:           plan.PIPLayout,
		Warnings:            plan.Warnings,
		DurationMs:          time.Since(startedAt).Milliseconds(),
		FFmpegPath:          exportResult.FFmpegPath,
		Plan:                plan,
	}, nil
}

type syntheticPackage struct {
	dataDir      string
	videoDir     string
	packageDir   string
	manifestPath string
	ffmpegPath   string
	width        int
	height       int
}

func prepareSyntheticPackage(opts options) (syntheticPackage, func(), error) {
	if opts.width <= 0 || opts.height <= 0 {
		return syntheticPackage{}, func() {}, fmt.Errorf("invalid synthetic canvas size %dx%d", opts.width, opts.height)
	}
	if opts.duration <= 0 {
		return syntheticPackage{}, func() {}, errors.New("synthetic duration must be positive")
	}
	ffmpegPath := strings.TrimSpace(opts.ffmpegPath)
	if ffmpegPath == "" {
		resolved, err := video.ResolveFFmpegPath()
		if err != nil {
			return syntheticPackage{}, func() {}, err
		}
		ffmpegPath = resolved
	}
	dataDir, cleanup, err := prepareDataDir(opts)
	if err != nil {
		return syntheticPackage{}, func() {}, err
	}
	videoDir := filepath.Join(dataDir, "data", "video")
	packageDir := filepath.Join(videoDir, "recording-pip-export-smoke-"+time.Now().Format("20060102-150405-000")+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		cleanup()
		return syntheticPackage{}, func() {}, err
	}
	if err := writeSyntheticVideo(ffmpegPath, filepath.Join(packageDir, recpackage.ScreenVideoFile), opts.width, opts.height, "black", opts.duration, opts.timeout); err != nil {
		cleanup()
		return syntheticPackage{}, func() {}, fmt.Errorf("write synthetic screen video: %w", err)
	}
	if err := writeSyntheticVideo(ffmpegPath, filepath.Join(packageDir, recpackage.WebcamVideoFile), opts.width/2, opts.height/2, "red", opts.duration, opts.timeout); err != nil {
		cleanup()
		return syntheticPackage{}, func() {}, fmt.Errorf("write synthetic webcam video: %w", err)
	}
	manifestPath, err := writeSyntheticManifest(packageDir, opts.width, opts.height, opts.duration)
	if err != nil {
		cleanup()
		return syntheticPackage{}, func() {}, err
	}
	return syntheticPackage{
		dataDir:      dataDir,
		videoDir:     videoDir,
		packageDir:   packageDir,
		manifestPath: manifestPath,
		ffmpegPath:   ffmpegPath,
		width:        opts.width,
		height:       opts.height,
	}, cleanup, nil
}

func prepareDataDir(opts options) (string, func(), error) {
	dataDir := strings.TrimSpace(opts.dataDir)
	if dataDir == "" {
		tempRoot, err := os.MkdirTemp("", "recordingfreedom-pip-export-smoke-*")
		if err != nil {
			return "", func() {}, err
		}
		cleanup := func() {
			if !opts.keep {
				_ = os.RemoveAll(tempRoot)
			}
		}
		return tempRoot, cleanup, nil
	}
	absolute, err := filepath.Abs(dataDir)
	if err != nil {
		return "", func() {}, err
	}
	return absolute, func() {}, nil
}

func writeSyntheticVideo(ffmpegPath string, outputPath string, width int, height int, color string, duration time.Duration, timeout time.Duration) error {
	if width <= 0 {
		width = 64
	}
	if height <= 0 {
		height = 64
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	args := []string{
		"-hide_banner", "-v", "error", "-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("color=c=%s:s=%dx%d:r=30:d=%.3f", color, width, height, duration.Seconds()),
		"-an",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		outputPath,
	}
	return runCommand(timeout, ffmpegPath, args, nil)
}

func writeSyntheticManifest(packageDir string, width int, height int, duration time.Duration) (string, error) {
	startedAt := time.Now()
	completedAt := startedAt.Add(duration)
	durationMs := duration.Milliseconds()
	pipConfig := pip.Config{
		Preset:      pip.PresetBottomRight,
		Shape:       pip.ShapeCircle,
		Mirror:      false,
		Position:    pip.Position{X: 1, Y: 1},
		Scale:       pip.MaximumScale,
		EdgeFeather: 0,
	}
	manifest := recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		CreatedAt:     startedAt.UTC(),
		CompletedAt:   &completedAt,
		Status:        recpackage.StatusReady,
		RecordingMode: recpackage.RecordingModeScreen,
		Media: recpackage.ManifestMedia{
			ScreenVideoPath: recpackage.ScreenVideoFile,
			WebcamVideoPath: recpackage.WebcamVideoFile,
		},
		Source: recpackage.ManifestSource{
			Type: "screen",
			ID:   "screen:pip-export-smoke",
			Name: "PIP Export Smoke",
			Geometry: &recpackage.ManifestSourceGeometry{
				X:            0,
				Y:            0,
				Width:        width,
				Height:       height,
				DisplayIndex: 1,
				NativeID:     "pip-export-smoke-display-1",
			},
		},
		Recording: recordingprofile.Profile{
			Quality:       recordingprofile.QualityBalanced,
			FPS:           30,
			CaptureCursor: true,
		},
		Audio: recpackage.ManifestAudio{
			MicrophoneNoiseSuppression: recpackage.NoiseSuppressionOff,
			SystemAudioIsNeverDenoised: true,
		},
		Camera: recpackage.ManifestCamera{
			Enabled:   true,
			DeviceID:  "camera:pip-export-smoke",
			PIPPreset: string(pip.PresetBottomRight),
			PIP:       pipConfig,
		},
		Diagnostics: recpackage.ManifestDiagnostics{
			Sync: &recpackage.ManifestSyncDiagnostics{
				TimelineBase:         recpackage.TimelineBaseMedia,
				VideoDiagnosticsPath: recpackage.VideoDiagnosticsFile,
				Screen: recpackage.ManifestTrackDiagnostics{
					Enabled:     true,
					Path:        recpackage.ScreenVideoFile,
					Clock:       recpackage.TimelineBaseMedia,
					EndOffsetMs: durationMs,
					DurationMs:  durationMs,
					FrameRate:   30,
				},
				Webcam: recpackage.ManifestTrackDiagnostics{
					Enabled:     true,
					Path:        recpackage.WebcamVideoFile,
					Clock:       recpackage.TimelineBaseMedia,
					EndOffsetMs: durationMs,
					DurationMs:  durationMs,
					FrameRate:   30,
				},
			},
		},
	}
	manifestPath := filepath.Join(packageDir, recpackage.ManifestFile)
	if err := recpackage.NewService().WriteManifest(manifestPath, manifest); err != nil {
		return "", err
	}
	return manifestPath, nil
}

func verifySyntheticPIPPixels(ffmpegPath string, outputPath string, width int, height int, layout pip.Placement, duration time.Duration, timeout time.Duration) (sampledPixel, sampledPixel, error) {
	sampleAt := duration / 2
	if sampleAt <= 0 {
		sampleAt = 500 * time.Millisecond
	}
	frame, err := extractRGBFrame(ffmpegPath, outputPath, width, height, sampleAt, timeout)
	if err != nil {
		return sampledPixel{}, sampledPixel{}, err
	}
	return verifyPIPPixelsFromFrame(frame, width, height, layout, syntheticPIPColor)
}

func verifyPIPPixelsFromFrame(frame []byte, width int, height int, layout pip.Placement, expected sampleColor) (sampledPixel, sampledPixel, error) {
	if width <= 0 || height <= 0 {
		return sampledPixel{}, sampledPixel{}, fmt.Errorf("invalid frame size %dx%d", width, height)
	}
	expectedBytes := width * height * 3
	if len(frame) != expectedBytes {
		return sampledPixel{}, sampledPixel{}, fmt.Errorf("frame has %d bytes, want %d", len(frame), expectedBytes)
	}
	if !layout.Visible || !layout.Rect.Visible || layout.Rect.Width <= 0 || layout.Rect.Height <= 0 {
		return sampledPixel{}, sampledPixel{}, errors.New("PIP layout is not visible")
	}
	pipX := clampInt(layout.Rect.X+layout.Rect.Width/2, 0, width-1)
	pipY := clampInt(layout.Rect.Y+layout.Rect.Height/2, 0, height-1)
	backgroundX := clampInt(width/8, 0, width-1)
	backgroundY := clampInt(height/8, 0, height-1)
	pipPixel := pixelAt(frame, width, pipX, pipY)
	background := pixelAt(frame, width, backgroundX, backgroundY)
	if !pixelNearColor(pipPixel, expected) {
		return pipPixel, background, fmt.Errorf("PIP pixel mismatch at %d,%d: got rgb(%d,%d,%d), want near rgb(%d,%d,%d)", pipPixel.X, pipPixel.Y, pipPixel.R, pipPixel.G, pipPixel.B, expected.R, expected.G, expected.B)
	}
	if background.R > 80 || background.G > 80 || background.B > 80 {
		return pipPixel, background, fmt.Errorf("background pixel is not dark after PIP export: %#v", background)
	}
	return pipPixel, background, nil
}

func extractRGBFrame(ffmpegPath string, outputPath string, width int, height int, at time.Duration, timeout time.Duration) ([]byte, error) {
	var stdout bytes.Buffer
	args := []string{
		"-hide_banner", "-v", "error",
		"-ss", fmt.Sprintf("%.3f", at.Seconds()),
		"-i", outputPath,
		"-frames:v", "1",
		"-vf", fmt.Sprintf("scale=%d:%d,format=rgb24", width, height),
		"-f", "rawvideo",
		"pipe:1",
	}
	if err := runCommand(timeout, ffmpegPath, args, &stdout); err != nil {
		return nil, fmt.Errorf("extract PIP export frame: %w", err)
	}
	expectedBytes := width * height * 3
	if stdout.Len() != expectedBytes {
		return nil, fmt.Errorf("extracted frame has %d bytes, want %d", stdout.Len(), expectedBytes)
	}
	return stdout.Bytes(), nil
}

func pixelAt(frame []byte, width int, x int, y int) sampledPixel {
	offset := (y*width + x) * 3
	if offset < 0 || offset+2 >= len(frame) {
		return sampledPixel{X: x, Y: y}
	}
	return sampledPixel{
		X: x,
		Y: y,
		R: frame[offset],
		G: frame[offset+1],
		B: frame[offset+2],
	}
}

func pixelNearColor(pixel sampledPixel, expected sampleColor) bool {
	return absInt(int(pixel.R)-int(expected.R)) <= sampleColorTolerance &&
		absInt(int(pixel.G)-int(expected.G)) <= sampleColorTolerance &&
		absInt(int(pixel.B)-int(expected.B)) <= sampleColorTolerance
}

func runCommand(timeout time.Duration, executable string, args []string, stdout io.Writer) error {
	if timeout <= 0 {
		timeout = defaultCommandTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	var stderr bytes.Buffer
	if stdout != nil {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = io.Discard
	}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

func clampInt(value int, minimum int, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "pip-export-smoke failed: %v\n", err)
	os.Exit(1)
}
