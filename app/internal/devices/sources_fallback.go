//go:build !windows && !darwin

package devices

import "fmt"

func listPlatformSources() ([]CaptureSource, error) {
	return nil, fmt.Errorf("native source enumeration for this platform is queued behind the capture backend")
}
