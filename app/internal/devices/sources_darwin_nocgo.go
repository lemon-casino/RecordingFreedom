//go:build darwin && !cgo

package devices

import "fmt"

func listPlatformSources() ([]CaptureSource, error) {
	return nil, fmt.Errorf("macOS source enumeration requires cgo for CoreGraphics")
}
