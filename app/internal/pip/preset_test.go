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
	if rect.Width != rect.Height {
		t.Fatalf("pip rect should be square for circle/square masks: %#v", rect)
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

func TestNormalizeConfigPreservesCustomPIPOptions(t *testing.T) {
	config := NormalizeConfig(Config{
		Preset:      PresetFree,
		Shape:       ShapeSquare,
		Mirror:      false,
		Position:    Position{X: 1.5, Y: -0.5},
		Scale:       0.5,
		EdgeFeather: 0.5,
	})
	if config.Shape != ShapeSquare {
		t.Fatalf("shape = %q, want square", config.Shape)
	}
	if config.Mirror {
		t.Fatal("mirror should preserve explicit false")
	}
	if config.Position.X != 1 || config.Position.Y != 0 {
		t.Fatalf("position = %#v, want clamped to 1,0", config.Position)
	}
	if config.Scale != MaximumScale {
		t.Fatalf("scale = %v, want max %v", config.Scale, MaximumScale)
	}
	if config.EdgeFeather != MaximumEdgeFeather {
		t.Fatalf("edge feather = %v, want max %v", config.EdgeFeather, MaximumEdgeFeather)
	}
}

func TestNormalizeConfigForPresetUsesLegacyPresetForEmptyConfig(t *testing.T) {
	config := NormalizeConfigForPreset(string(PresetBottomLeft), Config{})
	if config.Preset != PresetBottomLeft {
		t.Fatalf("preset = %q, want bottom-left", config.Preset)
	}
	if config.Shape != DefaultShape || !config.Mirror || config.Scale != DefaultScale || config.EdgeFeather != DefaultEdgeFeather {
		t.Fatalf("legacy config defaults = %#v", config)
	}
}

func TestPlaceFreeConfig(t *testing.T) {
	placement, err := Place(Config{
		Preset:      PresetFree,
		Shape:       ShapeSquare,
		Mirror:      true,
		Position:    Position{X: 0.5, Y: 0.25},
		Scale:       0.25,
		EdgeFeather: 0.2,
	}, Size{Width: 1920, Height: 1080})
	if err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	if !placement.Visible || !placement.Rect.Visible {
		t.Fatalf("placement should be visible: %#v", placement)
	}
	if placement.Shape != ShapeSquare || !placement.Mirror || placement.EdgeFeather != 0.2 {
		t.Fatalf("placement metadata = %#v, want square mirrored with edge feather", placement)
	}
	if placement.Rect.Width != 480 || placement.Rect.Height != 480 {
		t.Fatalf("rect size = %#v, want 480 square", placement.Rect)
	}
	if placement.Rect.X <= 0 || placement.Rect.Y <= 0 || placement.Rect.X+placement.Rect.Width >= 1920 || placement.Rect.Y+placement.Rect.Height >= 1080 {
		t.Fatalf("free placement should remain inside canvas: %#v", placement.Rect)
	}
}
