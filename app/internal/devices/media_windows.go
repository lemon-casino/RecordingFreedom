//go:build windows

package devices

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	coinitApartmentThreaded = 0x2
	clsctxAll               = 0x17
	deviceStateActive       = 0x1
	eRender                 = 0
	eCapture                = 1
	eConsole                = 0
	stgmRead                = 0x0
	vtLPWSTR                = 31
)

var (
	clsidMMDeviceEnumerator = windows.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidIMMDeviceEnumerator  = windows.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	iidIPropertyStore       = windows.GUID{Data1: 0x886D8EEB, Data2: 0x8CF2, Data3: 0x4446, Data4: [8]byte{0x8D, 0x02, 0xCD, 0xBA, 0x1D, 0xBD, 0xCF, 0x99}}
	pkeyDeviceFriendlyName  = propertyKey{Fmtid: windows.GUID{Data1: 0xA45C254E, Data2: 0xDF1C, Data3: 0x4EFD, Data4: [8]byte{0x80, 0x20, 0x67, 0xD1, 0x46, 0xA8, 0x50, 0xE0}}, Pid: 14}

	ole32                = windows.NewLazySystemDLL("ole32.dll")
	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree    = ole32.NewProc("CoTaskMemFree")
	procPropVariantClear = ole32.NewProc("PropVariantClear")
)

type mmDeviceEnumerator struct {
	lpVtbl *mmDeviceEnumeratorVtbl
}

type mmDeviceEnumeratorVtbl struct {
	QueryInterface                         uintptr
	AddRef                                 uintptr
	Release                                uintptr
	EnumAudioEndpoints                     uintptr
	GetDefaultAudioEndpoint                uintptr
	GetDevice                              uintptr
	RegisterEndpointNotificationCallback   uintptr
	UnregisterEndpointNotificationCallback uintptr
}

type mmDeviceCollection struct {
	lpVtbl *mmDeviceCollectionVtbl
}

type mmDeviceCollectionVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetCount       uintptr
	Item           uintptr
}

type mmDevice struct {
	lpVtbl *mmDeviceVtbl
}

type mmDeviceVtbl struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	Activate          uintptr
	OpenPropertyStore uintptr
	GetId             uintptr
	GetState          uintptr
}

type propertyStore struct {
	lpVtbl *propertyStoreVtbl
}

type propertyStoreVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetCount       uintptr
	GetAt          uintptr
	GetValue       uintptr
	SetValue       uintptr
	Commit         uintptr
}

type propertyKey struct {
	Fmtid windows.GUID
	Pid   uint32
}

type propVariant struct {
	Vt        uint16
	Reserved1 uint16
	Reserved2 uint16
	Reserved3 uint16
	Val       uintptr
	Reserved4 uintptr
	Reserved5 uintptr
}

type windowsAudioEndpoint struct {
	id        string
	name      string
	isDefault bool
}

func listPlatformMediaDevices() (MediaInventory, error) {
	if err := coInitialize(); err != nil {
		return MediaInventory{}, err
	}
	defer procCoUninitialize.Call()

	enumerator, err := createMMDeviceEnumerator()
	if err != nil {
		return MediaInventory{}, err
	}
	defer enumerator.release()

	render, err := listAudioEndpoints(enumerator, eRender)
	if err != nil {
		return MediaInventory{}, err
	}
	capture, err := listAudioEndpoints(enumerator, eCapture)
	if err != nil {
		return MediaInventory{}, err
	}

	return MediaInventory{
		SystemAudio: windowsMediaDevices(DeviceSystemAudio, render),
		Microphones: windowsMediaDevices(DeviceMicrophone, capture),
		Cameras:     listWindowsCameraDevices(),
		Enhancement: defaultAudioEnhancement("RNNoise native DSP is queued behind microphone capture plumbing."),
	}, nil
}

func windowsMediaDevices(deviceType MediaDeviceType, endpoints []windowsAudioEndpoint) []MediaDevice {
	devices := make([]MediaDevice, 0, len(endpoints)+1)
	for _, endpoint := range endpoints {
		id := fmt.Sprintf("%s:wasapi:%s", deviceType, endpoint.id)
		if endpoint.isDefault {
			id = fmt.Sprintf("%s:default", deviceType)
		}
		device := MediaDevice{
			ID:         id,
			Type:       deviceType,
			Name:       endpoint.name,
			Subtitle:   windowsAudioSubtitle(deviceType),
			NativeID:   endpoint.id,
			IsDefault:  endpoint.isDefault,
			Available:  true,
			Capability: CapabilityEnumerated,
		}
		if endpoint.isDefault {
			device.Name = "Default " + endpoint.name
		}
		if deviceType == DeviceMicrophone {
			device.RNNoiseEligible = true
		}
		devices = append(devices, device)
	}
	return devices
}

