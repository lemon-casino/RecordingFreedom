package devices

import (
	"fmt"
	"regexp"
	"strings"
)

var dshowQuotedValuePattern = regexp.MustCompile(`"([^"]+)"`)

func parseDirectShowCameraDevices(output string) []MediaDevice {
	lines := strings.Split(output, "\n")
	inVideoSection := false
	pending := -1
	devices := make([]MediaDevice, 0)
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "directshow video devices"):
			inVideoSection = true
			pending = -1
			continue
		case strings.Contains(lower, "directshow audio devices"):
			inVideoSection = false
			pending = -1
			continue
		}
		value := firstQuotedValue(line)
		if value == "" {
			continue
		}
		if strings.Contains(lower, "alternative name") {
			if pending >= 0 && pending < len(devices) {
				devices[pending].ID = directShowCameraID(value, pending+1)
			}
			continue
		}
		if strings.Contains(lower, "(audio)") {
			pending = -1
			continue
		}
		if !inVideoSection && !strings.Contains(lower, "(video)") {
			continue
		}
		device := MediaDevice{
			ID:              directShowCameraID(value, len(devices)+1),
			Type:            DeviceCamera,
			Name:            value,
			Subtitle:        "DirectShow camera endpoint",
			NativeID:        value,
			IsDefault:       len(devices) == 0,
			Available:       true,
			Capability:      CapabilityEnumerated,
			SidecarEligible: true,
		}
		if device.IsDefault {
			device.Name = "Default " + device.Name
		}
		devices = append(devices, device)
		pending = len(devices) - 1
	}
	return devices
}

func firstQuotedValue(line string) string {
	match := dshowQuotedValuePattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func directShowCameraID(nativeID string, index int) string {
	token := sanitizeMediaID(nativeID)
	if token == "" {
		token = fmt.Sprintf("camera-%d", index)
	}
	return "camera:dshow:" + token
}

func sanitizeMediaID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
