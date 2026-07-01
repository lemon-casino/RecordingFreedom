package devices

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise"
)

func TestListSourcesReturnsUsableContract(t *testing.T) {
	sources := NewService().ListSources()
	if len(sources) == 0 {
		t.Fatal("ListSources() returned no sources")
	}

	seenScreen := false
	for _, source := range sources {
		if source.ID == "" {
			t.Fatalf("source has empty ID: %#v", source)
		}
		if source.Name == "" {
			t.Fatalf("source has empty name: %#v", source)
		}
		if source.Capability == "" {
			t.Fatalf("source has empty capability: %#v", source)
		}
		switch source.Type {
		case SourceScreen, SourceAllScreens:
			seenScreen = true
		case SourceRegion, SourceWindow, SourceApplication:
		default:
			t.Fatalf("source has unsupported type %q: %#v", source.Type, source)
		}
		if !source.Available && source.UnavailableReason == "" {
			t.Fatalf("unavailable source must include a reason: %#v", source)
		}
	}

	if !seenScreen {
		t.Fatalf("ListSources() must include at least one screen-class source: %#v", sources)
	}
}

func TestNormalizeSourcesFillsStableDefaults(t *testing.T) {
	sources := normalizeSources([]CaptureSource{
		{Type: SourceWindow},
		{Type: SourceApplication, ID: "application:test", Name: "Test App", Capability: CapabilityUnavailable, UnavailableReason: "not ready"},
	})

	if got := sources[0].ID; got != "window:1" {
		t.Fatalf("default source ID = %q, want window:1", got)
	}
	if got := sources[0].Name; got != "Window" {
		t.Fatalf("default source name = %q, want Window", got)
	}
	if !sources[0].Available {
		t.Fatalf("enumerated source should default to available: %#v", sources[0])
	}
	if sources[1].Available {
		t.Fatalf("unavailable source should not be marked available: %#v", sources[1])
	}
}

func TestListSourcesAddsQueuedRegionSelectorContract(t *testing.T) {
	sources := addVirtualCaptureSources(normalizeSources([]CaptureSource{
		{Type: SourceScreen, ID: "screen:left", Name: "Left", X: -1920, Y: 0, Width: 1920, Height: 1080},
		{Type: SourceScreen, ID: "screen:right", Name: "Right", X: 0, Y: 0, Width: 2560, Height: 1440},
	}))

	var allScreens *CaptureSource
	var region *CaptureSource
	for index := range sources {
		switch sources[index].Type {
		case SourceAllScreens:
			allScreens = &sources[index]
		case SourceRegion:
			region = &sources[index]
		}
	}
	if allScreens == nil {
		t.Fatalf("all-screens virtual source missing: %#v", sources)
	}
	if allScreens.X != -1920 || allScreens.Y != 0 || allScreens.Width != 4480 || allScreens.Height != 1440 {
		t.Fatalf("all-screens bounds = (%d,%d %dx%d), want virtual desktop bounds", allScreens.X, allScreens.Y, allScreens.Width, allScreens.Height)
	}
	if runtime.GOOS == "windows" {
		if !allScreens.Available || allScreens.Capability != CapabilityEnumerated {
			t.Fatalf("all-screens source = %#v, want available Windows virtual desktop source", allScreens)
		}
	} else if allScreens.Available || allScreens.Capability != CapabilityNativeQueued {
		t.Fatalf("all-screens source = %#v, want queued until multi-display writer lands", allScreens)
	}
	if region == nil {
		t.Fatalf("region selector source missing: %#v", sources)
	}
	if region.Available || region.Capability != CapabilityNativeQueued || region.UnavailableReason == "" {
		t.Fatalf("region selector = %#v, want queued with reason", region)
	}
}

func TestListMediaDevicesReturnsSelectionContract(t *testing.T) {
	inventory := NewService().ListMediaDevices()
	assertMediaDevices(t, inventory.SystemAudio, DeviceSystemAudio)
	assertMediaDevices(t, inventory.Microphones, DeviceMicrophone)
	assertMediaDevices(t, inventory.Cameras, DeviceCamera)

	if inventory.Enhancement.Engine != "rnnoise" {
		t.Fatalf("enhancement engine = %q, want rnnoise", inventory.Enhancement.Engine)
	}
	if inventory.Enhancement.AppliesTo != "microphone-only" {
		t.Fatalf("enhancement appliesTo = %q, want microphone-only", inventory.Enhancement.AppliesTo)
	}
	if inventory.Enhancement.Capability == "" {
		t.Fatalf("enhancement capability is empty: %#v", inventory.Enhancement)
	}
}

func TestNormalizeMediaInventoryAddsFallbackDevices(t *testing.T) {
	inventory := normalizeMediaInventory(MediaInventory{})
	assertMediaDevices(t, inventory.SystemAudio, DeviceSystemAudio)
	assertMediaDevices(t, inventory.Microphones, DeviceMicrophone)
	assertMediaDevices(t, inventory.Cameras, DeviceCamera)

	if !inventory.Microphones[0].RNNoiseEligible {
		t.Fatalf("default microphone should be RNNoise eligible: %#v", inventory.Microphones[0])
	}
	if !inventory.Cameras[0].SidecarEligible {
		t.Fatalf("default camera should be sidecar eligible: %#v", inventory.Cameras[0])
	}
}

