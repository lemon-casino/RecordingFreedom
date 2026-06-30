//go:build cgo

package rnnoise

/*
#cgo CFLAGS: -std=c99 -O2 -D_GNU_SOURCE -I.
#cgo linux,darwin LDFLAGS: -lm
#include "likely_voice_enhancer.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

type Suppressor struct {
	ptr *C.LikelyVoiceEnhancer
}

func Available() bool {
	return true
}

func RequiredSampleRate() int {
	return int(C.likely_voice_enhancer_required_sample_rate())
}

func FrameSize() int {
	return int(C.likely_voice_enhancer_frame_size())
}

func New(outputGain float64) (*Suppressor, error) {
	if outputGain <= 0 {
		outputGain = 1
	}
	ptr := C.likely_voice_enhancer_create(C.int(RequiredSampleRate()), 1, C.float(outputGain))
	if ptr == nil {
		return nil, errors.New("rnnoise voice enhancer could not be created")
	}
	suppressor := &Suppressor{ptr: ptr}
	runtime.SetFinalizer(suppressor, (*Suppressor).Close)
	return suppressor, nil
}

func (s *Suppressor) Name() string {
	return "rnnoise-native"
}

func (s *Suppressor) ProcessFrame(frame []float32) error {
	if s == nil || s.ptr == nil {
		return errors.New("rnnoise suppressor is closed")
	}
	if len(frame) != FrameSize() {
		return fmt.Errorf("rnnoise requires %d samples per frame, got %d", FrameSize(), len(frame))
	}
	if len(frame) == 0 {
		return nil
	}
	ok := C.likely_voice_enhancer_process_interleaved_float(
		s.ptr,
		(*C.float)(unsafe.Pointer(&frame[0])),
		C.int(len(frame)),
		1,
	)
	if ok == 0 {
		return errors.New("rnnoise failed to process frame")
	}
	return nil
}

func (s *Suppressor) Reset() error {
	if s == nil || s.ptr == nil {
		return nil
	}
	C.likely_voice_enhancer_reset(s.ptr)
	return nil
}

func (s *Suppressor) Close() {
	if s == nil || s.ptr == nil {
		return
	}
	C.likely_voice_enhancer_destroy(s.ptr)
	s.ptr = nil
	runtime.SetFinalizer(s, nil)
}
