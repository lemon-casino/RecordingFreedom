//go:build darwin && cgo

#include "coreaudio_capture_darwin.h"
#include "_cgo_export.h"

#include <AudioToolbox/AudioToolbox.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

struct rf_coreaudio_recorder {
	AudioQueueRef queue;
	AudioStreamBasicDescription format;
	AudioQueueBufferRef buffers[3];
	UInt32 buffer_byte_size;
	uintptr_t handle;
	volatile int running;
};

static char *rf_coreaudio_strdup(const char *value) {
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

static void rf_coreaudio_set_error(char **target, const char *message, OSStatus status) {
	if (target == NULL) {
		return;
	}
	char buffer[192];
	snprintf(buffer, sizeof(buffer), "%s (OSStatus %d)", message, (int)status);
	*target = rf_coreaudio_strdup(buffer);
}

static void rf_coreaudio_input_callback(
	void *user_data,
	AudioQueueRef queue,
	AudioQueueBufferRef buffer,
	const AudioTimeStamp *start_time,
	UInt32 packet_count,
	const AudioStreamPacketDescription *packet_descriptions
) {
	(void)queue;
	(void)start_time;
	(void)packet_count;
	(void)packet_descriptions;

	rf_coreaudio_recorder *recorder = (rf_coreaudio_recorder *)user_data;
	if (recorder == NULL || buffer == NULL) {
		return;
	}
	if (recorder->running && buffer->mAudioData != NULL && buffer->mAudioDataByteSize > 0) {
		rfCoreAudioInputCallback(recorder->handle, buffer->mAudioData, buffer->mAudioDataByteSize);
	}
	if (recorder->running) {
		AudioQueueEnqueueBuffer(recorder->queue, buffer, 0, NULL);
	}
}

rf_coreaudio_recorder *rf_coreaudio_new(uintptr_t handle, const char *device_uid, double sample_rate, uint32_t channels, char **error_message) {
	if (sample_rate <= 0) {
		sample_rate = 48000.0;
	}
	if (channels == 0) {
		channels = 1;
	}

	rf_coreaudio_recorder *recorder = (rf_coreaudio_recorder *)calloc(1, sizeof(rf_coreaudio_recorder));
	if (recorder == NULL) {
		if (error_message != NULL) {
			*error_message = rf_coreaudio_strdup("CoreAudio recorder allocation failed");
		}
		return NULL;
	}
	recorder->handle = handle;
	recorder->format.mSampleRate = sample_rate;
	recorder->format.mFormatID = kAudioFormatLinearPCM;
	recorder->format.mFormatFlags = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked | kAudioFormatFlagsNativeEndian;
	recorder->format.mBytesPerPacket = channels * sizeof(float);
	recorder->format.mFramesPerPacket = 1;
	recorder->format.mBytesPerFrame = channels * sizeof(float);
	recorder->format.mChannelsPerFrame = channels;
	recorder->format.mBitsPerChannel = 32;
	recorder->buffer_byte_size = (UInt32)(sample_rate / 50.0) * recorder->format.mBytesPerFrame;
	if (recorder->buffer_byte_size < recorder->format.mBytesPerFrame) {
		recorder->buffer_byte_size = recorder->format.mBytesPerFrame * 256;
	}

	OSStatus status = AudioQueueNewInput(&recorder->format, rf_coreaudio_input_callback, recorder, NULL, NULL, 0, &recorder->queue);
	if (status != noErr) {
		rf_coreaudio_set_error(error_message, "AudioQueueNewInput failed", status);
		free(recorder);
		return NULL;
	}

	if (device_uid != NULL && strlen(device_uid) > 0) {
		CFStringRef uid = CFStringCreateWithCString(NULL, device_uid, kCFStringEncodingUTF8);
		if (uid == NULL) {
			if (error_message != NULL) {
				*error_message = rf_coreaudio_strdup("CoreAudio device UID conversion failed");
			}
			AudioQueueDispose(recorder->queue, true);
			free(recorder);
			return NULL;
		}
		status = AudioQueueSetProperty(recorder->queue, kAudioQueueProperty_CurrentDevice, &uid, sizeof(uid));
		CFRelease(uid);
		if (status != noErr) {
			rf_coreaudio_set_error(error_message, "AudioQueueSetProperty(CurrentDevice) failed", status);
			AudioQueueDispose(recorder->queue, true);
			free(recorder);
			return NULL;
		}
	}

	return recorder;
}

int rf_coreaudio_start(rf_coreaudio_recorder *recorder, char **error_message) {
	if (recorder == NULL || recorder->queue == NULL) {
		if (error_message != NULL) {
			*error_message = rf_coreaudio_strdup("CoreAudio recorder is nil");
		}
		return -1;
	}
	for (int index = 0; index < 3; index++) {
		OSStatus status = AudioQueueAllocateBuffer(recorder->queue, recorder->buffer_byte_size, &recorder->buffers[index]);
		if (status != noErr) {
			rf_coreaudio_set_error(error_message, "AudioQueueAllocateBuffer failed", status);
			return -1;
		}
		status = AudioQueueEnqueueBuffer(recorder->queue, recorder->buffers[index], 0, NULL);
		if (status != noErr) {
			rf_coreaudio_set_error(error_message, "AudioQueueEnqueueBuffer failed", status);
			return -1;
		}
	}
	recorder->running = 1;
	OSStatus status = AudioQueueStart(recorder->queue, NULL);
	if (status != noErr) {
		recorder->running = 0;
		rf_coreaudio_set_error(error_message, "AudioQueueStart failed", status);
		return -1;
	}
	return 0;
}

void rf_coreaudio_stop(rf_coreaudio_recorder *recorder) {
	if (recorder == NULL || recorder->queue == NULL) {
		return;
	}
	recorder->running = 0;
	AudioQueueStop(recorder->queue, true);
}

void rf_coreaudio_free(rf_coreaudio_recorder *recorder) {
	if (recorder == NULL) {
		return;
	}
	if (recorder->queue != NULL) {
		recorder->running = 0;
		AudioQueueDispose(recorder->queue, true);
		recorder->queue = NULL;
	}
	free(recorder);
}

void rf_coreaudio_free_string(char *value) {
	free(value);
}
