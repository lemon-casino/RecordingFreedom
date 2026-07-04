//go:build windows

package main

import (
	"fmt"
	"image"
	"syscall"
	"time"
	"unsafe"
)

const (
	windowsInputMouse      = 0
	windowsMouseEventWheel = 0x0800
	windowsWheelDelta      = 120
)

type windowsPoint struct {
	X int32
	Y int32
}

type windowsMouseInput struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type windowsInput struct {
	Type uint32
	Mi   windowsMouseInput
}

var (
	user32DLL          = syscall.NewLazyDLL("user32.dll")
	user32SetCursorPos = user32DLL.NewProc("SetCursorPos")
	user32GetCursorPos = user32DLL.NewProc("GetCursorPos")
	user32SendInput    = user32DLL.NewProc("SendInput")
)

func scrollDownAtRect(rect image.Rectangle) error {
	if rect.Empty() {
		return fmt.Errorf("scrolling screenshot rectangle is empty")
	}
	centerX := rect.Min.X + rect.Dx()/2
	centerY := rect.Min.Y + rect.Dy()/2
	var original windowsPoint
	if ok, _, err := user32GetCursorPos.Call(uintptr(unsafe.Pointer(&original))); ok == 0 {
		return fmt.Errorf("read cursor position: %w", err)
	}
	if ok, _, err := user32SetCursorPos.Call(uintptr(centerX), uintptr(centerY)); ok == 0 {
		return fmt.Errorf("move cursor to scrolling screenshot target: %w", err)
	}
	defer user32SetCursorPos.Call(uintptr(original.X), uintptr(original.Y))
	time.Sleep(35 * time.Millisecond)
	ticks := maxInt(2, minInt(5, rect.Dy()/120))
	wheelData := int32(-windowsWheelDelta * ticks)
	input := windowsInput{
		Type: windowsInputMouse,
		Mi: windowsMouseInput{
			MouseData: uint32(wheelData),
			DwFlags:   windowsMouseEventWheel,
		},
	}
	sent, _, err := user32SendInput.Call(
		1,
		uintptr(unsafe.Pointer(&input)),
		unsafe.Sizeof(input),
	)
	if sent != 1 {
		return fmt.Errorf("send scrolling screenshot wheel input: %w", err)
	}
	return nil
}
