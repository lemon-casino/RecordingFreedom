//go:build linux

package devices

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func listPlatformMediaDevices() (MediaInventory, error) {
	return MediaInventory{
		SystemAudio: []MediaDevice{
			defaultMediaDeviceForPlatform("linux", DeviceSystemAudio, mediaBackendMessage("linux", DeviceSystemAudio)),
		},
		Microphones: []MediaDevice{
			defaultMediaDevice(DeviceMicrophone, mediaBackendMessage("linux", DeviceMicrophone)),
		},
		Cameras:     listLinuxCameraDevices(),
		Enhancement: defaultAudioEnhancement("RNNoise native DSP is queued behind Linux microphone capture plumbing."),
	}, nil
}

func listLinuxCameraDevices() []MediaDevice {
	matches, err := filepath.Glob("/dev/video*")
	if err != nil || len(matches) == 0 {
		return []MediaDevice{defaultMediaDevice(DeviceCamera, "v4l2 camera enumeration returned no /dev/video* devices")}
	}
	sort.Strings(matches)
	devices := make([]MediaDevice, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		name := linuxCameraName(path)
		device := MediaDevice{
			ID:              linuxCameraID(path, len(devices)+1),
			Type:            DeviceCamera,
			Name:            name,
			Subtitle:        "v4l2 camera endpoint",
			NativeID:        path,
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
	if len(devices) == 0 {
		return []MediaDevice{defaultMediaDevice(DeviceCamera, "v4l2 camera enumeration returned no readable /dev/video* devices")}
	}
	return devices
}

func linuxCameraName(path string) string {
	base := filepath.Base(path)
	namePath := filepath.Join("/sys/class/video4linux", base, "name")
	data, err := os.ReadFile(namePath)
	if err == nil {
		if value := strings.TrimSpace(string(data)); value != "" {
			return value
		}
	}
	return base
}

func linuxCameraID(path string, index int) string {
	token := sanitizeMediaID(filepath.Base(path))
	if token == "" {
		token = fmt.Sprintf("camera-%d", index)
	}
	return "camera:v4l2:" + token
}
