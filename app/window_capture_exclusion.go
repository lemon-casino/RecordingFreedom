package main

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func setWindowCaptureExcluded(window *application.WebviewWindow, excluded bool) bool {
	if window == nil {
		return false
	}
	window.SetContentProtection(excluded)
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	default:
		return false
	}
}
