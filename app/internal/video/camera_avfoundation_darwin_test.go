//go:build darwin

package video

import (
	"slices"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestAVFoundationCameraInputArgsUseNativeIndex(t *testing.T) {
	args, err := avFoundationCameraInputArgs(CameraCaptureConfig{
		DeviceNativeID: "0",
	})(CaptureConfig{Profile: recordingprofile.Profile{FPS: 30}})
	if err != nil {
		t.Fatalf("avFoundationCameraInputArgs() error = %v", err)
	}
	if !slices.Contains(args, "0:none") {
		t.Fatalf("args = %#v, want AVFoundation video index with no audio", args)
	}
}

func TestAVFoundationCameraInputKeepsExplicitAudioPart(t *testing.T) {
	if got := avFoundationCameraInput("FaceTime HD Camera:none"); got != "FaceTime HD Camera:none" {
		t.Fatalf("avFoundationCameraInput() = %q", got)
	}
}
