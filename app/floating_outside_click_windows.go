//go:build windows

package main

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

const (
	floatingMouseHookAction = 0
)

type floatingMouseHookStruct struct {
	Point       w32.POINT
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

var (
	floatingOutsideHookMu      sync.Mutex
	floatingOutsideMouseHook   w32.HHOOK
	floatingOutsideHookService *RecordingFreedomService
)

func floatingOutsideHookCallback(nCode int, wParam w32.WPARAM, lParam w32.LPARAM) w32.LRESULT {
	if nCode == floatingMouseHookAction && floatingMouseMessageIsDown(uintptr(wParam)) {
		floatingOutsideHookMu.Lock()
		service := floatingOutsideHookService
		floatingOutsideHookMu.Unlock()
		if service != nil {
			service.handleFloatingOutsideMouseDown(lParam)
		}
	}
	floatingOutsideHookMu.Lock()
	hook := floatingOutsideMouseHook
	floatingOutsideHookMu.Unlock()
	return w32.CallNextHookEx(hook, nCode, wParam, lParam)
}

func (s *RecordingFreedomService) updateFloatingOutsideClickWatcher() {
	if s == nil || s.app == nil {
		return
	}
	s.floatingMu.Lock()
	enabled := s.floatingPanelState.Visible || s.floatingSelectState.Visible
	s.floatingMu.Unlock()
	if enabled {
		s.startFloatingOutsideClickWatcher()
		return
	}
	s.stopFloatingOutsideClickWatcher()
}

func (s *RecordingFreedomService) startFloatingOutsideClickWatcher() {
	floatingOutsideHookMu.Lock()
	if floatingOutsideMouseHook != 0 {
		floatingOutsideHookService = s
		floatingOutsideHookMu.Unlock()
		return
	}
	floatingOutsideHookService = s
	floatingOutsideHookMu.Unlock()

	if err := application.InvokeSyncWithError(func() error {
		floatingOutsideHookMu.Lock()
		defer floatingOutsideHookMu.Unlock()
		if floatingOutsideMouseHook != 0 {
			return nil
		}
		hook := w32.SetWindowsHookEx(w32.WH_MOUSE_LL, floatingOutsideHookCallback, 0, 0)
		if hook == 0 {
			return fmt.Errorf("install floating outside-click hook failed")
		}
		floatingOutsideMouseHook = hook
		return nil
	}); err != nil {
		s.logEvent("floating-panel", "outside-click-hook-error", map[string]string{"error": err.Error()})
	}
}

func (s *RecordingFreedomService) stopFloatingOutsideClickWatcher() {
	if err := application.InvokeSyncWithError(func() error {
		floatingOutsideHookMu.Lock()
		defer floatingOutsideHookMu.Unlock()
		if floatingOutsideMouseHook == 0 {
			floatingOutsideHookService = nil
			return nil
		}
		hook := floatingOutsideMouseHook
		floatingOutsideMouseHook = 0
		floatingOutsideHookService = nil
		if !w32.UnhookWindowsHookEx(hook) {
			return fmt.Errorf("uninstall floating outside-click hook failed")
		}
		return nil
	}); err != nil && s != nil {
		s.logEvent("floating-panel", "outside-click-unhook-error", map[string]string{"error": err.Error()})
	}
}

func (s *RecordingFreedomService) handleFloatingOutsideMouseDown(lParam w32.LPARAM) {
	if s == nil || lParam == 0 {
		return
	}
	event := (*floatingMouseHookStruct)(unsafe.Pointer(lParam))
	x := int(event.Point.X)
	y := int(event.Point.Y)

	s.floatingMu.Lock()
	panelState := s.floatingPanelState
	selectState := s.floatingSelectState
	s.floatingMu.Unlock()
	panelVisible := panelState.Visible
	selectVisible := selectState.Visible
	if !panelVisible && !selectVisible {
		s.updateFloatingOutsideClickWatcher()
		return
	}
	if floatingClickInsideOpenGrace(panelState, selectState) {
		return
	}

	insideCapsule := s.floatingPointInsideWindowSurface(s.capsuleWindow, s.capsuleHitRegions, x, y)
	insidePanel := panelVisible && s.floatingPointInsideWindowSurface(s.floatingPanelWindow, s.floatingPanelRegions, x, y)
	insideSelect := selectVisible && s.floatingPointInsideWindowSurface(s.floatingSelectWindow, s.floatingSelectRegions, x, y)

	if insideSelect {
		return
	}
	if selectVisible && (insideCapsule || insidePanel) {
		go func() {
			_ = s.HideFloatingSelect(0)
		}()
		return
	}
	if insideCapsule || insidePanel {
		return
	}
	go func() {
		_ = s.HideFloatingPanel(0)
	}()
}

func floatingClickInsideOpenGrace(panel FloatingPanelState, selectState FloatingSelectState) bool {
	openedAt := panel.UpdatedAt
	if selectState.Visible && selectState.UpdatedAt.After(openedAt) {
		openedAt = selectState.UpdatedAt
	}
	if openedAt.IsZero() {
		return false
	}
	return time.Since(openedAt) < 220*time.Millisecond
}

func (s *RecordingFreedomService) floatingPointInsideWindowSurface(window *application.WebviewWindow, regions capsuleWindowHitRegions, screenX int, screenY int) bool {
	if window == nil {
		return false
	}
	nativeWindow := window.NativeWindow()
	if nativeWindow == nil {
		return false
	}
	hwnd := uintptr(nativeWindow)
	rect := w32.GetWindowRect(w32.HWND(hwnd))
	if rect == nil || screenX < int(rect.Left) || screenX >= int(rect.Right) || screenY < int(rect.Top) || screenY >= int(rect.Bottom) {
		return false
	}
	clientX, clientY, ok := floatingScreenPointToClient(hwnd, screenX, screenY)
	if !ok {
		return true
	}
	clientWidth, clientHeight, ok := capsuleClientSize(hwnd)
	if !ok {
		return true
	}
	handled, hit := regions.TestClientPoint(clientX, clientY, clientWidth, clientHeight)
	if !handled {
		return true
	}
	return hit
}

func floatingScreenPointToClient(hwnd uintptr, screenX int, screenY int) (int, int, bool) {
	point := capsuleWinPoint{X: int32(screenX), Y: int32(screenY)}
	ret, _, _ := capsuleProcScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&point)))
	if ret == 0 {
		return 0, 0, false
	}
	return int(point.X), int(point.Y), true
}

func floatingMouseMessageIsDown(message uintptr) bool {
	switch message {
	case w32.WM_LBUTTONDOWN, w32.WM_RBUTTONDOWN, w32.WM_MBUTTONDOWN, w32.WM_XBUTTONDOWN:
		return true
	default:
		return false
	}
}
