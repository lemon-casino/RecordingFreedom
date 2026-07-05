//go:build darwin

package main

/*
#cgo darwin LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <stdint.h>

typedef struct {
	int x;
	int y;
	int width;
	int height;
} RFFocusedWindowRect;

static int rf_dict_number_int(CFDictionaryRef dict, const void *key, int *out) {
	CFTypeRef value = CFDictionaryGetValue(dict, key);
	if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
		return 0;
	}
	return CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, out);
}

static int rf_front_window_bounds(int skipPID, RFFocusedWindowRect *out) {
	CFArrayRef windows = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID
	);
	if (windows == NULL || out == NULL) {
		return 0;
	}
	CFIndex total = CFArrayGetCount(windows);
	for (CFIndex i = 0; i < total; i++) {
		CFDictionaryRef dict = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
		if (dict == NULL) {
			continue;
		}
		int layer = 0;
		if (!rf_dict_number_int(dict, kCGWindowLayer, &layer) || layer != 0) {
			continue;
		}
		int pid = 0;
		rf_dict_number_int(dict, kCGWindowOwnerPID, &pid);
		if (skipPID > 0 && pid == skipPID) {
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
		out->x = (int)bounds.origin.x;
		out->y = (int)bounds.origin.y;
		out->width = (int)bounds.size.width;
		out->height = (int)bounds.size.height;
		CFRelease(windows);
		return 1;
	}
	CFRelease(windows);
	return 0;
}
*/
import "C"

import (
	"image"
	"os"
)

func detectFocusedWindowScreenshotRect() (image.Rectangle, bool) {
	var rect C.RFFocusedWindowRect
	if C.rf_front_window_bounds(C.int(os.Getpid()), &rect) == 0 {
		return image.Rectangle{}, false
	}
	result := image.Rect(int(rect.x), int(rect.y), int(rect.x+rect.width), int(rect.y+rect.height))
	return result, !result.Empty()
}
