//go:build !darwin || !cgo

package video

import (
	"strings"
	"testing"
)

func TestNewPlatformSessionIsExplicitlyUnsupportedByDefault(t *testing.T) {
	_, err := NewPlatformSession(CaptureConfig{
		Backend:  "screencapturekit",
		SourceID: "screen:display-1",
	})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("NewPlatformSession() error = %v, want not implemented", err)
	}
}
