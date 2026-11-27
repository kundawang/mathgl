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

// CascadeSplit describes one depth slice of a camera frustum, as used by a
// cascaded shadow map.
//
// Near and Far are eye-space distances from the camera along the view
// direction; the slice covers the depth interval [Near, Far]. Corners holds
// the eight world-space corner points of the slice, in this fixed order:
//
//	Corners[0:4] are the near cross-section, Corners[4:8] the far one.
//	Within each cross-section the order is:
//	0/4: (-1, -1) in normalized device coordinates (bottom left)
//	1/5: (+1, -1) (bottom right)
//	2/6: (+1, +1) (top right)
//	3/7: (-1, +1) (top left)
//
// Because both cross-sections come from the same normalized-device-space
// rectangle, the corners of neighbouring splits at a shared boundary depth
// are bit-for-bit identical.
type CascadeSplit struct {
	Near    float32
	Far     float32
	Corners [8]Vec3
}

// NearCorners returns the four world-space corners of the split's near
// cross-section, in the order bottom left, bottom right, top right, top left.
func (s CascadeSplit) NearCorners() [4]Vec3 {
	return [4]Vec3{s.Corners[0], s.Corners[1], s.Corners[2], s.Corners[3]}
}

// FarCorners returns the four world-space corners of the split's far
// cross-section, in the order bottom left, bottom right, top right, top left.
func (s CascadeSplit) FarCorners() [4]Vec3 {
	return [4]Vec3{s.Corners[4], s.Corners[5], s.Corners[6], s.Corners[7]}
}

// finiteFloat reports whether f is neither NaN nor an infinity.
func finiteFloat(f float32) bool {
	return !math.IsNaN(float64(f)) && !math.IsInf(float64(f), 0)
}

// finiteVec3 reports whether every component of v is finite.
func finiteVec3(v Vec3) bool {
	return finiteFloat(v[0]) && finiteFloat(v[1]) && finiteFloat(v[2])
}

// finiteMat4 reports whether every element of m is finite.
func finiteMat4(m Mat4) bool {
	for _, v := range m {
		if !finiteFloat(v) {
			return false
		}
	}
	return true
}

// validateCascadeDepth checks the near/far distances shared by the cascade
// split queries. Distances are positive eye-space distances, so the near
// plane may not sit at or behind the camera and far must be beyond near.
func validateCascadeDepth(near, far float32) error {
	switch {
	case !finiteFloat(near) || !finiteFloat(far):
		return errors.New("cascade split: near and far must be finite")
	case near <= 0:
		return errors.New("cascade split: near must be greater than zero")
	case far <= near:
		return errors.New("cascade split: far must be greater than near")
	}
	return nil
}

// maxCascadeCount bounds the number of requested splits so that an absurd
// count fails with an error instead of attempting an enormous allocation.
const maxCascadeCount = 1 << 16

// validateCascadeSplit checks all parameters of a cascade split query.
func validateCascadeSplit(near, far float32, cascades int, lambda float32) error {
	if err := validateCascadeDepth(near, far); err != nil {
		return err
	}
	switch {
	case !finiteFloat(lambda):
		return errors.New("cascade split: lambda must be finite")
	case lambda < 0 || lambda > 1:
		return errors.New("cascade split: lambda must be between zero and one")
	case cascades < 1:
		return errors.New("cascade split: cascades must be at least one")
	case cascades > maxCascadeCount:
		return errors.New("cascade split: cascades is unreasonably large")
	}
	return nil
}

// CascadeSplitDepths computes the depth boundaries of a cascaded shadow map
// for the depth interval [near, far].
//
// cascades is the number of slices; the returned slice holds cascades+1
// monotonically increasing boundaries, starting at near and ending at far.
// Boundary i for 0 < i < cascades blends the two usual split schemes:
//
//	linear:      near + (far-near)*(i/cascades)
//	logarithmic: near * (far/near)^(i/cascades)
//	blended:     (1-lambda)*linear + lambda*logarithmic
//
// lambda must lie in [0, 1]: zero produces purely linear splits, one purely
// logarithmic splits (more resolution near the camera). The formula works on
// eye-space distances, so it is independent of the view matrix and of any
// off-axis frustum extents.
func CascadeSplitDepths(near, far float32, cascades int, lambda float32) ([]float32, error) {
	if err := validateCascadeSplit(near, far, cascades, lambda); err != nil {
		return nil, err
	}

	depths := make([]float32, cascades+1)
	depths[0] = near
	depths[cascades] = far
	for i := 1; i < cascades; i++ {
		f := float64(i) / float64(cascades)
		linear := float64(near) + float64(far-near)*f
		logarithmic := float64(near) * math.Pow(float64(far/near), f)
		blended := (1-float64(lambda))*linear + float64(lambda)*logarithmic
		depths[i] = float32(blended)
	}
	return depths, nil
}

// cascadeInverse returns the inverse of projection*view, which maps normalized
// device coordinates back to world space. A singular or non-finite matrix is
// rejected rather than silently producing garbage corners.
func cascadeInverse(view, projection Mat4) (Mat4, error) {
	mvp := projection.Mul4(view)
	if !finiteMat4(mvp) {
		return Mat4{}, errors.New("cascade split: view/projection matrix is not finite")
	}
	inv := mvp.Inv()
	var zero Mat4
	if inv == zero {
		return Mat4{}, errors.New("cascade split: view/projection matrix is singular")
	}
	if !finiteMat4(inv) {
		return Mat4{}, errors.New("cascade split: view/projection matrix is singular")
	}
	return inv, nil
}

