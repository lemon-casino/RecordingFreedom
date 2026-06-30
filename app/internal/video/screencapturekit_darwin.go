//go:build darwin && cgo

package video

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc -fblocks -Wno-deprecated-declarations -mmacosx-version-min=12.3
#cgo darwin LDFLAGS: -framework ScreenCaptureKit -framework AVFoundation -framework CoreMedia -framework CoreVideo -framework Foundation -mmacosx-version-min=12.3
#include "screencapturekit_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

type screenCaptureKitSession struct {
	config      CaptureConfig
	targetKind  C.int
	targetID    uint32
	handle      *C.RFSCKSession
	diagnostics Diagnostics

	mu       sync.Mutex
	released bool
}

func NewPlatformSession(config CaptureConfig) (Session, error) {
	config = NormalizeCaptureConfig(config)
	targetKind, targetID, err := screenCaptureKitTarget(config)
	if err != nil {
		return nil, err
	}
	if config.OutputPath == "" {
		return nil, errors.New("ScreenCaptureKit output path is required")
	}
	diagnostics := NewDiagnostics(config)
	diagnostics.Screen.Path = filepath.Base(config.OutputPath)
	return &screenCaptureKitSession{
		config:      config,
		targetKind:  targetKind,
		targetID:    targetID,
		diagnostics: diagnostics,
	}, nil
}

func screenCaptureKitTarget(config CaptureConfig) (C.int, uint32, error) {
	switch config.SourceType {
	case devices.SourceScreen:
		displayID, ok := DarwinDisplayID(config.SourceID)
		if !ok {
			return 0, 0, fmt.Errorf("ScreenCaptureKit display source id %q is invalid", config.SourceID)
		}
		return C.RF_SCK_TARGET_DISPLAY, displayID, nil
	case devices.SourceWindow:
		windowID, ok := DarwinWindowID(config.SourceID)
		if !ok {
			return 0, 0, fmt.Errorf("ScreenCaptureKit window source id %q is invalid", config.SourceID)
		}
		return C.RF_SCK_TARGET_WINDOW, windowID, nil
	default:
		return 0, 0, fmt.Errorf("ScreenCaptureKit recording does not support source type %q yet", config.SourceType)
	}
}

func (s *screenCaptureKitSession) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.released {
		return errors.New("ScreenCaptureKit session has already been released")
	}
	if s.handle != nil {
		return errors.New("ScreenCaptureKit session is already started")
	}

	outputPath := C.CString(s.config.OutputPath)
	quality := C.CString(s.config.Profile.Quality)
	defer C.free(unsafe.Pointer(outputPath))
	defer C.free(unsafe.Pointer(quality))

	var errMessage *C.char
	handle := C.rf_sck_session_create(
		s.targetKind,
		C.uint32_t(s.targetID),
		outputPath,
		C.int(s.config.Profile.FPS),
		boolToCInt(s.config.Profile.CaptureCursor),
		quality,
		&errMessage,
	)
	if handle == nil {
		return consumeSCKError(errMessage, "create ScreenCaptureKit session")
	}
	if C.rf_sck_session_start(handle, &errMessage) == 0 {
		err := consumeSCKError(errMessage, "start ScreenCaptureKit session")
		C.rf_sck_session_release(handle)
		return err
	}
	s.handle = handle
	s.diagnostics.Messages = append(s.diagnostics.Messages, fmt.Sprintf("ScreenCaptureKit %s capture started.", s.config.SourceType))
	return nil
}

func (s *screenCaptureKitSession) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle == nil || s.released {
		return nil
	}
	var errMessage *C.char
	if C.rf_sck_session_pause(s.handle, &errMessage) == 0 {
		return consumeSCKError(errMessage, "pause ScreenCaptureKit session")
	}
	return nil
}

func (s *screenCaptureKitSession) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle == nil || s.released {
		return nil
	}
	var errMessage *C.char
	if C.rf_sck_session_resume(s.handle, &errMessage) == 0 {
		return consumeSCKError(errMessage, "resume ScreenCaptureKit session")
	}
	return nil
}

func (s *screenCaptureKitSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.released {
		return nil
	}
	var stopErr error
	if s.handle != nil {
		var errMessage *C.char
		if C.rf_sck_session_stop(s.handle, &errMessage) == 0 {
			stopErr = consumeSCKError(errMessage, "stop ScreenCaptureKit session")
		}
		s.patchDiagnosticsLocked()
		C.rf_sck_session_release(s.handle)
		s.handle = nil
	}
	s.released = true
	writeErr := WriteDiagnostics(s.config.DiagnosticsPath, s.diagnostics)
	return errors.Join(stopErr, writeErr)
}

func (s *screenCaptureKitSession) Diagnostics() Diagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle != nil && !s.released {
		s.patchDiagnosticsLocked()
	}
	return s.diagnostics
}

func (s *screenCaptureKitSession) patchDiagnosticsLocked() {
	var raw C.RFSCKDiagnostics
	C.rf_sck_session_diagnostics(s.handle, &raw)
	defer C.rf_sck_free_string(raw.message)

	s.diagnostics.Screen.Enabled = raw.enabled != 0
	s.diagnostics.Screen.Path = filepath.Base(s.config.OutputPath)
	s.diagnostics.Screen.Clock = "media-timestamp"
	s.diagnostics.Screen.Width = int(raw.width)
	s.diagnostics.Screen.Height = int(raw.height)
	s.diagnostics.Screen.FrameRate = int(raw.frameRate)
	s.diagnostics.Screen.FramesWritten = int64(raw.framesWritten)
	s.diagnostics.Screen.DroppedFrames = int64(raw.droppedFrames)
	s.diagnostics.Screen.AppendFailures = int64(raw.appendFailures)
	s.diagnostics.Screen.StartOffsetMs = int64(raw.startOffsetMs)
	s.diagnostics.Screen.EndOffsetMs = int64(raw.endOffsetMs)
	s.diagnostics.Screen.DurationMs = int64(raw.durationMs)
	if raw.message != nil {
		s.diagnostics.Screen.Message = C.GoString(raw.message)
	}
}

func consumeSCKError(value *C.char, fallback string) error {
	defer C.rf_sck_free_string(value)
	if value == nil {
		return errors.New(fallback)
	}
	message := C.GoString(value)
	if message == "" {
		message = fallback
	}
	return errors.New(message)
}

func boolToCInt(value bool) C.int {
	if value {
		return 1
	}
	return 0
}
