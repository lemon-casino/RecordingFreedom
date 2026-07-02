package recording

import "testing"

import (
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestNormalizeStartRequestDefaultsSelectedDevices(t *testing.T) {
	got, err := NormalizeStartRequest(StartRequest{
		SourceID:   " screen:primary ",
		SourceType: SourceScreen,
		SourceName: " Primary Display ",
		Audio: AudioRequest{
			System:           true,
			Microphone:       true,
			NoiseSuppression: true,
		},
		Camera: CameraRequest{Enabled: true, DeviceNativeID: " Integrated Camera "},
	})
	if err != nil {
		t.Fatalf("NormalizeStartRequest() error = %v", err)
	}
	if got.SourceID != "screen:primary" || got.SourceName != "Primary Display" {
		t.Fatalf("source normalized to id:%q name:%q", got.SourceID, got.SourceName)
	}
	if got.Recording != recordingprofile.Default() {
		t.Fatalf("recording profile = %#v, want %#v", got.Recording, recordingprofile.Default())
	}
	if got.Audio.SystemDeviceID != defaultSystemAudioID {
		t.Fatalf("system device id = %q, want %q", got.Audio.SystemDeviceID, defaultSystemAudioID)
	}
	if got.Audio.MicrophoneID != defaultMicrophoneID {
		t.Fatalf("microphone id = %q, want %q", got.Audio.MicrophoneID, defaultMicrophoneID)
	}
	if got.Audio.MicrophoneGain != defaultMicrophoneGain {
		t.Fatalf("microphone gain = %v, want %v", got.Audio.MicrophoneGain, defaultMicrophoneGain)
	}
	if got.Camera.DeviceID != defaultCameraID {
		t.Fatalf("camera id = %q, want %q", got.Camera.DeviceID, defaultCameraID)
	}
	if got.Camera.DeviceNativeID != "Integrated Camera" {
		t.Fatalf("camera native id = %q, want trimmed native id", got.Camera.DeviceNativeID)
	}
	if got.Camera.PIPPreset != "bottom-right" {
		t.Fatalf("pip preset = %q, want bottom-right", got.Camera.PIPPreset)
	}
	if got.Camera.PIP.Shape != pip.DefaultShape || !got.Camera.PIP.Mirror {
		t.Fatalf("pip config = %#v, want default shape and mirror", got.Camera.PIP)
	}
}

func TestNormalizeStartRequestPreservesCustomPIPConfig(t *testing.T) {
	got, err := NormalizeStartRequest(StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Camera: CameraRequest{
			Enabled:   true,
			PIPPreset: "bottom-right",
			PIP: pip.Config{
				Preset:      pip.PresetFree,
				Shape:       pip.ShapeSquare,
				Mirror:      false,
				Position:    pip.Position{X: 0.3, Y: 0.6},
				Scale:       0.3,
				EdgeFeather: 0.24,
			},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeStartRequest() error = %v", err)
	}
	if got.Camera.PIPPreset != "free" {
		t.Fatalf("pip preset = %q, want free from custom pip config", got.Camera.PIPPreset)
	}
	if got.Camera.PIP.Shape != pip.ShapeSquare || got.Camera.PIP.Mirror {
		t.Fatalf("pip config = %#v, want square and non-mirrored", got.Camera.PIP)
	}
	if got.Camera.PIP.Position.X != 0.3 || got.Camera.PIP.Position.Y != 0.6 || got.Camera.PIP.Scale != 0.3 || got.Camera.PIP.EdgeFeather != 0.24 {
		t.Fatalf("pip layout = %#v, want preserved custom layout", got.Camera.PIP)
	}
}

func TestNormalizeStartRequestKeepsRecordingProfile(t *testing.T) {
	got, err := NormalizeStartRequest(StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Recording: recordingprofile.Profile{
			Quality:          recordingprofile.QualityHigh,
			FPS:              60,
			CaptureCursor:    false,
			CountdownSeconds: 3,
		},
	})
	if err != nil {
		t.Fatalf("NormalizeStartRequest() error = %v", err)
	}
	if got.Recording.Quality != recordingprofile.QualityHigh || got.Recording.FPS != 60 || got.Recording.CaptureCursor || got.Recording.CountdownSeconds != 3 {
		t.Fatalf("recording profile was not preserved: %#v", got.Recording)
	}
}

func TestNormalizeStartRequestAcceptsVideoSourceModes(t *testing.T) {
	for _, sourceType := range []CaptureSourceType{SourceScreen, SourceAllScreens, SourceRegion, SourceWindow, SourceApplication} {
		got, err := NormalizeStartRequest(StartRequest{
			SourceID:   string(sourceType) + ":test",
			SourceType: sourceType,
			SourceGeometry: &SourceGeometry{
				X:        -1920,
				Y:        0,
				Width:    1280,
				Height:   720,
				NativeID: " display:test ",
			},
		})
		if err != nil {
			t.Fatalf("NormalizeStartRequest(%q) error = %v", sourceType, err)
		}
		if got.SourceGeometry == nil || got.SourceGeometry.NativeID != "display:test" {
			t.Fatalf("source geometry for %q = %#v, want trimmed native id", sourceType, got.SourceGeometry)
		}
	}
}

func TestNormalizeStartRequestClearsDisabledStreams(t *testing.T) {
	got, err := NormalizeStartRequest(StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Audio: AudioRequest{
			System:           false,
			SystemDeviceID:   "system-audio:default",
			Microphone:       false,
			MicrophoneID:     "microphone:default",
			NoiseSuppression: true,
			MicrophoneGain:   2,
		},
		Camera: CameraRequest{Enabled: false, DeviceID: "camera:default", DeviceNativeID: "Integrated Camera", PIPPreset: "bottom-left"},
	})
	if err != nil {
		t.Fatalf("NormalizeStartRequest() error = %v", err)
	}
	if got.Audio.SystemDeviceID != "" {
		t.Fatalf("disabled system audio kept device id %q", got.Audio.SystemDeviceID)
	}
	if got.Audio.MicrophoneID != "" || got.Audio.NoiseSuppression || got.Audio.MicrophoneGain != 0 {
		t.Fatalf("disabled microphone was not cleared: %#v", got.Audio)
	}
	if got.Camera.DeviceID != "" || got.Camera.DeviceNativeID != "" || got.Camera.PIPPreset != "off" || got.Camera.PIP.Preset != pip.PresetOff {
		t.Fatalf("disabled camera was not cleared: %#v", got.Camera)
	}
}

