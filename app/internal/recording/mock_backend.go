package recording

import (
	"context"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type MockBackend struct {
	packages *recpackage.Service
}

func NewMockBackend(packages *recpackage.Service) *MockBackend {
	if packages == nil {
		packages = recpackage.NewService()
	}
	return &MockBackend{packages: packages}
}

func (b *MockBackend) ID() string {
	return "mock-package"
}

func (b *MockBackend) Start(_ context.Context, req BackendStartRequest) (BackendStartResult, error) {
	pkg, err := b.packages.CreateMock(req.VideoDir, recpackage.CreateMockRequest{
		CreatedAt: req.CreatedAt,
		Status:    recpackage.StatusRecording,
		Source: recpackage.ManifestSource{
			Type: string(req.StartRequest.SourceType),
			ID:   req.StartRequest.SourceID,
			Name: req.StartRequest.SourceName,
		},
		Recording: req.StartRequest.Recording,
		Audio: recpackage.ManifestAudio{
			System:                     req.StartRequest.Audio.System,
			SystemDeviceID:             req.StartRequest.Audio.SystemDeviceID,
			Microphone:                 req.StartRequest.Audio.Microphone,
			MicrophoneDeviceID:         req.StartRequest.Audio.MicrophoneID,
			MicrophoneNoiseSuppression: noiseSuppressionLabel(req.StartRequest.Audio.NoiseSuppression),
			MicrophoneGain:             req.StartRequest.Audio.MicrophoneGain,
			MockPipeline:               true,
		},
		Camera: recpackage.ManifestCamera{
			Enabled:   req.StartRequest.Camera.Enabled,
			DeviceID:  req.StartRequest.Camera.DeviceID,
			PIPPreset: req.StartRequest.Camera.PIPPreset,
		},
	})
	if err != nil {
		return BackendStartResult{}, err
	}
	return BackendStartResult{Package: pkg}, nil
}

func (b *MockBackend) Pause(context.Context, BackendControlRequest) error {
	return nil
}

func (b *MockBackend) Resume(context.Context, BackendControlRequest) error {
	return nil
}

func (b *MockBackend) Stop(context.Context, BackendControlRequest) (BackendStopResult, error) {
	return BackendStopResult{}, nil
}
