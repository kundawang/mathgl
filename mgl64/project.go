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

// ScreenBounds is an axis-aligned rectangle of pixel coordinates describing
// the extent of something drawn to the screen. The rectangle includes both
// Min and Max, so Width and Height are Max-Minus-Min rather than the count of
// discrete pixels covered.
type ScreenBounds struct {
	Min, Max Vec2
}

// Width returns the horizontal extent of the bounds.
func (b ScreenBounds) Width() float64 { return b.Max[0] - b.Min[0] }

// Height returns the vertical extent of the bounds.
func (b ScreenBounds) Height() float64 { return b.Max[1] - b.Min[1] }

// SphereScreenBounds returns a conservative pixel-space bounding rectangle of
// the on-screen projection of a world-space sphere under a symmetric
// perspective camera. It is meant for scissor/viewport culling of screen-space
// effects: the returned rectangle contains every pixel the sphere may cover,
// while staying tight to the sphere's projected silhouette instead of falling
// back to the whole viewport.
//
// view is the world-to-camera (view) matrix and projection must be a symmetric
// perspective matrix, such as one produced by Perspective. The sphere is given
// by its world-space center and radius. The viewport is described by its lower
// left corner (originX, originY) and its width and height in pixels, matching
// the window-coordinate convention used by Project.
//
// The boolean result is false when no part of the sphere can be visible, in
// which case the returned bounds are zero-valued and must not be used. It is
// true otherwise, including when the sphere surrounds the camera or straddles
// the camera plane, in which case its projection is unbounded and the whole
// viewport is returned. A sphere that merely intersects the near plane still
// yields a tight rectangle.
//
// Non-finite inputs, a non-positive viewport, a zero or negative radius, or a
// projection that is not a symmetric perspective matrix are reported as not
// visible rather than producing a meaningless rectangle.
func SphereScreenBounds(center Vec3, radius float64, view, projection Mat4, originX, originY, width, height int) (ScreenBounds, bool) {
	if width <= 0 || height <= 0 || radius < 0 || !isFiniteV3(center) || !isFinite(radius) ||
		!isFiniteM4(view) || !isFiniteM4(projection) {
		return ScreenBounds{}, false
	}

	// A symmetric perspective matrix is zero everywhere except the diagonal
	// scale terms m[0], m[5], the depth terms m[10], m[14] and the m[11] = -1
	// entry that performs the perspective divide. Reject anything with off-axis
	// shear or a non-perspective structure.
	if projection[1] != 0 || projection[2] != 0 || projection[3] != 0 ||
		projection[4] != 0 || projection[6] != 0 || projection[7] != 0 ||
		projection[8] != 0 || projection[9] != 0 || projection[11] != -1 ||
		projection[12] != 0 || projection[13] != 0 || projection[15] != 0 ||
		projection[0] == 0 || projection[5] == 0 {
		return ScreenBounds{}, false
	}

	// Move the sphere into camera space, where the camera sits at the origin
	// looking down the negative Z axis.
	eye := view.Mul4x1(center.Vec4(1)).Vec3()
	if !isFiniteV3(eye) {
		return ScreenBounds{}, false
	}

	cx, cy, cz := eye[0], eye[1], eye[2]
	rsq := radius * radius

	if radius == 0 {
		// A zero radius sphere is a single point: it is visible only when it is
		// strictly in front of the camera plane.
		if cz >= 0 {
			return ScreenBounds{}, false
		}
		return ndcToPixelBounds(cx/(-cz), cx/(-cz), cy/(-cz), cy/(-cz),
			projection[0], projection[5], originX, originY, width, height)
	}

	distSq := eye.Dot(eye)
	switch {
	case distSq <= rsq:
		// The camera is inside (or on) the sphere: every viewing ray hits it, so
		// its projection covers the whole screen.
		return fullViewportBounds(originX, originY, width, height), true
	case cz-radius >= 0:
		// The closest point of the sphere is on or behind the camera plane, so
		// nothing projects. (cz is the eye-space depth, negative in front.)
		return ScreenBounds{}, false
	case cz+radius >= 0:
		// The sphere straddles the camera plane without containing the camera.
		// Its silhouette cone is not strictly forward-facing, so the projection
		// is unbounded and covers the whole screen.
		return fullViewportBounds(originX, originY, width, height), true
	}

	// The sphere is fully in front of the camera and the camera is outside it.
	//
	// Rays through the pinhole camera that intersect the sphere form a circular
	// cone. In slope space (p = x/-z, q = y/-z) its cross-section is a conic.
	// Substituting the vertical/horizontal tangents of that conic gives the
	// exact axis-aligned extents below; unlike adding a linear screen-space
	// radius to the projected center, this accounts for perspective foreshortening
	// and never clips the silhouette edge of a near sphere.
	denom := cz*cz - rsq
	pCenter := -cx * cz / denom
	pHalf := radius * float64(math.Sqrt(float64(cz*cz-rsq+cx*cx))) / denom
	qCenter := -cy * cz / denom
	qHalf := radius * float64(math.Sqrt(float64(cz*cz-rsq+cy*cy))) / denom

	if !isFinite(pCenter) || !isFinite(pHalf) || !isFinite(qCenter) || !isFinite(qHalf) {
		return fullViewportBounds(originX, originY, width, height), true
	}

	return ndcToPixelBounds(pCenter-pHalf, pCenter+pHalf, qCenter-qHalf, qCenter+qHalf,
		projection[0], projection[5], originX, originY, width, height)
}

// ndcToPixelBounds maps NDC-space extents (ndcX/ndcY in [-1, 1] for the visible
// frustum) to a pixel rectangle, clipping the result to the viewport. Bounds that
// do not overlap the viewport are reported as not visible.
func ndcToPixelBounds(minX, maxX, minY, maxY, scaleX, scaleY float64, originX, originY, width, height int) (ScreenBounds, bool) {
	minX *= scaleX
	maxX *= scaleX
	minY *= scaleY
	maxY *= scaleY

	// A small NDC tolerance keeps a sphere whose projection is truly tangent to
	// the frustum edge from being culled by floating-point rounding.
	const edgeEps = 1e-6
	if maxX < -1-edgeEps || minX > 1+edgeEps || maxY < -1-edgeEps || minY > 1+edgeEps {
		return ScreenBounds{}, false
	}

	w, h := float64(width), float64(height)
	ox, oy := float64(originX), float64(originY)

	// Round outward so the integer pixel rectangle cannot miss a covered pixel.
	minPixelX := ox + float64(math.Floor(float64(w*(Clamp(minX, -1, 1)+1)/2)))
	maxPixelX := ox + float64(math.Ceil(float64(w*(Clamp(maxX, -1, 1)+1)/2)))
	minPixelY := oy + float64(math.Floor(float64(h*(Clamp(minY, -1, 1)+1)/2)))
	maxPixelY := oy + float64(math.Ceil(float64(h*(Clamp(maxY, -1, 1)+1)/2)))

	return ScreenBounds{
		Min: Vec2{minPixelX, minPixelY},
		Max: Vec2{maxPixelX, maxPixelY},
	}, true
}

func fullViewportBounds(originX, originY, width, height int) ScreenBounds {
	return ScreenBounds{
		Min: Vec2{float64(originX), float64(originY)},
		Max: Vec2{float64(originX + width), float64(originY + height)},
	}
}

func isFinite(v float64) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
}

func isFiniteV3(v Vec3) bool {
	return isFinite(v[0]) && isFinite(v[1]) && isFinite(v[2])
}

func isFiniteM4(m Mat4) bool {
	for _, v := range m {
		if !isFinite(v) {
			return false
		}
	}
	return true
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
