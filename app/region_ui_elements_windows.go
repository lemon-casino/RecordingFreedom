//go:build windows

package main

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	uiaCoInitApartmentThreaded = 0x2
	uiaClsctxInprocServer      = 0x1
	uiaSFalse                  = 0x1
	uiaRPCChangedMode          = 0x80010106
	uiaMaxAncestorDepth        = 18
	uiaMaxSiblingScan          = 2048
)

var (
	uiaCLSIDCUIAutomation = windows.GUID{Data1: 0xFF48DBA4, Data2: 0x60EF, Data3: 0x4201, Data4: [8]byte{0xAA, 0x87, 0x54, 0x10, 0x3E, 0xEF, 0x59, 0x4E}}
	uiaIIDIUIAutomation   = windows.GUID{Data1: 0x30CBE57D, Data2: 0xD9D0, Data3: 0x452A, Data4: [8]byte{0xAB, 0x13, 0x7A, 0xC5, 0xAC, 0x48, 0x25, 0xEE}}

	uiaOle32                = windows.NewLazySystemDLL("ole32.dll")
	uiaProcCoInitializeEx   = uiaOle32.NewProc("CoInitializeEx")
	uiaProcCoUninitialize   = uiaOle32.NewProc("CoUninitialize")
	uiaProcCoCreateInstance = uiaOle32.NewProc("CoCreateInstance")

	uiaOleAut32          = windows.NewLazySystemDLL("oleaut32.dll")
	uiaProcSysStringLen  = uiaOleAut32.NewProc("SysStringLen")
	uiaProcSysFreeString = uiaOleAut32.NewProc("SysFreeString")

	uiaUser32                       = windows.NewLazySystemDLL("user32.dll")
	uiaProcEnumWindows              = uiaUser32.NewProc("EnumWindows")
	uiaProcGetDesktopWindow         = uiaUser32.NewProc("GetDesktopWindow")
	uiaProcGetShellWindow           = uiaUser32.NewProc("GetShellWindow")
	uiaProcClientToScreen           = uiaUser32.NewProc("ClientToScreen")
	uiaProcGetClientRect            = uiaUser32.NewProc("GetClientRect")
	uiaProcGetCursorPos             = uiaUser32.NewProc("GetCursorPos")
	uiaProcGetWindowTextLength      = uiaUser32.NewProc("GetWindowTextLengthW")
	uiaProcGetWindowText            = uiaUser32.NewProc("GetWindowTextW")
	uiaProcGetWindowRect            = uiaUser32.NewProc("GetWindowRect")
	uiaProcGetWindowThreadProcessID = uiaUser32.NewProc("GetWindowThreadProcessId")
	uiaProcIsIconic                 = uiaUser32.NewProc("IsIconic")
	uiaProcIsWindowVisible          = uiaUser32.NewProc("IsWindowVisible")
)

type uiaWalkerKind string

const (
	uiaContentWalker uiaWalkerKind = "content"
	uiaControlWalker uiaWalkerKind = "control"
	uiaRawWalker     uiaWalkerKind = "raw"
)

type uiaAutomation struct {
	lpVtbl *uiaAutomationVtbl
}

type uiaAutomationVtbl struct {
	QueryInterface              uintptr
	AddRef                      uintptr
	Release                     uintptr
	CompareElements             uintptr
	CompareRuntimeIds           uintptr
	GetRootElement              uintptr
	ElementFromHandle           uintptr
	ElementFromPoint            uintptr
	GetFocusedElement           uintptr
	GetRootElementBuildCache    uintptr
	ElementFromHandleBuildCache uintptr
	ElementFromPointBuildCache  uintptr
	GetFocusedElementBuildCache uintptr
	CreateTreeWalker            uintptr
	GetControlViewWalker        uintptr
	GetContentViewWalker        uintptr
	GetRawViewWalker            uintptr
	GetRawViewCondition         uintptr
	GetControlViewCondition     uintptr
	GetContentViewCondition     uintptr
}

type uiaAutomationElement struct {
	lpVtbl *uiaAutomationElementVtbl
}

