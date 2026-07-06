package main

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

const (
	clsInputWidth       = 192
	clsInputHeight      = 48
	recInputHeight      = 48
	recBaseInputWidth   = 320
	recMaxInputWidth    = 960
	recMinInputWidth    = 32
	clsRotateThreshold  = 0.90
	minRecognizedLength = 1
)

type ocrRecognitionOutput struct {
	Result    *ocr.Result
	Inference ortInferenceSummary
}

type recognizedTextBlock struct {
	Text       string
	Score      float64
	Box        detCandidateBox
	Angle      int
	AngleScore float32
}

var runONNXRecognition = runONNXRecognitionWithPurego

func runONNXRecognitionWithPurego(bundle *ortModelBundle, input imagePreprocessResult) (ocrRecognitionOutput, error) {
	if bundle == nil {
		return ocrRecognitionOutput{}, errors.New("model bundle is not initialized")
	}
	detection, err := bundle.runDetection(input)
	if err != nil {
		return ocrRecognitionOutput{}, err
	}
	recognized, err := bundle.recognizeCandidates(input, detection.Candidates)
	if err != nil {
		return ocrRecognitionOutput{Inference: detection}, err
	}
	if len(recognized) == 0 {
		return ocrRecognitionOutput{Inference: detection}, errors.New("OCR recognition returned no text blocks")
	}
	sortRecognizedBlocks(recognized)
	blocks := make([]ocr.Block, 0, len(recognized))
	lines := assignLineIndexes(recognized)
	plainLines := make([]string, 0, len(recognized))
	for i, item := range recognized {
		block := ocr.Block{
			ID:         fmt.Sprintf("b%d", i+1),
			Text:       item.Text,
			Confidence: item.Score,
			Box:        pointsFromDetBox(item.Box),
			LineIndex:  lines[i],
		}
		if looksMostlyASCII(item.Text) {
			block.LanguageHint = "en"
		} else {
			block.LanguageHint = "zh"
		}
		blocks = append(blocks, block)
		plainLines = append(plainLines, item.Text)
	}
	return ocrRecognitionOutput{
		Result: &ocr.Result{
			Width:     input.Summary.OriginalWidth,
			Height:    input.Summary.OriginalHeight,
			Blocks:    blocks,
			PlainText: strings.Join(plainLines, "\n"),
			CreatedAt: time.Now(),
		},
		Inference: detection,
	}, nil
}

func (b *ortModelBundle) recognizeCandidates(input imagePreprocessResult, candidates []detCandidateBox) ([]recognizedTextBlock, error) {
	if input.SourceImage == nil {
		return nil, errors.New("source image is not available")
	}
	clsModel := b.findModel("cls")
	recModel := b.findModel("rec")
	orientationMode := b.textlineOrientationMode
	if orientationMode == "" {
		orientationMode = ocr.TextlineOrientationCLS
	}
	if orientationMode == ocr.TextlineOrientationCLS && clsModel == nil {
		return nil, errors.New("cls model is not loaded")
	}
	if recModel == nil {
		return nil, errors.New("rec model is not loaded")
	}
	if len(b.characters) == 0 {
		return nil, errors.New("OCR character set is empty")
	}
	recognized := make([]recognizedTextBlock, 0, len(candidates))
	for _, candidate := range candidates {
		crop := cropCandidate(input.SourceImage, candidate)
		if crop == nil {
			continue
		}
		angle := 0
		var angleScore float32 = 1
		if orientationMode == ocr.TextlineOrientationCLS {
			var err error
			angle, angleScore, err = b.classifyAngle(clsModel, crop)
			if err != nil {
				return nil, err
			}
		}
		if angle == 180 {
			crop = rotateImage180(crop)
		}
		text, score, err := b.recognizeCrop(recModel, crop)
		if err != nil {
			return nil, err
		}
		text = strings.TrimSpace(text)
		if len([]rune(text)) < minRecognizedLength {
			continue
		}
		recognized = append(recognized, recognizedTextBlock{
			Text:       text,
			Score:      score,
			Box:        candidate,
			Angle:      angle,
			AngleScore: angleScore,
		})
	}
	return recognized, nil
}

