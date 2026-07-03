package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
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
	defaultWidth               = 320
	defaultHeight              = 180
	defaultDuration            = 2 * time.Second
	defaultCommandTimeout      = 2 * time.Minute
	defaultTimelineSegments    = 2
	defaultTimelineMode        = "snapshot-segments"
	timelineModeSnapshot       = "snapshot-segments"
	timelineModeElementPNGs    = "element-pngs"
	maxTimelineSmokeSegments   = 200
	minTimelineDuration        = 1500 * time.Millisecond
	minTimelineSegmentDuration = 250 * time.Millisecond
	sampleColorTolerance       = 95
)

type options struct {
	dataDir   string
	keep      bool
	ffmpeg    string
	width     int
	height    int
	duration  time.Duration
	timeout   time.Duration
	outputRel string
	segments  int
	timeline  string
	source    sourceOptions
}

type sourceOptions struct {
	Type         string
	ID           string
	Name         string
	X            int
	Y            int
	DisplayIndex int
	NativeID     string
}

type report struct {
	OK                  bool                                `json:"ok"`
	DataDir             string                              `json:"dataDir"`
	VideoDir            string                              `json:"videoDir"`
	PackageDir          string                              `json:"packageDir"`
	ManifestPath        string                              `json:"manifestPath"`
	OutputPath          string                              `json:"outputPath"`
	OutputBytes         int64                               `json:"outputBytes"`
	OutputVerified      bool                                `json:"outputVerified"`
	FFmpegPath          string                              `json:"ffmpegPath"`
	AnnotationsVisible  bool                                `json:"annotationsVisible"`
	AnnotationTimeline  string                              `json:"annotationTimeline"`
	AnnotationInputPath string                              `json:"annotationInputPath,omitempty"`
	AnnotationSnapshots int                                 `json:"annotationSnapshots"`
	Source              recpackage.ManifestSource           `json:"source"`
	AnnotationTarget    recpackage.ManifestAnnotationTarget `json:"annotationTarget"`
	TimelineSwitchMs    int64                               `json:"timelineSwitchMs"`
	RedPixel            sampledPixel                        `json:"redPixel"`
	GreenPixel          sampledPixel                        `json:"greenPixel"`
	BackgroundPixel     sampledPixel                        `json:"backgroundPixel"`
	SegmentSamples      []timelineSample                    `json:"segmentSamples,omitempty"`
	DurationMs          int64                               `json:"durationMs"`
	KeptDataDir         bool                                `json:"keptDataDir"`
	Warnings            []string                            `json:"warnings,omitempty"`
	Plan                exportplan.Plan                     `json:"plan"`
}

type timelineSample struct {
	Segment  int          `json:"segment"`
	AtMs     int64        `json:"atMs"`
	Expected sampleColor  `json:"expected"`
	Pixel    sampledPixel `json:"pixel"`
}

type sampleColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type sampledPixel struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type annotationTimelineSpec struct {
	SwitchMs int64
	Samples  []annotationSampleTarget
}

type annotationSampleTarget struct {
	Segment  int
	At       time.Duration
	Expected color.NRGBA
}

