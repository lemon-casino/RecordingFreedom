//go:build (!darwin && !windows) || (darwin && !cgo)

package video

import "fmt"

func NewPlatformSession(config CaptureConfig) (Session, error) {
	config = NormalizeCaptureConfig(config)
	return nil, fmt.Errorf("native video capture backend %q is not implemented for source %q yet", config.Backend, config.SourceID)
}
