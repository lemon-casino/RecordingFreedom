package ocr

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	scrollingOCRTileHeight    = 2200
	scrollingOCRTileOverlap   = 120
	scrollingOCRMaxTileCount  = 80
	scrollingOCRDuplicateIoU  = 0.35
	scrollingOCRDuplicateYGap = 48
)

type scrollingOCRTile struct {
	Index  int
	X      int
	Y      int
	Width  int
	Height int
}

type blockBounds struct {
	left   float64
	top    float64
	right  float64
	bottom float64
}

func shouldTileScrollingImage(req RecognizeRequest, imagePath string) bool {
	if req.SourceKind != SourceScrollingScreenshot {
		return false
	}
	_, height, err := ImageDimensions(imagePath)
	return err == nil && height > defaultWorkerMaxSide
}

func (s *Service) recognizeScrollingImageTiles(req RecognizeRequest, imagePath string, imageSHA256 string, model ModelInfo) (Result, error) {
	fullImage, err := decodeOCRImage(imagePath)
	if err != nil {
		return Result{}, err
	}
	bounds := fullImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return Result{}, fmt.Errorf("scrolling OCR image %q has invalid dimensions %dx%d", imagePath, width, height)
	}
	tiles := splitScrollingOCRTiles(width, height)
	if len(tiles) <= 1 {
		return s.runWorkerRecognize(req, imageSHA256, model)
	}

	tempDir, err := os.MkdirTemp("", "recordingfreedom-ocr-scroll-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(tempDir)

	started := time.Now()
	blocks := make([]Block, 0, len(tiles)*4)
	for _, tile := range tiles {
		tilePath := filepath.Join(tempDir, fmt.Sprintf("tile-%03d-y-%06d.png", tile.Index, tile.Y))
		if err := writeScrollingOCRTile(tilePath, fullImage, tile); err != nil {
			return Result{}, err
		}
		tileSHA, err := fileSHA256(tilePath)
		if err != nil {
			return Result{}, err
		}
		tileReq := req
		tileReq.ImagePath = tilePath
		tileReq.SourceID = fmt.Sprintf("%s#tile:%d:%d", req.SourceID, tile.Y, tile.Height)
		tileResult, err := s.runWorkerRecognize(tileReq, tileSHA, model)
		if err != nil {
			return Result{}, fmt.Errorf("scrolling OCR tile %d at y=%d failed: %w", tile.Index, tile.Y, err)
		}
		for blockIndex, block := range tileResult.Blocks {
			blocks = append(blocks, translateScrollingOCRBlock(block, tile, tile.Index, blockIndex))
		}
	}

	blocks = dedupeScrollingOCRBlocks(blocks)
	plainText := plainTextFromBlocks(blocks)
	return Result{
		Width:      width,
		Height:     height,
		Blocks:     blocks,
		PlainText:  plainText,
		CreatedAt:  time.Now().UTC(),
		DurationMS: int(time.Since(started).Milliseconds()),
	}, nil
}

func splitScrollingOCRTiles(width int, height int) []scrollingOCRTile {
	if width <= 0 || height <= 0 {
		return nil
	}
	if height <= defaultWorkerMaxSide {
		return []scrollingOCRTile{{Index: 0, Width: width, Height: height}}
	}
	tileHeight := scrollingOCRTileHeight
	overlap := scrollingOCRTileOverlap
	if tileHeight <= overlap {
		overlap = tileHeight / 10
	}
	tiles := make([]scrollingOCRTile, 0, (height/tileHeight)+2)
	for y := 0; y < height && len(tiles) < scrollingOCRMaxTileCount; {
		bottom := y + tileHeight
		if bottom > height {
			bottom = height
		}
		tiles = append(tiles, scrollingOCRTile{
			Index:  len(tiles),
			Width:  width,
			Y:      y,
			Height: bottom - y,
		})
		if bottom >= height {
			break
		}
		nextY := bottom - overlap
		if nextY <= y {
			nextY = bottom
		}
		y = nextY
	}
	return tiles
}

func decodeOCRImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func writeScrollingOCRTile(path string, fullImage image.Image, tile scrollingOCRTile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	rect := image.Rect(tile.X, tile.Y, tile.X+tile.Width, tile.Y+tile.Height).Add(fullImage.Bounds().Min)
	dst := image.NewRGBA(image.Rect(0, 0, tile.Width, tile.Height))
	draw.Draw(dst, dst.Bounds(), fullImage, rect.Min, draw.Src)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, dst)
}

