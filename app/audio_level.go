package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
)

const audioLevelEmitInterval = 50 * time.Millisecond

type AudioLevelEvent struct {
	DeviceID string  `json:"deviceId"`
	Level    float64 `json:"level"`
	RMS      float64 `json:"rms"`
	Peak     float64 `json:"peak"`
	Active   bool    `json:"active"`
	Error    string  `json:"error,omitempty"`
}

type audioLevelCaptureSource interface {
	ID() string
	Kind() audio.StreamKind
	Start(func(audio.TimedPCMBuffer) error) error
	Pause() error
	Resume() error
	Stop() error
}

func (s *RecordingFreedomService) StartMicrophoneLevelMonitor(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = "microphone:default"
	}
	if err := s.StopMicrophoneLevelMonitor(); err != nil {
		return err
	}

	sources, err := audio.NewPlatformCaptureSources(audio.CaptureConfig{
		Backend:          "microphone-level-monitor",
		Microphone:       audio.StreamConfig{Enabled: true, DeviceID: deviceID},
		MicrophoneGain:   1,
		TargetSampleRate: audio.RNNoiseSampleRate,
		TargetChannels:   1,
	})
	if err != nil {
		s.emitAudioLevel(AudioLevelEvent{DeviceID: deviceID, Active: false, Error: err.Error()})
		return err
	}
	if len(sources) == 0 || sources[0] == nil {
		err := fmt.Errorf("microphone level monitor has no capture source")
		s.emitAudioLevel(AudioLevelEvent{DeviceID: deviceID, Active: false, Error: err.Error()})
		return err
	}
	source := sources[0]

	s.micLevelMu.Lock()
	s.micLevelToken++
	token := s.micLevelToken
	s.micLevelSource = source
	s.micLevelDevice = deviceID
	s.micLevelMu.Unlock()

	lastEmit := time.Time{}
	smoothedLevel := 0.0
	err = source.Start(func(frame audio.TimedPCMBuffer) error {
		level := audioLevelFromSamples(frame.Buffer.Samples)
		now := time.Now()
		if !lastEmit.IsZero() && now.Sub(lastEmit) < audioLevelEmitInterval {
			return nil
		}
		lastEmit = now
		if level.Level > smoothedLevel {
			smoothedLevel = level.Level
		} else {
			smoothedLevel = smoothedLevel*0.78 + level.Level*0.22
		}
		level.Level = clampUnit(smoothedLevel)
		s.emitAudioLevelForToken(token, level)
		return nil
	})
	if err != nil {
		s.clearAudioLevelSource(token)
		s.emitAudioLevel(AudioLevelEvent{DeviceID: deviceID, Active: false, Error: err.Error()})
		return err
	}

	s.emitAudioLevelForToken(token, AudioLevelEvent{DeviceID: deviceID, Active: true})
	return nil
}

func (s *RecordingFreedomService) StopMicrophoneLevelMonitor() error {
	s.micLevelMu.Lock()
	source := s.micLevelSource
	deviceID := s.micLevelDevice
	if deviceID == "" {
		deviceID = "microphone:default"
	}
	s.micLevelSource = nil
	s.micLevelDevice = ""
	s.micLevelToken++
	s.micLevelMu.Unlock()

	if source == nil {
		return nil
	}
	err := source.Stop()
	event := AudioLevelEvent{DeviceID: deviceID, Active: false}
	if err != nil {
		event.Error = err.Error()
	}
	s.emitAudioLevel(event)
	return err
}

func (s *RecordingFreedomService) clearAudioLevelSource(token uint64) {
	s.micLevelMu.Lock()
	defer s.micLevelMu.Unlock()
	if s.micLevelToken != token {
		return
	}
	s.micLevelSource = nil
	s.micLevelDevice = ""
	s.micLevelToken++
}

func (s *RecordingFreedomService) emitAudioLevelForToken(token uint64, event AudioLevelEvent) {
	s.micLevelMu.Lock()
	currentToken := s.micLevelToken
	deviceID := s.micLevelDevice
	s.micLevelMu.Unlock()
	if currentToken != token {
		return
	}
	if event.DeviceID == "" {
		event.DeviceID = deviceID
	}
	event.Active = true
	s.emitAudioLevel(event)
}

func (s *RecordingFreedomService) emitAudioLevel(event AudioLevelEvent) {
	if s.app == nil {
		return
	}
	s.app.Event.Emit("audio.level", event)
}

func audioLevelFromSamples(samples []float32) AudioLevelEvent {
	if len(samples) == 0 {
		return AudioLevelEvent{Active: true}
	}
	sumSquares := 0.0
	peak := 0.0
	for _, sample := range samples {
		value := math.Abs(float64(sample))
		if value > peak {
			peak = value
		}
		sumSquares += value * value
	}
	rms := math.Sqrt(sumSquares / float64(len(samples)))
	return AudioLevelEvent{
		Level:  clampUnit(math.Sqrt(clampUnit(rms)) * 1.18),
		RMS:    clampUnit(rms),
		Peak:   clampUnit(peak),
		Active: true,
	}
}

func clampUnit(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
