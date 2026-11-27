// Copyright 2026 The go-gl/mathgl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl32

import "testing"

// fixedTriangle builds the canonical test triangle: three screen positions
// whose centroid has screen barycentric weights of exactly one third each,
// clip-space w values 1, 2 and 4 respectively, UVs (0,0), (1,0) and (0,1) and
// scalar attributes 0, 1 and 2.
func fixedTriangle() (clips [3]Vec4, uvs [3]Vec2, attrs [3]float32, screens [3]Vec2) {
	clips = [3]Vec4{
		{0, 0, 0, 1},
		{0, 0, 0, 2},
		{0, 0, 0, 4},
	}
	uvs = [3]Vec2{{0, 0}, {1, 0}, {0, 1}}
	attrs = [3]float32{0, 1, 2}
	screens = [3]Vec2{{0, 0}, {6, 0}, {0, 12}}
	return
}

// TestInterpolateTrianglePerspectiveUV checks the canonical case: screen
// weights of one third each over a triangle with w values 1, 2 and 4.
// Perspective-correct UV at the centroid is (2/7, 1/7), never the naive
// screen-weighted (1/3, 1/3).
func TestInterpolateTrianglePerspectiveUV(t *testing.T) {
	t.Parallel()

	clips, uvs, attrs, screens := fixedTriangle()
	centroid := Vec2{2, 4} // (screens[0]+screens[1]+screens[2])/3

	interp, ok := InterpolateTriangle(
		clips[0], clips[1], clips[2],
		uvs[0], uvs[1], uvs[2],
		attrs[0], attrs[1], attrs[2],
		screens[0], screens[1], screens[2], centroid)
	if !ok {
		t.Fatal("expected valid interpolation, got ok == false")
	}

	wantWeights := Vec3{1. / 3, 1. / 3, 1. / 3}
	if !interp.Weights.ApproxEqualThreshold(wantWeights, 1e-6) {
		t.Errorf("weights = %v, want %v", interp.Weights, wantWeights)
	}

	wantUV := Vec2{2. / 7, 1. / 7}
	if !interp.UV.ApproxEqualThreshold(wantUV, 1e-6) {
		t.Errorf("perspective-correct UV = %v, want %v", interp.UV, wantUV)
	}

	naiveUV := Vec2{1. / 3, 1. / 3}
	if interp.UV.ApproxEqualThreshold(naiveUV, 1e-3) {
		t.Errorf("UV %v matches the naive screen interpolation %v; correction missing", interp.UV, naiveUV)
	}

	// attr = (l1*1/2 + l2*2/4) / (1/w) = (1/6 + 1/6) * (4/7) = 4/7.
	wantAttr := float32(4. / 7)
	if !FloatEqualThreshold(interp.Attr, wantAttr, 1e-6) {
		t.Errorf("attr = %v, want %v", interp.Attr, wantAttr)
	}
}

// TestInterpolateTriangleDepth tests that NDC depth follows the plain screen
// weights while the attributes remain perspective-corrected.
func TestInterpolateTriangleDepth(t *testing.T) {
	t.Parallel()

	clips, uvs, attrs, screens := fixedTriangle()
	// Give the vertices distinct NDC depths z/w.
	clips[0][2] = 0.3 * clips[0].W()
	clips[1][2] = 0.6 * clips[1].W()
	clips[2][2] = -0.9 * clips[2].W()

	centroid := Vec2{2, 4}
	interp, ok := InterpolateTriangle(
		clips[0], clips[1], clips[2],
		uvs[0], uvs[1], uvs[2],
		attrs[0], attrs[1], attrs[2],
		screens[0], screens[1], screens[2], centroid)
	if !ok {
		t.Fatal("expected valid interpolation, got ok == false")
	}

	wantDepth := float32((0.3 + 0.6 - 0.9) / 3) // screen-weighted NDC depth
	if !FloatEqualThreshold(interp.Depth, wantDepth, 1e-6) {
		t.Errorf("depth = %v, want %v", interp.Depth, wantDepth)
	}

	// Changing z must not move the perspective-corrected attribute results.
	wantUV := Vec2{2. / 7, 1. / 7}
	if !interp.UV.ApproxEqualThreshold(wantUV, 1e-6) {
		t.Errorf("UV = %v, want %v", interp.UV, wantUV)
	}
}

