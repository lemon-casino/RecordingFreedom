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

func TestEvaluateBlocksQueuedVirtualSourcesForNativeBackend(t *testing.T) {
	tests := []struct {
		name       string
		sourceID   string
		sourceType devices.CaptureSourceType
	}{
		{name: "all screens composition", sourceID: "all-screens:virtual-desktop", sourceType: devices.SourceAllScreens},
		{name: "region crop", sourceID: "region:custom", sourceType: devices.SourceRegion},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			summary := NewService().Evaluate(recording.StartRequest{
				SourceID:   test.sourceID,
				SourceType: test.sourceType,
			}, Inputs{
				Backend: "screencapturekit",
				Sources: []devices.CaptureSource{{
					ID:                test.sourceID,
					Type:              test.sourceType,
					Name:              test.name,
					Capability:        devices.CapabilityNativeQueued,
					UnavailableReason: "native writer is queued",
				}},
				Media:        queuedMediaInventory(),
				Capabilities: availableCapabilities(),
			})
			if summary.Status != StatusBlocked {
				t.Fatalf("preflight status = %q, want blocked: %#v", summary.Status, summary.Checks)
			}
			if !hasCheck(summary.Checks, "source", StatusBlocked) {
				t.Fatalf("queued virtual source block missing: %#v", summary.Checks)
			}
		})
	}
}

