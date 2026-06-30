package video

import (
	"strconv"
	"strings"
)

const windowsScreenSourcePrefix = "screen:"
const windowsWindowSourcePrefix = "window:"
const windowsApplicationSourcePrefix = "application:"

func WindowsScreenID(sourceID string) (string, bool) {
	raw := strings.TrimSpace(sourceID)
	if !strings.HasPrefix(raw, windowsScreenSourcePrefix) {
		return "", false
	}
	value := strings.TrimSpace(strings.TrimPrefix(raw, windowsScreenSourcePrefix))
	if value == "" || value == "native-backend-queued" {
		return "", false
	}
	return value, true
}

func WindowsWindowHWND(sourceID string) (uintptr, bool) {
	raw := strings.TrimSpace(sourceID)
	if !strings.HasPrefix(raw, windowsWindowSourcePrefix) {
		return 0, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(raw, windowsWindowSourcePrefix))
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(value, 16, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uintptr(id), true
}

func WindowsApplicationPID(sourceID string) (uint32, bool) {
	raw := strings.TrimSpace(sourceID)
	if !strings.HasPrefix(raw, windowsApplicationSourcePrefix) {
		return 0, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(raw, windowsApplicationSourcePrefix))
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint32(id), true
}
