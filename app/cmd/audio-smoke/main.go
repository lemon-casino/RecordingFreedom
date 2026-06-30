package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func main() {
	var (
		root             string
		duration         time.Duration
		microphone       bool
		systemAudio      bool
		microphoneDevice string
		systemDevice     string
		noiseSuppression bool
		gain             float64
		keep             bool
	)
	flag.StringVar(&root, "root", "", "data root; defaults to a temporary root")
	flag.DurationVar(&duration, "duration", 3*time.Second, "capture duration")
	flag.BoolVar(&microphone, "microphone", true, "capture microphone")
	flag.BoolVar(&systemAudio, "system", false, "capture system audio loopback")
	flag.StringVar(&microphoneDevice, "microphone-device", "microphone:default", "microphone device id")
	flag.StringVar(&systemDevice, "system-device", "system-audio:default", "system audio device id")
	flag.BoolVar(&noiseSuppression, "rnnoise", false, "enable RNNoise microphone suppression; requires cgo build")
	flag.Float64Var(&gain, "gain", 1, "microphone gain")
	flag.BoolVar(&keep, "keep", false, "keep temporary root when -root is not provided")
	flag.Parse()
	explicitRoot := root != ""

	if !microphone && !systemAudio {
		fail("at least one of -microphone or -system must be enabled")
	}
	if duration <= 0 {
		fail("-duration must be positive")
	}
	if root == "" {
		tempRoot, err := os.MkdirTemp("", "recordingfreedom-audio-smoke-*")
		if err != nil {
			fail(err.Error())
		}
		root = tempRoot
		if !keep {
			defer os.RemoveAll(root)
		}
	}

	videoDir, err := appdata.NewService(root).VideoDir()
	if err != nil {
		fail(err.Error())
	}
	packages := recpackage.NewService()
	plan, err := packages.CreateAudioOnly(videoDir, audioOnlyPackageRequest(microphone, systemAudio, microphoneDevice, systemDevice, noiseSuppression, gain))
	if err != nil {
		fail(err.Error())
	}
	failPackage := func(message string) {
		_ = packages.PatchStatus(plan.Package.ManifestPath, recpackage.StatusFailed, nil)
		fail(message)
	}

	config := audio.CaptureConfig{
		Backend:               "audio-smoke",
		TargetSampleRate:      audio.RNNoiseSampleRate,
		TargetChannels:        2,
		MicrophoneGain:        gain,
		NoiseSuppression:      noiseSuppression,
		SystemAudioOutputPath: plan.SystemAudioPath,
		MicrophoneAudioPath:   plan.MicrophoneAudioPath,
		DiagnosticsPath:       plan.AudioDiagnosticsPath,
		SystemAudio:           audio.StreamConfig{Enabled: systemAudio, DeviceID: systemDevice},
		Microphone:            audio.StreamConfig{Enabled: microphone, DeviceID: microphoneDevice},
	}

	var suppressor audio.NoiseSuppressor
	if noiseSuppression {
		nativeSuppressor, err := rnnoise.New(gain)
		if err != nil {
			failPackage(err.Error())
		}
		defer nativeSuppressor.Close()
		suppressor = nativeSuppressor
	}
	session, err := audio.NewNativeCaptureSession(config, suppressor)
	if err != nil {
		failPackage(err.Error())
	}
	if err := session.Start(context.Background()); err != nil {
		failPackage(err.Error())
	}
	time.Sleep(duration)
	if err := session.Stop(); err != nil {
		failPackage(err.Error())
	}
	manifest, err := packages.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		failPackage(err.Error())
	}
	if err := packages.PatchSyncDiagnostics(plan.Package.ManifestPath, audioOnlySyncDiagnostics(session.Diagnostics(), manifest)); err != nil {
		failPackage(err.Error())
	}
	if err := packages.ValidateReady(plan.Package.ManifestPath); err != nil {
		failPackage(err.Error())
	}
	completedAt := time.Now()
	if err := packages.PatchStatus(plan.Package.ManifestPath, recpackage.StatusReady, &completedAt); err != nil {
		failPackage(err.Error())
	}

	result := map[string]any{
		"ok":                  true,
		"dataRoot":            root,
		"videoDir":            videoDir,
		"packageDir":          plan.Package.Dir,
		"manifestPath":        plan.Package.ManifestPath,
		"audioPath":           plan.AudioOnlyPath,
		"microphoneAudioPath": config.MicrophoneAudioPath,
		"systemAudioPath":     config.SystemAudioOutputPath,
		"diagnosticsPath":     config.DiagnosticsPath,
		"rnnoiseAvailable":    rnnoise.Available(),
		"rnnoiseEnabled":      noiseSuppression,
		"keptDataRoot":        keep || explicitRoot,
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	fmt.Println(string(data))
}

