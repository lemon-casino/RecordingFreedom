package video

import "testing"

func TestWindowsScreenID(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		wantID   string
		wantOK   bool
	}{
		{name: "display device token", sourceID: "screen:--display1", wantID: "--display1", wantOK: true},
		{name: "trimmed", sourceID: " screen:display-2 ", wantID: "display-2", wantOK: true},
		{name: "window", sourceID: "window:1234"},
		{name: "empty", sourceID: "screen:"},
		{name: "queued fallback", sourceID: "screen:native-backend-queued"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := WindowsScreenID(tt.sourceID)
			if gotOK != tt.wantOK || gotID != tt.wantID {
				t.Fatalf("WindowsScreenID(%q) = (%q, %v), want (%q, %v)", tt.sourceID, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestWindowsWindowHWND(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		wantID   uintptr
		wantOK   bool
	}{
		{name: "hex hwnd", sourceID: "window:1f04aa", wantID: 0x1f04aa, wantOK: true},
		{name: "trimmed", sourceID: " window:2A ", wantID: 0x2a, wantOK: true},
		{name: "screen", sourceID: "screen:--display1"},
		{name: "empty", sourceID: "window:"},
		{name: "zero", sourceID: "window:0"},
		{name: "invalid", sourceID: "window:not-hex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := WindowsWindowHWND(tt.sourceID)
			if gotOK != tt.wantOK || gotID != tt.wantID {
				t.Fatalf("WindowsWindowHWND(%q) = (%x, %v), want (%x, %v)", tt.sourceID, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestWindowsApplicationPID(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		wantID   uint32
		wantOK   bool
	}{
		{name: "pid", sourceID: "application:1201", wantID: 1201, wantOK: true},
		{name: "trimmed", sourceID: " application:99 ", wantID: 99, wantOK: true},
		{name: "window", sourceID: "window:99"},
		{name: "empty", sourceID: "application:"},
		{name: "zero", sourceID: "application:0"},
		{name: "negative", sourceID: "application:-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := WindowsApplicationPID(tt.sourceID)
			if gotOK != tt.wantOK || gotID != tt.wantID {
				t.Fatalf("WindowsApplicationPID(%q) = (%d, %v), want (%d, %v)", tt.sourceID, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}