// clipDepthForDepth returns the normalized device z coordinate that the
// projection matrix assigns to a view-space point at the given eye-space
// depth. Every standard projection in this package (Perspective, Frustum and
// Ortho) maps depth through the projection's z row alone, so the point on the
// view axis yields the right clip depth for the whole cross-section.
func clipDepthForDepth(projection Mat4, depth float32) (float32, error) {
	clip := projection.Mul4x1(Vec4{0, 0, -depth, 1})
	w := clip.W()
	if w == 0 || !finiteFloat(w) {
		return 0, errors.New("cascade split: projection maps a depth to an invalid clip coordinate")
	}
	z := clip.Z() / w
	if !finiteFloat(z) {
		return 0, errors.New("cascade split: projection maps a depth to an invalid clip coordinate")
	}
	return z, nil
}

// frustumCrossSection returns the four world-space corners of the camera
// frustum at the given eye-space depth. The corners correspond to the
// normalized device coordinates (-1,-1), (+1,-1), (+1,+1), (-1,+1), so an
// off-axis (asymmetric) frustum is handled just like a symmetric one.
func frustumCrossSection(inv, projection Mat4, depth float32) ([4]Vec3, error) {
	z, err := clipDepthForDepth(projection, depth)
	if err != nil {
		return [4]Vec3{}, err
	}

	var corners [4]Vec3
	ndcX := [4]float32{-1, 1, 1, -1}
	ndcY := [4]float32{-1, -1, 1, 1}
	for i := range corners {
		clip := inv.Mul4x1(Vec4{ndcX[i], ndcY[i], z, 1})
		w := clip.W()
		if w == 0 || !finiteFloat(w) {
			return [4]Vec3{}, errors.New("cascade split: view/projection matrix maps a frustum corner to infinity")
		}
		corner := Vec3{clip.X() / w, clip.Y() / w, clip.Z() / w}
		if !finiteVec3(corner) {
			return [4]Vec3{}, errors.New("cascade split: view/projection matrix maps a frustum corner to a non-finite point")
		}
		corners[i] = corner
	}
	return corners, nil
}

// FrustumCorners returns the eight world-space corners of the camera frustum
// covering the eye-space depth interval [near, far].
//
// view is the world-to-view transform and projection the view-to-clip
// transform; together they define the camera. The corner order is the same as
// for CascadeSplit: the four near corners first (bottom left, bottom right,
// top right, top left in normalized device coordinates), then the four far
// corners in the same order. Asymmetric (off-axis) frustums are supported,
// since the corners are recovered by inverting projection*view at the fixed
// normalized device coordinates.
//
// An error is returned if near/far are illegal, if the combined matrix is
// singular, or if a corner does not map back to a finite world-space point.
func FrustumCorners(view, projection Mat4, near, far float32) ([8]Vec3, error) {
	if err := validateCascadeDepth(near, far); err != nil {
		return [8]Vec3{}, err
	}
	inv, err := cascadeInverse(view, projection)
	if err != nil {
		return [8]Vec3{}, err
	}
	nearCorners, err := frustumCrossSection(inv, projection, near)
	if err != nil {
		return [8]Vec3{}, err
	}
	farCorners, err := frustumCrossSection(inv, projection, far)
	if err != nil {
		return [8]Vec3{}, err
	}

	var corners [8]Vec3
	copy(corners[0:4], nearCorners[:])
	copy(corners[4:8], farCorners[:])
	return corners, nil
}

// CascadeSplits computes the geometry of cascaded shadow map splits for a
// camera defined by view (world-to-view) and projection (view-to-clip).
//
// It returns cascades slices covering [near, far]. Each slice carries its
// depth interval and the eight world-space corners of the frustum section at
// those depths; see CascadeSplit for the corner order. The split depths follow
// CascadeSplitDepths, so lambda blends linear and logarithmic partitioning
// (zero is fully linear, one fully logarithmic).
//
// The camera may use any of the package's perspective projections, including
// asymmetric (off-axis) frustums, because the corners are recovered through
// the inverse of projection*view. Adjacent slices share the exact same corner
// values at the boundary depth. An error is returned for illegal depths or
// weights, for a singular projection*view, or when the matrix does not map a
// corner to a finite world-space point.
func CascadeSplits(view, projection Mat4, near, far float32, cascades int, lambda float32) ([]CascadeSplit, error) {
	depths, err := CascadeSplitDepths(near, far, cascades, lambda)
	if err != nil {
		return nil, err
	}
	inv, err := cascadeInverse(view, projection)
	if err != nil {
		return nil, err
	}

	// Compute each cross-section once so the slices on either side of a
	// boundary reuse exactly the same corner values.
	crossSections := make([][4]Vec3, len(depths))
	for i, depth := range depths {
		corners, err := frustumCrossSection(inv, projection, depth)
		if err != nil {
			return nil, err
		}
		crossSections[i] = corners
	}

	splits := make([]CascadeSplit, cascades)
	for i := range splits {
		splits[i].Near = depths[i]
		splits[i].Far = depths[i+1]
		copy(splits[i].Corners[0:4], crossSections[i][:])
		copy(splits[i].Corners[4:8], crossSections[i+1][:])
	}
	return splits, nil
}

// LookAt generates a transform matrix from world space to the given eye space.
func LookAt(eyeX, eyeY, eyeZ, centerX, centerY, centerZ, upX, upY, upZ float32) Mat4 {
	return LookAtV(Vec3{eyeX, eyeY, eyeZ}, Vec3{centerX, centerY, centerZ}, Vec3{upX, upY, upZ})
}

//  LookAtV generates a transform matrix from world space into the specific eye space
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
