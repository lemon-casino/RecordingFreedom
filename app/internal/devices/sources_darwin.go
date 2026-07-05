//go:build darwin && cgo

package devices

/*
#cgo darwin LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
	uint32_t displayID;
	int x;
	int y;
	int width;
	int height;
	int primary;
} RFDisplay;

typedef struct {
	uint32_t windowID;
	int pid;
	char *owner;
	char *title;
	int x;
	int y;
	int width;
	int height;
} RFWindow;

static char* rf_copy_cfstring(CFStringRef value) {
	if (value == NULL) {
		return NULL;
	}
	CFIndex length = CFStringGetLength(value);
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
	char *buffer = (char*)calloc((size_t)maxSize, sizeof(char));
	if (buffer == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(value, buffer, maxSize, kCFStringEncodingUTF8)) {
		free(buffer);
		return NULL;
	}
	return buffer;
}

static int rf_dict_number_int(CFDictionaryRef dict, const void *key, int *out) {
	CFTypeRef value = CFDictionaryGetValue(dict, key);
	if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
		return 0;
	}
	return CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, out);
}

static int rf_dict_number_u32(CFDictionaryRef dict, const void *key, uint32_t *out) {
	CFTypeRef value = CFDictionaryGetValue(dict, key);
	if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
		return 0;
	}
	return CFNumberGetValue((CFNumberRef)value, kCFNumberSInt32Type, out);
}

static int rf_list_displays(RFDisplay **outItems, int *outCount) {
	uint32_t count = 0;
	CGError countError = CGGetActiveDisplayList(0, NULL, &count);
	if (countError != kCGErrorSuccess || count == 0) {
		*outItems = NULL;
		*outCount = 0;
		return -1;
	}

	CGDirectDisplayID *ids = (CGDirectDisplayID*)calloc(count, sizeof(CGDirectDisplayID));
	if (ids == NULL) {
		*outItems = NULL;
		*outCount = 0;
		return -2;
	}
	CGError listError = CGGetActiveDisplayList(count, ids, &count);
	if (listError != kCGErrorSuccess || count == 0) {
		free(ids);
		*outItems = NULL;
		*outCount = 0;
		return -3;
	}

	RFDisplay *items = (RFDisplay*)calloc(count, sizeof(RFDisplay));
	if (items == NULL) {
		free(ids);
		*outItems = NULL;
		*outCount = 0;
		return -4;
	}

	for (uint32_t i = 0; i < count; i++) {
		CGDirectDisplayID id = ids[i];
		CGRect bounds = CGDisplayBounds(id);
		items[i].displayID = (uint32_t)id;
		items[i].x = (int)bounds.origin.x;
		items[i].y = (int)bounds.origin.y;
		items[i].width = (int)CGDisplayPixelsWide(id);
		items[i].height = (int)CGDisplayPixelsHigh(id);
		items[i].primary = CGDisplayIsMain(id) ? 1 : 0;
	}

	free(ids);
	*outItems = items;
	*outCount = (int)count;
	return 0;
}

static int rf_list_windows(RFWindow **outItems, int *outCount) {
	CFArrayRef windows = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID
	);
	if (windows == NULL) {
		*outItems = NULL;
		*outCount = 0;
		return -1;
	}

	CFIndex total = CFArrayGetCount(windows);
	if (total == 0) {
		CFRelease(windows);
		*outItems = NULL;
		*outCount = 0;
		return 0;
	}

	RFWindow *items = (RFWindow*)calloc((size_t)total, sizeof(RFWindow));
	if (items == NULL) {
		CFRelease(windows);
		*outItems = NULL;
		*outCount = 0;
		return -2;
	}

	int used = 0;
	for (CFIndex i = 0; i < total; i++) {
		CFDictionaryRef dict = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
		if (dict == NULL) {
			continue;
		}

		int layer = 0;
		if (!rf_dict_number_int(dict, kCGWindowLayer, &layer) || layer != 0) {
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
		if (!rf_dict_number_u32(dict, kCGWindowNumber, &windowID) || windowID == 0) {
			continue;
		}

		int pid = 0;
		rf_dict_number_int(dict, kCGWindowOwnerPID, &pid);
		char *owner = rf_copy_cfstring((CFStringRef)CFDictionaryGetValue(dict, kCGWindowOwnerName));
		char *title = rf_copy_cfstring((CFStringRef)CFDictionaryGetValue(dict, kCGWindowName));
		if ((owner == NULL || owner[0] == '\0') && (title == NULL || title[0] == '\0')) {
			free(owner);
			free(title);
			continue;
		}

		items[used].windowID = windowID;
		items[used].pid = pid;
		items[used].owner = owner;
		items[used].title = title;
		items[used].x = (int)bounds.origin.x;
		items[used].y = (int)bounds.origin.y;
		items[used].width = (int)bounds.size.width;
		items[used].height = (int)bounds.size.height;
		used++;
	}

	CFRelease(windows);
	if (used == 0) {
		free(items);
		items = NULL;
	}
	*outItems = items;
	*outCount = used;
	return 0;
}

static void rf_free_windows(RFWindow *items, int count) {
	if (items == NULL) {
		return;
	}
	for (int i = 0; i < count; i++) {
		free(items[i].owner);
		free(items[i].title);
	}
	free(items);
}
*/
import "C"

