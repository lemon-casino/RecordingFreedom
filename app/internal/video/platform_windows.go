//go:build windows

package video

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
	winapi "golang.org/x/sys/windows"
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

type windowsMonitorBounds struct {
	Index  int
	X      int
	Y      int
	Width  int
	Height int
}

type windowsMonitorInfoEx struct {
	Size     uint32
	Monitor  windowsRect
	WorkArea windowsRect
	Flags    uint32
	Device   [32]uint16
}

type windowsRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

var (
	windowsVideoUser32                  = winapi.NewLazySystemDLL("user32.dll")
	windowsVideoProcEnumDisplayMonitors = windowsVideoUser32.NewProc("EnumDisplayMonitors")
	windowsVideoProcGetMonitorInfoW     = windowsVideoUser32.NewProc("GetMonitorInfoW")
	windowsDisplayNumberPattern         = regexp.MustCompile(`display(\d+)$`)
)

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
	return func(config CaptureConfig) (ffmpegInputSpec, error) {
		config = NormalizeCaptureConfig(config)
		switch target.Kind {
		case windowsTargetScreen:
			if input, ok := windowsDDAGrabInputSpec(config, target); ok {
				return input, nil
			}
			return windowsGDIGGrabInputSpec(config, target, "screen"), nil
		case windowsTargetAllScreens:
			if config.Profile.CaptureCursor {
				return windowsGDIGGrabInputSpec(config, target, "all-screens virtual desktop cursor capture"), nil
			}
			if input, ok := windowsDDAGrabInputSpec(config, target); ok {
				return input, nil
			}
			return windowsGDIGGrabInputSpec(config, target, "multi-display desktop fallback"), nil
		case windowsTargetRegion:
			if config.SourceGeometry == nil || config.SourceGeometry.Width <= 0 || config.SourceGeometry.Height <= 0 {
				return ffmpegInputSpec{}, fmt.Errorf("Windows FFmpeg region capture requires source geometry")
			}
			if input, ok := windowsDDAGrabInputSpec(config, target); ok {
				return input, nil
			}
			return windowsGDIGGrabInputSpec(config, target, "region fallback"), nil
		case windowsTargetWindow:
			return windowsGDIGGrabInputSpec(config, target, "locked window"), nil
		case windowsTargetApplication:
			return ffmpegInputSpec{}, fmt.Errorf("Windows FFmpeg program capture is queued; select a locked window instead")
		default:
			return ffmpegInputSpec{}, fmt.Errorf("Windows FFmpeg target %q is not supported", target.Kind)
		}
	}
}

func windowsDDAGrabInputSpec(config CaptureConfig, target windowsGraphicsCaptureTarget) (ffmpegInputSpec, bool) {
	geometry := config.SourceGeometry
	if geometry == nil || geometry.Width <= 0 || geometry.Height <= 0 {
		return ffmpegInputSpec{}, false
	}

	outputIndex, ok := windowsDDAGrabOutputIndex(config, target)
	if !ok {
		return ffmpegInputSpec{}, false
	}

	offsetX := 0
	offsetY := 0
	videoSize := fmt.Sprintf("%dx%d", geometry.Width, geometry.Height)
	switch target.Kind {
	case windowsTargetScreen:
		offsetX = 0
		offsetY = 0
	case windowsTargetRegion:
		bounds, hasBounds := windowsMonitorBoundsByIndex(geometry.DisplayIndex)
		if !hasBounds {
			return ffmpegInputSpec{}, false
		}
		offsetX = geometry.X - bounds.X
		offsetY = geometry.Y - bounds.Y
		if offsetX < 0 || offsetY < 0 || offsetX+geometry.Width > bounds.Width || offsetY+geometry.Height > bounds.Height {
			return ffmpegInputSpec{}, false
		}
	case windowsTargetAllScreens:
		monitors := windowsMonitorBoundsList()
		if len(monitors) == 0 {
			return ffmpegInputSpec{}, false
		}
		if len(monitors) > 1 {
			return windowsDDAGrabAllScreensInputSpec(config, monitors), true
		}
		offsetX = 0
		offsetY = 0
	default:
		return ffmpegInputSpec{}, false
	}

	filter := fmt.Sprintf(
		"ddagrab=output_idx=%d:framerate=%d:draw_mouse=%s:video_size=%s:offset_x=%d:offset_y=%d:output_fmt=bgra:dup_frames=1",
		outputIndex,
		config.Profile.FPS,
		ffmpegBool(config.Profile.CaptureCursor),
		videoSize,
		offsetX,
		offsetY,
	)
	return ffmpegInputSpec{
		Args:        []string{"-f", "lavfi", "-i", filter},
		VideoFilter: "hwdownload,format=bgra,pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p",
		Engine:      "windows-dda",
		Messages: []string{
			"Windows Desktop Duplication capture is enabled for stable cursor recording; GDI cursor drawing is bypassed for this source.",
		},
	}, true
}

