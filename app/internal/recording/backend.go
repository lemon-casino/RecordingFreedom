package recording

import (
	"context"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type Backend interface {
	ID() string
	Start(context.Context, BackendStartRequest) (BackendStartResult, error)
	Pause(context.Context, BackendControlRequest) error
	Resume(context.Context, BackendControlRequest) error
	Stop(context.Context, BackendControlRequest) (BackendStopResult, error)
}

type BackendStartRequest struct {
	StartRequest StartRequest
	VideoDir     string
	CreatedAt    time.Time
}

type BackendStartResult struct {
	Package recpackage.Package
}

type BackendControlRequest struct {
	Session Session
}

type BackendStopResult struct {
	SyncDiagnostics *recpackage.ManifestSyncDiagnostics
}
