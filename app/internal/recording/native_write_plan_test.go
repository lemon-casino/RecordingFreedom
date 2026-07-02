package recording

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestCreateNativeWritePlanMapsNormalizedStartRequest(t *testing.T) {
	packages := recpackage.NewService()
	videoDir := t.TempDir()
	createdAt := time.Date(2026, 6, 30, 18, 55, 0, 123000000, time.UTC)

	plan, err := CreateNativeWritePlan(packages, BackendScreenCaptureKit, BackendStartRequest{
		VideoDir:  videoDir,
		CreatedAt: createdAt,
		StartRequest: StartRequest{
			SourceID:   " cgdisplay:1 ",
			SourceType: SourceScreen,
			SourceName: " Built-in Display ",
			Recording: recordingprofile.Profile{
				Quality:          recordingprofile.QualityHigh,
				FPS:              60,
				CaptureCursor:    true,
				CountdownSeconds: 3,
			},
			Audio: AudioRequest{
				System:           true,
				Microphone:       true,
				NoiseSuppression: true,
			},
			Camera: CameraRequest{
				Enabled:   true,
				PIPPreset: "bottom-left",
				PIP: pip.Config{
					Preset:      pip.PresetFree,
					Shape:       pip.ShapeSquare,
					Mirror:      false,
					Position:    pip.Position{X: 0.2, Y: 0.7},
					Scale:       0.32,
					EdgeFeather: 0.18,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNativeWritePlan() error = %v", err)
	}
	if filepath.Dir(plan.Package.Dir) != videoDir {
		t.Fatalf("package parent = %q, want %q", filepath.Dir(plan.Package.Dir), videoDir)
	}
	if plan.ScreenVideoPath != filepath.Join(plan.Package.Dir, recpackage.ScreenVideoFile) {
		t.Fatalf("screen write path = %q, want package screen file", plan.ScreenVideoPath)
	}
	if plan.WebcamVideoPath != filepath.Join(plan.Package.Dir, recpackage.WebcamVideoFile) {
		t.Fatalf("webcam write path = %q, want package webcam file", plan.WebcamVideoPath)
	}

	manifest, err := packages.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Source.ID != "cgdisplay:1" || manifest.Source.Name != "Built-in Display" {
		t.Fatalf("source = %#v, want trimmed native source", manifest.Source)
	}
	if manifest.Audio.SystemDeviceID != defaultSystemAudioID {
		t.Fatalf("system device = %q, want default", manifest.Audio.SystemDeviceID)
	}
	if manifest.Audio.MicrophoneDeviceID != defaultMicrophoneID {
		t.Fatalf("microphone device = %q, want default", manifest.Audio.MicrophoneDeviceID)
	}
	if manifest.Audio.MicrophoneNoiseSuppression != recpackage.NoiseSuppressionOn {
		t.Fatalf("noise suppression = %q, want rnnoise", manifest.Audio.MicrophoneNoiseSuppression)
	}
	if manifest.Camera.DeviceID != defaultCameraID || manifest.Camera.PIPPreset != "free" {
		t.Fatalf("camera = %#v, want default camera free layout", manifest.Camera)
	}
	if manifest.Camera.PIP.Shape != pip.ShapeSquare || manifest.Camera.PIP.Mirror || manifest.Camera.PIP.Scale != pip.MaximumScale || manifest.Camera.PIP.EdgeFeather != 0.18 {
		t.Fatalf("camera pip = %#v, want custom square layout", manifest.Camera.PIP)
	}
	if manifest.Diagnostics.Mock || manifest.Diagnostics.Sync != nil {
		t.Fatalf("native write plan diagnostics = %#v, want non-mock without sync", manifest.Diagnostics)
	}
}

func TestCreateNativeWritePlanOmitsDisabledStreams(t *testing.T) {
	packages := recpackage.NewService()
	plan, err := CreateNativeWritePlan(packages, BackendWindowsGraphicsCapture, BackendStartRequest{
		VideoDir: t.TempDir(),
		StartRequest: StartRequest{
			SourceID:   "window:123",
			SourceType: SourceWindow,
			Audio: AudioRequest{
				System:           false,
				SystemDeviceID:   "system-audio:stale",
				Microphone:       false,
				MicrophoneID:     "microphone:stale",
				NoiseSuppression: true,
				MicrophoneGain:   2,
			},
			Camera: CameraRequest{
				Enabled:   false,
				DeviceID:  "camera:stale",
				PIPPreset: "bottom-right",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNativeWritePlan() error = %v", err)
	}
	if plan.WebcamVideoPath != "" {
		t.Fatalf("disabled camera webcam path = %q, want empty", plan.WebcamVideoPath)
	}
	manifest, err := packages.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Audio.SystemDeviceID != "" || manifest.Audio.MicrophoneDeviceID != "" || manifest.Audio.MicrophoneNoiseSuppression != recpackage.NoiseSuppressionOff {
		t.Fatalf("disabled audio manifest = %#v, want cleared streams", manifest.Audio)
	}
	if manifest.Camera.Enabled || manifest.Camera.DeviceID != "" || manifest.Camera.PIPPreset != "off" || manifest.Camera.PIP.Preset != pip.PresetOff {
		t.Fatalf("disabled camera manifest = %#v, want off", manifest.Camera)
	}
	if manifest.Media.WebcamVideoPath != "" {
		t.Fatalf("disabled camera media = %#v, want no webcam", manifest.Media)
	}
}

func TestCreateNativeWritePlanUsesWindowsWebcamMP4(t *testing.T) {
	packages := recpackage.NewService()
	plan, err := CreateNativeWritePlan(packages, BackendFFmpegDesktopCapture, BackendStartRequest{
		VideoDir: t.TempDir(),
		StartRequest: StartRequest{
			SourceID:   "screen:primary",
			SourceType: SourceScreen,
			Camera: CameraRequest{
				Enabled:        true,
				DeviceID:       "camera:dshow:integrated-camera",
				DeviceNativeID: "Integrated Camera",
				PIPPreset:      "bottom-right",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNativeWritePlan() error = %v", err)
	}
	if plan.WebcamVideoPath != filepath.Join(plan.Package.Dir, recpackage.WindowsWebcamVideoFile) {
		t.Fatalf("windows webcam path = %q, want package webcam.mp4", plan.WebcamVideoPath)
	}
	manifest, err := packages.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Media.WebcamVideoPath != recpackage.WindowsWebcamVideoFile {
		t.Fatalf("manifest webcam path = %q, want webcam.mp4", manifest.Media.WebcamVideoPath)
	}
}

func TestCreateNativeWritePlanRequiresPackageService(t *testing.T) {
	_, err := CreateNativeWritePlan(nil, BackendScreenCaptureKit, BackendStartRequest{
		VideoDir:     t.TempDir(),
		StartRequest: StartRequest{SourceID: "screen:primary", SourceType: SourceScreen},
	})
	if err == nil {
		t.Fatal("CreateNativeWritePlan() error = nil, want package service error")
	}
}