// TestInterpolateTriangleVerticesAndEdges checks points exactly on a vertex
// and on an edge: zero weights are allowed and the attributes reduce to the
// corresponding vertex or edge values.
func TestInterpolateTriangleVerticesAndEdges(t *testing.T) {
	t.Parallel()

	clips, uvs, attrs, screens := fixedTriangle()

	// On vertex 1.
	interp, ok := InterpolateTriangle(
		clips[0], clips[1], clips[2],
		uvs[0], uvs[1], uvs[2],
		attrs[0], attrs[1], attrs[2],
		screens[0], screens[1], screens[2], screens[1])
	if !ok {
		t.Fatal("vertex point rejected")
	}
	if !interp.Weights.ApproxEqualThreshold(Vec3{0, 1, 0}, 1e-6) {
		t.Errorf("vertex weights = %v, want {0 1 0}", interp.Weights)
	}
	if !interp.UV.ApproxEqualThreshold(uvs[1], 1e-6) {
		t.Errorf("vertex UV = %v, want %v", interp.UV, uvs[1])
	}
	if !FloatEqualThreshold(interp.Attr, attrs[1], 1e-6) {
		t.Errorf("vertex attr = %v, want %v", interp.Attr, attrs[1])
	}

	// Midpoint of edge 0-2 on screen: screen weights (0.5, 0, 0.5).
	edgeMid := screens[0].Add(screens[2]).Mul(0.5)
	interp, ok = InterpolateTriangle(
		clips[0], clips[1], clips[2],
		uvs[0], uvs[1], uvs[2],
		attrs[0], attrs[1], attrs[2],
		screens[0], screens[1], screens[2], edgeMid)
	if !ok {
		t.Fatal("edge point rejected")
	}
	if !interp.Weights.ApproxEqualThreshold(Vec3{0.5, 0, 0.5}, 1e-6) {
		t.Errorf("edge weights = %v, want {0.5 0 0.5}", interp.Weights)
	}

	// Hand-checked perspective-correct edge values: w0=1, w2=4, l0=l2=0.5.
	invW := 0.5/1 + 0.5/4
	wantUV := Vec2{0, float32((0.5 * 1 / 4) / invW)}
	wantAttr := float32((0.5 * 2 / 4) / invW)
	if !interp.UV.ApproxEqualThreshold(wantUV, 1e-6) {
		t.Errorf("edge UV = %v, want %v", interp.UV, wantUV)
	}
	if !FloatEqualThreshold(interp.Attr, wantAttr, 1e-6) {
		t.Errorf("edge attr = %v, want %v", interp.Attr, wantAttr)
	}
}

// TestBarycentricReversedWinding checks that flipping the winding order flips
// the signed area but leaves the weights unchanged, and that weights sum to
// one and reproduce the input point.
func TestBarycentricReversedWinding(t *testing.T) {
	t.Parallel()

	a, b, c := Vec2{0, 0}, Vec2{6, 0}, Vec2{0, 12}
	p := Vec2{1.5, 3}

	l0, l1, l2, ok := Barycentric(p, a, b, c)
	if !ok {
		t.Fatal("forward winding reported degenerate")
	}
	r0, r1, r2, ok := Barycentric(p, a, c, b)
	if !ok {
		t.Fatal("reversed winding reported degenerate")
	}
	got := Vec3{l0, l1, l2}
	reversed := Vec3{r0, r2, r1} // weights follow vertex order
	if !got.ApproxEqualThreshold(reversed, 1e-6) {
		t.Errorf("reversed winding weights %v != %v", reversed, got)
	}
	if !FloatEqualThreshold(l0+l1+l2, 1, 1e-6) {
		t.Errorf("weights sum to %v, want 1", l0+l1+l2)
	}

	reconstructed := a.Mul(l0).Add(b.Mul(l1)).Add(c.Mul(l2))
	if !reconstructed.ApproxEqualThreshold(p, 1e-6) {
		t.Errorf("weights reconstruct %v, want %v", reconstructed, p)
	}
}