func TestEvaluateAllowsSelectedRegionBoundToDisplayWhenScreenBackendIsAvailable(t *testing.T) {
	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "region:custom",
		SourceType: recording.SourceRegion,
		SourceGeometry: &recording.SourceGeometry{
			X:        120,
			Y:        80,
			Width:    1280,
			Height:   720,
			NativeID: "cgdisplay:42",
		},
	}, Inputs{
		Backend: "screencapturekit",
		Sources: []devices.CaptureSource{{
			ID:                "region:custom",
			Type:              devices.SourceRegion,
			Name:              "Custom Region",
			Capability:        devices.CapabilityNativeQueued,
			UnavailableReason: "region selector source waits for user geometry",
		}},
		Media:        queuedMediaInventory(),
		Capabilities: availableCapabilities(),
	})
	if summary.Status != StatusReady {
		t.Fatalf("preflight status = %q, want ready for selected single-display region: %#v", summary.Status, summary.Checks)
	}
	if hasCheck(summary.Checks, "source", StatusBlocked) {
		t.Fatalf("selected region source should not be blocked by queued selector shell: %#v", summary.Checks)
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

func TestEvaluateBlocksQueuedWindowCapabilityForNativeBackend(t *testing.T) {
	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "window:1a2b",
		SourceType: recording.SourceWindow,
	}, Inputs{
		Backend:      "windows-graphics-capture",
		Sources:      []devices.CaptureSource{availableSource("window:1a2b", devices.SourceWindow)},
		Media:        queuedMediaInventory(),
		Capabilities: queuedCapabilities(),
	})
	if summary.Status != StatusBlocked {
		t.Fatalf("preflight status = %q, want blocked: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "source-backend", StatusBlocked) {
		t.Fatalf("window backend block missing: %#v", summary.Checks)
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

func TestEvaluateAllowsCameraSidecarWhenPIPExportIsQueued(t *testing.T) {
	media := queuedMediaInventory()
	media.Cameras = []devices.MediaDevice{{
		ID:              "camera:dshow:integrated-camera",
		Type:            devices.DeviceCamera,
		Name:            "Integrated Camera",
		NativeID:        "Integrated Camera",
		Available:       true,
		Capability:      devices.CapabilityEnumerated,
		SidecarEligible: true,
	}}
	capabilities := availableCapabilities()
	capabilities.PIPExport = queuedCapability("pip-export")

	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Camera: recording.CameraRequest{
			Enabled:   true,
			DeviceID:  "camera:dshow:integrated-camera",
			PIPPreset: "bottom-right",
		},
	}, Inputs{
		Backend:      "ffmpeg-desktop-capture",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        media,
		Capabilities: capabilities,
	})
	if summary.Status != StatusWarning {
		t.Fatalf("preflight status = %q, want warning because PIP export is queued but recording can start: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "camera-sidecar", StatusReady) || !hasCheck(summary.Checks, "pip-export", StatusWarning) {
		t.Fatalf("camera/PIP checks = %#v, want camera ready and PIP warning", summary.Checks)
	}
}

func TestEvaluateBlocksCameraWithoutNativeID(t *testing.T) {
	media := queuedMediaInventory()
	media.Cameras = []devices.MediaDevice{{
		ID:              "camera:dshow:missing-native-id",
		Type:            devices.DeviceCamera,
		Name:            "Missing Native ID",
		Available:       true,
		Capability:      devices.CapabilityEnumerated,
		SidecarEligible: true,
	}}

	summary := NewService().Evaluate(recording.StartRequest{
		SourceID:   "screen:primary",
		SourceType: recording.SourceScreen,
		Camera: recording.CameraRequest{
			Enabled:   true,
			DeviceID:  "camera:dshow:missing-native-id",
			PIPPreset: "off",
		},
	}, Inputs{
		Backend:      "ffmpeg-desktop-capture",
		Sources:      []devices.CaptureSource{availableSource("screen:primary", devices.SourceScreen)},
		Media:        media,
		Capabilities: availableCapabilities(),
	})
	if summary.Status != StatusBlocked {
		t.Fatalf("preflight status = %q, want blocked without native camera id: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "camera-native-id", StatusBlocked) {
		t.Fatalf("camera native id block missing: %#v", summary.Checks)
	}
}

func TestEvaluateAudioOnlyReadyForAvailableMicrophone(t *testing.T) {
	media := queuedMediaInventory()
	media.Microphones[0].Available = true
	media.Microphones[0].Capability = devices.CapabilityEnumerated
	media.Microphones[0].UnavailableReason = ""

	summary := NewService().EvaluateAudioOnly(recording.AudioOnlyRequest{
		Audio: recording.AudioRequest{Microphone: true},
	}, Inputs{
		Backend:      recording.BackendAudioOnlyNative,
		Media:        media,
		Capabilities: availableCapabilities(),
		Storage: appdata.StorageStatus{
			Writable: true,
			Status:   appdata.StorageStatusReady,
			Reason:   "ready",
		},
	})
	if summary.Status != StatusReady {
		t.Fatalf("audio-only preflight status = %q, want ready: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "microphone", StatusReady) || !hasCheck(summary.Checks, "recording-backend", StatusReady) {
		t.Fatalf("audio-only ready checks missing: %#v", summary.Checks)
	}
}

func TestEvaluateAudioOnlyBlocksMacOSSystemAudioMuxCapability(t *testing.T) {
	media := queuedMediaInventory()
	media.SystemAudio[0].Available = true
	media.SystemAudio[0].Capability = devices.CapabilityEnumerated
	media.SystemAudio[0].UnavailableReason = ""
	capabilities := availableCapabilities()
	capabilities.Platform = "darwin"

	summary := NewService().EvaluateAudioOnly(recording.AudioOnlyRequest{
		Audio: recording.AudioRequest{System: true},
	}, Inputs{
		Backend:      recording.BackendAudioOnlyNative,
		Media:        media,
		Capabilities: capabilities,
		Storage: appdata.StorageStatus{
			Writable: true,
			Status:   appdata.StorageStatusReady,
			Reason:   "ready",
		},
	})
	if summary.Status != StatusBlocked {
		t.Fatalf("audio-only macOS system preflight status = %q, want blocked: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "system-audio", StatusBlocked) {
		t.Fatalf("audio-only macOS system audio block missing: %#v", summary.Checks)
	}
}

func TestEvaluateAudioOnlyAllowsMacOSMicrophoneWithCoreAudioCapture(t *testing.T) {
	media := queuedMediaInventory()
	media.Microphones[0].Available = true
	media.Microphones[0].Capability = devices.CapabilityEnumerated
	media.Microphones[0].UnavailableReason = ""
	capabilities := availableCapabilities()
	capabilities.Platform = "darwin"

	summary := NewService().EvaluateAudioOnly(recording.AudioOnlyRequest{
		Audio: recording.AudioRequest{Microphone: true},
	}, Inputs{
		Backend:      recording.BackendAudioOnlyNative,
		Media:        media,
		Capabilities: capabilities,
		Storage: appdata.StorageStatus{
			Writable: true,
			Status:   appdata.StorageStatusReady,
			Reason:   "ready",
		},
	})
	if summary.Status != StatusReady {
		t.Fatalf("audio-only macOS microphone preflight status = %q, want ready: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "microphone", StatusReady) {
		t.Fatalf("audio-only macOS microphone ready check missing: %#v", summary.Checks)
	}
}

func TestEvaluateAudioOnlyBlocksRNNoiseWhenEnhancementQueued(t *testing.T) {
	media := queuedMediaInventory()
	media.Microphones[0].Available = true
	media.Microphones[0].Capability = devices.CapabilityEnumerated
	media.Microphones[0].UnavailableReason = ""

	summary := NewService().EvaluateAudioOnly(recording.AudioOnlyRequest{
		Audio: recording.AudioRequest{Microphone: true, NoiseSuppression: true},
	}, Inputs{
		Backend:      recording.BackendAudioOnlyNative,
		Media:        media,
		Capabilities: queuedCapabilities(),
		Storage: appdata.StorageStatus{
			Writable: true,
			Status:   appdata.StorageStatusReady,
			Reason:   "ready",
		},
	})
	if summary.Status != StatusBlocked {
		t.Fatalf("audio-only preflight status = %q, want blocked: %#v", summary.Status, summary.Checks)
	}
	if !hasCheck(summary.Checks, "microphone-enhancement", StatusBlocked) {
		t.Fatalf("audio-only RNNoise block missing: %#v", summary.Checks)
	}
}

func TestEvaluateAudioOnlyRejectsNoStreams(t *testing.T) {
	summary := NewService().EvaluateAudioOnly(recording.AudioOnlyRequest{}, Inputs{Backend: recording.BackendAudioOnlyNative})
	if summary.Status != StatusBlocked {
		t.Fatalf("audio-only preflight status = %q, want blocked", summary.Status)
	}
	if !hasCheck(summary.Checks, "request", StatusBlocked) {
		t.Fatalf("audio-only request block missing: %#v", summary.Checks)
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
		Platform:              "windows",
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
		Platform:              "windows",
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
