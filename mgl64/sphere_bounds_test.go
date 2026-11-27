// This file is generated from mgl32/sphere_bounds_test.go; DO NOT EDIT

// Copyright 2024 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"math"
	"math/rand"
	"testing"
)

// canonicalSphereView builds the FOV 90 degree, square-viewport setup used by
// the reference checks: identity view, sphere center (0,0,-5), radius 1.
func canonicalSphereView() (Mat4, Mat4) {
	return Ident4(), Perspective(DegToRad(90), 1, 0.1, 100)
}

func TestSphereScreenBoundsCanonical(t *testing.T) {
	t.Parallel()

	view, projection := canonicalSphereView()
	b, ok := SphereScreenBounds(Vec3{0, 0, -5}, 1, view, projection, 0, 0, 1000, 1000)
	if !ok {
		t.Fatalf("canonical sphere reported as not visible")
	}

	// The silhouette half-angle is asin(r/d) = asin(1/5). A tangent ray has
	// slope tan(half-angle) = 0.2/sqrt(1-0.04) = 0.204124..., which is the NDC
	// extent for a 90 degree FOV. Pixel edges: 500*(1 +/- 0.204124...).
	extent := float64(math.Tan(math.Asin(0.2)))
	wantMin := 500 * (1 - extent)
	wantMax := 500 * (1 + extent)

	if b.Min[0] > wantMin || b.Max[0] < wantMax || b.Min[1] > wantMin || b.Max[1] < wantMax {
		t.Errorf("bounds %v do not contain exact silhouette edge [%.4f, %.4f]", b, wantMin, wantMax)
	}

	// Outward rounding must stay within one pixel of the exact edge.
	if b.Min[0] < float64(math.Floor(float64(wantMin)))-1 || b.Max[0] > float64(math.Ceil(float64(wantMax)))+1 {
		t.Errorf("horizontal bounds [%.4f, %.4f] are not tight around [%.4f, %.4f]", b.Min[0], b.Max[0], wantMin, wantMax)
	}
	if b.Min[1] < float64(math.Floor(float64(wantMin)))-1 || b.Max[1] > float64(math.Ceil(float64(wantMax)))+1 {
		t.Errorf("vertical bounds [%.4f, %.4f] are not tight around [%.4f, %.4f]", b.Min[1], b.Max[1], wantMin, wantMax)
	}

	// Symmetric setup: horizontal and vertical bounds must agree and be centered.
	if b.Min[0] != b.Min[1] || b.Max[0] != b.Max[1] {
		t.Errorf("horizontal and vertical bounds differ for symmetric setup: %v", b)
	}
	if math.Abs(float64(b.Min[0]+b.Max[0]-1000)) > 1 {
		t.Errorf("bounds %v not centered on the 1000px viewport", b)
	}
}

// TestSphereScreenBoundsTighterThanLinear guards the whole point of the query:
// a linear center +/- world-radius projection overestimates near spheres, while
// the conic-based result must be tighter.
func TestSphereScreenBoundsTighterThanLinear(t *testing.T) {
	t.Parallel()

	view, projection := canonicalSphereView()
	b, ok := SphereScreenBounds(Vec3{0, 0, -5}, 1, view, projection, 0, 0, 1000, 1000)
	if !ok {
		t.Fatalf("sphere reported as not visible")
	}

	// Naive approach: projected center (500,500) +/- radius/d * viewport scale,
	// i.e. +/- 100 pixels -> [400, 600]. The correct box is about [398, 603],
	// which is strictly wider than the naive one (the naive box clips edges),
	// but far smaller than the full viewport.
	if b.Width() >= 1000 || b.Height() >= 1000 {
		t.Errorf("bounds %v collapse to the full viewport for a small far sphere", b)
	}
	if b.Min[0] > 400 || b.Max[0] < 600 {
		t.Errorf("bounds %v miss the naive linear extent [400,600]", b)
	}
}