type uiaAutomationElementVtbl struct {
	QueryInterface                 uintptr
	AddRef                         uintptr
	Release                        uintptr
	SetFocus                       uintptr
	GetRuntimeId                   uintptr
	FindFirst                      uintptr
	FindAll                        uintptr
	FindFirstBuildCache            uintptr
	FindAllBuildCache              uintptr
	BuildUpdatedCache              uintptr
	GetCurrentPropertyValue        uintptr
	GetCurrentPropertyValueEx      uintptr
	GetCachedPropertyValue         uintptr
	GetCachedPropertyValueEx       uintptr
	GetCurrentPatternAs            uintptr
	GetCachedPatternAs             uintptr
	GetCurrentPattern              uintptr
	GetCachedPattern               uintptr
	GetCachedParent                uintptr
	GetCachedChildren              uintptr
	GetCurrentProcessID            uintptr
	GetCurrentControlType          uintptr
	GetCurrentLocalizedControlType uintptr
	GetCurrentName                 uintptr
	GetCurrentAcceleratorKey       uintptr
	GetCurrentAccessKey            uintptr
	GetCurrentHasKeyboardFocus     uintptr
	GetCurrentIsKeyboardFocusable  uintptr
	GetCurrentIsEnabled            uintptr
	GetCurrentAutomationID         uintptr
	GetCurrentClassName            uintptr
	GetCurrentHelpText             uintptr
	GetCurrentCulture              uintptr
	GetCurrentIsControlElement     uintptr
	GetCurrentIsContentElement     uintptr
	GetCurrentIsPassword           uintptr
	GetCurrentNativeWindowHandle   uintptr
	GetCurrentItemType             uintptr
	GetCurrentIsOffscreen          uintptr
	GetCurrentOrientation          uintptr
	GetCurrentFrameworkID          uintptr
	GetCurrentIsRequiredForForm    uintptr
	GetCurrentItemStatus           uintptr
	GetCurrentBoundingRectangle    uintptr
}

type uiaAutomationTreeWalker struct {
	lpVtbl *uiaAutomationTreeWalkerVtbl
}

type uiaAutomationTreeWalkerVtbl struct {
	QueryInterface                  uintptr
	AddRef                          uintptr
	Release                         uintptr
	GetParentElement                uintptr
	GetFirstChildElement            uintptr
	GetLastChildElement             uintptr
	GetNextSiblingElement           uintptr
	GetPreviousSiblingElement       uintptr
	NormalizeElement                uintptr
	GetParentElementBuildCache      uintptr
	GetFirstChildElementBuildCache  uintptr
	GetLastChildElementBuildCache   uintptr
	GetNextSiblingElementBuildCache uintptr
}

type uiaRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type uiaWinRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type uiaWinPoint struct {
	X int32
	Y int32
}

type regionUIElementRect struct {
	Rect        image.Rectangle
	Label       string
	ControlType int32
	Source      string
}

func (s *RecordingFreedomService) regionElementCandidatesAtPoint(session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	capturePoint := mapRegionPointToCapturePoint(session, point)
	captureBounds := captureBoundsForRegionSession(session)
	absolutePoint := mapRegionPointToAbsolutePoint(session, point)
	buildCandidates := func(rects []regionUIElementRect, bounds image.Rectangle, hitPoint image.Point, mapRect func(image.Rectangle) RegionRect) []RegionSmartCandidate {
		if bounds.Empty() || len(rects) == 0 {
			return nil
		}
		candidates := make([]RegionSmartCandidate, 0, len(rects))
		seen := map[string]bool{}
		minWidth, minHeight := regionSmartCandidateMinimumSize(session)
		for index, element := range rects {
			rect := element.Rect.Intersect(bounds)
			if rect.Empty() || !hitPoint.In(rect) {
				continue
			}
			if regionUIRectCoversCapture(rect, bounds) {
				continue
			}
			relative := mapRect(rect)
			relative = clampRegionCandidateBounds(relative, session.Bounds)
			if relative.Width < minWidth || relative.Height < minHeight {
				continue
			}
			key := fmt.Sprintf("%d:%d:%d:%d", relative.X, relative.Y, relative.Width, relative.Height)
			if seen[key] {
				continue
			}
			seen[key] = true
			label := safeRegionElementLabel(element.Label, regionUIControlTypeName(element.ControlType))
			candidates = append(candidates, RegionSmartCandidate{
				ID:       fmt.Sprintf("element:%d:%d:%d:%d", rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy()),
				Kind:     regionSmartKindElement,
				Label:    label,
				SourceID: element.Source,
				Bounds:   relative,
				Score:    0.94 - float64(index)*0.015,
			})
		}
		return candidates
	}

	if !captureBounds.Empty() && capturePoint.In(captureBounds) {
		rects, err := collectRegionUIElementRects(capturePoint)
		if err == nil {
			candidates := buildCandidates(rects, captureBounds, capturePoint, func(rect image.Rectangle) RegionRect {
				return mapCaptureRectToRegionSelection(session, rect)
			})
			if len(candidates) > 0 {
				return candidates
			}
		}
	}

	if absolutePoint != capturePoint {
		bounds := boundsForRegionPoint(session, point)
		if !bounds.Empty() && absolutePoint.In(bounds) {
			rects, err := collectRegionUIElementRects(absolutePoint)
			if err == nil {
				return buildCandidates(rects, bounds, absolutePoint, func(rect image.Rectangle) RegionRect {
					return mapAbsoluteRectToRegionSelection(session, rect)
				})
			}
		}
	}
	return nil
}

