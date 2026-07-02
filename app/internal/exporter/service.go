package exporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

const (
	defaultExportTimeout = 10 * time.Minute
	defaultVideoPreset   = "veryfast"
	defaultVideoCRF      = "20"
	minReadableMP4Bytes  = 45
)

type CommandRunner interface {
	Run(context.Context, string, []string) error
}

type CommandRunnerFunc func(context.Context, string, []string) error

func (f CommandRunnerFunc) Run(ctx context.Context, executable string, args []string) error {
	return f(ctx, executable, args)
}

type Options struct {
	FFmpegPath             string
	Timeout                time.Duration
	VideoPreset            string
	CRF                    string
	SkipOutputVerification bool
}

type Result struct {
	OutputPath      string `json:"outputPath"`
	Bytes           int64  `json:"bytes"`
	ScreenInputPath string `json:"screenInputPath"`
	WebcamInputPath string `json:"webcamInputPath,omitempty"`
	PIPVisible      bool   `json:"pipVisible"`
	FFmpegPath      string `json:"ffmpegPath"`
	OutputVerified  bool   `json:"outputVerified"`
}

type Service struct {
	runner CommandRunner
}

func NewService() *Service {
	return &Service{runner: defaultRunner{}}
}

func NewServiceWithRunner(runner CommandRunner) *Service {
	if runner == nil {
		runner = defaultRunner{}
	}
	return &Service{runner: runner}
}

