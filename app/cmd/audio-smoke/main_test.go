package main

import (
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestAudioOnlyPackageRequestUsesWAVFallbackForSingleStream(t *testing.T) {
	request := audioOnlyPackageRequest(true, false, "microphone:default", "system-audio:default", true, 1.5)

	if request.AudioPath != recpackage.AudioOnlyWAVFile {
		t.Fatalf("audio path = %q, want %q", request.AudioPath, recpackage.AudioOnlyWAVFile)
	}
	if request.MicrophoneAudioPath != recpackage.AudioOnlyWAVFile || request.MicrophoneAudioStorage != recpackage.AudioStorageSidecar {
		t.Fatalf("microphone media = %q/%q, want audio.wav sidecar", request.MicrophoneAudioPath, request.MicrophoneAudioStorage)
	}
	if request.SystemAudioPath != "" || request.SystemAudioStorage != "" {
		t.Fatalf("disabled system audio media = %q/%q, want empty", request.SystemAudioPath, request.SystemAudioStorage)
	}
	if request.Audio.MicrophoneNoiseSuppression != recpackage.NoiseSuppressionOn || request.Audio.MicrophoneGain != 1.5 {
		t.Fatalf("audio request = %#v, want rnnoise gain", request.Audio)
	}
}

func TestAudioOnlyPackageRequestKeepsDualStreamFallbackSidecars(t *testing.T) {
	request := audioOnlyPackageRequest(true, true, "microphone:default", "system-audio:default", false, 1)

	if request.AudioPath != recpackage.MicrophoneAudioFile {
		t.Fatalf("dual-stream primary fallback = %q, want %q", request.AudioPath, recpackage.MicrophoneAudioFile)
	}
	if request.MicrophoneAudioPath != recpackage.MicrophoneAudioFile || request.MicrophoneAudioStorage != recpackage.AudioStorageSidecar {
		t.Fatalf("microphone media = %q/%q, want microphone sidecar", request.MicrophoneAudioPath, request.MicrophoneAudioStorage)
	}
	if request.SystemAudioPath != recpackage.SystemAudioFile || request.SystemAudioStorage != recpackage.AudioStorageSidecar {
		t.Fatalf("system media = %q/%q, want system sidecar", request.SystemAudioPath, request.SystemAudioStorage)
	}
	if request.Audio.MicrophoneNoiseSuppression != recpackage.NoiseSuppressionOff {
		t.Fatalf("noise suppression = %q, want off", request.Audio.MicrophoneNoiseSuppression)
	}
}

func TestAudioOnlySyncDiagnosticsMapsEnabledTracks(t *testing.T) {
	sync := audioOnlySyncDiagnostics(audio.Diagnostics{
		SystemAudio: audio.StreamDiagnostics{Enabled: true, SampleRate: 48000, DurationMs: 1000},
		Microphone:  audio.StreamDiagnostics{Enabled: true, SampleRate: 48000, DurationMs: 990},
	}, recpackage.Manifest{
		Media: recpackage.ManifestMedia{
			SystemAudioPath:     recpackage.SystemAudioFile,
			MicrophoneAudioPath: recpackage.MicrophoneAudioFile,
		},
	})

	if sync.AudioDiagnosticsPath != recpackage.AudioDiagnosticsFile {
		t.Fatalf("audio diagnostics path = %q, want package relative diagnostics", sync.AudioDiagnosticsPath)
	}
	if sync.SystemAudio.Path != recpackage.SystemAudioFile || sync.SystemAudio.DurationMs != 1000 {
		t.Fatalf("system sync = %#v", sync.SystemAudio)
	}
	if sync.Microphone.Path != recpackage.MicrophoneAudioFile || sync.Microphone.DurationMs != 990 {
		t.Fatalf("microphone sync = %#v", sync.Microphone)
	}
}
