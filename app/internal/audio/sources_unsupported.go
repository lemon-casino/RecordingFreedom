//go:build !windows

package audio

import "fmt"

func NewPlatformCaptureSources(config CaptureConfig) ([]CaptureSource, error) {
	config = normalizeCaptureConfig(config)
	if config.Microphone.Enabled || config.SystemAudio.Enabled {
		return nil, fmt.Errorf("native audio capture backend %q is not implemented on this platform yet", config.Backend)
	}
	return nil, fmt.Errorf("native audio capture backend %q has no enabled streams", config.Backend)
}
