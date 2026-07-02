//go:build windows

package video

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestDirectShowCameraInputArgsUseNativeDeviceName(t *testing.T) {
	args, err := directShowCameraInputArgs(CameraCaptureConfig{
		DeviceID:         "camera:dshow:integrated-camera",
		DeviceNativeID:   "Integrated Camera",
		PreviewImagePath: filepath.Join("cache", "pip-camera-preview.jpg"),
	})(CaptureConfig{})
	if err != nil {
		t.Fatalf("directShowCameraInputArgs() error = %v", err)
	}
	if !slices.Contains(args.Args, "-f") || !slices.Contains(args.Args, "dshow") {
		t.Fatalf("args = %#v, want DirectShow input", args)
	}
	if !slices.Contains(args.Args, "video=Integrated Camera") {
		t.Fatalf("args = %#v, want native DirectShow camera name", args)
	}
	if args.PreviewImagePath != filepath.Join("cache", "pip-camera-preview.jpg") || args.PreviewImageFPS != 8 || args.PreviewImageWidth != 360 {
		t.Fatalf("preview image spec = path:%q fps:%d width:%d", args.PreviewImagePath, args.PreviewImageFPS, args.PreviewImageWidth)
	}
}

func TestNewPlatformCameraSessionRequiresNativeDeviceName(t *testing.T) {
	withFakeFFmpeg(t)
	_, err := NewPlatformCameraSession(CameraCaptureConfig{
		DeviceID:   "camera:dshow:integrated-camera",
		OutputPath: filepath.Join(t.TempDir(), "webcam.mp4"),
	})
	if err == nil {
		t.Fatal("NewPlatformCameraSession() error = nil, want native device id requirement")
	}
}

func TestNewPlatformCameraSessionConstructsDirectShowWriter(t *testing.T) {
	withFakeFFmpeg(t)
	session, err := NewPlatformCameraSession(CameraCaptureConfig{
		Backend:        "ffmpeg-desktop-capture",
		DeviceID:       "camera:dshow:integrated-camera",
		DeviceNativeID: "Integrated Camera",
		OutputPath:     filepath.Join(t.TempDir(), "webcam.mp4"),
	})
	if err != nil {
		t.Fatalf("NewPlatformCameraSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("NewPlatformCameraSession() returned nil session")
	}
}
