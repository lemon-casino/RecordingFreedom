package main

import "testing"

func TestCapsuleHitRegionsPassThroughOutsideVisibleRegions(t *testing.T) {
	var regions capsuleWindowHitRegions
	state := regions.Set(CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		ViewportWidth:  760,
		ViewportHeight: 600,
		Regions: []CapsuleWindowHitRegion{
			{X: 18, Y: 8, Width: 704, Height: 64, Kind: "pill", Radius: 999},
			{X: 68, Y: 86, Width: 400, Height: 500, Kind: "round-rect", Radius: 22},
		},
	})
	if !state.enabled || len(state.regions) != 2 {
		t.Fatalf("state = %#v, want enabled with 2 regions", state)
	}
	if state.regions[0].Radius != 34 {
		t.Fatalf("capsule radius = %v, want clamped half height 34", state.regions[0].Radius)
	}

	for _, tt := range []struct {
		name string
		x    int
		y    int
		hit  bool
	}{
		{name: "capsule", x: 380, y: 40, hit: true},
		{name: "panel", x: 120, y: 180, hit: true},
		{name: "left transparent gutter", x: 40, y: 200, hit: false},
		{name: "right transparent gutter", x: 700, y: 200, hit: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handled, hit := regions.TestClientPoint(tt.x, tt.y, 760, 600)
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
		ViewportWidth:  760,
		ViewportHeight: 600,
		Regions: []CapsuleWindowHitRegion{
			{X: 18, Y: 8, Width: 704, Height: 64},
		},
	})

	handled, hit := regions.TestClientPoint(760, 80, 1520, 1200)
	if !handled || !hit {
		t.Fatalf("scaled capsule point handled/hit = %v/%v, want true/true", handled, hit)
	}

	handled, hit = regions.TestClientPoint(120, 400, 1520, 1200)
	if !handled || hit {
		t.Fatalf("scaled blank point handled/hit = %v/%v, want true/false", handled, hit)
	}
}

func TestAnnotationOverlayHitRegionsKeepCanvasTransparentUntilDrawing(t *testing.T) {
	var regions capsuleWindowHitRegions
	regions.Set(CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		ViewportWidth:  1280,
		ViewportHeight: 720,
		Regions: []CapsuleWindowHitRegion{
			{X: 430, Y: 14, Width: 420, Height: 52, Kind: "pill", Radius: 999},
		},
	})

	for _, tt := range []struct {
		name string
		x    int
		y    int
		hit  bool
	}{
		{name: "annotation capsule controls", x: 640, y: 40, hit: true},
		{name: "recorded canvas center", x: 640, y: 360, hit: false},
		{name: "recorded canvas lower corner", x: 1200, y: 680, hit: false},
	} {
		t.Run("pass-through/"+tt.name, func(t *testing.T) {
			handled, hit := regions.TestClientPoint(tt.x, tt.y, 1280, 720)
			if !handled {
				t.Fatalf("TestClientPoint handled = false, want true")
			}
			if hit != tt.hit {
				t.Fatalf("TestClientPoint hit = %v, want %v", hit, tt.hit)
			}
		})
	}

	regions.Set(CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		ViewportWidth:  1280,
		ViewportHeight: 720,
		Regions: []CapsuleWindowHitRegion{
			{X: 430, Y: 14, Width: 420, Height: 52, Kind: "pill", Radius: 999},
			{X: 0, Y: 0, Width: 1280, Height: 720, Kind: "rect"},
		},
	})
	handled, hit := regions.TestClientPoint(640, 360, 2560, 1440)
	if !handled || !hit {
		t.Fatalf("scaled drawing canvas point handled/hit = %v/%v, want true/true", handled, hit)
	}
}

func TestCapsuleHitRegionsDisabledUntilValidGeometry(t *testing.T) {
	var regions capsuleWindowHitRegions
	regions.Set(CapsuleWindowHitRegionsRequest{Enabled: true})

	handled, hit := regions.TestClientPoint(10, 10, 760, 96)
	if handled || !hit {
		t.Fatalf("empty geometry handled/hit = %v/%v, want false/true", handled, hit)
	}
}

func TestCapsuleHitRegionsUpdateSkipsUnchangedState(t *testing.T) {
	var regions capsuleWindowHitRegions
	request := CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		ViewportWidth:  760,
		ViewportHeight: 96,
		Regions: []CapsuleWindowHitRegion{
			{X: 18, Y: 8, Width: 704, Height: 64, Kind: "pill", Radius: 999},
		},
	}

	if _, changed := regions.Update(request); !changed {
		t.Fatalf("first update changed = false, want true")
	}
	if _, changed := regions.Update(request); changed {
		t.Fatalf("unchanged update changed = true, want false")
	}
	if _, changed := regions.Update(CapsuleWindowHitRegionsRequest{}); !changed {
		t.Fatalf("disable update changed = false, want true")
	}
	if _, changed := regions.Update(CapsuleWindowHitRegionsRequest{}); changed {
		t.Fatalf("repeated disabled update changed = true, want false")
	}
}

func TestCapsuleHitRegionsForceDoesNotChangeGeometrySignature(t *testing.T) {
	var regions capsuleWindowHitRegions
	request := CapsuleWindowHitRegionsRequest{
		Enabled:        true,
		ViewportWidth:  760,
		ViewportHeight: 96,
		Regions: []CapsuleWindowHitRegion{
			{X: 18, Y: 8, Width: 704, Height: 64, Kind: "pill", Radius: 999},
		},
	}
	state, changed := regions.Update(request)
	if !changed {
		t.Fatalf("first update changed = false, want true")
	}
	forced := request
	forced.Force = true
	next, changed := regions.Update(forced)
	if changed {
		t.Fatalf("forced update changed geometry = true, want false")
	}
	if !capsuleHitRegionStatesEqual(state, next) {
		t.Fatalf("forced state = %#v, want unchanged %#v", next, state)
	}
}
