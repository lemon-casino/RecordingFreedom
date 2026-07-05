//go:build darwin && cgo

package main

/*
#cgo darwin LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
	double x;
	double y;
	double width;
	double height;
	int pid;
	char role[96];
	char title[512];
} RFAXElementInfo;

typedef struct {
	uint32_t windowID;
	int pid;
	double x;
	double y;
	double width;
	double height;
	char owner[256];
	char title[512];
} RFAXWindowInfo;

static void rf_ax_copy_cfstring_to_buffer(CFStringRef value, char *out, int cap) {
	if (out == NULL || cap <= 0) {
		return;
	}
	out[0] = '\0';
	if (value == NULL) {
		return;
	}
	CFStringGetCString(value, out, cap, kCFStringEncodingUTF8);
}

static void rf_ax_copy_string_attribute(AXUIElementRef element, CFStringRef attribute, char *out, int cap) {
	if (out == NULL || cap <= 0) {
		return;
	}
	out[0] = '\0';
	if (element == NULL || attribute == NULL) {
		return;
	}
	CFTypeRef value = NULL;
	if (AXUIElementCopyAttributeValue(element, attribute, &value) != kAXErrorSuccess || value == NULL) {
		return;
	}
	if (CFGetTypeID(value) == CFStringGetTypeID()) {
		CFStringGetCString((CFStringRef)value, out, cap, kCFStringEncodingUTF8);
	}
	CFRelease(value);
}

static int rf_ax_is_trusted(void) {
	return AXIsProcessTrusted() ? 1 : 0;
}

static int rf_ax_cursor_position(double *x, double *y) {
	if (x == NULL || y == NULL) {
		return 0;
	}
	CGEventRef event = CGEventCreate(NULL);
	if (event == NULL) {
		return 0;
	}
	CGPoint point = CGEventGetLocation(event);
	CFRelease(event);
	*x = point.x;
	*y = point.y;
	return 1;
}

static int rf_ax_element_at_position(double x, double y, AXUIElementRef *out) {
	if (out == NULL) {
		return 0;
	}
	*out = NULL;
	AXUIElementRef system = AXUIElementCreateSystemWide();
	if (system == NULL) {
		return 0;
	}
	AXError error = AXUIElementCopyElementAtPosition(system, (float)x, (float)y, out);
	CFRelease(system);
	return error == kAXErrorSuccess && *out != NULL;
}

static AXUIElementRef rf_ax_copy_parent(AXUIElementRef element) {
	if (element == NULL) {
		return NULL;
	}
	CFTypeRef parent = NULL;
	if (AXUIElementCopyAttributeValue(element, kAXParentAttribute, &parent) != kAXErrorSuccess || parent == NULL) {
		return NULL;
	}
	return (AXUIElementRef)parent;
}

static int rf_ax_element_info(AXUIElementRef element, RFAXElementInfo *out) {
	if (element == NULL || out == NULL) {
		return 0;
	}
	memset(out, 0, sizeof(RFAXElementInfo));
	CFTypeRef positionValue = NULL;
	CFTypeRef sizeValue = NULL;
	CGPoint position = CGPointZero;
	CGSize size = CGSizeZero;
	if (AXUIElementCopyAttributeValue(element, kAXPositionAttribute, &positionValue) != kAXErrorSuccess || positionValue == NULL) {
		return 0;
	}
	if (AXUIElementCopyAttributeValue(element, kAXSizeAttribute, &sizeValue) != kAXErrorSuccess || sizeValue == NULL) {
		CFRelease(positionValue);
		return 0;
	}
	int ok = 0;
	if (CFGetTypeID(positionValue) == AXValueGetTypeID() &&
		CFGetTypeID(sizeValue) == AXValueGetTypeID() &&
		AXValueGetValue((AXValueRef)positionValue, kAXValueCGPointType, &position) &&
		AXValueGetValue((AXValueRef)sizeValue, kAXValueCGSizeType, &size) &&
		size.width > 0 &&
		size.height > 0) {
		out->x = position.x;
		out->y = position.y;
		out->width = size.width;
		out->height = size.height;
		pid_t pid = 0;
		if (AXUIElementGetPid(element, &pid) == kAXErrorSuccess) {
			out->pid = (int)pid;
		}
		rf_ax_copy_string_attribute(element, kAXRoleAttribute, out->role, sizeof(out->role));
		rf_ax_copy_string_attribute(element, kAXTitleAttribute, out->title, sizeof(out->title));
		if (out->title[0] == '\0') {
			rf_ax_copy_string_attribute(element, kAXDescriptionAttribute, out->title, sizeof(out->title));
		}
		ok = 1;
	}
	CFRelease(positionValue);
	CFRelease(sizeValue);
	return ok;
}

static AXUIElementRef rf_ax_copy_best_child_at_position(AXUIElementRef element, double x, double y) {
	if (element == NULL) {
		return NULL;
	}
	CFTypeRef childrenValue = NULL;
	if (AXUIElementCopyAttributeValue(element, kAXChildrenAttribute, &childrenValue) != kAXErrorSuccess || childrenValue == NULL) {
		return NULL;
	}
	if (CFGetTypeID(childrenValue) != CFArrayGetTypeID()) {
		CFRelease(childrenValue);
		return NULL;
	}
	CFArrayRef children = (CFArrayRef)childrenValue;
	CFIndex count = CFArrayGetCount(children);
	AXUIElementRef best = NULL;
	double bestArea = 9000000000000000000.0;
	for (CFIndex i = 0; i < count; i++) {
		AXUIElementRef child = (AXUIElementRef)CFArrayGetValueAtIndex(children, i);
		if (child == NULL) {
			continue;
		}
		RFAXElementInfo info;
		if (!rf_ax_element_info(child, &info)) {
			continue;
		}
		double right = info.x + info.width;
		double bottom = info.y + info.height;
		if (x < info.x || x >= right || y < info.y || y >= bottom) {
			continue;
		}
		double area = info.width * info.height;
		if (area > 0 && area < bestArea) {
			best = child;
			bestArea = area;
		}
	}
	if (best != NULL) {
		CFRetain(best);
	}
	CFRelease(childrenValue);
	return best;
}

static int rf_ax_dict_number_int(CFDictionaryRef dict, const void *key, int *out) {
	CFTypeRef value = CFDictionaryGetValue(dict, key);
	if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
		return 0;
	}
	return CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, out);
}

static int rf_ax_dict_number_u32(CFDictionaryRef dict, const void *key, uint32_t *out) {
	CFTypeRef value = CFDictionaryGetValue(dict, key);
	if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
		return 0;
	}
	return CFNumberGetValue((CFNumberRef)value, kCFNumberSInt32Type, out);
}

static int rf_ax_collect_windows(RFAXWindowInfo *out, int cap) {
	if (out == NULL || cap <= 0) {
		return 0;
	}
	CFArrayRef windows = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID
	);
	if (windows == NULL) {
		return 0;
	}
	CFIndex total = CFArrayGetCount(windows);
	int used = 0;
	for (CFIndex i = 0; i < total && used < cap; i++) {
		CFDictionaryRef dict = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
		if (dict == NULL) {
			continue;
		}
		int layer = 0;
		if (!rf_ax_dict_number_int(dict, kCGWindowLayer, &layer) || layer != 0) {
			continue;
		}
		CFDictionaryRef boundsDict = (CFDictionaryRef)CFDictionaryGetValue(dict, kCGWindowBounds);
		CGRect bounds = CGRectZero;
		if (boundsDict == NULL || !CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds)) {
			continue;
		}
		if (bounds.size.width < 1 || bounds.size.height < 1) {
			continue;
		}
		uint32_t windowID = 0;
		if (!rf_ax_dict_number_u32(dict, kCGWindowNumber, &windowID) || windowID == 0) {
			continue;
		}
		memset(&out[used], 0, sizeof(RFAXWindowInfo));
		out[used].windowID = windowID;
		rf_ax_dict_number_int(dict, kCGWindowOwnerPID, &out[used].pid);
		out[used].x = bounds.origin.x;
		out[used].y = bounds.origin.y;
		out[used].width = bounds.size.width;
		out[used].height = bounds.size.height;
		rf_ax_copy_cfstring_to_buffer((CFStringRef)CFDictionaryGetValue(dict, kCGWindowOwnerName), out[used].owner, sizeof(out[used].owner));
		rf_ax_copy_cfstring_to_buffer((CFStringRef)CFDictionaryGetValue(dict, kCGWindowName), out[used].title, sizeof(out[used].title));
		if (out[used].owner[0] == '\0' && out[used].title[0] == '\0') {
			continue;
		}
		used++;
	}
	CFRelease(windows);
	return used;
}
*/
import "C"

