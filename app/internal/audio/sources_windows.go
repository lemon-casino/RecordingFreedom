//go:build windows

package audio

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wasapiCoInitApartmentThreaded      = 0x2
	wasapiClsctxAll                    = 0x17
	wasapiDeviceStateActive            = 0x1
	wasapiERender                      = 0
	wasapiECapture                     = 1
	wasapiEConsole                     = 0
	wasapiShareModeShared              = 0
	wasapiStreamFlagsLoopback          = 0x00020000
	wasapiBufferFlagsDataDiscontinuity = 0x1
	wasapiBufferFlagsSilent            = 0x2
	wasapiBufferDurationHNS            = 10_000_000
	wasapiHNSPerSecond                 = 10_000_000
	waveFormatPCM                      = 0x0001
	waveFormatIEEEFloat                = 0x0003
	waveFormatExtensible               = 0xFFFE
)

var (
	wasapiCLSIDMMDeviceEnumerator = windows.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	wasapiIIDIMMDeviceEnumerator  = windows.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	wasapiIIDIAudioClient         = windows.GUID{Data1: 0x1CB9AD4C, Data2: 0xDBFA, Data3: 0x4c32, Data4: [8]byte{0xB1, 0x78, 0xC2, 0xF5, 0x68, 0xA7, 0x03, 0xB2}}
	wasapiIIDIAudioCaptureClient  = windows.GUID{Data1: 0xC8ADBD64, Data2: 0xE71E, Data3: 0x48a0, Data4: [8]byte{0xA4, 0xDE, 0x18, 0x5C, 0x39, 0x5C, 0xD3, 0x17}}
	wasapiSubFormatIEEEFloat      = windows.GUID{Data1: 0x00000003, Data2: 0x0000, Data3: 0x0010, Data4: [8]byte{0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}}
	wasapiSubFormatPCM            = windows.GUID{Data1: 0x00000001, Data2: 0x0000, Data3: 0x0010, Data4: [8]byte{0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}}

	wasapiOle32                = windows.NewLazySystemDLL("ole32.dll")
	wasapiProcCoInitializeEx   = wasapiOle32.NewProc("CoInitializeEx")
	wasapiProcCoUninitialize   = wasapiOle32.NewProc("CoUninitialize")
	wasapiProcCoCreateInstance = wasapiOle32.NewProc("CoCreateInstance")
	wasapiProcCoTaskMemFree    = wasapiOle32.NewProc("CoTaskMemFree")
)

type wasapiMMDeviceEnumerator struct {
	lpVtbl *wasapiMMDeviceEnumeratorVtbl
}

type wasapiMMDeviceEnumeratorVtbl struct {
	QueryInterface                         uintptr
	AddRef                                 uintptr
	Release                                uintptr
	EnumAudioEndpoints                     uintptr
	GetDefaultAudioEndpoint                uintptr
	GetDevice                              uintptr
	RegisterEndpointNotificationCallback   uintptr
	UnregisterEndpointNotificationCallback uintptr
}

type wasapiMMDevice struct {
	lpVtbl *wasapiMMDeviceVtbl
}

type wasapiMMDeviceVtbl struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	Activate          uintptr
	OpenPropertyStore uintptr
	GetId             uintptr
	GetState          uintptr
}

type wasapiAudioClient struct {
	lpVtbl *wasapiAudioClientVtbl
}

type wasapiAudioClientVtbl struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	Initialize        uintptr
	GetBufferSize     uintptr
	GetStreamLatency  uintptr
	GetCurrentPadding uintptr
	IsFormatSupported uintptr
	GetMixFormat      uintptr
	GetDevicePeriod   uintptr
	Start             uintptr
	Stop              uintptr
	Reset             uintptr
	SetEventHandle    uintptr
	GetService        uintptr
}

type wasapiAudioCaptureClient struct {
	lpVtbl *wasapiAudioCaptureClientVtbl
}

type wasapiAudioCaptureClientVtbl struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	GetBuffer         uintptr
	ReleaseBuffer     uintptr
	GetNextPacketSize uintptr
}

