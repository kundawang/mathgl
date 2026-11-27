// Copyright 2026 The go-gl/mathgl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl32

// isFinite reports whether f is neither NaN nor an infinity. NaN is the only
// value unequal to itself, and every finite value has a magnitude below the
// package's maximum finite float, so no math-package conversions are needed
// and the generated mgl64 variant stays identical in shape.
func isFinite(f float32) bool {
	return f == f && -MaxValue < f && f < MaxValue
}

// Barycentric returns the barycentric weights of the point p with respect to
// the triangle (a, b, c) given in screen/window coordinates.
//
// The returned weights l0, l1, l2 satisfy
//
//	l0 + l1 + l2 = 1
//	p = l0*a + l1*b + l2*c
//
// A point strictly inside the triangle has three positive weights; a point on
// an edge or exactly on a vertex has one or two zero weights. Points outside
// the triangle are still valid and produce one or more negative weights, so
// callers that require containment must check that all weights are
// non-negative themselves. The winding order of the triangle does not matter.
//
// ok is false when the triangle is degenerate, i.e. the three screen positions
// are collinear (or coincident) and span no area; in that case the weights are
// meaningless and are returned as zero.
func Barycentric(p, a, b, c Vec2) (l0, l1, l2 float32, ok bool) {
	v0 := b.Sub(a)
	v1 := c.Sub(a)
	v2 := p.Sub(a)

	// Twice the signed area of the triangle. Its sign carries the winding
	// order, which cancels out in the weight ratios below.
	denom := v0.X()*v1.Y() - v1.X()*v0.Y()
	if Abs(denom) <= Epsilon {
		return 0, 0, 0, false
	}

	inv := 1 / denom
	l1 = (v2.X()*v1.Y() - v1.X()*v2.Y()) * inv
	l2 = (v0.X()*v2.Y() - v2.X()*v0.Y()) * inv
	l0 = 1 - l1 - l2

	return l0, l1, l2, true
}

// TriangleInterpolation holds the result of interpolating vertex attributes at
// a screen position inside a triangle.
type TriangleInterpolation struct {
	// Weights are the screen-space barycentric weights (summing to 1) with
	// respect to the three vertices, in the original vertex order.
	Weights Vec3

	// UV is the perspective-correct interpolation of the per-vertex texture
	// coordinates.
	UV Vec2

	// Attr is the perspective-correct interpolation of the per-vertex scalar
	// attributes.
	Attr float32

	// Depth is the NDC depth (clip-space z divided by clip-space w)
	// interpolated directly with the screen-space barycentric weights, as a
	// standard hardware depth buffer stores it. This is deliberately not
	// perspective-corrected.
	Depth float32
}

// InterpolateTriangle performs perspective-correct interpolation of the
// attributes of a triangle at the screen position p.
//
// clip0, clip1 and clip2 are the three vertices in clip space; their W
// component is the perspective divide factor. uv0, uv1, uv2 and attr0, attr1,
// attr2 are the corresponding texture coordinates and scalar attributes.
// screen0, screen1 and screen2 are the positions the vertices project to on
// screen (for example the xy components returned by Project). The original
// vertex order is preserved everywhere, including in the returned weights.
//
// The screen-space barycentric weights are computed first. The attributes are
// then corrected for perspective, so that for example a checkerboard texture
// stays straight instead of bending across the triangle:
//
//	1/w   = l0/w0 + l1/w1 + l2/w2
//	uv    = (l0*uv0/w0 + l1*uv1/w1 + l2*uv2/w2) * w
//	attr  = (l0*a0/w0  + l1*a1/w1  + l2*a2/w2 ) * w
//
// Depth, by contrast, is the NDC depth interpolated by the plain screen-space
// weights (l0*z0/w0 + l1*z1/w1 + l2*z2/w2), matching depth-buffer behaviour.
//
// ok is false, and the returned value is the zero TriangleInterpolation, when
// the triangle is degenerate on screen, when any clip-space w is not a finite
// positive number, or when any w is effectively zero (relative to the largest
// of the three, within Epsilon). Points outside the triangle are otherwise
// accepted and produce extrapolated (negative-weight) results.
func InterpolateTriangle(clip0, clip1, clip2 Vec4, uv0, uv1, uv2 Vec2,
	attr0, attr1, attr2 float32, screen0, screen1, screen2 Vec2, p Vec2) (interp TriangleInterpolation, ok bool) {

	l0, l1, l2, baryOK := Barycentric(p, screen0, screen1, screen2)
	if !baryOK {
		return TriangleInterpolation{}, false
	}

	w0, w1, w2 := clip0.W(), clip1.W(), clip2.W()

	// Reject non-finite or non-positive w first: NaN would silently slip
	// through ordinary ordered comparisons.
	for _, w := range [3]float32{w0, w1, w2} {
		if !isFinite(w) || w <= 0 {
			return TriangleInterpolation{}, false
		}
	}

	// Reject w effectively zero relative to the triangle's largest w, so the
	// check is independent of the triangle's overall scale.
	wMax := w0
	if w1 > wMax {
		wMax = w1
	}
	if w2 > wMax {
		wMax = w2
	}
	if w0 <= Epsilon*wMax || w1 <= Epsilon*wMax || w2 <= Epsilon*wMax {
		return TriangleInterpolation{}, false
	}

	invW0, invW1, invW2 := 1/w0, 1/w1, 1/w2

	// Perspective-correct reciprocal depth at p.
	invW := l0*invW0 + l1*invW1 + l2*invW2
	w := 1 / invW

	uv := Vec2{
		(l0*uv0.X()*invW0 + l1*uv1.X()*invW1 + l2*uv2.X()*invW2) * w,
		(l0*uv0.Y()*invW0 + l1*uv1.Y()*invW1 + l2*uv2.Y()*invW2) * w,
	}
	attr := (l0*attr0*invW0 + l1*attr1*invW1 + l2*attr2*invW2) * w

	// NDC depth is linear in screen space: interpolate z/w with the plain
	// barycentric weights.
	depth := l0*(clip0.Z()*invW0) + l1*(clip1.Z()*invW1) + l2*(clip2.Z()*invW2)

	return TriangleInterpolation{
		Weights: Vec3{l0, l1, l2},
		UV:      uv,
		Attr:    attr,
		Depth:   depth,
	}, true
}