func windowsDDAGrabAllScreensInputSpec(config CaptureConfig, monitors []windowsMonitorBounds) ffmpegInputSpec {
	minX, minY := monitors[0].X, monitors[0].Y
	for _, monitor := range monitors[1:] {
		if monitor.X < minX {
			minX = monitor.X
		}
		if monitor.Y < minY {
			minY = monitor.Y
		}
	}

	var graph strings.Builder
	layout := make([]string, 0, len(monitors))
	for index, monitor := range monitors {
		if index > 0 {
			graph.WriteByte(';')
		}
		fmt.Fprintf(
			&graph,
			"ddagrab=output_idx=%d:framerate=%d:draw_mouse=%s:video_size=%dx%d:offset_x=0:offset_y=0:output_fmt=bgra:dup_frames=1,hwdownload,format=bgra[v%d]",
			monitor.Index-1,
			config.Profile.FPS,
			ffmpegBool(config.Profile.CaptureCursor),
			monitor.Width,
			monitor.Height,
			index,
		)
		layout = append(layout, fmt.Sprintf("%d_%d", monitor.X-minX, monitor.Y-minY))
	}
	graph.WriteByte(';')
	for index := range monitors {
		fmt.Fprintf(&graph, "[v%d]", index)
	}
	fmt.Fprintf(
		&graph,
		"xstack=inputs=%d:layout=%s:fill=black,pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p[v]",
		len(monitors),
		strings.Join(layout, "|"),
	)

	return ffmpegInputSpec{
		Args:             []string{"-filter_complex", graph.String(), "-map", "[v]"},
		VideoPreFiltered: true,
		Engine:           "windows-dda",
		Messages: []string{
			fmt.Sprintf("Windows Desktop Duplication multi-output capture is enabled for %d displays; GDI cursor drawing is bypassed for all-screens capture.", len(monitors)),
		},
	}
}

func windowsDDAGrabOutputIndex(config CaptureConfig, target windowsGraphicsCaptureTarget) (int, bool) {
	if config.SourceGeometry != nil && config.SourceGeometry.DisplayIndex > 0 {
		return config.SourceGeometry.DisplayIndex - 1, true
	}
	if target.Kind == windowsTargetScreen {
		if match := windowsDisplayNumberPattern.FindStringSubmatch(target.ScreenID); len(match) == 2 {
			var displayNumber int
			if _, err := fmt.Sscanf(match[1], "%d", &displayNumber); err == nil && displayNumber > 0 {
				return displayNumber - 1, true
			}
		}
	}
	if target.Kind == windowsTargetAllScreens && len(windowsMonitorBoundsList()) == 1 {
		return 0, true
	}
	return 0, target.Kind == windowsTargetScreen
}

func windowsGDIGGrabInputSpec(config CaptureConfig, target windowsGraphicsCaptureTarget, label string) ffmpegInputSpec {
	args := []string{
		"-f", "gdigrab",
		"-framerate", fmt.Sprintf("%d", config.Profile.FPS),
		"-draw_mouse", ffmpegBool(config.Profile.CaptureCursor),
	}
	switch target.Kind {
	case windowsTargetWindow:
		args = append(args, "-i", fmt.Sprintf("hwnd=%d", target.HWND))
	default:
		args = append(args, windowsFFmpegGeometryArgs(config.SourceGeometry)...)
		args = append(args, "-i", "desktop")
	}
	messages := []string{}
	if config.Profile.CaptureCursor {
		messages = append(messages, fmt.Sprintf("GDI cursor drawing is only used for %s capture because this source cannot use a single-output Desktop Duplication input.", label))
		if target.Kind == windowsTargetAllScreens {
			messages = append(messages, "All-screens cursor recording uses one virtual-desktop input to avoid multi-output cursor flicker on mixed-resolution displays.")
		}
	}
	return ffmpegInputSpec{
		Args:     args,
		Engine:   "windows-gdi",
		Messages: messages,
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

func windowsMonitorBoundsByIndex(displayIndex int) (windowsMonitorBounds, bool) {
	if displayIndex <= 0 {
		return windowsMonitorBounds{}, false
	}
	for _, bounds := range windowsMonitorBoundsList() {
		if bounds.Index == displayIndex {
			return bounds, true
		}
	}
	return windowsMonitorBounds{}, false
}

func windowsMonitorBoundsList() []windowsMonitorBounds {
	monitors := make([]windowsMonitorBounds, 0, 4)
	callback := syscall.NewCallback(func(monitor uintptr, hdc uintptr, rect uintptr, data uintptr) uintptr {
		info := windowsMonitorInfoEx{Size: uint32(unsafe.Sizeof(windowsMonitorInfoEx{}))}
		ok, _, _ := windowsVideoProcGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(&info)))
		if ok == 0 {
			return 1
		}
		monitors = append(monitors, windowsMonitorBounds{
			Index:  len(monitors) + 1,
			X:      int(info.Monitor.Left),
			Y:      int(info.Monitor.Top),
			Width:  int(info.Monitor.Right - info.Monitor.Left),
			Height: int(info.Monitor.Bottom - info.Monitor.Top),
		})
		return 1
	})
	result, _, _ := windowsVideoProcEnumDisplayMonitors.Call(0, 0, callback, 0)
	if result == 0 {
		return nil
	}
	return monitors
}