func (s *RecordingFreedomService) regionWindowCandidatesAtPoint(session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	capturePoint := mapRegionPointToCapturePoint(session, point)
	captureBounds := captureBoundsForRegionPoint(session, point)
	absolutePoint := mapRegionPointToAbsolutePoint(session, point)
	buildCandidates := func(windows []uiaTopLevelWindowCandidate, bounds image.Rectangle, hitPoint image.Point, mapRect func(image.Rectangle) RegionRect) []RegionSmartCandidate {
		if bounds.Empty() || len(windows) == 0 {
			return nil
		}
		candidates := make([]RegionSmartCandidate, 0, len(windows))
		seen := map[string]bool{}
		minWidth, minHeight := regionSmartCandidateMinimumSize(session)
		for index, window := range windows {
			rect := window.Rect.Intersect(bounds)
			if rect.Empty() || !hitPoint.In(rect) || regionUIRectCoversCapture(rect, bounds) {
				continue
			}
			relative := mapRect(rect)
			relative = clampRegionCandidateBounds(relative, session.Bounds)
			if relative.Width < minWidth || relative.Height < minHeight {
				continue
			}
			key := fmt.Sprintf("%d:%d:%d:%d", relative.X, relative.Y, relative.Width, relative.Height)
			if seen[key] {
				continue
			}
			seen[key] = true
			candidates = append(candidates, RegionSmartCandidate{
				ID:       fmt.Sprintf("window:%x", window.HWND),
				Kind:     regionSmartKindWindow,
				Label:    safeRegionElementLabel(window.Title, "Window"),
				SourceID: fmt.Sprintf("hwnd:%x", window.HWND),
				Bounds:   relative,
				Score:    0.82 - float64(index)*0.01,
			})
		}
		return candidates
	}

	currentPID := uint32(os.Getpid())
	if !captureBounds.Empty() && capturePoint.In(captureBounds) {
		windows := uiaTopLevelWindowsAtPoint(capturePoint, currentPID)
		candidates := buildCandidates(windows, captureBounds, capturePoint, func(rect image.Rectangle) RegionRect {
			return mapCaptureRectToRegionSelection(session, rect)
		})
		if len(candidates) > 0 {
			return candidates
		}
	}

	if absolutePoint != capturePoint {
		bounds := boundsForRegionPoint(session, point)
		if !bounds.Empty() && absolutePoint.In(bounds) {
			windows := uiaTopLevelWindowsAtPoint(absolutePoint, currentPID)
			return buildCandidates(windows, bounds, absolutePoint, func(rect image.Rectangle) RegionRect {
				return mapAbsoluteRectToRegionSelection(session, rect)
			})
		}
	}
	return nil
}

func regionUIRectCoversCapture(rect image.Rectangle, capture image.Rectangle) bool {
	if capture.Empty() || rect.Empty() {
		return false
	}
	return rect.Min.X <= capture.Min.X+2 &&
		rect.Min.Y <= capture.Min.Y+2 &&
		rect.Max.X >= capture.Max.X-2 &&
		rect.Max.Y >= capture.Max.Y-2
}

