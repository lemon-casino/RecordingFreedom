package exportplan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestPlanReadyPackageWithPIPAndSyncDiagnostics(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Camera: true})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		Canvas:      pip.Size{Width: 1920, Height: 1080},
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.PackageDir != packageDir {
		t.Fatalf("packageDir = %q, want %q", plan.PackageDir, packageDir)
	}
	if plan.OutputPath != filepath.Join(packageDir, DefaultOutputPath) {
		t.Fatalf("output path = %q, want package exports output", plan.OutputPath)
	}
	if plan.ScreenInputPath != filepath.Join(packageDir, "screen.mp4") {
		t.Fatalf("screen input = %q, want package screen.mp4", plan.ScreenInputPath)
	}
	if plan.WebcamInputPath != filepath.Join(packageDir, "webcam.mov") {
		t.Fatalf("webcam input = %q, want package webcam.mov", plan.WebcamInputPath)
	}
	if plan.WebcamStartOffsetMs != 120 {
		t.Fatalf("webcam offset = %d, want 120", plan.WebcamStartOffsetMs)
	}
	if plan.PIPPreset != string(pip.PresetFree) || !plan.PIPRect.Visible {
		t.Fatalf("pip = preset:%q rect:%#v, want visible free layout", plan.PIPPreset, plan.PIPRect)
	}
	if plan.PIPConfig.Shape != pip.ShapeSquare || plan.PIPConfig.Mirror || plan.PIPConfig.EdgeFeather != 0.2 {
		t.Fatalf("pip config = %#v, want square non-mirrored feathered layout", plan.PIPConfig)
	}
	if !plan.PIPLayout.Visible || plan.PIPLayout.Shape != pip.ShapeSquare || plan.PIPLayout.Mirror {
		t.Fatalf("pip layout = %#v, want visible square non-mirrored layout", plan.PIPLayout)
	}
	if plan.PIPRect.X >= 1920/2 || plan.PIPRect.Y <= 1080/2 {
		t.Fatalf("pip rect = %#v, want lower-left-ish free overlay", plan.PIPRect)
	}
	if plan.TimelineBase != recpackage.TimelineBaseMedia {
		t.Fatalf("timeline base = %q, want media timestamp", plan.TimelineBase)
	}
	if len(plan.PauseSegments) != 1 || plan.PauseSegments[0].DurationMs != 250 {
		t.Fatalf("pause segments = %#v, want one 250ms pause", plan.PauseSegments)
	}
}

func TestPlanUsesManifestSourceGeometryAsPIPCanvasFallback(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Camera: true, SourceGeometry: true})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:   videoDir,
		PackageDir: packageDir,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !plan.PIPRect.Visible || plan.PIPRect.Width <= 0 || plan.PIPRect.X >= 1280 {
		t.Fatalf("pip rect = %#v, want placement derived from manifest source geometry", plan.PIPRect)
	}
}

func TestPlanScreenOnlyPackageHidesPIP(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{})

	plan, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	})
	if err != nil {
		t.Fatalf("Plan(screen only) error = %v", err)
	}
	if plan.PIPRect.Visible || plan.WebcamInputPath != "" || plan.WebcamStartOffsetMs != 0 {
		t.Fatalf("screen-only pip fields = rect:%#v webcam:%q offset:%d", plan.PIPRect, plan.WebcamInputPath, plan.WebcamStartOffsetMs)
	}
}

func TestPlanRejectsPackageOutsideVideoDir(t *testing.T) {
	videoDir := t.TempDir()
	outsideDir := createReadyPackage(t, t.TempDir(), readyPackageOptions{})

	if _, err := NewService(nil).Plan(Request{VideoDir: videoDir, PackageDir: outsideDir}); err == nil {
		t.Fatal("Plan() accepted a package outside videoDir")
	}
}

func TestPlanRejectsEscapingOutputPath(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{})

	if _, err := NewService(nil).Plan(Request{
		VideoDir:   videoDir,
		PackageDir: packageDir,
		OutputPath: "../export.mp4",
	}); err == nil {
		t.Fatal("Plan() accepted an escaping output path")
	}
}

func TestPlanRejectsMockPackageByDefault(t *testing.T) {
	videoDir := t.TempDir()
	pkg, err := recpackage.NewService().CreateMock(videoDir, recpackage.CreateMockRequest{
		Status: recpackage.StatusReady,
		Source: recpackage.ManifestSource{Type: "screen", ID: "screen:primary"},
	})
	if err != nil {
		t.Fatalf("CreateMock() error = %v", err)
	}

	if _, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  pkg.Dir,
		RequireSync: true,
	}); err == nil {
		t.Fatal("Plan() accepted a mock package as real media")
	}
}

