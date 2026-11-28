// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mathgl_test holds tests that exercise both scalar packages together.
// The mgl32 and mgl64 packages are generated from the same sources, so the
// most useful cross-check is that they produce the same results for the same
// input within a tolerance appropriate to single precision.
package mathgl_test

import (
	"math"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
)

// TestSH9CrossPrecision projects identical spherical-harmonic samples through
// both packages and checks that the coefficients, reconstructed radiance and
// irradiance agree to single-precision accuracy.
func TestSH9CrossPrecision(t *testing.T) {
	const n = 64

	dirs := make([][3]float64, n)
	radiances := make([][3]float64, n)
	for i := 0; i < n; i++ {
		// Fibonacci-sphere directions, deterministic.
		z := 1 - 2*(float64(i)+0.5)/n
		r := math.Sqrt(math.Max(0, 1-z*z))
		phi := float64(i) * 2.399963229728653
		dirs[i] = [3]float64{r * math.Cos(phi), r * math.Sin(phi), z}
		radiances[i] = [3]float64{
			0.2 + 0.1*float64(i%3),
			0.4 + 0.05*float64(i%5),
			0.1 + 0.03*float64(i%7),
		}
	}
	weight := 4 * math.Pi / n

	samples32 := make([]mgl32.SHSample, n)
	samples64 := make([]mgl64.SHSample, n)
	for i := 0; i < n; i++ {
		d32 := mgl32.Vec3{float32(dirs[i][0]), float32(dirs[i][1]), float32(dirs[i][2])}
		d64 := mgl64.Vec3{dirs[i][0], dirs[i][1], dirs[i][2]}
		l32 := mgl32.Vec3{float32(radiances[i][0]), float32(radiances[i][1]), float32(radiances[i][2])}
		l64 := mgl64.Vec3{radiances[i][0], radiances[i][1], radiances[i][2]}
		samples32[i] = mgl32.SHSample{Dir: d32, Weight: float32(weight), Radiance: l32}
		samples64[i] = mgl64.SHSample{Dir: d64, Weight: weight, Radiance: l64}
	}

	sh32, err := mgl32.ProjectSH9(samples32)
	if err != nil {
		t.Fatalf("mgl32.ProjectSH9: %v", err)
	}
	sh64, err := mgl64.ProjectSH9(samples64)
	if err != nil {
		t.Fatalf("mgl64.ProjectSH9: %v", err)
	}

	approx := func(a, b float64, what string) {
		t.Helper()
		if math.Abs(a-b) > 2e-3*(1+math.Abs(b)) {
			t.Errorf("%s: mgl32 = %v, mgl64 = %v", what, a, b)
		}
	}

	for i := 0; i < 9; i++ {
		for c := 0; c < 3; c++ {
			approx(float64(sh32[i][c]), sh64[i][c], "coefficient")
		}
	}

	normals := [][3]float64{{0, 0, 1}, {1, 0, 0}, {0, 1, 0}, {1, 1, 1}, {-0.5, 2, 0.25}}
	for _, nv := range normals {
		n32 := mgl32.Vec3{float32(nv[0]), float32(nv[1]), float32(nv[2])}
		n64 := mgl64.Vec3{nv[0], nv[1], nv[2]}
		e32 := sh32.Evaluate(n32)
		e64 := sh64.Evaluate(n64)
		i32 := sh32.Irradiance(n32)
		i64 := sh64.Irradiance(n64)
		for c := 0; c < 3; c++ {
			approx(float64(e32[c]), e64[c], "Evaluate")
			approx(float64(i32[c]), i64[c], "Irradiance")
		}
	}
}

// TestSH9CrossPrecisionConstant checks that the constant-environment identity
// (irradiance == pi * radiance) holds in both packages at a realistic error.
func TestSH9CrossPrecisionConstant(t *testing.T) {
	const L = 0.75
	samples32 := make([]mgl32.SHSample, 0, 6)
	samples64 := make([]mgl64.SHSample, 0, 6)
	dirs := [][3]float64{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	for _, d := range dirs {
		samples32 = append(samples32, mgl32.SHSample{
			Dir:    mgl32.Vec3{float32(d[0]), float32(d[1]), float32(d[2])},
			Weight: float32(4 * math.Pi / 6), Radiance: mgl32.Vec3{L, L, L},
		})
		samples64 = append(samples64, mgl64.SHSample{
			Dir:    mgl64.Vec3{d[0], d[1], d[2]},
			Weight: 4 * math.Pi / 6, Radiance: mgl64.Vec3{L, L, L},
		})
	}

	sh32, err := mgl32.ProjectSH9(samples32)
	if err != nil {
		t.Fatalf("mgl32.ProjectSH9: %v", err)
	}
	sh64, err := mgl64.ProjectSH9(samples64)
	if err != nil {
		t.Fatalf("mgl64.ProjectSH9: %v", err)
	}

	want := math.Pi * L
	normal32 := mgl32.Vec3{0, 0, 1}
	normal64 := mgl64.Vec3{0, 0, 1}
	irr32 := sh32.Irradiance(normal32)
	irr64 := sh64.Irradiance(normal64)
	for c := 0; c < 3; c++ {
		if d := math.Abs(float64(irr32[c]) - want); d > 1e-4 {
			t.Errorf("mgl32 constant irradiance channel %d off by %v", c, d)
		}
		if d := math.Abs(irr64[c] - want); d > 1e-12 {
			t.Errorf("mgl64 constant irradiance channel %d off by %v", c, d)
		}
	}
}
