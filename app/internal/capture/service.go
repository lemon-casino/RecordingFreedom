package capture

import (
	"runtime"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

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
		return available("screen-recording", "Screen Recording", "ScreenCaptureKit", PermissionScreenRecording, "ScreenCaptureKit display capture is implemented; system-audio mux is wired in code and still needs real-device smoke validation.")
	case "windows":
		if path, ok, reason := video.FFmpegAvailability(); ok {
			return available("screen-recording", "Screen Recording", "FFmpeg gdigrab", PermissionNotRequired, "FFmpeg gdigrab desktop writer is available at "+path+" and records screen/all-screen/region video to screen.mp4; enabled WASAPI audio is muxed into screen.mp4 at stop.")
		} else {
			return blocked("screen-recording", "Screen Recording", "FFmpeg gdigrab", PermissionNotRequired, reason)
		}
	case "linux":
		return queued("screen-recording", "Screen Recording", "XDG Desktop Portal + PipeWire", PermissionUnknown, "Linux capture backend is experimental and queued.")
	default:
		return unsupported("screen-recording", "Screen Recording", "unknown", "No native screen capture backend is planned for this platform yet.")
	}
}

func windowRecordingCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return available("window-recording", "Window Recording", "ScreenCaptureKit", PermissionScreenRecording, "ScreenCaptureKit single-window capture is implemented; system-audio mux is wired in code and still needs real-device smoke validation.")
	case "windows":
		if path, ok, reason := video.FFmpegAvailability(); ok {
			return available("window-recording", "Window Recording", "FFmpeg gdigrab hwnd", PermissionNotRequired, "FFmpeg gdigrab HWND writer is available at "+path+" and records locked window video to screen.mp4; enabled WASAPI audio is muxed into screen.mp4 at stop.")
		} else {
			return blocked("window-recording", "Window Recording", "FFmpeg gdigrab hwnd", PermissionNotRequired, reason)
		}
	case "linux":
		return queued("window-recording", "Window Recording", "XDG Desktop Portal + PipeWire", PermissionUnknown, "Portal-based window capture is queued.")
	default:
		return unsupported("window-recording", "Window Recording", "unknown", "No native window capture backend is planned for this platform yet.")
	}
}

func applicationRecordingCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("application-recording", "Program Recording", "ScreenCaptureKit", PermissionScreenRecording, "Program sources are intentionally hidden from the current capsule menu; select a locked window for the first video writer.")
	case "windows":
		return queued("application-recording", "Program Recording", "Windows desktop capture", PermissionNotRequired, "Program sources are intentionally hidden from the current capsule menu; select a locked window for the first Windows video writer.")
	case "linux":
		return queued("application-recording", "Program Recording", "XDG Desktop Portal + PipeWire", PermissionUnknown, "Program grouping depends on portal/PipeWire source metadata.")
	default:
		return unsupported("application-recording", "Program Recording", "unknown", "No native program capture backend is planned for this platform yet.")
	}
}

func systemAudioCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return available("system-audio", "System Audio", "ScreenCaptureKit", PermissionScreenRecording, "ScreenCaptureKit system audio mux into screen.mp4 is implemented; real-device smoke validation is still required.")
	case "windows":
		return available("system-audio", "System Audio", "WASAPI loopback", PermissionNotRequired, "WASAPI loopback capture is implemented; screen recordings mux it into screen.mp4 at stop, and active samples require system playback.")
	case "linux":
		return queued("system-audio", "System Audio", "PipeWire/PulseAudio", PermissionUnknown, "PipeWire monitor source enumeration and capture are queued.")
	default:
		return unsupported("system-audio", "System Audio", "unknown", "No system audio backend is planned for this platform yet.")
	}
}

func microphoneCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return available("microphone", "Microphone", "CoreAudio", PermissionMicrophone, "CoreAudio microphone PCM capture is implemented for device level monitoring and microphone sidecar recording.")
	case "windows":
		return available("microphone", "Microphone", "WASAPI capture", PermissionNotRequired, "WASAPI microphone PCM capture is implemented; screen recordings mux it into screen.mp4 at stop and retain diagnostics.")
	case "linux":
		return queued("microphone", "Microphone", "PipeWire/PulseAudio", PermissionUnknown, "PipeWire microphone capture is queued.")
	default:
		return unsupported("microphone", "Microphone", "unknown", "No microphone backend is planned for this platform yet.")
	}
}

func microphoneEnhancementCapability() Capability {
	if rnnoise.Available() {
		return available("microphone-enhancement", "Microphone RNNoise", "RNNoise native DSP", PermissionNotRequired, "RNNoise native DSP is available and wired into microphone capture; system audio is never denoised.")
	}
	return queued("microphone-enhancement", "Microphone RNNoise", "RNNoise native DSP", PermissionNotRequired, "RNNoise requires a cgo build with the rnnoise_native tag; the current build cannot create the native suppressor.")
}

func cameraSidecarCapability(platform string) Capability {
	switch platform {
	case "darwin":
		return queued("camera-sidecar", "Camera Sidecar", "AVFoundation", PermissionCamera, "AVFoundation camera sidecar capture is queued.")
	case "windows":
		if path, ok, reason := video.FFmpegAvailability(); ok {
			return available("camera-sidecar", "Camera Sidecar", "FFmpeg DirectShow", PermissionNotRequired, "FFmpeg DirectShow camera sidecar writer is available at "+path+" and records package-local webcam.mp4.")
		} else {
			return blocked("camera-sidecar", "Camera Sidecar", "FFmpeg DirectShow", PermissionNotRequired, reason)
		}
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

func blocked(id string, label string, backend string, permission Permission, reason string) Capability {
	return Capability{ID: id, Label: label, Status: StatusBlocked, Backend: backend, Permission: permission, Reason: reason}
}

func unsupported(id string, label string, backend string, reason string) Capability {
	return Capability{ID: id, Label: label, Status: StatusUnsupported, Backend: backend, Permission: PermissionNotRequired, Reason: reason}
}
