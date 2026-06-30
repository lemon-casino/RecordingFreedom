package main

import (
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/devices"
)

func TestParseSourceType(t *testing.T) {
	tests := []struct {
		value string
		want  devices.CaptureSourceType
		ok    bool
	}{
		{value: "screen", want: devices.SourceScreen, ok: true},
		{value: " window ", want: devices.SourceWindow, ok: true},
		{value: "application", want: devices.SourceApplication, ok: true},
		{value: "camera"},
	}
	for _, test := range tests {
		got, err := parseSourceType(test.value)
		if test.ok && (err != nil || got != test.want) {
			t.Fatalf("parseSourceType(%q) = %q, %v; want %q", test.value, got, err, test.want)
		}
		if !test.ok && err == nil {
			t.Fatalf("parseSourceType(%q) error = nil, want error", test.value)
		}
	}
}

func TestChooseSourcePrefersAvailableMatchingType(t *testing.T) {
	sources := []devices.CaptureSource{
		{ID: "screen:queued", Type: devices.SourceScreen, Available: false},
		{ID: "window:1", Type: devices.SourceWindow, Available: true},
		{ID: "screen:display-1", Type: devices.SourceScreen, Available: true},
	}
	got, err := chooseSource(sources, "", devices.SourceScreen)
	if err != nil {
		t.Fatalf("chooseSource() error = %v", err)
	}
	if got.ID != "screen:display-1" {
		t.Fatalf("chooseSource() = %q, want screen:display-1", got.ID)
	}
}

func TestChooseSourceHonorsExplicitID(t *testing.T) {
	sources := []devices.CaptureSource{
		{ID: "screen:display-1", Type: devices.SourceScreen, Available: true},
		{ID: "screen:display-2", Type: devices.SourceScreen, Available: true},
	}
	got, err := chooseSource(sources, "screen:display-2", devices.SourceScreen)
	if err != nil {
		t.Fatalf("chooseSource() error = %v", err)
	}
	if got.ID != "screen:display-2" {
		t.Fatalf("chooseSource() = %q, want screen:display-2", got.ID)
	}
}
