package recording

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestCreateAudioOnlyWritePlanUsesWAVFallback(t *testing.T) {
	packages := recpackage.NewService()
	createdAt := time.Date(2026, 7, 1, 10, 0, 0, 321000000, time.UTC)
	plan, normalized, err := CreateAudioOnlyWritePlan(packages, BackendAudioOnlyNative, t.TempDir(), createdAt, AudioOnlyRequest{
		Audio: AudioRequest{
			Microphone:       true,
			NoiseSuppression: true,
		},
	})
	if err != nil {
		t.Fatalf("CreateAudioOnlyWritePlan() error = %v", err)
	}
	if normalized.Audio.MicrophoneID != defaultMicrophoneID || normalized.Audio.MicrophoneGain != defaultMicrophoneGain {
		t.Fatalf("normalized audio = %#v, want default microphone", normalized.Audio)
	}
	if plan.ScreenVideoPath != "" || plan.VideoDiagnosticsPath != "" {
		t.Fatalf("audio-only video paths = screen:%q diagnostics:%q, want empty", plan.ScreenVideoPath, plan.VideoDiagnosticsPath)
	}
	if plan.AudioOnlyPath != filepath.Join(plan.Package.Dir, recpackage.AudioOnlyWAVFile) {
		t.Fatalf("audio-only path = %q, want package audio.wav", plan.AudioOnlyPath)
	}
	if plan.MicrophoneAudioPath != plan.AudioOnlyPath || plan.SystemAudioPath != "" {
		t.Fatalf("stream paths = mic:%q system:%q, want mic audio.wav only", plan.MicrophoneAudioPath, plan.SystemAudioPath)
	}

	manifest, err := packages.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.RecordingMode != recpackage.RecordingModeAudio || manifest.Media.ScreenVideoPath != "" {
		t.Fatalf("manifest mode/media = %q/%#v, want audio-only without screen", manifest.RecordingMode, manifest.Media)
	}
	if manifest.Media.AudioPath != recpackage.AudioOnlyWAVFile ||
		manifest.Media.MicrophoneAudioPath != recpackage.AudioOnlyWAVFile ||
		manifest.Media.MicrophoneAudioStorage != recpackage.AudioStorageSidecar {
		t.Fatalf("manifest media = %#v, want audio.wav sidecar fallback", manifest.Media)
	}
}

func TestCreateAudioOnlyCaptureConfigMapsPlan(t *testing.T) {
	packageDir := t.TempDir()
	plan := recpackage.RecordingWritePlan{
		SystemAudioPath:      filepath.Join(packageDir, recpackage.SystemAudioFile),
		MicrophoneAudioPath:  filepath.Join(packageDir, recpackage.MicrophoneAudioFile),
		AudioDiagnosticsPath: filepath.Join(packageDir, recpackage.AudioDiagnosticsFile),
	}

	config, err := CreateAudioOnlyCaptureConfig(BackendAudioOnlyNative, AudioOnlyRequest{
		Audio: AudioRequest{
			System:           true,
			SystemDeviceID:   "system-audio:default",
			Microphone:       true,
			MicrophoneID:     "microphone:default",
			NoiseSuppression: true,
			MicrophoneGain:   1.25,
		},
	}, plan)
	if err != nil {
		t.Fatalf("CreateAudioOnlyCaptureConfig() error = %v", err)
	}
	if config.Backend != BackendAudioOnlyNative {
		t.Fatalf("backend = %q, want audio-only backend", config.Backend)
	}
	if !config.SystemAudio.Enabled || !config.Microphone.Enabled || !config.NoiseSuppression {
		t.Fatalf("audio config = %#v, want both streams with RNNoise", config)
	}
	if config.TargetSampleRate != audio.RNNoiseSampleRate || config.TargetChannels != 2 {
		t.Fatalf("target format = %d/%d", config.TargetSampleRate, config.TargetChannels)
	}
	if config.SystemAudioOutputPath != plan.SystemAudioPath || config.MicrophoneAudioPath != plan.MicrophoneAudioPath || config.DiagnosticsPath != plan.AudioDiagnosticsPath {
		t.Fatalf("paths = system:%q mic:%q diagnostics:%q", config.SystemAudioOutputPath, config.MicrophoneAudioPath, config.DiagnosticsPath)
	}
}