func (b *ortModelBundle) classifyAngle(model *ortModelSession, crop *image.RGBA) (int, float32, error) {
	tensor, shape := tensorFromCrop(crop, clsInputHeight, clsInputWidth)
	output, err := b.runtime.api.runFloat32Session(model.Session, firstTensorName(model.Inputs), firstTensorName(model.Outputs), tensor, shape)
	if err != nil {
		return 0, 0, fmt.Errorf("cls inference failed: %w", err)
	}
	if len(output.Data) < 2 {
		return 0, 0, fmt.Errorf("cls output len = %d, want at least 2", len(output.Data))
	}
	if output.Data[1] > output.Data[0] && output.Data[1] >= clsRotateThreshold {
		return 180, output.Data[1], nil
	}
	return 0, output.Data[0], nil
}

func (b *ortModelBundle) recognizeCrop(model *ortModelSession, crop *image.RGBA) (string, float64, error) {
	targetWidth := recTargetWidth(crop.Bounds().Dx(), crop.Bounds().Dy())
	tensor, shape := tensorFromCrop(crop, recInputHeight, targetWidth)
	output, err := b.runtime.api.runFloat32Session(model.Session, firstTensorName(model.Inputs), firstTensorName(model.Outputs), tensor, shape)
	if err != nil {
		return "", 0, fmt.Errorf("rec inference failed: %w", err)
	}
	text, score, err := decodeCTC(output, b.characters)
	if err != nil {
		return "", 0, err
	}
	return text, score, nil
}

func loadOCRCharacters(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open keys.txt: %w", err)
	}
	defer file.Close()
	characters := []string{"blank"}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		value := strings.TrimRight(scanner.Text(), "\r\n")
		if value == "" {
			continue
		}
		characters = append(characters, value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read keys.txt: %w", err)
	}
	characters = append(characters, " ")
	return characters, nil
}

func decodeCTC(output ortTensorOutput, characters []string) (string, float64, error) {
	if len(output.Shape) < 3 {
		return "", 0, fmt.Errorf("rec output shape must be NTC, got %#v", output.Shape)
	}
	steps := int(output.Shape[len(output.Shape)-2])
	classes := int(output.Shape[len(output.Shape)-1])
	if steps <= 0 || classes <= 0 {
		return "", 0, fmt.Errorf("invalid rec output shape %#v", output.Shape)
	}
	if len(output.Data) < steps*classes {
		return "", 0, fmt.Errorf("rec output data len = %d, want at least %d", len(output.Data), steps*classes)
	}
	var builder strings.Builder
	var total float64
	var selected int
	previous := -1
	for step := 0; step < steps; step++ {
		offset := step * classes
		bestIndex := 0
		bestScore := output.Data[offset]
		for classIndex := 1; classIndex < classes; classIndex++ {
			value := output.Data[offset+classIndex]
			if value > bestScore {
				bestScore = value
				bestIndex = classIndex
			}
		}
		if bestIndex == 0 || bestIndex == previous {
			previous = bestIndex
			continue
		}
		if bestIndex >= len(characters) {
			previous = bestIndex
			continue
		}
		builder.WriteString(characters[bestIndex])
		total += float64(bestScore)
		selected++
		previous = bestIndex
	}
	if selected == 0 {
		return "", 0, nil
	}
	return builder.String(), total / float64(selected), nil
}

func cropCandidate(src *image.RGBA, candidate detCandidateBox) *image.RGBA {
	bounds := detBoxBounds(candidate)
	srcBounds := src.Bounds()
	minX := clampInt(int(math.Floor(bounds.minX)), srcBounds.Min.X, srcBounds.Max.X)
	minY := clampInt(int(math.Floor(bounds.minY)), srcBounds.Min.Y, srcBounds.Max.Y)
	maxX := clampInt(int(math.Ceil(bounds.maxX)), srcBounds.Min.X, srcBounds.Max.X)
	maxY := clampInt(int(math.Ceil(bounds.maxY)), srcBounds.Min.Y, srcBounds.Max.Y)
	if maxX-minX < 2 || maxY-minY < 2 {
		return nil
	}
	out := image.NewRGBA(image.Rect(0, 0, maxX-minX, maxY-minY))
	stddraw.Draw(out, out.Bounds(), src, image.Point{X: minX, Y: minY}, stddraw.Src)
	return out
}

