// This file is generated from mgl32/project.go; DO NOT EDIT

// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"errors"
	"math"
)

// Ortho generates an Ortho Matrix.
func Ortho(left, right, bottom, top, near, far float64) Mat4 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)

	return Mat4{float64(2. / rml), 0, 0, 0, 0, float64(2. / tmb), 0, 0, 0, 0, float64(-2. / fmn), 0, float64(-(right + left) / rml), float64(-(top + bottom) / tmb), float64(-(far + near) / fmn), 1}
}

// Ortho2D is equivalent to Ortho with the near and far planes being -1 and 1,
// respectively.
func Ortho2D(left, right, bottom, top float64) Mat4 {
	return Ortho(left, right, bottom, top, -1, 1)
}

// Perspective generates a Perspective Matrix.
func Perspective(fovy, aspect, near, far float64) Mat4 {
	// fovy = (fovy * math.Pi) / 180.0 // convert from degrees to radians
	nmf, f := near-far, float64(1./math.Tan(float64(fovy)/2.0))

	return Mat4{float64(f / aspect), 0, 0, 0, 0, float64(f), 0, 0, 0, 0, float64((near + far) / nmf), -1, 0, 0, float64((2. * far * near) / nmf), 0}
}

// allFinite reports whether every argument is a finite, non-NaN floating
// point value.
func allFinite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return true
}

// PerspectiveReverseZ generates a reverse-Z perspective projection matrix.
//
// It shares the left/right and top/bottom frustum extents of Perspective, but
// maps eye-space depth onto the OpenGL clip-space depth range in the reverse
// direction: the near plane (eye z = -near) maps to a clip depth of +1 and the
// far plane (eye z = -far) maps to a clip depth of -1. Keeping the full depth
// buffer precision near the camera makes this useful together with a floating
// point depth buffer.
//
// near and far are positive distances from the camera, as with Perspective.
// All parameters must be finite; near must be greater than zero, far must be
// greater than near, fovy must be between zero and pi (exclusive) and aspect
// must be greater than zero. If any parameter is invalid, a zero matrix and a
// non-nil error are returned instead of an invalid projection.
func PerspectiveReverseZ(fovy, aspect, near, far float64) (Mat4, error) {
	switch {
	case !allFinite(fovy, aspect, near, far):
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZ requires all parameters to be finite")
	case near <= 0:
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZ requires a positive near plane")
	case far <= near:
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZ requires far to be greater than near")
	case fovy <= 0 || fovy >= math.Pi:
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZ requires fovy to be between 0 and pi")
	case aspect <= 0:
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZ requires a positive aspect ratio")
	}

	f := float64(1. / math.Tan(float64(fovy)/2.0))

	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, float64((near + far) / (far - near)), -1,
		0, 0, float64((2. * far * near) / (far - near)), 0,
	}, nil
}

// PerspectiveReverseZInf generates a reverse-Z perspective projection matrix
// with an infinitely distant far plane.
//
// It is the infinite-far limit of PerspectiveReverseZ: the near plane (eye
// z = -near) maps to a clip depth of +1 and the depth approaches -1 as eye
// space depth goes to negative infinity. The left/right and top/bottom
// frustum extents match those of Perspective.
//
// near is a positive distance from the camera. All parameters must be finite;
// near must be greater than zero, fovy must be between zero and pi (exclusive)
// and aspect must be greater than zero. If any parameter is invalid, a zero
// matrix and a non-nil error are returned instead of an invalid projection.
func PerspectiveReverseZInf(fovy, aspect, near float64) (Mat4, error) {
	switch {
	case !allFinite(fovy, aspect, near):
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZInf requires all parameters to be finite")
	case near <= 0:
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZInf requires a positive near plane")
	case fovy <= 0 || fovy >= math.Pi:
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZInf requires fovy to be between 0 and pi")
	case aspect <= 0:
		return Mat4{}, errors.New("mgl64: PerspectiveReverseZInf requires a positive aspect ratio")
	}

	f := float64(1. / math.Tan(float64(fovy)/2.0))

	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, 1, -1,
		0, 0, 2 * near, 0,
	}, nil
}

