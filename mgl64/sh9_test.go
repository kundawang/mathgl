// This file is generated from mgl32/sh9_test.go; DO NOT EDIT

// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"math"
	"testing"
)

// The reference basis below is computed independently from the definitions in
// float64, so the tests do not simply mirror the implementation constants.
func sh9RefBasis(x, y, z float64) [9]float64 {
	y00 := 1 / (2 * math.Sqrt(math.Pi))
	k1 := math.Sqrt(3 / (4 * math.Pi))
	k2 := math.Sqrt(15 / (4 * math.Pi))
	k3 := math.Sqrt(5 / (16 * math.Pi))
	k4 := math.Sqrt(15 / (16 * math.Pi))
	return [9]float64{
		y00,
		k1 * y,
		k1 * z,
		k1 * x,
		k2 * x * y,
		k2 * y * z,
		k3 * (3*z*z - 1),
		k2 * x * z,
		k4 * (x*x - y*y),
	}
}

// sh9Vec converts float64 coordinates to the package scalar type. In mgl64 this
// file is regenerated with float32 rewritten to float64, making the conversion
// a no-op there.
func sh9Vec(x, y, z float64) Vec3 {
	return Vec3{float64(x), float64(y), float64(z)}
}

func sh9UnitRef(v Vec3) (x, y, z float64) {
	x, y, z = float64(v[0]), float64(v[1]), float64(v[2])
	l := math.Sqrt(x*x + y*y + z*z)
	return x / l, y / l, z / l
}

func sh9CheckVec(t *testing.T, what string, got Vec3, want [3]float64, tol float64) {
	t.Helper()
	for c := 0; c < 3; c++ {
		g := float64(got[c])
		if math.Abs(g-want[c]) > tol {
			t.Errorf("%s: channel %d = %v, want %v (tolerance %v)", what, c, g, want[c], tol)
		}
	}
}

// TestSH9BasisValues checks the closed-form value of every basis function on
// the coordinate axes, and that non-unit directions are normalized.
func TestSH9BasisValues(t *testing.T) {
	t.Parallel()

	y00 := 1 / (2 * math.Sqrt(math.Pi))
	k1 := math.Sqrt(3 / (4 * math.Pi))
	k3 := math.Sqrt(5 / (16 * math.Pi))
	k4 := math.Sqrt(15 / (16 * math.Pi))

	zAxis := SH9Basis(Vec3{0, 0, 1})
	zWant := [9]float64{y00, 0, k1, 0, 0, 0, 2 * k3, 0, 0}
	for i := range zWant {
		if d := math.Abs(float64(zAxis[i]) - zWant[i]); d > 1e-6 {
			t.Errorf("basis at +z: index %d = %v, want %v", i, zAxis[i], zWant[i])
		}
	}

	xAxis := SH9Basis(Vec3{1, 0, 0})
	xWant := [9]float64{y00, 0, 0, k1, 0, 0, -k3, 0, k4}
	for i := range xWant {
		if d := math.Abs(float64(xAxis[i]) - xWant[i]); d > 1e-6 {
			t.Errorf("basis at +x: index %d = %v, want %v", i, xAxis[i], xWant[i])
		}
	}

	// Scaling the direction must not change the basis.
	scaled := SH9Basis(Vec3{0, 0, 7})
	for i := range zAxis {
		if !FloatEqual(zAxis[i], scaled[i]) {
			t.Errorf("basis should be invariant under direction scaling, index %d: %v vs %v", i, zAxis[i], scaled[i])
		}
	}

	// A degenerate direction yields the zero basis rather than NaN.
	zero := SH9Basis(Vec3{0, 0, 0})
	for i := range zero {
		if zero[i] != 0 {
			t.Errorf("basis at zero direction should be zero, index %d = %v", i, zero[i])
		}
	}

	// Very large and very small finite magnitudes must normalize without
	// overflowing or underflowing to zero.
	for _, d := range []Vec3{
		{MaxValue, MaxValue, 0},
		{MinNormal, 0, MinNormal},
		{1e-30, 0, 2e-30},
	} {
		b := SH9Basis(d)
		if b[0] != float64(shY00) {
			t.Errorf("basis at extreme direction %v: index 0 = %v, want %v", d, b[0], float64(shY00))
		}
	}
}

