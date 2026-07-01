package recording

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func TestSelectBackendDefaultsToPlatformNativeBackend(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "darwin", "")
	if backend.ID() != BackendScreenCaptureKit {
		t.Fatalf("backend = %q, want %q", backend.ID(), BackendScreenCaptureKit)
	}
}

func TestSelectBackendCanExplicitlyUseMockPackage(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "darwin", "mock-package")
	if backend.ID() != BackendMockPackage {
		t.Fatalf("backend = %q, want %q", backend.ID(), BackendMockPackage)
	}
}

func TestSelectBackendNativeUsesPlatformBackendID(t *testing.T) {
	tests := []struct {
		platform string
		want     string
	}{
		{platform: "darwin", want: BackendScreenCaptureKit},
		{platform: "windows", want: BackendFFmpegDesktopCapture},
		{platform: "linux", want: BackendPipeWirePortal},
		{platform: "plan9", want: BackendNativeUnsupported},
	}

	for _, test := range tests {
		backend := SelectBackend(nil, test.platform, "native")
		if backend.ID() != test.want {
			t.Fatalf("SelectBackend(native, %s) = %q, want %q", test.platform, backend.ID(), test.want)
		}
	}
}

func TestBackendRegistrySelectsRegisteredNativeBackend(t *testing.T) {
	packages := recpackage.NewService()
	var receivedPackages *recpackage.Service
	registry := NewBackendRegistry().WithNativeBackend(BackendScreenCaptureKit, func(p *recpackage.Service) Backend {
		receivedPackages = p
		return &registryBackend{id: BackendScreenCaptureKit}
	})

	backend := registry.Select(packages, "darwin", "native")
	if backend.ID() != BackendScreenCaptureKit {
		t.Fatalf("registry.Select(native darwin) = %q, want %q", backend.ID(), BackendScreenCaptureKit)
	}
	if receivedPackages != packages {
		t.Fatalf("registered factory received packages %p, want %p", receivedPackages, packages)
	}

	backend = registry.Select(packages, "windows", "sck")
	if backend.ID() != BackendScreenCaptureKit {
		t.Fatalf("registry.Select(sck) = %q, want %q", backend.ID(), BackendScreenCaptureKit)
	}
}

func TestBackendRegistryCanSelectNativeRuntimeBackend(t *testing.T) {
	registry := NewBackendRegistry().WithNativeBackend(BackendScreenCaptureKit, func(packages *recpackage.Service) Backend {
		return NewNativeRuntimeBackend(BackendScreenCaptureKit, packages, NativeBackendRuntimeOptions{
			VideoSessionFactory: func(video.CaptureConfig) (NativeVideoSession, error) {
				return &fakeNativeVideoSession{}, nil
			},
		})
	})

	backend := registry.Select(recpackage.NewService(), "darwin", "native")
	runtimeBackend, ok := backend.(*NativeRuntimeBackend)
	if !ok {
		t.Fatalf("registry.Select(native darwin) = %T, want *NativeRuntimeBackend", backend)
	}
	if runtimeBackend.ID() != BackendScreenCaptureKit {
		t.Fatalf("runtime backend id = %q, want %q", runtimeBackend.ID(), BackendScreenCaptureKit)
	}
}

func TestBackendRegistryFallsBackToQueuedNativeBackend(t *testing.T) {
	registry := NewBackendRegistry().WithNativeBackend(BackendScreenCaptureKit, nil)

	backend := registry.Select(nil, "darwin", "native")
	if backend.ID() != BackendScreenCaptureKit {
		t.Fatalf("registry.Select(native darwin) = %q, want queued %q", backend.ID(), BackendScreenCaptureKit)
	}
	if _, ok := backend.(*QueuedNativeBackend); !ok {
		t.Fatalf("registry.Select(native darwin) = %T, want queued backend fallback", backend)
	}
}

func TestQueuedNativeBackendCannotStartCapture(t *testing.T) {
	backend := NewQueuedNativeBackend(BackendScreenCaptureKit)
	result, err := backend.Start(context.Background(), BackendStartRequest{})
	if err == nil {
		t.Fatal("Start() error = nil, want queued backend error")
	}
	if result.Package.Dir != "" {
		t.Fatalf("Start() package dir = %q, want empty result", result.Package.Dir)
	}
}

type registryBackend struct {
	id string
}

func (b *registryBackend) ID() string {
	return b.id
}

func (b *registryBackend) Start(context.Context, BackendStartRequest) (BackendStartResult, error) {
	return BackendStartResult{}, nil
}

func (b *registryBackend) Pause(context.Context, BackendControlRequest) error {
	return nil
}

func (b *registryBackend) Resume(context.Context, BackendControlRequest) error {
	return nil
}

func (b *registryBackend) Stop(context.Context, BackendControlRequest) (BackendStopResult, error) {
	return BackendStopResult{}, nil
}

func TestDefaultBackendHonorsNativeEnvironment(t *testing.T) {
	t.Setenv(EnvRecordingBackend, "native")

	backend := DefaultBackend(nil)
	if backend.ID() == BackendMockPackage {
		t.Fatalf("DefaultBackend() = %q with %s=native, want native queued backend", backend.ID(), EnvRecordingBackend)
	}
}

func TestServiceQueuedNativeBackendCannotCreatePackage(t *testing.T) {
	root := t.TempDir()
	service := NewServiceWithBackend(appdata.NewService(root), NewQueuedNativeBackend(BackendScreenCaptureKit))

	if service.BackendID() != BackendScreenCaptureKit {
		t.Fatalf("BackendID() = %q, want queued native backend", service.BackendID())
	}
	_, err := service.StartRecording(StartRequest{SourceID: "screen:primary", SourceType: SourceScreen})
	if err == nil {
		t.Fatal("StartRecording() error = nil, want queued native backend error")
	}
	if service.State() != StateFailed {
		t.Fatalf("State() = %q, want %q", service.State(), StateFailed)
	}
	matches, err := filepath.Glob(filepath.Join(root, "data", "video", "*.rfrec"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("queued native backend created packages = %#v, want none", matches)
	}
}
