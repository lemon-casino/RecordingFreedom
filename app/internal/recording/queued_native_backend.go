package recording

import (
	"context"
	"fmt"
)

type QueuedNativeBackend struct {
	id string
}

func NewQueuedNativeBackend(id string) *QueuedNativeBackend {
	if id == "" {
		id = BackendNativeUnsupported
	}
	return &QueuedNativeBackend{id: id}
}

func (b *QueuedNativeBackend) ID() string {
	return b.id
}

func (b *QueuedNativeBackend) Start(context.Context, BackendStartRequest) (BackendStartResult, error) {
	return BackendStartResult{}, fmt.Errorf("native recording backend %q is queued and cannot start capture yet", b.id)
}

func (b *QueuedNativeBackend) Pause(context.Context, BackendControlRequest) error {
	return fmt.Errorf("native recording backend %q is not running", b.id)
}

func (b *QueuedNativeBackend) Resume(context.Context, BackendControlRequest) error {
	return fmt.Errorf("native recording backend %q is not running", b.id)
}

func (b *QueuedNativeBackend) Stop(context.Context, BackendControlRequest) (BackendStopResult, error) {
	return BackendStopResult{}, fmt.Errorf("native recording backend %q is not running", b.id)
}
