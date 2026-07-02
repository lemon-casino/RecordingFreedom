//go:build windows

package video

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	winapi "golang.org/x/sys/windows"
)

const (
	cursorOverlayClassNamePrefix = "RecordingFreedomStableCursorOverlay"
	cursorOverlayStartTimeout    = 2 * time.Second
	cursorOverlayStopTimeout     = 2 * time.Second
	cursorOverlayMinFPS          = 15
	cursorOverlayMaxFPS          = 60
	cursorOverlayFallbackSize    = 32
	cursorOverlayMaxSize         = 256

	cursorShowing = 0x00000001

	cursorWsPopup         = 0x80000000
	cursorWsExTopmost     = 0x00000008
	cursorWsExToolWin     = 0x00000080
	cursorWsExLayered     = 0x00080000
	cursorWsExTransparent = 0x00000020
	cursorWsExNoActivate  = 0x08000000

	cursorSwHide         = 0
	cursorSwShowNoActive = 4

	cursorHwndTopmost = ^uintptr(0)

	cursorSwpNoSize     = 0x0001
	cursorSwpNoMove     = 0x0002
	cursorSwpNoActivate = 0x0010
	cursorSwpShowWindow = 0x0040

	cursorUlwAlpha   = 0x00000002
	cursorAcSrcOver  = 0x00
	cursorAcSrcAlpha = 0x01

	cursorDibRGBColors = 0
	cursorBiRGB        = 0
	cursorDiNormal     = 0x0003

	cursorSmCXCursor = 13
	cursorSmCYCursor = 14

	cursorWmNCHitTest   = 0x0084
	cursorHtTransparent = ^uintptr(0)

	cursorErrorClassAlreadyExists syscall.Errno = 1410
)

var (
	cursorUser32 = winapi.NewLazySystemDLL("user32.dll")
	cursorGDI32  = winapi.NewLazySystemDLL("gdi32.dll")
	cursorKernel = winapi.NewLazySystemDLL("kernel32.dll")

	cursorProcRegisterClassExW    = cursorUser32.NewProc("RegisterClassExW")
	cursorProcCreateWindowExW     = cursorUser32.NewProc("CreateWindowExW")
	cursorProcDestroyWindow       = cursorUser32.NewProc("DestroyWindow")
	cursorProcDefWindowProcW      = cursorUser32.NewProc("DefWindowProcW")
	cursorProcShowWindow          = cursorUser32.NewProc("ShowWindow")
	cursorProcSetWindowPos        = cursorUser32.NewProc("SetWindowPos")
	cursorProcGetCursorInfo       = cursorUser32.NewProc("GetCursorInfo")
	cursorProcGetIconInfo         = cursorUser32.NewProc("GetIconInfo")
	cursorProcDrawIconEx          = cursorUser32.NewProc("DrawIconEx")
	cursorProcGetDC               = cursorUser32.NewProc("GetDC")
	cursorProcReleaseDC           = cursorUser32.NewProc("ReleaseDC")
	cursorProcUpdateLayeredWindow = cursorUser32.NewProc("UpdateLayeredWindow")
	cursorProcGetSystemMetrics    = cursorUser32.NewProc("GetSystemMetrics")
	cursorProcPeekMessageW        = cursorUser32.NewProc("PeekMessageW")
	cursorProcTranslateMessage    = cursorUser32.NewProc("TranslateMessage")
	cursorProcDispatchMessageW    = cursorUser32.NewProc("DispatchMessageW")

	cursorProcCreateCompatibleDC = cursorGDI32.NewProc("CreateCompatibleDC")
	cursorProcCreateDIBSection   = cursorGDI32.NewProc("CreateDIBSection")
	cursorProcDeleteDC           = cursorGDI32.NewProc("DeleteDC")
	cursorProcDeleteObject       = cursorGDI32.NewProc("DeleteObject")
	cursorProcSelectObject       = cursorGDI32.NewProc("SelectObject")
	cursorProcGetObjectW         = cursorGDI32.NewProc("GetObjectW")

	cursorProcGetModuleHandleW = cursorKernel.NewProc("GetModuleHandleW")
)

