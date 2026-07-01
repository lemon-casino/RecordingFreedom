//go:build windows

package audio

import "testing"

func TestSyntheticSilenceFramesUsesFiftyMilliseconds(t *testing.T) {
	if got := syntheticSilenceFrames(48000); got != 2400 {
		t.Fatalf("syntheticSilenceFrames(48000) = %d, want 2400", got)
	}
	if got := syntheticSilenceFrames(0); got != 480 {
		t.Fatalf("syntheticSilenceFrames(0) = %d, want conservative fallback", got)
	}
}
