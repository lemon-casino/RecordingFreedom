package audio

import "testing"

func TestMonoResamplerCopiesMatchingRate(t *testing.T) {
	resampler, err := NewMonoResampler(48000, 48000)
	if err != nil {
		t.Fatalf("NewMonoResampler() error = %v", err)
	}
	input := []float32{0, 0.5, 1}
	output := resampler.Convert(input)
	if len(output) != len(input) {
		t.Fatalf("output samples = %d, want %d", len(output), len(input))
	}
	output[0] = 9
	if input[0] == 9 {
		t.Fatal("resampler returned input slice instead of a copy")
	}
}

func TestMonoResamplerUpsamplesAcrossChunks(t *testing.T) {
	resampler, err := NewMonoResampler(24000, 48000)
	if err != nil {
		t.Fatalf("NewMonoResampler() error = %v", err)
	}

	first := resampler.Convert([]float32{0, 1})
	second := resampler.Convert([]float32{2, 3})
	combined := append(first, second...)
	if len(combined) < 5 {
		t.Fatalf("upsampled samples = %d, want at least 5", len(combined))
	}
	if combined[0] != 0 || combined[1] != 0.5 || combined[2] != 1 {
		t.Fatalf("upsampled prefix = %#v, want linear interpolation", combined[:3])
	}
}
