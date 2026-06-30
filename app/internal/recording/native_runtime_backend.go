package recording

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type NativeRuntimeBackend struct {
	id       string
	packages *recpackage.Service
	options  NativeBackendRuntimeOptions

	mu       sync.Mutex
	runtimes map[string]*NativeBackendRuntime
}

func NewNativeRuntimeBackend(id string, packages *recpackage.Service, options NativeBackendRuntimeOptions) *NativeRuntimeBackend {
	id = normalizeBackendRequest(id)
	if id == "" {
		id = BackendNativeUnsupported
	}
	if packages == nil {
		packages = recpackage.NewService()
	}
	return &NativeRuntimeBackend{
		id:       id,
		packages: packages,
		options:  options,
		runtimes: map[string]*NativeBackendRuntime{},
	}
}

func (b *NativeRuntimeBackend) ID() string {
	return b.id
}

func (b *NativeRuntimeBackend) Start(ctx context.Context, req BackendStartRequest) (BackendStartResult, error) {
	runtime, err := NewNativeBackendRuntime(b.packages, b.id, req, b.options)
	if err != nil {
		return BackendStartResult{}, err
	}
	if err := runtime.Start(ctx); err != nil {
		_ = runtime.MarkPackageFailed()
		return BackendStartResult{}, err
	}

	b.mu.Lock()
	b.runtimes[runtime.Plan.Package.ID] = runtime
	b.mu.Unlock()
	return BackendStartResult{Package: runtime.Plan.Package}, nil
}

func (b *NativeRuntimeBackend) Pause(_ context.Context, req BackendControlRequest) error {
	runtime, err := b.runtime(req.Session.ID)
	if err != nil {
		return err
	}
	return runtime.Pause()
}

func (b *NativeRuntimeBackend) Resume(_ context.Context, req BackendControlRequest) error {
	runtime, err := b.runtime(req.Session.ID)
	if err != nil {
		return err
	}
	return runtime.Resume()
}

func (b *NativeRuntimeBackend) Stop(_ context.Context, req BackendControlRequest) (BackendStopResult, error) {
	runtime, err := b.runtime(req.Session.ID)
	if err != nil {
		return BackendStopResult{}, err
	}
	if err := runtime.Stop(); err != nil {
		_ = runtime.MarkPackageFailed()
		return BackendStopResult{}, err
	}

	b.mu.Lock()
	delete(b.runtimes, req.Session.ID)
	b.mu.Unlock()
	return BackendStopResult{SyncDiagnostics: runtime.SyncDiagnostics()}, nil
}

func (b *NativeRuntimeBackend) runtime(sessionID string) (*NativeBackendRuntime, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionID == "" {
		return nil, errors.New("native runtime session id is required")
	}
	runtime := b.runtimes[sessionID]
	if runtime == nil {
		return nil, fmt.Errorf("native runtime session %q is not running", sessionID)
	}
	return runtime, nil
}