// OrthoReverseZ generates a reverse-Z orthographic projection matrix.
//
// It shares the left/right and top/bottom bounds of Ortho, but maps eye-space
// depth onto the OpenGL clip-space depth range in the reverse direction: the
// near plane (eye z = -near) maps to a clip depth of +1 and the far plane
// (eye z = -far) maps to a clip depth of -1.
//
// As with glOrtho and Ortho, near and far are positive plane distances; the
// planes themselves lie at eye-space z values -near and -far. All parameters
// must be finite; near must be greater than zero, far must be greater than
// near, left must differ from right and bottom must differ from top. If any
// parameter is invalid, a zero matrix and a non-nil error are returned instead
// of an invalid projection.
func OrthoReverseZ(left, right, bottom, top, near, far float64) (Mat4, error) {
	switch {
	case !allFinite(left, right, bottom, top, near, far):
		return Mat4{}, errors.New("mgl64: OrthoReverseZ requires all parameters to be finite")
	case near <= 0:
		return Mat4{}, errors.New("mgl64: OrthoReverseZ requires a positive near plane")
	case far <= near:
		return Mat4{}, errors.New("mgl64: OrthoReverseZ requires far to be greater than near")
	case left == right:
		return Mat4{}, errors.New("mgl64: OrthoReverseZ requires left to differ from right")
	case bottom == top:
		return Mat4{}, errors.New("mgl64: OrthoReverseZ requires bottom to differ from top")
	}

	rml, tmb, fmn := (right - left), (top - bottom), (far - near)

	return Mat4{
		float64(2. / rml), 0, 0, 0,
		0, float64(2. / tmb), 0, 0,
		0, 0, float64(2. / fmn), 0,
		float64(-(right + left) / rml), float64(-(top + bottom) / tmb), float64((far + near) / fmn), 1,
	}, nil
}

// Frustum generates a Frustum Matrix.
func Frustum(left, right, bottom, top, near, far float64) Mat4 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	A, B, C, D := (right+left)/rml, (top+bottom)/tmb, -(far+near)/fmn, -(2*far*near)/fmn

	return Mat4{float64((2. * near) / rml), 0, 0, 0, 0, float64((2. * near) / tmb), 0, 0, float64(A), float64(B), float64(C), -1, 0, 0, float64(D), 0}
}

// LookAt generates a transform matrix from world space to the given eye space.
func LookAt(eyeX, eyeY, eyeZ, centerX, centerY, centerZ, upX, upY, upZ float64) Mat4 {
	return LookAtV(Vec3{eyeX, eyeY, eyeZ}, Vec3{centerX, centerY, centerZ}, Vec3{upX, upY, upZ})
}

// LookAtV generates a transform matrix from world space into the specific eye
// space.
func LookAtV(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Normalize()
	s := f.Cross(up.Normalize()).Normalize()
	u := s.Cross(f)

	M := Mat4{
		s[0], u[0], -f[0], 0,
		s[1], u[1], -f[1], 0,
		s[2], u[2], -f[2], 0,
		0, 0, 0, 1,
	}

	return M.Mul4(Translate3D(float64(-eye[0]), float64(-eye[1]), float64(-eye[2])))
}

// Project transforms a set of coordinates from object space (in obj) to window
// coordinates (with depth).
//
// Window coordinates are continuous, not discrete (well, as continuous as an
// IEEE Floating Point can be), so you won't get exact pixel locations without
// rounding or similar
func Project(obj Vec3, modelview, projection Mat4, initialX, initialY, width, height int) (win Vec3) {
	obj4 := obj.Vec4(1)

	vpp := projection.Mul4(modelview).Mul4x1(obj4)
	vpp = vpp.Mul(1 / vpp.W())
	win[0] = float64(initialX) + (float64(width)*(vpp[0]+1))/2
	win[1] = float64(initialY) + (float64(height)*(vpp[1]+1))/2
	win[2] = (vpp[2] + 1) / 2

	return win
}

// UnProject transforms a set of window coordinates to object space. If your MVP
// (projection.Mul(modelview) matrix is not invertible, this will return an
// error.
//
// Note that the projection may not be perfect if you use strict pixel locations
// rather than the exact values given by Projectf. (It's still unlikely to be
// perfect due to precision errors, but it will be closer)
func UnProject(win Vec3, modelview, projection Mat4, initialX, initialY, width, height int) (obj Vec3, err error) {
	inv := projection.Mul4(modelview).Inv()
	var blank Mat4
	if inv == blank {
		return Vec3{}, errors.New("Could not find matrix inverse (projection times modelview is probably non-singular)")
	}

	obj4 := inv.Mul4x1(Vec4{
		(2 * (win[0] - float64(initialX)) / float64(width)) - 1,
		(2 * (win[1] - float64(initialY)) / float64(height)) - 1,
		2*win[2] - 1,
		1.0,
	})
	obj = obj4.Vec3()

	//if obj4[3] > MinValue {}
	obj[0] /= obj4[3]
	obj[1] /= obj4[3]
	obj[2] /= obj4[3]

	return obj, nil
}
