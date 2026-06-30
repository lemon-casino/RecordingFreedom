package recordingprofile

import "testing"

func TestNormalizeDefaultProfile(t *testing.T) {
	got := Normalize(Profile{})
	if got != Default() {
		t.Fatalf("Normalize(zero) = %#v, want %#v", got, Default())
	}
}

func TestNormalizeInvalidProfile(t *testing.T) {
	got := Normalize(Profile{
		Quality:          "cinema",
		FPS:              120,
		CaptureCursor:    false,
		CountdownSeconds: -3,
	})
	if got.Quality != DefaultQuality {
		t.Fatalf("quality = %q, want %q", got.Quality, DefaultQuality)
	}
	if got.FPS != DefaultFPS {
		t.Fatalf("fps = %d, want %d", got.FPS, DefaultFPS)
	}
	if got.CountdownSeconds != DefaultCountdownSeconds {
		t.Fatalf("countdown = %d, want %d", got.CountdownSeconds, DefaultCountdownSeconds)
	}
	if got.CaptureCursor {
		t.Fatal("explicit false captureCursor should be preserved when profile is non-zero")
	}
}

func TestNormalizeClampsCountdown(t *testing.T) {
	got := Normalize(Profile{
		Quality:          QualityHigh,
		FPS:              60,
		CaptureCursor:    true,
		CountdownSeconds: 99,
	})
	if got.CountdownSeconds != MaxCountdownSeconds {
		t.Fatalf("countdown = %d, want %d", got.CountdownSeconds, MaxCountdownSeconds)
	}
}
