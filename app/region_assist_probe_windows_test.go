//go:build windows

package main

import (
	"os"
	"strings"
	"testing"
)

func TestRegionAssistDesktopProbeFromEnv(t *testing.T) {
	value := strings.TrimSpace(os.Getenv("RECORDINGFREEDOM_REGION_ASSIST_PROBE"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("RECORDINGFREEDOM_REGION_PROBE"))
	}
	if value == "" {
		t.Skip("set RECORDINGFREEDOM_REGION_ASSIST_PROBE=cursor or x,y to verify full region assist element recognition")
	}
	point, err := parseRegionUIProbePoint(value)
	if err != nil {
		t.Fatal(err)
	}
	runRegionAssistDesktopProbe(t, point)
}
