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

func TestParseDirectShowCameraDevicesFromInlineMediaTypeOutput(t *testing.T) {
	output := `
[in#0 @ 000001] "HD Webcam" (video)
[in#0 @ 000001]   Alternative name "@device_pnp_\\?\usb#vid_5986&pid_211c&mi_00#6&4d9c116&0&0000#{65e8773d-8f56-11d0-a3b9-00a0c9223196}\global"
[in#0 @ 000001] "麦克风阵列 (2- 适用于数字麦克风的英特尔® 智音技术)" (audio)
[in#0 @ 000001]   Alternative name "@device_cm_{33D9A762-90C8-11D0-BD43-00A0C911CE86}\wave_{7B4C2AF0-E362-406D-BD2F-5C6112CA59F0}"
Error opening input file dummy.
`
	devices := parseDirectShowCameraDevices(output)
	if len(devices) != 1 {
		t.Fatalf("devices = %#v, want one inline video camera", devices)
	}
	if devices[0].Name != "Default HD Webcam" || devices[0].NativeID != "HD Webcam" {
		t.Fatalf("camera = %#v, want HD Webcam as DirectShow native device", devices[0])
	}
	if devices[0].ID != "camera:dshow:device-pnp-usb-vid-5986-pid-211c-mi-00-6-4d9c116-0-0000-65e8773d-8f56-11d0-a3b9-00a0c9223196-global" {
		t.Fatalf("camera id = %q, want stable alternative-name id", devices[0].ID)
	}
}