func TestDarwinDefaultSystemAudioUsesScreenCaptureKitStream(t *testing.T) {
	inventory := defaultMediaInventory("darwin")
	if len(inventory.SystemAudio) == 0 {
		t.Fatal("darwin system audio devices are empty")
	}
	device := inventory.SystemAudio[0]
	if !device.Available || device.Capability != CapabilityEnumerated || device.ID != "system-audio:default" {
		t.Fatalf("darwin system audio device = %#v, want available ScreenCaptureKit default", device)
	}
}

func TestListMediaDevicesUsesInjectedProvider(t *testing.T) {
	service := NewServiceWithMediaProvider(stubMediaProvider{
		inventory: MediaInventory{
			SystemAudio: []MediaDevice{
				{ID: "system-audio:built-in", Name: "Built-in Output", Capability: CapabilityEnumerated},
			},
			Microphones: []MediaDevice{
				{NativeID: "coreaudio:mic-1", Name: "Studio Microphone", Capability: CapabilityEnumerated, RNNoiseEligible: true},
			},
			Cameras: []MediaDevice{
				{ID: "camera:facetime", Name: "FaceTime Camera", Capability: CapabilityEnumerated, SidecarEligible: true},
			},
			Enhancement: AudioEnhancement{
				Engine:     "rnnoise",
				AppliesTo:  "microphone-only",
				Available:  true,
				Capability: CapabilityEnumerated,
			},
		},
	})

	inventory := service.ListMediaDevices()
	if inventory.SystemAudio[0].ID != "system-audio:built-in" || !inventory.SystemAudio[0].Available {
		t.Fatalf("system audio = %#v, want injected available device", inventory.SystemAudio[0])
	}
	if inventory.Microphones[0].ID != "microphone:1" || inventory.Microphones[0].NativeID != "coreaudio:mic-1" || !inventory.Microphones[0].Available {
		t.Fatalf("microphone = %#v, want normalized injected device", inventory.Microphones[0])
	}
	if inventory.Cameras[0].ID != "camera:facetime" || !inventory.Cameras[0].SidecarEligible || !inventory.Cameras[0].Available {
		t.Fatalf("camera = %#v, want injected sidecar device", inventory.Cameras[0])
	}
	if !inventory.Enhancement.Available || inventory.Enhancement.Capability != CapabilityEnumerated {
		t.Fatalf("enhancement = %#v, want injected available rnnoise", inventory.Enhancement)
	}
}

func TestListMediaDevicesFallsBackWhenProviderFails(t *testing.T) {
	service := NewServiceWithMediaProvider(stubMediaProvider{err: errors.New("CoreAudio permission unavailable")})

	inventory := service.ListMediaDevices()
	assertMediaDevices(t, inventory.SystemAudio, DeviceSystemAudio)
	assertMediaDevices(t, inventory.Microphones, DeviceMicrophone)
	assertMediaDevices(t, inventory.Cameras, DeviceCamera)
	if !strings.Contains(inventory.Microphones[0].UnavailableReason, "CoreAudio permission unavailable") {
		t.Fatalf("fallback microphone reason = %q, want provider error", inventory.Microphones[0].UnavailableReason)
	}
	if rnnoise.Available() {
		if !inventory.Enhancement.Available || inventory.Enhancement.Capability != CapabilityEnumerated {
			t.Fatalf("fallback enhancement = %#v, want available native rnnoise", inventory.Enhancement)
		}
	} else if inventory.Enhancement.Available || inventory.Enhancement.Capability != CapabilityNativeQueued {
		t.Fatalf("fallback enhancement = %#v, want queued unavailable", inventory.Enhancement)
	}
}

func assertMediaDevices(t *testing.T, devices []MediaDevice, deviceType MediaDeviceType) {
	t.Helper()
	if len(devices) == 0 {
		t.Fatalf("%s devices are empty", deviceType)
	}
	for _, device := range devices {
		if device.ID == "" {
			t.Fatalf("%s device has empty ID: %#v", deviceType, device)
		}
		if device.Type != deviceType {
			t.Fatalf("device type = %q, want %q: %#v", device.Type, deviceType, device)
		}
		if device.Name == "" {
			t.Fatalf("%s device has empty name: %#v", deviceType, device)
		}
		if device.Capability == "" {
			t.Fatalf("%s device has empty capability: %#v", deviceType, device)
		}
		if !device.Available && device.UnavailableReason == "" {
			t.Fatalf("unavailable %s device must include a reason: %#v", deviceType, device)
		}
	}
}

type stubMediaProvider struct {
	inventory MediaInventory
	err       error
}

func (p stubMediaProvider) ListMediaDevices() (MediaInventory, error) {
	return p.inventory, p.err
}
