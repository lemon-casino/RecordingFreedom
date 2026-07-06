package main

import "testing"

func TestDetectCandidateBoxesMapsAndSortsByReadingOrder(t *testing.T) {
	data := make([]float32, 20*10)
	fillDetRect(data, 20, 2, 1, 7, 3, 0.91)
	fillDetRect(data, 20, 10, 1, 16, 3, 0.82)
	fillDetRect(data, 20, 1, 6, 5, 8, 0.77)
	output := ortTensorOutput{
		Name:  "fetch_name_0",
		Shape: []int64{1, 1, 10, 20},
		Data:  data,
	}
	boxes, err := detectCandidateBoxes(output, imagePreprocessSummary{
		OriginalWidth:  200,
		OriginalHeight: 100,
		InputWidth:     200,
		InputHeight:    100,
		ScaleX:         1,
		ScaleY:         1,
	})
	if err != nil {
		t.Fatalf("detectCandidateBoxes() error = %v", err)
	}
	if len(boxes) != 3 {
		t.Fatalf("len(boxes) = %d, want 3: %#v", len(boxes), boxes)
	}
	first := detBoxBounds(boxes[0])
	second := detBoxBounds(boxes[1])
	third := detBoxBounds(boxes[2])
	if first.minX >= second.minX {
		t.Fatalf("first two boxes are not sorted left-to-right on the same line: %#v %#v", first, second)
	}
	if third.minY <= first.minY {
		t.Fatalf("third box should be on the next line: %#v %#v", first, third)
	}
	if boxes[0].Score < 0.90 || boxes[1].Score < 0.80 || boxes[2].Score < 0.70 {
		t.Fatalf("scores = %#v", boxes)
	}
}

func TestDetectCandidateBoxesRejectsInvalidOutput(t *testing.T) {
	_, err := detectCandidateBoxes(ortTensorOutput{Shape: []int64{1, 1, 2, 2}, Data: []float32{1}}, imagePreprocessSummary{
		OriginalWidth:  10,
		OriginalHeight: 10,
		InputWidth:     10,
		InputHeight:    10,
		ScaleX:         1,
		ScaleY:         1,
	})
	if err == nil {
		t.Fatal("detectCandidateBoxes() succeeded, want invalid data error")
	}
}

func fillDetRect(data []float32, width int, minX int, minY int, maxX int, maxY int, value float32) {
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			data[y*width+x] = value
		}
	}
}