func TestNormalizeStartRequestRejectsInvalidSource(t *testing.T) {
	if _, err := NormalizeStartRequest(StartRequest{SourceType: SourceScreen}); err == nil {
		t.Fatal("NormalizeStartRequest() accepted missing source id")
	}
	if _, err := NormalizeStartRequest(StartRequest{SourceID: "screen:primary", SourceType: CaptureSourceType("display")}); err == nil {
		t.Fatal("NormalizeStartRequest() accepted invalid source type")
	}
}

func TestNormalizeStartRequestRejectsInvalidGain(t *testing.T) {
	if _, err := NormalizeStartRequest(StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Audio:      AudioRequest{Microphone: true, MicrophoneGain: maxMicrophoneGain + 1},
	}); err == nil {
		t.Fatal("NormalizeStartRequest() accepted gain above max")
	}
}

func TestNormalizeAudioOnlyRequestDefaultsDevices(t *testing.T) {
	got, err := NormalizeAudioOnlyRequest(AudioOnlyRequest{
		Audio: AudioRequest{
			System:           true,
			Microphone:       true,
			NoiseSuppression: true,
		},
	})
	if err != nil {
		t.Fatalf("NormalizeAudioOnlyRequest() error = %v", err)
	}
	if got.Recording != recordingprofile.Default() {
		t.Fatalf("recording profile = %#v, want default", got.Recording)
	}
	if got.Audio.SystemDeviceID != defaultSystemAudioID {
		t.Fatalf("system device = %q, want default", got.Audio.SystemDeviceID)
	}
	if got.Audio.MicrophoneID != defaultMicrophoneID {
		t.Fatalf("microphone device = %q, want default", got.Audio.MicrophoneID)
	}
	if got.Audio.MicrophoneGain != defaultMicrophoneGain {
		t.Fatalf("microphone gain = %v, want default", got.Audio.MicrophoneGain)
	}
}

func TestNormalizeAudioOnlyRequestRejectsNoStreams(t *testing.T) {
	if _, err := NormalizeAudioOnlyRequest(AudioOnlyRequest{}); err == nil {
		t.Fatal("NormalizeAudioOnlyRequest() accepted no audio streams")
	}
}
