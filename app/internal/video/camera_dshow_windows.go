//go:build windows

package video

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

type directShowCameraSession struct {
	config CameraCaptureConfig
	writer *ffmpegDesktopSession
}

func NewPlatformCameraSession(config CameraCaptureConfig) (CameraSession, error) {
	config = NormalizeCameraCaptureConfig(config)
	if config.OutputPath == "" {
		return nil, errors.New("DirectShow camera output path is required")
	}
	if config.DeviceNativeID == "" {
		return nil, errors.New("DirectShow camera native device name is required")
	}
	writer, err := newFFmpegDesktopSession(CaptureConfig{
		Backend:    config.Backend,
		SourceID:   config.DeviceID,
		SourceName: config.DeviceNativeID,
		OutputPath: config.OutputPath,
		Profile:    config.Profile,
	}, directShowCameraInputArgs(config))
	if err != nil {
		return nil, err
	}
	return &directShowCameraSession{
		config: config,
		writer: writer,
	}, nil
}

func directShowCameraInputArgs(camera CameraCaptureConfig) ffmpegInputArgsBuilder {
	camera = NormalizeCameraCaptureConfig(camera)
	return func(config CaptureConfig) (ffmpegInputSpec, error) {
		if camera.DeviceNativeID == "" {
			return ffmpegInputSpec{}, errors.New("DirectShow camera native device name is required")
		}
		config = NormalizeCaptureConfig(config)
		return ffmpegInputSpec{
			Args: []string{
				"-rtbufsize", "256M",
				"-f", "dshow",
				"-framerate", fmt.Sprintf("%d", config.Profile.FPS),
				"-i", "video=" + camera.DeviceNativeID,
			},
			Engine:            "windows-dshow-camera",
			PreviewImagePath:  camera.PreviewImagePath,
			PreviewImageFPS:   8,
			PreviewImageWidth: 360,
		}, nil
	}
}

func (s *directShowCameraSession) Start(ctx context.Context) error {
	if s == nil || s.writer == nil {
		return errors.New("DirectShow camera session is not initialized")
	}
	return s.writer.Start(ctx)
}

func (s *directShowCameraSession) Pause() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Pause()
}

func (s *directShowCameraSession) Resume() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Resume()
}

func (s *directShowCameraSession) Stop() error {
	if s == nil || s.writer == nil {
		return nil
	}
	return s.writer.Stop()
}

func (s *directShowCameraSession) Diagnostics() TrackDiagnostics {
	if s == nil || s.writer == nil {
		return TrackDiagnostics{}
	}
	diagnostics := s.writer.Diagnostics()
	track := diagnostics.Screen
	track.Path = filepath.Base(s.config.OutputPath)
	if track.Message == "" {
		track.Message = "DirectShow camera sidecar writer finalized."
	}
	return track
}
