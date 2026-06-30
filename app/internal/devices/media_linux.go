//go:build linux

package devices

func listPlatformMediaDevices() (MediaInventory, error) {
	return defaultMediaInventory("linux"), nil
}
