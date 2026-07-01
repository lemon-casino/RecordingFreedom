//go:build !windows

package video

import "fmt"

func NewPlatformCameraSession(config CameraCaptureConfig) (CameraSession, error) {
	config = NormalizeCameraCaptureConfig(config)
	return nil, fmt.Errorf("camera sidecar capture is not implemented for backend %q on this platform", config.Backend)
}
