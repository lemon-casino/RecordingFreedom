//go:build windows

package main

import (
	"image"
	"os"
	"unsafe"

	winapi "golang.org/x/sys/windows"
)

const focusedDwmExtendedFrameBounds = 9

var (
	focusedUser32                       = winapi.NewLazySystemDLL("user32.dll")
	focusedDwmapi                       = winapi.NewLazySystemDLL("dwmapi.dll")
	focusedProcGetForegroundWindow      = focusedUser32.NewProc("GetForegroundWindow")
	focusedProcGetWindowRect            = focusedUser32.NewProc("GetWindowRect")
	focusedProcIsWindowVisible          = focusedUser32.NewProc("IsWindowVisible")
	focusedProcIsIconic                 = focusedUser32.NewProc("IsIconic")
	focusedProcGetWindowThreadProcessID = focusedUser32.NewProc("GetWindowThreadProcessId")
	focusedProcDwmGetWindowAttribute    = focusedDwmapi.NewProc("DwmGetWindowAttribute")
)

type focusedWinRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func detectFocusedWindowScreenshotRect() (image.Rectangle, bool) {
	hwnd, _, _ := focusedProcGetForegroundWindow.Call()
	if hwnd == 0 || focusedWindowProcessID(hwnd) == uint32(os.Getpid()) {
		return image.Rectangle{}, false
	}
	if !focusedWindowIsUsable(hwnd) {
		return image.Rectangle{}, false
	}
	if rect, ok := focusedWindowExtendedFrameBounds(hwnd); ok {
		return rect, true
	}
	return focusedWindowRect(hwnd)
}

func focusedWindowIsUsable(hwnd uintptr) bool {
	visible, _, _ := focusedProcIsWindowVisible.Call(hwnd)
	if visible == 0 {
		return false
	}
	iconic, _, _ := focusedProcIsIconic.Call(hwnd)
	return iconic == 0
}

func focusedWindowProcessID(hwnd uintptr) uint32 {
	var pid uint32
	focusedProcGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

func focusedWindowExtendedFrameBounds(hwnd uintptr) (image.Rectangle, bool) {
	var rect focusedWinRect
	result, _, _ := focusedProcDwmGetWindowAttribute.Call(
		hwnd,
		uintptr(focusedDwmExtendedFrameBounds),
		uintptr(unsafe.Pointer(&rect)),
		unsafe.Sizeof(rect),
	)
	if result != 0 {
		return image.Rectangle{}, false
	}
	return focusedRectToImage(rect)
}

func focusedWindowRect(hwnd uintptr) (image.Rectangle, bool) {
	var rect focusedWinRect
	ok, _, _ := focusedProcGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 {
		return image.Rectangle{}, false
	}
	return focusedRectToImage(rect)
}

func focusedRectToImage(rect focusedWinRect) (image.Rectangle, bool) {
	result := image.Rect(int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom))
	return result, !result.Empty()
}
