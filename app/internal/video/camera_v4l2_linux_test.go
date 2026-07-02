//go:build linux

package video

import (
	"slices"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestV4L2CameraInputArgsUseDevicePath(t *testing.T) {
	args, err := v4l2CameraInputArgs(CameraCaptureConfig{
		DeviceNativeID: "/dev/video0",
	})(CaptureConfig{Profile: recordingprofile.Profile{FPS: 30}})
	if err != nil {
		t.Fatalf("v4l2CameraInputArgs() error = %v", err)
	}
	if !slices.Contains(args.Args, "/dev/video0") {
		t.Fatalf("args = %#v, want v4l2 device path", args)
	}
}