func collectRegionUIElementRects(point image.Point) ([]regionUIElementRect, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	needsUninitialize, err := uiaCoInitialize()
	if err != nil {
		return nil, err
	}
	if needsUninitialize {
		defer uiaProcCoUninitialize.Call()
	}

	automation, err := uiaCreateAutomation()
	if err != nil {
		return nil, err
	}
	defer automation.release()

	currentPID := uint32(os.Getpid())
	var collected []regionUIElementRect
	var firstErr error
	windowClip := image.Rectangle{}
	if hwnd := uiaTopLevelWindowAtPoint(point, currentPID); hwnd != 0 {
		windowClip = uiaWindowClientRectangle(hwnd)
		if windowClip.Empty() {
			windowClip = uiaWindowRectangle(hwnd)
		}
		for _, kind := range []uiaWalkerKind{uiaContentWalker, uiaControlWalker, uiaRawWalker} {
			walker, err := automation.viewWalker(kind)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			rects, err := collectRegionUIElementRectsFromWindow(automation, walker, kind, hwnd, point)
			walker.release()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			collected = append(collected, rects...)
		}
	}

	for _, kind := range []uiaWalkerKind{uiaContentWalker, uiaControlWalker, uiaRawWalker} {
		walker, err := automation.viewWalker(kind)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		rects, err := collectRegionUIElementRectsFromPoint(automation, walker, kind, point, currentPID, windowClip)
		walker.release()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		collected = append(collected, rects...)
	}

	collected = normalizeRegionUIElementRects(collected, point)
	if len(collected) > 0 {
		return collected, nil
	}
	return nil, firstErr
}

func collectRegionUIElementRectsFromPoint(automation *uiaAutomation, walker *uiaAutomationTreeWalker, kind uiaWalkerKind, point image.Point, currentPID uint32, clip image.Rectangle) ([]regionUIElementRect, error) {
	element, err := automation.elementFromPoint(point)
	if err != nil {
		return nil, err
	}
	if element == nil {
		return nil, nil
	}

	if pid, err := element.currentProcessID(); err == nil && uint32(pid) == currentPID {
		element.release()
		return nil, nil
	}

	rects := make([]regionUIElementRect, 0, 12)
	current := element
	for depth := 0; current != nil && depth < uiaMaxAncestorDepth; depth++ {
		offscreen, _ := current.currentIsOffscreen()
		rect, rectErr := current.currentBoundingRectangle()
		label := current.currentName()
		controlType, _ := current.currentControlType()
		if !offscreen && rectErr == nil && rect.Dx() > 0 && rect.Dy() > 0 {
			if !clip.Empty() {
				rect = rect.Intersect(clip)
			}
			rects = append(rects, regionUIElementRect{
				Rect:        rect,
				Label:       label,
				ControlType: controlType,
				Source:      string(kind) + ":point",
			})
		}
		parent, parentErr := walker.parentElement(current)
		current.release()
		if parentErr != nil || parent == nil {
			break
		}
		current = parent
	}
	return rects, nil
}

func collectRegionUIElementRectsFromWindow(automation *uiaAutomation, walker *uiaAutomationTreeWalker, kind uiaWalkerKind, hwnd uintptr, point image.Point) ([]regionUIElementRect, error) {
	root, err := automation.elementFromHandle(hwnd)
	if err != nil || root == nil {
		return nil, err
	}
	defer root.release()

	windowRect := uiaWindowClientRectangle(hwnd)
	if windowRect.Empty() {
		windowRect = uiaWindowRectangle(hwnd)
	}
	path, heldElements := collectRegionUIElementPath(walker, root, point, windowRect, string(kind)+":window")
	for _, element := range heldElements {
		element.release()
	}
	if len(path) == 0 {
		return nil, nil
	}
	return path, nil
}

