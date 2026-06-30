package recording

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"

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

type BackendFactory func(*recpackage.Service) Backend

type BackendRegistry struct {
	factories map[string]BackendFactory
}

var (
	defaultRegistryMu sync.RWMutex
	defaultRegistry   = NewBackendRegistry()
)

func DefaultBackend(packages *recpackage.Service) Backend {
	return SelectBackend(packages, runtime.GOOS, os.Getenv(EnvRecordingBackend))
}

func SelectBackend(packages *recpackage.Service, platform string, requested string) Backend {
	return DefaultBackendRegistry().Select(packages, platform, requested)
}

func NewBackendRegistry() BackendRegistry {
	return BackendRegistry{factories: map[string]BackendFactory{}}
}

func DefaultBackendRegistry() BackendRegistry {
	defaultRegistryMu.RLock()
	defer defaultRegistryMu.RUnlock()
	return defaultRegistry.clone()
}

func RegisterNativeBackend(id string, factory BackendFactory) error {
	id = normalizeBackendRequest(id)
	if id == "" {
		return errors.New("native backend id is required")
	}
	if factory == nil {
		return errors.New("native backend factory is required")
	}

	defaultRegistryMu.Lock()
	defer defaultRegistryMu.Unlock()
	defaultRegistry = defaultRegistry.WithNativeBackend(id, factory)
	return nil
}

func (r BackendRegistry) WithNativeBackend(id string, factory BackendFactory) BackendRegistry {
	next := r.clone()
	id = normalizeBackendRequest(id)
	if id == "" || factory == nil {
		return next
	}
	next.factories[id] = factory
	return next
}

func (r BackendRegistry) Select(packages *recpackage.Service, platform string, requested string) Backend {
	switch normalizeBackendRequest(requested) {
	case "", "auto", "mock", BackendMockPackage:
		return NewMockBackend(packages)
	case "native":
		return r.nativeOrQueued(packages, nativeBackendID(platform))
	case BackendScreenCaptureKit, "sck":
		return r.nativeOrQueued(packages, BackendScreenCaptureKit)
	case BackendWindowsGraphicsCapture, "wgc":
		return r.nativeOrQueued(packages, BackendWindowsGraphicsCapture)
	case BackendPipeWirePortal, "pipewire":
		return r.nativeOrQueued(packages, BackendPipeWirePortal)
	default:
		return NewQueuedNativeBackend(BackendNativeUnsupported)
	}
}

func (r BackendRegistry) nativeOrQueued(packages *recpackage.Service, id string) Backend {
	id = normalizeBackendRequest(id)
	if factory := r.factories[id]; factory != nil {
		if backend := factory(packages); backend != nil {
			return backend
		}
	}
	return NewQueuedNativeBackend(id)
}

func (r BackendRegistry) clone() BackendRegistry {
	next := NewBackendRegistry()
	for id, factory := range r.factories {
		next.factories[id] = factory
	}
	return next
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
