//go:build linux

package video

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestV4L2CameraInputArgsUseDevicePath(t *testing.T) {
	previewPath := filepath.Join("cache", "pip-camera-preview.jpg")
	args, err := v4l2CameraInputArgs(CameraCaptureConfig{
		DeviceNativeID:   "/dev/video0",
		PreviewImagePath: previewPath,
	})(CaptureConfig{Profile: recordingprofile.Profile{FPS: 30}})
	if err != nil {
		t.Fatalf("v4l2CameraInputArgs() error = %v", err)
	}
	if !slices.Contains(args.Args, "/dev/video0") {
		t.Fatalf("args = %#v, want v4l2 device path", args)
	}
	if args.PreviewImagePath != previewPath || args.PreviewImageFPS != 8 || args.PreviewImageWidth != 360 {
		t.Fatalf("preview image spec = path:%q fps:%d width:%d", args.PreviewImagePath, args.PreviewImageFPS, args.PreviewImageWidth)
	}
}
