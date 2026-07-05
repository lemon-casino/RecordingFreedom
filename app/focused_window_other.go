//go:build !darwin && !windows

package main

import "image"

func detectFocusedWindowScreenshotRect() (image.Rectangle, bool) {
	return image.Rectangle{}, false
}
