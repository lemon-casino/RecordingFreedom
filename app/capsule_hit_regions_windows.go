package main

import (
	"fmt"
	"math"
	"unsafe"

	winapi "golang.org/x/sys/windows"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	wmNCHitTest   = 0x0084
	htTransparent = ^uintptr(0)
	rgnOr         = 2
)

var (
	capsuleUser32             = winapi.NewLazySystemDLL("user32.dll")
	capsuleGDI32              = winapi.NewLazySystemDLL("gdi32.dll")
	capsuleProcScreenToClient = capsuleUser32.NewProc("ScreenToClient")
	capsuleProcGetClientRect  = capsuleUser32.NewProc("GetClientRect")
	capsuleProcSetWindowRgn   = capsuleUser32.NewProc("SetWindowRgn")
	capsuleProcCreateRectRgn  = capsuleGDI32.NewProc("CreateRectRgn")
	capsuleProcRoundRectRgn   = capsuleGDI32.NewProc("CreateRoundRectRgn")
	capsuleProcCombineRgn     = capsuleGDI32.NewProc("CombineRgn")
	capsuleProcDeleteObject   = capsuleGDI32.NewProc("DeleteObject")
)

type capsuleWinPoint struct {
	X int32
	Y int32
}

type capsuleWinRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func (s *RecordingFreedomService) capsuleWindowWndProcInterceptor(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) (uintptr, bool) {
	if msg != wmNCHitTest || s == nil {
		return 0, false
	}
	if s.capsuleWindow != nil {
		nativeWindow := s.capsuleWindow.NativeWindow()
		if nativeWindow != nil && hwnd == uintptr(nativeWindow) {
			return s.hitTestWindowRegions(hwnd, lParam, s.capsuleHitRegions)
		}
	}
	if s.annotationOverlay != nil {
		nativeWindow := s.annotationOverlay.NativeWindow()
		if nativeWindow != nil && hwnd == uintptr(nativeWindow) {
			return s.hitTestWindowRegions(hwnd, lParam, s.annotationHitRegions)
		}
	}
	return 0, false
}

func (s *RecordingFreedomService) hitTestWindowRegions(hwnd uintptr, lParam uintptr, regions capsuleWindowHitRegions) (uintptr, bool) {
	clientX, clientY, ok := capsuleClientPoint(hwnd, lParam)
	if !ok {
		return 0, false
	}
	clientWidth, clientHeight, ok := capsuleClientSize(hwnd)
	if !ok {
		return 0, false
	}
	handled, hit := regions.TestClientPoint(clientX, clientY, clientWidth, clientHeight)
	if !handled || hit {
		return 0, false
	}
	return htTransparent, true
}

func (s *RecordingFreedomService) applyCapsuleWindowRegion(state capsuleWindowHitRegionState) error {
	if s == nil || s.capsuleWindow == nil {
		return nil
	}
	return application.InvokeSyncWithError(func() error {
		nativeWindow := s.capsuleWindow.NativeWindow()
		if nativeWindow == nil {
			return nil
		}
		hwnd := uintptr(nativeWindow)
		if !state.enabled || len(state.regions) == 0 {
			return capsuleSetWindowRegion(hwnd, 0)
		}
		clientWidth, clientHeight, ok := capsuleClientSize(hwnd)
		if !ok {
			return nil
		}
		region, err := capsuleCreateWindowRegion(state, clientWidth, clientHeight)
		if err != nil {
			return err
		}
		if err := capsuleSetWindowRegion(hwnd, region); err != nil {
			capsuleDeleteObject(region)
			return err
		}
		return nil
	})
}

func (s *RecordingFreedomService) applyAnnotationOverlayHitRegions() error {
	return nil
}

func capsuleClientPoint(hwnd uintptr, lParam uintptr) (int, int, bool) {
	screenX := int32(int16(lParam & 0xffff))
	screenY := int32(int16((lParam >> 16) & 0xffff))
	point := capsuleWinPoint{X: screenX, Y: screenY}
	ret, _, _ := capsuleProcScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&point)))
	if ret == 0 {
		return 0, 0, false
	}
	return int(point.X), int(point.Y), true
}

func capsuleClientSize(hwnd uintptr) (int, int, bool) {
	rect := capsuleWinRect{}
	ret, _, _ := capsuleProcGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return 0, 0, false
	}
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func capsuleCreateWindowRegion(state capsuleWindowHitRegionState, clientWidth int, clientHeight int) (uintptr, error) {
	scaleX := float64(clientWidth) / state.viewportWidth
	scaleY := float64(clientHeight) / state.viewportHeight
	if !isSaneScale(scaleX) {
		scaleX = 1
	}
	if !isSaneScale(scaleY) {
		scaleY = 1
	}

	var combined uintptr
	for _, region := range state.regions {
		next := capsuleCreateScaledRegion(region, scaleX, scaleY)
		if next == 0 {
			continue
		}
		if combined == 0 {
			combined = next
			continue
		}
		ret, _, callErr := capsuleProcCombineRgn.Call(combined, combined, next, rgnOr)
		capsuleDeleteObject(next)
		if ret == 0 {
			capsuleDeleteObject(combined)
			return 0, fmt.Errorf("combine capsule window region failed: %w", callErr)
		}
	}
	if combined == 0 {
		return 0, fmt.Errorf("capsule window region has no drawable regions")
	}
	return combined, nil
}

func capsuleCreateScaledRegion(region CapsuleWindowHitRegion, scaleX float64, scaleY float64) uintptr {
	left := int32(math.Max(0, math.Floor(region.X*scaleX)))
	top := int32(math.Max(0, math.Floor(region.Y*scaleY)))
	right := int32(math.Ceil((region.X + region.Width) * scaleX))
	bottom := int32(math.Ceil((region.Y + region.Height) * scaleY))
	if right <= left || bottom <= top {
		return 0
	}
	radiusX := int32(math.Round(region.Radius * scaleX * 2))
	radiusY := int32(math.Round(region.Radius * scaleY * 2))
	if region.Kind == "pill" || region.Kind == "round-rect" || radiusX > 0 || radiusY > 0 {
		if radiusX <= 0 {
			radiusX = 1
		}
		if radiusY <= 0 {
			radiusY = 1
		}
		region, _, _ := capsuleProcRoundRectRgn.Call(
			uintptr(left),
			uintptr(top),
			uintptr(right),
			uintptr(bottom),
			uintptr(radiusX),
			uintptr(radiusY),
		)
		return region
	}
	regionHandle, _, _ := capsuleProcCreateRectRgn.Call(uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
	return regionHandle
}

func capsuleSetWindowRegion(hwnd uintptr, region uintptr) error {
	ret, _, callErr := capsuleProcSetWindowRgn.Call(hwnd, region, 1)
	if ret == 0 {
		return fmt.Errorf("set capsule window region failed: %w", callErr)
	}
	return nil
}

func capsuleDeleteObject(object uintptr) {
	if object != 0 {
		capsuleProcDeleteObject.Call(object)
	}
}