func (s *Service) Export(ctx context.Context, plan exportplan.Plan, options Options) (Result, error) {
	if s == nil {
		s = NewService()
	}
	if s.runner == nil {
		s.runner = defaultRunner{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validatePlan(plan); err != nil {
		return Result{}, err
	}
	ffmpegPath, err := resolveFFmpegPath(options.FFmpegPath)
	if err != nil {
		return Result{}, err
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultExportTimeout
	}

	outputDir := filepath.Dir(plan.OutputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, err
	}
	tmp, err := os.CreateTemp(outputDir, ".recording-export-*.mp4")
	if err != nil {
		return Result{}, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath)

	args, err := FFmpegArgs(plan, tmpPath, options)
	if err != nil {
		return Result{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := s.runner.Run(runCtx, ffmpegPath, args); err != nil {
		_ = os.Remove(tmpPath)
		return Result{}, err
	}
	info, err := requireReadableFile(tmpPath, minReadableMP4Bytes)
	if err != nil {
		_ = os.Remove(tmpPath)
		return Result{}, fmt.Errorf("export output: %w", err)
	}
	outputVerified := false
	if !options.SkipOutputVerification {
		verifyCtx, verifyCancel := context.WithTimeout(ctx, timeout)
		defer verifyCancel()
		if err := verifyExportOutput(verifyCtx, s.runner, ffmpegPath, tmpPath); err != nil {
			_ = os.Remove(tmpPath)
			return Result{}, err
		}
		outputVerified = true
	}
	if err := installFileAtomically(plan.OutputPath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return Result{}, err
	}
	return Result{
		OutputPath:      plan.OutputPath,
		Bytes:           info.Size(),
		ScreenInputPath: plan.ScreenInputPath,
		WebcamInputPath: plan.WebcamInputPath,
		PIPVisible:      plan.PIPLayout.Visible,
		FFmpegPath:      ffmpegPath,
		OutputVerified:  outputVerified,
	}, nil
}

func FFmpegArgs(plan exportplan.Plan, outputPath string, options Options) ([]string, error) {
	if err := validatePlanForOutput(plan, outputPath); err != nil {
		return nil, err
	}
	if !plan.PIPLayout.Visible || strings.TrimSpace(plan.WebcamInputPath) == "" {
		return screenOnlyArgs(plan.ScreenInputPath, outputPath), nil
	}
	filter, err := pipFilter(plan)
	if err != nil {
		return nil, err
	}
	return pipArgs(plan.ScreenInputPath, plan.WebcamInputPath, filter, outputPath, options), nil
}

func screenOnlyArgs(screenInput string, outputPath string) []string {
	return []string{
		"-hide_banner", "-loglevel", "warning", "-y",
		"-i", screenInput,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c", "copy",
		"-movflags", "+faststart",
		outputPath,
	}
}

func pipArgs(screenInput string, webcamInput string, filter string, outputPath string, options Options) []string {
	preset := strings.TrimSpace(options.VideoPreset)
	if preset == "" {
		preset = defaultVideoPreset
	}
	crf := strings.TrimSpace(options.CRF)
	if crf == "" {
		crf = defaultVideoCRF
	}
	return []string{
		"-hide_banner", "-loglevel", "warning", "-y",
		"-i", screenInput,
		"-i", webcamInput,
		"-filter_complex", filter,
		"-map", "[vout]",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-preset", preset,
		"-crf", crf,
		"-c:a", "copy",
		"-movflags", "+faststart",
		outputPath,
	}
}

func pipFilter(plan exportplan.Plan) (string, error) {
	layout := plan.PIPLayout
	rect := layout.Rect
	if !layout.Visible || !rect.Visible {
		return "", errors.New("visible PIP export requires a visible PIP rect")
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return "", fmt.Errorf("visible PIP export requires positive PIP size, got %dx%d", rect.Width, rect.Height)
	}
	width := evenDimension(rect.Width)
	height := evenDimension(rect.Height)
	parts := []string{
		"[0:v]setpts=PTS-STARTPTS[base]",
		webcamFilter(width, height, plan.WebcamStartOffsetMs, layout),
		fmt.Sprintf("[base][pip]overlay=%d:%d:eof_action=pass:repeatlast=0,format=yuv420p,pad=ceil(iw/2)*2:ceil(ih/2)*2[vout]", rect.X, rect.Y),
	}
	return strings.Join(parts, ";"), nil
}

func webcamFilter(width int, height int, offsetMs int, layout pip.Placement) string {
	filters := []string{
		fmt.Sprintf("[1:v]%s", setPTSExpr(offsetMs)),
		fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase", width, height),
		fmt.Sprintf("crop=%d:%d", width, height),
		"format=rgba",
	}
	if layout.Mirror {
		filters = append(filters, "hflip")
	}
	if alpha := alphaMaskFilter(layout.Shape, layout.EdgeFeather); alpha != "" {
		filters = append(filters, alpha)
	}
	return strings.Join(filters, ",") + "[pip]"
}

func setPTSExpr(offsetMs int) string {
	if offsetMs == 0 {
		return "setpts=PTS-STARTPTS"
	}
	seconds := float64(absInt(offsetMs)) / 1000
	if offsetMs > 0 {
		return fmt.Sprintf("setpts=PTS-STARTPTS+%.3f/TB", seconds)
	}
	return fmt.Sprintf("setpts=PTS-STARTPTS-%.3f/TB", seconds)
}

func alphaMaskFilter(shape pip.Shape, edgeFeather float64) string {
	edgeFeather = clampFloat(edgeFeather, 0, pip.MaximumEdgeFeather)
	switch pip.NormalizeShape(shape) {
	case pip.ShapeCircle:
		distance := "sqrt(pow(X-W/2,2)+pow(Y-H/2,2))"
		radius := "min(W,H)/2"
		if edgeFeather <= 0.001 {
			return fmt.Sprintf("geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':a='if(lte(%s,%s),255,0)'", distance, radius)
		}
		inner := fmt.Sprintf("(%s)*(1-%.3f)", radius, edgeFeather)
		return fmt.Sprintf("geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':a='if(lte(%s,%s),255,if(gte(%s,%s),0,255*(%s-%s)/(%s-%s)))'", distance, inner, distance, radius, radius, distance, radius, inner)
	case pip.ShapeSquare:
		if edgeFeather <= 0.001 {
			return ""
		}
		edge := fmt.Sprintf("(min(W,H)*%.3f)", edgeFeather)
		distance := "min(min(X,W-1-X),min(Y,H-1-Y))"
		return fmt.Sprintf("geq=r='r(X,Y)':g='g(X,Y)':b='b(X,Y)':a='if(gte(%s,%s),255,255*%s/%s)'", distance, edge, distance, edge)
	default:
		return ""
	}
}

func validatePlan(plan exportplan.Plan) error {
	if strings.TrimSpace(plan.OutputPath) == "" {
		return errors.New("export output path is required")
	}
	if strings.TrimSpace(plan.ScreenInputPath) == "" {
		return errors.New("screen input path is required")
	}
	if sameCleanPath(plan.OutputPath, plan.ScreenInputPath) {
		return errors.New("export output must not overwrite the raw screen video")
	}
	if strings.TrimSpace(plan.WebcamInputPath) != "" && sameCleanPath(plan.OutputPath, plan.WebcamInputPath) {
		return errors.New("export output must not overwrite the raw webcam sidecar")
	}
	if _, err := requireReadableFile(plan.ScreenInputPath, 1); err != nil {
		return fmt.Errorf("screen input: %w", err)
	}
	if plan.PIPLayout.Visible {
		if strings.TrimSpace(plan.WebcamInputPath) == "" {
			return errors.New("visible PIP export requires a webcam input")
		}
		if _, err := requireReadableFile(plan.WebcamInputPath, 1); err != nil {
			return fmt.Errorf("webcam input: %w", err)
		}
	}
	return nil
}

func validatePlanForOutput(plan exportplan.Plan, outputPath string) error {
	if err := validatePlan(plan); err != nil {
		return err
	}
	if strings.TrimSpace(outputPath) == "" {
		return errors.New("temporary export output path is required")
	}
	if sameCleanPath(outputPath, plan.ScreenInputPath) || sameCleanPath(outputPath, plan.WebcamInputPath) {
		return errors.New("temporary export output must not overwrite raw media")
	}
	return nil
}

func resolveFFmpegPath(configured string) (string, error) {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured), nil
	}
	return video.ResolveFFmpegPath()
}

func verifyExportOutput(ctx context.Context, runner CommandRunner, ffmpegPath string, outputPath string) error {
	if runner == nil {
		runner = defaultRunner{}
	}
	args := []string{
		"-hide_banner", "-v", "error",
		"-i", outputPath,
		"-map", "0:v:0",
		"-frames:v", "1",
		"-f", "null",
		"-",
	}
	if err := runner.Run(ctx, ffmpegPath, args); err != nil {
		return fmt.Errorf("verify export output video track: %w", err)
	}
	return nil
}

func requireReadableFile(path string, minBytes int64) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() || info.Size() < minBytes {
		return nil, fmt.Errorf("%q is not readable media", path)
	}
	return info, nil
}

func installFileAtomically(target string, replacement string) error {
	if _, err := os.Stat(target); err == nil {
		backup := target + ".before-export"
		_ = os.Remove(backup)
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("prepare export replacement: %w", err)
		}
		if err := os.Rename(replacement, target); err != nil {
			_ = os.Rename(backup, target)
			return fmt.Errorf("install export replacement: %w", err)
		}
		_ = os.Remove(backup)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat export target: %w", err)
	}
	if err := os.Rename(replacement, target); err != nil {
		return fmt.Errorf("install export output: %w", err)
	}
	return nil
}

type defaultRunner struct{}

func (defaultRunner) Run(ctx context.Context, executable string, args []string) error {
	cmd := exec.CommandContext(ctx, executable, args...)
	configureBackgroundCommand(cmd)
	stderr := &bytes.Buffer{}
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg export failed: %w%s", err, stderrSuffix(stderr))
	}
	return nil
}

func stderrSuffix(stderr *bytes.Buffer) string {
	if stderr == nil || stderr.Len() == 0 {
		return ""
	}
	message := strings.TrimSpace(stderr.String())
	if message == "" {
		return ""
	}
	if len(message) > 1200 {
		message = message[len(message)-1200:]
	}
	return ": FFmpeg: " + message
}

func evenDimension(value int) int {
	if value <= 0 {
		return 2
	}
	if value%2 == 0 {
		return value
	}
	return value + 1
}

func sameCleanPath(a string, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return filepath.Clean(absA) == filepath.Clean(absB)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func clampFloat(value float64, minimum float64, maximum float64) float64 {
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