func TestPlanRejectsMissingWebcamSidecarForVisiblePIP(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{Camera: true, MissingWebcam: true})

	if _, err := NewService(nil).Plan(Request{
		VideoDir:   videoDir,
		PackageDir: packageDir,
		Canvas:     pip.Size{Width: 1920, Height: 1080},
	}); err == nil || !strings.Contains(err.Error(), "webcamVideoPath") {
		t.Fatalf("Plan() error = %v, want webcamVideoPath error", err)
	}
}

func TestPlanRejectsEscapingDiagnosticsPath(t *testing.T) {
	videoDir := t.TempDir()
	packageDir := createReadyPackage(t, videoDir, readyPackageOptions{EscapingDiagnostics: true})

	if _, err := NewService(nil).Plan(Request{
		VideoDir:    videoDir,
		PackageDir:  packageDir,
		RequireSync: true,
	}); err == nil {
		t.Fatal("Plan() accepted an escaping diagnostics path")
	}
}

type readyPackageOptions struct {
	Camera              bool
	MissingWebcam       bool
	EscapingDiagnostics bool
	SourceGeometry      bool
}

func createReadyPackage(t *testing.T, videoDir string, opts readyPackageOptions) string {
	t.Helper()
	packageDir := filepath.Join(videoDir, "recording-export-test"+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(package) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "screen.mp4"), []byte("screen media"), 0o644); err != nil {
		t.Fatalf("WriteFile(screen) error = %v", err)
	}
	media := recpackage.ManifestMedia{ScreenVideoPath: "screen.mp4"}
	camera := recpackage.ManifestCamera{PIPPreset: string(pip.PresetOff)}
	webcamSync := recpackage.ManifestTrackDiagnostics{}
	if opts.Camera {
		camera = recpackage.ManifestCamera{
			Enabled:   true,
			DeviceID:  "camera:default",
			PIPPreset: string(pip.PresetBottomLeft),
			PIP: pip.Config{
				Preset:      pip.PresetFree,
				Shape:       pip.ShapeSquare,
				Mirror:      false,
				Position:    pip.Position{X: 0.1, Y: 0.9},
				Scale:       0.22,
				EdgeFeather: 0.2,
			},
		}
		media.WebcamVideoPath = "webcam.mov"
		media.WebcamStartOffsetMs = 120
		webcamSync = recpackage.ManifestTrackDiagnostics{
			Enabled:       true,
			Path:          "webcam.mov",
			Clock:         recpackage.TimelineBaseMedia,
			StartOffsetMs: 120,
			EndOffsetMs:   6120,
			DurationMs:    6000,
			FrameRate:     30,
		}
		if !opts.MissingWebcam {
			if err := os.WriteFile(filepath.Join(packageDir, "webcam.mov"), []byte("webcam media"), 0o644); err != nil {
				t.Fatalf("WriteFile(webcam) error = %v", err)
			}
		}
	}
	audioDiagnosticsPath := recpackage.AudioDiagnosticsFile
	if opts.EscapingDiagnostics {
		audioDiagnosticsPath = "../audio-diagnostics.json"
	}
	source := recpackage.ManifestSource{Type: "screen", ID: "screen:primary"}
	if opts.SourceGeometry {
		source.Geometry = &recpackage.ManifestSourceGeometry{Width: 1280, Height: 720}
	}
	manifest := recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		CreatedAt:     time.Date(2026, 6, 30, 18, 0, 0, 0, time.UTC),
		Status:        recpackage.StatusReady,
		Media:         media,
		Source:        source,
		Recording:     recordingprofile.Profile{Quality: recordingprofile.QualityHigh, FPS: 30, CaptureCursor: true},
		Camera:        camera,
		Diagnostics: recpackage.ManifestDiagnostics{
			Sync: &recpackage.ManifestSyncDiagnostics{
				TimelineBase:         recpackage.TimelineBaseMedia,
				AudioDiagnosticsPath: audioDiagnosticsPath,
				VideoDiagnosticsPath: recpackage.VideoDiagnosticsFile,
				Screen: recpackage.ManifestTrackDiagnostics{
					Enabled:       true,
					Path:          "screen.mp4",
					Clock:         recpackage.TimelineBaseMedia,
					EndOffsetMs:   6000,
					DurationMs:    6000,
					DroppedFrames: 0,
					FrameRate:     30,
				},
				Webcam:        webcamSync,
				PauseSegments: []recpackage.ManifestPauseSegment{{StartOffsetMs: 1000, EndOffsetMs: 1250, DurationMs: 250}},
			},
		},
	}
	if opts.EscapingDiagnostics {
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatalf("MarshalIndent(manifest) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(packageDir, recpackage.ManifestFile), append(data, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile(manifest) error = %v", err)
		}
		return packageDir
	}
	if err := recpackage.NewService().WriteManifest(filepath.Join(packageDir, recpackage.ManifestFile), manifest); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}
	return packageDir
}