import (
	"fmt"
	"image"
	"math"
	"os"
	"sort"
	"strings"
	"unsafe"
)

const darwinMaxAXAncestorDepth = 18
const darwinMaxWindowCandidates = 128

type regionAXElementRect struct {
	Rect   image.Rectangle
	Label  string
	Role   string
	PID    int
	Source string
}

type regionDarwinWindowRect struct {
	Rect     image.Rectangle
	Label    string
	PID      int
	WindowID uint32
}

func (s *RecordingFreedomService) regionElementCandidatesAtPoint(session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	absolutePoint := mapRegionPointToAbsolutePoint(session, point)
	bounds := boundsForRegionPoint(session, point)
	if bounds.Empty() || !absolutePoint.In(bounds) {
		return nil
	}
	rects, err := collectRegionAXElementRects(absolutePoint, os.Getpid())
	if err != nil || len(rects) == 0 {
		return nil
	}
	candidates := make([]RegionSmartCandidate, 0, len(rects))
	seen := map[string]bool{}
	minWidth, minHeight := regionSmartCandidateMinimumSize(session)
	for index, element := range rects {
		rect := element.Rect.Intersect(bounds)
		if rect.Empty() || !absolutePoint.In(rect) {
			continue
		}
		if regionUIRectCoversCapture(rect, bounds) {
			continue
		}
		relative := mapAbsoluteRectToRegionSelection(session, rect)
		relative = clampRegionCandidateBounds(relative, session.Bounds)
		if relative.Width < minWidth || relative.Height < minHeight {
			continue
		}
		key := fmt.Sprintf("%d:%d:%d:%d", relative.X, relative.Y, relative.Width, relative.Height)
		if seen[key] {
			continue
		}
		seen[key] = true
		label := safeRegionElementLabel(element.Label, strings.TrimPrefix(strings.TrimSpace(element.Role), "AX"))
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

func (s *RecordingFreedomService) regionWindowCandidatesAtPoint(session RegionSelectionSession, point image.Point) []RegionSmartCandidate {
	absolutePoint := mapRegionPointToAbsolutePoint(session, point)
	bounds := boundsForRegionPoint(session, point)
	if bounds.Empty() || !absolutePoint.In(bounds) {
		return nil
	}
	rects := collectRegionDarwinWindowRects(os.Getpid())
	if len(rects) == 0 {
		return nil
	}
	candidates := make([]RegionSmartCandidate, 0, len(rects))
	seen := map[string]bool{}
	minWidth, minHeight := regionSmartCandidateMinimumSize(session)
	for index, window := range rects {
		rect := window.Rect.Intersect(bounds)
		if rect.Empty() || !absolutePoint.In(rect) || regionUIRectCoversCapture(rect, bounds) {
			continue
		}
		relative := mapAbsoluteRectToRegionSelection(session, rect)
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
			ID:       fmt.Sprintf("window:%d", window.WindowID),
			Kind:     regionSmartKindWindow,
			Label:    safeRegionElementLabel(window.Label, "Window"),
			SourceID: fmt.Sprintf("cgwindow:%d", window.WindowID),
			Bounds:   relative,
			Score:    0.82 - float64(index)*0.01,
		})
	}
	return candidates
}

