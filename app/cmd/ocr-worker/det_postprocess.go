package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	detScoreThreshold       = 0.30
	detMinComponentArea     = 8
	detMaxCandidateBoxes    = 128
	detReadingOrderYOverlap = 0.55
)

type detCandidateBox struct {
	Score float32      `json:"score"`
	Box   [][2]float64 `json:"box"`
}

type detComponent struct {
	minX  int
	minY  int
	maxX  int
	maxY  int
	area  int
	total float64
}

func detectCandidateBoxes(output ortTensorOutput, summary imagePreprocessSummary) ([]detCandidateBox, error) {
	if len(output.Shape) < 4 {
		return nil, fmt.Errorf("det output shape must be NCHW, got %#v", output.Shape)
	}
	height := int(output.Shape[len(output.Shape)-2])
	width := int(output.Shape[len(output.Shape)-1])
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid det output size %dx%d", width, height)
	}
	if len(output.Data) < width*height {
		return nil, fmt.Errorf("det output data len = %d, want at least %d", len(output.Data), width*height)
	}
	if summary.OriginalWidth <= 0 || summary.OriginalHeight <= 0 || summary.InputWidth <= 0 || summary.InputHeight <= 0 {
		return nil, errors.New("invalid preprocess summary for det coordinate mapping")
	}

	components := connectedDetComponents(output.Data[:width*height], width, height)
	boxes := make([]detCandidateBox, 0, len(components))
	for _, component := range components {
		if component.area < detMinComponentArea {
			continue
		}
		boxes = append(boxes, componentToOriginalBox(component, width, height, summary))
	}
	sortDetCandidateBoxes(boxes)
	if len(boxes) > detMaxCandidateBoxes {
		boxes = boxes[:detMaxCandidateBoxes]
	}
	return boxes, nil
}

func connectedDetComponents(scores []float32, width int, height int) []detComponent {
	visited := make([]bool, width*height)
	queue := make([]int, 0, width*height/8)
	components := make([]detComponent, 0)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			start := y*width + x
			if visited[start] || scores[start] < detScoreThreshold {
				continue
			}
			component := detComponent{
				minX: x,
				minY: y,
				maxX: x,
				maxY: y,
			}
			visited[start] = true
			queue = append(queue[:0], start)
			for len(queue) > 0 {
				index := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				px := index % width
				py := index / width
				component.area++
				component.total += float64(scores[index])
				if px < component.minX {
					component.minX = px
				}
				if px > component.maxX {
					component.maxX = px
				}
				if py < component.minY {
					component.minY = py
				}
				if py > component.maxY {
					component.maxY = py
				}
				for _, next := range detNeighborIndexes(px, py, width, height) {
					if visited[next] || scores[next] < detScoreThreshold {
						continue
					}
					visited[next] = true
					queue = append(queue, next)
				}
			}
			components = append(components, component)
		}
	}
	return components
}

func detNeighborIndexes(x int, y int, width int, height int) []int {
	neighbors := make([]int, 0, 4)
	if x > 0 {
		neighbors = append(neighbors, y*width+x-1)
	}
	if x+1 < width {
		neighbors = append(neighbors, y*width+x+1)
	}
	if y > 0 {
		neighbors = append(neighbors, (y-1)*width+x)
	}
	if y+1 < height {
		neighbors = append(neighbors, (y+1)*width+x)
	}
	return neighbors
}

func componentToOriginalBox(component detComponent, mapWidth int, mapHeight int, summary imagePreprocessSummary) detCandidateBox {
	inputScaleX := float64(summary.InputWidth) / float64(mapWidth)
	inputScaleY := float64(summary.InputHeight) / float64(mapHeight)
	componentWidth := float64(component.maxX-component.minX+1) * inputScaleX
	componentHeight := float64(component.maxY-component.minY+1) * inputScaleY
	paddingX := clampFloat(componentWidth*0.08, 4, 32)
	paddingY := clampFloat(componentHeight*0.75, 6, 32)
	minX := (float64(component.minX)*inputScaleX - paddingX) / summary.ScaleX
	minY := (float64(component.minY)*inputScaleY - paddingY) / summary.ScaleY
	maxX := (float64(component.maxX+1)*inputScaleX + paddingX) / summary.ScaleX
	maxY := (float64(component.maxY+1)*inputScaleY + paddingY) / summary.ScaleY
	minX = clampFloat(minX, 0, float64(summary.OriginalWidth))
	minY = clampFloat(minY, 0, float64(summary.OriginalHeight))
	maxX = clampFloat(maxX, 0, float64(summary.OriginalWidth))
	maxY = clampFloat(maxY, 0, float64(summary.OriginalHeight))
	return detCandidateBox{
		Score: float32(component.total / math.Max(1, float64(component.area))),
		Box: [][2]float64{
			{minX, minY},
			{maxX, minY},
			{maxX, maxY},
			{minX, maxY},
		},
	}
}

func sortDetCandidateBoxes(boxes []detCandidateBox) {
	sort.SliceStable(boxes, func(i int, j int) bool {
		a := detBoxBounds(boxes[i])
		b := detBoxBounds(boxes[j])
		aHeight := a.maxY - a.minY
		bHeight := b.maxY - b.minY
		overlap := math.Min(a.maxY, b.maxY) - math.Max(a.minY, b.minY)
		if overlap > math.Min(aHeight, bHeight)*detReadingOrderYOverlap {
			return a.minX < b.minX
		}
		return a.minY < b.minY
	})
}

type detBounds struct {
	minX float64
	minY float64
	maxX float64
	maxY float64
}

func detBoxBounds(box detCandidateBox) detBounds {
	bounds := detBounds{minX: math.Inf(1), minY: math.Inf(1), maxX: math.Inf(-1), maxY: math.Inf(-1)}
	for _, point := range box.Box {
		if point[0] < bounds.minX {
			bounds.minX = point[0]
		}
		if point[0] > bounds.maxX {
			bounds.maxX = point[0]
		}
		if point[1] < bounds.minY {
			bounds.minY = point[1]
		}
		if point[1] > bounds.maxY {
			bounds.maxY = point[1]
		}
	}
	return bounds
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
