//go:build darwin

package video

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

func TestAVFoundationCameraInputArgsUseNativeIndex(t *testing.T) {
	previewPath := filepath.Join("cache", "pip-camera-preview.jpg")
	args, err := avFoundationCameraInputArgs(CameraCaptureConfig{
		DeviceNativeID:   "0",
		PreviewImagePath: previewPath,
	})(CaptureConfig{Profile: recordingprofile.Profile{FPS: 30}})
	if err != nil {
		t.Fatalf("avFoundationCameraInputArgs() error = %v", err)
	}
	if !slices.Contains(args.Args, "0:none") {
		t.Fatalf("args = %#v, want AVFoundation video index with no audio", args)
	}
	if args.PreviewImagePath != previewPath || args.PreviewImageFPS != 8 || args.PreviewImageWidth != 360 {
		t.Fatalf("preview image spec = path:%q fps:%d width:%d", args.PreviewImagePath, args.PreviewImageFPS, args.PreviewImageWidth)
	}
}

func TestAVFoundationCameraInputKeepsExplicitAudioPart(t *testing.T) {
	if got := avFoundationCameraInput("FaceTime HD Camera:none"); got != "FaceTime HD Camera:none" {
		t.Fatalf("avFoundationCameraInput() = %q", got)
	}
}
