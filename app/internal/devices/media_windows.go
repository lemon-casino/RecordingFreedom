//go:build windows

package devices

func listPlatformMediaDevices() (MediaInventory, error) {
	return defaultMediaInventory("windows"), nil
}
