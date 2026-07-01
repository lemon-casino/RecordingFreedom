//go:build darwin && !cgo

package devices

import "fmt"

func listPlatformMediaDevices() (MediaInventory, error) {
	return MediaInventory{}, fmt.Errorf("CoreAudio microphone enumeration requires a macOS cgo build")
}
