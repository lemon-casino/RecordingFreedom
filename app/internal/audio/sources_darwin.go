//go:build darwin && cgo

package audio

/*
#cgo darwin LDFLAGS: -framework AudioToolbox -framework CoreAudio -framework CoreFoundation
#include "coreaudio_capture_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime/cgo"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type coreAudioCaptureSource struct {
	id         string
	kind       StreamKind
	deviceID   string
	gain       float64
	sampleRate int
	channels   int

	mu           sync.Mutex
	recorder     *C.rf_coreaudio_recorder
	handle       cgo.Handle
	handleActive bool
	onFrame      func(TimedPCMBuffer) error
	frameOffset  uint64
	err          error

	paused  atomic.Bool
	stopped atomic.Bool
}

func NewPlatformCaptureSources(config CaptureConfig) ([]CaptureSource, error) {
	config = normalizeCaptureConfig(config)
	if config.SystemAudio.Enabled {
		return nil, fmt.Errorf("CoreAudio system-audio capture is not used directly; ScreenCaptureKit owns macOS system audio capture")
	}
	if !config.Microphone.Enabled {
		return nil, fmt.Errorf("native audio capture backend %q has no enabled streams", config.Backend)
	}
	return []CaptureSource{
		newCoreAudioCaptureSource(config.Microphone.DeviceID, config.MicrophoneGain, config.TargetSampleRate),
	}, nil
}

func newCoreAudioCaptureSource(deviceID string, gain float64, sampleRate int) *coreAudioCaptureSource {
	if sampleRate <= 0 {
		sampleRate = RNNoiseSampleRate
	}
	if gain <= 0 {
		gain = 1
	}
	return &coreAudioCaptureSource{
		id:         "coreaudio:microphone",
		kind:       StreamMicrophone,
		deviceID:   deviceID,
		gain:       gain,
		sampleRate: sampleRate,
		channels:   1,
	}
}

func (s *coreAudioCaptureSource) ID() string {
	return s.id
}

func (s *coreAudioCaptureSource) Kind() StreamKind {
	return s.kind
}

func (s *coreAudioCaptureSource) Start(onFrame func(TimedPCMBuffer) error) error {
	if onFrame == nil {
		return fmt.Errorf("coreaudio source %q requires a frame callback", s.id)
	}
	s.mu.Lock()
	if s.recorder != nil {
		s.mu.Unlock()
		return fmt.Errorf("coreaudio source %q is already started", s.id)
	}
	s.stopped.Store(false)
	s.onFrame = onFrame
	s.handle = cgo.NewHandle(s)
	s.handleActive = true
	s.mu.Unlock()

	uid := C.CString(coreAudioDeviceUID(s.deviceID))
	defer C.free(unsafe.Pointer(uid))

	var errMessage *C.char
	recorder := C.rf_coreaudio_new(C.uintptr_t(s.handle), uid, C.double(s.sampleRate), C.uint(s.channels), &errMessage)
	if recorder == nil {
		err := consumeCoreAudioError(errMessage, "create CoreAudio recorder")
		s.deleteHandle()
		return err
	}
	if C.rf_coreaudio_start(recorder, &errMessage) != 0 {
		err := consumeCoreAudioError(errMessage, "start CoreAudio recorder")
		C.rf_coreaudio_free(recorder)
		s.deleteHandle()
		return err
	}

	s.mu.Lock()
	s.recorder = recorder
	s.frameOffset = 0
	s.mu.Unlock()
	return nil
}

func (s *coreAudioCaptureSource) Pause() error {
	s.paused.Store(true)
	return nil
}

func (s *coreAudioCaptureSource) Resume() error {
	s.paused.Store(false)
	return nil
}

func (s *coreAudioCaptureSource) Stop() error {
	if s.stopped.Swap(true) {
		return nil
	}
	s.mu.Lock()
	recorder := s.recorder
	s.recorder = nil
	s.onFrame = nil
	err := s.err
	s.mu.Unlock()

	if recorder != nil {
		C.rf_coreaudio_stop(recorder)
		C.rf_coreaudio_free(recorder)
	}
	s.deleteHandle()
	return err
}

func (s *coreAudioCaptureSource) deleteHandle() {
	s.mu.Lock()
	if s.handleActive {
		s.handle.Delete()
		s.handleActive = false
	}
	s.mu.Unlock()
}

func (s *coreAudioCaptureSource) emit(data unsafe.Pointer, byteCount int) {
	if data == nil || byteCount <= 0 || s.paused.Load() || s.stopped.Load() {
		return
	}
	sampleCount := byteCount / 4
	if sampleCount == 0 {
		return
	}
	raw := unsafe.Slice((*float32)(data), sampleCount)
	samples := make([]float32, sampleCount)
	copy(samples, raw)
	applyCoreAudioGain(samples, s.gain)

	s.mu.Lock()
	onFrame := s.onFrame
	frameCount := len(samples) / s.channels
	timestamp := time.Duration(s.frameOffset) * time.Second / time.Duration(s.sampleRate)
	duration := time.Duration(frameCount) * time.Second / time.Duration(s.sampleRate)
	s.frameOffset += uint64(frameCount)
	s.mu.Unlock()
	if onFrame == nil || frameCount == 0 {
		return
	}
	if err := onFrame(TimedPCMBuffer{
		Buffer: PCMBuffer{
			Kind:       StreamMicrophone,
			SampleRate: s.sampleRate,
			Channels:   s.channels,
			Samples:    samples,
		},
		Timestamp: timestamp,
		Duration:  duration,
	}); err != nil {
		s.setErr(err)
	}
}

func (s *coreAudioCaptureSource) setErr(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

func coreAudioDeviceUID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || deviceID == "microphone:default" || deviceID == "default" {
		return ""
	}
	const prefix = "microphone:coreaudio:"
	if strings.HasPrefix(deviceID, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(deviceID, prefix))
	}
	if strings.HasPrefix(deviceID, "microphone:") {
		return ""
	}
	return deviceID
}

func applyCoreAudioGain(samples []float32, gain float64) {
	if gain <= 0 || gain == 1 {
		return
	}
	for index, sample := range samples {
		value := float64(sample) * gain
		if value > 1 {
			value = 1
		} else if value < -1 {
			value = -1
		}
		samples[index] = float32(value)
	}
}

func consumeCoreAudioError(message *C.char, fallback string) error {
	if message == nil {
		return errors.New(fallback)
	}
	defer C.rf_coreaudio_free_string(message)
	text := strings.TrimSpace(C.GoString(message))
	if text == "" {
		return errors.New(fallback)
	}
	return fmt.Errorf("%s: %s", fallback, text)
}

//export rfCoreAudioInputCallback
func rfCoreAudioInputCallback(handle C.uintptr_t, data unsafe.Pointer, byteCount C.uint) {
	sourceHandle := cgo.Handle(handle)
	source, ok := sourceHandle.Value().(*coreAudioCaptureSource)
	if !ok || source == nil {
		return
	}
	source.emit(data, int(byteCount))
}