type windowsCursorOverlay struct {
	fps int

	mu   sync.Mutex
	stop chan struct{}
	done chan error
}

type cursorPoint struct {
	X int32
	Y int32
}

type cursorSize struct {
	CX int32
	CY int32
}

type cursorBlendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type cursorInfo struct {
	CBSize      uint32
	Flags       uint32
	HCursor     uintptr
	PTScreenPos cursorPoint
}

type cursorIconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type cursorBitmap struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     uint16
	BitsPixel  uint16
	Bits       uintptr
}

type cursorWndClassEx struct {
	CBSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type cursorMsg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       cursorPoint
	LPrivate uint32
}

type cursorBitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type cursorRGBQuad struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

type cursorBitmapInfo struct {
	Header cursorBitmapInfoHeader
	Colors [1]cursorRGBQuad
}

func newWindowsCursorOverlay(fps int) *windowsCursorOverlay {
	if fps < cursorOverlayMinFPS {
		fps = cursorOverlayMinFPS
	}
	if fps > cursorOverlayMaxFPS {
		fps = cursorOverlayMaxFPS
	}
	return &windowsCursorOverlay{fps: fps}
}

func (o *windowsCursorOverlay) Start() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	if o.stop != nil {
		o.mu.Unlock()
		return nil
	}
	stop := make(chan struct{})
	done := make(chan error, 1)
	ready := make(chan error, 1)
	o.stop = stop
	o.done = done
	o.mu.Unlock()

	go o.run(stop, ready, done)

	select {
	case err := <-ready:
		if err != nil {
			o.mu.Lock()
			if o.stop == stop {
				o.stop = nil
				o.done = nil
			}
			o.mu.Unlock()
			return err
		}
		return nil
	case <-time.After(cursorOverlayStartTimeout):
		_ = o.Stop()
		return fmt.Errorf("cursor overlay did not become ready within %s", cursorOverlayStartTimeout)
	}
}

func (o *windowsCursorOverlay) Stop() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	stop := o.stop
	done := o.done
	if stop == nil || done == nil {
		o.mu.Unlock()
		return nil
	}
	o.stop = nil
	o.done = nil
	close(stop)
	o.mu.Unlock()

	select {
	case err := <-done:
		return err
	case <-time.After(cursorOverlayStopTimeout):
		return fmt.Errorf("cursor overlay did not stop within %s", cursorOverlayStopTimeout)
	}
}

func (o *windowsCursorOverlay) run(stop <-chan struct{}, ready chan<- error, done chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hwnd, cleanup, err := createWindowsCursorOverlayWindow()
	if err != nil {
		ready <- err
		done <- err
		return
	}
	defer cleanup()

	ready <- nil
	ticker := time.NewTicker(time.Second / time.Duration(o.fps))
	defer ticker.Stop()

	for {
		pumpWindowsCursorOverlayMessages()
		select {
		case <-stop:
			hideWindowsCursorOverlay(hwnd)
			done <- nil
			return
		case <-ticker.C:
			_ = updateWindowsCursorOverlay(hwnd)
		}
	}
}

