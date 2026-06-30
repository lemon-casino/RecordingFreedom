package video

import (
	"strconv"
	"strings"
)

const darwinDisplaySourcePrefix = "screen:display-"

func DarwinDisplayID(sourceID string) (uint32, bool) {
	raw := strings.TrimSpace(sourceID)
	if !strings.HasPrefix(raw, darwinDisplaySourcePrefix) {
		return 0, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(raw, darwinDisplaySourcePrefix))
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint32(id), true
}