func main() {
	var opts options
	flag.StringVar(&opts.dataDir, "data-dir", "", "data root for the generated .rfrec package; defaults to a temporary directory")
	flag.BoolVar(&opts.keep, "keep", false, "keep the generated data root for manual inspection")
	flag.StringVar(&opts.ffmpeg, "ffmpeg", "", "optional FFmpeg executable path")
	flag.IntVar(&opts.width, "width", defaultWidth, "synthetic screen video width")
	flag.IntVar(&opts.height, "height", defaultHeight, "synthetic screen video height")
	flag.DurationVar(&opts.duration, "duration", defaultDuration, "synthetic screen video duration")
	flag.DurationVar(&opts.timeout, "timeout", defaultCommandTimeout, "FFmpeg command timeout")
	flag.StringVar(&opts.outputRel, "output", exportplan.DefaultOutputPath, "package-relative export path")
	flag.IntVar(&opts.segments, "segments", defaultTimelineSegments, "number of annotation timeline snapshots to synthesize and verify")
	flag.StringVar(&opts.timeline, "timeline", defaultTimelineMode, "annotation timeline mode to verify: snapshot-segments or element-pngs")
	flag.StringVar(&opts.source.Type, "source-type", "screen", "recording source type to write into the smoke manifest: screen, all-screens, region, or window")
	flag.StringVar(&opts.source.ID, "source-id", "", "recording source id; defaults from source-type")
	flag.StringVar(&opts.source.Name, "source-name", "Annotation Export Smoke", "recording source display name")
	flag.IntVar(&opts.source.X, "source-x", 0, "recording source geometry x")
	flag.IntVar(&opts.source.Y, "source-y", 0, "recording source geometry y")
	flag.IntVar(&opts.source.DisplayIndex, "source-display-index", 1, "recording source display index")
	flag.StringVar(&opts.source.NativeID, "source-native-id", "annotation-export-smoke-display-1", "recording source native display/window id")
	flag.Parse()

	result, err := run(opts)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err != nil {
		_ = encoder.Encode(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "encode annotation export smoke report: %v\n", err)
		os.Exit(1)
	}
}

