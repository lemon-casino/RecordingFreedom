package capture

import "runtime"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Capabilities() Capabilities {
	return Capabilities{
		Platform:              runtime.GOOS,
		SourceEnumeration:     sourceEnumerationCapability(),
		ScreenRecording:       screenRecordingCapability(runtime.GOOS),
		WindowRecording:       windowRecordingCapability(runtime.GOOS),
		ApplicationRecording:  applicationRecordingCapability(runtime.GOOS),
		SystemAudio:           systemAudioCapability(runtime.GOOS),
		Microphone:            microphoneCapability(runtime.GOOS),
		MicrophoneEnhancement: microphoneEnhancementCapability(),
		CameraSidecar:         cameraSidecarCapability(runtime.GOOS),
		PIPExport:             queued("pip-export", "PIP Export", "export-compositor", PermissionNotRequired, "PIP export will compose screen video and webcam sidecar after native capture lands."),
		PackageRecovery:       available("package-recovery", "Recording Package Recovery", "recpackage", PermissionNotRequired, "Scans .rfrec packages and marks unfinished packages as recoverable."),
	}
}

func screenRecordingCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("screen-recording", "Screen Recording", "ScreenCaptureKit", PermissionScreenRecording, "ScreenCaptureKit capture backend is the next macOS milestone.")
	case "windows":
		return queued("screen-recording", "Screen Recording", "Windows.Graphics.Capture", PermissionNotRequired, "WGC capture backend is queued behind source and package contracts.")
	case "linux":
		return queued("screen-recording", "Screen Recording", "XDG Desktop Portal + PipeWire", PermissionUnknown, "Linux capture backend is experimental and queued.")
	default:
		return unsupported("screen-recording", "Screen Recording", "unknown", "No native screen capture backend is planned for this platform yet.")
	}
}

func windowRecordingCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("window-recording", "Window Recording", "ScreenCaptureKit", PermissionScreenRecording, "CoreGraphics can enumerate windows; ScreenCaptureKit capture target mapping is still queued.")
	case "windows":
		return queued("window-recording", "Window Recording", "Windows.Graphics.Capture", PermissionNotRequired, "Win32 can enumerate windows; WGC capture target mapping is still queued.")
	case "linux":
		return queued("window-recording", "Window Recording", "XDG Desktop Portal + PipeWire", PermissionUnknown, "Portal-based window capture is queued.")
	default:
		return unsupported("window-recording", "Window Recording", "unknown", "No native window capture backend is planned for this platform yet.")
	}
}

func applicationRecordingCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("application-recording", "Program Recording", "ScreenCaptureKit", PermissionScreenRecording, "Program sources are currently PID groups that must map to ScreenCaptureKit windows.")
	case "windows":
		return queued("application-recording", "Program Recording", "Windows.Graphics.Capture", PermissionNotRequired, "Program sources are currently process groups that must map to WGC windows.")
	case "linux":
		return queued("application-recording", "Program Recording", "XDG Desktop Portal + PipeWire", PermissionUnknown, "Program grouping depends on portal/PipeWire source metadata.")
	default:
		return unsupported("application-recording", "Program Recording", "unknown", "No native program capture backend is planned for this platform yet.")
	}
}

func systemAudioCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("system-audio", "System Audio", "ScreenCaptureKit", PermissionScreenRecording, "ScreenCaptureKit system audio capture is queued.")
	case "windows":
		return available("system-audio", "System Audio", "WASAPI loopback", PermissionNotRequired, "WASAPI loopback capture source is implemented; active samples require system playback.")
	case "linux":
		return queued("system-audio", "System Audio", "PipeWire/PulseAudio", PermissionUnknown, "PipeWire monitor source enumeration and capture are queued.")
	default:
		return unsupported("system-audio", "System Audio", "unknown", "No system audio backend is planned for this platform yet.")
	}
}

func microphoneCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("microphone", "Microphone", "CoreAudio", PermissionMicrophone, "CoreAudio microphone capture is queued.")
	case "windows":
		return available("microphone", "Microphone", "WASAPI capture", PermissionNotRequired, "WASAPI microphone PCM capture is implemented and writes package audio sidecars.")
	case "linux":
		return queued("microphone", "Microphone", "PipeWire/PulseAudio", PermissionUnknown, "PipeWire microphone capture is queued.")
	default:
		return unsupported("microphone", "Microphone", "unknown", "No microphone backend is planned for this platform yet.")
	}
}

func microphoneEnhancementCapability() Capability {
	return queued("microphone-enhancement", "Microphone RNNoise", "RNNoise native DSP", PermissionNotRequired, "RNNoise native wrapper is implemented for the audio pipeline, but it is not wired into the app recording backend yet.")
}

func cameraSidecarCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("camera-sidecar", "Camera Sidecar", "AVFoundation", PermissionCamera, "AVFoundation camera sidecar capture is queued.")
	case "windows":
		return queued("camera-sidecar", "Camera Sidecar", "Media Foundation", PermissionNotRequired, "Media Foundation camera sidecar capture is queued.")
	case "linux":
		return queued("camera-sidecar", "Camera Sidecar", "PipeWire", PermissionUnknown, "PipeWire camera sidecar capture is queued.")
	default:
		return unsupported("camera-sidecar", "Camera Sidecar", "unknown", "No camera sidecar backend is planned for this platform yet.")
	}
}

func available(id string, label string, backend string, permission Permission, reason string) Capability {
	return Capability{ID: id, Label: label, Status: StatusAvailable, Backend: backend, Permission: permission, Reason: reason}
}

func queued(id string, label string, backend string, permission Permission, reason string) Capability {
	return Capability{ID: id, Label: label, Status: StatusQueued, Backend: backend, Permission: permission, Reason: reason}
}

func unsupported(id string, label string, backend string, reason string) Capability {
	return Capability{ID: id, Label: label, Status: StatusUnsupported, Backend: backend, Permission: PermissionNotRequired, Reason: reason}
}