func audioOnlyPackageRequest(microphone bool, systemAudio bool, microphoneDevice string, systemDevice string, noiseSuppression bool, gain float64) recpackage.CreateAudioOnlyRequest {
	request := recpackage.CreateAudioOnlyRequest{
		CreatedAt: time.Now(),
		Status:    recpackage.StatusRecording,
		Backend:   "audio-smoke",
		Audio: recpackage.ManifestAudio{
			System:                     systemAudio,
			SystemDeviceID:             systemDevice,
			Microphone:                 microphone,
			MicrophoneDeviceID:         microphoneDevice,
			MicrophoneNoiseSuppression: noiseSuppressionLabel(noiseSuppression),
			MicrophoneGain:             gain,
		},
	}
	if microphone && systemAudio {
		request.AudioPath = recpackage.MicrophoneAudioFile
		request.MicrophoneAudioPath = recpackage.MicrophoneAudioFile
		request.MicrophoneAudioStorage = recpackage.AudioStorageSidecar
		request.SystemAudioPath = recpackage.SystemAudioFile
		request.SystemAudioStorage = recpackage.AudioStorageSidecar
		return request
	}
	request.AudioPath = recpackage.AudioOnlyWAVFile
	if microphone {
		request.MicrophoneAudioPath = recpackage.AudioOnlyWAVFile
		request.MicrophoneAudioStorage = recpackage.AudioStorageSidecar
	}
	if systemAudio {
		request.SystemAudioPath = recpackage.AudioOnlyWAVFile
		request.SystemAudioStorage = recpackage.AudioStorageSidecar
	}
	return request
}

func audioOnlySyncDiagnostics(diagnostics audio.Diagnostics, manifest recpackage.Manifest) recpackage.ManifestSyncDiagnostics {
	return recpackage.ManifestSyncDiagnostics{
		TimelineBase:         recpackage.TimelineBaseMedia,
		AudioDiagnosticsPath: recpackage.AudioDiagnosticsFile,
		SystemAudio:          audioTrackDiagnostics(diagnostics.SystemAudio, manifest.Media.SystemAudioPath),
		Microphone:           audioTrackDiagnostics(diagnostics.Microphone, manifest.Media.MicrophoneAudioPath),
	}
}

func audioTrackDiagnostics(diagnostics audio.StreamDiagnostics, path string) recpackage.ManifestTrackDiagnostics {
	if !diagnostics.Enabled {
		return recpackage.ManifestTrackDiagnostics{}
	}
	return recpackage.ManifestTrackDiagnostics{
		Enabled:        true,
		Path:           path,
		Clock:          recpackage.TimelineBaseMedia,
		StartOffsetMs:  diagnostics.StartOffsetMs,
		EndOffsetMs:    diagnostics.EndOffsetMs,
		DurationMs:     diagnostics.DurationMs,
		DroppedSamples: diagnostics.DroppedSamples,
		AppendFailures: diagnostics.AppendFailures,
		SampleRate:     diagnostics.SampleRate,
		Message:        diagnostics.Message,
	}
}

func noiseSuppressionLabel(enabled bool) string {
	if enabled {
		return recpackage.NoiseSuppressionOn
	}
	return recpackage.NoiseSuppressionOff
}

func fail(message string) {
	fmt.Fprintf(os.Stderr, "audio-smoke failed: %s\n", message)
	os.Exit(1)
}