type wasapiWaveFormatEx struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	ExtraSize      uint16
}

type wasapiInputFormat struct {
	sampleRate    int
	channels      int
	bitsPerSample int
	blockAlign    int
	float32       bool
	pcm           bool
}

type wasapiCaptureSource struct {
	id         string
	kind       StreamKind
	deviceID   string
	gain       float64
	targetRate int
	stopCh     chan struct{}
	doneCh     chan struct{}
	readyCh    chan error
	stopOnce   sync.Once
	paused     atomic.Bool
	errMu      sync.Mutex
	err        error
}

func NewPlatformCaptureSources(config CaptureConfig) ([]CaptureSource, error) {
	config = normalizeCaptureConfig(config)
	sources := make([]CaptureSource, 0, 2)
	if config.SystemAudio.Enabled {
		sources = append(sources, newWasapiCaptureSource(StreamSystemAudio, config.SystemAudio.DeviceID, 1, config.TargetSampleRate))
	}
	if config.Microphone.Enabled {
		sources = append(sources, newWasapiCaptureSource(StreamMicrophone, config.Microphone.DeviceID, config.MicrophoneGain, RNNoiseSampleRate))
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("native audio capture backend %q has no enabled streams", config.Backend)
	}
	return sources, nil
}

func newWasapiCaptureSource(kind StreamKind, deviceID string, gain float64, targetRate int) *wasapiCaptureSource {
	if targetRate <= 0 {
		targetRate = RNNoiseSampleRate
	}
	if gain <= 0 {
		gain = 1
	}
	return &wasapiCaptureSource{
		id:         fmt.Sprintf("wasapi:%s", kind),
		kind:       kind,
		deviceID:   deviceID,
		gain:       gain,
		targetRate: targetRate,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
		readyCh:    make(chan error, 1),
	}
}

func (s *wasapiCaptureSource) ID() string {
	return s.id
}

func (s *wasapiCaptureSource) Kind() StreamKind {
	return s.kind
}

func (s *wasapiCaptureSource) Start(onFrame func(TimedPCMBuffer) error) error {
	if onFrame == nil {
		return fmt.Errorf("wasapi source %q requires a frame callback", s.id)
	}
	go s.run(onFrame)
	return <-s.readyCh
}

func (s *wasapiCaptureSource) Pause() error {
	s.paused.Store(true)
	return nil
}

func (s *wasapiCaptureSource) Resume() error {
	s.paused.Store(false)
	return nil
}