func createWindowsCursorOverlayWindow() (uintptr, func(), error) {
	className, err := winapi.UTF16PtrFromString(fmt.Sprintf("%s-%d", cursorOverlayClassNamePrefix, os.Getpid()))
	if err != nil {
		return 0, nil, err
	}
	instance, _, _ := cursorProcGetModuleHandleW.Call(0)
	class := cursorWndClassEx{
		CBSize:    uint32(unsafe.Sizeof(cursorWndClassEx{})),
		WndProc:   syscall.NewCallback(windowsCursorOverlayWndProc),
		Instance:  instance,
		ClassName: className,
	}
	atom, _, registerErr := cursorProcRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 && registerErr != cursorErrorClassAlreadyExists {
		return 0, nil, fmt.Errorf("register cursor overlay window class: %w", registerErr)
	}

	exStyle := uintptr(cursorWsExLayered | cursorWsExTransparent | cursorWsExNoActivate | cursorWsExToolWin | cursorWsExTopmost)
	hwnd, _, createErr := cursorProcCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(className)),
		cursorWsPopup,
		0,
		0,
		1,
		1,
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		return 0, nil, fmt.Errorf("create cursor overlay window: %w", createErr)
	}
	cursorProcSetWindowPos.Call(hwnd, cursorHwndTopmost, 0, 0, 0, 0, cursorSwpNoMove|cursorSwpNoSize|cursorSwpNoActivate|cursorSwpShowWindow)
	cleanup := func() {
		hideWindowsCursorOverlay(hwnd)
		cursorProcDestroyWindow.Call(hwnd)
	}
	return hwnd, cleanup, nil
}

