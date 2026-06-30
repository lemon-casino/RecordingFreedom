//go:build windows

package devices

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	winapi "golang.org/x/sys/windows"
)

var (
	user32                       = winapi.NewLazySystemDLL("user32.dll")
	procEnumDisplayMonitors      = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetShellWindow           = user32.NewProc("GetShellWindow")
	procGetDesktopWindow         = user32.NewProc("GetDesktopWindow")
)

const (
	monitorInfoPrimary = 1
	maxWindowTitle     = 512
)

type winRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type monitorInfoEx struct {
	Size     uint32
	Monitor  winRect
	WorkArea winRect
	Flags    uint32
	Device   [32]uint16
}

func listPlatformSources() ([]CaptureSource, error) {
	sources := make([]CaptureSource, 0, 32)
	sources = append(sources, listDisplaySources()...)
	windowSources := listWindowSources()
	sources = append(sources, windowSources...)
	sources = append(sources, applicationSources(windowSources)...)
	if len(sources) == 0 {
		return nil, fmt.Errorf("Windows APIs returned no display, window, or application sources")
	}
	return sources, nil
}

func listDisplaySources() []CaptureSource {
	sources := make([]CaptureSource, 0, 4)
	callback := syscall.NewCallback(func(monitor uintptr, hdc uintptr, rect uintptr, data uintptr) uintptr {
		info := monitorInfoEx{Size: uint32(unsafe.Sizeof(monitorInfoEx{}))}
		ok, _, _ := procGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(&info)))
		if ok == 0 {
			return 1
		}

		index := len(sources) + 1
		width := int(info.Monitor.Right - info.Monitor.Left)
		height := int(info.Monitor.Bottom - info.Monitor.Top)
		deviceName := strings.TrimSpace(winapi.UTF16ToString(info.Device[:]))
		if deviceName == "" {
			deviceName = fmt.Sprintf("DISPLAY%d", index)
		}

		name := fmt.Sprintf("Display %d", index)
		if info.Flags&monitorInfoPrimary != 0 {
			name = "Primary Display"
		}
		subtitle := fmt.Sprintf("%d x %d source pixels", width, height)
		if deviceName != "" {
			subtitle = fmt.Sprintf("%s · %s", subtitle, deviceName)
		}

		sources = append(sources, CaptureSource{
			ID:           fmt.Sprintf("screen:%s", sanitizeID(deviceName)),
			Type:         SourceScreen,
			Name:         name,
			Subtitle:     subtitle,
			Width:        width,
			Height:       height,
			NativeID:     deviceName,
			DisplayIndex: index,
			Available:    true,
			Capability:   CapabilityEnumerated,
		})
		return 1
	})

	result, _, _ := procEnumDisplayMonitors.Call(0, 0, callback, 0)
	if result == 0 || len(sources) == 0 {
		return nil
	}
	return sources
}

func listWindowSources() []CaptureSource {
	sources := make([]CaptureSource, 0, 24)
	shellWindow, _, _ := procGetShellWindow.Call()
	desktopWindow, _, _ := procGetDesktopWindow.Call()

	callback := syscall.NewCallback(func(hwnd uintptr, data uintptr) uintptr {
		if hwnd == 0 || hwnd == shellWindow || hwnd == desktopWindow || !isVisibleTopLevelWindow(hwnd) {
			return 1
		}

		title := windowTitle(hwnd)
		if title == "" {
			return 1
		}

		var pid uint32
		procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		process := processName(pid)
		subtitle := "Top-level window"
		if process != "" {
			subtitle = process
		}

		sources = append(sources, CaptureSource{
			ID:         fmt.Sprintf("window:%x", hwnd),
			Type:       SourceWindow,
			Name:       title,
			Subtitle:   subtitle,
			NativeID:   fmt.Sprintf("hwnd:%x", hwnd),
			ProcessID:  int(pid),
			Available:  true,
			Capability: CapabilityEnumerated,
		})
		return 1
	})

	result, _, _ := procEnumWindows.Call(callback, 0)
	if result == 0 || len(sources) == 0 {
		return nil
	}

	sort.SliceStable(sources, func(i, j int) bool {
		return strings.ToLower(sources[i].Name) < strings.ToLower(sources[j].Name)
	})
	return sources
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
			if name == "" || name == "Top-level window" {
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

func isVisibleTopLevelWindow(hwnd uintptr) bool {
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	if visible == 0 {
		return false
	}
	iconic, _, _ := procIsIconic.Call(hwnd)
	if iconic != 0 {
		return false
	}
	return true
}

func windowTitle(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return ""
	}
	if length > maxWindowTitle {
		length = maxWindowTitle
	}
	buffer := make([]uint16, int(length)+1)
	copied, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if copied == 0 {
		return ""
	}
	return strings.TrimSpace(winapi.UTF16ToString(buffer[:copied]))
}

func processName(pid uint32) string {
	if pid == 0 {
		return ""
	}
	handle, err := winapi.OpenProcess(winapi.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return fmt.Sprintf("PID %d", pid)
	}
	defer winapi.CloseHandle(handle)

	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := winapi.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil || size == 0 {
		return fmt.Sprintf("PID %d", pid)
	}
	return filepath.Base(winapi.UTF16ToString(buffer[:size]))
}

func sanitizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, ".", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "display"
	}
	return value
}