func windowsAudioSubtitle(deviceType MediaDeviceType) string {
	switch deviceType {
	case DeviceSystemAudio:
		return "WASAPI render endpoint for system loopback"
	case DeviceMicrophone:
		return "WASAPI capture endpoint"
	default:
		return defaultMediaDeviceSubtitle(deviceType)
	}
}

func listWindowsCameraDevices() []MediaDevice {
	ffmpegPath, err := resolveWindowsFFmpegPath()
	if err != nil {
		return []MediaDevice{defaultMediaDevice(DeviceCamera, "DirectShow camera enumeration requires FFmpeg; "+err.Error())}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath, "-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	configureBackgroundCommand(cmd)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	_ = cmd.Run()
	devices := parseDirectShowCameraDevices(output.String())
	if len(devices) == 0 {
		reason := "DirectShow returned no camera devices"
		if ctx.Err() != nil {
			reason = "DirectShow camera enumeration timed out"
		}
		return []MediaDevice{defaultMediaDevice(DeviceCamera, reason)}
	}
	return devices
}

func resolveWindowsFFmpegPath() (string, error) {
	const envFFmpegPath = "RECORDINGFREEDOM_FFMPEG_PATH"
	if configured := strings.TrimSpace(os.Getenv(envFFmpegPath)); configured != "" {
		return validateWindowsFFmpegPath(configured, envFFmpegPath)
	}
	candidates := make([]string, 0, 8)
	if executable, err := os.Executable(); err == nil {
		base := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(base, "ffmpeg.exe"),
			filepath.Join(base, "tools", "ffmpeg.exe"),
			filepath.Join(base, "bin", "ffmpeg.exe"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "ffmpeg.exe"),
			filepath.Join(cwd, "tools", "ffmpeg.exe"),
			filepath.Join(cwd, "bin", "ffmpeg.exe"),
		)
	}
	for _, candidate := range candidates {
		if path, err := validateWindowsFFmpegPath(candidate, "bundled ffmpeg"); err == nil {
			return path, nil
		}
	}
	if path, err := exec.LookPath("ffmpeg.exe"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("FFmpeg executable was not found; set %s or bundle ffmpeg beside the app under tools/", envFFmpegPath)
}

func validateWindowsFFmpegPath(path string, source string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s %q is not readable: %w", source, path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s %q is a directory, not an ffmpeg executable", source, path)
	}
	return path, nil
}

func coInitialize() error {
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failed(hr) {
		return fmt.Errorf("CoInitializeEx failed: %s", hresultString(hr))
	}
	return nil
}

func createMMDeviceEnumerator() (*mmDeviceEnumerator, error) {
	var enumerator *mmDeviceEnumerator
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidMMDeviceEnumerator)),
		0,
		clsctxAll,
		uintptr(unsafe.Pointer(&iidIMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enumerator)),
	)
	if failed(hr) {
		return nil, fmt.Errorf("CoCreateInstance(MMDeviceEnumerator) failed: %s", hresultString(hr))
	}
	if enumerator == nil {
		return nil, fmt.Errorf("CoCreateInstance(MMDeviceEnumerator) returned nil enumerator")
	}
	return enumerator, nil
}

func listAudioEndpoints(enumerator *mmDeviceEnumerator, flow uintptr) ([]windowsAudioEndpoint, error) {
	defaultID := ""
	defaultDevice, err := enumerator.defaultAudioEndpoint(flow)
	if err == nil && defaultDevice != nil {
		defaultID, _ = defaultDevice.id()
		defaultDevice.release()
	}

	collection, err := enumerator.enumAudioEndpoints(flow)
	if err != nil {
		return nil, err
	}
	defer collection.release()

	count, err := collection.count()
	if err != nil {
		return nil, err
	}
	endpoints := make([]windowsAudioEndpoint, 0, count)
	for index := uint32(0); index < count; index++ {
		device, err := collection.item(index)
		if err != nil {
			continue
		}
		id, idErr := device.id()
		name := device.friendlyName()
		device.release()
		if idErr != nil || strings.TrimSpace(id) == "" {
			continue
		}
		if strings.TrimSpace(name) == "" {
			name = "WASAPI Endpoint"
		}
		endpoints = append(endpoints, windowsAudioEndpoint{
			id:        id,
			name:      name,
			isDefault: id == defaultID,
		})
	}
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("WASAPI endpoint enumeration returned no active devices")
	}
	return endpoints, nil
}

