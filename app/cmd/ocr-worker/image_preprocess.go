package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	defaultImagePreprocessMaxSide = 2400
	ocrInputDimensionMultiple     = 32
)

var (
	ocrImageMean = [3]float32{0.485, 0.456, 0.406}
	ocrImageStd  = [3]float32{0.229, 0.224, 0.225}
)

type imagePreprocessResult struct {
	Summary     imagePreprocessSummary
	Tensor      []float32   `json:"-"`
	SourceImage *image.RGBA `json:"-"`
}

type imagePreprocessSummary struct {
	Format         string  `json:"format,omitempty"`
	OriginalWidth  int     `json:"originalWidth"`
	OriginalHeight int     `json:"originalHeight"`
	InputWidth     int     `json:"inputWidth"`
	InputHeight    int     `json:"inputHeight"`
	ScaleX         float64 `json:"scaleX"`
	ScaleY         float64 `json:"scaleY"`
	MaxSide        int     `json:"maxSide"`
	Shape          []int64 `json:"shape"`
}

func preprocessImageFile(imagePath string, maxSide int) (imagePreprocessResult, error) {
	imagePath = strings.TrimSpace(imagePath)
	if imagePath == "" {
		return imagePreprocessResult{}, errors.New("imagePath is required")
	}
	file, err := os.Open(imagePath)
	if err != nil {
		return imagePreprocessResult{}, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return imagePreprocessResult{}, fmt.Errorf("decode image: %w", err)
	}
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	if originalWidth <= 0 || originalHeight <= 0 {
		return imagePreprocessResult{}, fmt.Errorf("invalid image size %dx%d", originalWidth, originalHeight)
	}
	normalizedMaxSide := normalizeMaxSide(maxSide)
	inputWidth, inputHeight := scaledOCRDimensions(originalWidth, originalHeight, normalizedMaxSide)
	whiteRGB := compositeImageToWhite(img)
	inputImage := resizeImage(whiteRGB, inputWidth, inputHeight)
	tensor := imageToNormalizedCHW(inputImage)

	return imagePreprocessResult{
		Summary: imagePreprocessSummary{
			Format:         format,
			OriginalWidth:  originalWidth,
			OriginalHeight: originalHeight,
			InputWidth:     inputWidth,
			InputHeight:    inputHeight,
			ScaleX:         float64(inputWidth) / float64(originalWidth),
			ScaleY:         float64(inputHeight) / float64(originalHeight),
			MaxSide:        normalizedMaxSide,
			Shape:          []int64{1, 3, int64(inputHeight), int64(inputWidth)},
		},
		Tensor:      tensor,
		SourceImage: whiteRGB,
	}, nil
}

func normalizeMaxSide(maxSide int) int {
	if maxSide <= 0 {
		maxSide = defaultImagePreprocessMaxSide
	}
	if maxSide < ocrInputDimensionMultiple {
		return ocrInputDimensionMultiple
	}
	return maxSide
}

func scaledOCRDimensions(width int, height int, maxSide int) (int, int) {
	longest := width
	if height > longest {
		longest = height
	}
	scale := 1.0
	if longest > maxSide {
		scale = float64(maxSide) / float64(longest)
	}
	targetWidth := int(math.Round(float64(width) * scale))
	targetHeight := int(math.Round(float64(height) * scale))
	return roundedOCRDimension(targetWidth, maxSide), roundedOCRDimension(targetHeight, maxSide)
}

func roundedOCRDimension(value int, maxSide int) int {
	if value <= 0 {
		return ocrInputDimensionMultiple
	}
	rounded := int(math.Round(float64(value)/float64(ocrInputDimensionMultiple))) * ocrInputDimensionMultiple
	if rounded < ocrInputDimensionMultiple {
		rounded = ocrInputDimensionMultiple
	}
	if rounded > maxSide {
		rounded = (maxSide / ocrInputDimensionMultiple) * ocrInputDimensionMultiple
		if rounded < ocrInputDimensionMultiple {
			rounded = ocrInputDimensionMultiple
		}
	}
	return rounded
}

func compositeImageToWhite(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			alpha := float64(a) / 65535.0
			out.SetRGBA(x, y, color.RGBA{
				R: compositePremultipliedChannelOnWhite(r, alpha),
				G: compositePremultipliedChannelOnWhite(g, alpha),
				B: compositePremultipliedChannelOnWhite(b, alpha),
				A: 255,
			})
		}
	}
	return out
}

func compositePremultipliedChannelOnWhite(value uint32, alpha float64) uint8 {
	blended := float64(value)/257.0 + 255.0*(1.0-alpha)
	if blended < 0 {
		return 0
	}
	if blended > 255 {
		return 255
	}
	return uint8(math.Round(blended))
}

func resizeImage(src *image.RGBA, width int, height int) *image.RGBA {
	if src.Bounds().Dx() == width && src.Bounds().Dy() == height {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst
}

func imageToNormalizedCHW(img *image.RGBA) []float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	planeSize := width * height
	tensor := make([]float32, 3*planeSize)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			index := y*width + x
			tensor[index] = normalizeImageChannel(pixel.R, ocrImageMean[0], ocrImageStd[0])
			tensor[planeSize+index] = normalizeImageChannel(pixel.G, ocrImageMean[1], ocrImageStd[1])
			tensor[2*planeSize+index] = normalizeImageChannel(pixel.B, ocrImageMean[2], ocrImageStd[2])
		}
	}
	return tensor
}

func normalizeImageChannel(value uint8, mean float32, std float32) float32 {
	return (float32(value)/255.0 - mean) / std
}
