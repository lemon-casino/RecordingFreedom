package appdata

// AvailableBytes reports the free space on the volume that contains path.
func AvailableBytes(path string) (uint64, error) {
	return availableBytes(path)
}