func (e *mmDeviceEnumerator) enumAudioEndpoints(flow uintptr) (*mmDeviceCollection, error) {
	var collection *mmDeviceCollection
	hr, _, _ := syscall.SyscallN(
		e.lpVtbl.EnumAudioEndpoints,
		uintptr(unsafe.Pointer(e)),
		flow,
		deviceStateActive,
		uintptr(unsafe.Pointer(&collection)),
	)
	if failed(hr) {
		return nil, fmt.Errorf("IMMDeviceEnumerator::EnumAudioEndpoints failed: %s", hresultString(hr))
	}
	if collection == nil {
		return nil, fmt.Errorf("IMMDeviceEnumerator::EnumAudioEndpoints returned nil collection")
	}
	return collection, nil
}

func (e *mmDeviceEnumerator) defaultAudioEndpoint(flow uintptr) (*mmDevice, error) {
	var device *mmDevice
	hr, _, _ := syscall.SyscallN(
		e.lpVtbl.GetDefaultAudioEndpoint,
		uintptr(unsafe.Pointer(e)),
		flow,
		eConsole,
		uintptr(unsafe.Pointer(&device)),
	)
	if failed(hr) {
		return nil, fmt.Errorf("IMMDeviceEnumerator::GetDefaultAudioEndpoint failed: %s", hresultString(hr))
	}
	return device, nil
}

func (e *mmDeviceEnumerator) release() {
	syscall.SyscallN(e.lpVtbl.Release, uintptr(unsafe.Pointer(e)))
}

func (c *mmDeviceCollection) count() (uint32, error) {
	var count uint32
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.GetCount,
		uintptr(unsafe.Pointer(c)),
		uintptr(unsafe.Pointer(&count)),
	)
	if failed(hr) {
		return 0, fmt.Errorf("IMMDeviceCollection::GetCount failed: %s", hresultString(hr))
	}
	return count, nil
}

func (c *mmDeviceCollection) item(index uint32) (*mmDevice, error) {
	var device *mmDevice
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.Item,
		uintptr(unsafe.Pointer(c)),
		uintptr(index),
		uintptr(unsafe.Pointer(&device)),
	)
	if failed(hr) {
		return nil, fmt.Errorf("IMMDeviceCollection::Item(%d) failed: %s", index, hresultString(hr))
	}
	if device == nil {
		return nil, fmt.Errorf("IMMDeviceCollection::Item(%d) returned nil device", index)
	}
	return device, nil
}

func (c *mmDeviceCollection) release() {
	syscall.SyscallN(c.lpVtbl.Release, uintptr(unsafe.Pointer(c)))
}

func (d *mmDevice) id() (string, error) {
	var raw *uint16
	hr, _, _ := syscall.SyscallN(
		d.lpVtbl.GetId,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if failed(hr) {
		return "", fmt.Errorf("IMMDevice::GetId failed: %s", hresultString(hr))
	}
	if raw == nil {
		return "", nil
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(raw)))
	return windows.UTF16PtrToString(raw), nil
}

func (d *mmDevice) friendlyName() string {
	store, err := d.openPropertyStore()
	if err != nil {
		return ""
	}
	defer store.release()
	return store.friendlyName()
}

func (d *mmDevice) openPropertyStore() (*propertyStore, error) {
	var store *propertyStore
	hr, _, _ := syscall.SyscallN(
		d.lpVtbl.OpenPropertyStore,
		uintptr(unsafe.Pointer(d)),
		stgmRead,
		uintptr(unsafe.Pointer(&store)),
	)
	if failed(hr) {
		return nil, fmt.Errorf("IMMDevice::OpenPropertyStore failed: %s", hresultString(hr))
	}
	if store == nil {
		return nil, fmt.Errorf("IMMDevice::OpenPropertyStore returned nil store")
	}
	return store, nil
}

func (d *mmDevice) release() {
	syscall.SyscallN(d.lpVtbl.Release, uintptr(unsafe.Pointer(d)))
}

func (s *propertyStore) friendlyName() string {
	var value propVariant
	hr, _, _ := syscall.SyscallN(
		s.lpVtbl.GetValue,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(&pkeyDeviceFriendlyName)),
		uintptr(unsafe.Pointer(&value)),
	)
	if failed(hr) {
		return ""
	}
	defer procPropVariantClear.Call(uintptr(unsafe.Pointer(&value)))
	if value.Vt != vtLPWSTR || value.Val == 0 {
		return ""
	}
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(value.Val)))
}

func (s *propertyStore) release() {
	syscall.SyscallN(s.lpVtbl.Release, uintptr(unsafe.Pointer(s)))
}

func failed(hr uintptr) bool {
	return int32(hr) < 0
}

func hresultString(hr uintptr) string {
	return fmt.Sprintf("0x%08X", uint32(hr))
}