import (
	"fmt"
	"sort"
	"strings"
	"unsafe"
)

func listPlatformSources() ([]CaptureSource, error) {
	sources := make([]CaptureSource, 0, 32)
	sources = append(sources, listDarwinDisplaySources()...)
	windowSources := listDarwinWindowSources()
	sources = append(sources, windowSources...)
	sources = append(sources, applicationSources(windowSources)...)
	if len(sources) == 0 {
		return nil, fmt.Errorf("macOS CoreGraphics returned no display, window, or application sources")
	}
	return sources, nil
}

func listDarwinDisplaySources() []CaptureSource {
	var raw *C.RFDisplay
	var count C.int
	if C.rf_list_displays(&raw, &count) != 0 || raw == nil || count == 0 {
		return nil
	}
	defer C.free(unsafe.Pointer(raw))

	displays := unsafe.Slice(raw, int(count))
	sources := make([]CaptureSource, 0, len(displays))
	for index, display := range displays {
		displayID := uint32(display.displayID)
		name := fmt.Sprintf("Display %d", index+1)
		if display.primary == 1 {
			name = "Primary Display"
		}
		sources = append(sources, CaptureSource{
			ID:           fmt.Sprintf("screen:display-%d", displayID),
			Type:         SourceScreen,
			Name:         name,
			Subtitle:     fmt.Sprintf("%d x %d source pixels · CGDirectDisplayID %d", int(display.width), int(display.height), displayID),
			X:            int(display.x),
			Y:            int(display.y),
			Width:        int(display.width),
			Height:       int(display.height),
			NativeID:     fmt.Sprintf("cgdisplay:%d", displayID),
			DisplayIndex: index + 1,
			Available:    true,
			Capability:   CapabilityEnumerated,
		})
	}
	return sources
}

func listDarwinWindowSources() []CaptureSource {
	var raw *C.RFWindow
	var count C.int
	if C.rf_list_windows(&raw, &count) != 0 || raw == nil || count == 0 {
		return nil
	}
	defer C.rf_free_windows(raw, count)

	windows := unsafe.Slice(raw, int(count))
	sources := make([]CaptureSource, 0, len(windows))
	for _, window := range windows {
		owner := strings.TrimSpace(cString(window.owner))
		title := strings.TrimSpace(cString(window.title))
		name := title
		if name == "" {
			name = owner
		}
		if name == "" {
			continue
		}

		subtitle := owner
		if subtitle == "" {
			subtitle = fmt.Sprintf("PID %d", int(window.pid))
		}
		if window.width > 0 && window.height > 0 {
			subtitle = fmt.Sprintf("%s · %d x %d", subtitle, int(window.width), int(window.height))
		}

		windowID := uint32(window.windowID)
		sources = append(sources, CaptureSource{
			ID:         fmt.Sprintf("window:%d", windowID),
			Type:       SourceWindow,
			Name:       name,
			Subtitle:   subtitle,
			X:          int(window.x),
			Y:          int(window.y),
			Width:      int(window.width),
			Height:     int(window.height),
			NativeID:   fmt.Sprintf("cgwindow:%d", windowID),
			ProcessID:  int(window.pid),
			Available:  true,
			Capability: CapabilityEnumerated,
		})
	}

	sort.SliceStable(sources, func(i, j int) bool {
		return strings.ToLower(sources[i].Name) < strings.ToLower(sources[j].Name)
	})
	return sources
}

func cString(value *C.char) string {
	if value == nil {
		return ""
	}
	return C.GoString(value)
}

func applicationSources(windows []CaptureSource) []CaptureSource {
	type app struct {
		processID int
		name      string
		count     int
	}
	appsByPID := make(map[int]*app)
	for _, source := range windows {
		if source.ProcessID <= 0 {
			continue
		}
		current := appsByPID[source.ProcessID]
		if current == nil {
			name := strings.TrimSpace(source.Subtitle)
			if index := strings.Index(name, " · "); index >= 0 {
				name = name[:index]
			}
			if name == "" {
				name = fmt.Sprintf("PID %d", source.ProcessID)
			}
			current = &app{processID: source.ProcessID, name: name}
			appsByPID[source.ProcessID] = current
		}
		current.count++
	}

	apps := make([]*app, 0, len(appsByPID))
	for _, item := range appsByPID {
		apps = append(apps, item)
	}
	sort.SliceStable(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].name) < strings.ToLower(apps[j].name)
	})

	sources := make([]CaptureSource, 0, len(apps))
	for _, item := range apps {
		sources = append(sources, CaptureSource{
			ID:         fmt.Sprintf("application:%d", item.processID),
			Type:       SourceApplication,
			Name:       item.name,
			Subtitle:   fmt.Sprintf("%d visible window(s)", item.count),
			ProcessID:  item.processID,
			Available:  true,
			Capability: CapabilityEnumerated,
		})
	}
	return sources
}
