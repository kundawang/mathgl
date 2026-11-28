// This file is generated from mgl32/sh9.go; DO NOT EDIT

// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"errors"
	"fmt"
	"math"
)

// Normalization constants of the nine orthonormal real spherical-harmonic
// basis functions used by SH9. These are the standard constants for the real
// basis, chosen so that the integral of Yi*Yj over the unit sphere is the
// Kronecker delta.
const (
	shY00 = 0.28209479177387814 // 1/(2*sqrt(pi))
	shK1  = 0.4886025119029199  // sqrt(3/(4*pi))
	shK2  = 1.0925484305920792  // sqrt(15/(4*pi))
	shK3  = 0.31539156525252005 // sqrt(5/(16*pi))
	shK4  = 0.5462742152960396  // sqrt(15/(16*pi))
)

// SH9 is a second-order (bands l = 0, 1 and 2) real spherical-harmonic
// expansion of an RGB function on the unit sphere. Each of the nine elements
// holds the coefficient for one basis function and stores the red, green and
// blue channels of that coefficient, so an SH9 describes linear RGB radiance
// as a function of direction.
//
// The basis order is fixed and is documented on SH9Basis. The elements hold
// the raw projection coefficients of the radiance function (see ProjectSH9);
// the Lambertian cosine convolution is applied by Irradiance.
type SH9 [9]Vec3

// SHSample is a single directional radiance sample used by ProjectSH9.
//
// Dir is the direction the radiance arrives from. It is normalized internally
// and must be finite and non-zero. Weight is the solid angle, in steradians,
// that the sample represents. Radiance is linear RGB radiance and must be
// finite.
type SHSample struct {
	Dir      Vec3
	Weight   float64
	Radiance Vec3
}

// sh9BasisAt evaluates the nine real spherical-harmonic basis functions for a
// unit direction (x, y, z). The order here is the order returned by SH9Basis:
//
//	0: Y(0, 0)  = shY00
//	1: Y(1,-1)  = shK1 * y
//	2: Y(1, 0)  = shK1 * z
//	3: Y(1, 1)  = shK1 * x
//	4: Y(2,-2)  = shK2 * x*y
//	5: Y(2,-1)  = shK2 * y*z
//	6: Y(2, 0)  = shK3 * (3*z*z - 1)
//	7: Y(2, 1)  = shK2 * x*z
//	8: Y(2, 2)  = shK4 * (x*x - y*y)
func sh9BasisAt(x, y, z float64) [9]float64 {
	return [9]float64{
		shY00,
		shK1 * y,
		shK1 * z,
		shK1 * x,
		shK2 * x * y,
		shK2 * y * z,
		shK3 * (3*z*z - 1),
		shK2 * x * z,
		shK4 * (x*x - y*y),
	}
}

// sh9Unit normalizes v and reports whether the result is usable. It performs
// all arithmetic in float64 so that mgl32 and mgl64 agree closely, and scales
// by the largest component first so that very large or very small finite
// directions neither overflow nor underflow.
func sh9Unit(v Vec3) (x, y, z float64, ok bool) {
	x, y, z = float64(v[0]), float64(v[1]), float64(v[2])
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) ||
		math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		return 0, 0, 0, false
	}
	m := math.Max(math.Abs(x), math.Max(math.Abs(y), math.Abs(z)))
	if m == 0 {
		return 0, 0, 0, false
	}
	x, y, z = x/m, y/m, z/m
	inv := 1 / math.Sqrt(x*x+y*y+z*z)
	return x * inv, y * inv, z * inv, true
}

// sh9Finite reports whether every component of v is finite.
func sh9Finite(v Vec3) bool {
	for i := range v {
		f := float64(v[i])
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return false
		}
	}
	return true
}

// SH9Basis evaluates the nine real spherical-harmonic basis functions at dir.
// The direction is normalized internally; a zero or non-finite direction
// yields the zero array.
//
// Writing d = (x, y, z) for the normalized direction, the basis is, in order:
//
//	index 0: 0.28209479                          (l=0, m= 0)
//	index 1: 0.48860251 * y                      (l=1, m=-1)
//	index 2: 0.48860251 * z                      (l=1, m= 0)
//	index 3: 0.48860251 * x                      (l=1, m=+1)
//	index 4: 1.09254843 * x*y                    (l=2, m=-2)
//	index 5: 1.09254843 * y*z                    (l=2, m=-1)
//	index 6: 0.31539157 * (3*z*z - 1)            (l=2, m= 0)
//	index 7: 1.09254843 * x*z                    (l=2, m=+1)
//	index 8: 0.54627422 * (x*x - y*y)            (l=2, m=+2)
//
// These functions are orthonormal over the unit sphere: the integral of the
// product of any two distinct basis functions is zero, and the integral of
// the square of each is one. This is the normalization assumed by ProjectSH9
// and Irradiance.
func SH9Basis(dir Vec3) [9]float64 {
	x, y, z, ok := sh9Unit(dir)
	if !ok {
		return [9]float64{}
	}
	b := sh9BasisAt(x, y, z)
	var out [9]float64
	for i := range b {
		out[i] = float64(b[i])
	}
	return out
}