func windowsCursorOverlayWndProc(hwnd uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	if message == cursorWmNCHitTest {
		return cursorHtTransparent
	}
	result, _, _ := cursorProcDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func pumpWindowsCursorOverlayMessages() {
	var msg cursorMsg
	for {
		ok, _, _ := cursorProcPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
		if ok == 0 {
			return
		}
		cursorProcTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		cursorProcDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func updateWindowsCursorOverlay(hwnd uintptr) error {
	info := cursorInfo{CBSize: uint32(unsafe.Sizeof(cursorInfo{}))}
	ok, _, err := cursorProcGetCursorInfo.Call(uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		hideWindowsCursorOverlay(hwnd)
		return fmt.Errorf("get cursor info: %w", err)
	}
	if info.Flags&cursorShowing == 0 || info.HCursor == 0 {
		hideWindowsCursorOverlay(hwnd)
		return nil
	}

	width, height, hotX, hotY := windowsCursorOverlayMetrics(info.HCursor)
	screenDC, _, err := cursorProcGetDC.Call(0)
	if screenDC == 0 {
		hideWindowsCursorOverlay(hwnd)
		return fmt.Errorf("get screen dc: %w", err)
	}
	defer cursorProcReleaseDC.Call(0, screenDC)

	memDC, _, err := cursorProcCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		hideWindowsCursorOverlay(hwnd)
		return fmt.Errorf("create cursor memory dc: %w", err)
	}
	defer cursorProcDeleteDC.Call(memDC)

	var bits uintptr
	bitmapInfo := cursorBitmapInfo{
		Header: cursorBitmapInfoHeader{
			Size:        uint32(unsafe.Sizeof(cursorBitmapInfoHeader{})),
			Width:       width,
			Height:      -height,
			Planes:      1,
			BitCount:    32,
			Compression: cursorBiRGB,
		},
	}
	bitmap, _, err := cursorProcCreateDIBSection.Call(
		screenDC,
		uintptr(unsafe.Pointer(&bitmapInfo)),
		cursorDibRGBColors,
		uintptr(unsafe.Pointer(&bits)),
		0,
		0,
	)
	if bitmap == 0 {
		hideWindowsCursorOverlay(hwnd)
		return fmt.Errorf("create cursor dib section: %w", err)
	}
	defer cursorProcDeleteObject.Call(bitmap)

	if bits != 0 {
		clear(unsafe.Slice((*byte)(unsafe.Pointer(bits)), int(width*height*4)))
	}
	oldBitmap, _, _ := cursorProcSelectObject.Call(memDC, bitmap)
	if oldBitmap != 0 {
		defer cursorProcSelectObject.Call(memDC, oldBitmap)
	}

	drawn, _, drawErr := cursorProcDrawIconEx.Call(memDC, 0, 0, info.HCursor, uintptr(width), uintptr(height), 0, 0, cursorDiNormal)
	if drawn == 0 {
		hideWindowsCursorOverlay(hwnd)
		return fmt.Errorf("draw cursor icon: %w", drawErr)
	}

	dst := cursorPoint{X: info.PTScreenPos.X - hotX, Y: info.PTScreenPos.Y - hotY}
	size := cursorSize{CX: width, CY: height}
	src := cursorPoint{}
	blend := cursorBlendFunction{
		BlendOp:             cursorAcSrcOver,
		SourceConstantAlpha: 255,
		AlphaFormat:         cursorAcSrcAlpha,
	}
	updated, _, updateErr := cursorProcUpdateLayeredWindow.Call(
		hwnd,
		screenDC,
		uintptr(unsafe.Pointer(&dst)),
		uintptr(unsafe.Pointer(&size)),
		memDC,
		uintptr(unsafe.Pointer(&src)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		cursorUlwAlpha,
	)
	if updated == 0 {
		hideWindowsCursorOverlay(hwnd)
		return fmt.Errorf("update cursor overlay window: %w", updateErr)
	}
	cursorProcShowWindow.Call(hwnd, cursorSwShowNoActive)
	return nil
}

func windowsCursorOverlayMetrics(cursor uintptr) (width int32, height int32, hotX int32, hotY int32) {
	width = windowsCursorMetric(cursorSmCXCursor)
	height = windowsCursorMetric(cursorSmCYCursor)
	if width <= 0 {
		width = cursorOverlayFallbackSize
	}
	if height <= 0 {
		height = cursorOverlayFallbackSize
	}

	icon := cursorIconInfo{}
	ok, _, _ := cursorProcGetIconInfo.Call(cursor, uintptr(unsafe.Pointer(&icon)))
	if ok != 0 {
		hotX = int32(icon.XHotspot)
		hotY = int32(icon.YHotspot)
		if icon.HbmColor != 0 {
			if bitmapWidth, bitmapHeight, hasSize := windowsCursorBitmapSize(icon.HbmColor, false); hasSize {
				width = bitmapWidth
				height = bitmapHeight
			}
		} else if icon.HbmMask != 0 {
			if bitmapWidth, bitmapHeight, hasSize := windowsCursorBitmapSize(icon.HbmMask, true); hasSize {
				width = bitmapWidth
				height = bitmapHeight
			}
		}
		if icon.HbmColor != 0 {
			cursorProcDeleteObject.Call(icon.HbmColor)
		}
		if icon.HbmMask != 0 {
			cursorProcDeleteObject.Call(icon.HbmMask)
		}
	}
	return clampWindowsCursorOverlaySize(width), clampWindowsCursorOverlaySize(height), hotX, hotY
}

func windowsCursorMetric(index int32) int32 {
	value, _, _ := cursorProcGetSystemMetrics.Call(uintptr(index))
	return int32(value)
}

func windowsCursorBitmapSize(bitmap uintptr, mask bool) (int32, int32, bool) {
	info := cursorBitmap{}
	got, _, _ := cursorProcGetObjectW.Call(bitmap, unsafe.Sizeof(info), uintptr(unsafe.Pointer(&info)))
	if got == 0 || info.Width <= 0 || info.Height <= 0 {
		return 0, 0, false
	}
	height := info.Height
	if mask {
		height = height / 2
	}
	if height <= 0 {
		return 0, 0, false
	}
	return info.Width, height, true
}

func clampWindowsCursorOverlaySize(value int32) int32 {
	if value <= 0 {
		return cursorOverlayFallbackSize
	}
	if value > cursorOverlayMaxSize {
		return cursorOverlayMaxSize
	}
	return value
}

func hideWindowsCursorOverlay(hwnd uintptr) {
	if hwnd != 0 {
		cursorProcShowWindow.Call(hwnd, cursorSwHide)
	}
}
