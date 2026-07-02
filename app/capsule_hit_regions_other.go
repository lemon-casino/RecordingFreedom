//go:build !windows

package main

func (s *RecordingFreedomService) capsuleWindowWndProcInterceptor(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) (uintptr, bool) {
	return 0, false
}

func (s *RecordingFreedomService) applyCapsuleWindowRegion(state capsuleWindowHitRegionState) error {
	return nil
}