func translateScrollingOCRBlock(block Block, tile scrollingOCRTile, tileIndex int, blockIndex int) Block {
	if strings.TrimSpace(block.ID) == "" {
		block.ID = fmt.Sprintf("tile-%03d-block-%03d", tileIndex, blockIndex)
	} else {
		block.ID = fmt.Sprintf("tile-%03d-%s", tileIndex, block.ID)
	}
	for index := range block.Box {
		block.Box[index].X += float64(tile.X)
		block.Box[index].Y += float64(tile.Y)
	}
	return block
}

func dedupeScrollingOCRBlocks(blocks []Block) []Block {
	if len(blocks) <= 1 {
		return blocks
	}
	sortBlocksByPosition(blocks)
	kept := make([]Block, 0, len(blocks))
	for _, block := range blocks {
		if strings.TrimSpace(block.Text) == "" {
			continue
		}
		duplicateAt := -1
		for index, candidate := range kept {
			if scrollingOCRBlocksDuplicate(candidate, block) {
				duplicateAt = index
				break
			}
		}
		if duplicateAt >= 0 {
			if block.Confidence > kept[duplicateAt].Confidence {
				kept[duplicateAt] = block
			}
			continue
		}
		kept = append(kept, block)
	}
	sortBlocksByPosition(kept)
	for index := range kept {
		kept[index].LineIndex = index
	}
	return kept
}

func scrollingOCRBlocksDuplicate(left Block, right Block) bool {
	if normalizeOCRText(left.Text) != normalizeOCRText(right.Text) {
		return false
	}
	leftBounds, ok := ocrBlockBounds(left)
	if !ok {
		return false
	}
	rightBounds, ok := ocrBlockBounds(right)
	if !ok {
		return false
	}
	if boundsIoU(leftBounds, rightBounds) >= scrollingOCRDuplicateIoU {
		return true
	}
	return math.Abs(leftBounds.top-rightBounds.top) <= scrollingOCRDuplicateYGap &&
		math.Abs(leftBounds.bottom-rightBounds.bottom) <= scrollingOCRDuplicateYGap &&
		horizontalOverlapRatio(leftBounds, rightBounds) >= 0.7
}

func normalizeOCRText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func ocrBlockBounds(block Block) (blockBounds, bool) {
	if len(block.Box) == 0 {
		return blockBounds{}, false
	}
	left := math.Inf(1)
	top := math.Inf(1)
	right := math.Inf(-1)
	bottom := math.Inf(-1)
	for _, point := range block.Box {
		if point.X < left {
			left = point.X
		}
		if point.X > right {
			right = point.X
		}
		if point.Y < top {
			top = point.Y
		}
		if point.Y > bottom {
			bottom = point.Y
		}
	}
	if !isFiniteBounds(left, top, right, bottom) || right <= left || bottom <= top {
		return blockBounds{}, false
	}
	return blockBounds{left: left, top: top, right: right, bottom: bottom}, true
}

func isFiniteBounds(values ...float64) bool {
	for _, value := range values {
		if math.IsInf(value, 0) || math.IsNaN(value) {
			return false
		}
	}
	return true
}

func boundsIoU(left blockBounds, right blockBounds) float64 {
	interLeft := math.Max(left.left, right.left)
	interTop := math.Max(left.top, right.top)
	interRight := math.Min(left.right, right.right)
	interBottom := math.Min(left.bottom, right.bottom)
	if interRight <= interLeft || interBottom <= interTop {
		return 0
	}
	intersection := (interRight - interLeft) * (interBottom - interTop)
	leftArea := (left.right - left.left) * (left.bottom - left.top)
	rightArea := (right.right - right.left) * (right.bottom - right.top)
	union := leftArea + rightArea - intersection
	if union <= 0 {
		return 0
	}
	return intersection / union
}

func horizontalOverlapRatio(left blockBounds, right blockBounds) float64 {
	interLeft := math.Max(left.left, right.left)
	interRight := math.Min(left.right, right.right)
	if interRight <= interLeft {
		return 0
	}
	leftWidth := left.right - left.left
	rightWidth := right.right - right.left
	minWidth := math.Min(leftWidth, rightWidth)
	if minWidth <= 0 {
		return 0
	}
	return (interRight - interLeft) / minWidth
}

func sortBlocksByPosition(blocks []Block) {
	sort.SliceStable(blocks, func(i, j int) bool {
		leftBounds, leftOK := ocrBlockBounds(blocks[i])
		rightBounds, rightOK := ocrBlockBounds(blocks[j])
		if !leftOK || !rightOK {
			return blocks[i].ID < blocks[j].ID
		}
		if math.Abs(leftBounds.top-rightBounds.top) > 8 {
			return leftBounds.top < rightBounds.top
		}
		return leftBounds.left < rightBounds.left
	})
}

func plainTextFromBlocks(blocks []Block) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		text := strings.TrimSpace(block.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}
