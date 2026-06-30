package pip

import "fmt"

type Preset string

const (
	PresetOff         Preset = "off"
	PresetBottomRight Preset = "bottom-right"
	PresetBottomLeft  Preset = "bottom-left"
	PresetFree        Preset = "free"
	DefaultPreset     Preset = PresetBottomRight
)

type Size struct {
	Width  int
	Height int
}

type Rect struct {
	X       int  `json:"x"`
	Y       int  `json:"y"`
	Width   int  `json:"width"`
	Height  int  `json:"height"`
	Visible bool `json:"visible"`
}

func Normalize(value string) Preset {
	preset := Preset(value)
	if IsValid(preset) {
		return preset
	}
	return DefaultPreset
}

func IsValid(preset Preset) bool {
	switch preset {
	case PresetOff, PresetBottomRight, PresetBottomLeft, PresetFree:
		return true
	default:
		return false
	}
}

func Validate(preset Preset) error {
	if IsValid(preset) {
		return nil
	}
	return fmt.Errorf("unsupported pip preset %q", preset)
}

func Layout(preset Preset, canvas Size) (Rect, error) {
	if err := Validate(preset); err != nil {
		return Rect{}, err
	}
	if canvas.Width <= 0 || canvas.Height <= 0 {
		return Rect{}, fmt.Errorf("invalid canvas size %dx%d", canvas.Width, canvas.Height)
	}
	if preset == PresetOff {
		return Rect{Visible: false}, nil
	}

	margin := maxInt(16, canvas.Width/40)
	width := clampInt(canvas.Width/5, 160, maxInt(1, canvas.Width-(margin*2)))
	height := width * 9 / 16
	maxHeight := maxInt(1, canvas.Height-(margin*2))
	if height > maxHeight {
		height = maxHeight
		width = height * 16 / 9
	}

	x := canvas.Width - width - margin
	if preset == PresetBottomLeft {
		x = margin
	}
	// Until custom coordinates exist, free mode uses the default visible corner.
	if preset == PresetFree {
		x = canvas.Width - width - margin
	}
	y := canvas.Height - height - margin
	return Rect{X: x, Y: y, Width: width, Height: height, Visible: true}, nil
}

func clampInt(value int, minimum int, maximum int) int {
	if maximum < minimum {
		return maximum
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