func TestSphereScreenBoundsSpecialCases(t *testing.T) {
	t.Parallel()

	view, projection := canonicalSphereView()

	t.Run("camera inside sphere", func(t *testing.T) {
		b, ok := SphereScreenBounds(Vec3{0, 0, 0}, 5, view, projection, 0, 0, 1000, 1000)
		if !ok {
			t.Fatalf("sphere containing camera reported as not visible")
		}
		if b.Min != (Vec2{0, 0}) || b.Max != (Vec2{1000, 1000}) {
			t.Errorf("inside-sphere bounds = %v, want full viewport", b)
		}
	})

	t.Run("sphere crosses camera plane", func(t *testing.T) {
		// Center offset so the camera stays outside the sphere, yet the sphere
		// straddles the z = 0 camera plane: projection is unbounded.
		b, ok := SphereScreenBounds(Vec3{3, 0, -0.5}, 1, view, projection, 0, 0, 800, 600)
		if !ok {
			t.Fatalf("plane-straddling sphere reported as not visible")
		}
		if b.Min != (Vec2{0, 0}) || b.Max != (Vec2{800, 600}) {
			t.Errorf("plane-straddling bounds = %v, want full viewport", b)
		}
	})

	t.Run("sphere intersects near plane only", func(t *testing.T) {
		// Fully in front of the camera plane but crossing the z = -0.1 near
		// plane. The projection stays bounded and must be a tight, finite box.
		b, ok := SphereScreenBounds(Vec3{0, 0, -0.6}, 1, view, projection, 0, 0, 1000, 1000)
		if !ok {
			t.Fatalf("near-plane-intersecting sphere reported as not visible")
		}
		if !(b.Min[0] > -1e6 && b.Max[0] < 1e6) || b.Width() <= 0 || b.Height() <= 0 {
			t.Errorf("near-plane intersection gave degenerate bounds %v", b)
		}
	})

	t.Run("sphere fully behind camera", func(t *testing.T) {
		if _, ok := SphereScreenBounds(Vec3{0, 0, 5}, 1, view, projection, 0, 0, 1000, 1000); ok {
			t.Errorf("sphere behind camera reported as visible")
		}
		// Off-axis: even with |cz| < radius the whole sphere stays behind the
		// camera plane, so it must be culled rather than treated as unbounded.
		if _, ok := SphereScreenBounds(Vec3{8, 0, 1.5}, 1, view, projection, 0, 0, 1000, 1000); ok {
			t.Errorf("off-axis sphere fully behind camera reported as visible")
		}
		// Entirely off to the side, beyond the frustum.
		if _, ok := SphereScreenBounds(Vec3{30, 0, -5}, 1, view, projection, 0, 0, 1000, 1000); ok {
			t.Errorf("off-screen sphere reported as visible")
		}
	})

	t.Run("partially on screen is clipped", func(t *testing.T) {
		b, ok := SphereScreenBounds(Vec3{6, 0, -5}, 1, view, projection, 0, 0, 1000, 1000)
		if !ok {
			t.Fatalf("partially visible sphere reported as not visible")
		}
		if b.Max[0] != 1000 {
			t.Errorf("right edge = %v, want clipping at viewport edge 1000", b.Max[0])
		}
		if b.Min[0] <= 0 {
			t.Errorf("left edge = %v, want a positive inset", b.Min[0])
		}
	})

	t.Run("zero radius point", func(t *testing.T) {
		b, ok := SphereScreenBounds(Vec3{0, 0, -5}, 0, view, projection, 0, 0, 1000, 1000)
		if !ok {
			t.Fatalf("on-screen point reported as not visible")
		}
		if b.Min != (Vec2{500, 500}) || b.Max != (Vec2{500, 500}) {
			t.Errorf("point bounds = %v, want single pixel at center (500,500)", b)
		}
		if _, ok := SphereScreenBounds(Vec3{0, 0, 5}, 0, view, projection, 0, 0, 1000, 1000); ok {
			t.Errorf("point behind camera reported as visible")
		}
		if _, ok := SphereScreenBounds(Vec3{30, 0, -5}, 0, view, projection, 0, 0, 1000, 1000); ok {
			t.Errorf("off-screen point reported as visible")
		}
	})

	t.Run("invalid inputs", func(t *testing.T) {
		cases := []struct {
			name       string
			center     Vec3
			radius     float64
			view, proj Mat4
			w, h       int
		}{
			{"NaN center", Vec3{NaN, 0, -5}, 1, view, projection, 1000, 1000},
			{"inf radius", Vec3{0, 0, -5}, InfPos, view, projection, 1000, 1000},
			{"negative radius", Vec3{0, 0, -5}, -1, view, projection, 1000, 1000},
			{"zero width", Vec3{0, 0, -5}, 1, view, projection, 0, 1000},
			{"orthographic projection", Vec3{0, 0, -5}, 1, view, Ortho(-1, 1, -1, 1, 0.1, 100), 1000, 1000},
			{"off-axis frustum", Vec3{0, 0, -5}, 1, view, Frustum(0, 1, -1, 1, 0.1, 100), 1000, 1000},
			{"inf in view", Vec3{0, 0, -5}, 1, Mat4{InfPos, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}, projection, 1000, 1000},
		}
		for _, c := range cases {
			if _, ok := SphereScreenBounds(c.center, c.radius, c.view, c.proj, 0, 0, c.w, c.h); ok {
				t.Errorf("%s: expected not visible", c.name)
			}
		}
	})

	t.Run("viewport origin offset", func(t *testing.T) {
		b, ok := SphereScreenBounds(Vec3{0, 0, -5}, 1, view, projection, 100, 200, 1000, 1000)
		if !ok {
			t.Fatalf("offset-viewport sphere reported as not visible")
		}
		if b.Min[0] < 100 || b.Max[0] > 1100 || b.Min[1] < 200 || b.Max[1] > 1200 {
			t.Errorf("bounds %v outside offset viewport [100,1100]x[200,1200]", b)
		}
	})
}

