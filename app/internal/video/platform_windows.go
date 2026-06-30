//go:build windows

package video

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

type windowsGraphicsCaptureTargetKind string

const (
	windowsTargetScreen      windowsGraphicsCaptureTargetKind = "screen"
	windowsTargetWindow      windowsGraphicsCaptureTargetKind = "window"
	windowsTargetApplication windowsGraphicsCaptureTargetKind = "application"
)

type windowsGraphicsCaptureTarget struct {
	Kind     windowsGraphicsCaptureTargetKind
	ScreenID string
	HWND     uintptr
	PID      uint32
}

type windowsGraphicsCaptureSession struct {
	config      CaptureConfig
	target      windowsGraphicsCaptureTarget
	diagnostics Diagnostics

	mu      sync.Mutex
	started bool
	stopped bool
}

func NewPlatformSession(config CaptureConfig) (Session, error) {
	config = NormalizeCaptureConfig(config)
	if config.OutputPath == "" {
		return nil, errors.New("Windows.Graphics.Capture output path is required")
	}
	target, err := windowsGraphicsCaptureTargetForConfig(config)
	if err != nil {
		return nil, err
	}
	diagnostics := NewDiagnostics(config)
	diagnostics.Screen.Path = filepath.Base(config.OutputPath)
	diagnostics.Screen.Clock = "media-timestamp"
	diagnostics.Messages = append(diagnostics.Messages, fmt.Sprintf("Windows.Graphics.Capture target resolved: %s", target))
	return &windowsGraphicsCaptureSession{
		config:      config,
		target:      target,
		diagnostics: diagnostics,
	}, nil
}

func windowsGraphicsCaptureTargetForConfig(config CaptureConfig) (windowsGraphicsCaptureTarget, error) {
	switch config.SourceType {
	case devices.SourceScreen:
		screenID, ok := WindowsScreenID(config.SourceID)
		if !ok {
			return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows.Graphics.Capture screen source id %q is invalid", config.SourceID)
		}
		return windowsGraphicsCaptureTarget{Kind: windowsTargetScreen, ScreenID: screenID}, nil
	case devices.SourceWindow:
		hwnd, ok := WindowsWindowHWND(config.SourceID)
		if !ok {
			return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows.Graphics.Capture window source id %q is invalid", config.SourceID)
		}
		return windowsGraphicsCaptureTarget{Kind: windowsTargetWindow, HWND: hwnd}, nil
	case devices.SourceApplication:
		pid, ok := WindowsApplicationPID(config.SourceID)
		if !ok {
			return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows.Graphics.Capture application source id %q is invalid", config.SourceID)
		}
		return windowsGraphicsCaptureTarget{Kind: windowsTargetApplication, PID: pid}, nil
	default:
		return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows.Graphics.Capture recording does not support source type %q yet", config.SourceType)
	}
}

func (t windowsGraphicsCaptureTarget) String() string {
	switch t.Kind {
	case windowsTargetScreen:
		return fmt.Sprintf("screen:%s", t.ScreenID)
	case windowsTargetWindow:
		return fmt.Sprintf("window:%x", t.HWND)
	case windowsTargetApplication:
		return fmt.Sprintf("application:%d", t.PID)
	default:
		return string(t.Kind)
	}
}

func (s *windowsGraphicsCaptureSession) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("Windows.Graphics.Capture session is stopped")
	}
	if s.started {
		return errors.New("Windows.Graphics.Capture session is already started")
	}
	s.diagnostics.Screen.Enabled = false
	s.diagnostics.Screen.Message = "Windows.Graphics.Capture writer is not implemented yet; no media was written."
	return fmt.Errorf("Windows.Graphics.Capture writer is not implemented yet for %s", s.target)
}

func (s *windowsGraphicsCaptureSession) Pause() error {
	return nil
}

func (s *windowsGraphicsCaptureSession) Resume() error {
	return nil
}

func (s *windowsGraphicsCaptureSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}
	s.stopped = true
	return WriteDiagnostics(s.config.DiagnosticsPath, s.diagnostics)
}

func (s *windowsGraphicsCaptureSession) Diagnostics() Diagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.diagnostics
}
