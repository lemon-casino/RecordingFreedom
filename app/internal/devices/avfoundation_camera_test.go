package devices

import "testing"

func TestParseAVFoundationCameraDevices(t *testing.T) {
	output := `
[AVFoundation indev @ 0x123] AVFoundation video devices:
[AVFoundation indev @ 0x123] [0] FaceTime HD Camera
[AVFoundation indev @ 0x123] [1] OBS Virtual Camera
[AVFoundation indev @ 0x123] [2] Capture screen 0
[AVFoundation indev @ 0x123] AVFoundation audio devices:
[AVFoundation indev @ 0x123] [0] MacBook Pro Microphone
`
	devices := parseAVFoundationCameraDevices(output)
	if len(devices) != 2 {
		t.Fatalf("devices = %#v, want 2 cameras", devices)
	}
	if !devices[0].IsDefault || devices[1].IsDefault {
		t.Fatalf("default flags = %v/%v, want first camera only", devices[0].IsDefault, devices[1].IsDefault)
	}
	if devices[0].ID != "camera:avfoundation:facetime-hd-camera" || devices[0].NativeID != "0" {
		t.Fatalf("first camera = %#v, want stable AVFoundation index id", devices[0])
	}
	if devices[1].ID != "camera:avfoundation:obs-virtual-camera" || devices[1].NativeID != "1" {
		t.Fatalf("second camera = %#v, want OBS camera", devices[1])
	}
}

func TestParseAVFoundationCameraDevicesIgnoresAudioSection(t *testing.T) {
	output := `
[AVFoundation indev @ 0x123] AVFoundation audio devices:
[AVFoundation indev @ 0x123] [0] MacBook Pro Microphone
`
	if devices := parseAVFoundationCameraDevices(output); len(devices) != 0 {
		t.Fatalf("devices = %#v, want no cameras", devices)
	}
}
