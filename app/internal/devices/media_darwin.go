//go:build darwin

package devices

func listPlatformMediaDevices() (MediaInventory, error) {
	return defaultMediaInventory("darwin"), nil
}
