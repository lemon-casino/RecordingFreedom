//go:build windows

package recording

import (
	"log"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

func init() {
	if err := RegisterNativeBackend(BackendFFmpegDesktopCapture, newWindowsDesktopCaptureBackend(BackendFFmpegDesktopCapture)); err != nil {
		log.Printf("register FFmpeg desktop capture backend: %v", err)
	}
	if err := RegisterNativeBackend(BackendWindowsGraphicsCapture, newWindowsDesktopCaptureBackend(BackendWindowsGraphicsCapture)); err != nil {
		log.Printf("register Windows capture compatibility backend: %v", err)
	}
}

func newWindowsDesktopCaptureBackend(id string) BackendFactory {
	return func(packages *recpackage.Service) Backend {
		return NewNativeRuntimeBackend(id, packages, NativeBackendRuntimeOptions{
			VideoSessionFactory: func(config video.CaptureConfig) (NativeVideoSession, error) {
				return video.NewPlatformSession(config)
			},
			CameraSessionFactory: func(config video.CameraCaptureConfig) (NativeCameraSession, error) {
				return video.NewPlatformCameraSession(config)
			},
			PostStopProcessor: windowsMuxAudioSidecarsIntoScreen,
		})
	}
}

func windowsMuxAudioSidecarsIntoScreen(runtime *NativeBackendRuntime) error {
	if runtime == nil || runtime.packages == nil {
		return nil
	}
	manifest := runtime.Plan.Package.Manifest
	inputs := make([]video.AudioMuxInput, 0, 2)
	muxSystem := false
	muxMicrophone := false
	if manifest.Audio.System && manifest.Media.SystemAudioStorage == recpackage.AudioStorageSidecar && readableAudioSidecar(runtime.Plan.SystemAudioPath) {
		inputs = append(inputs, video.AudioMuxInput{Path: runtime.Plan.SystemAudioPath, Label: "system"})
		muxSystem = true
	}
	if manifest.Audio.Microphone && manifest.Media.MicrophoneAudioStorage == recpackage.AudioStorageSidecar && readableAudioSidecar(runtime.Plan.MicrophoneAudioPath) {
		inputs = append(inputs, video.AudioMuxInput{Path: runtime.Plan.MicrophoneAudioPath, Label: "microphone"})
		muxMicrophone = true
	}
	if len(inputs) == 0 {
		return nil
	}
	if _, err := video.MuxAudioIntoMP4(video.AudioMuxConfig{
		VideoPath: runtime.Plan.ScreenVideoPath,
		Inputs:    inputs,
	}); err != nil {
		return err
	}
	manifest, err := runtime.packages.PatchScreenAudioMuxed(runtime.Plan.Package.ManifestPath, muxSystem, muxMicrophone)
	if err != nil {
		return err
	}
	runtime.Plan.Package.Manifest = manifest
	if muxSystem {
		runtime.Plan.SystemAudioPath = runtime.Plan.ScreenVideoPath
	}
	if muxMicrophone {
		runtime.Plan.MicrophoneAudioPath = runtime.Plan.ScreenVideoPath
	}
	return nil
}