func collectRegionUIElementPath(walker *uiaAutomationTreeWalker, root *uiaAutomationElement, point image.Point, clip image.Rectangle, source string) ([]regionUIElementRect, []*uiaAutomationElement) {
	path := make([]regionUIElementRect, 0, 12)
	held := make([]*uiaAutomationElement, 0, 12)
	current := root
	for depth := 0; current != nil && depth < uiaMaxAncestorDepth; depth++ {
		if elementRect, ok := current.regionUIElementRect(point); ok {
			if !clip.Empty() {
				elementRect.Rect = elementRect.Rect.Intersect(clip)
			}
			if !elementRect.Rect.Empty() && point.In(elementRect.Rect) {
				elementRect.Source = source
				path = append(path, elementRect)
			}
		}
		child := walker.bestChildElementAtPoint(current, point, clip)
		if child == nil {
			break
		}
		held = append(held, child)
		current = child
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path, held
}

func normalizeRegionUIElementRects(rects []regionUIElementRect, point image.Point) []regionUIElementRect {
	if len(rects) == 0 {
		return nil
	}
	next := make([]regionUIElementRect, 0, len(rects))
	seen := map[string]bool{}
	for _, rect := range rects {
		if rect.Rect.Empty() || !point.In(rect.Rect) {
			continue
		}
		if rect.Rect.Dx() <= 0 || rect.Rect.Dy() <= 0 {
			continue
		}
		if rect.Rect.Dx() > 200000 || rect.Rect.Dy() > 200000 {
			continue
		}
		key := fmt.Sprintf("%d:%d:%d:%d", rect.Rect.Min.X, rect.Rect.Min.Y, rect.Rect.Dx(), rect.Rect.Dy())
		if seen[key] {
			continue
		}
		seen[key] = true
		next = append(next, rect)
	}
	sort.SliceStable(next, func(left, right int) bool {
		leftArea := next[left].Rect.Dx() * next[left].Rect.Dy()
		rightArea := next[right].Rect.Dx() * next[right].Rect.Dy()
		if leftArea != rightArea {
			return leftArea < rightArea
		}
		if next[left].Rect.Min.Y != next[right].Rect.Min.Y {
			return next[left].Rect.Min.Y > next[right].Rect.Min.Y
		}
		return next[left].Rect.Min.X > next[right].Rect.Min.X
	})
	return next
}

func uiaTopLevelWindowAtPoint(point image.Point, currentPID uint32) uintptr {
	windows := uiaTopLevelWindowsAtPoint(point, currentPID)
	if len(windows) == 0 {
		return 0
	}
	return windows[0].HWND
}

type uiaTopLevelWindowCandidate struct {
	HWND  uintptr
	Rect  image.Rectangle
	Title string
}

func uiaTopLevelWindowsAtPoint(point image.Point, currentPID uint32) []uiaTopLevelWindowCandidate {
	shellWindow, _, _ := uiaProcGetShellWindow.Call()
	desktopWindow, _, _ := uiaProcGetDesktopWindow.Call()
	windows := make([]uiaTopLevelWindowCandidate, 0, 4)
	callback := syscall.NewCallback(func(hwnd uintptr, data uintptr) uintptr {
		if hwnd == 0 || hwnd == shellWindow || hwnd == desktopWindow || !uiaIsVisibleTopLevelWindow(hwnd) {
			return 1
		}
		if pid := uiaWindowProcessID(hwnd); pid == 0 || pid == currentPID {
			return 1
		}
		rect := uiaWindowClientRectangle(hwnd)
		if rect.Empty() {
			rect = uiaWindowRectangle(hwnd)
		}
		if rect.Empty() || rect.Dx() < minRegionWidth || rect.Dy() < minRegionHeight || !point.In(rect) {
			return 1
		}
		windows = append(windows, uiaTopLevelWindowCandidate{
			HWND:  hwnd,
			Rect:  rect,
			Title: uiaWindowTitle(hwnd),
		})
		return 1
	})
	uiaProcEnumWindows.Call(callback, 0)
	return windows
}

func uiaIsVisibleTopLevelWindow(hwnd uintptr) bool {
	visible, _, _ := uiaProcIsWindowVisible.Call(hwnd)
	if visible == 0 {
		return false
	}
	iconic, _, _ := uiaProcIsIconic.Call(hwnd)
	return iconic == 0
}

func uiaWindowProcessID(hwnd uintptr) uint32 {
	var pid uint32
	uiaProcGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

func uiaWindowRectangle(hwnd uintptr) image.Rectangle {
	var rect uiaWinRect
	ok, _, _ := uiaProcGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 || rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return image.Rectangle{}
	}
	return image.Rect(int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom))
}

func uiaWindowTitle(hwnd uintptr) string {
	length, _, _ := uiaProcGetWindowTextLength.Call(hwnd)
	if length == 0 {
		return ""
	}
	if length > 512 {
		length = 512
	}
	buffer := make([]uint16, int(length)+1)
	copied, _, _ := uiaProcGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if copied == 0 {
		return ""
	}
	return strings.TrimSpace(string(utf16.Decode(buffer[:copied])))
}

func uiaCursorPosition() (image.Point, error) {
	var point uiaWinPoint
	ok, _, callErr := uiaProcGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if ok == 0 {
		return image.Point{}, fmt.Errorf("GetCursorPos failed: %w", callErr)
	}
	return image.Point{X: int(point.X), Y: int(point.Y)}, nil
}

func currentRegionCursorPoint(session RegionSelectionSession) (image.Point, bool) {
	cursor, err := uiaCursorPosition()
	if err != nil {
		return image.Point{}, false
	}
	if absolute, ok := regionCapturePointToAbsolutePoint(session.DisplayBounds, cursor); ok {
		point := image.Point{X: absolute.X - session.Bounds.X, Y: absolute.Y - session.Bounds.Y}
		if regionRectContainsPoint(RegionRect{Width: session.Bounds.Width, Height: session.Bounds.Height}, point) {
			return point, true
		}
	}
	point := image.Point{X: cursor.X - session.Bounds.X, Y: cursor.Y - session.Bounds.Y}
	if regionRectContainsPoint(RegionRect{Width: session.Bounds.Width, Height: session.Bounds.Height}, point) {
		return point, true
	}
	return image.Point{}, false
}

