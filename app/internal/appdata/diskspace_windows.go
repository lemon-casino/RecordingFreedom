//go:build windows

package appdata

import "golang.org/x/sys/windows"

func availableBytes(path string) (uint64, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var freeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(pathPtr, &freeBytes, nil, nil); err != nil {
		return 0, err
	}
	return freeBytes, nil
}
