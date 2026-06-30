package recording

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestCreateAudioCaptureConfigKeepsEnabledAudioContract(t *testing.T) {
	packages := recpackage.NewService()
	plan, err := CreateNativeWritePlan(packages, BackendScreenCaptureKit, BackendStartRequest{
		VideoDir:  t.TempDir(),
		CreatedAt: time.Now(),
		StartRequest: StartRequest{
			SourceID:   "cgdisplay:1",
			SourceType: SourceScreen,
			Audio: AudioRequest{
				System:           true,
				SystemDeviceID:   "system-audio:display",
				Microphone:       true,
				MicrophoneID:     "microphone:studio",
				NoiseSuppression: true,
				MicrophoneGain:   1.25,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateNativeWritePlan() error = %v", err)
	}

	config, err := CreateAudioCaptureConfig(BackendScreenCaptureKit, audioStartRequestForTest(plan.Package.Manifest.Audio), plan)
	if err != nil {
		t.Fatalf("CreateAudioCaptureConfig() error = %v", err)
	}
	if !config.SystemAudio.Enabled || config.SystemAudio.DeviceID != "system-audio:display" {
		t.Fatalf("system audio config = %#v", config.SystemAudio)
	}
	if !config.Microphone.Enabled || config.Microphone.DeviceID != "microphone:studio" {
		t.Fatalf("microphone config = %#v", config.Microphone)
	}
	if !config.NoiseSuppression {
		t.Fatal("noise suppression disabled, want enabled")
	}
	if config.TargetSampleRate != audio.RNNoiseSampleRate || config.TargetChannels != 2 {
		t.Fatalf("target format = %d/%d", config.TargetSampleRate, config.TargetChannels)
	}
	if config.DiagnosticsPath != filepath.Join(plan.Package.Dir, recpackage.AudioDiagnosticsFile) {
		t.Fatalf("diagnostics path = %q, want package audio diagnostics path", config.DiagnosticsPath)
	}
	if config.SystemAudioOutputPath != plan.SystemAudioPath || config.MicrophoneAudioPath != plan.MicrophoneAudioPath {
		t.Fatalf("audio output paths = system:%q mic:%q want system:%q mic:%q", config.SystemAudioOutputPath, config.MicrophoneAudioPath, plan.SystemAudioPath, plan.MicrophoneAudioPath)
	}
	if !config.SystemAudioIsNeverDenoised {
		t.Fatal("system audio denoise bypass policy was not carried into config")
	}
}

func TestCreateAudioCaptureConfigClearsDisabledAudio(t *testing.T) {
	packageDir := t.TempDir()
	plan := recpackage.RecordingWritePlan{
		Package:              recpackage.Package{Dir: packageDir},
		ScreenVideoPath:      filepath.Join(packageDir, recpackage.ScreenVideoFile),
		SystemAudioPath:      filepath.Join(packageDir, recpackage.SystemAudioFile),
		MicrophoneAudioPath:  filepath.Join(packageDir, recpackage.MicrophoneAudioFile),
		AudioDiagnosticsPath: filepath.Join(packageDir, recpackage.AudioDiagnosticsFile),
	}

	config, err := CreateAudioCaptureConfig(BackendWindowsGraphicsCapture, StartRequest{
		SourceID:   "screen:primary",
		SourceType: SourceScreen,
		Audio: AudioRequest{
			System:           false,
			SystemDeviceID:   "system-audio:stale",
			Microphone:       false,
			MicrophoneID:     "microphone:stale",
			NoiseSuppression: true,
		},
	}, plan)
	if err != nil {
		t.Fatalf("CreateAudioCaptureConfig() error = %v", err)
	}
	if config.SystemAudio.Enabled || config.SystemAudio.DeviceID != "" || config.SystemAudioOutputPath != "" {
		t.Fatalf("disabled system audio leaked into config: %#v", config)
	}
	if config.Microphone.Enabled || config.Microphone.DeviceID != "" || config.MicrophoneAudioPath != "" {
		t.Fatalf("disabled microphone leaked into config: %#v", config)
	}
	if config.NoiseSuppression {
		t.Fatal("RNNoise stayed enabled after microphone was disabled")
	}
}

func audioStartRequestForTest(a recpackage.ManifestAudio) StartRequest {
	return StartRequest{
		SourceID:   "cgdisplay:1",
		SourceType: SourceScreen,
		Audio: AudioRequest{
			System:           a.System,
			SystemDeviceID:   a.SystemDeviceID,
			Microphone:       a.Microphone,
			MicrophoneID:     a.MicrophoneDeviceID,
			NoiseSuppression: a.MicrophoneNoiseSuppression == recpackage.NoiseSuppressionOn,
			MicrophoneGain:   a.MicrophoneGain,
		},
	}
}