func rotateImage180(src *image.RGBA) *image.RGBA {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			out.SetRGBA(width-1-x, height-1-y, src.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return out
}

func tensorFromCrop(crop *image.RGBA, targetHeight int, targetWidth int) ([]float32, []int64) {
	resizedWidth := resizedCropWidth(crop.Bounds().Dx(), crop.Bounds().Dy(), targetHeight, targetWidth)
	resized := resizeImage(crop, resizedWidth, targetHeight)
	planeSize := targetWidth * targetHeight
	tensor := make([]float32, 3*planeSize)
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < resizedWidth; x++ {
			pixel := resized.RGBAAt(x, y)
			index := y*targetWidth + x
			tensor[index] = normalizeHalf(pixel.R)
			tensor[planeSize+index] = normalizeHalf(pixel.G)
			tensor[2*planeSize+index] = normalizeHalf(pixel.B)
		}
	}
	return tensor, []int64{1, 3, int64(targetHeight), int64(targetWidth)}
}

func resizedCropWidth(width int, height int, targetHeight int, targetWidth int) int {
	if width <= 0 || height <= 0 {
		return 1
	}
	resizedWidth := int(math.Ceil(float64(targetHeight) * float64(width) / float64(height)))
	if resizedWidth < 1 {
		return 1
	}
	if resizedWidth > targetWidth {
		return targetWidth
	}
	return resizedWidth
}

func recTargetWidth(width int, height int) int {
	if width <= 0 || height <= 0 {
		return recBaseInputWidth
	}
	target := int(math.Ceil(float64(recInputHeight) * float64(width) / float64(height)))
	if target < recBaseInputWidth {
		target = recBaseInputWidth
	}
	if target > recMaxInputWidth {
		target = recMaxInputWidth
	}
	if target < recMinInputWidth {
		target = recMinInputWidth
	}
	return target
}

func normalizeHalf(value uint8) float32 {
	return (float32(value)/255.0 - 0.5) / 0.5
}

func sortRecognizedBlocks(blocks []recognizedTextBlock) {
	sort.SliceStable(blocks, func(i int, j int) bool {
		a := detBoxBounds(blocks[i].Box)
		b := detBoxBounds(blocks[j].Box)
		aHeight := a.maxY - a.minY
		bHeight := b.maxY - b.minY
		overlap := math.Min(a.maxY, b.maxY) - math.Max(a.minY, b.minY)
		if overlap > math.Min(aHeight, bHeight)*detReadingOrderYOverlap {
			return a.minX < b.minX
		}
		return a.minY < b.minY
	})
}

func assignLineIndexes(blocks []recognizedTextBlock) []int {
	lines := make([]int, len(blocks))
	line := 0
	var previous detBounds
	for i, block := range blocks {
		current := detBoxBounds(block.Box)
		if i > 0 && current.minY > previous.maxY {
			line++
		}
		lines[i] = line
		previous = current
	}
	return lines
}

func pointsFromDetBox(candidate detCandidateBox) []ocr.Point {
	points := make([]ocr.Point, 0, len(candidate.Box))
	for _, point := range candidate.Box {
		points = append(points, ocr.Point{X: point[0], Y: point[1]})
	}
	return points
}

func looksMostlyASCII(value string) bool {
	var ascii int
	var total int
	for _, r := range value {
		if r == ' ' {
			continue
		}
		total++
		if r <= 127 {
			ascii++
		}
	}
	return total > 0 && ascii*2 >= total
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func fillImage(img *image.RGBA, c color.RGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetRGBA(x, y, c)
		}
	}
}
