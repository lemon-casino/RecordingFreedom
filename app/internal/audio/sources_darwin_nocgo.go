//go:build darwin && !cgo

package audio

import "fmt"

func NewPlatformCaptureSources(config CaptureConfig) ([]CaptureSource, error) {
	config = normalizeCaptureConfig(config)
	if config.Microphone.Enabled || config.SystemAudio.Enabled {
		return nil, fmt.Errorf("CoreAudio capture requires a macOS cgo build")
	}
	return nil, fmt.Errorf("native audio capture backend %q has no enabled streams", config.Backend)
}
