package main

import (
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
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
