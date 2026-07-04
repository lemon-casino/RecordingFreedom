package main

import (
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestNormalizePIPCameraPrefersDisplayName(t *testing.T) {
	got := normalizePIPCamera(PIPCamera{
		DeviceID: " camera:avfoundation:facetime ",
		NativeID: " 0 ",
		Name:     " Default FaceTime HD Camera ",
	}, "")
	if got.DeviceID != "camera:avfoundation:facetime" || got.NativeID != "0" || got.Name != "Default FaceTime HD Camera" {
		t.Fatalf("camera = %#v, want trimmed ids and display name", got)
	}
}

func TestNormalizePIPCameraFallsBackToNativeIDThenDeviceID(t *testing.T) {
	if got := normalizePIPCamera(PIPCamera{DeviceID: "camera:default", NativeID: "Integrated Camera"}, ""); got.Name != "Integrated Camera" {
		t.Fatalf("native fallback name = %q, want Integrated Camera", got.Name)
	}
	if got := normalizePIPCamera(PIPCamera{DeviceID: "camera:default"}, ""); got.Name != "camera:default" {
		t.Fatalf("device fallback name = %q, want camera:default", got.Name)
	}
}

func TestPIPCameraFromRecordingRequestUsesMediaInventoryName(t *testing.T) {
	service := &RecordingFreedomService{
		devices: devices.NewServiceWithMediaProvider(stubPIPMediaProvider{}),
	}
	got := service.pipCameraFromRecordingRequest(recording.CameraRequest{
		DeviceID:       "camera:avfoundation:facetime",
		DeviceNativeID: "0",
	})
	if got.DeviceID != "camera:avfoundation:facetime" || got.NativeID != "0" || got.Name != "FaceTime HD Camera" {
		t.Fatalf("camera target = %#v, want media inventory display name for preview matching", got)
	}
}

func TestPIPOverlayStateCarriesRecordingPreviewImagePath(t *testing.T) {
	service := &RecordingFreedomService{}
	previewPath := filepath.Join("data", "video", "recording.rfrec", "cache", "pip-camera-preview.jpg")

	state, err := service.pipOverlayState(pip.DefaultConfig(), "recording", PIPCamera{
		DeviceID: "camera:dshow:integrated-camera",
		Name:     "Integrated Camera",
	}, " "+previewPath+" ")
	if err != nil {
		t.Fatalf("pipOverlayState() error = %v", err)
	}
	if state.Mode != "recording" || state.PreviewImagePath != previewPath {
		t.Fatalf("state mode/path = %q/%q, want recording preview path %q", state.Mode, state.PreviewImagePath, previewPath)
	}
}

func TestPIPOverlayBoundsFollowCapsuleScreenWorkArea(t *testing.T) {
	screens := []*application.Screen{
		{
			Bounds:   application.Rect{X: 0, Y: 0, Width: 1440, Height: 900},
			WorkArea: application.Rect{X: 0, Y: 0, Width: 1440, Height: 860},
		},
		{
			Bounds:   application.Rect{X: 1440, Y: 0, Width: 1920, Height: 1080},
			WorkArea: application.Rect{X: 1440, Y: 0, Width: 1920, Height: 1040},
		},
	}
	capsule := application.Rect{X: 1840, Y: 900, Width: 380, Height: 96}

	got, ok := pipOverlayBoundsForCapsule(screens, capsule, true)
	if !ok {
		t.Fatal("pipOverlayBoundsForCapsule() did not find the capsule screen")
	}
	want := screens[1].WorkArea
	if got != want {
		t.Fatalf("pip overlay bounds = %#v, want capsule screen work area %#v", got, want)
	}
}

func TestPIPOverlayPlacementUsesCapsuleScreenInsteadOfVirtualDesktop(t *testing.T) {
	screens := []*application.Screen{
		{Bounds: application.Rect{X: 0, Y: 0, Width: 1440, Height: 900}},
		{
			Bounds:   application.Rect{X: 1440, Y: 0, Width: 1920, Height: 1080},
			WorkArea: application.Rect{X: 1440, Y: 0, Width: 1920, Height: 1040},
		},
	}
	overlayBounds, ok := pipOverlayBoundsForCapsule(screens, application.Rect{X: 1840, Y: 900, Width: 380, Height: 96}, true)
	if !ok {
		t.Fatal("pipOverlayBoundsForCapsule() did not find the capsule screen")
	}
	placement, err := pip.Place(pip.ConfigFromPreset(pip.PresetBottomLeft), pip.Size{Width: overlayBounds.Width, Height: overlayBounds.Height})
	if err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	windowBounds := application.Rect{
		X:      overlayBounds.X + placement.Rect.X - pipOverlayPadding,
		Y:      overlayBounds.Y + placement.Rect.Y - pipOverlayPadding,
		Width:  placement.Rect.Width + pipOverlayPadding*2,
		Height: placement.Rect.Height + pipOverlayPadding*2,
	}
	if windowBounds.X < overlayBounds.X {
		t.Fatalf("bottom-left PIP window x = %d, want inside capsule screen starting at %d", windowBounds.X, overlayBounds.X)
	}
	if windowBounds.X >= overlayBounds.X+overlayBounds.Width/2 {
		t.Fatalf("bottom-left PIP window x = %d, want left side of capsule screen %#v", windowBounds.X, overlayBounds)
	}
}

type stubPIPMediaProvider struct{}

func (stubPIPMediaProvider) ListMediaDevices() (devices.MediaInventory, error) {
	return devices.MediaInventory{
		Cameras: []devices.MediaDevice{
			{
				ID:              "camera:avfoundation:facetime",
				Type:            devices.DeviceCamera,
				Name:            "FaceTime HD Camera",
				NativeID:        "0",
				Available:       true,
				Capability:      devices.CapabilityEnumerated,
				SidecarEligible: true,
			},
		},
	}, nil
}
