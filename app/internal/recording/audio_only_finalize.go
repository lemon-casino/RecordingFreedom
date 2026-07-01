package recording

import (
	"errors"
	"fmt"
	"os"

	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func defaultAudioOnlyPostStopProcessor(runtime *AudioOnlyRuntime) error {
	if runtime == nil || runtime.packages == nil {
		return nil
	}
	manifest := runtime.Plan.Package.Manifest
	inputs := make([]video.AudioMuxInput, 0, 2)
	muxSystem := false
	muxMicrophone := false
	if manifest.Audio.System {
		if !readableAudioSidecar(runtime.Plan.SystemAudioPath) {
			return fmt.Errorf("audio-only system sidecar %q is missing or empty", runtime.Plan.SystemAudioPath)
		}
		inputs = append(inputs, video.AudioMuxInput{Path: runtime.Plan.SystemAudioPath, Label: "system"})
		muxSystem = true
	}
	if manifest.Audio.Microphone {
		if !readableAudioSidecar(runtime.Plan.MicrophoneAudioPath) {
			return fmt.Errorf("audio-only microphone sidecar %q is missing or empty", runtime.Plan.MicrophoneAudioPath)
		}
		inputs = append(inputs, video.AudioMuxInput{Path: runtime.Plan.MicrophoneAudioPath, Label: "microphone"})
		muxMicrophone = true
	}
	if len(inputs) == 0 {
		return errors.New("audio-only finalization requires at least one enabled audio stream")
	}
	if _, err := video.MuxAudioOnlyToM4A(video.AudioOnlyMuxConfig{
		OutputPath: runtime.Plan.AudioOnlyPath,
		Inputs:     inputs,
	}); err != nil {
		return err
	}
	manifest, err := runtime.packages.PatchAudioOnlyMuxed(runtime.Plan.Package.ManifestPath, muxSystem, muxMicrophone)
	if err != nil {
		return err
	}
	runtime.Plan.Package.Manifest = manifest
	if muxSystem {
		runtime.Plan.SystemAudioPath = runtime.Plan.AudioOnlyPath
	}
	if muxMicrophone {
		runtime.Plan.MicrophoneAudioPath = runtime.Plan.AudioOnlyPath
	}
	return nil
}

func readableAudioSidecar(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 45
}