func run(opts options) (report, error) {
	if opts.width <= 0 || opts.height <= 0 {
		return report{}, fmt.Errorf("invalid synthetic canvas size %dx%d", opts.width, opts.height)
	}
	if opts.duration <= 0 {
		return report{}, errors.New("duration must be positive")
	}
	if opts.duration < minTimelineDuration {
		return report{}, fmt.Errorf("duration must be at least %s to verify annotation snapshot timeline switching", minTimelineDuration)
	}
	if opts.segments < 2 {
		return report{}, errors.New("segments must be at least 2 to verify annotation timeline switching")
	}
	if opts.segments > maxTimelineSmokeSegments {
		return report{}, fmt.Errorf("segments must be <= %d to keep the smoke test bounded", maxTimelineSmokeSegments)
	}
	if opts.duration/time.Duration(opts.segments) < minTimelineSegmentDuration {
		return report{}, fmt.Errorf("duration %s is too short for %d segments; each segment must be at least %s", opts.duration, opts.segments, minTimelineSegmentDuration)
	}
	opts.timeline = strings.TrimSpace(opts.timeline)
	if opts.timeline == "" {
		opts.timeline = defaultTimelineMode
	}
	if opts.timeline != timelineModeSnapshot && opts.timeline != timelineModeElementPNGs {
		return report{}, fmt.Errorf("unsupported annotation timeline %q; use %q or %q", opts.timeline, timelineModeSnapshot, timelineModeElementPNGs)
	}
	source, target, err := annotationSmokeSource(opts.source, opts.width, opts.height)
	if err != nil {
		return report{}, err
	}
	if opts.timeout <= 0 {
		opts.timeout = defaultCommandTimeout
	}
	ffmpegPath := strings.TrimSpace(opts.ffmpeg)
	if ffmpegPath == "" {
		resolved, err := video.ResolveFFmpegPath()
		if err != nil {
			return report{}, err
		}
		ffmpegPath = resolved
	}

	startedAt := time.Now()
	dataDir, cleanup, err := prepareDataDir(opts)
	if err != nil {
		return report{}, err
	}
	defer cleanup()

	videoDir := filepath.Join(dataDir, "data", "video")
	packageDir := filepath.Join(videoDir, "recording-annotation-export-smoke-"+startedAt.Format("20060102-150405-000")+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		return report{}, err
	}

	screenPath := filepath.Join(packageDir, recpackage.ScreenVideoFile)
	if err := writeSyntheticScreenVideo(ffmpegPath, screenPath, opts.width, opts.height, opts.duration, opts.timeout); err != nil {
		return report{}, err
	}
	timeline, err := writeAnnotationAssets(packageDir, opts.width, opts.height, opts.duration, opts.segments, opts.timeline)
	if err != nil {
		return report{}, err
	}
	manifestPath, err := writeManifest(packageDir, opts.width, opts.height, opts.duration, startedAt, source, target)
	if err != nil {
		return report{}, err
	}

	includeAnnotations := true
	plan, err := exportplan.NewService(nil).Plan(exportplan.Request{
		VideoDir:                videoDir,
		PackageDir:              packageDir,
		OutputPath:              opts.outputRel,
		Canvas:                  pip.Size{Width: opts.width, Height: opts.height},
		RequireSync:             true,
		IncludeAnnotations:      &includeAnnotations,
		PrepareAnnotationAssets: opts.timeline == timelineModeElementPNGs,
	})
	if err != nil {
		return report{}, fmt.Errorf("create annotation export plan: %w", err)
	}
	if !plan.AnnotationsVisible {
		return report{}, errors.New("annotation export plan did not enable annotations")
	}
	if len(plan.AnnotationSnapshots) == 0 && strings.TrimSpace(plan.AnnotationInputPath) == "" {
		return report{}, errors.New("annotation export plan has no annotation input")
	}
	if plan.AnnotationTimeline != opts.timeline || len(plan.AnnotationSnapshots) != opts.segments {
		return report{}, fmt.Errorf("annotation export plan timeline = %q snapshots = %d, want %d %s", plan.AnnotationTimeline, len(plan.AnnotationSnapshots), opts.segments, opts.timeline)
	}
	if opts.timeline == timelineModeElementPNGs && plan.AnnotationRenderMode != timelineModeElementPNGs {
		return report{}, fmt.Errorf("annotation render mode = %q, want %q", plan.AnnotationRenderMode, timelineModeElementPNGs)
	}

	exportResult, err := exporter.NewService().Export(context.Background(), plan, exporter.Options{
		FFmpegPath:  ffmpegPath,
		Timeout:     opts.timeout,
		VideoPreset: "ultrafast",
		CRF:         "18",
	})
	if err != nil {
		return report{}, fmt.Errorf("export annotated package: %w", err)
	}
	samples, background, err := verifyAnnotationPixels(ffmpegPath, exportResult.OutputPath, opts.width, opts.height, opts.duration, timeline.Samples, opts.timeout)
	if err != nil {
		return report{}, err
	}
	red := samples[0].Pixel
	green := samples[1].Pixel

	return report{
		OK:                  true,
		DataDir:             dataDir,
		VideoDir:            videoDir,
		PackageDir:          packageDir,
		ManifestPath:        manifestPath,
		OutputPath:          exportResult.OutputPath,
		OutputBytes:         exportResult.Bytes,
		OutputVerified:      exportResult.OutputVerified,
		FFmpegPath:          ffmpegPath,
		AnnotationsVisible:  plan.AnnotationsVisible,
		AnnotationTimeline:  plan.AnnotationTimeline,
		AnnotationInputPath: plan.AnnotationInputPath,
		AnnotationSnapshots: len(plan.AnnotationSnapshots),
		Source:              source,
		AnnotationTarget:    target,
		TimelineSwitchMs:    timeline.SwitchMs,
		RedPixel:            red,
		GreenPixel:          green,
		BackgroundPixel:     background,
		SegmentSamples:      samples,
		DurationMs:          time.Since(startedAt).Milliseconds(),
		KeptDataDir:         opts.keep,
		Warnings:            plan.Warnings,
		Plan:                plan,
	}, nil
}

