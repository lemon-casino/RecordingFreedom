package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
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
	recorder := recording.NewService(appdata.NewService(root))
	session, err := recorder.StartAudioOnlyRecording(recording.AudioOnlyRequest{
		Audio: recording.AudioRequest{
			System:           systemAudio,
			SystemDeviceID:   systemDevice,
			Microphone:       microphone,
			MicrophoneID:     microphoneDevice,
			NoiseSuppression: noiseSuppression,
			MicrophoneGain:   gain,
		},
	})
	if err != nil {
		fail(err.Error())
	}

	time.Sleep(duration)
	stopped, err := recorder.Stop()
	if err != nil {
		fail(err.Error())
	}
	manifest, err := recpackage.NewService().ReadManifest(stopped.Manifest)
	if err != nil {
		fail(err.Error())
	}
	diagnosticsPath := absPackagePath(stopped.PackageDir, recpackage.AudioDiagnosticsFile)
	diagnostics, err := readAudioDiagnostics(diagnosticsPath)
	if err != nil {
		fail(err.Error())
	}

	result := map[string]any{
		"ok":                  true,
		"dataRoot":            root,
		"videoDir":            videoDir,
		"sessionId":           session.ID,
		"packageDir":          stopped.PackageDir,
		"manifestPath":        stopped.Manifest,
		"audioPath":           absPackagePath(stopped.PackageDir, manifest.Media.AudioPath),
		"microphoneAudioPath": absPackagePath(stopped.PackageDir, manifest.Media.MicrophoneAudioPath),
		"systemAudioPath":     absPackagePath(stopped.PackageDir, manifest.Media.SystemAudioPath),
		"diagnosticsPath":     diagnosticsPath,
		"queue":               diagnostics.Queue,
		"microphone":          diagnostics.Microphone,
		"systemAudio":         diagnostics.SystemAudio,
		"recordingMode":       stopped.RecordingMode,
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

func absPackagePath(packageDir string, relativePath string) string {
	if relativePath == "" {
		return ""
	}
	return filepath.Join(packageDir, relativePath)
}

func readAudioDiagnostics(path string) (audio.Diagnostics, error) {
	if path == "" {
		return audio.Diagnostics{}, fmt.Errorf("audio diagnostics path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return audio.Diagnostics{}, err
	}
	var diagnostics audio.Diagnostics
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		return audio.Diagnostics{}, err
	}
	return diagnostics, nil
}

func fail(message string) {
	fmt.Fprintf(os.Stderr, "audio-smoke failed: %s\n", message)
	os.Exit(1)
}
