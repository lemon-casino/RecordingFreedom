package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const pipOverlayPadding = 24

type PIPOverlayRequest struct {
	Config     pip.Config `json:"config"`
	Mode       string     `json:"mode,omitempty"`
	CameraName string     `json:"cameraName,omitempty"`
	Camera     PIPCamera  `json:"camera,omitempty"`
}

type PIPOverlayState struct {
	Config          pip.Config    `json:"config"`
	Placement       pip.Placement `json:"placement"`
	OverlayBounds   RegionRect    `json:"overlayBounds"`
	WindowBounds    RegionRect    `json:"windowBounds"`
	ContentBounds   RegionRect    `json:"contentBounds"`
	Mode            string        `json:"mode"`
	CameraName      string        `json:"cameraName,omitempty"`
	Camera          PIPCamera     `json:"camera,omitempty"`
	CaptureExcluded bool          `json:"captureExcluded"`
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
	if config.Preset == pip.PresetOff {
		_ = s.HidePIPOverlay()
		return s.pipOverlayState(config, req.Mode, camera)
	}
	state, err := s.pipOverlayState(config, req.Mode, camera)
	if err != nil {
		return PIPOverlayState{}, err
	}
	s.applyPIPOverlayState(state)
	return state, nil
}

func (s *RecordingFreedomService) UpdatePIPOverlay(req PIPOverlayRequest) (PIPOverlayState, error) {
	config := pip.NormalizeConfig(req.Config)
	camera := normalizePIPCamera(req.Camera, req.CameraName)
	if err := s.persistCameraPIPConfig(config); err != nil {
		return PIPOverlayState{}, err
	}
	if err := s.recorder.PatchActiveCameraPIP(config); err != nil {
		return PIPOverlayState{}, err
	}
	if config.Preset == pip.PresetOff {
		_ = s.HidePIPOverlay()
		return s.pipOverlayState(config, req.Mode, camera)
	}
	state, err := s.pipOverlayState(config, req.Mode, camera)
	if err != nil {
		return PIPOverlayState{}, err
	}
	s.applyPIPOverlayState(state)
	return state, nil
}

func (s *RecordingFreedomService) HidePIPOverlay() error {
	if s.pipOverlay == nil {
		return nil
	}
	s.pipOverlay.Hide()
	return nil
}

func (s *RecordingFreedomService) showRecordingPIPOverlay(req recording.StartRequest) {
	if !req.Camera.Enabled {
		return
	}
	config := pip.NormalizeConfigForPreset(req.Camera.PIPPreset, req.Camera.PIP)
	if config.Preset == pip.PresetOff {
		return
	}
	_, _ = s.ShowPIPOverlay(PIPOverlayRequest{
		Config: config,
		Mode:   "recording",
		Camera: s.pipCameraFromRecordingRequest(req.Camera),
	})
}

func (s *RecordingFreedomService) persistCameraPIPConfig(config pip.Config) error {
	if s.settings == nil {
		return nil
	}
	current, err := s.settings.Load()
	if err != nil {
		return err
	}
	current.Camera.PIP = config
	current.Camera.PIPPreset = string(config.Preset)
	if config.Preset != pip.PresetOff {
		current.Camera.Enabled = true
	}
	saved, err := s.settings.Save(current)
	if err != nil {
		return err
	}
	s.emitSettingsChanged(saved)
	return nil
}

func (s *RecordingFreedomService) pipOverlayState(config pip.Config, mode string, camera PIPCamera) (PIPOverlayState, error) {
	config = pip.NormalizeConfig(config)
	camera = normalizePIPCamera(camera, "")
	overlayBounds := application.Rect{Width: 1280, Height: 720}
	if s.app != nil {
		overlayBounds, _ = regionOverlayBounds(s.app.Screen.GetAll())
	}
	placement, err := pip.Place(config, pip.Size{Width: overlayBounds.Width, Height: overlayBounds.Height})
	if err != nil {
		return PIPOverlayState{}, err
	}
	size := placement.Rect.Width
	if size <= 0 {
		size = 96
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
		Mode:       normalizePIPOverlayMode(mode),
		CameraName: cameraDisplayName(camera),
		Camera:     camera,
	}, nil
}

func (s *RecordingFreedomService) applyPIPOverlayState(state PIPOverlayState) {
	if s.pipOverlay == nil {
		return
	}
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
	state.CaptureExcluded = setWindowCaptureExcluded(s.pipOverlay, true)
	s.broadcastPIPOverlayState(state)
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
