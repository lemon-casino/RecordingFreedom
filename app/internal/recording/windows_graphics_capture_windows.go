//go:build windows

package recording

import (
	"log"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func init() {
	if err := RegisterNativeBackend(BackendWindowsGraphicsCapture, newWindowsGraphicsCaptureBackend); err != nil {
		log.Printf("register Windows.Graphics.Capture backend: %v", err)
	}
}

func newWindowsGraphicsCaptureBackend(packages *recpackage.Service) Backend {
	return NewNativeRuntimeBackend(BackendWindowsGraphicsCapture, packages, NativeBackendRuntimeOptions{
		VideoSessionFactory: func(config video.CaptureConfig) (NativeVideoSession, error) {
			return video.NewPlatformSession(config)
		},
	})
}
