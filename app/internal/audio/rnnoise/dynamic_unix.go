//go:build rnnoise_dynamic && cgo && (darwin || linux)

package rnnoise

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

typedef void* (*create_milli_gain_fn)(int, int, int);
typedef void (*destroy_fn)(void*);
typedef void (*reset_fn)(void*);
typedef int (*process_float_fn)(void*, float*, int, int);
typedef int (*int_fn)(void);

static void* rf_dlopen(const char* path) {
	return dlopen(path, RTLD_NOW | RTLD_LOCAL);
}

static void* rf_dlsym(void* handle, const char* name) {
	return dlsym(handle, name);
}

static const char* rf_dlerror(void) {
	return dlerror();
}

static int rf_call_int(void* fn) {
	return ((int_fn)fn)();
}

static void* rf_call_create_milli_gain(void* fn, int sample_rate, int channels, int output_gain_milli) {
	return ((create_milli_gain_fn)fn)(sample_rate, channels, output_gain_milli);
}

static void rf_call_destroy(void* fn, void* enhancer) {
	((destroy_fn)fn)(enhancer);
}

static void rf_call_reset(void* fn, void* enhancer) {
	((reset_fn)fn)(enhancer);
}

static int rf_call_process_float(void* fn, void* enhancer, float* samples, int frame_count, int channels) {
	return ((process_float_fn)fn)(enhancer, samples, frame_count, channels);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

const envDLLPath = "RECORDINGFREEDOM_RNNOISE_PATH"

type dynamicLibrary struct {
	path               string
	handle             unsafe.Pointer
	createMilliGain    unsafe.Pointer
	destroy            unsafe.Pointer
	reset              unsafe.Pointer
	processFloat       unsafe.Pointer
	requiredSampleRate unsafe.Pointer
	frameSize          unsafe.Pointer
}

type Suppressor struct {
	ptr unsafe.Pointer
	lib *dynamicLibrary
}

var (
	loadOnce sync.Once
	loadLib  *dynamicLibrary
	loadErr  error
)

func Available() bool {
	return loadDynamicLibrary() == nil
}

func RequiredSampleRate() int {
	if err := loadDynamicLibrary(); err == nil {
		if value := callIntProc(loadLib.requiredSampleRate); value > 0 {
			return value
		}
	}
	return 48000
}

func FrameSize() int {
	if err := loadDynamicLibrary(); err == nil {
		if value := callIntProc(loadLib.frameSize); value > 0 {
			return value
		}
	}
	return 480
}

func New(outputGain float64) (*Suppressor, error) {
	if err := loadDynamicLibrary(); err != nil {
		return nil, err
	}
	ptr := C.rf_call_create_milli_gain(
		loadLib.createMilliGain,
		C.int(RequiredSampleRate()),
		C.int(1),
		C.int(outputGainMilli(outputGain)),
	)
	if ptr == nil {
		return nil, fmt.Errorf("rnnoise dynamic module create failed at %s", loadLib.path)
	}
	suppressor := &Suppressor{ptr: ptr, lib: loadLib}
	runtime.SetFinalizer(suppressor, (*Suppressor).Close)
	return suppressor, nil
}

func (s *Suppressor) Name() string {
	return "rnnoise-dynamic"
}

func (s *Suppressor) ProcessFrame(frame []float32) error {
	if s == nil || s.ptr == nil || s.lib == nil {
		return errors.New("rnnoise suppressor is closed")
	}
	if len(frame) != FrameSize() {
		return fmt.Errorf("rnnoise requires %d samples per frame, got %d", FrameSize(), len(frame))
	}
	if len(frame) == 0 {
		return nil
	}
	ok := C.rf_call_process_float(
		s.lib.processFloat,
		s.ptr,
		(*C.float)(unsafe.Pointer(&frame[0])),
		C.int(len(frame)),
		C.int(1),
	)
	if ok == 0 {
		return errors.New("rnnoise dynamic module failed to process frame")
	}
	return nil
}

func (s *Suppressor) Reset() error {
	if s == nil || s.ptr == nil || s.lib == nil {
		return nil
	}
	C.rf_call_reset(s.lib.reset, s.ptr)
	return nil
}

func (s *Suppressor) Close() {
	if s == nil || s.ptr == nil || s.lib == nil {
		return
	}
	C.rf_call_destroy(s.lib.destroy, s.ptr)
	s.ptr = nil
	runtime.SetFinalizer(s, nil)
}

func loadDynamicLibrary() error {
	loadOnce.Do(func() {
		loadLib, loadErr = openDynamicLibrary()
	})
	return loadErr
}

func openDynamicLibrary() (*dynamicLibrary, error) {
	candidates := dynamicLibraryCandidates()
	var failures []string
	for _, candidate := range candidates {
		cPath := C.CString(candidate)
		handle := C.rf_dlopen(cPath)
		C.free(unsafe.Pointer(cPath))
		if handle == nil {
			failures = append(failures, candidate+": "+lastDLError())
			continue
		}
		lib, err := bindDynamicLibrary(candidate, handle)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		return lib, nil
	}
	if len(failures) == 0 {
		return nil, errors.New("rnnoise dynamic module was not found")
	}
	return nil, fmt.Errorf("rnnoise dynamic module is unavailable: %s", strings.Join(failures, " | "))
}

func bindDynamicLibrary(path string, handle unsafe.Pointer) (*dynamicLibrary, error) {
	lib := &dynamicLibrary{path: path, handle: handle}
	var err error
	if lib.createMilliGain, err = findSymbol(handle, "likely_voice_enhancer_create_milli_gain"); err != nil {
		return nil, err
	}
	if lib.destroy, err = findSymbol(handle, "likely_voice_enhancer_destroy"); err != nil {
		return nil, err
	}
	if lib.reset, err = findSymbol(handle, "likely_voice_enhancer_reset"); err != nil {
		return nil, err
	}
	if lib.processFloat, err = findSymbol(handle, "likely_voice_enhancer_process_interleaved_float"); err != nil {
		return nil, err
	}
	if lib.requiredSampleRate, err = findSymbol(handle, "likely_voice_enhancer_required_sample_rate"); err != nil {
		return nil, err
	}
	if lib.frameSize, err = findSymbol(handle, "likely_voice_enhancer_frame_size"); err != nil {
		return nil, err
	}
	return lib, nil
}

func findSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	cName := C.CString(name)
	symbol := C.rf_dlsym(handle, cName)
	C.free(unsafe.Pointer(cName))
	if symbol == nil {
		return nil, fmt.Errorf("missing symbol %s: %s", name, lastDLError())
	}
	return symbol, nil
}

func dynamicLibraryCandidates() []string {
	name := dynamicLibraryName()
	var candidates []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}

	if envPath := os.Getenv(envDLLPath); envPath != "" {
		add(envPath)
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(filepath.Join(dir, "tools", name))
		add(filepath.Join(dir, name))
	}
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, "tools", name))
		add(filepath.Join(cwd, "app", "tools", name))
		add(filepath.Join(cwd, name))
	}
	return candidates
}

func dynamicLibraryName() string {
	if runtime.GOOS == "darwin" {
		return "librnnoise.dylib"
	}
	return "librnnoise.so"
}

func outputGainMilli(outputGain float64) int {
	if outputGain <= 0 || math.IsNaN(outputGain) || math.IsInf(outputGain, 0) {
		outputGain = 1
	}
	if outputGain < 0.25 {
		outputGain = 0.25
	}
	if outputGain > 3.5 {
		outputGain = 3.5
	}
	return int(math.Round(outputGain * 1000))
}

func callIntProc(proc unsafe.Pointer) int {
	if proc == nil {
		return 0
	}
	return int(C.rf_call_int(proc))
}

func lastDLError() string {
	message := C.rf_dlerror()
	if message == nil {
		return "unknown dlopen error"
	}
	return C.GoString(message)
}