func (s *wasapiCaptureSource) Stop() error {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	<-s.doneCh
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

func (s *wasapiCaptureSource) run(onFrame func(TimedPCMBuffer) error) {
	defer close(s.doneCh)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := wasapiCoInitialize(); err != nil {
		s.readyCh <- err
		return
	}
	defer wasapiProcCoUninitialize.Call()

	enumerator, err := wasapiCreateMMDeviceEnumerator()
	if err != nil {
		s.readyCh <- err
		return
	}
	defer enumerator.release()

	device, err := enumerator.resolveDevice(s.kind, s.deviceID)
	if err != nil {
		s.readyCh <- err
		return
	}
	defer device.release()

	client, err := device.activateAudioClient()
	if err != nil {
		s.readyCh <- err
		return
	}
	defer client.release()

	rawFormat, inputFormat, err := client.mixFormat()
	if err != nil {
		if rawFormat != nil {
			wasapiProcCoTaskMemFree.Call(uintptr(unsafe.Pointer(rawFormat)))
		}
		s.readyCh <- err
		return
	}
	defer wasapiProcCoTaskMemFree.Call(uintptr(unsafe.Pointer(rawFormat)))

	flags := uintptr(0)
	if s.kind == StreamSystemAudio {
		flags = wasapiStreamFlagsLoopback
	}
	if err := client.initialize(rawFormat, flags); err != nil {
		s.readyCh <- err
		return
	}

	captureClient, err := client.captureClient()
	if err != nil {
		s.readyCh <- err
		return
	}
	defer captureClient.release()

	if err := client.start(); err != nil {
		s.readyCh <- err
		return
	}
	defer client.stop()

	resampler, err := NewMonoResampler(inputFormat.sampleRate, s.targetRate)
	if err != nil {
		s.readyCh <- err
		return
	}

	s.readyCh <- nil
	s.captureLoop(captureClient, inputFormat, resampler, onFrame)
}

func (s *wasapiCaptureSource) captureLoop(captureClient *wasapiAudioCaptureClient, format wasapiInputFormat, resampler *MonoResampler, onFrame func(TimedPCMBuffer) error) {
	var lastDevicePositionEnd uint64
	hasLastDevicePosition := false
	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		packetFrames, err := captureClient.nextPacketSize()
		if err != nil {
			s.setErr(err)
			return
		}
		sleepDuration := 5 * time.Millisecond
		if packetFrames == 0 && s.kind == StreamSystemAudio && !s.paused.Load() {
			silenceFrames := syntheticSilenceFrames(format.sampleRate)
			if err := s.emitSilence(format, resampler, lastDevicePositionEnd, silenceFrames, onFrame); err != nil {
				s.setErr(err)
				return
			}
			lastDevicePositionEnd += silenceFrames
			hasLastDevicePosition = true
			sleepDuration = time.Duration(silenceFrames) * time.Second / time.Duration(format.sampleRate)
		}
		for packetFrames > 0 {
			select {
			case <-s.stopCh:
				return
			default:
			}
			packet, err := captureClient.buffer()
			if err != nil {
				s.setErr(err)
				return
			}

			if !s.paused.Load() {
				if hasLastDevicePosition && packet.devicePosition > lastDevicePositionEnd && (packet.flags&wasapiBufferFlagsDataDiscontinuity) != 0 {
					gapFrames := packet.devicePosition - lastDevicePositionEnd
					if err := s.emitSilence(format, resampler, lastDevicePositionEnd, gapFrames, onFrame); err != nil {
						_ = captureClient.releaseBuffer(packet.frames)
						s.setErr(err)
						return
					}
				}
				if err := s.emitPacket(format, resampler, packet, onFrame); err != nil {
					_ = captureClient.releaseBuffer(packet.frames)
					s.setErr(err)
					return
				}
			}

			lastDevicePositionEnd = packet.devicePosition + uint64(packet.frames)
			hasLastDevicePosition = true
			if err := captureClient.releaseBuffer(packet.frames); err != nil {
				s.setErr(err)
				return
			}
			packetFrames, err = captureClient.nextPacketSize()
			if err != nil {
				s.setErr(err)
				return
			}
		}
		time.Sleep(sleepDuration)
	}
}

func syntheticSilenceFrames(sampleRate int) uint64 {
	if sampleRate <= 0 {
		return 480
	}
	frames := sampleRate / 20
	if frames <= 0 {
		return 1
	}
	return uint64(frames)
}

func (s *wasapiCaptureSource) emitSilence(format wasapiInputFormat, resampler *MonoResampler, devicePosition uint64, frames uint64, onFrame func(TimedPCMBuffer) error) error {
	const maxSilenceFrames = 4800
	remaining := frames
	position := devicePosition
	for remaining > 0 {
		chunk := remaining
		if chunk > maxSilenceFrames {
			chunk = maxSilenceFrames
		}
		samples := make([]float32, int(chunk)*format.channels)
		packet := wasapiPacket{
			frames:         uint32(chunk),
			devicePosition: position,
			silent:         true,
			decoded:        samples,
		}
		if err := s.emitDecoded(format, resampler, packet, onFrame); err != nil {
			return err
		}
		remaining -= chunk
		position += chunk
	}
	return nil
}

