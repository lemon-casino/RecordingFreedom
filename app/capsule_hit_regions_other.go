//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func (s *RecordingFreedomService) capsuleWindowWndProcInterceptor(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) (uintptr, bool) {
	return 0, false
}

func (s *RecordingFreedomService) applyCapsuleWindowRegion(state capsuleWindowHitRegionState) error {
	return nil
}

func (s *RecordingFreedomService) applyAnnotationOverlayHitRegions() error {
	return nil
}

func (s *RecordingFreedomService) applyFloatingPanelWindowRegion(state capsuleWindowHitRegionState) error {
	return nil
}

func (s *RecordingFreedomService) applyFloatingSelectWindowRegion(state capsuleWindowHitRegionState) error {
	return nil
}

func (s *RecordingFreedomService) applyWindowRegion(window *application.WebviewWindow, state capsuleWindowHitRegionState) error {
	return nil
}