// TestSH9Orthonormality numerically integrates products of the basis functions
// over the sphere and checks that they form an orthonormal set. This is the
// normalization convention used by ProjectSH9.
func TestSH9Orthonormality(t *testing.T) {
	t.Parallel()

	const nz = 200
	const nphi = 400
	dz := 2.0 / nz
	dphi := 2 * math.Pi / nphi
	dw := dz * dphi

	var gram [9][9]float64
	for iz := 0; iz < nz; iz++ {
		z := -1 + (float64(iz)+0.5)*dz
		r := math.Sqrt(1 - z*z)
		for ip := 0; ip < nphi; ip++ {
			phi := (float64(ip) + 0.5) * dphi
			b := SH9Basis(sh9Vec(r*math.Cos(phi), r*math.Sin(phi), z))
			for i := 0; i < 9; i++ {
				bi := float64(b[i])
				for j := 0; j < 9; j++ {
					gram[i][j] += dw * bi * float64(b[j])
				}
			}
		}
	}

	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			want := 0.0
			if i == j {
				want = 1
			}
			if math.Abs(gram[i][j]-want) > 2e-3 {
				t.Errorf("integral of Y%d*Y%d = %v, want %v", i, j, gram[i][j], want)
			}
		}
	}
}

// TestProjectSH9Constant uses the six axis directions of an octahedron, which
// form a spherical 3-design, so a constant environment projects exactly onto
// the l=0 band with every other coefficient zero.
func TestProjectSH9Constant(t *testing.T) {
	t.Parallel()

	radiance := [3]float64{0.5, 1.0, 1.5}
	samples := make([]SHSample, 0, 6)
	dirs := []Vec3{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	for _, d := range dirs {
		samples = append(samples, SHSample{
			Dir:      d,
			Weight:   float64(4 * math.Pi / 6),
			Radiance: sh9Vec(radiance[0], radiance[1], radiance[2]),
		})
	}

	sh, err := ProjectSH9(samples)
	if err != nil {
		t.Fatalf("ProjectSH9 returned error: %v", err)
	}

	for i := 0; i < 9; i++ {
		want := [3]float64{0, 0, 0}
		if i == 0 {
			for c := 0; c < 3; c++ {
				want[c] = math.Sqrt(4*math.Pi) * radiance[c]
			}
		}
		sh9CheckVec(t, "constant environment coefficient", sh[i], want, 1e-4)
	}

	// Evaluate must reproduce the constant radiance, and irradiance must be
	// pi times it for every normal.
	normals := []Vec3{{0, 0, 1}, {1, 0, 0}, {0, -1, 0}, {0.3, -0.7, 0.2}}
	for _, n := range normals {
		sh9CheckVec(t, "constant environment Evaluate", sh.Evaluate(n), radiance, 1e-5)
		e := sh.Irradiance(n)
		want := [3]float64{math.Pi * radiance[0], math.Pi * radiance[1], math.Pi * radiance[2]}
		sh9CheckVec(t, "constant environment Irradiance", e, want, 1e-4)
	}
}

// TestProjectSH9SingleDirection checks the projection of a single directional
// sample against the closed form, and checks the l=0, l=1 and l=2 response of
// Irradiance at normals parallel, anti-parallel and perpendicular to it.
func TestProjectSH9SingleDirection(t *testing.T) {
	t.Parallel()

	dir := sh9Vec(1, 2, 3)
	x, y, z := sh9UnitRef(dir)
	weight := 0.5
	radiance := [3]float64{2, 1, 0.5}

	sh, err := ProjectSH9([]SHSample{{
		Dir:      dir,
		Weight:   float64(weight),
		Radiance: sh9Vec(radiance[0], radiance[1], radiance[2]),
	}})
	if err != nil {
		t.Fatalf("ProjectSH9 returned error: %v", err)
	}

	basis := sh9RefBasis(x, y, z)
	for i := 0; i < 9; i++ {
		want := [3]float64{
			weight * radiance[0] * basis[i],
			weight * radiance[1] * basis[i],
			weight * radiance[2] * basis[i],
		}
		sh9CheckVec(t, "single-direction coefficient", sh[i], want, 1e-5)
	}

	// For a single sample of weight w and radiance L, the truncated irradiance
	// is w*L*(1/4 + c/2 + 5*(3c^2-1)/32) with c = n.d.
	cosToWant := func(c float64) [3]float64 {
		f := 0.25 + c/2 + 5*(3*c*c-1)/32
		return [3]float64{weight * radiance[0] * f, weight * radiance[1] * f, weight * radiance[2] * f}
	}

	nParallel := dir
	sh9CheckVec(t, "irradiance parallel", sh.Irradiance(nParallel), cosToWant(1), 1e-5)
	sh9CheckVec(t, "irradiance anti-parallel", sh.Irradiance(sh9Vec(-x, -y, -z)), cosToWant(-1), 1e-5)

	// Build a unit normal exactly perpendicular to dir and check c = 0.
	q := Vec3{2, -1, 0}
	qx, qy, qz := sh9UnitRef(q)
	d := x*qx + y*qy + z*qz
	perpUnit := sh9Vec(qx-x*d, qy-y*d, qz-z*d)
	ppx, ppy, ppz := sh9UnitRef(perpUnit)
	if math.Abs(x*ppx+y*ppy+z*ppz) > 1e-5 {
		t.Fatalf("test setup: perpendicular normal is not perpendicular")
	}
	sh9CheckVec(t, "irradiance perpendicular", sh.Irradiance(perpUnit), cosToWant(0), 1e-5)
}

// TestProjectSH9Axisymmetric projects a fixed axisymmetric environment built
// from three-point Gauss-Legendre rings. The quadrature integrates the z
// polynomial exactly, so every coefficient has a closed form. The environment
// L(z) = a + b*z + c*z^2 populates the l=0, l=1 and l=2 bands.
func TestProjectSH9Axisymmetric(t *testing.T) {
	t.Parallel()

	// Three-point Gauss-Legendre nodes and weights on [-1, 1]. With 4 samples
	// per ring, the phi integration is exact for the m=0 modes and cancels the
	// m=1,2 modes.
	zn := [3]float64{-math.Sqrt(3.0 / 5.0), 0, math.Sqrt(3.0 / 5.0)}
	gw := [3]float64{5.0 / 9.0, 8.0 / 9.0, 5.0 / 9.0}
	const nphi = 4
	dphi := 2 * math.Pi / nphi

	a := [3]float64{0.7, 1.1, 0.2}
	b := [3]float64{0.3, -0.4, 0.9}
	c := [3]float64{0.2, 0.5, 0.3}

	samples := make([]SHSample, 0, len(zn)*nphi)
	for k, z := range zn {
		r := math.Sqrt(1 - z*z)
		w := gw[k] * dphi
		for p := 0; p < nphi; p++ {
			phi := float64(p) * dphi
			l := [3]float64{
				a[0] + b[0]*z + c[0]*z*z,
				a[1] + b[1]*z + c[1]*z*z,
				a[2] + b[2]*z + c[2]*z*z,
			}
			samples = append(samples, SHSample{
				Dir:      sh9Vec(r*math.Cos(phi), r*math.Sin(phi), z),
				Weight:   float64(w),
				Radiance: sh9Vec(l[0], l[1], l[2]),
			})
		}
	}

	sh, err := ProjectSH9(samples)
	if err != nil {
		t.Fatalf("ProjectSH9 returned error: %v", err)
	}

	y00 := 1 / (2 * math.Sqrt(math.Pi))
	k1 := math.Sqrt(3 / (4 * math.Pi))
	k3 := math.Sqrt(5 / (16 * math.Pi))

	// Closed forms of the projection of a + b*z + c*z^2.
	want0 := [3]float64{}
	want2 := [3]float64{}
	want6 := [3]float64{}
	for ch := 0; ch < 3; ch++ {
		want0[ch] = 4 * math.Pi * y00 * (a[ch] + c[ch]/3)
		want2[ch] = k1 * b[ch] * 4 * math.Pi / 3
		want6[ch] = c[ch] / (3 * k3)
	}

	sh9CheckVec(t, "axisymmetric Y00", sh[0], want0, 1e-4)
	sh9CheckVec(t, "axisymmetric Y10", sh[2], want2, 1e-4)
	sh9CheckVec(t, "axisymmetric Y20", sh[6], want6, 1e-4)
	for _, i := range []int{1, 3, 4, 5, 7, 8} {
		sh9CheckVec(t, "axisymmetric odd/azimuthal coefficient", sh[i], [3]float64{0, 0, 0}, 1e-5)
	}

	// Normal response: only the m=0 bands contribute, so
	// E(n) = pi*c0*Y00 + (2*pi/3)*c2*Y10(n) + (pi/4)*c6*Y20(n).
	evalAt := func(n Vec3, nz float64) [3]float64 {
		out := [3]float64{}
		b10 := k1 * nz
		b20 := k3 * (3*nz*nz - 1)
		for ch := 0; ch < 3; ch++ {
			out[ch] = math.Pi*want0[ch]*y00 +
				(2*math.Pi/3)*want2[ch]*b10 +
				(math.Pi/4)*want6[ch]*b20
		}
		return out
	}

	sh9CheckVec(t, "axisymmetric irradiance +z", sh.Irradiance(Vec3{0, 0, 1}), evalAt(Vec3{0, 0, 1}, 1), 1e-4)
	sh9CheckVec(t, "axisymmetric irradiance -z", sh.Irradiance(Vec3{0, 0, -1}), evalAt(Vec3{0, 0, -1}, -1), 1e-4)
	sh9CheckVec(t, "axisymmetric irradiance +x", sh.Irradiance(Vec3{1, 0, 0}), evalAt(Vec3{1, 0, 0}, 0), 1e-4)
}

// TestProjectSH9LinearEnvironment projects a non-axisymmetric linear
// environment sampled on a uniform sphere. A linear environment lies entirely
// in the l<=1 bands, so the truncated irradiance is exact:
//
//	E(n) = pi*const + (2*pi/3)*(gradient . n)
//
// This is an independent check of the l=0 and l=1 eigenvalues.
func TestProjectSH9LinearEnvironment(t *testing.T) {
	t.Parallel()

	const (
		nz   = 200
		nphi = 400
		base = 0.5
	)
	grad := [3]float64{0.3, -0.2, 0.1}
	scale := [3]float64{1, 0.5, 2} // per-channel scale

	dz := 2.0 / nz
	dphi := 2 * math.Pi / nphi
	dw := dz * dphi

	samples := make([]SHSample, 0, nz*nphi)
	for iz := 0; iz < nz; iz++ {
		z := -1 + (float64(iz)+0.5)*dz
		r := math.Sqrt(1 - z*z)
		for ip := 0; ip < nphi; ip++ {
			phi := (float64(ip) + 0.5) * dphi
			x, y := r*math.Cos(phi), r*math.Sin(phi)
			l := base + grad[0]*x + grad[1]*y + grad[2]*z
			samples = append(samples, SHSample{
				Dir:      sh9Vec(x, y, z),
				Weight:   float64(dw),
				Radiance: sh9Vec(scale[0]*l, scale[1]*l, scale[2]*l),
			})
		}
	}

	sh, err := ProjectSH9(samples)
	if err != nil {
		t.Fatalf("ProjectSH9 returned error: %v", err)
	}

	normals := []Vec3{{0, 0, 1}, {1, 0, 0}, {0, 1, 0}, {1, 1, 1}, {-2, 1, 0.5}}
	for _, n := range normals {
		nx, ny, nzz := sh9UnitRef(n)
		wantIrr := [3]float64{}
		wantEval := [3]float64{}
		for ch := 0; ch < 3; ch++ {
			wantIrr[ch] = math.Pi*base*scale[ch] + (2*math.Pi/3)*scale[ch]*(grad[0]*nx+grad[1]*ny+grad[2]*nzz)
			// Evaluate must also reproduce the field itself (l<=2 is exact).
			wantEval[ch] = scale[ch] * (base + grad[0]*nx + grad[1]*ny + grad[2]*nzz)
		}
		sh9CheckVec(t, "linear environment irradiance", sh.Irradiance(n), wantIrr, 2e-3)
		sh9CheckVec(t, "linear environment Evaluate", sh.Evaluate(n), wantEval, 2e-3)
	}
}

// TestProjectSH9SampleOrder checks that permuting the samples only changes
// floating-point rounding.
func TestProjectSH9SampleOrder(t *testing.T) {
	t.Parallel()

	const n = 24
	samples := make([]SHSample, n)
	for i := 0; i < n; i++ {
		// Deterministic, varied directions.
		theta := math.Acos(1 - 2*(float64(i)+0.5)/n)
		phi := float64(i) * 2.399963229728653
		d := sh9Vec(math.Sin(theta)*math.Cos(phi), math.Sin(theta)*math.Sin(phi), math.Cos(theta))
		samples[i] = SHSample{
			Dir:      d,
			Weight:   float64(4 * math.Pi / n),
			Radiance: sh9Vec(0.2+0.1*float64(i%3), 0.4+0.05*float64(i%5), 0.1+0.03*float64(i)),
		}
	}
	sh, err := ProjectSH9(samples)
	if err != nil {
		t.Fatalf("ProjectSH9 returned error: %v", err)
	}

	reversed := make([]SHSample, n)
	for i := range samples {
		reversed[i] = samples[n-1-i]
	}
	shRev, err := ProjectSH9(reversed)
	if err != nil {
		t.Fatalf("ProjectSH9(reversed) returned error: %v", err)
	}

	for i := 0; i < 9; i++ {
		sh9CheckVec(t, "coefficient independent of sample order", shRev[i],
			[3]float64{float64(sh[i][0]), float64(sh[i][1]), float64(sh[i][2])}, 1e-4)
	}

	// Evaluate and irradiance must agree as well.
	for _, nrm := range []Vec3{{0, 0, 1}, {1, 0, 0}, {0.4, 0.5, -0.7}} {
		sh9CheckVec(t, "Evaluate independent of sample order", shRev.Evaluate(nrm),
			[3]float64{float64(sh.Evaluate(nrm)[0]), float64(sh.Evaluate(nrm)[1]), float64(sh.Evaluate(nrm)[2])}, 1e-4)
		sh9CheckVec(t, "Irradiance independent of sample order", shRev.Irradiance(nrm),
			[3]float64{float64(sh.Irradiance(nrm)[0]), float64(sh.Irradiance(nrm)[1]), float64(sh.Irradiance(nrm)[2])}, 1e-4)
	}
}

// TestSH9Rotation checks that rotating the environment rotates the
// reconstructed radiance and irradiance with it.
func TestSH9Rotation(t *testing.T) {
	t.Parallel()

	const n = 16
	quat := QuatRotate(0.7, Vec3{1, 2, -1}.Normalize()).Normalize()

	samples := make([]SHSample, n)
	rotated := make([]SHSample, n)
	for i := 0; i < n; i++ {
		theta := math.Acos(1 - 2*(float64(i)+0.5)/n)
		phi := float64(i) * 2.399963229728653
		d := sh9Vec(math.Sin(theta)*math.Cos(phi), math.Sin(theta)*math.Sin(phi), math.Cos(theta))
		samples[i] = SHSample{
			Dir:      d,
			Weight:   float64(4 * math.Pi / n),
			Radiance: sh9Vec(0.2+0.1*float64(i%3), 0.4+0.05*float64(i%5), 0.1+0.03*float64(i)),
		}
		rotated[i] = samples[i]
		rotated[i].Dir = quat.Rotate(d)
	}

	sh, err := ProjectSH9(samples)
	if err != nil {
		t.Fatalf("ProjectSH9 returned error: %v", err)
	}
	shRot, err := ProjectSH9(rotated)
	if err != nil {
		t.Fatalf("ProjectSH9(rotated) returned error: %v", err)
	}

	normals := []Vec3{{0, 0, 1}, {1, 0, 0}, {0, 1, 0}, {0.3, -0.4, 0.8}, {-1, 2, 0.5}}
	for _, nrm := range normals {
		wantEval := sh.Evaluate(nrm)
		gotEval := shRot.Evaluate(quat.Rotate(nrm))
		sh9CheckVec(t, "rotation-invariant Evaluate", gotEval,
			[3]float64{float64(wantEval[0]), float64(wantEval[1]), float64(wantEval[2])}, 5e-4)

		wantIrr := sh.Irradiance(nrm)
		gotIrr := shRot.Irradiance(quat.Rotate(nrm))
		sh9CheckVec(t, "rotation-invariant Irradiance", gotIrr,
			[3]float64{float64(wantIrr[0]), float64(wantIrr[1]), float64(wantIrr[2])}, 5e-4)
	}
}

// TestSH9DegenerateDirection checks that degenerate normals and directions
// return zero instead of NaN.
func TestSH9DegenerateDirection(t *testing.T) {
	t.Parallel()

	var sh SH9
	for _, n := range []Vec3{{0, 0, 0}, {float64(math.Inf(1)), 0, 0}, {float64(math.NaN()), 0, 0}} {
		ev := sh.Evaluate(n)
		if ev != (Vec3{}) {
			t.Errorf("Evaluate(%v) = %v, want zero", n, ev)
		}
		irr := sh.Irradiance(n)
		if irr != (Vec3{}) {
			t.Errorf("Irradiance(%v) = %v, want zero", n, irr)
		}
	}
}

// TestProjectSH9Errors checks that invalid samples and zero total weight are
// rejected with an error.
func TestProjectSH9Errors(t *testing.T) {
	t.Parallel()

	valid := SHSample{Dir: Vec3{0, 0, 1}, Weight: 1, Radiance: Vec3{1, 1, 1}}
	withDir := func(d Vec3) SHSample { s := valid; s.Dir = d; return s }
	withWeight := func(w float64) SHSample { s := valid; s.Weight = w; return s }
	withRadiance := func(r Vec3) SHSample { s := valid; s.Radiance = r; return s }

	cases := []struct {
		name    string
		samples []SHSample
	}{
		{"empty", nil},
		{"all zero weights", []SHSample{withWeight(0), withWeight(0)}},
		{"negative weight", []SHSample{withWeight(-1)}},
		{"NaN weight", []SHSample{withWeight(float64(math.NaN()))}},
		{"Inf weight", []SHSample{withWeight(float64(math.Inf(1)))}},
		{"zero direction", []SHSample{withDir(Vec3{0, 0, 0})}},
		{"NaN direction", []SHSample{withDir(Vec3{float64(math.NaN()), 0, 0})}},
		{"Inf direction", []SHSample{withDir(Vec3{0, float64(math.Inf(1)), 0})}},
		{"NaN radiance", []SHSample{withRadiance(Vec3{0, float64(math.NaN()), 0})}},
		{"Inf radiance", []SHSample{withRadiance(Vec3{float64(math.Inf(-1)), 0, 0})}},
		{"valid sample after invalid", []SHSample{valid, withWeight(-1)}},
	}

	for _, tc := range cases {
		if _, err := ProjectSH9(tc.samples); err == nil {
			t.Errorf("%s: expected an error, got nil", tc.name)
		}
	}

	// A single valid sample must not error.
	if _, err := ProjectSH9([]SHSample{valid}); err != nil {
		t.Errorf("valid sample returned error: %v", err)
	}

	// Huge but individually finite weights and radiance can still overflow the
	// scalar range once multiplied; that is reported rather than returned as
	// Inf coefficients.
	huge := SHSample{Dir: Vec3{0, 0, 1}, Weight: MaxValue, Radiance: Vec3{MaxValue, 1, 1}}
	if _, err := ProjectSH9([]SHSample{huge}); err == nil {
		t.Errorf("overflowing coefficients: expected an error, got nil")
	}
}