func uiaWindowClientRectangle(hwnd uintptr) image.Rectangle {
	var rect uiaWinRect
	ok, _, _ := uiaProcGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 || rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return image.Rectangle{}
	}
	topLeft := uiaWinPoint{X: rect.Left, Y: rect.Top}
	bottomRight := uiaWinPoint{X: rect.Right, Y: rect.Bottom}
	ok, _, _ = uiaProcClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&topLeft)))
	if ok == 0 {
		return image.Rectangle{}
	}
	ok, _, _ = uiaProcClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&bottomRight)))
	if ok == 0 {
		return image.Rectangle{}
	}
	return image.Rect(int(topLeft.X), int(topLeft.Y), int(bottomRight.X), int(bottomRight.Y))
}

func uiaCoInitialize() (bool, error) {
	hr, _, _ := uiaProcCoInitializeEx.Call(0, uiaCoInitApartmentThreaded)
	switch hr {
	case 0, uiaSFalse:
		return true, nil
	case uiaRPCChangedMode:
		return false, nil
	default:
		if uiaFailed(hr) {
			return false, fmt.Errorf("CoInitializeEx failed: %s", uiaHRESULTString(hr))
		}
		return true, nil
	}
}

func uiaCreateAutomation() (*uiaAutomation, error) {
	var automation *uiaAutomation
	hr, _, _ := uiaProcCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&uiaCLSIDCUIAutomation)),
		0,
		uiaClsctxInprocServer,
		uintptr(unsafe.Pointer(&uiaIIDIUIAutomation)),
		uintptr(unsafe.Pointer(&automation)),
	)
	if uiaFailed(hr) {
		return nil, fmt.Errorf("CoCreateInstance(CUIAutomation) failed: %s", uiaHRESULTString(hr))
	}
	if automation == nil {
		return nil, fmt.Errorf("CoCreateInstance(CUIAutomation) returned nil automation")
	}
	return automation, nil
}

func (a *uiaAutomation) elementFromPoint(point image.Point) (*uiaAutomationElement, error) {
	if a == nil {
		return nil, fmt.Errorf("IUIAutomation is nil")
	}
	var element *uiaAutomationElement
	hr, _, _ := syscall.SyscallN(
		a.lpVtbl.ElementFromPoint,
		uintptr(unsafe.Pointer(a)),
		uiaPackPoint(point),
		uintptr(unsafe.Pointer(&element)),
	)
	if uiaFailed(hr) {
		return nil, fmt.Errorf("IUIAutomation::ElementFromPoint failed: %s", uiaHRESULTString(hr))
	}
	return element, nil
}

func (a *uiaAutomation) elementFromHandle(hwnd uintptr) (*uiaAutomationElement, error) {
	if a == nil {
		return nil, fmt.Errorf("IUIAutomation is nil")
	}
	var element *uiaAutomationElement
	hr, _, _ := syscall.SyscallN(
		a.lpVtbl.ElementFromHandle,
		uintptr(unsafe.Pointer(a)),
		hwnd,
		uintptr(unsafe.Pointer(&element)),
	)
	if uiaFailed(hr) {
		return nil, fmt.Errorf("IUIAutomation::ElementFromHandle failed: %s", uiaHRESULTString(hr))
	}
	return element, nil
}

func (a *uiaAutomation) viewWalker(kind uiaWalkerKind) (*uiaAutomationTreeWalker, error) {
	if a == nil {
		return nil, fmt.Errorf("IUIAutomation is nil")
	}
	method := a.lpVtbl.GetContentViewWalker
	name := "ContentViewWalker"
	switch kind {
	case uiaControlWalker:
		method = a.lpVtbl.GetControlViewWalker
		name = "ControlViewWalker"
	case uiaRawWalker:
		method = a.lpVtbl.GetRawViewWalker
		name = "RawViewWalker"
	}
	var walker *uiaAutomationTreeWalker
	hr, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&walker)),
	)
	if uiaFailed(hr) {
		return nil, fmt.Errorf("IUIAutomation::get_%s failed: %s", name, uiaHRESULTString(hr))
	}
	return walker, nil
}

func (a *uiaAutomation) release() {
	if a != nil {
		syscall.SyscallN(a.lpVtbl.Release, uintptr(unsafe.Pointer(a)))
	}
}

