package devices

import (
	"fmt"
	"runtime"
	"strings"
)

type MediaDeviceProvider interface {
	ListMediaDevices() (MediaInventory, error)
}

type Service struct {
	mediaProvider MediaDeviceProvider
}

type platformMediaDeviceProvider struct{}

func (platformMediaDeviceProvider) ListMediaDevices() (MediaInventory, error) {
	return listPlatformMediaDevices()
}

func NewService() *Service {
	return NewServiceWithMediaProvider(platformMediaDeviceProvider{})
}

func NewServiceWithMediaProvider(provider MediaDeviceProvider) *Service {
	if provider == nil {
		provider = platformMediaDeviceProvider{}
	}
	return &Service{mediaProvider: provider}
}

func (s *Service) ListSources() []CaptureSource {
	sources, err := listPlatformSources()
	if err != nil || len(sources) == 0 {
		reason := "native source enumeration returned no sources"
		if err != nil {
			reason = err.Error()
		}
		return fallbackSources(reason)
	}
	return normalizeSources(sources)
}

func (s *Service) ListMediaDevices() MediaInventory {
	inventory, err := s.mediaProvider.ListMediaDevices()
	if err != nil {
		inventory = fallbackMediaInventory(runtime.GOOS, fmt.Sprintf("native media device enumeration failed: %v", err))
	}
	return normalizeMediaInventory(inventory)
}

func normalizeSources(sources []CaptureSource) []CaptureSource {
	normalized := make([]CaptureSource, 0, len(sources))
	for index, source := range sources {
		if source.Type == "" {
			continue
		}
		if strings.TrimSpace(source.ID) == "" {
			source.ID = fmt.Sprintf("%s:%d", source.Type, index+1)
		}
		if strings.TrimSpace(source.Name) == "" {
			source.Name = defaultSourceName(source.Type)
		}
		if source.Capability == "" {
			source.Capability = CapabilityEnumerated
		}
		if source.Capability != CapabilityUnavailable && source.UnavailableReason == "" {
			source.Available = true
		}
		normalized = append(normalized, source)
	}
	if len(normalized) == 0 {
		return fallbackSources("source normalization removed every platform source")
	}
	return normalized
}

func normalizeMediaInventory(inventory MediaInventory) MediaInventory {
	inventory.SystemAudio = normalizeMediaDevices(inventory.SystemAudio, DeviceSystemAudio)
	inventory.Microphones = normalizeMediaDevices(inventory.Microphones, DeviceMicrophone)
	inventory.Cameras = normalizeMediaDevices(inventory.Cameras, DeviceCamera)
	if inventory.Enhancement.Engine == "" {
		inventory.Enhancement = defaultAudioEnhancement("audio enhancement backend is not configured")
	}
	if inventory.Enhancement.Capability == "" {
		inventory.Enhancement.Capability = CapabilityNativeQueued
	}
	return inventory
}

func normalizeMediaDevices(devices []MediaDevice, deviceType MediaDeviceType) []MediaDevice {
	normalized := make([]MediaDevice, 0, len(devices))
	for index, device := range devices {
		device.Type = deviceType
		if strings.TrimSpace(device.ID) == "" {
			device.ID = fmt.Sprintf("%s:%d", deviceType, index+1)
		}
		if strings.TrimSpace(device.Name) == "" {
			device.Name = defaultMediaDeviceName(deviceType)
		}
		if device.Capability == "" {
			device.Capability = CapabilityNativeQueued
		}
		if device.Capability != CapabilityUnavailable && device.UnavailableReason == "" {
			device.Available = true
		}
		normalized = append(normalized, device)
	}
	if len(normalized) == 0 {
		normalized = append(normalized, defaultMediaDevice(deviceType, "native media device enumeration returned no devices"))
	}
	return normalized
}

func fallbackSources(reason string) []CaptureSource {
	platform := runtime.GOOS
	return []CaptureSource{
		{
			ID:                "screen:native-backend-queued",
			Type:              SourceScreen,
			Name:              "Native Screen Source",
			Subtitle:          platformBackendMessage(platform),
			Available:         false,
			Capability:        CapabilityNativeQueued,
			UnavailableReason: reason,
		},
		{
			ID:                "window:native-backend-queued",
			Type:              SourceWindow,
			Name:              "Native Window Source",
			Subtitle:          "Window enumeration is reserved for the platform capture backend.",
			Available:         false,
			Capability:        CapabilityNativeQueued,
			UnavailableReason: reason,
		},
		{
			ID:                "application:native-backend-queued",
			Type:              SourceApplication,
			Name:              "Native Program Source",
			Subtitle:          "Program grouping is reserved for the platform capture backend.",
			Available:         false,
			Capability:        CapabilityNativeQueued,
			UnavailableReason: reason,
		},
	}
}

func defaultMediaInventory(platform string) MediaInventory {
	return MediaInventory{
		SystemAudio: []MediaDevice{
			defaultMediaDeviceForPlatform(platform, DeviceSystemAudio, mediaBackendMessage(platform, DeviceSystemAudio)),
		},
		Microphones: []MediaDevice{
			defaultMediaDevice(DeviceMicrophone, mediaBackendMessage(platform, DeviceMicrophone)),
		},
		Cameras: []MediaDevice{
			defaultMediaDevice(DeviceCamera, mediaBackendMessage(platform, DeviceCamera)),
		},
		Enhancement: defaultAudioEnhancement("RNNoise native DSP is queued behind microphone capture plumbing."),
	}
}