// ProjectSH9 projects weighted directional radiance samples onto the nine
// real spherical-harmonic basis functions and returns their coefficients.
//
// Each sample supplies a direction, the solid angle of directions it
// represents (in steradians), and linear RGB radiance. The projection is
//
//	c[i] = sum over samples of weight * radiance * Yi(direction)
//
// which is the Riemann-sum approximation of the spherical integral
//
//	c[i] = integral over the sphere of L(w) * Yi(w) dw
//
// Weights are therefore solid-angle weights, not probabilities. For a uniform
// set of N samples covering the whole sphere, use weight 4*pi/N, so the
// weights sum to 4*pi. No extra 1/(4*pi), 1/pi or sqrt(4*pi) factor is applied
// to the coefficients.
//
// Directions are normalized before projection; they must be finite and
// non-zero. Weights must be finite and non-negative (individual zero weights
// are allowed), and the total weight must be strictly positive. Radiance must
// be finite. Invalid input returns an error that names the offending sample;
// an empty sample set (which has zero total weight) is also an error.
//
// Because the basis is orthonormal, summing the samples in a different order
// changes only floating-point rounding, and rotating every direction and
// every evaluation normal by the same rotation rotates the reconstructed
// lighting with it. The reconstructed function is truncated at l = 2; an
// environment with higher-frequency detail is approximated, not reproduced.
func ProjectSH9(samples []SHSample) (SH9, error) {
	var acc [9][3]float64
	var total float64

	for i, s := range samples {
		w := float64(s.Weight)
		if math.IsNaN(w) || math.IsInf(w, 0) {
			return SH9{}, fmt.Errorf("spherical harmonics: sample %d has a non-finite weight", i)
		}
		if w < 0 {
			return SH9{}, fmt.Errorf("spherical harmonics: sample %d has a negative weight (%v)", i, s.Weight)
		}

		x, y, z, ok := sh9Unit(s.Dir)
		if !ok {
			return SH9{}, fmt.Errorf("spherical harmonics: sample %d has a zero or non-finite direction", i)
		}
		if !sh9Finite(s.Radiance) {
			return SH9{}, fmt.Errorf("spherical harmonics: sample %d has non-finite radiance", i)
		}

		r0 := float64(s.Radiance[0])
		r1 := float64(s.Radiance[1])
		r2 := float64(s.Radiance[2])
		b := sh9BasisAt(x, y, z)
		for j := 0; j < 9; j++ {
			k := w * b[j]
			acc[j][0] += k * r0
			acc[j][1] += k * r1
			acc[j][2] += k * r2
		}
		total += w
	}

	if math.IsNaN(total) || math.IsInf(total, 0) || total <= 0 {
		return SH9{}, errors.New("spherical harmonics: total sample weight must be positive and finite")
	}

	var sh SH9
	for j := 0; j < 9; j++ {
		sh[j] = Vec3{float64(acc[j][0]), float64(acc[j][1]), float64(acc[j][2])}
		if !sh9Finite(sh[j]) {
			return SH9{}, errors.New("spherical harmonics: coefficients overflow the scalar range")
		}
	}
	return sh, nil
}

// Evaluate reconstructs the truncated radiance function at dir:
//
//	L(dir) ~= sum over i of c[i] * Yi(dir)
//
// The cosine convolution used for Lambertian shading is not applied here; use
// Irradiance for that. The direction is normalized internally; a zero or
// non-finite direction yields the zero vector.
func (sh SH9) Evaluate(dir Vec3) Vec3 {
	x, y, z, ok := sh9Unit(dir)
	if !ok {
		return Vec3{}
	}
	b := sh9BasisAt(x, y, z)
	var r [3]float64
	for i := 0; i < 9; i++ {
		r[0] += float64(sh[i][0]) * b[i]
		r[1] += float64(sh[i][1]) * b[i]
		r[2] += float64(sh[i][2]) * b[i]
	}
	return Vec3{float64(r[0]), float64(r[1]), float64(r[2])}
}

// Irradiance returns the Lambertian diffuse irradiance for a surface with
// normal n, using the truncated second-order expansion:
//
//	E(n) = sum over i of Al * c[i] * Yi(n)
//
// where Al are the eigenvalues of the clamped cosine kernel:
//
//	A0 = pi, A1 = 2*pi/3, A2 = pi/4
//
// This is the standard Ramamoorthi-Hanrahan l <= 2 approximation of
//
//	E(n) = integral over the sphere of L(w) * max(n.w, 0) dw
//
// for a white Lambertian surface. The result is irradiance, not outgoing
// radiance; multiply by albedo/pi to obtain reflected radiance.
//
// A constant environment L yields exactly pi*L for every normal. A single
// directional sample of weight w and radiance L yields
//
//	E(n) = w*L*(1/4 + (n.d)/2 + 5*(3*(n.d)^2 - 1)/32)
//
// which peaks at 1.0625*w*L when n == d instead of w*L; the excess is ringing
// caused by truncating at l = 2. Results can be negative for high-contrast
// lighting, so clamp at shading time if negative values are unacceptable.
//
// n is normalized internally; a zero or non-finite n yields the zero vector.
func (sh SH9) Irradiance(n Vec3) Vec3 {
	x, y, z, ok := sh9Unit(n)
	if !ok {
		return Vec3{}
	}
	b := sh9BasisAt(x, y, z)
	var e [3]float64
	for i := 0; i < 9; i++ {
		var a float64
		switch {
		case i == 0:
			a = math.Pi
		case i < 4:
			a = 2 * math.Pi / 3
		default:
			a = math.Pi / 4
		}
		k := a * b[i]
		e[0] += float64(sh[i][0]) * k
		e[1] += float64(sh[i][1]) * k
		e[2] += float64(sh[i][2]) * k
	}
	return Vec3{float64(e[0]), float64(e[1]), float64(e[2])}
}
