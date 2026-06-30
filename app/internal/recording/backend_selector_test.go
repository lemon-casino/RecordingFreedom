package recording

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestSelectBackendDefaultsToMockPackage(t *testing.T) {
	backend := SelectBackend(recpackage.NewService(), "darwin", "")
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
		{platform: "windows", want: BackendWindowsGraphicsCapture},
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

func TestDefaultBackendHonorsNativeEnvironment(t *testing.T) {
	t.Setenv(EnvRecordingBackend, "native")

	backend := DefaultBackend(nil)
	if backend.ID() == BackendMockPackage {
		t.Fatalf("DefaultBackend() = %q with %s=native, want native queued backend", backend.ID(), EnvRecordingBackend)
	}
}

func TestServiceNativeBackendCannotCreatePackage(t *testing.T) {
	t.Setenv(EnvRecordingBackend, "native")
	root := t.TempDir()
	service := NewService(appdata.NewService(root))

	if service.BackendID() == BackendMockPackage {
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
