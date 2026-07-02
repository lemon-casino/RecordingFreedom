//go:build darwin

package video

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type avFoundationCameraSession struct {
	config CameraCaptureConfig
	writer *ffmpegDesktopSession
}

func NewPlatformCameraSession(config CameraCaptureConfig) (CameraSession, error) {
	config = NormalizeCameraCaptureConfig(config)
	if config.OutputPath == "" {
		return nil, errors.New("AVFoundation camera output path is required")
	}
	if config.DeviceNativeID == "" {
		return nil, errors.New("AVFoundation camera native device id is required")
	}
	writer, err := newFFmpegDesktopSession(CaptureConfig{
		Backend:    config.Backend,
		SourceID:   config.DeviceID,
		SourceName: config.DeviceNativeID,
		OutputPath: config.OutputPath,
		Profile:    config.Profile,
	}, avFoundationCameraInputArgs(config))
	if err != nil {
		return nil, err
	}
	return &avFoundationCameraSession{
		config: config,
		writer: writer,
	}, nil
}

func avFoundationCameraInputArgs(camera CameraCaptureConfig) ffmpegInputArgsBuilder {
	camera = NormalizeCameraCaptureConfig(camera)
	return func(config CaptureConfig) ([]string, error) {
		if camera.DeviceNativeID == "" {
			return nil, errors.New("AVFoundation camera native device id is required")
		}
		config = NormalizeCaptureConfig(config)
		return []string{
			"-f", "avfoundation",
			"-framerate", fmt.Sprintf("%d", config.Profile.FPS),
			"-i", avFoundationCameraInput(camera.DeviceNativeID),
		}, nil
	}
}

func avFoundationCameraInput(nativeID string) string {
	nativeID = strings.TrimSpace(nativeID)
	if nativeID == "" || nativeID == "default" {
		nativeID = "0"
	}
	if strings.Contains(nativeID, ":") {
		return nativeID
	}
	return nativeID + ":none"
}

func (s *avFoundationCameraSession) Start(ctx context.Context) error {
	if s == nil || s.writer == nil {
		return errors.New("AVFoundation camera session is not initialized")
	}
	return s.writer.Start(ctx)
}

func (s *avFoundationCameraSession) Pause() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Pause()
}

func (s *avFoundationCameraSession) Resume() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Resume()
}

func (s *avFoundationCameraSession) Stop() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Stop()
}

func (s *avFoundationCameraSession) Diagnostics() TrackDiagnostics {
	if s == nil || s.writer == nil {
		return TrackDiagnostics{}
	}
	diagnostics := s.writer.Diagnostics()
	track := diagnostics.Screen
	track.Path = filepath.Base(s.config.OutputPath)
	if track.Message == "" {
		track.Message = "AVFoundation camera sidecar writer finalized."
	}
	return track
}