func collectRegionAXElementRects(point image.Point, currentPID int) ([]regionAXElementRect, error) {
	if C.rf_ax_is_trusted() == 0 {
		return nil, nil
	}
	var element C.AXUIElementRef
	if C.rf_ax_element_at_position(C.double(point.X), C.double(point.Y), &element) == 0 || darwinAXElementIsZero(element) {
		return nil, nil
	}
	defer C.CFRelease(C.CFTypeRef(element))

	rects := make([]regionAXElementRect, 0, darwinMaxAXAncestorDepth)
	heldChildren := make([]C.AXUIElementRef, 0, darwinMaxAXAncestorDepth)
	appendInfo := func(element C.AXUIElementRef, source string) {
		info, ok := darwinAXElementInfo(element)
		if ok && info.PID != currentPID && point.In(info.Rect) {
			info.Source = source
			rects = append(rects, info)
		}
	}

	current := element
	for depth := 0; !darwinAXElementIsZero(current) && depth < darwinMaxAXAncestorDepth; depth++ {
		source := "accessibility:child"
		if depth == 0 {
			source = "accessibility:point"
		}
		appendInfo(current, source)
		child := C.rf_ax_copy_best_child_at_position(current, C.double(point.X), C.double(point.Y))
		if darwinAXElementIsZero(child) {
			break
		}
		heldChildren = append(heldChildren, child)
		current = child
	}
	parent := C.rf_ax_copy_parent(current)
	for depth := 0; !darwinAXElementIsZero(parent) && depth < darwinMaxAXAncestorDepth; depth++ {
		appendInfo(parent, "accessibility:ancestor")
		next := C.rf_ax_copy_parent(parent)
		C.CFRelease(C.CFTypeRef(parent))
		parent = next
	}
	for _, child := range heldChildren {
		C.CFRelease(C.CFTypeRef(child))
	}
	rects = normalizeRegionAXElementRects(rects, point)
	return rects, nil
}

