//go:build !windows && !darwin && !linux

package main

import (
	"fmt"
	"image"
	"runtime"
)

func scrollDownAtRect(rect image.Rectangle) error {
	return fmt.Errorf("scrolling screenshot is not supported on %s", runtime.GOOS)
}
