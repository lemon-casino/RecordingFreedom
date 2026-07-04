package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const pipOverlayPadding = 24

const (
	releasePIPOverlayMediaScript = `window.__RF_PIP_STOP_TOKEN__=(window.__RF_PIP_STOP_TOKEN__||0)+1;if(window.__RF_STOP_PIP_CAMERA__){window.__RF_STOP_PIP_CAMERA__();}document.querySelectorAll("video").forEach((video)=>{try{video.pause();}catch(e){}const stream=video.srcObject;if(stream&&typeof stream.getTracks==="function"){stream.getTracks().forEach((track)=>track.stop());}video.srcObject=null;video.removeAttribute("src");try{video.load();}catch(e){}});`
	stopPIPOverlayMediaScript    = releasePIPOverlayMediaScript + `window.__RF_PIP_OVERLAY__=undefined;`
	pipOverlayCameraReleaseDelay = 350 * time.Millisecond
)

type PIPOverlayRequest struct {
	Config            pip.Config `json:"config"`
	Mode              string     `json:"mode,omitempty"`
	CameraName        string     `json:"cameraName,omitempty"`
	Camera            PIPCamera  `json:"camera,omitempty"`
	PreviewImagePath  string     `json:"previewImagePath,omitempty"`
	ClientOperationID uint64     `json:"clientOperationId,omitempty"`
}

type PIPOverlayState struct {
	Config            pip.Config    `json:"config"`
	Placement         pip.Placement `json:"placement"`
	OverlayBounds     RegionRect    `json:"overlayBounds"`
	WindowBounds      RegionRect    `json:"windowBounds"`
	ContentBounds     RegionRect    `json:"contentBounds"`
	Mode              string        `json:"mode"`
	CameraName        string        `json:"cameraName,omitempty"`
	Camera            PIPCamera     `json:"camera,omitempty"`
	PreviewImagePath  string        `json:"previewImagePath,omitempty"`
	CaptureExcluded   bool          `json:"captureExcluded"`
	ClientOperationID uint64        `json:"clientOperationId,omitempty"`
}

