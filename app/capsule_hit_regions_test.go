package main

import "testing"

func TestCapsuleHitRegionsPassThroughOutsideVisibleRegions(t *testing.T) {
	var regions capsuleWindowHitRegions
	state := regions.Set(CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		ViewportWidth:  960,
		ViewportHeight: 640,
		Regions: []CapsuleWindowHitRegion{
			{X: 24, Y: 10, Width: 912, Height: 72, Kind: "pill", Radius: 999},
			{X: 86, Y: 86, Width: 430, Height: 520, Kind: "round-rect", Radius: 22},
		},
	})
	if !state.enabled || len(state.regions) != 2 {
		t.Fatalf("state = %#v, want enabled with 2 regions", state)
	}
	if state.regions[0].Radius != 38 {
		t.Fatalf("capsule radius = %v, want clamped half height 38", state.regions[0].Radius)
	}

	for _, tt := range []struct {
		name string
		x    int
		y    int
		hit  bool
	}{
		{name: "capsule", x: 500, y: 42, hit: true},
		{name: "panel", x: 120, y: 180, hit: true},
		{name: "left transparent gutter", x: 40, y: 200, hit: false},
		{name: "right transparent gutter", x: 760, y: 200, hit: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handled, hit := regions.TestClientPoint(tt.x, tt.y, 960, 640)
			if !handled {
				t.Fatalf("TestClientPoint handled = false, want true")
			}
			if hit != tt.hit {
				t.Fatalf("TestClientPoint hit = %v, want %v", hit, tt.hit)
			}
		})
	}
}

func TestCapsuleHitRegionsScaleClientPixelsToCSSViewport(t *testing.T) {
	var regions capsuleWindowHitRegions
	regions.Set(CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		ViewportWidth:  960,
		ViewportHeight: 640,
		Regions: []CapsuleWindowHitRegion{
			{X: 24, Y: 10, Width: 912, Height: 72},
		},
	})

	handled, hit := regions.TestClientPoint(960, 84, 1920, 1280)
	if !handled || !hit {
		t.Fatalf("scaled capsule point handled/hit = %v/%v, want true/true", handled, hit)
	}

	handled, hit = regions.TestClientPoint(120, 400, 1920, 1280)
	if !handled || hit {
		t.Fatalf("scaled blank point handled/hit = %v/%v, want true/false", handled, hit)
	}
}

func TestCapsuleHitRegionsDisabledUntilValidGeometry(t *testing.T) {
	var regions capsuleWindowHitRegions
	regions.Set(CapsuleWindowHitRegionsRequest{Enabled: true})

	handled, hit := regions.TestClientPoint(10, 10, 960, 112)
	if handled || !hit {
		t.Fatalf("empty geometry handled/hit = %v/%v, want false/true", handled, hit)
	}
}
