//go:build darwin && !cgo

package main

import (
	"fmt"
	"image"
)

func scrollDownAtRect(rect image.Rectangle) error {
	return fmt.Errorf("scrolling screenshot requires macOS Accessibility scroll automation; this macOS build was compiled without cgo")
}