func fallbackMediaInventory(platform string, reason string) MediaInventory {
	inventory := defaultMediaInventory(platform)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return inventory
	}
	for index := range inventory.SystemAudio {
		inventory.SystemAudio[index].Available = false
		inventory.SystemAudio[index].Capability = CapabilityNativeQueued
		inventory.SystemAudio[index].UnavailableReason = reason
	}
	for index := range inventory.Microphones {
		inventory.Microphones[index].Available = false
		inventory.Microphones[index].Capability = CapabilityNativeQueued
		inventory.Microphones[index].UnavailableReason = reason
	}
	for index := range inventory.Cameras {
		inventory.Cameras[index].Available = false
		inventory.Cameras[index].Capability = CapabilityNativeQueued
		inventory.Cameras[index].UnavailableReason = reason
	}
	inventory.Enhancement.UnavailableReason = reason
	return inventory
}

func defaultMediaDeviceForPlatform(platform string, deviceType MediaDeviceType, reason string) MediaDevice {
	device := defaultMediaDevice(deviceType, reason)
	if platform == "darwin" && deviceType == DeviceSystemAudio {
		device.Available = true
		device.Capability = CapabilityEnumerated
		device.UnavailableReason = ""
		device.Subtitle = "ScreenCaptureKit system audio stream"
	}
	return device
}

func defaultMediaDevice(deviceType MediaDeviceType, reason string) MediaDevice {
	device := MediaDevice{
		ID:                fmt.Sprintf("%s:default", deviceType),
		Type:              deviceType,
		Name:              defaultMediaDeviceName(deviceType),
		Subtitle:          defaultMediaDeviceSubtitle(deviceType),
		NativeID:          "default",
		IsDefault:         true,
		Available:         false,
		Capability:        CapabilityNativeQueued,
		UnavailableReason: reason,
	}
	if deviceType == DeviceMicrophone {
		device.RNNoiseEligible = true
	}
	if deviceType == DeviceCamera {
		device.SidecarEligible = true
	}
	return device
}

func defaultAudioEnhancement(reason string) AudioEnhancement {
	if strings.TrimSpace(reason) == "" {
		reason = "RNNoise native wrapper is implemented for the audio pipeline, but it is not wired into the app recording backend yet."
	}
	return AudioEnhancement{
		Engine:            "rnnoise",
		AppliesTo:         "microphone-only",
		Available:         false,
		Capability:        CapabilityNativeQueued,
		UnavailableReason: reason,
	}
}

func platformBackendMessage(platform string) string {
	switch platform {
	case "darwin":
		return "ScreenCaptureKit source enumeration is the next macOS backend milestone."
	case "windows":
		return "Windows.Graphics.Capture source enumeration is the next Windows backend milestone."
	case "linux":
		return "XDG Desktop Portal and PipeWire source enumeration is the next Linux backend milestone."
	default:
		return "Native source enumeration is not implemented for this platform yet."
	}
}

func mediaBackendMessage(platform string, deviceType MediaDeviceType) string {
	switch deviceType {
	case DeviceSystemAudio:
		switch platform {
		case "darwin":
			return "ScreenCaptureKit system audio uses the default system mix stream."
		case "windows":
			return "WASAPI loopback endpoint enumeration is queued for the Windows backend."
		case "linux":
			return "PipeWire monitor source enumeration is queued for the Linux backend."
		}
	case DeviceMicrophone:
		switch platform {
		case "darwin":
			return "CoreAudio microphone enumeration is queued for the macOS backend."
		case "windows":
			return "WASAPI microphone endpoint enumeration is queued for the Windows backend."
		case "linux":
			return "PipeWire microphone source enumeration is queued for the Linux backend."
		}
	case DeviceCamera:
		switch platform {
		case "darwin":
			return "AVFoundation camera enumeration is queued for the macOS sidecar backend."
		case "windows":
			return "Media Foundation camera enumeration is queued for the Windows sidecar backend."
		case "linux":
			return "PipeWire camera enumeration is queued for the Linux sidecar backend."
		}
	}
	return "Native media device enumeration is not implemented for this platform yet."
}

func defaultSourceName(sourceType CaptureSourceType) string {
	switch sourceType {
	case SourceScreen:
		return "Display"
	case SourceWindow:
		return "Window"
	case SourceApplication:
		return "Program"
	default:
		return "Capture Source"
	}
}

func defaultMediaDeviceName(deviceType MediaDeviceType) string {
	switch deviceType {
	case DeviceSystemAudio:
		return "Default System Audio"
	case DeviceMicrophone:
		return "Default Microphone"
	case DeviceCamera:
		return "Default Camera"
	default:
		return "Default Device"
	}
}

func defaultMediaDeviceSubtitle(deviceType MediaDeviceType) string {
	switch deviceType {
	case DeviceSystemAudio:
		return "System sound capture endpoint"
	case DeviceMicrophone:
		return "Microphone capture endpoint"
	case DeviceCamera:
		return "Camera sidecar endpoint"
	default:
		return "Media device endpoint"
	}
}
