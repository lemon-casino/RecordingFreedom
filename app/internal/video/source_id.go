package video

import (
	"strconv"
	"strings"
)

const darwinDisplaySourcePrefix = "screen:display-"
const darwinDisplayNativeIDPrefix = "cgdisplay:"
const darwinWindowSourcePrefix = "window:"
const darwinApplicationSourcePrefix = "application:"

func DarwinDisplayID(sourceID string) (uint32, bool) {
	return parseDarwinUint32SourceID(sourceID, darwinDisplaySourcePrefix)
}

func DarwinDisplayNativeID(nativeID string) (uint32, bool) {
	return parseDarwinUint32SourceID(nativeID, darwinDisplayNativeIDPrefix)
}

func DarwinWindowID(sourceID string) (uint32, bool) {
	return parseDarwinUint32SourceID(sourceID, darwinWindowSourcePrefix)
}

func DarwinApplicationPID(sourceID string) (uint32, bool) {
	return parseDarwinUint32SourceID(sourceID, darwinApplicationSourcePrefix)
}

func parseDarwinUint32SourceID(sourceID string, prefix string) (uint32, bool) {
	raw := strings.TrimSpace(sourceID)
	if !strings.HasPrefix(raw, prefix) {
		return 0, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint32(id), true
}