func prepareDataDir(opts options) (string, func(), error) {
	dataDir := strings.TrimSpace(opts.dataDir)
	if dataDir == "" {
		tempRoot, err := os.MkdirTemp("", "recordingfreedom-annotation-export-smoke-*")
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

func writeSyntheticScreenVideo(ffmpegPath string, outputPath string, width int, height int, duration time.Duration, timeout time.Duration) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	seconds := duration.Seconds()
	args := []string{
		"-hide_banner", "-v", "error", "-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("color=c=black:s=%dx%d:r=30:d=%.3f", width, height, seconds),
		"-an",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		outputPath,
	}
	return runFFmpeg(timeout, ffmpegPath, args, nil)
}

func writeAnnotationAssets(packageDir string, width int, height int, duration time.Duration, segments int, timelineMode string) (annotationTimelineSpec, error) {
	scenePath := filepath.Join(packageDir, recpackage.AnnotationSceneFile)
	eventsPath := filepath.Join(packageDir, recpackage.AnnotationEventsFile)
	snapshotPath := filepath.Join(packageDir, recpackage.AnnotationSnapshotFile)
	timeline := buildAnnotationTimeline(duration, segments)
	paths := []string{scenePath, eventsPath, snapshotPath}
	for index := range timeline.Samples {
		if timelineMode == timelineModeElementPNGs {
			paths = append(paths, filepath.Join(packageDir, renderedTimelineRelativePath(index+1)))
		} else {
			paths = append(paths, filepath.Join(packageDir, timelineSnapshotRelativePath(index+1)))
		}
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return annotationTimelineSpec{}, err
		}
	}
	scene := fmt.Sprintf(`{"type":"excalidraw","version":2,"source":"RecordingFreedom annotation-export-smoke","elements":[],"appState":{"viewBackgroundColor":"transparent","width":%d,"height":%d},"files":{}}`+"\n", width, height)
	if err := os.WriteFile(scenePath, []byte(scene), 0o644); err != nil {
		return annotationTimelineSpec{}, err
	}
	var events []string
	var finalPNG []byte
	for index, sample := range timeline.Samples {
		sequence := index + 1
		pngData, err := annotationPNG(width, height, sample.Expected)
		if err != nil {
			return annotationTimelineSpec{}, err
		}
		finalPNG = pngData
		relativePath := timelineSnapshotRelativePath(sequence)
		if timelineMode == timelineModeElementPNGs {
			relativePath = renderedTimelineRelativePath(sequence)
		}
		if err := os.WriteFile(filepath.Join(packageDir, relativePath), pngData, 0o644); err != nil {
			return annotationTimelineSpec{}, err
		}
		offsetMs := int64(index) * duration.Milliseconds() / int64(segments)
		if timelineMode == timelineModeElementPNGs {
			events = append(events, annotationElementEvent(sequence, offsetMs))
		} else {
			events = append(events, fmt.Sprintf(`{"type":"scene-snapshot","schemaVersion":1,"sequence":%d,"eventId":"annotation-export-smoke-%06d","recordingOffsetMs":%d,"wallOffsetMs":%d,"scenePath":"annotations/scene.excalidraw","snapshotPath":%q}`, sequence, sequence, offsetMs, offsetMs, filepath.ToSlash(relativePath)))
		}
	}
	if err := os.WriteFile(snapshotPath, finalPNG, 0o644); err != nil {
		return annotationTimelineSpec{}, err
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Join(events, "\n")+"\n"), 0o644); err != nil {
		return annotationTimelineSpec{}, err
	}
	return timeline, nil
}

