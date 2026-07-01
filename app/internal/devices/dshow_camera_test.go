package devices

import "testing"

func TestParseDirectShowCameraDevices(t *testing.T) {
	output := `
[dshow @ 000001] DirectShow video devices (some may be both video and audio devices)
[dshow @ 000001]  "Integrated Camera"
[dshow @ 000001]     Alternative name "@device_pnp_\\?\usb#vid_04f2&pid_b6f1#camera"
[dshow @ 000001]  "USB Capture HDMI"
[dshow @ 000001] DirectShow audio devices
[dshow @ 000001]  "Microphone Array"
`
	devices := parseDirectShowCameraDevices(output)
	if len(devices) != 2 {
		t.Fatalf("devices = %#v, want 2 cameras", devices)
	}
	if devices[0].Type != DeviceCamera || !devices[0].Available || !devices[0].SidecarEligible {
		t.Fatalf("first camera = %#v, want available sidecar camera", devices[0])
	}
	if !devices[0].IsDefault || devices[1].IsDefault {
		t.Fatalf("default flags = %v/%v, want first only", devices[0].IsDefault, devices[1].IsDefault)
	}
	if devices[0].ID != "camera:dshow:device-pnp-usb-vid-04f2-pid-b6f1-camera" {
		t.Fatalf("first camera id = %q", devices[0].ID)
	}
	if devices[0].NativeID != "Integrated Camera" {
		t.Fatalf("native id = %q, want original device name for ffmpeg input", devices[0].NativeID)
	}
	if devices[1].ID != "camera:dshow:usb-capture-hdmi" {
		t.Fatalf("second camera id = %q", devices[1].ID)
	}
}

func TestParseDirectShowCameraDevicesIgnoresAudioSection(t *testing.T) {
	output := `
[dshow @ 000001] DirectShow audio devices
[dshow @ 000001]  "OBS Virtual Audio"
`
	if devices := parseDirectShowCameraDevices(output); len(devices) != 0 {
		t.Fatalf("devices = %#v, want no cameras", devices)
	}
}
