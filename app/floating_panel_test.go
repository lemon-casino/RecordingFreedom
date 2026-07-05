package main

import "testing"

func TestPatchSourceStateKeepsRegionGeometry(t *testing.T) {
	service := &RecordingFreedomService{}
	geometry := &SourceGeometry{
		X:            120,
		Y:            80,
		Width:        640,
		Height:       360,
		DisplayIndex: 2,
		NativeID:     "display:2",
	}
	state, err := service.PatchSourceState(SourceStatePatchRequest{
		RecordingMode:  "video",
		SourceID:       "region:custom",
		SourceType:     "region",
		SourceGeometry: geometry,
	})
	if err != nil {
		t.Fatalf("PatchSourceState(region) error = %v", err)
	}
	if state.SourceGeometry == nil {
		t.Fatalf("region geometry missing after region patch")
	}
	if state.SourceGeometry.Width != 640 || state.SourceGeometry.Height != 360 || state.SourceGeometry.NativeID != "display:2" {
		t.Fatalf("region geometry = %+v, want original geometry", state.SourceGeometry)
	}

	state, err = service.PatchSourceState(SourceStatePatchRequest{RecordingMode: "audio"})
	if err != nil {
		t.Fatalf("PatchSourceState(audio) error = %v", err)
	}
	if state.SourceGeometry == nil {
		t.Fatalf("region geometry was lost after recording-mode-only patch")
	}

	state, err = service.PatchSourceState(SourceStatePatchRequest{
		RecordingMode: "video",
		SourceID:      "screen:1",
		SourceType:    "screen",
	})
	if err != nil {
		t.Fatalf("PatchSourceState(screen) error = %v", err)
	}
	if state.SourceGeometry != nil {
		t.Fatalf("screen source kept region geometry: %+v", state.SourceGeometry)
	}
}

func TestPatchSourceStateIgnoresInvalidRecordingMode(t *testing.T) {
	service := &RecordingFreedomService{}
	state, err := service.PatchSourceState(SourceStatePatchRequest{RecordingMode: "video"})
	if err != nil {
		t.Fatalf("PatchSourceState(video) error = %v", err)
	}
	state, err = service.PatchSourceState(SourceStatePatchRequest{RecordingMode: "invalid"})
	if err != nil {
		t.Fatalf("PatchSourceState(invalid) error = %v", err)
	}
	if state.RecordingMode != "video" {
		t.Fatalf("recording mode = %q, want video", state.RecordingMode)
	}
}