func (s *wasapiCaptureSource) emitPacket(format wasapiInputFormat, resampler *MonoResampler, packet wasapiPacket, onFrame func(TimedPCMBuffer) error) error {
	if packet.frames == 0 {
		return nil
	}
	if packet.silent || packet.data == nil {
		packet.decoded = make([]float32, int(packet.frames)*format.channels)
	} else {
		decoded, err := decodeWasapiInterleaved(packet.data, packet.frames, format)
		if err != nil {
			return err
		}
		packet.decoded = decoded
	}
	return s.emitDecoded(format, resampler, packet, onFrame)
}

func (s *wasapiCaptureSource) emitDecoded(format wasapiInputFormat, resampler *MonoResampler, packet wasapiPacket, onFrame func(TimedPCMBuffer) error) error {
	timestamp := time.Duration(packet.devicePosition) * time.Second / time.Duration(format.sampleRate)
	if s.kind == StreamMicrophone {
		mono := downmixMono(packet.decoded, format.channels, s.gain)
		samples := resampler.Convert(mono)
		if len(samples) == 0 {
			return nil
		}
		duration := time.Duration(len(samples)) * time.Second / time.Duration(s.targetRate)
		return onFrame(TimedPCMBuffer{
			Buffer: PCMBuffer{
				Kind:       StreamMicrophone,
				SampleRate: s.targetRate,
				Channels:   1,
				Samples:    samples,
			},
			Timestamp: timestamp,
			Duration:  duration,
		})
	}
	duration := time.Duration(packet.frames) * time.Second / time.Duration(format.sampleRate)
	return onFrame(TimedPCMBuffer{
		Buffer: PCMBuffer{
			Kind:       StreamSystemAudio,
			SampleRate: format.sampleRate,
			Channels:   format.channels,
			Samples:    packet.decoded,
		},
		Timestamp: timestamp,
		Duration:  duration,
	})
}

func (s *wasapiCaptureSource) setErr(err error) {
	if err == nil {
		return
	}
	s.errMu.Lock()
	defer s.errMu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

type wasapiPacket struct {
	data           *byte
	decoded        []float32
	frames         uint32
	flags          uint32
	devicePosition uint64
	silent         bool
}

func wasapiCoInitialize() error {
	hr, _, _ := wasapiProcCoInitializeEx.Call(0, wasapiCoInitApartmentThreaded)
	if wasapiFailed(hr) {
		return fmt.Errorf("CoInitializeEx failed: %s", wasapiHRESULTString(hr))
	}
	return nil
}

func wasapiCreateMMDeviceEnumerator() (*wasapiMMDeviceEnumerator, error) {
	var enumerator *wasapiMMDeviceEnumerator
	hr, _, _ := wasapiProcCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&wasapiCLSIDMMDeviceEnumerator)),
		0,
		wasapiClsctxAll,
		uintptr(unsafe.Pointer(&wasapiIIDIMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enumerator)),
	)
	if wasapiFailed(hr) {
		return nil, fmt.Errorf("CoCreateInstance(MMDeviceEnumerator) failed: %s", wasapiHRESULTString(hr))
	}
	if enumerator == nil {
		return nil, fmt.Errorf("CoCreateInstance(MMDeviceEnumerator) returned nil enumerator")
	}
	return enumerator, nil
}

func (e *wasapiMMDeviceEnumerator) resolveDevice(kind StreamKind, deviceID string) (*wasapiMMDevice, error) {
	flow := uintptr(wasapiECapture)
	prefix := "microphone:wasapi:"
	if kind == StreamSystemAudio {
		flow = wasapiERender
		prefix = "system-audio:wasapi:"
	}
	nativeID := strings.TrimSpace(strings.TrimPrefix(deviceID, prefix))
	if nativeID != "" && nativeID != deviceID && nativeID != "default" {
		device, err := e.device(nativeID)
		if err == nil {
			return device, nil
		}
	}
	return e.defaultEndpoint(flow)
}

