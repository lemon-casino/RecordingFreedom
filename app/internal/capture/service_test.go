package capture

import "testing"

func TestCapabilitiesContract(t *testing.T) {
	capabilities := NewService().Capabilities()
	if capabilities.Platform == "" {
		t.Fatal("platform is empty")
	}
	assertCapability(t, capabilities.SourceEnumeration)
	assertCapability(t, capabilities.ScreenRecording)
	assertCapability(t, capabilities.WindowRecording)
	assertCapability(t, capabilities.ApplicationRecording)
	assertCapability(t, capabilities.SystemAudio)
	assertCapability(t, capabilities.Microphone)
	assertCapability(t, capabilities.MicrophoneEnhancement)
	assertCapability(t, capabilities.CameraSidecar)
	assertCapability(t, capabilities.PIPExport)
	assertCapability(t, capabilities.PackageRecovery)
	if capabilities.PackageRecovery.Status != StatusAvailable {
		t.Fatalf("package recovery status = %q, want %q", capabilities.PackageRecovery.Status, StatusAvailable)
	}
}

func TestDarwinScreenRecordingCapabilityAvailable(t *testing.T) {
	capability := screenRecordingCapability("darwin")
	if capability.Status != StatusAvailable {
		t.Fatalf("darwin screen recording status = %q, want %q", capability.Status, StatusAvailable)
	}
	if capability.Backend != "ScreenCaptureKit" {
		t.Fatalf("darwin screen recording backend = %q, want ScreenCaptureKit", capability.Backend)
	}
}

func TestDarwinWindowRecordingCapabilityAvailable(t *testing.T) {
	capability := windowRecordingCapability("darwin")
	if capability.Status != StatusAvailable {
		t.Fatalf("darwin window recording status = %q, want %q", capability.Status, StatusAvailable)
	}
	if capability.Backend != "ScreenCaptureKit" {
		t.Fatalf("darwin window recording backend = %q, want ScreenCaptureKit", capability.Backend)
	}
}

func TestDarwinProgramRecordingCapabilityQueued(t *testing.T) {
	capability := applicationRecordingCapability("darwin")
	if capability.Status != StatusQueued {
		t.Fatalf("darwin program recording status = %q, want %q", capability.Status, StatusQueued)
	}
	if capability.Backend != "ScreenCaptureKit" {
		t.Fatalf("darwin program recording backend = %q, want ScreenCaptureKit", capability.Backend)
	}
}

func assertCapability(t *testing.T, capability Capability) {
	t.Helper()
	if capability.ID == "" {
		t.Fatalf("capability has empty ID: %#v", capability)
	}
	if capability.Label == "" {
		t.Fatalf("capability has empty label: %#v", capability)
	}
	if capability.Status == "" {
		t.Fatalf("capability has empty status: %#v", capability)
	}
	if capability.Backend == "" {
		t.Fatalf("capability has empty backend: %#v", capability)
	}
	if capability.Permission == "" {
		t.Fatalf("capability has empty permission: %#v", capability)
	}
	if capability.Status != StatusAvailable && capability.Reason == "" {
		t.Fatalf("non-available capability must include a reason: %#v", capability)
	}
}
