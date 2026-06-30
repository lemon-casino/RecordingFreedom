//go:build darwin && cgo

package recording

import (
	"log"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func init() {
	if err := RegisterNativeBackend(BackendScreenCaptureKit, newScreenCaptureKitBackend); err != nil {
		log.Printf("register ScreenCaptureKit backend: %v", err)
	}
}

func newScreenCaptureKitBackend(packages *recpackage.Service) Backend {
	return NewNativeRuntimeBackend(BackendScreenCaptureKit, packages, NativeBackendRuntimeOptions{
		VideoSessionFactory: video.NewPlatformSession,
	})
}
