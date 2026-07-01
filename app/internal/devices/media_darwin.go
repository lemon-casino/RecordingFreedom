//go:build darwin && cgo

package devices

/*
#cgo darwin LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

enum {
	// CoreAudio's main property element is ABI value 0. Avoid the SDK macro because
	// current headers expand it through the deprecated Master alias and warn in CI.
	rfcaAudioObjectPropertyElementMain = 0,
};

typedef struct {
	char *uid;
	char *name;
	int isDefault;
} rf_coreaudio_device;

typedef struct {
	rf_coreaudio_device *items;
	int count;
	char *error;
} rf_coreaudio_device_list;

static char *rfca_strdup(const char *value) {
	if (value == NULL) {
		return NULL;
	}
	size_t length = strlen(value);
	char *copy = (char *)malloc(length + 1);
	if (copy == NULL) {
		return NULL;
	}
	memcpy(copy, value, length + 1);
	return copy;
}

static char *rfca_copy_cfstring(CFStringRef value) {
	if (value == NULL) {
		return NULL;
	}
	const char *direct = CFStringGetCStringPtr(value, kCFStringEncodingUTF8);
	if (direct != NULL) {
		return rfca_strdup(direct);
	}
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(CFStringGetLength(value), kCFStringEncodingUTF8) + 1;
	char *buffer = (char *)malloc((size_t)maxSize);
	if (buffer == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(value, buffer, maxSize, kCFStringEncodingUTF8)) {
		free(buffer);
		return NULL;
	}
	return buffer;
}

static char *rfca_audio_string_property(AudioObjectID objectID, AudioObjectPropertySelector selector) {
	AudioObjectPropertyAddress address = {
		.mSelector = selector,
		.mScope = kAudioObjectPropertyScopeGlobal,
		.mElement = rfcaAudioObjectPropertyElementMain,
	};
	CFStringRef value = NULL;
	UInt32 size = sizeof(value);
	OSStatus status = AudioObjectGetPropertyData(objectID, &address, 0, NULL, &size, &value);
	if (status != noErr || value == NULL) {
		return NULL;
	}
	char *result = rfca_copy_cfstring(value);
	CFRelease(value);
	return result;
}

static int rfca_input_channel_count(AudioDeviceID deviceID) {
	AudioObjectPropertyAddress address = {
		.mSelector = kAudioDevicePropertyStreamConfiguration,
		.mScope = kAudioDevicePropertyScopeInput,
		.mElement = rfcaAudioObjectPropertyElementMain,
	};
	UInt32 size = 0;
	OSStatus status = AudioObjectGetPropertyDataSize(deviceID, &address, 0, NULL, &size);
	if (status != noErr || size == 0) {
		return 0;
	}
	AudioBufferList *bufferList = (AudioBufferList *)malloc(size);
	if (bufferList == NULL) {
		return 0;
	}
	status = AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &size, bufferList);
	if (status != noErr) {
		free(bufferList);
		return 0;
	}
	int channels = 0;
	for (UInt32 index = 0; index < bufferList->mNumberBuffers; index++) {
		channels += (int)bufferList->mBuffers[index].mNumberChannels;
	}
	free(bufferList);
	return channels;
}

static AudioDeviceID rfca_default_input_device(void) {
	AudioDeviceID deviceID = kAudioObjectUnknown;
	AudioObjectPropertyAddress address = {
		.mSelector = kAudioHardwarePropertyDefaultInputDevice,
		.mScope = kAudioObjectPropertyScopeGlobal,
		.mElement = rfcaAudioObjectPropertyElementMain,
	};
	UInt32 size = sizeof(deviceID);
	OSStatus status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &address, 0, NULL, &size, &deviceID);
	if (status != noErr) {
		return kAudioObjectUnknown;
	}
	return deviceID;
}

static rf_coreaudio_device_list rfca_list_input_devices(void) {
	rf_coreaudio_device_list result;
	result.items = NULL;
	result.count = 0;
	result.error = NULL;

	AudioObjectPropertyAddress address = {
		.mSelector = kAudioHardwarePropertyDevices,
		.mScope = kAudioObjectPropertyScopeGlobal,
		.mElement = rfcaAudioObjectPropertyElementMain,
	};
	UInt32 size = 0;
	OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &address, 0, NULL, &size);
	if (status != noErr || size == 0) {
		result.error = rfca_strdup("CoreAudio returned no hardware devices");
		return result;
	}

	UInt32 deviceCount = size / sizeof(AudioDeviceID);
	AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(size);
	if (deviceIDs == NULL) {
		result.error = rfca_strdup("CoreAudio device allocation failed");
		return result;
	}
	status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &address, 0, NULL, &size, deviceIDs);
	if (status != noErr) {
		free(deviceIDs);
		result.error = rfca_strdup("CoreAudio device enumeration failed");
		return result;
	}

	rf_coreaudio_device *items = (rf_coreaudio_device *)calloc(deviceCount, sizeof(rf_coreaudio_device));
	if (items == NULL) {
		free(deviceIDs);
		result.error = rfca_strdup("CoreAudio result allocation failed");
		return result;
	}

	AudioDeviceID defaultInput = rfca_default_input_device();
	int outputCount = 0;
	for (UInt32 index = 0; index < deviceCount; index++) {
		AudioDeviceID deviceID = deviceIDs[index];
		if (rfca_input_channel_count(deviceID) <= 0) {
			continue;
		}
		char *uid = rfca_audio_string_property(deviceID, kAudioDevicePropertyDeviceUID);
		if (uid == NULL || strlen(uid) == 0) {
			free(uid);
			continue;
		}
		char *name = rfca_audio_string_property(deviceID, kAudioObjectPropertyName);
		if (name == NULL || strlen(name) == 0) {
			free(name);
			name = rfca_strdup("CoreAudio Microphone");
		}
		items[outputCount].uid = uid;
		items[outputCount].name = name;
		items[outputCount].isDefault = deviceID == defaultInput ? 1 : 0;
		outputCount++;
	}
	free(deviceIDs);

	if (outputCount == 0) {
		free(items);
		result.error = rfca_strdup("CoreAudio returned no input devices");
		return result;
	}

	result.items = items;
	result.count = outputCount;
	return result;
}

static void rfca_free_device_list(rf_coreaudio_device_list list) {
	if (list.items != NULL) {
		for (int index = 0; index < list.count; index++) {
			free(list.items[index].uid);
			free(list.items[index].name);
		}
		free(list.items);
	}
	free(list.error);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func listPlatformMediaDevices() (MediaInventory, error) {
	microphones, err := listCoreAudioMicrophones()
	if err != nil {
		return MediaInventory{}, err
	}
	return MediaInventory{
		SystemAudio: []MediaDevice{
			defaultMediaDeviceForPlatform("darwin", DeviceSystemAudio, mediaBackendMessage("darwin", DeviceSystemAudio)),
		},
		Microphones: microphones,
		Cameras: []MediaDevice{
			defaultMediaDevice(DeviceCamera, mediaBackendMessage("darwin", DeviceCamera)),
		},
		Enhancement: defaultAudioEnhancement("RNNoise requires the rnnoise_native tag; CoreAudio microphone PCM is available without denoising."),
	}, nil
}

func listCoreAudioMicrophones() ([]MediaDevice, error) {
	list := C.rfca_list_input_devices()
	defer C.rfca_free_device_list(list)
	if list.error != nil {
		return nil, fmt.Errorf("%s", C.GoString(list.error))
	}
	count := int(list.count)
	if count == 0 || list.items == nil {
		return nil, fmt.Errorf("CoreAudio returned no input devices")
	}
	items := unsafe.Slice(list.items, count)
	devices := make([]MediaDevice, 0, count)
	for _, item := range items {
		uid := C.GoString(item.uid)
		name := C.GoString(item.name)
		if uid == "" {
			continue
		}
		device := MediaDevice{
			ID:              "microphone:coreaudio:" + uid,
			Type:            DeviceMicrophone,
			Name:            name,
			Subtitle:        "CoreAudio input device",
			NativeID:        uid,
			IsDefault:       item.isDefault == 1,
			Available:       true,
			Capability:      CapabilityEnumerated,
			RNNoiseEligible: true,
		}
		if device.IsDefault {
			device.ID = "microphone:default"
			device.Name = "Default " + name
		}
		devices = append(devices, device)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("CoreAudio returned no input devices with stable UIDs")
	}
	return devices, nil
}
