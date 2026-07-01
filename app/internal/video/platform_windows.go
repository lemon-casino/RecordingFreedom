//go:build windows

package video

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

type windowsGraphicsCaptureTargetKind string

const (
	windowsTargetScreen      windowsGraphicsCaptureTargetKind = "screen"
	windowsTargetAllScreens  windowsGraphicsCaptureTargetKind = "all-screens"
	windowsTargetRegion      windowsGraphicsCaptureTargetKind = "region"
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
	writer      *ffmpegDesktopSession

	mu      sync.Mutex
	started bool
	stopped bool
}

func NewPlatformSession(config CaptureConfig) (Session, error) {
	config = NormalizeCaptureConfig(config)
	if config.OutputPath == "" {
		return nil, errors.New("Windows desktop capture output path is required")
	}
	target, err := windowsGraphicsCaptureTargetForConfig(config)
	if err != nil {
		return nil, err
	}
	writer, err := newFFmpegDesktopSession(config, windowsFFmpegInputArgs(target))
	if err != nil {
		return nil, err
	}
	diagnostics := writer.Diagnostics()
	diagnostics.Messages = append(diagnostics.Messages, fmt.Sprintf("Windows desktop capture target resolved: %s", target))
	return &windowsGraphicsCaptureSession{
		config:      config,
		target:      target,
		diagnostics: diagnostics,
		writer:      writer,
	}, nil
}

func windowsGraphicsCaptureTargetForConfig(config CaptureConfig) (windowsGraphicsCaptureTarget, error) {
	switch config.SourceType {
	case devices.SourceScreen:
		screenID, ok := WindowsScreenID(config.SourceID)
		if !ok {
			return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows desktop capture screen source id %q is invalid", config.SourceID)
		}
		return windowsGraphicsCaptureTarget{Kind: windowsTargetScreen, ScreenID: screenID}, nil
	case devices.SourceAllScreens:
		return windowsGraphicsCaptureTarget{Kind: windowsTargetAllScreens, ScreenID: "virtual-desktop"}, nil
	case devices.SourceRegion:
		if config.SourceGeometry == nil || config.SourceGeometry.Width <= 0 || config.SourceGeometry.Height <= 0 {
			return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows desktop capture region source %q is missing source geometry", config.SourceID)
		}
		return windowsGraphicsCaptureTarget{Kind: windowsTargetRegion, ScreenID: "virtual-desktop"}, nil
	case devices.SourceWindow:
		hwnd, ok := WindowsWindowHWND(config.SourceID)
		if !ok {
			return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows desktop capture window source id %q is invalid", config.SourceID)
		}
		return windowsGraphicsCaptureTarget{Kind: windowsTargetWindow, HWND: hwnd}, nil
	case devices.SourceApplication:
		pid, ok := WindowsApplicationPID(config.SourceID)
		if !ok {
			return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows desktop capture application source id %q is invalid", config.SourceID)
		}
		return windowsGraphicsCaptureTarget{Kind: windowsTargetApplication, PID: pid}, nil
	default:
		return windowsGraphicsCaptureTarget{}, fmt.Errorf("Windows desktop capture recording does not support source type %q yet", config.SourceType)
	}
}

func (t windowsGraphicsCaptureTarget) String() string {
	switch t.Kind {
	case windowsTargetScreen:
		return fmt.Sprintf("screen:%s", t.ScreenID)
	case windowsTargetAllScreens:
		return "all-screens:virtual-desktop"
	case windowsTargetRegion:
		return "region:custom"
	case windowsTargetWindow:
		return fmt.Sprintf("window:%x", t.HWND)
	case windowsTargetApplication:
		return fmt.Sprintf("application:%d", t.PID)
	default:
		return string(t.Kind)
	}
}

func (s *windowsGraphicsCaptureSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("Windows desktop capture session is stopped")
	}
	if s.started {
		return errors.New("Windows desktop capture session is already started")
	}
	if s.target.Kind == windowsTargetApplication {
		s.diagnostics.Screen.Enabled = false
		s.diagnostics.Screen.Message = "Windows program capture is queued; select a locked window or screen source."
		return fmt.Errorf("Windows program capture is queued for %s; select a locked window or screen source", s.target)
	}
	if err := s.writer.Start(ctx); err != nil {
		s.diagnostics = s.writer.Diagnostics()
		return err
	}
	s.started = true
	s.diagnostics = s.writer.Diagnostics()
	return nil
}

func (s *windowsGraphicsCaptureSession) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer == nil || !s.started || s.stopped {
		return nil
	}
	if err := s.writer.Pause(); err != nil {
		s.diagnostics = s.writer.Diagnostics()
		return err
	}
	s.diagnostics = s.writer.Diagnostics()
	return nil
}

func (s *windowsGraphicsCaptureSession) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer == nil || !s.started || s.stopped {
		return nil
	}
	if err := s.writer.Resume(); err != nil {
		s.diagnostics = s.writer.Diagnostics()
		return err
	}
	s.diagnostics = s.writer.Diagnostics()
	return nil
}

func (s *windowsGraphicsCaptureSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}
	s.stopped = true
	if s.writer == nil {
		return WriteDiagnostics(s.config.DiagnosticsPath, s.diagnostics)
	}
	err := s.writer.Stop()
	s.diagnostics = s.writer.Diagnostics()
	return err
}

func (s *windowsGraphicsCaptureSession) Diagnostics() Diagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer != nil {
		s.diagnostics = s.writer.Diagnostics()
	}
	return s.diagnostics
}

func windowsFFmpegInputArgs(target windowsGraphicsCaptureTarget) ffmpegInputArgsBuilder {
	return func(config CaptureConfig) ([]string, error) {
		config = NormalizeCaptureConfig(config)
		args := []string{
			"-f", "gdigrab",
			"-framerate", fmt.Sprintf("%d", config.Profile.FPS),
			"-draw_mouse", ffmpegBool(config.Profile.CaptureCursor),
		}
		switch target.Kind {
		case windowsTargetScreen, windowsTargetAllScreens:
			args = append(args, windowsFFmpegGeometryArgs(config.SourceGeometry)...)
			args = append(args, "-i", "desktop")
			return args, nil
		case windowsTargetRegion:
			if config.SourceGeometry == nil || config.SourceGeometry.Width <= 0 || config.SourceGeometry.Height <= 0 {
				return nil, fmt.Errorf("Windows FFmpeg region capture requires source geometry")
			}
			args = append(args, windowsFFmpegGeometryArgs(config.SourceGeometry)...)
			args = append(args, "-i", "desktop")
			return args, nil
		case windowsTargetWindow:
			args = append(args, "-i", fmt.Sprintf("hwnd=%d", target.HWND))
			return args, nil
		case windowsTargetApplication:
			return nil, fmt.Errorf("Windows FFmpeg program capture is queued; select a locked window instead")
		default:
			return nil, fmt.Errorf("Windows FFmpeg target %q is not supported", target.Kind)
		}
	}
}

func windowsFFmpegGeometryArgs(geometry *SourceGeometry) []string {
	if geometry == nil || geometry.Width <= 0 || geometry.Height <= 0 {
		return nil
	}
	return []string{
		"-offset_x", fmt.Sprintf("%d", geometry.X),
		"-offset_y", fmt.Sprintf("%d", geometry.Y),
		"-video_size", fmt.Sprintf("%dx%d", geometry.Width, geometry.Height),
	}
}

func ffmpegBool(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
