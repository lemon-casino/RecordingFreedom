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
		Recording:  recordingprofile.Profile{Quality: recordingprofile.QualityHigh, FPS: 60, CaptureCursor: true},
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
	if config.OutputPath != filepath.Join(plan.Package.Dir, recpackage.ScreenVideoFile) {
		t.Fatalf("output path = %q, want package screen video", config.OutputPath)
	}
	if config.DiagnosticsPath != filepath.Join(plan.Package.Dir, recpackage.VideoDiagnosticsFile) {
		t.Fatalf("diagnostics path = %q, want package video diagnostics", config.DiagnosticsPath)
	}
	if config.Profile.Quality != recordingprofile.QualityHigh || config.Profile.FPS != 60 || !config.Profile.CaptureCursor {
		t.Fatalf("profile = %#v", config.Profile)
	}
}
