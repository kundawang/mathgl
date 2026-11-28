// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl32

import (
	"errors"
	"math"
)

func Ortho(left, right, bottom, top, near, far float32) Mat4 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)

	return Mat4{float32(2. / rml), 0, 0, 0, 0, float32(2. / tmb), 0, 0, 0, 0, float32(-2. / fmn), 0, float32(-(right + left) / rml), float32(-(top + bottom) / tmb), float32(-(far + near) / fmn), 1}
}

// Equivalent to Ortho with the near and far planes being -1 and 1, respectively
func Ortho2D(left, right, bottom, top float32) Mat4 {
	return Ortho(left, right, bottom, top, -1, 1)
}

func Perspective(fovy, aspect, near, far float32) Mat4 {
	// fovy = (fovy * math.Pi) / 180.0 // convert from degrees to radians
	nmf, f := near-far, float32(1./math.Tan(float64(fovy)/2.0))

	return Mat4{float32(f / aspect), 0, 0, 0, 0, float32(f), 0, 0, 0, 0, float32((near + far) / nmf), -1, 0, 0, float32((2. * far * near) / nmf), 0}
}

func Frustum(left, right, bottom, top, near, far float32) Mat4 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	A, B, C, D := (right+left)/rml, (top+bottom)/tmb, -(far+near)/fmn, -(2*far*near)/fmn

	return Mat4{float32((2. * near) / rml), 0, 0, 0, 0, float32((2. * near) / tmb), 0, 0, float32(A), float32(B), float32(C), -1, 0, 0, float32(D), 0}
}

func LookAt(eyeX, eyeY, eyeZ, centerX, centerY, centerZ, upX, upY, upZ float32) Mat4 {
	return LookAtV(Vec3{eyeX, eyeY, eyeZ}, Vec3{centerX, centerY, centerZ}, Vec3{upX, upY, upZ})
}

// LookAtV generates a transform matrix from world space into the specific eye space
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

	return M.Mul4(Translate3D(float32(-eye[0]), float32(-eye[1]), float32(-eye[2])))
}

// StereoPerspective builds a pair of off-axis (asymmetric frustum) stereo
// cameras from a central "cyclopean" camera pose, following the same
// right-handed, column-major conventions as LookAtV, Frustum and Perspective.
//
// eye, center and up describe the central camera exactly as in LookAtV.
// eyeSep is the distance between the two eyes (interocular distance); it must
// be >= 0, and 0 degenerates to a monocular camera identical to
// LookAtV/Perspective. convDist is the distance from the central camera to
// the convergence (zero-parallax) plane, measured along the view direction;
// points on that plane project to the same horizontal position in both eyes.
// fovy, aspect, near and far are the perspective parameters, as in
// Perspective.
//
// The returned views are parallel (no toe-in): each eye is the central
// camera translated by +-eyeSep/2 along the camera's right vector, while the
// projections are horizontally shifted asymmetric frusta chosen so the two
// frusta coincide exactly on the convergence plane. Simply translating a
// symmetric projection would give wrong disparities on the convergence plane
// (and keystone distortion); this construction does not.
//
// An error is returned if any parameter is non-finite, eyeSep < 0,
// convDist <= 0, near <= 0, far <= near, fovy is not in (0, pi),
// aspect <= 0, or the pose is degenerate (eye == center, zero-length up, or
// up parallel to the view direction).
func StereoPerspective(eye, center, up Vec3, eyeSep, convDist, fovy, aspect, near, far float32) (leftView, rightView, leftProj, rightProj Mat4, err error) {
	fail := func(msg string) (Mat4, Mat4, Mat4, Mat4, error) {
		return Mat4{}, Mat4{}, Mat4{}, Mat4{}, errors.New(msg)
	}

	for _, v := range []float32{
		eye[0], eye[1], eye[2],
		center[0], center[1], center[2],
		up[0], up[1], up[2],
		eyeSep, convDist, fovy, aspect, near, far,
	} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return fail("StereoPerspective: all parameters must be finite")
		}
	}
	if eyeSep < 0 {
		return fail("StereoPerspective: eye separation must be >= 0")
	}
	if convDist <= 0 {
		return fail("StereoPerspective: convergence distance must be > 0")
	}
	if near <= 0 || far <= near {
		return fail("StereoPerspective: require 0 < near < far")
	}
	if fovy <= 0 || fovy >= float32(math.Pi) {
		return fail("StereoPerspective: fovy must be in (0, pi)")
	}
	if aspect <= 0 {
		return fail("StereoPerspective: aspect must be > 0")
	}

	forward := center.Sub(eye)
	if forward.Len() == 0 {
		return fail("StereoPerspective: eye and center must differ")
	}
	forward = forward.Normalize()
	if up.Len() == 0 {
		return fail("StereoPerspective: up must be non-zero")
	}
	right := forward.Cross(up)
	if right.Len() == 0 {
		return fail("StereoPerspective: up must not be parallel to the view direction")
	}
	right = right.Normalize()

	halfSep := eyeSep / 2
	leftEye := eye.Sub(right.Mul(halfSep))
	rightEye := eye.Add(right.Mul(halfSep))

	// Parallel views: same orientation as the central camera, translated
	// along its right vector.
	leftView = LookAtV(leftEye, leftEye.Add(forward), up)
	rightView = LookAtV(rightEye, rightEye.Add(forward), up)

	// Off-axis frusta: the symmetric frustum of half-width halfW at the near
	// plane, shifted horizontally so both eyes' frusta intersect exactly on
	// the convergence plane.
	top := near * float32(math.Tan(float64(fovy)/2))
	halfW := aspect * top
	shift := near * halfSep / convDist
	leftProj = Frustum(-halfW+shift, halfW+shift, -top, top, near, far)
	rightProj = Frustum(-halfW-shift, halfW-shift, -top, top, near, far)

	return leftView, rightView, leftProj, rightProj, nil
}

// Transform a set of coordinates from object space (in obj) to window coordinates (with depth)
//
// Window coordinates are continuous, not discrete (well, as continuous as an IEEE Floating Point can be), so you won't get exact pixel locations
// without rounding or similar
func Project(obj Vec3, modelview, projection Mat4, initialX, initialY, width, height int) (win Vec3) {
	obj4 := obj.Vec4(1)

	vpp := projection.Mul4(modelview).Mul4x1(obj4)
	vpp = vpp.Mul(1 / vpp.W())
	win[0] = float32(initialX) + (float32(width)*(vpp[0]+1))/2
	win[1] = float32(initialY) + (float32(height)*(vpp[1]+1))/2
	win[2] = (vpp[2] + 1) / 2

	return win
}

// Transform a set of window coordinates to object space. If your MVP (projection.Mul(modelview) matrix is not invertible, this will return an error
//
// Note that the projection may not be perfect if you use strict pixel locations rather than the exact values given by Projectf.
// (It's still unlikely to be perfect due to precision errors, but it will be closer)
func UnProject(win Vec3, modelview, projection Mat4, initialX, initialY, width, height int) (obj Vec3, err error) {
	inv := projection.Mul4(modelview).Inv()
	var blank Mat4
	if inv == blank {
		return Vec3{}, errors.New("Could not find matrix inverse (projection times modelview is probably non-singular)")
	}

	obj4 := inv.Mul4x1(Vec4{
		(2 * (win[0] - float32(initialX)) / float32(width)) - 1,
		(2 * (win[1] - float32(initialY)) / float32(height)) - 1,
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
