//go:build linux

package main

import (
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"
)

func scrollDownAtRect(rect image.Rectangle) error {
	if rect.Empty() {
		return fmt.Errorf("scrolling screenshot rectangle is empty")
	}
	xdotool, err := exec.LookPath("xdotool")
	if err != nil {
		return fmt.Errorf("scrolling screenshot requires xdotool on Linux/X11; Wayland or missing xdotool cannot be controlled safely: %w", err)
	}
	originalX, originalY, hasOriginal := linuxMouseLocation(xdotool)
	centerX := rect.Min.X + rect.Dx()/2
	centerY := rect.Min.Y + rect.Dy()/2
	clicks := strconv.Itoa(maxInt(2, minInt(5, rect.Dy()/120)))
	if err := exec.Command(xdotool, "mousemove", strconv.Itoa(centerX), strconv.Itoa(centerY), "click", "--repeat", clicks, "5").Run(); err != nil {
		return fmt.Errorf("send Linux scrolling screenshot wheel input: %w", err)
	}
	if hasOriginal {
		_ = exec.Command(xdotool, "mousemove", strconv.Itoa(originalX), strconv.Itoa(originalY)).Run()
	}
	return nil
}

func linuxMouseLocation(xdotool string) (int, int, bool) {
	output, err := exec.Command(xdotool, "getmouselocation", "--shell").Output()
	if err != nil {
		return 0, 0, false
	}
	values := map[string]int{}
	for _, line := range strings.Split(string(output), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		values[strings.TrimSpace(key)] = parsed
	}
	x, okX := values["X"]
	y, okY := values["Y"]
	return x, y, okX && okY
}
