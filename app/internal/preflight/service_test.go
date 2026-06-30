package preflight

import (
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
)

func TestEvaluateWarnsForQueuedNativeCapabilitiesOnMockBackend(t *testing.T) {
	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Audio: recording.AudioRequest{
			System:           true,
			Microphone:       true,
			NoiseSuppression: true,
		},
	}, Inputs{
		Backend:      "mock-package",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        queuedMediaInventory(),
		Capabilities: queuedCapabilities(),
	})
	if summary.Status != StatusWarning {
		t.Fatalf("preflight status = %q, want warning: %#v", summary.Status, summary.Checks)
	}
	if summary.NormalizedRequest.Audio.SystemDeviceID != "system-audio:default" {
		t.Fatalf("normalized system device = %q", summary.NormalizedRequest.Audio.SystemDeviceID)
	}
	if !hasCheck(summary.Checks, "mock-backend", StatusWarning) {
		t.Fatalf("mock backend warning missing: %#v", summary.Checks)
	}
}

func TestEvaluateBlocksMissingSource(t *testing.T) {
	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:missing",
		SourceType: recording.SourceScreen,
	}, Inputs{
		Backend:      "mock-package",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        queuedMediaInventory(),
		Capabilities: queuedCapabilities(),
	})
	if summary.Status != StatusBlocked {
		t.Fatalf("preflight status = %q, want blocked", summary.Status)
	}
	if !hasCheck(summary.Checks, "source", StatusBlocked) {
		t.Fatalf("source blocked check missing: %#v", summary.Checks)
	}
}

func TestEvaluateBlocksQueuedCapabilitiesForNativeBackend(t *testing.T) {
	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Audio:      recording.AudioRequest{System: true},
	}, Inputs{
		Backend:      "screencapturekit",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        queuedMediaInventory(),
		Capabilities: queuedCapabilities(),
	})
	if summary.Status != StatusBlocked {
		t.Fatalf("preflight status = %q, want blocked: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "source-backend", StatusBlocked) {
		t.Fatalf("native source backend block missing: %#v", summary.Checks)
	}
}

func TestEvaluateReadyForAvailableNativeBackend(t *testing.T) {
	media := queuedMediaInventory()
	media.SystemAudio[0].Available = true
	media.SystemAudio[0].Capability = devices.CapabilityEnumerated
	media.SystemAudio[0].UnavailableReason = ""

	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Audio:      recording.AudioRequest{System: true},
	}, Inputs{
		Backend:      "screencapturekit",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        media,
		Capabilities: availableCapabilities(),
	})
	if summary.Status != StatusReady {
		t.Fatalf("preflight status = %q, want ready: %#v", summary.Status, summary.Checks)
	}
}

func TestEvaluateBlocksUnwritableStorage(t *testing.T) {
	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
	}, Inputs{
		Backend:      "mock-package",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        queuedMediaInventory(),
		Capabilities: queuedCapabilities(),
		Storage: appdata.StorageStatus{
			VideoDir: "/blocked/data/video",
			Writable: false,
			Status:   appdata.StorageStatusBlocked,
			Reason:   "data/video is not writable",
		},
	})
	if summary.Status != StatusBlocked {
		t.Fatalf("preflight status = %q, want blocked: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "storage", StatusBlocked) {
		t.Fatalf("storage blocked check missing: %#v", summary.Checks)
	}
}

func TestEvaluateWarnsForLowStorage(t *testing.T) {
	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
	}, Inputs{
		Backend:      "mock-package",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        queuedMediaInventory(),
		Capabilities: queuedCapabilities(),
		Storage: appdata.StorageStatus{
			VideoDir:       "/low/data/video",
			Writable:       true,
			FreeSpaceKnown: true,
			AvailableBytes: 512,
			Status:         appdata.StorageStatusWarning,
			Reason:         "available space is below the recommended threshold",
		},
	})
	if summary.Status != StatusWarning {
		t.Fatalf("preflight status = %q, want warning: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "storage", StatusWarning) {
		t.Fatalf("storage warning check missing: %#v", summary.Checks)
	}
}

func availableSource(id string, sourceType devices.CaptureSourceType) devices.CaptureSource {
	return devices.CaptureSource{
		ID:         id,
		Type:       sourceType,
		Name:       "Source",
		Available:  true,
		Capability: devices.CapabilityEnumerated,
	}
}

func queuedMediaInventory() devices.MediaInventory {
	return devices.MediaInventory{
		SystemAudio: []devices.MediaDevice{queuedDevice("system-audio:default", devices.DeviceSystemAudio)},
		Microphones: []devices.MediaDevice{queuedDevice("microphone:default", devices.DeviceMicrophone)},
		Cameras:     []devices.MediaDevice{queuedDevice("camera:default", devices.DeviceCamera)},
		Enhancement: devices.AudioEnhancement{
			Engine:            "rnnoise",
			AppliesTo:         "microphone-only",
			Available:         false,
			Capability:        devices.CapabilityNativeQueued,
			UnavailableReason: "native RNNoise queued",
		},
	}
}

func queuedDevice(id string, deviceType devices.MediaDeviceType) devices.MediaDevice {
	device := devices.MediaDevice{
		ID:                id,
		Type:              deviceType,
		Name:              id,
		Available:         false,
		Capability:        devices.CapabilityNativeQueued,
		UnavailableReason: "native backend queued",
	}
	if deviceType == devices.DeviceMicrophone {
		device.RNNoiseEligible = true
	}
	if deviceType == devices.DeviceCamera {
		device.SidecarEligible = true
	}
	return device
}

func queuedCapabilities() capture.Capabilities {
	return capture.Capabilities{
		ScreenRecording:       queuedCapability("screen-recording"),
		WindowRecording:       queuedCapability("window-recording"),
		ApplicationRecording:  queuedCapability("application-recording"),
		SystemAudio:           queuedCapability("system-audio"),
		Microphone:            queuedCapability("microphone"),
		MicrophoneEnhancement: queuedCapability("microphone-enhancement"),
		CameraSidecar:         queuedCapability("camera-sidecar"),
		PIPExport:             queuedCapability("pip-export"),
	}
}

func availableCapabilities() capture.Capabilities {
	return capture.Capabilities{
		ScreenRecording:       availableCapability("screen-recording"),
		WindowRecording:       availableCapability("window-recording"),
		ApplicationRecording:  availableCapability("application-recording"),
		SystemAudio:           availableCapability("system-audio"),
		Microphone:            availableCapability("microphone"),
		MicrophoneEnhancement: availableCapability("microphone-enhancement"),
		CameraSidecar:         availableCapability("camera-sidecar"),
		PIPExport:             availableCapability("pip-export"),
	}
}

func queuedCapability(id string) capture.Capability {
	return capture.Capability{ID: id, Label: id, Status: capture.StatusQueued, Reason: "queued"}
}

func availableCapability(id string) capture.Capability {
	return capture.Capability{ID: id, Label: id, Status: capture.StatusAvailable, Reason: "ready"}
}

func hasCheck(checks []Check, id string, status Status) bool {
	for _, check := range checks {
		if check.ID == id && check.Status == status {
			return true
		}
	}
	return false
}