// TestBarycentricOutsideAndDegenerate checks extrapolation outside the
// triangle and rejection of degenerate (collinear/coincident) triangles.
func TestBarycentricOutsideAndDegenerate(t *testing.T) {
	t.Parallel()

	a, b, c := Vec2{0, 0}, Vec2{6, 0}, Vec2{0, 12}

	// Point outside: a negative weight is expected, but interpolation is
	// still well defined.
	outside := Vec2{9, 9}
	l0, l1, l2, ok := Barycentric(outside, a, b, c)
	if !ok {
		t.Fatal("outside point reported degenerate")
	}
	if l0 >= 0 && l1 >= 0 && l2 >= 0 {
		t.Errorf("outside point gave all non-negative weights %v %v %v", l0, l1, l2)
	}
	reconstructed := a.Mul(l0).Add(b.Mul(l1)).Add(c.Mul(l2))
	if !reconstructed.ApproxEqualThreshold(outside, 1e-6) {
		t.Errorf("outside weights reconstruct %v, want %v", reconstructed, outside)
	}

	// Collinear vertices: zero area.
	if _, _, _, ok := Barycentric(Vec2{1, 0}, a, b, Vec2{3, 0}); ok {
		t.Error("collinear triangle not reported degenerate")
	}
	// Coincident vertices.
	if _, _, _, ok := Barycentric(a, a, b, b); ok {
		t.Error("coincident-vertex triangle not reported degenerate")
	}
}

// TestInterpolateTriangleDegenerate checks that screen-space degenerate
// triangles are rejected even with healthy clip w values.
func TestInterpolateTriangleDegenerate(t *testing.T) {
	t.Parallel()

	clips, uvs, attrs, _ := fixedTriangle()
	screens := [3]Vec2{{0, 0}, {4, 0}, {8, 0}} // collinear

	if _, ok := InterpolateTriangle(
		clips[0], clips[1], clips[2],
		uvs[0], uvs[1], uvs[2],
		attrs[0], attrs[1], attrs[2],
		screens[0], screens[1], screens[2], Vec2{4, 0}); ok {
		t.Error("degenerate triangle was not rejected")
	}
}

// TestInterpolateTriangleInvalidW exercises the invalid w cases: zero,
// negative, near-zero relative to the other vertices, NaN and infinity.
func TestInterpolateTriangleInvalidW(t *testing.T) {
	t.Parallel()

	clips, uvs, attrs, screens := fixedTriangle()
	p := Vec2{2, 4}

	cases := map[string]Vec4{
		"zero w":      {0, 0, 0, 0},
		"negative w":  {0, 0, 0, -2},
		"near-zero w": {0, 0, 0, Epsilon / 2},
		"NaN w":       {0, 0, 0, NaN},
		"+Inf w":      {0, 0, 0, InfPos},
		"-Inf w":      {0, 0, 0, InfNeg},
	}
	for name, bad := range cases {
		clipsBad := clips
		clipsBad[1] = bad
		if _, ok := InterpolateTriangle(
			clipsBad[0], clipsBad[1], clipsBad[2],
			uvs[0], uvs[1], uvs[2],
			attrs[0], attrs[1], attrs[2],
			screens[0], screens[1], screens[2], p); ok {
			t.Errorf("case %q: expected rejection, got ok == true", name)
		}
	}
}

// TestInterpolateTriangleUniformW is the orthographic sanity check: with equal
// w the perspective-correct result must equal plain screen interpolation.
func TestInterpolateTriangleUniformW(t *testing.T) {
	t.Parallel()

	clips := [3]Vec4{{0, 0, 0, 3}, {0, 0, 0, 3}, {0, 0, 0, 3}}
	uvs := [3]Vec2{{0, 0}, {1, 0}, {0, 1}}
	attrs := [3]float32{0, 1, 2}
	screens := [3]Vec2{{0, 0}, {6, 0}, {0, 12}}
	p := Vec2{2, 4} // screen weights 1/3 each

	interp, ok := InterpolateTriangle(
		clips[0], clips[1], clips[2],
		uvs[0], uvs[1], uvs[2],
		attrs[0], attrs[1], attrs[2],
		screens[0], screens[1], screens[2], p)
	if !ok {
		t.Fatal("uniform-w triangle rejected")
	}
	wantUV := Vec2{1. / 3, 1. / 3}
	wantAttr := float32(1)
	if !interp.UV.ApproxEqualThreshold(wantUV, 1e-6) {
		t.Errorf("uniform-w UV = %v, want %v", interp.UV, wantUV)
	}
	if !FloatEqualThreshold(interp.Attr, wantAttr, 1e-6) {
		t.Errorf("uniform-w attr = %v, want %v", interp.Attr, wantAttr)
	}
}
