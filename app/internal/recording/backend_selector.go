package recording

import (
	"os"
	"runtime"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

const (
	EnvRecordingBackend = "RECORDINGFREEDOM_RECORDING_BACKEND"

	BackendMockPackage            = "mock-package"
	BackendScreenCaptureKit       = "screencapturekit"
	BackendWindowsGraphicsCapture = "windows-graphics-capture"
	BackendPipeWirePortal         = "pipewire-portal"
	BackendNativeUnsupported      = "native-unsupported"
)

func DefaultBackend(packages *recpackage.Service) Backend {
	return SelectBackend(packages, runtime.GOOS, os.Getenv(EnvRecordingBackend))
}

func SelectBackend(packages *recpackage.Service, platform string, requested string) Backend {
	switch normalizeBackendRequest(requested) {
	case "", "auto", "mock", BackendMockPackage:
		return NewMockBackend(packages)
	case "native":
		return NewQueuedNativeBackend(nativeBackendID(platform))
	case BackendScreenCaptureKit, "sck":
		return NewQueuedNativeBackend(BackendScreenCaptureKit)
	case BackendWindowsGraphicsCapture, "wgc":
		return NewQueuedNativeBackend(BackendWindowsGraphicsCapture)
	case BackendPipeWirePortal, "pipewire":
		return NewQueuedNativeBackend(BackendPipeWirePortal)
	default:
		return NewQueuedNativeBackend(BackendNativeUnsupported)
	}
}

func normalizeBackendRequest(requested string) string {
	return strings.ToLower(strings.TrimSpace(requested))
}

func nativeBackendID(platform string) string {
	switch platform {
	case "darwin":
		return BackendScreenCaptureKit
	case "windows":
		return BackendWindowsGraphicsCapture
	case "linux":
		return BackendPipeWirePortal
	default:
		return BackendNativeUnsupported
	}
}
