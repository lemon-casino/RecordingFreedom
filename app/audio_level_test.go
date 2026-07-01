package main

import "testing"

func TestAudioLevelFromSamplesUsesRealSamples(t *testing.T) {
	level := audioLevelFromSamples([]float32{0, 0.25, -0.5, 1})
	if level.RMS <= 0 || level.RMS > 1 {
		t.Fatalf("RMS = %v, want normalized non-zero value", level.RMS)
	}
	if level.Peak != 1 {
		t.Fatalf("Peak = %v, want 1", level.Peak)
	}
	if level.Level <= level.RMS {
		t.Fatalf("display level = %v, want perceptual level above RMS %v", level.Level, level.RMS)
	}
	if !level.Active {
		t.Fatalf("level should be active")
	}
}

func TestAudioLevelFromSamplesHandlesSilenceAndInvalidNumbers(t *testing.T) {
	silence := audioLevelFromSamples([]float32{0, 0, 0})
	if silence.RMS != 0 || silence.Peak != 0 || silence.Level != 0 {
		t.Fatalf("silence level = %#v, want all zeros", silence)
	}
	if got := clampUnit(2); got != 1 {
		t.Fatalf("clampUnit(2) = %v, want 1", got)
	}
	if got := clampUnit(-1); got != 0 {
		t.Fatalf("clampUnit(-1) = %v, want 0", got)
	}
}