func (e *uiaAutomationElement) currentProcessID() (int32, error) {
	var pid int32
	hr, _, _ := syscall.SyscallN(e.lpVtbl.GetCurrentProcessID, uintptr(unsafe.Pointer(e)), uintptr(unsafe.Pointer(&pid)))
	if uiaFailed(hr) {
		return 0, fmt.Errorf("IUIAutomationElement::get_CurrentProcessId failed: %s", uiaHRESULTString(hr))
	}
	return pid, nil
}

func (e *uiaAutomationElement) currentControlType() (int32, error) {
	var controlType int32
	hr, _, _ := syscall.SyscallN(e.lpVtbl.GetCurrentControlType, uintptr(unsafe.Pointer(e)), uintptr(unsafe.Pointer(&controlType)))
	if uiaFailed(hr) {
		return 0, fmt.Errorf("IUIAutomationElement::get_CurrentControlType failed: %s", uiaHRESULTString(hr))
	}
	return controlType, nil
}

func (e *uiaAutomationElement) currentIsOffscreen() (bool, error) {
	var offscreen int32
	hr, _, _ := syscall.SyscallN(e.lpVtbl.GetCurrentIsOffscreen, uintptr(unsafe.Pointer(e)), uintptr(unsafe.Pointer(&offscreen)))
	if uiaFailed(hr) {
		return true, fmt.Errorf("IUIAutomationElement::get_CurrentIsOffscreen failed: %s", uiaHRESULTString(hr))
	}
	return offscreen != 0, nil
}

func (e *uiaAutomationElement) currentName() string {
	var bstr uintptr
	hr, _, _ := syscall.SyscallN(e.lpVtbl.GetCurrentName, uintptr(unsafe.Pointer(e)), uintptr(unsafe.Pointer(&bstr)))
	if uiaFailed(hr) || bstr == 0 {
		return ""
	}
	defer uiaProcSysFreeString.Call(bstr)
	length, _, _ := uiaProcSysStringLen.Call(bstr)
	if length == 0 {
		return ""
	}
	raw := unsafe.Slice((*uint16)(unsafe.Pointer(bstr)), int(length))
	return strings.TrimSpace(string(utf16.Decode(raw)))
}

func (e *uiaAutomationElement) currentBoundingRectangle() (image.Rectangle, error) {
	var rect uiaRect
	hr, _, _ := syscall.SyscallN(e.lpVtbl.GetCurrentBoundingRectangle, uintptr(unsafe.Pointer(e)), uintptr(unsafe.Pointer(&rect)))
	if uiaFailed(hr) {
		return image.Rectangle{}, fmt.Errorf("IUIAutomationElement::get_CurrentBoundingRectangle failed: %s", uiaHRESULTString(hr))
	}
	if rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return image.Rectangle{}, fmt.Errorf("IUIAutomationElement returned empty rectangle")
	}
	return image.Rect(int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom)), nil
}

func (e *uiaAutomationElement) regionUIElementRect(point image.Point) (regionUIElementRect, bool) {
	offscreen, _ := e.currentIsOffscreen()
	if offscreen {
		return regionUIElementRect{}, false
	}
	rect, err := e.currentBoundingRectangle()
	if err != nil || rect.Dx() <= 0 || rect.Dy() <= 0 || !point.In(rect) {
		return regionUIElementRect{}, false
	}
	controlType, _ := e.currentControlType()
	return regionUIElementRect{
		Rect:        rect,
		Label:       e.currentName(),
		ControlType: controlType,
	}, true
}

func (e *uiaAutomationElement) release() {
	if e != nil {
		syscall.SyscallN(e.lpVtbl.Release, uintptr(unsafe.Pointer(e)))
	}
}

func (w *uiaAutomationTreeWalker) parentElement(element *uiaAutomationElement) (*uiaAutomationElement, error) {
	var parent *uiaAutomationElement
	hr, _, _ := syscall.SyscallN(
		w.lpVtbl.GetParentElement,
		uintptr(unsafe.Pointer(w)),
		uintptr(unsafe.Pointer(element)),
		uintptr(unsafe.Pointer(&parent)),
	)
	if uiaFailed(hr) {
		return nil, fmt.Errorf("IUIAutomationTreeWalker::GetParentElement failed: %s", uiaHRESULTString(hr))
	}
	return parent, nil
}

