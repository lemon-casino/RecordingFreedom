package recording

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type AudioOnlyRuntimeBackend struct {
	id       string
	packages *recpackage.Service
	options  AudioOnlyRuntimeOptions

	mu       sync.Mutex
	runtimes map[string]*AudioOnlyRuntime
}

func NewAudioOnlyRuntimeBackend(packages *recpackage.Service, options AudioOnlyRuntimeOptions) *AudioOnlyRuntimeBackend {
	if packages == nil {
		packages = recpackage.NewService()
	}
	return &AudioOnlyRuntimeBackend{
		id:       BackendAudioOnlyNative,
		packages: packages,
		options:  options,
		runtimes: map[string]*AudioOnlyRuntime{},
	}
}

func (b *AudioOnlyRuntimeBackend) ID() string {
	return b.id
}

func (b *AudioOnlyRuntimeBackend) Start(ctx context.Context, videoDir string, createdAt time.Time, req AudioOnlyRequest) (BackendStartResult, error) {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	runtime, err := NewAudioOnlyRuntime(b.packages, b.id, videoDir, createdAt, req, b.options)
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

func (b *AudioOnlyRuntimeBackend) Pause(_ context.Context, req BackendControlRequest) error {
	runtime, err := b.runtime(req.Session.ID)
	if err != nil {
		return err
	}
	return runtime.Pause()
}

func (b *AudioOnlyRuntimeBackend) Resume(_ context.Context, req BackendControlRequest) error {
	runtime, err := b.runtime(req.Session.ID)
	if err != nil {
		return err
	}
	return runtime.Resume()
}

func (b *AudioOnlyRuntimeBackend) Stop(_ context.Context, req BackendControlRequest) (BackendStopResult, error) {
	runtime, err := b.runtime(req.Session.ID)
	if err != nil {
		return BackendStopResult{}, err
	}
	if err := runtime.Stop(); err != nil {
		_ = runtime.MarkPackageFailed()
		return BackendStopResult{}, err
	}
	postStop := b.options.PostStopProcessor
	if postStop == nil {
		postStop = defaultAudioOnlyPostStopProcessor
	}
	if err := postStop(runtime); err != nil {
		_ = runtime.MarkPackageFailed()
		return BackendStopResult{}, err
	}

	b.mu.Lock()
	delete(b.runtimes, req.Session.ID)
	b.mu.Unlock()
	return BackendStopResult{SyncDiagnostics: runtime.SyncDiagnostics()}, nil
}

func (b *AudioOnlyRuntimeBackend) runtime(sessionID string) (*AudioOnlyRuntime, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionID == "" {
		return nil, errors.New("audio-only runtime session id is required")
	}
	runtime := b.runtimes[sessionID]
	if runtime == nil {
		return nil, fmt.Errorf("audio-only runtime session %q is not running", sessionID)
	}
	return runtime, nil
}
