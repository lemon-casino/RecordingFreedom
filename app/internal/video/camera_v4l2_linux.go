//go:build linux

package video

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

type v4l2CameraSession struct {
	config CameraCaptureConfig
	writer *ffmpegDesktopSession
}

func NewPlatformCameraSession(config CameraCaptureConfig) (CameraSession, error) {
	config = NormalizeCameraCaptureConfig(config)
	if config.OutputPath == "" {
		return nil, errors.New("v4l2 camera output path is required")
	}
	if config.DeviceNativeID == "" {
		return nil, errors.New("v4l2 camera device path is required")
	}
	writer, err := newFFmpegDesktopSession(CaptureConfig{
		Backend:    config.Backend,
		SourceID:   config.DeviceID,
		SourceName: config.DeviceNativeID,
		OutputPath: config.OutputPath,
		Profile:    config.Profile,
	}, v4l2CameraInputArgs(config))
	if err != nil {
		return nil, err
	}
	return &v4l2CameraSession{
		config: config,
		writer: writer,
	}, nil
}

func v4l2CameraInputArgs(camera CameraCaptureConfig) ffmpegInputArgsBuilder {
	camera = NormalizeCameraCaptureConfig(camera)
	return func(config CaptureConfig) (ffmpegInputSpec, error) {
		if camera.DeviceNativeID == "" {
			return ffmpegInputSpec{}, errors.New("v4l2 camera device path is required")
		}
		config = NormalizeCaptureConfig(config)
		return ffmpegInputSpec{
			Args: []string{
				"-f", "v4l2",
				"-framerate", fmt.Sprintf("%d", config.Profile.FPS),
				"-i", camera.DeviceNativeID,
			},
			Engine: "v4l2-camera",
		}, nil
	}
}

func (s *v4l2CameraSession) Start(ctx context.Context) error {
	if s == nil || s.writer == nil {
		return errors.New("v4l2 camera session is not initialized")
	}
	return s.writer.Start(ctx)
}

func (s *v4l2CameraSession) Pause() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Pause()
}

func (s *v4l2CameraSession) Resume() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Resume()
}

func (s *v4l2CameraSession) Stop() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Stop()
}

func (s *v4l2CameraSession) Diagnostics() TrackDiagnostics {
	if s == nil || s.writer == nil {
		return TrackDiagnostics{}
	}
	diagnostics := s.writer.Diagnostics()
	track := diagnostics.Screen
	track.Path = filepath.Base(s.config.OutputPath)
	if track.Message == "" {
		track.Message = "v4l2 camera sidecar writer finalized."
	}
	return track
}