// TestSphereScreenBoundsConservative brute-forces many random spheres and
// verifies that every projectable surface point is contained in the returned
// pixel rectangle, using the existing Project as an independent projection.
func TestSphereScreenBoundsConservative(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(1))
	f := func(x float64) float64 { return float64(x) }
	const samples = 400
	for iter := 0; iter < 120; iter++ {
		eye := Vec3{f(rng.NormFloat64() * 8), f(rng.NormFloat64() * 8), f(rng.NormFloat64()*4 + 10)}
		target := Vec3{f(rng.NormFloat64()), f(rng.NormFloat64()), f(rng.NormFloat64())}
		view := LookAtV(eye, target, Vec3{0, 1, 0})
		projection := Perspective(DegToRad(f(45+rng.Float64()*60)), f(0.5+rng.Float64()), 0.1, 100)

		center := Vec3{f(rng.NormFloat64() * 10), f(rng.NormFloat64() * 10), f(rng.NormFloat64()*6 - 5)}
		radius := f(0.1 + rng.Float64()*4)

		b, visible := SphereScreenBounds(center, radius, view, projection, 0, 0, 1024, 768)

		var onScreen int
		for s := 0; s < samples; s++ {
			// Uniform point on the sphere surface.
			z := 1 - 2*rng.Float64()
			phi := 2 * math.Pi * rng.Float64()
			rr := math.Sqrt(1 - z*z)
			p := center.Add(Vec3{
				radius * f(rr*math.Cos(phi)),
				radius * f(rr*math.Sin(phi)),
				radius * f(z),
			})

			eyePoint := view.Mul4x1(p.Vec4(1)).Vec3()
			if eyePoint[2] >= 0 {
				continue // behind the camera plane: not projectable
			}

			win := Project(p, view, projection, 0, 0, 1024, 768)
			if win[0] < 0 || win[0] > 1024 || win[1] < 0 || win[1] > 768 {
				continue // outside the frustum/viewport: imposes no bounds requirement
			}
			onScreen++

			if !visible {
				t.Fatalf("iter %d: sphere reported invisible but point %v projects on-screen to %v",
					iter, p, win)
			}
			// One pixel slack for the independent rounding done by Project.
			if win[0] < b.Min[0]-1 || win[0] > b.Max[0]+1 ||
				win[1] < b.Min[1]-1 || win[1] > b.Max[1]+1 {
				t.Fatalf("iter %d: on-screen point %v falls outside bounds %v", iter, win, b)
			}
		}

		// A culled sphere must have no on-screen surface points.
		if !visible && onScreen > 0 {
			t.Fatalf("iter %d: %d on-screen points but sphere culled", iter, onScreen)
		}
	}
}