func (e *wasapiMMDeviceEnumerator) device(deviceID string) (*wasapiMMDevice, error) {
	raw, err := windows.UTF16PtrFromString(deviceID)
	if err != nil {
		return nil, err
	}
	var device *wasapiMMDevice
	hr, _, _ := syscall.SyscallN(
		e.lpVtbl.GetDevice,
		uintptr(unsafe.Pointer(e)),
		uintptr(unsafe.Pointer(raw)),
		uintptr(unsafe.Pointer(&device)),
	)
	if wasapiFailed(hr) {
		return nil, fmt.Errorf("IMMDeviceEnumerator::GetDevice failed: %s", wasapiHRESULTString(hr))
	}
	if device == nil {
		return nil, fmt.Errorf("IMMDeviceEnumerator::GetDevice returned nil device")
	}
	return device, nil
}

func (e *wasapiMMDeviceEnumerator) defaultEndpoint(flow uintptr) (*wasapiMMDevice, error) {
	var device *wasapiMMDevice
	hr, _, _ := syscall.SyscallN(
		e.lpVtbl.GetDefaultAudioEndpoint,
		uintptr(unsafe.Pointer(e)),
		flow,
		wasapiEConsole,
		uintptr(unsafe.Pointer(&device)),
	)
	if wasapiFailed(hr) {
		return nil, fmt.Errorf("IMMDeviceEnumerator::GetDefaultAudioEndpoint failed: %s", wasapiHRESULTString(hr))
	}
	if device == nil {
		return nil, fmt.Errorf("IMMDeviceEnumerator::GetDefaultAudioEndpoint returned nil device")
	}
	return device, nil
}

func (e *wasapiMMDeviceEnumerator) release() {
	syscall.SyscallN(e.lpVtbl.Release, uintptr(unsafe.Pointer(e)))
}

func (d *wasapiMMDevice) activateAudioClient() (*wasapiAudioClient, error) {
	var client *wasapiAudioClient
	hr, _, _ := syscall.SyscallN(
		d.lpVtbl.Activate,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(&wasapiIIDIAudioClient)),
		wasapiClsctxAll,
		0,
		uintptr(unsafe.Pointer(&client)),
	)
	if wasapiFailed(hr) {
		return nil, fmt.Errorf("IMMDevice::Activate(IAudioClient) failed: %s", wasapiHRESULTString(hr))
	}
	if client == nil {
		return nil, fmt.Errorf("IMMDevice::Activate(IAudioClient) returned nil client")
	}
	return client, nil
}

func (d *wasapiMMDevice) release() {
	syscall.SyscallN(d.lpVtbl.Release, uintptr(unsafe.Pointer(d)))
}

func (c *wasapiAudioClient) mixFormat() (*wasapiWaveFormatEx, wasapiInputFormat, error) {
	var raw *wasapiWaveFormatEx
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.GetMixFormat,
		uintptr(unsafe.Pointer(c)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if wasapiFailed(hr) {
		return nil, wasapiInputFormat{}, fmt.Errorf("IAudioClient::GetMixFormat failed: %s", wasapiHRESULTString(hr))
	}
	if raw == nil {
		return nil, wasapiInputFormat{}, fmt.Errorf("IAudioClient::GetMixFormat returned nil format")
	}
	format, err := parseWasapiFormat(raw)
	return raw, format, err
}

func (c *wasapiAudioClient) initialize(format *wasapiWaveFormatEx, streamFlags uintptr) error {
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.Initialize,
		uintptr(unsafe.Pointer(c)),
		wasapiShareModeShared,
		streamFlags,
		wasapiBufferDurationHNS,
		0,
		uintptr(unsafe.Pointer(format)),
		0,
	)
	if wasapiFailed(hr) {
		return fmt.Errorf("IAudioClient::Initialize failed: %s", wasapiHRESULTString(hr))
	}
	return nil
}

func (c *wasapiAudioClient) captureClient() (*wasapiAudioCaptureClient, error) {
	var captureClient *wasapiAudioCaptureClient
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.GetService,
		uintptr(unsafe.Pointer(c)),
		uintptr(unsafe.Pointer(&wasapiIIDIAudioCaptureClient)),
		uintptr(unsafe.Pointer(&captureClient)),
	)
	if wasapiFailed(hr) {
		return nil, fmt.Errorf("IAudioClient::GetService(IAudioCaptureClient) failed: %s", wasapiHRESULTString(hr))
	}
	if captureClient == nil {
		return nil, fmt.Errorf("IAudioClient::GetService returned nil capture client")
	}
	return captureClient, nil
}

