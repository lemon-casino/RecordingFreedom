package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exporter"
	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
)

type report struct {
	OK                  bool            `json:"ok"`
	VideoDir            string          `json:"videoDir"`
	PackageDir          string          `json:"packageDir"`
	OutputPath          string          `json:"outputPath"`
	OutputBytes         int64           `json:"outputBytes"`
	OutputVerified      bool            `json:"outputVerified"`
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
	return report{
		OK:                  true,
		VideoDir:            opts.videoDir,
		PackageDir:          opts.packageDir,
		OutputPath:          exportResult.OutputPath,
		OutputBytes:         exportResult.Bytes,
		OutputVerified:      exportResult.OutputVerified,
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

func fail(err error) {
	fmt.Fprintf(os.Stderr, "pip-export-smoke failed: %v\n", err)
	os.Exit(1)
}
