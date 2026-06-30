package video

import "testing"

func TestDarwinDisplayID(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		wantID   uint32
		wantOK   bool
	}{
		{name: "display", sourceID: "screen:display-69733632", wantID: 69733632, wantOK: true},
		{name: "trimmed", sourceID: " screen:display-42 ", wantID: 42, wantOK: true},
		{name: "window", sourceID: "window:20"},
		{name: "empty display", sourceID: "screen:display-"},
		{name: "zero", sourceID: "screen:display-0"},
		{name: "negative", sourceID: "screen:display--1"},
		{name: "overflow", sourceID: "screen:display-4294967296"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := DarwinDisplayID(tt.sourceID)
			if gotOK != tt.wantOK || gotID != tt.wantID {
				t.Fatalf("DarwinDisplayID(%q) = (%d, %v), want (%d, %v)", tt.sourceID, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestDarwinWindowID(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		wantID   uint32
		wantOK   bool
	}{
		{name: "window", sourceID: "window:402", wantID: 402, wantOK: true},
		{name: "trimmed", sourceID: " window:9 ", wantID: 9, wantOK: true},
		{name: "screen", sourceID: "screen:display-1"},
		{name: "empty", sourceID: "window:"},
		{name: "zero", sourceID: "window:0"},
		{name: "hex", sourceID: "window:0x22"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := DarwinWindowID(tt.sourceID)
			if gotOK != tt.wantOK || gotID != tt.wantID {
				t.Fatalf("DarwinWindowID(%q) = (%d, %v), want (%d, %v)", tt.sourceID, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}
