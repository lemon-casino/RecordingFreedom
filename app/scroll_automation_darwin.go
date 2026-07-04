//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>

static void rf_scroll_down_at(double x, double y, int pixels) {
	CGEventRef current = CGEventCreate(NULL);
	CGPoint original = CGPointMake(x, y);
	if (current != NULL) {
		original = CGEventGetLocation(current);
		CFRelease(current);
	}
	CGWarpMouseCursorPosition(CGPointMake(x, y));
	CGAssociateMouseAndMouseCursorPosition(true);
	CGEventRef event = CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitPixel, 1, -pixels);
	if (event != NULL) {
		CGEventPost(kCGHIDEventTap, event);
		CFRelease(event);
	}
	CGWarpMouseCursorPosition(original);
	CGAssociateMouseAndMouseCursorPosition(true);
}
*/
import "C"

import (
	"fmt"
	"image"
	"time"
)

func scrollDownAtRect(rect image.Rectangle) error {
	if rect.Empty() {
		return fmt.Errorf("scrolling screenshot rectangle is empty")
	}
	centerX := rect.Min.X + rect.Dx()/2
	centerY := rect.Min.Y + rect.Dy()/2
	pixels := maxInt(80, minInt(420, rect.Dy()*2/3))
	C.rf_scroll_down_at(C.double(centerX), C.double(centerY), C.int(pixels))
	time.Sleep(35 * time.Millisecond)
	return nil
}
