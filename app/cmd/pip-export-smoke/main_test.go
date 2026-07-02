package main

import (
	"testing"
	"time"
)

func TestRunRejectsNegativeCanvas(t *testing.T) {
	if _, err := run(options{canvasWidth: -1, timeout: time.Second}); err == nil {
		t.Fatal("run() error = nil, want negative canvas rejection")
	}
}