func (c *wasapiAudioClient) start() error {
	hr, _, _ := syscall.SyscallN(c.lpVtbl.Start, uintptr(unsafe.Pointer(c)))
	if wasapiFailed(hr) {
		return fmt.Errorf("IAudioClient::Start failed: %s", wasapiHRESULTString(hr))
	}
	return nil
}

func (c *wasapiAudioClient) stop() {
	syscall.SyscallN(c.lpVtbl.Stop, uintptr(unsafe.Pointer(c)))
}

func (c *wasapiAudioClient) release() {
	syscall.SyscallN(c.lpVtbl.Release, uintptr(unsafe.Pointer(c)))
}

func (c *wasapiAudioCaptureClient) nextPacketSize() (uint32, error) {
	var packetFrames uint32
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.GetNextPacketSize,
		uintptr(unsafe.Pointer(c)),
		uintptr(unsafe.Pointer(&packetFrames)),
	)
	if wasapiFailed(hr) {
		return 0, fmt.Errorf("IAudioCaptureClient::GetNextPacketSize failed: %s", wasapiHRESULTString(hr))
	}
	return packetFrames, nil
}

func (c *wasapiAudioCaptureClient) buffer() (wasapiPacket, error) {
	var data *byte
	var frames uint32
	var flags uint32
	var devicePosition uint64
	var qpcPosition uint64
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.GetBuffer,
		uintptr(unsafe.Pointer(c)),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&frames)),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Pointer(&devicePosition)),
		uintptr(unsafe.Pointer(&qpcPosition)),
	)
	_ = qpcPosition
	if wasapiFailed(hr) {
		return wasapiPacket{}, fmt.Errorf("IAudioCaptureClient::GetBuffer failed: %s", wasapiHRESULTString(hr))
	}
	return wasapiPacket{
		data:           data,
		frames:         frames,
		flags:          flags,
		devicePosition: devicePosition,
		silent:         flags&wasapiBufferFlagsSilent != 0 || data == nil,
	}, nil
}

func (c *wasapiAudioCaptureClient) releaseBuffer(frames uint32) error {
	hr, _, _ := syscall.SyscallN(
		c.lpVtbl.ReleaseBuffer,
		uintptr(unsafe.Pointer(c)),
		uintptr(frames),
	)
	if wasapiFailed(hr) {
		return fmt.Errorf("IAudioCaptureClient::ReleaseBuffer failed: %s", wasapiHRESULTString(hr))
	}
	return nil
}

func (c *wasapiAudioCaptureClient) release() {
	syscall.SyscallN(c.lpVtbl.Release, uintptr(unsafe.Pointer(c)))
}

func parseWasapiFormat(raw *wasapiWaveFormatEx) (wasapiInputFormat, error) {
	format := wasapiInputFormat{
		sampleRate:    int(raw.SamplesPerSec),
		channels:      int(raw.Channels),
		bitsPerSample: int(raw.BitsPerSample),
		blockAlign:    int(raw.BlockAlign),
	}
	switch raw.FormatTag {
	case waveFormatIEEEFloat:
		format.float32 = true
	case waveFormatPCM:
		format.pcm = true
	case waveFormatExtensible:
		subFormat, err := wasapiExtensibleSubFormat(raw)
		if err != nil {
			return wasapiInputFormat{}, err
		}
		switch subFormat {
		case wasapiSubFormatIEEEFloat:
			format.float32 = true
		case wasapiSubFormatPCM:
			format.pcm = true
		default:
			return wasapiInputFormat{}, fmt.Errorf("unsupported WASAPI extensible subtype: %s", subFormat.String())
		}
	default:
		return wasapiInputFormat{}, fmt.Errorf("unsupported WASAPI wave format tag: 0x%04X", raw.FormatTag)
	}
	if format.sampleRate <= 0 || format.channels <= 0 || format.blockAlign <= 0 {
		return wasapiInputFormat{}, fmt.Errorf("invalid WASAPI format: %dHz %dch blockAlign=%d", format.sampleRate, format.channels, format.blockAlign)
	}
	if format.float32 && format.bitsPerSample != 32 {
		return wasapiInputFormat{}, fmt.Errorf("unsupported WASAPI float depth: %d", format.bitsPerSample)
	}
	if format.pcm {
		switch format.bitsPerSample {
		case 16, 24, 32:
		default:
			return wasapiInputFormat{}, fmt.Errorf("unsupported WASAPI PCM depth: %d", format.bitsPerSample)
		}
	}
	return format, nil
}

