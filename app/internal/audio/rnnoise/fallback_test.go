//go:build !cgo

package rnnoise

import "testing"

func TestFallbackReportsUnavailable(t *testing.T) {
	if Available() {
		t.Fatal("fallback build reported RNNoise as available")
	}
	if _, err := New(1); err == nil {
		t.Fatal("fallback New() returned nil error")
	}
	if RequiredSampleRate() != 48000 || FrameSize() != 480 {
		t.Fatalf("fallback requirements = %d/%d, want 48000/480", RequiredSampleRate(), FrameSize())
	}
}
