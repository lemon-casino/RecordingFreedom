//go:build cgo

package rnnoise

import "testing"

func TestNativeSuppressorProcessesOneFrame(t *testing.T) {
	if !Available() {
		t.Fatal("native build reported RNNoise as unavailable")
	}
	suppressor, err := New(1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer suppressor.Close()

	frame := make([]float32, FrameSize())
	for index := range frame {
		frame[index] = 0.01
	}
	if err := suppressor.ProcessFrame(frame); err != nil {
		t.Fatalf("ProcessFrame() error = %v", err)
	}
	if err := suppressor.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
}