func wasapiExtensibleSubFormat(raw *wasapiWaveFormatEx) (windows.GUID, error) {
	if raw.ExtraSize < 22 {
		return windows.GUID{}, fmt.Errorf("WASAPI extensible format has cbSize=%d, want at least 22", raw.ExtraSize)
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(raw)), 18+int(raw.ExtraSize))
	if len(data) < 40 {
		return windows.GUID{}, fmt.Errorf("WASAPI extensible format has %d bytes, want at least 40", len(data))
	}
	var guid windows.GUID
	guid.Data1 = binary.LittleEndian.Uint32(data[24:28])
	guid.Data2 = binary.LittleEndian.Uint16(data[28:30])
	guid.Data3 = binary.LittleEndian.Uint16(data[30:32])
	copy(guid.Data4[:], data[32:40])
	return guid, nil
}

func decodeWasapiInterleaved(data *byte, frames uint32, format wasapiInputFormat) ([]float32, error) {
	byteCount := int(frames) * format.blockAlign
	if data == nil || byteCount == 0 {
		return make([]float32, int(frames)*format.channels), nil
	}
	raw := unsafe.Slice(data, byteCount)
	samples := make([]float32, int(frames)*format.channels)
	switch {
	case format.float32:
		for index := range samples {
			offset := index * 4
			samples[index] = math.Float32frombits(binary.LittleEndian.Uint32(raw[offset : offset+4]))
		}
	case format.pcm && format.bitsPerSample == 16:
		for index := range samples {
			offset := index * 2
			samples[index] = float32(int16(binary.LittleEndian.Uint16(raw[offset:offset+2]))) / 32768
		}
	case format.pcm && format.bitsPerSample == 24:
		for index := range samples {
			offset := index * 3
			value := int32(raw[offset]) | int32(raw[offset+1])<<8 | int32(raw[offset+2])<<16
			if value&0x800000 != 0 {
				value |= ^0xFFFFFF
			}
			samples[index] = float32(value) / 8388608
		}
	case format.pcm && format.bitsPerSample == 32:
		for index := range samples {
			offset := index * 4
			samples[index] = float32(int32(binary.LittleEndian.Uint32(raw[offset:offset+4]))) / 2147483648
		}
	default:
		return nil, fmt.Errorf("unsupported WASAPI sample format")
	}
	return samples, nil
}

func downmixMono(interleaved []float32, channels int, gain float64) []float32 {
	if channels <= 0 {
		return nil
	}
	frames := len(interleaved) / channels
	mono := make([]float32, frames)
	for frame := 0; frame < frames; frame++ {
		sum := float64(0)
		for channel := 0; channel < channels; channel++ {
			sum += float64(interleaved[frame*channels+channel])
		}
		value := sum / float64(channels) * gain
		if value > 1 {
			value = 1
		} else if value < -1 {
			value = -1
		}
		mono[frame] = float32(value)
	}
	return mono
}

func wasapiFailed(hr uintptr) bool {
	return int32(hr) < 0
}

func wasapiHRESULTString(hr uintptr) string {
	return fmt.Sprintf("0x%08X", uint32(hr))
}