func (w *uiaAutomationTreeWalker) firstChildElement(element *uiaAutomationElement) (*uiaAutomationElement, error) {
	var child *uiaAutomationElement
	hr, _, _ := syscall.SyscallN(
		w.lpVtbl.GetFirstChildElement,
		uintptr(unsafe.Pointer(w)),
		uintptr(unsafe.Pointer(element)),
		uintptr(unsafe.Pointer(&child)),
	)
	if uiaFailed(hr) {
		return nil, fmt.Errorf("IUIAutomationTreeWalker::GetFirstChildElement failed: %s", uiaHRESULTString(hr))
	}
	return child, nil
}

func (w *uiaAutomationTreeWalker) nextSiblingElement(element *uiaAutomationElement) (*uiaAutomationElement, error) {
	var sibling *uiaAutomationElement
	hr, _, _ := syscall.SyscallN(
		w.lpVtbl.GetNextSiblingElement,
		uintptr(unsafe.Pointer(w)),
		uintptr(unsafe.Pointer(element)),
		uintptr(unsafe.Pointer(&sibling)),
	)
	if uiaFailed(hr) {
		return nil, fmt.Errorf("IUIAutomationTreeWalker::GetNextSiblingElement failed: %s", uiaHRESULTString(hr))
	}
	return sibling, nil
}

func (w *uiaAutomationTreeWalker) bestChildElementAtPoint(parent *uiaAutomationElement, point image.Point, clip image.Rectangle) *uiaAutomationElement {
	child, err := w.firstChildElement(parent)
	if err != nil || child == nil {
		return nil
	}
	var best *uiaAutomationElement
	bestArea := int(^uint(0) >> 1)
	for siblingCount := 0; child != nil && siblingCount < uiaMaxSiblingScan; siblingCount++ {
		next, _ := w.nextSiblingElement(child)
		if rect, ok := child.regionUIElementRect(point); ok {
			if !clip.Empty() {
				rect.Rect = rect.Rect.Intersect(clip)
			}
			area := rect.Rect.Dx() * rect.Rect.Dy()
			if area > 0 && point.In(rect.Rect) && area < bestArea {
				if best != nil {
					best.release()
				}
				best = child
				bestArea = area
			} else {
				child.release()
			}
		} else {
			child.release()
		}
		child = next
	}
	return best
}

func (w *uiaAutomationTreeWalker) release() {
	if w != nil {
		syscall.SyscallN(w.lpVtbl.Release, uintptr(unsafe.Pointer(w)))
	}
}

func uiaPackPoint(point image.Point) uintptr {
	return uintptr(uint32(int32(point.X))) | uintptr(uint64(uint32(int32(point.Y)))<<32)
}

func uiaFailed(hr uintptr) bool {
	return int32(hr) < 0
}

func uiaHRESULTString(hr uintptr) string {
	return fmt.Sprintf("HRESULT 0x%08X", uint32(hr))
}

func regionUIControlTypeName(controlType int32) string {
	switch controlType {
	case 50000:
		return "Button"
	case 50001:
		return "Calendar"
	case 50002:
		return "Check box"
	case 50003:
		return "Combo box"
	case 50004:
		return "Edit"
	case 50005:
		return "Hyperlink"
	case 50006:
		return "Image"
	case 50007:
		return "List item"
	case 50008:
		return "List"
	case 50009:
		return "Menu"
	case 50010:
		return "Menu bar"
	case 50011:
		return "Menu item"
	case 50012:
		return "Progress bar"
	case 50013:
		return "Radio button"
	case 50014:
		return "Scroll bar"
	case 50015:
		return "Slider"
	case 50016:
		return "Spinner"
	case 50017:
		return "Status bar"
	case 50018:
		return "Tab"
	case 50019:
		return "Tab item"
	case 50020:
		return "Text"
	case 50021:
		return "Tool bar"
	case 50022:
		return "Tool tip"
	case 50023:
		return "Tree"
	case 50024:
		return "Tree item"
	case 50025:
		return "Custom"
	case 50026:
		return "Group"
	case 50027:
		return "Thumb"
	case 50028:
		return "Data grid"
	case 50029:
		return "Data item"
	case 50030:
		return "Document"
	case 50031:
		return "Split button"
	case 50032:
		return "Window"
	case 50033:
		return "Pane"
	case 50034:
		return "Header"
	case 50035:
		return "Header item"
	case 50036:
		return "Table"
	case 50037:
		return "Title bar"
	case 50038:
		return "Separator"
	case 50039:
		return "Semantic zoom"
	case 50040:
		return "App bar"
	default:
		return ""
	}
}