func darwinAXElementIsZero(element C.AXUIElementRef) bool {
	return element == C.AXUIElementRef(0)
}

func collectRegionDarwinWindowRects(currentPID int) []regionDarwinWindowRect {
	items := make([]C.RFAXWindowInfo, darwinMaxWindowCandidates)
	count := int(C.rf_ax_collect_windows(&items[0], C.int(len(items))))
	if count <= 0 {
		return nil
	}
	windows := make([]regionDarwinWindowRect, 0, count)
	for index := 0; index < count; index++ {
		item := items[index]
		pid := int(item.pid)
		if pid == 0 || pid == currentPID {
			continue
		}
		left := int(math.Floor(float64(item.x)))
		top := int(math.Floor(float64(item.y)))
		right := int(math.Ceil(float64(item.x + item.width)))
		bottom := int(math.Ceil(float64(item.y + item.height)))
		if right <= left || bottom <= top {
			continue
		}
		title := cStringFromArray(unsafe.Pointer(&item.title[0]))
		owner := cStringFromArray(unsafe.Pointer(&item.owner[0]))
		label := title
		if label == "" {
			label = owner
		}
		windows = append(windows, regionDarwinWindowRect{
			Rect:     image.Rect(left, top, right, bottom),
			Label:    label,
			PID:      pid,
			WindowID: uint32(item.windowID),
		})
	}
	return windows
}

func darwinAXCursorPosition() (image.Point, error) {
	var x C.double
	var y C.double
	if C.rf_ax_cursor_position(&x, &y) == 0 {
		return image.Point{}, fmt.Errorf("CGEventGetLocation failed")
	}
	return image.Point{X: int(math.Round(float64(x))), Y: int(math.Round(float64(y)))}, nil
}

func currentRegionCursorPoint(session RegionSelectionSession) (image.Point, bool) {
	cursor, err := darwinAXCursorPosition()
	if err != nil {
		return image.Point{}, false
	}
	point := image.Point{X: cursor.X - session.Bounds.X, Y: cursor.Y - session.Bounds.Y}
	if regionRectContainsPoint(RegionRect{Width: session.Bounds.Width, Height: session.Bounds.Height}, point) {
		return point, true
	}
	if absolute, ok := regionCapturePointToAbsolutePoint(session.DisplayBounds, cursor); ok {
		point = image.Point{X: absolute.X - session.Bounds.X, Y: absolute.Y - session.Bounds.Y}
		if regionRectContainsPoint(RegionRect{Width: session.Bounds.Width, Height: session.Bounds.Height}, point) {
			return point, true
		}
	}
	return image.Point{}, false
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

func darwinAXElementInfo(element C.AXUIElementRef) (regionAXElementRect, bool) {
	var info C.RFAXElementInfo
	if C.rf_ax_element_info(element, &info) == 0 {
		return regionAXElementRect{}, false
	}
	left := int(math.Floor(float64(info.x)))
	top := int(math.Floor(float64(info.y)))
	right := int(math.Ceil(float64(info.x + info.width)))
	bottom := int(math.Ceil(float64(info.y + info.height)))
	if right <= left || bottom <= top {
		return regionAXElementRect{}, false
	}
	return regionAXElementRect{
		Rect:  image.Rect(left, top, right, bottom),
		Label: cStringFromArray(unsafe.Pointer(&info.title[0])),
		Role:  cStringFromArray(unsafe.Pointer(&info.role[0])),
		PID:   int(info.pid),
	}, true
}

func cStringFromArray(value unsafe.Pointer) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(C.GoString((*C.char)(value)))
}

func normalizeRegionAXElementRects(rects []regionAXElementRect, point image.Point) []regionAXElementRect {
	if len(rects) == 0 {
		return nil
	}
	next := make([]regionAXElementRect, 0, len(rects))
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
