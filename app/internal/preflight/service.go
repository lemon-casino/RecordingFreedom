package preflight

import (
	"fmt"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
)

type Inputs struct {
	Backend      string
	Sources      []devices.CaptureSource
	Media        devices.MediaInventory
	Capabilities capture.Capabilities
	Storage      appdata.StorageStatus
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Evaluate(req recording.StartRequest, inputs Inputs) Summary {
	backend := strings.TrimSpace(inputs.Backend)
	if backend == "" {
		backend = "unknown"
	}
	normalized, err := recording.NormalizeStartRequest(req)
	if err != nil {
		return Summary{
			Status:            StatusBlocked,
			Backend:           backend,
			Message:           "Recording request is invalid.",
			NormalizedRequest: recording.StartRequest{},
			Checks: []Check{
				blocked("request", "Recording Request", err.Error()),
			},
		}
	}

	evaluator := evaluator{backend: backend, mockBackend: isMockBackend(backend)}
	evaluator.checkSource(normalized, inputs.Sources, inputs.Capabilities)
	evaluator.checkAudio(normalized, inputs.Media, inputs.Capabilities)
	evaluator.checkCamera(normalized, inputs.Media, inputs.Capabilities)
	evaluator.checkStorage(inputs.Storage)
	evaluator.checkPackageShell()

	status := aggregate(evaluator.checks)
	return Summary{
		Status:            status,
		Backend:           backend,
		Message:           messageFor(status, evaluator.mockBackend),
		Checks:            evaluator.checks,
		NormalizedRequest: normalized,
	}
}

func (s *Service) EvaluateAudioOnly(req recording.AudioOnlyRequest, inputs Inputs) Summary {
	backend := strings.TrimSpace(inputs.Backend)
	if backend == "" {
		backend = recording.BackendAudioOnlyNative
	}
	normalized, err := recording.NormalizeAudioOnlyRequest(req)
	if err != nil {
		return Summary{
			Status:  StatusBlocked,
			Backend: backend,
			Message: "Audio recording request is invalid.",
			Checks: []Check{
				blocked("request", "Recording Request", err.Error()),
			},
		}
	}

	evaluator := evaluator{backend: backend, mockBackend: isMockBackend(backend)}
	evaluator.checkAudioOnly(normalized, inputs.Media, inputs.Capabilities)
	evaluator.checkStorage(inputs.Storage)
	evaluator.checkPackageShell()

	status := aggregate(evaluator.checks)
	return Summary{
		Status:  status,
		Backend: backend,
		Message: messageFor(status, evaluator.mockBackend),
		Checks:  evaluator.checks,
	}
}

type evaluator struct {
	backend     string
	mockBackend bool
	checks      []Check
}

func (e *evaluator) checkSource(req recording.StartRequest, sources []devices.CaptureSource, capabilities capture.Capabilities) {
	source, ok := findSource(sources, req.SourceID, req.SourceType)
	if !ok {
		e.add(blocked("source", "Capture Source", fmt.Sprintf("source %q (%s) was not returned by DeviceService", req.SourceID, req.SourceType)))
		return
	}
	if !source.Available {
		if !selectedRegionSourceIsBoundToDisplay(req) {
			reason := source.UnavailableReason
			if reason == "" {
				reason = fmt.Sprintf("source capability is %q", source.Capability)
			}
			e.add(e.statusForNative("source", "Capture Source", reason))
		}
	}
	e.add(e.checkCapability("source-backend", sourceCapabilityLabel(req.SourceType), sourceCaptureCapability(req.SourceType, capabilities)))
}

func (e *evaluator) checkAudio(req recording.StartRequest, media devices.MediaInventory, capabilities capture.Capabilities) {
	if req.Audio.System {
		e.checkDevice("system-audio-device", "System Audio Device", req.Audio.SystemDeviceID, media.SystemAudio)
		e.add(e.checkCapability("system-audio", "System Audio", capabilities.SystemAudio))
	}
	if req.Audio.Microphone {
		mic, ok := e.checkDevice("microphone-device", "Microphone Device", req.Audio.MicrophoneID, media.Microphones)
		e.add(e.checkCapability("microphone", "Microphone", capabilities.Microphone))
		if req.Audio.NoiseSuppression {
			if ok && !mic.RNNoiseEligible {
				e.add(e.statusForNative("microphone-rnnoise-device", "Microphone RNNoise", fmt.Sprintf("microphone %q is not marked RNNoise eligible", mic.ID)))
			}
			if !media.Enhancement.Available {
				reason := media.Enhancement.UnavailableReason
				if reason == "" {
					reason = fmt.Sprintf("audio enhancement capability is %q", media.Enhancement.Capability)
				}
				e.add(e.statusForNative("microphone-rnnoise", "Microphone RNNoise", reason))
			}
			e.add(e.checkCapability("microphone-enhancement", "Microphone Enhancement", capabilities.MicrophoneEnhancement))
		}
	}
}

func (e *evaluator) checkAudioOnly(req recording.AudioOnlyRequest, media devices.MediaInventory, capabilities capture.Capabilities) {
	if req.Audio.System {
		e.checkDevice("system-audio-device", "System Audio Device", req.Audio.SystemDeviceID, media.SystemAudio)
		e.add(e.checkCapability("system-audio", "System Audio", audioOnlySystemAudioCapability(capabilities)))
	}
	if req.Audio.Microphone {
		mic, ok := e.checkDevice("microphone-device", "Microphone Device", req.Audio.MicrophoneID, media.Microphones)
		e.add(e.checkCapability("microphone", "Microphone", audioOnlyMicrophoneCapability(capabilities)))
		if req.Audio.NoiseSuppression {
			if ok && !mic.RNNoiseEligible {
				e.add(e.statusForNative("microphone-rnnoise-device", "Microphone RNNoise", fmt.Sprintf("microphone %q is not marked RNNoise eligible", mic.ID)))
			}
			if !media.Enhancement.Available {
				reason := media.Enhancement.UnavailableReason
				if reason == "" {
					reason = fmt.Sprintf("audio enhancement capability is %q", media.Enhancement.Capability)
				}
				e.add(e.statusForNative("microphone-rnnoise", "Microphone RNNoise", reason))
			}
			e.add(e.checkCapability("microphone-enhancement", "Microphone Enhancement", capabilities.MicrophoneEnhancement))
		}
	}
}

func (e *evaluator) checkCamera(req recording.StartRequest, media devices.MediaInventory, capabilities capture.Capabilities) {
	if !req.Camera.Enabled {
		return
	}
	camera, ok := e.checkDevice("camera-device", "Camera Device", req.Camera.DeviceID, media.Cameras)
	e.add(e.checkCapability("camera-sidecar", "Camera Sidecar", capabilities.CameraSidecar))
	if ok && !camera.SidecarEligible {
		e.add(e.statusForNative("camera-sidecar-device", "Camera Sidecar", fmt.Sprintf("camera %q is not marked sidecar eligible", camera.ID)))
	}
	if ok && strings.TrimSpace(camera.NativeID) == "" {
		e.add(e.statusForNative("camera-native-id", "Camera Native ID", fmt.Sprintf("camera %q does not expose a native capture id", camera.ID)))
	}
	if req.Camera.PIPPreset != "off" {
		check := e.checkCapability("pip-export", "PIP Export", capabilities.PIPExport)
		if check.Status == StatusBlocked {
			check.Status = StatusWarning
			check.Reason = "Recording will capture a webcam sidecar now; PIP preview/export remains a later pipeline: " + check.Reason
		}
		e.add(check)
	}
}

func (e *evaluator) checkStorage(storage appdata.StorageStatus) {
	if storage.Status == "" {
		return
	}
	reason := storage.Reason
	if reason == "" {
		reason = fmt.Sprintf("recordings will be written to %s", storage.VideoDir)
	}
	switch storage.Status {
	case appdata.StorageStatusReady:
		e.add(ready("storage", "Recording Storage", reason))
	case appdata.StorageStatusWarning:
		e.add(warning("storage", "Recording Storage", reason))
	case appdata.StorageStatusBlocked:
		e.add(blocked("storage", "Recording Storage", reason))
	default:
		e.add(warning("storage", "Recording Storage", fmt.Sprintf("storage status %q is unknown: %s", storage.Status, reason)))
	}
}

func (e *evaluator) checkPackageShell() {
	if e.mockBackend {
		e.add(warning("mock-backend", "Recording Backend", "mock-package writes a verifiable .rfrec package but does not capture real media"))
		return
	}
	e.add(ready("recording-backend", "Recording Backend", fmt.Sprintf("using backend %q", e.backend)))
}

func (e *evaluator) checkDevice(id string, label string, deviceID string, inventory []devices.MediaDevice) (devices.MediaDevice, bool) {
	device, ok := findMediaDevice(inventory, deviceID)
	if !ok {
		check := blocked(id, label, fmt.Sprintf("device %q was not returned by DeviceService", deviceID))
		e.add(check)
		return devices.MediaDevice{}, false
	}
	if !device.Available {
		reason := device.UnavailableReason
		if reason == "" {
			reason = fmt.Sprintf("device capability is %q", device.Capability)
		}
		e.add(e.statusForNative(id, label, reason))
	}
	return device, true
}

func (e *evaluator) checkCapability(id string, label string, capability capture.Capability) Check {
	switch capability.Status {
	case capture.StatusAvailable:
		return ready(id, label, capability.Reason)
	case capture.StatusQueued:
		return e.statusForNative(id, label, capability.Reason)
	case capture.StatusBlocked, capture.StatusUnsupported:
		if e.mockBackend {
			return warning(id, label, capability.Reason)
		}
		return blocked(id, label, capability.Reason)
	default:
		if e.mockBackend {
			return warning(id, label, fmt.Sprintf("capability %q has unknown status %q", capability.ID, capability.Status))
		}
		return blocked(id, label, fmt.Sprintf("capability %q has unknown status %q", capability.ID, capability.Status))
	}
}

func (e *evaluator) statusForNative(id string, label string, reason string) Check {
	if e.mockBackend {
		return warning(id, label, reason)
	}
	return blocked(id, label, reason)
}

func (e *evaluator) add(check Check) {
	e.checks = append(e.checks, check)
}

func findSource(sources []devices.CaptureSource, id string, sourceType devices.CaptureSourceType) (devices.CaptureSource, bool) {
	for _, source := range sources {
		if source.ID == id && source.Type == sourceType {
			return source, true
		}
	}
	return devices.CaptureSource{}, false
}

func findMediaDevice(inventory []devices.MediaDevice, id string) (devices.MediaDevice, bool) {
	for _, device := range inventory {
		if device.ID == id {
			return device, true
		}
	}
	return devices.MediaDevice{}, false
}

func sourceCaptureCapability(sourceType devices.CaptureSourceType, capabilities capture.Capabilities) capture.Capability {
	switch sourceType {
	case devices.SourceScreen, devices.SourceAllScreens, devices.SourceRegion:
		return capabilities.ScreenRecording
	case devices.SourceWindow:
		return capabilities.WindowRecording
	case devices.SourceApplication:
		return capabilities.ApplicationRecording
	default:
		return capture.Capability{ID: "unknown-source", Label: "Unknown Source", Status: capture.StatusUnsupported, Reason: fmt.Sprintf("unsupported source type %q", sourceType)}
	}
}

func sourceCapabilityLabel(sourceType devices.CaptureSourceType) string {
	switch sourceType {
	case devices.SourceScreen:
		return "Screen Recording"
	case devices.SourceAllScreens:
		return "All-Screens Recording"
	case devices.SourceRegion:
		return "Region Recording"
	case devices.SourceWindow:
		return "Window Recording"
	case devices.SourceApplication:
		return "Program Recording"
	default:
		return "Source Recording"
	}
}

func audioOnlySystemAudioCapability(capabilities capture.Capabilities) capture.Capability {
	switch capabilities.Platform {
	case "windows":
		return capabilities.SystemAudio
	case "darwin":
		return capture.Capability{
			ID:         "system-audio",
			Label:      "System Audio",
			Status:     capture.StatusQueued,
			Backend:    "audio-only-native",
			Permission: capture.PermissionScreenRecording,
			Reason:     "audio-only system audio is not implemented on macOS yet; ScreenCaptureKit system audio currently muxes into screen.mp4 during video recording only",
		}
	case "linux":
		return capture.Capability{
			ID:         "system-audio",
			Label:      "System Audio",
			Status:     capture.StatusQueued,
			Backend:    "audio-only-native",
			Permission: capture.PermissionUnknown,
			Reason:     "audio-only PipeWire/PulseAudio capture is queued",
		}
	default:
		return capture.Capability{
			ID:         "system-audio",
			Label:      "System Audio",
			Status:     capture.StatusUnsupported,
			Backend:    "audio-only-native",
			Permission: capture.PermissionNotRequired,
			Reason:     fmt.Sprintf("audio-only system audio is not implemented for platform %q", capabilities.Platform),
		}
	}
}

func audioOnlyMicrophoneCapability(capabilities capture.Capabilities) capture.Capability {
	switch capabilities.Platform {
	case "windows", "darwin":
		return capabilities.Microphone
	case "linux":
		return capture.Capability{
			ID:         "microphone",
			Label:      "Microphone",
			Status:     capture.StatusQueued,
			Backend:    "audio-only-native",
			Permission: capture.PermissionUnknown,
			Reason:     "audio-only PipeWire/PulseAudio microphone capture is queued",
		}
	default:
		return capture.Capability{
			ID:         "microphone",
			Label:      "Microphone",
			Status:     capture.StatusUnsupported,
			Backend:    "audio-only-native",
			Permission: capture.PermissionNotRequired,
			Reason:     fmt.Sprintf("audio-only microphone capture is not implemented for platform %q", capabilities.Platform),
		}
	}
}

func selectedRegionSourceIsBoundToDisplay(req recording.StartRequest) bool {
	return req.SourceType == devices.SourceRegion &&
		req.SourceGeometry != nil &&
		req.SourceGeometry.Width > 0 &&
		req.SourceGeometry.Height > 0 &&
		strings.TrimSpace(req.SourceGeometry.NativeID) != "" &&
		strings.TrimSpace(req.SourceGeometry.NativeID) != "region:virtual-desktop"
}

func isMockBackend(backend string) bool {
	return backend == "mock-package" || backend == "browser-mock" || backend == "ui-preview"
}

func aggregate(checks []Check) Status {
	status := StatusReady
	for _, check := range checks {
		if check.Status == StatusBlocked {
			return StatusBlocked
		}
		if check.Status == StatusWarning {
			status = StatusWarning
		}
	}
	return status
}

func messageFor(status Status, mockBackend bool) string {
	switch status {
	case StatusReady:
		return "Ready to start recording."
	case StatusWarning:
		if mockBackend {
			return "Ready for UI shell package recording; native capture checks are still queued."
		}
		return "Recording can continue with warnings."
	case StatusBlocked:
		return "Recording cannot start until blocked checks are resolved."
	default:
		return "Recording preflight completed."
	}
}

func ready(id string, label string, reason string) Check {
	return Check{ID: id, Label: label, Status: StatusReady, Reason: reason}
}

func warning(id string, label string, reason string) Check {
	return Check{ID: id, Label: label, Status: StatusWarning, Reason: reason}
}

func blocked(id string, label string, reason string) Check {
	return Check{ID: id, Label: label, Status: StatusBlocked, Reason: reason}
}
