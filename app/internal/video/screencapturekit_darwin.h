#ifndef RECORDINGFREEDOM_SCREENCAPTUREKIT_DARWIN_H
#define RECORDINGFREEDOM_SCREENCAPTUREKIT_DARWIN_H

#include <stdint.h>

typedef struct RFSCKSession RFSCKSession;

enum {
	RF_SCK_TARGET_DISPLAY = 1,
	RF_SCK_TARGET_WINDOW = 2
};

typedef struct {
	int enabled;
	int width;
	int height;
	int frameRate;
	int64_t framesWritten;
	int64_t droppedFrames;
	int64_t appendFailures;
	int64_t startOffsetMs;
	int64_t endOffsetMs;
	int64_t durationMs;
	char *message;
} RFSCKDiagnostics;

RFSCKSession *rf_sck_session_create(
	int target_kind,
	uint32_t target_id,
	const char *output_path,
	int fps,
	int capture_cursor,
	const char *quality,
	char **error_message
);

int rf_sck_session_start(RFSCKSession *session, char **error_message);
int rf_sck_session_pause(RFSCKSession *session, char **error_message);
int rf_sck_session_resume(RFSCKSession *session, char **error_message);
int rf_sck_session_stop(RFSCKSession *session, char **error_message);
void rf_sck_session_diagnostics(RFSCKSession *session, RFSCKDiagnostics *diagnostics);
void rf_sck_session_release(RFSCKSession *session);
void rf_sck_free_string(char *value);

#endif