type PIPCamera struct {
	DeviceID string `json:"deviceId,omitempty"`
	NativeID string `json:"nativeId,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (s *RecordingFreedomService) ShowPIPOverlay(req PIPOverlayRequest) (PIPOverlayState, error) {
	if s.app == nil || s.pipOverlay == nil {
		return PIPOverlayState{}, errors.New("PIP overlay window is not configured")
	}
	config := pip.NormalizeConfig(req.Config)
	camera := normalizePIPCamera(req.Camera, req.CameraName)
	s.logEvent("pip-overlay", "show-request", map[string]string{
		"preset":   string(config.Preset),
		"mode":     normalizePIPOverlayMode(req.Mode),
		"cameraId": camera.DeviceID,
		"nativeId": camera.NativeID,
		"name":     camera.Name,
	})
	if config.Preset == pip.PresetOff {
		_ = s.HidePIPOverlay()
		state, err := s.pipOverlayState(config, req.Mode, camera, req.PreviewImagePath)
		state.ClientOperationID = req.ClientOperationID
		return state, err
	}
	state, err := s.pipOverlayState(config, req.Mode, camera, req.PreviewImagePath)
	if err != nil {
		return PIPOverlayState{}, err
	}
	state.ClientOperationID = req.ClientOperationID
	s.applyPIPOverlayState(state)
	return state, nil
}

func (s *RecordingFreedomService) UpdatePIPOverlay(req PIPOverlayRequest) (PIPOverlayState, error) {
	config := pip.NormalizeConfig(req.Config)
	camera := normalizePIPCamera(req.Camera, req.CameraName)
	s.logEvent("pip-overlay", "update-request", map[string]string{
		"preset":   string(config.Preset),
		"mode":     normalizePIPOverlayMode(req.Mode),
		"cameraId": camera.DeviceID,
		"nativeId": camera.NativeID,
		"name":     camera.Name,
	})
	if err := s.persistCameraPIPConfig(config); err != nil {
		return PIPOverlayState{}, err
	}
	if err := s.recorder.PatchActiveCameraPIP(config); err != nil {
		return PIPOverlayState{}, err
	}
	if config.Preset == pip.PresetOff {
		_ = s.HidePIPOverlay()
		state, err := s.pipOverlayState(config, req.Mode, camera, req.PreviewImagePath)
		state.ClientOperationID = req.ClientOperationID
		return state, err
	}
	state, err := s.pipOverlayState(config, req.Mode, camera, req.PreviewImagePath)
	if err != nil {
		return PIPOverlayState{}, err
	}
	state.ClientOperationID = req.ClientOperationID
	s.applyPIPOverlayState(state)
	return state, nil
}

func (s *RecordingFreedomService) HidePIPOverlay() error {
	if s.pipOverlay == nil {
		return nil
	}
	s.logEvent("pip-overlay", "hide", nil)
	s.nextPIPOverlayToken()
	s.pipOverlay.ExecJS(stopPIPOverlayMediaScript)
	if state, err := s.pipOverlayState(pip.OffConfig(), "edit", PIPCamera{}, ""); err == nil {
		s.broadcastPIPOverlayState(state)
	}
	s.pipOverlay.ExecJS(stopPIPOverlayMediaScript)
	s.pipOverlay.Hide()
	return nil
}

func (s *RecordingFreedomService) showRecordingPIPOverlay(req recording.StartRequest, session recording.Session) {
	if !req.Camera.Enabled {
		return
	}
	config := pip.NormalizeConfigForPreset(req.Camera.PIPPreset, req.Camera.PIP)
	if config.Preset == pip.PresetOff {
		return
	}
	s.logEvent("pip-overlay", "show-recording", map[string]string{
		"preset":   string(config.Preset),
		"cameraId": strings.TrimSpace(req.Camera.DeviceID),
		"nativeId": strings.TrimSpace(req.Camera.DeviceNativeID),
	})
	_, _ = s.ShowPIPOverlay(PIPOverlayRequest{
		Config:           config,
		Mode:             "recording",
		Camera:           s.pipCameraFromRecordingRequest(req.Camera),
		PreviewImagePath: recording.CameraPreviewImagePath(session.PackageDir),
	})
}

func (s *RecordingFreedomService) releasePIPOverlayMediaForRecording(req recording.StartRequest) {
	if !req.Camera.Enabled || s.pipOverlay == nil {
		return
	}
	s.logEvent("pip-overlay", "release-media-for-recording", map[string]string{
		"cameraId": strings.TrimSpace(req.Camera.DeviceID),
		"nativeId": strings.TrimSpace(req.Camera.DeviceNativeID),
	})
	s.pipOverlay.ExecJS(releasePIPOverlayMediaScript)
	time.Sleep(pipOverlayCameraReleaseDelay)
}

func (s *RecordingFreedomService) persistCameraPIPConfig(config pip.Config) error {
	if s.settings == nil {
		return nil
	}
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	current, err := s.settings.Load()
	if err != nil {
		return err
	}
	current.Camera.PIP = config
	current.Camera.PIPPreset = string(config.Preset)
	if config.Preset == pip.PresetOff {
		current.Camera.Enabled = false
	} else {
		current.Camera.Enabled = true
	}
	saved, err := s.settings.Save(current)
	if err != nil {
		return err
	}
	s.emitSettingsChanged(saved)
	return nil
}

func (s *RecordingFreedomService) pipOverlayState(config pip.Config, mode string, camera PIPCamera, previewImagePath string) (PIPOverlayState, error) {
	config = pip.NormalizeConfig(config)
	camera = normalizePIPCamera(camera, "")
	previewImagePath = strings.TrimSpace(previewImagePath)
	overlayBounds := s.pipOverlayCanvasBounds()
	placement, err := pip.Place(config, pip.Size{Width: overlayBounds.Width, Height: overlayBounds.Height})
	if err != nil {
		return PIPOverlayState{}, err
	}
	size := placement.Rect.Width
	if size <= 0 {
		size = pip.MinimumPixelSize
	}
	windowBounds := application.Rect{
		X:      overlayBounds.X + placement.Rect.X - pipOverlayPadding,
		Y:      overlayBounds.Y + placement.Rect.Y - pipOverlayPadding,
		Width:  size + pipOverlayPadding*2,
		Height: size + pipOverlayPadding*2,
	}
	return PIPOverlayState{
		Config:        config,
		Placement:     placement,
		OverlayBounds: regionRectFromAppRect(overlayBounds),
		WindowBounds:  regionRectFromAppRect(windowBounds),
		ContentBounds: RegionRect{
			X:      pipOverlayPadding,
			Y:      pipOverlayPadding,
			Width:  size,
			Height: size,
		},
		Mode:             normalizePIPOverlayMode(mode),
		CameraName:       cameraDisplayName(camera),
		Camera:           camera,
		PreviewImagePath: previewImagePath,
	}, nil
}

func (s *RecordingFreedomService) pipOverlayCanvasBounds() application.Rect {
	if s == nil || s.app == nil {
		return application.Rect{Width: 1280, Height: 720}
	}
	screens := s.app.Screen.GetAll()
	capsuleBounds, hasCapsuleBounds := s.currentCapsuleWindowBounds()
	if bounds, ok := pipOverlayBoundsForCapsule(screens, capsuleBounds, hasCapsuleBounds); ok {
		return bounds
	}
	bounds, _ := regionOverlayBounds(screens)
	return bounds
}

func (s *RecordingFreedomService) currentCapsuleWindowBounds() (application.Rect, bool) {
	if s == nil || s.capsuleWindow == nil {
		return application.Rect{}, false
	}
	bounds := s.capsuleWindow.Bounds()
	if validAppRect(bounds) {
		return bounds, true
	}
	x, y := s.capsuleWindow.Position()
	width, height := s.capsuleWindow.Size()
	if width <= 0 {
		width = capsuleWindowWidth
	}
	if height <= 0 {
		height = capsuleWindowCollapsedHeight
	}
	return application.Rect{X: x, Y: y, Width: width, Height: height}, true
}

func pipOverlayBoundsForCapsule(screens []*application.Screen, capsuleBounds application.Rect, hasCapsuleBounds bool) (application.Rect, bool) {
	if !hasCapsuleBounds || !validAppRect(capsuleBounds) {
		return application.Rect{}, false
	}
	var best *application.Screen
	bestArea := 0
	for _, screen := range screens {
		if screen == nil || !validAppRect(screen.Bounds) {
			continue
		}
		area := rectIntersectionArea(screen.Bounds, capsuleBounds)
		if area > bestArea {
			bestArea = area
			best = screen
		}
	}
	if best != nil && bestArea > 0 {
		return pipOverlayUsableScreenBounds(best), true
	}

	centerX := capsuleBounds.X + maxInt(1, capsuleBounds.Width)/2
	centerY := capsuleBounds.Y + maxInt(1, capsuleBounds.Height)/2
	for _, screen := range screens {
		if screen == nil || !validAppRect(screen.Bounds) {
			continue
		}
		if pointInsideAppRect(centerX, centerY, screen.Bounds) {
			return pipOverlayUsableScreenBounds(screen), true
		}
	}

	var nearest *application.Screen
	var nearestDistance int64
	for _, screen := range screens {
		if screen == nil || !validAppRect(screen.Bounds) {
			continue
		}
		distance := distanceToAppRectSquared(centerX, centerY, screen.Bounds)
		if nearest == nil || distance < nearestDistance {
			nearest = screen
			nearestDistance = distance
		}
	}
	if nearest == nil {
		return application.Rect{}, false
	}
	return pipOverlayUsableScreenBounds(nearest), true
}

func pipOverlayUsableScreenBounds(screen *application.Screen) application.Rect {
	if screen == nil {
		return application.Rect{}
	}
	if validAppRect(screen.WorkArea) {
		return screen.WorkArea
	}
	return screen.Bounds
}

func validAppRect(rect application.Rect) bool {
	return rect.Width > 0 && rect.Height > 0
}

func pointInsideAppRect(x int, y int, rect application.Rect) bool {
	return x >= rect.X && y >= rect.Y && x < rect.X+rect.Width && y < rect.Y+rect.Height
}

func distanceToAppRectSquared(x int, y int, rect application.Rect) int64 {
	dx := 0
	if x < rect.X {
		dx = rect.X - x
	} else if x >= rect.X+rect.Width {
		dx = x - (rect.X + rect.Width - 1)
	}
	dy := 0
	if y < rect.Y {
		dy = rect.Y - y
	} else if y >= rect.Y+rect.Height {
		dy = y - (rect.Y + rect.Height - 1)
	}
	return int64(dx*dx + dy*dy)
}

func (s *RecordingFreedomService) applyPIPOverlayState(state PIPOverlayState) {
	if s.pipOverlay == nil {
		return
	}
	token := s.nextPIPOverlayToken()
	windowBounds := application.Rect{
		X:      state.WindowBounds.X,
		Y:      state.WindowBounds.Y,
		Width:  state.WindowBounds.Width,
		Height: state.WindowBounds.Height,
	}
	s.pipOverlay.SetBounds(windowBounds)
	s.pipOverlay.SetAlwaysOnTop(true)
	s.pipOverlay.Show()
	s.pipOverlay.SetBounds(windowBounds)
	// FFmpeg desktop capture on Windows records content-protected transparent windows
	// as a black rectangle. Keep the PIP preview visible and transparent while the
	// export compositor uses the webcam sidecar for the final clean overlay.
	_ = setWindowCaptureExcluded(s.pipOverlay, false)
	state.CaptureExcluded = false
	s.broadcastPIPOverlayState(state)
	go s.rebroadcastPIPOverlayState(state, token)
}

func (s *RecordingFreedomService) rebroadcastPIPOverlayState(state PIPOverlayState, token uint64) {
	for _, delay := range []time.Duration{120 * time.Millisecond, 500 * time.Millisecond} {
		time.Sleep(delay)
		if !s.isPIPOverlayTokenCurrent(token) {
			return
		}
		s.broadcastPIPOverlayState(state)
	}
}

func (s *RecordingFreedomService) nextPIPOverlayToken() uint64 {
	s.pipOverlayMu.Lock()
	defer s.pipOverlayMu.Unlock()
	s.pipOverlayToken++
	return s.pipOverlayToken
}

func (s *RecordingFreedomService) isPIPOverlayTokenCurrent(token uint64) bool {
	s.pipOverlayMu.Lock()
	defer s.pipOverlayMu.Unlock()
	return token != 0 && s.pipOverlayToken == token
}

func (s *RecordingFreedomService) broadcastPIPOverlayState(state PIPOverlayState) {
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	script := fmt.Sprintf(
		"window.__RF_PIP_OVERLAY__=%s;window.dispatchEvent(new CustomEvent('rf-pip-overlay',{detail:window.__RF_PIP_OVERLAY__}));",
		string(payload),
	)
	if s.pipOverlay != nil {
		s.pipOverlay.ExecJS(script)
	}
}

func normalizePIPOverlayMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "recording":
		return "recording"
	default:
		return "edit"
	}
}

func (s *RecordingFreedomService) pipCameraFromRecordingRequest(camera recording.CameraRequest) PIPCamera {
	target := PIPCamera{
		DeviceID: strings.TrimSpace(camera.DeviceID),
		NativeID: strings.TrimSpace(camera.DeviceNativeID),
	}
	if s.devices != nil {
		for _, device := range s.devices.ListMediaDevices().Cameras {
			if device.ID == target.DeviceID || (target.DeviceID == "camera:default" && device.IsDefault) {
				target.DeviceID = device.ID
				target.NativeID = strings.TrimSpace(device.NativeID)
				target.Name = strings.TrimSpace(device.Name)
				break
			}
		}
	}
	return normalizePIPCamera(target, "")
}

func normalizePIPCamera(camera PIPCamera, fallbackName string) PIPCamera {
	camera.DeviceID = strings.TrimSpace(camera.DeviceID)
	camera.NativeID = strings.TrimSpace(camera.NativeID)
	camera.Name = strings.TrimSpace(camera.Name)
	if camera.Name == "" {
		camera.Name = strings.TrimSpace(fallbackName)
	}
	if camera.Name == "" {
		camera.Name = camera.NativeID
	}
	if camera.Name == "" {
		camera.Name = camera.DeviceID
	}
	return camera
}

func cameraDisplayName(camera PIPCamera) string {
	camera = normalizePIPCamera(camera, "")
	return camera.Name
}
