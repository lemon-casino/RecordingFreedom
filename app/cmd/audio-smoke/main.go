package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	packageDir := filepath.Join(videoDir, "audio-smoke-"+recpackage.SessionID(time.Now())+recpackage.PackageDirSuffix)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		fail(err.Error())
	}

	config := audio.CaptureConfig{
		Backend:               "audio-smoke",
		TargetSampleRate:      audio.RNNoiseSampleRate,
		TargetChannels:        2,
		MicrophoneGain:        gain,
		NoiseSuppression:      noiseSuppression,
		SystemAudioOutputPath: filepath.Join(packageDir, recpackage.SystemAudioFile),
		MicrophoneAudioPath:   filepath.Join(packageDir, recpackage.MicrophoneAudioFile),
		DiagnosticsPath:       filepath.Join(packageDir, recpackage.AudioDiagnosticsFile),
		SystemAudio:           audio.StreamConfig{Enabled: systemAudio, DeviceID: systemDevice},
		Microphone:            audio.StreamConfig{Enabled: microphone, DeviceID: microphoneDevice},
	}

	var suppressor audio.NoiseSuppressor
	if noiseSuppression {
		nativeSuppressor, err := rnnoise.New(gain)
		if err != nil {
			fail(err.Error())
		}
		defer nativeSuppressor.Close()
		suppressor = nativeSuppressor
	}
	session, err := audio.NewNativeCaptureSession(config, suppressor)
	if err != nil {
		fail(err.Error())
	}
	if err := session.Start(context.Background()); err != nil {
		fail(err.Error())
	}
	time.Sleep(duration)
	if err := session.Stop(); err != nil {
		fail(err.Error())
	}

	result := map[string]any{
		"ok":                  true,
		"dataRoot":            root,
		"videoDir":            videoDir,
		"packageDir":          packageDir,
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

func fail(message string) {
	fmt.Fprintf(os.Stderr, "audio-smoke failed: %s\n", message)
	os.Exit(1)
}