func annotationPNG(width int, height int, fill color.NRGBA) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	rect := image.Rect(width/10, height/6, width/2, height/2)
	if rect.Dx() < 8 || rect.Dy() < 8 {
		rect = image.Rect(4, 4, maxInt(12, width-4), maxInt(12, height-4))
	}
	draw.Draw(img, rect, &image.Uniform{C: fill}, image.Point{}, draw.Src)
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeManifest(packageDir string, width int, height int, duration time.Duration, startedAt time.Time, source recpackage.ManifestSource, target recpackage.ManifestAnnotationTarget) (string, error) {
	completedAt := startedAt.Add(duration)
	durationMs := duration.Milliseconds()
	manifest := recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		CreatedAt:     startedAt.UTC(),
		CompletedAt:   &completedAt,
		Status:        recpackage.StatusReady,
		RecordingMode: recpackage.RecordingModeScreen,
		Media: recpackage.ManifestMedia{
			ScreenVideoPath: recpackage.ScreenVideoFile,
		},
		Source: source,
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
			PIPPreset: string(pip.PresetOff),
		},
		Annotations: &recpackage.ManifestAnnotations{
			Enabled:       true,
			Mode:          "overlay",
			ScenePath:     recpackage.AnnotationSceneFile,
			EventsPath:    recpackage.AnnotationEventsFile,
			SnapshotPath:  recpackage.AnnotationSnapshotFile,
			CapturePolicy: "export-compose",
			Target:        target,
		},
		Diagnostics: recpackage.ManifestDiagnostics{
			Sync: &recpackage.ManifestSyncDiagnostics{
				TimelineBase: recpackage.TimelineBaseMedia,
				Screen: recpackage.ManifestTrackDiagnostics{
					Enabled:     true,
					Path:        recpackage.ScreenVideoFile,
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

func annotationSmokeSource(opts sourceOptions, width int, height int) (recpackage.ManifestSource, recpackage.ManifestAnnotationTarget, error) {
	sourceType := strings.TrimSpace(opts.Type)
	if sourceType == "" {
		sourceType = "screen"
	}
	switch sourceType {
	case "screen", "all-screens", "region", "window":
	default:
		return recpackage.ManifestSource{}, recpackage.ManifestAnnotationTarget{}, fmt.Errorf("unsupported source-type %q; use screen, all-screens, region, or window", sourceType)
	}
	sourceID := strings.TrimSpace(opts.ID)
	if sourceID == "" {
		sourceID = defaultAnnotationSmokeSourceID(sourceType)
	}
	sourceName := strings.TrimSpace(opts.Name)
	if sourceName == "" {
		sourceName = "Annotation Export Smoke"
	}
	displayIndex := opts.DisplayIndex
	if displayIndex <= 0 {
		displayIndex = 1
	}
	nativeID := strings.TrimSpace(opts.NativeID)
	if nativeID == "" {
		nativeID = "annotation-export-smoke-display-1"
	}
	geometry := &recpackage.ManifestSourceGeometry{
		X:            opts.X,
		Y:            opts.Y,
		Width:        width,
		Height:       height,
		DisplayIndex: displayIndex,
		NativeID:     nativeID,
	}
	source := recpackage.ManifestSource{
		Type:     sourceType,
		ID:       sourceID,
		Name:     sourceName,
		Geometry: geometry,
	}
	targetGeometry := *geometry
	target := recpackage.ManifestAnnotationTarget{
		Type:     sourceType,
		ID:       sourceID,
		Geometry: &targetGeometry,
	}
	return source, target, nil
}

func defaultAnnotationSmokeSourceID(sourceType string) string {
	switch sourceType {
	case "all-screens":
		return "all-screens:virtual-desktop"
	case "region":
		return "region:annotation-export-smoke"
	case "window":
		return "window:annotation-export-smoke"
	default:
		return "screen:annotation-export-smoke"
	}
}

func verifyAnnotationPixels(ffmpegPath string, outputPath string, width int, height int, duration time.Duration, targets []annotationSampleTarget, timeout time.Duration) ([]timelineSample, sampledPixel, error) {
	if len(targets) < 2 {
		return nil, sampledPixel{}, errors.New("at least two annotation samples are required")
	}
	sampleX := clampInt(width/5, 0, width-1)
	sampleY := clampInt(height/4, 0, height-1)
	backgroundX := clampInt(width*4/5, 0, width-1)
	backgroundY := clampInt(height*3/4, 0, height-1)
	samples := make([]timelineSample, 0, len(targets))
	var background sampledPixel
	for _, target := range targets {
		if target.At <= 0 || target.At >= duration {
			return samples, background, fmt.Errorf("invalid annotation sample time segment=%d at=%s duration=%s", target.Segment, target.At, duration)
		}
		frame, err := extractRGBFrame(ffmpegPath, outputPath, width, height, target.At, timeout)
		if err != nil {
			return samples, background, err
		}
		pixel := pixelAt(frame, width, sampleX, sampleY)
		background = pixelAt(frame, width, backgroundX, backgroundY)
		sample := timelineSample{
			Segment: target.Segment,
			AtMs:    target.At.Milliseconds(),
			Expected: sampleColor{
				R: target.Expected.R,
				G: target.Expected.G,
				B: target.Expected.B,
			},
			Pixel: pixel,
		}
		samples = append(samples, sample)
		if !pixelNearColor(pixel, target.Expected) {
			return samples, background, fmt.Errorf("annotation segment %d pixel mismatch at %dms: got %#v, want near rgb(%d,%d,%d)", target.Segment, sample.AtMs, pixel, target.Expected.R, target.Expected.G, target.Expected.B)
		}
	}
	if background.R > 80 || background.G > 80 || background.B > 80 {
		return samples, background, fmt.Errorf("background pixel is not dark after annotation export: %#v", background)
	}
	return samples, background, nil
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
	if err := runFFmpeg(timeout, ffmpegPath, args, &stdout); err != nil {
		return nil, fmt.Errorf("extract annotated frame: %w", err)
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

func runFFmpeg(timeout time.Duration, ffmpegPath string, args []string, stdout io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
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

func buildAnnotationTimeline(duration time.Duration, segments int) annotationTimelineSpec {
	durationMs := duration.Milliseconds()
	samples := make([]annotationSampleTarget, 0, segments)
	for index := 0; index < segments; index++ {
		startMs := int64(index) * durationMs / int64(segments)
		endMs := int64(index+1) * durationMs / int64(segments)
		sampleMs := startMs + maxInt64(1, (endMs-startMs)/2)
		if sampleMs >= durationMs {
			sampleMs = durationMs - 1
		}
		samples = append(samples, annotationSampleTarget{
			Segment:  index + 1,
			At:       time.Duration(sampleMs) * time.Millisecond,
			Expected: annotationSegmentColor(index),
		})
	}
	return annotationTimelineSpec{
		SwitchMs: durationMs / int64(segments),
		Samples:  samples,
	}
}

func annotationSegmentColor(index int) color.NRGBA {
	palette := []color.NRGBA{
		{R: 239, G: 68, B: 68, A: 255},
		{R: 34, G: 197, B: 94, A: 255},
		{R: 59, G: 130, B: 246, A: 255},
		{R: 245, G: 158, B: 11, A: 255},
		{R: 168, G: 85, B: 247, A: 255},
		{R: 6, G: 182, B: 212, A: 255},
		{R: 249, G: 115, B: 22, A: 255},
		{R: 236, G: 72, B: 153, A: 255},
	}
	return palette[index%len(palette)]
}

func timelineSnapshotRelativePath(sequence int) string {
	return filepath.ToSlash(filepath.Join(recpackage.AnnotationSnapshotsDir, fmt.Sprintf("annotation-%06d.png", sequence)))
}

func renderedTimelineRelativePath(sequence int) string {
	return filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, fmt.Sprintf("annotation-%06d.png", sequence)))
}

func annotationElementEvent(sequence int, offsetMs int64) string {
	eventType := "element-updated"
	if sequence == 1 {
		eventType = "element-created"
	}
	element := map[string]any{
		"id":      "annotation-export-smoke-element",
		"type":    "rectangle",
		"version": sequence,
		"x":       32,
		"y":       32,
		"width":   128,
		"height":  72,
	}
	data := map[string]any{
		"type":              eventType,
		"schemaVersion":     1,
		"sequence":          sequence,
		"eventId":           fmt.Sprintf("annotation-export-smoke-element-%06d", sequence),
		"recordingOffsetMs": offsetMs,
		"wallOffsetMs":      offsetMs,
		"scenePath":         recpackage.AnnotationSceneFile,
		"elementId":         "annotation-export-smoke-element",
		"elementType":       "rectangle",
		"elementVersion":    sequence,
		"element":           element,
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func pixelNearColor(pixel sampledPixel, expected color.NRGBA) bool {
	return absInt(int(pixel.R)-int(expected.R)) <= sampleColorTolerance &&
		absInt(int(pixel.G)-int(expected.G)) <= sampleColorTolerance &&
		absInt(int(pixel.B)-int(expected.B)) <= sampleColorTolerance
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
