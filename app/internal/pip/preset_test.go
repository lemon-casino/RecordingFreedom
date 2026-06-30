package pip

import "testing"

func TestNormalizeDefaultsInvalidPreset(t *testing.T) {
	if got := Normalize("diagonal"); got != DefaultPreset {
		t.Fatalf("Normalize(invalid) = %q, want %q", got, DefaultPreset)
	}
	if got := Normalize(string(PresetBottomLeft)); got != PresetBottomLeft {
		t.Fatalf("Normalize(bottom-left) = %q", got)
	}
}

func TestLayoutBottomRight(t *testing.T) {
	rect, err := Layout(PresetBottomRight, Size{Width: 1920, Height: 1080})
	if err != nil {
		t.Fatalf("Layout() error = %v", err)
	}
	if !rect.Visible {
		t.Fatal("bottom-right layout should be visible")
	}
	if rect.X <= 0 || rect.Y <= 0 {
		t.Fatalf("bottom-right rect should be inset: %#v", rect)
	}
	if rect.X+rect.Width >= 1920 || rect.Y+rect.Height >= 1080 {
		t.Fatalf("bottom-right rect should stay inside canvas: %#v", rect)
	}
}

func TestLayoutBottomLeft(t *testing.T) {
	rect, err := Layout(PresetBottomLeft, Size{Width: 1920, Height: 1080})
	if err != nil {
		t.Fatalf("Layout() error = %v", err)
	}
	if rect.X >= 1920/2 {
		t.Fatalf("bottom-left x = %d, want left half", rect.X)
	}
}

func TestLayoutOff(t *testing.T) {
	rect, err := Layout(PresetOff, Size{Width: 1920, Height: 1080})
	if err != nil {
		t.Fatalf("Layout(off) error = %v", err)
	}
	if rect.Visible || rect.Width != 0 || rect.Height != 0 {
		t.Fatalf("off rect = %#v, want hidden zero rect", rect)
	}
}

func TestLayoutRejectsInvalidPreset(t *testing.T) {
	if _, err := Layout(Preset("top-right"), Size{Width: 1920, Height: 1080}); err == nil {
		t.Fatal("Layout() accepted invalid preset")
	}
}
