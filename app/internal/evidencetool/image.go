package evidencetool

import (
	"fmt"
	"image"
	_ "image/jpeg" // register decoders so ImageSize keeps working for the tools whose local copies were converged
	_ "image/png"
	"os"
)

// SampledPixel is one sampled RGB point from an extracted video frame.
// Source: cmd/annotation-export-smoke, cmd/pip-export-smoke (identical sampledPixel).
type SampledPixel struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// PixelAt samples the pixel at (x, y) from frame (3 bytes per pixel) and
// returns a zero-color pixel when the offset is out of range.
// Source: cmd/annotation-export-smoke, cmd/pip-export-smoke.
func PixelAt(frame []byte, width int, x int, y int) SampledPixel {
	offset := (y*width + x) * 3
	if offset < 0 || offset+2 >= len(frame) {
		return SampledPixel{X: x, Y: y}
	}
	return SampledPixel{
		X: x,
		Y: y,
		R: frame[offset],
		G: frame[offset+1],
		B: frame[offset+2],
	}
}

// ImageSize returns the decoded width and height of the image at path and
// rejects non-positive dimensions. The image/jpeg and image/png decoders are
// registered by this package.
// Source: cmd/ocr-desktop-evidence-export, cmd/ocr-desktop-evidence-plan.
func ImageSize(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	if config.Width <= 0 || config.Height <= 0 {
		return 0, 0, fmt.Errorf("invalid image dimensions %dx%d", config.Width, config.Height)
	}
	return config.Width, config.Height, nil
}
