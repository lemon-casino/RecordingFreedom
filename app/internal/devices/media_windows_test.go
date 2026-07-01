//go:build windows

package devices

import (
	"strings"
	"testing"
)

func TestWindowsMediaDevicesKeepConcreteWasapiIDForDefaultEndpoint(t *testing.T) {
	devices := windowsMediaDevices(DeviceMicrophone, []windowsAudioEndpoint{
		{
			id:        `{0.0.1.00000000}.{11111111-2222-3333-4444-555555555555}`,
			name:      "Microphone Array (Realtek(R) Audio)",
			isDefault: true,
		},
	})
	if len(devices) != 1 {
		t.Fatalf("devices len = %d, want 1", len(devices))
	}
	device := devices[0]
	if device.ID == "microphone:default" {
		t.Fatalf("default microphone lost concrete WASAPI endpoint id: %#v", device)
	}
	if !strings.HasPrefix(device.ID, "microphone:wasapi:") {
		t.Fatalf("device id = %q, want concrete microphone:wasapi endpoint", device.ID)
	}
	if device.NativeID == "" || !strings.Contains(device.ID, device.NativeID) {
		t.Fatalf("native id = %q, id = %q; want the UI selection id to carry the native endpoint", device.NativeID, device.ID)
	}
	if strings.HasPrefix(device.Name, "Default ") {
		t.Fatalf("device name = %q, want the real Windows endpoint name without replacing it with a generic default label", device.Name)
	}
	if !device.IsDefault {
		t.Fatalf("default flag was not preserved: %#v", device)
	}
	if !device.RNNoiseEligible {
		t.Fatalf("microphone should remain RNNoise eligible: %#v", device)
	}
	if !strings.Contains(device.Subtitle, "Default") {
		t.Fatalf("subtitle = %q, want default marker outside the selectable device name", device.Subtitle)
	}
}
