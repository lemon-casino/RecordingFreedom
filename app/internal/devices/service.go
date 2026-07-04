package devices

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise"
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
		return addVirtualCaptureSources(fallbackSources(reason))
	}
	return addVirtualCaptureSources(normalizeSources(sources))
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

func addVirtualCaptureSources(sources []CaptureSource) []CaptureSource {
	displays := make([]CaptureSource, 0, len(sources))
	for _, source := range sources {
		if source.Type == SourceScreen {
			displays = append(displays, source)
		}
	}

	next := make([]CaptureSource, 0, len(sources)+2)
	if len(displays) > 1 {
		next = append(next, allScreensSource(displays))
	}
	next = append(next, sources...)
	next = append(next, regionSelectorSource(displays))
	return next
}

func allScreensSource(displays []CaptureSource) CaptureSource {
	x, y, width, height := virtualBounds(displays)
	source := CaptureSource{
		ID:                "all-screens:virtual-desktop",
		Type:              SourceAllScreens,
		Name:              "All Screens",
		Subtitle:          fmt.Sprintf("%d displays · %d x %d virtual desktop", len(displays), width, height),
		X:                 x,
		Y:                 y,
		Width:             width,
		Height:            height,
		Capability:        CapabilityNativeQueued,
		UnavailableReason: "multi-display composition is queued behind the native video writer",
	}
	if runtime.GOOS == "windows" {
		source.Available = true
		source.Capability = CapabilityEnumerated
		source.UnavailableReason = ""
	}
	return source
}

func regionSelectorSource(displays []CaptureSource) CaptureSource {
	x, y, width, height := virtualBounds(displays)
	if width == 0 || height == 0 {
		width = 1
		height = 1
	}
	return CaptureSource{
		ID:                "region:custom",
		Type:              SourceRegion,
		Name:              "Custom Region",
		Subtitle:          "Drag-select a recording rectangle on the virtual desktop",
		X:                 x,
		Y:                 y,
		Width:             width,
		Height:            height,
		Capability:        CapabilityNativeQueued,
		UnavailableReason: "region selector overlay is available; native region crop writer is queued",
	}
}

func virtualBounds(displays []CaptureSource) (int, int, int, int) {
	if len(displays) == 0 {
		return 0, 0, 0, 0
	}
	minX := displays[0].X
	minY := displays[0].Y
	maxX := displays[0].X + displays[0].Width
	maxY := displays[0].Y + displays[0].Height
	for _, display := range displays[1:] {
		if display.X < minX {
			minX = display.X
		}
		if display.Y < minY {
			minY = display.Y
		}
		if display.X+display.Width > maxX {
			maxX = display.X + display.Width
		}
		if display.Y+display.Height > maxY {
			maxY = display.Y + display.Height
		}
	}
	return minX, minY, maxX - minX, maxY - minY
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
	if !inventory.Enhancement.Available {
		inventory.Enhancement.UnavailableReason = reason
	}
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
	if rnnoise.Available() {
		return AudioEnhancement{
			Engine:     "rnnoise",
			AppliesTo:  "microphone-only",
			Available:  true,
			Capability: CapabilityEnumerated,
		}
	}
	if strings.TrimSpace(reason) == "" {
		reason = "RNNoise requires a packaged native module under tools/ and a build with the rnnoise_dynamic tag; the current build cannot create the suppressor."
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
		return "Windows source enumeration is implemented; FFmpeg Desktop Duplication is used for screen, all-screen, and region video, with FFmpeg HWND capture reserved for locked windows."
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
			return "CoreAudio microphone input device enumeration is implemented."
		case "windows":
			return "WASAPI microphone endpoint enumeration is implemented."
		case "linux":
			return "PipeWire microphone source enumeration is queued for the Linux backend."
		}
	case DeviceCamera:
		switch platform {
		case "darwin":
			return "AVFoundation camera enumeration is queued for the macOS sidecar backend."
		case "windows":
			return "DirectShow camera enumeration uses the bundled FFmpeg tool; Windows camera sidecar recording writes package-local webcam.mp4 when FFmpeg is available."
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
	case SourceAllScreens:
		return "All Screens"
	case SourceRegion:
		return "Custom Region"
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
