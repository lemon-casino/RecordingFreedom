package recording

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestCreateVideoCaptureConfigMapsNativeWritePlan(t *testing.T) {
	packages := recpackage.NewService()
	plan, err := CreateNativeWritePlan(packages, BackendScreenCaptureKit, BackendStartRequest{
		VideoDir:  t.TempDir(),
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "screen:display-42",
			SourceType: SourceScreen,
			SourceName: "Primary Display",
			SourceGeometry: &SourceGeometry{
				X:            -1440,
				Y:            0,
				Width:        2560,
				Height:       1440,
				DisplayIndex: 2,
				NativeID:     " display:42 ",
			},
			Audio: AudioRequest{System: true},
			Recording: recordingprofile.Profile{
				Quality:       recordingprofile.QualityHigh,
				FPS:           60,
				CaptureCursor: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNativeWritePlan() error = %v", err)
	}

	config, err := CreateVideoCaptureConfig(BackendScreenCaptureKit, StartRequest{
		SourceID:   " screen:display-42 ",
		SourceType: SourceScreen,
		SourceName: " Primary Display ",
		SourceGeometry: &SourceGeometry{
			X:            -1440,
			Y:            0,
			Width:        2560,
			Height:       1440,
			DisplayIndex: 2,
			NativeID:     " display:42 ",
		},
		Audio:     AudioRequest{System: true},
		Recording: recordingprofile.Profile{Quality: recordingprofile.QualityHigh, FPS: 60, CaptureCursor: true},
	}, plan)
	if err != nil {
		t.Fatalf("CreateVideoCaptureConfig() error = %v", err)
	}

	if config.Backend != BackendScreenCaptureKit {
		t.Fatalf("backend = %q, want %q", config.Backend, BackendScreenCaptureKit)
	}
	if config.SourceID != "screen:display-42" || config.SourceType != SourceScreen || config.SourceName != "Primary Display" {
		t.Fatalf("source = %#v", config)
	}
	if config.SourceGeometry == nil {
		t.Fatalf("source geometry = nil, want request geometry")
	}
	if config.SourceGeometry.X != -1440 ||
		config.SourceGeometry.Y != 0 ||
		config.SourceGeometry.Width != 2560 ||
		config.SourceGeometry.Height != 1440 ||
		config.SourceGeometry.DisplayIndex != 2 ||
		config.SourceGeometry.NativeID != "display:42" {
		t.Fatalf("source geometry = %#v, want normalized request geometry", config.SourceGeometry)
	}
	if config.OutputPath != filepath.Join(plan.Package.Dir, recpackage.ScreenVideoFile) {
		t.Fatalf("output path = %q, want package screen video", config.OutputPath)
	}
	if config.DiagnosticsPath != filepath.Join(plan.Package.Dir, recpackage.VideoDiagnosticsFile) {
		t.Fatalf("diagnostics path = %q, want package video diagnostics", config.DiagnosticsPath)
	}
	if config.Profile.Quality != recordingprofile.QualityHigh || config.Profile.FPS != 60 || !config.Profile.CaptureCursor {
		t.Fatalf("profile = %#v", config.Profile)
	}
	if !config.SystemAudio {
		t.Fatal("ScreenCaptureKit muxed system audio was not carried into video config")
	}
}

func TestCreateVideoCaptureConfigCarriesRegionGeometry(t *testing.T) {
	packages := recpackage.NewService()
	plan, err := CreateNativeWritePlan(packages, BackendWindowsGraphicsCapture, BackendStartRequest{
		VideoDir:  t.TempDir(),
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "region:custom",
			SourceType: SourceRegion,
			SourceName: "Custom Region",
			SourceGeometry: &SourceGeometry{
				X:        120,
				Y:        80,
				Width:    1728,
				Height:   906,
				NativeID: "region:virtual-desktop",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNativeWritePlan() error = %v", err)
	}

	config, err := CreateVideoCaptureConfig(BackendWindowsGraphicsCapture, StartRequest{
		SourceID:   "region:custom",
		SourceType: SourceRegion,
		SourceName: "Custom Region",
		SourceGeometry: &SourceGeometry{
			X:        120,
			Y:        80,
			Width:    1728,
			Height:   906,
			NativeID: "region:virtual-desktop",
		},
	}, plan)
	if err != nil {
		t.Fatalf("CreateVideoCaptureConfig() error = %v", err)
	}

	if config.SourceType != SourceRegion || config.SourceGeometry == nil {
		t.Fatalf("region config source = %#v, want region geometry", config)
	}
	if config.SourceGeometry.X != 120 || config.SourceGeometry.Y != 80 || config.SourceGeometry.Width != 1728 || config.SourceGeometry.Height != 906 || config.SourceGeometry.NativeID != "region:virtual-desktop" {
		t.Fatalf("region geometry = %#v, want selected virtual desktop rect", config.SourceGeometry)
	}
}
