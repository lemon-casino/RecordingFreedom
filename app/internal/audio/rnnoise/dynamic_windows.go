//go:build windows && rnnoise_dynamic

package rnnoise

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

	"golang.org/x/sys/windows"
)

const (
	dynamicDLLName = "rnnoise.dll"
	envDLLPath     = "RECORDINGFREEDOM_RNNOISE_PATH"
)

type dynamicLibrary struct {
	path               string
	dll                *windows.DLL
	createMilliGain    *windows.Proc
	destroy            *windows.Proc
	reset              *windows.Proc
	processFloat       *windows.Proc
	requiredSampleRate *windows.Proc
	frameSize          *windows.Proc
}

type Suppressor struct {
	ptr uintptr
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
	gainMilli := outputGainMilli(outputGain)
	ptr, _, callErr := loadLib.createMilliGain.Call(uintptr(RequiredSampleRate()), uintptr(1), uintptr(gainMilli))
	if ptr == 0 {
		return nil, fmt.Errorf("rnnoise DLL create failed at %s: %w", loadLib.path, callErr)
	}
	suppressor := &Suppressor{ptr: ptr, lib: loadLib}
	runtime.SetFinalizer(suppressor, (*Suppressor).Close)
	return suppressor, nil
}

func (s *Suppressor) Name() string {
	return "rnnoise-dynamic"
}

func (s *Suppressor) ProcessFrame(frame []float32) error {
	if s == nil || s.ptr == 0 || s.lib == nil {
		return errors.New("rnnoise suppressor is closed")
	}
	if len(frame) != FrameSize() {
		return fmt.Errorf("rnnoise requires %d samples per frame, got %d", FrameSize(), len(frame))
	}
	if len(frame) == 0 {
		return nil
	}
	ok, _, callErr := s.lib.processFloat.Call(
		s.ptr,
		uintptr(unsafe.Pointer(&frame[0])),
		uintptr(len(frame)),
		uintptr(1),
	)
	if ok == 0 {
		return fmt.Errorf("rnnoise DLL failed to process frame: %w", callErr)
	}
	return nil
}

func (s *Suppressor) Reset() error {
	if s == nil || s.ptr == 0 || s.lib == nil {
		return nil
	}
	s.lib.reset.Call(s.ptr)
	return nil
}

func (s *Suppressor) Close() {
	if s == nil || s.ptr == 0 || s.lib == nil {
		return
	}
	s.lib.destroy.Call(s.ptr)
	s.ptr = 0
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
		dll, err := windows.LoadDLL(candidate)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		lib, err := bindDynamicLibrary(candidate, dll)
		if err != nil {
			_ = dll.Release()
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		return lib, nil
	}
	if len(failures) == 0 {
		return nil, errors.New("rnnoise DLL was not found")
	}
	return nil, fmt.Errorf("rnnoise DLL is unavailable: %s", strings.Join(failures, " | "))
}

func bindDynamicLibrary(path string, dll *windows.DLL) (*dynamicLibrary, error) {
	lib := &dynamicLibrary{path: path, dll: dll}
	var err error
	if lib.createMilliGain, err = dll.FindProc("likely_voice_enhancer_create_milli_gain"); err != nil {
		return nil, err
	}
	if lib.destroy, err = dll.FindProc("likely_voice_enhancer_destroy"); err != nil {
		return nil, err
	}
	if lib.reset, err = dll.FindProc("likely_voice_enhancer_reset"); err != nil {
		return nil, err
	}
	if lib.processFloat, err = dll.FindProc("likely_voice_enhancer_process_interleaved_float"); err != nil {
		return nil, err
	}
	if lib.requiredSampleRate, err = dll.FindProc("likely_voice_enhancer_required_sample_rate"); err != nil {
		return nil, err
	}
	if lib.frameSize, err = dll.FindProc("likely_voice_enhancer_frame_size"); err != nil {
		return nil, err
	}
	return lib, nil
}

func dynamicLibraryCandidates() []string {
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
			if strings.EqualFold(existing, path) {
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
		add(filepath.Join(dir, "tools", dynamicDLLName))
		add(filepath.Join(dir, dynamicDLLName))
	}
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, "tools", dynamicDLLName))
		add(filepath.Join(cwd, "app", "tools", dynamicDLLName))
		add(filepath.Join(cwd, dynamicDLLName))
	}
	return candidates
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

func callIntProc(proc *windows.Proc) int {
	if proc == nil {
		return 0
	}
	value, _, _ := proc.Call()
	return int(value)
}
