//go:build darwin && cgo

#pragma once

#include <stdint.h>

typedef struct rf_coreaudio_recorder rf_coreaudio_recorder;

rf_coreaudio_recorder *rf_coreaudio_new(uintptr_t handle, const char *device_uid, double sample_rate, uint32_t channels, char **error_message);
int rf_coreaudio_start(rf_coreaudio_recorder *recorder, char **error_message);
void rf_coreaudio_stop(rf_coreaudio_recorder *recorder);
void rf_coreaudio_free(rf_coreaudio_recorder *recorder);
void rf_coreaudio_free_string(char *value);
