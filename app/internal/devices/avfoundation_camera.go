package devices

import (
	"fmt"
	"regexp"
	"strings"
)

var avFoundationDeviceLinePattern = regexp.MustCompile(`\[(\d+)\]\s+(.+)$`)

func parseAVFoundationCameraDevices(output string) []MediaDevice {
	lines := strings.Split(output, "\n")
	inVideoSection := false
	devices := make([]MediaDevice, 0)
	seenIDs := map[string]int{}
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "avfoundation video devices"):
			inVideoSection = true
			continue
		case strings.Contains(lower, "avfoundation audio devices"):
			inVideoSection = false
			continue
		}
		if !inVideoSection {
			continue
		}
		match := avFoundationDeviceLinePattern.FindStringSubmatch(line)
		if len(match) < 3 {
			continue
		}
		index := strings.TrimSpace(match[1])
		name := strings.TrimSpace(match[2])
		if name == "" || isAVFoundationScreenDevice(name) {
			continue
		}
		id := avFoundationCameraID(name, len(devices)+1)
		if count := seenIDs[id]; count > 0 {
			id = fmt.Sprintf("%s-%d", id, count+1)
		}
		seenIDs[id]++
		device := MediaDevice{
			ID:              id,
			Type:            DeviceCamera,
			Name:            name,
			Subtitle:        "AVFoundation camera endpoint",
			NativeID:        index,
			IsDefault:       len(devices) == 0,
			Available:       true,
			Capability:      CapabilityEnumerated,
			SidecarEligible: true,
		}
		if device.IsDefault {
			device.Name = "Default " + device.Name
		}
		devices = append(devices, device)
	}
	return devices
}

func isAVFoundationScreenDevice(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(lower, "capture screen") || strings.Contains(lower, "screen capture")
}

func avFoundationCameraID(name string, index int) string {
	token := sanitizeMediaID(name)
	if token == "" {
		token = fmt.Sprintf("camera-%d", index)
	}
	return "camera:avfoundation:" + token
}
