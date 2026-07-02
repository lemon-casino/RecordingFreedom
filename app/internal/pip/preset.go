package pip

import "fmt"

type Preset string
type Shape string

const (
	PresetOff         Preset = "off"
	PresetBottomRight Preset = "bottom-right"
	PresetBottomLeft  Preset = "bottom-left"
	PresetFree        Preset = "free"
	DefaultPreset     Preset = PresetBottomRight

	ShapeCircle        Shape = "circle"
	ShapeSquare        Shape = "square"
	DefaultShape       Shape = ShapeCircle
	DefaultScale             = 0.08
	DefaultEdgeFeather       = 0.16
	MinimumScale             = 0.08
	MaximumScale             = 0.32
	MinimumPixelSize         = 72
	MaximumEdgeFeather       = 0.42
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

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Config struct {
	Preset      Preset   `json:"preset"`
	Shape       Shape    `json:"shape"`
	Mirror      bool     `json:"mirror"`
	Position    Position `json:"position"`
	Scale       float64  `json:"scale"`
	EdgeFeather float64  `json:"edgeFeather"`
}

type Placement struct {
	Visible     bool    `json:"visible"`
	Rect        Rect    `json:"rect"`
	Shape       Shape   `json:"shape"`
	Mirror      bool    `json:"mirror"`
	EdgeFeather float64 `json:"edgeFeather"`
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

func NormalizeShape(value Shape) Shape {
	switch value {
	case ShapeCircle, ShapeSquare:
		return value
	default:
		return DefaultShape
	}
}

func DefaultConfig() Config {
	return ConfigFromPreset(DefaultPreset)
}

func OffConfig() Config {
	config := ConfigFromPreset(PresetOff)
	config.Preset = PresetOff
	return config
}

func ConfigFromPreset(preset Preset) Config {
	preset = Normalize(string(preset))
	position := Position{X: 1, Y: 1}
	if preset == PresetBottomLeft {
		position = Position{X: 0, Y: 1}
	}
	return Config{
		Preset:      preset,
		Shape:       DefaultShape,
		Mirror:      true,
		Position:    position,
		Scale:       DefaultScale,
		EdgeFeather: DefaultEdgeFeather,
	}
}

func NormalizeConfig(config Config) Config {
	if config.Preset == "" {
		config.Preset = DefaultPreset
	}
	if config.Shape == "" &&
		!config.Mirror &&
		config.Position.X == 0 &&
		config.Position.Y == 0 &&
		config.Scale == 0 &&
		config.EdgeFeather == 0 {
		config = ConfigFromPreset(Normalize(string(config.Preset)))
	}
	config.Preset = Normalize(string(config.Preset))
	config.Shape = NormalizeShape(config.Shape)
	config.Position = normalizePosition(config.Position)
	config.Scale = normalizeRatio(config.Scale, DefaultScale, MinimumScale, MaximumScale)
	config.EdgeFeather = normalizeRatio(config.EdgeFeather, DefaultEdgeFeather, 0, MaximumEdgeFeather)
	return config
}

func NormalizeConfigForPreset(preset string, config Config) Config {
	if config.Preset == "" {
		if isZeroConfig(config) {
			config = ConfigFromPreset(Normalize(preset))
		} else {
			config.Preset = Normalize(preset)
		}
	}
	return NormalizeConfig(config)
}

func isZeroConfig(config Config) bool {
	return config.Preset == "" &&
		config.Shape == "" &&
		!config.Mirror &&
		config.Position.X == 0 &&
		config.Position.Y == 0 &&
		config.Scale == 0 &&
		config.EdgeFeather == 0
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
	placement, err := Place(ConfigFromPreset(preset), canvas)
	if err != nil {
		return Rect{}, err
	}
	return placement.Rect, nil
}

func Place(config Config, canvas Size) (Placement, error) {
	config = NormalizeConfig(config)
	if canvas.Width <= 0 || canvas.Height <= 0 {
		return Placement{}, fmt.Errorf("invalid canvas size %dx%d", canvas.Width, canvas.Height)
	}
	if config.Preset == PresetOff {
		return Placement{
			Visible:     false,
			Rect:        Rect{Visible: false},
			Shape:       config.Shape,
			Mirror:      config.Mirror,
			EdgeFeather: config.EdgeFeather,
		}, nil
	}

	margin := maxInt(16, canvas.Width/40)
	size := clampInt(int(float64(canvas.Width)*config.Scale), MinimumPixelSize, maxInt(1, minInt(canvas.Width, canvas.Height)-(margin*2)))
	x := canvas.Width - size - margin
	if config.Preset == PresetBottomLeft {
		x = margin
	}
	if config.Preset == PresetFree {
		x = margin + int(float64(maxInt(0, canvas.Width-size-(margin*2)))*config.Position.X)
	}
	y := canvas.Height - size - margin
	if config.Preset == PresetFree {
		y = margin + int(float64(maxInt(0, canvas.Height-size-(margin*2)))*config.Position.Y)
	}
	rect := Rect{X: x, Y: y, Width: size, Height: size, Visible: true}
	return Placement{
		Visible:     true,
		Rect:        rect,
		Shape:       config.Shape,
		Mirror:      config.Mirror,
		EdgeFeather: config.EdgeFeather,
	}, nil
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

func normalizePosition(position Position) Position {
	position.X = clampFloat(position.X, 0, 1)
	position.Y = clampFloat(position.Y, 0, 1)
	return position
}

func normalizeRatio(value float64, fallback float64, minimum float64, maximum float64) float64 {
	if value <= 0 {
		value = fallback
	}
	return clampFloat(value, minimum, maximum)
}

func clampFloat(value float64, minimum float64, maximum float64) float64 {
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

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
