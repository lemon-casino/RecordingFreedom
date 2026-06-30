//go:build !darwin && !windows && !linux

package devices

import "runtime"

func listPlatformMediaDevices() (MediaInventory, error) {
	return defaultMediaInventory(runtime.GOOS), nil
}
