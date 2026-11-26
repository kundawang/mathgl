// This file is generated from mgl32/obb.go; DO NOT EDIT

// Copyright 2024 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"errors"
	"fmt"
	"math"
)

// OBB is a 3D oriented bounding box: a center, three positive half extents
// and an orthonormal frame giving the directions of the box's faces and edges.
//
// Axes must be orthonormal: each axis has unit length and every pair is
// perpendicular. Either handedness is accepted. HalfExtents are the
// half-widths along Axes[0], Axes[1] and Axes[2] and must be strictly
// positive; zero collapses a dimension and negative values are sign errors,
// so Validate rejects both. Degenerate zero-half boxes are supported through
// ValidateDegenerate and TryOBBIntersectionDegenerate. Use Validate or
// TryOBBIntersection to detect malformed boxes without panicking.
type OBB struct {
	Center      Vec3
	HalfExtents Vec3
	Axes        [3]Vec3
}

// OrientedBoundingBox is an alias for OBB for callers that prefer the long
// name.
type OrientedBoundingBox = OBB

// OBBHit is the result of an OBB intersection query.
//
// The Separating Axis Theorem compares the projection intervals of both boxes
// on fifteen axes (the six face normals and the nine pairwise cross products
// of edges). The axis with the smallest interval overlap is reported in Axis
// together with that overlap.
type OBBHit struct {
	// Intersects is true when the boxes overlap or exactly touch. Contact with
	// zero depth counts as intersection.
	Intersects bool

	// Axis is the candidate axis with the smallest projected overlap (the
	// canonical SAT witness): the minimum penetration axis while
	// intersecting, the strongest separating axis while disjoint. It is a
	// unit vector oriented from B's center towards A's center (its sign is
	// arbitrary when the centers coincide).
	Axis Vec3

	// Depth is the unsigned magnitude of SignedDepth: the minimum
	// penetration depth while Intersects is true (0 at exact contact), or
	// the separating-axis gap while false. With a single separating axis,
	// the common "boxes pulled apart on one axis" case, it is exactly the
	// distance they must close to touch.
	Depth float64

	// SignedDepth is positive for penetration, zero at exact contact and
	// negative for a separation gap.
	SignedDepth float64

	// Penetration is max(SignedDepth, 0): the penetration depth, or 0 when
	// the boxes only touch or are disjoint.
	Penetration float64

	// Separation is max(-SignedDepth, 0): the gap between disjoint boxes, or
	// 0 when they touch or overlap.
	Separation float64
}

const (
	// Orthonormality tolerances. Rotation matrices produced by this package
	// stay well inside them; clearly non-orthogonal hand-built frames are
	// rejected.
	obbOrthoLengthEps = 1e-9
	obbOrthoDotEps    = 1e-9

	// Cross products shorter than this are treated as parallel axes and
	// skipped: a zero cross product is not a valid axis, and the parallel
	// configuration is already covered by the face axes.
	obbParallelEps = 1e-12

	// Scale-independent floating point slack used to absorb rounding at
	// exact contact.
	obbTouchSlackFactor = 16 * 2.2204460492503131e-16 // 16 * machine epsilon
)

// NewOBB builds an OBB from its center, half extents and orthonormal axes.
// It panics if the definition is malformed; see Validate for the rules.
func NewOBB(center, halfExtents Vec3, axes [3]Vec3) OBB {
	o := OBB{Center: center, HalfExtents: halfExtents, Axes: axes}
	if err := o.Validate(); err != nil {
		panic(err)
	}
	return o
}

// NewAxisAlignedOBB builds an OBB whose axes are the world X, Y and Z axes.
func NewAxisAlignedOBB(center, halfExtents Vec3) OBB {
	return NewOBB(center, halfExtents, [3]Vec3{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	})
}

// OBBFromMat3 builds an OBB taking the columns of orientation as its three
// axes. Mat3 stores columns contiguously, so this is the counterpart of
// Mat3FromCols and of rotation matrices returned by the package.
func OBBFromMat3(center, halfExtents Vec3, orientation Mat3) OBB {
	c0, c1, c2 := orientation.Cols()
	return NewOBB(center, halfExtents, [3]Vec3{c0, c1, c2})
}

// OBBFromQuat builds an OBB whose axes are the rotated world axes of q. The
// quaternion must be a unit rotation; a non-unit quaternion yields
// non-orthonormal axes and is rejected by Validate.
func OBBFromQuat(center, halfExtents Vec3, q Quat) OBB {
	return OBBFromMat3(center, halfExtents, q.Mat4().Mat3())
}

// IsValid reports whether o is a well-defined OBB according to Validate.
func (o OBB) IsValid() bool {
	return o.Validate() == nil
}

// Validate checks the OBB definition as a strictly three-dimensional box.
//
// It returns a non-nil error when any component of Center, HalfExtents or
// Axes is NaN or infinite, when a half extent is zero or negative, or when
// Axes are not orthonormal (non-unit or not pairwise perpendicular). Zero
// half extents describe legitimate degenerate shapes; if you want to intersect
// boxes that are flat, thin rods or points, use ValidateDegenerate and
// TryOBBIntersectionDegenerate instead.
func (o OBB) Validate() error {
	return o.validate(false)
}

// ValidatePositive is an alias for Validate. It exists for callers that want
// to state explicitly that zero half extents are rejected.
func (o OBB) ValidatePositive() error {
	return o.Validate()
}

// ValidateDegenerate is like Validate but accepts zero half extents,
// collapsing that dimension of the box. Negative half extents are still
// rejected.
func (o OBB) ValidateDegenerate() error {
	return o.validate(true)
}

func (o OBB) validate(allowZero bool) error {
	if err := obbCheckFiniteVec3(o.Center, "Center"); err != nil {
		return err
	}
	if err := obbCheckFiniteVec3(o.HalfExtents, "HalfExtents"); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		switch {
		case o.HalfExtents[i] < 0:
			return fmt.Errorf("mgl64: OBB HalfExtents[%d] is %v, must not be negative", i, o.HalfExtents[i])
		case o.HalfExtents[i] == 0 && !allowZero:
			return fmt.Errorf("mgl64: OBB HalfExtents[%d] is zero; use ValidateDegenerate for a flat/rod/point box", i)
		}
	}
	for i := 0; i < 3; i++ {
		if err := obbCheckFiniteVec3(o.Axes[i], fmt.Sprintf("Axes[%d]", i)); err != nil {
			return err
		}
	}
	for i := 0; i < 3; i++ {
		lenSqr := o.Axes[i].LenSqr()
		if math.Abs(lenSqr-1) > obbOrthoLengthEps {
			return fmt.Errorf("mgl64: OBB Axes[%d] is not unit length (|axis|^2 = %v)", i, lenSqr)
		}
		for j := i + 1; j < 3; j++ {
			if d := math.Abs(o.Axes[i].Dot(o.Axes[j])); d > obbOrthoDotEps {
				return fmt.Errorf("mgl64: OBB Axes[%d] and Axes[%d] are not perpendicular (|dot| = %v)", i, j, d)
			}
		}
	}
	return nil
}

func obbCheckFiniteVec3(v Vec3, name string) error {
	for i := 0; i < 3; i++ {
		if f := v[i]; math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("mgl64: OBB %s[%d] is non-finite (%v)", name, i, f)
		}
	}
	return nil
}

// Corners returns the eight corners of the box. Corner bit i selects the
// sign along Axes[i], with sign -1 for a zero bit and +1 for a set bit.
func (o OBB) Corners() [8]Vec3 {
	var corners [8]Vec3
	for bits := 0; bits < 8; bits++ {
		p := o.Center
		for i := 0; i < 3; i++ {
			sign := -1.0
			if bits&(1<<uint(i)) != 0 {
				sign = 1.0
			}
			p = p.Add(o.Axes[i].Mul(sign * o.HalfExtents[i]))
		}
		corners[bits] = p
	}
	return corners
}

// AABB returns the axis-aligned bounding box of o as its minimum and maximum
// corners in world coordinates.
func (o OBB) AABB() (min, max Vec3) {
	corners := o.Corners()
	min = corners[0]
	max = corners[0]
	for _, c := range corners[1:] {
		for i := 0; i < 3; i++ {
			if c[i] < min[i] {
				min[i] = c[i]
			}
			if c[i] > max[i] {
				max[i] = c[i]
			}
		}
	}
	return min, max
}

// Intersects reports whether o and other overlap or exactly touch. It panics
// if either box is malformed.
func (a OBB) Intersects(b OBB) bool {
	hit, err := obbIntersection(a, b, false)
	if err != nil {
		panic(err)
	}
	return hit.Intersects
}

// Intersect returns whether a and b intersect (exact contact counts), the
// minimum separation/penetration axis and the unsigned depth along it:
// penetration depth while intersecting (0 at contact), separation gap while
// disjoint. It panics if either box is malformed.
func (a OBB) Intersect(b OBB) (intersects bool, axis Vec3, depth float64) {
	hit, err := obbIntersection(a, b, false)
	if err != nil {
		panic(err)
	}
	return hit.Intersects, hit.Axis, hit.Depth
}

// Intersection returns the detailed OBBHit for the pair. It panics if either
// box is malformed; use TryIntersection to receive the error instead.
func (a OBB) Intersection(b OBB) OBBHit {
	hit, err := obbIntersection(a, b, false)
	if err != nil {
		panic(err)
	}
	return hit
}

// TryIntersection is Intersection without panics: a malformed box yields a
// non-nil error and a zero OBBHit. Zero half extents are rejected; use
// TryIntersectionDegenerate for flat/rod/point boxes.
func (a OBB) TryIntersection(b OBB) (OBBHit, error) {
	return obbIntersection(a, b, false)
}

// TryIntersectionDegenerate is like TryIntersection but accepts zero half
// extents. Negative half extents and non-orthonormal frames still error.
func (a OBB) TryIntersectionDegenerate(b OBB) (OBBHit, error) {
	return obbIntersection(a, b, true)
}

// OBBIntersects reports whether a and b overlap or exactly touch. It panics
// if either box is malformed.
func OBBIntersects(a, b OBB) bool { return a.Intersects(b) }

// OBBIntersect is the package-level form of OBB.Intersect.
func OBBIntersect(a, b OBB) (intersects bool, axis Vec3, depth float64) {
	return a.Intersect(b)
}

// OBBIntersection is the package-level form of OBB.Intersection.
func OBBIntersection(a, b OBB) OBBHit { return a.Intersection(b) }

// TryOBBIntersection is the package-level form of OBB.TryIntersection.
func TryOBBIntersection(a, b OBB) (OBBHit, error) { return obbIntersection(a, b, false) }

// TryOBBIntersectionDegenerate is the package-level form of
// OBB.TryIntersectionDegenerate.
func TryOBBIntersectionDegenerate(a, b OBB) (OBBHit, error) {
	return obbIntersection(a, b, true)
}

// obbIntersection runs the fifteen-axis SAT. It validates first so malformed
// input can never leak NaN into the comparisons.
func obbIntersection(a, b OBB, allowZero bool) (OBBHit, error) {
	if err := a.validate(allowZero); err != nil {
		return OBBHit{}, err
	}
	if err := b.validate(allowZero); err != nil {
		return OBBHit{}, err
	}

	// d = center A - center B gives the sign convention "Axis points from B
	// towards A".
	d := a.Center.Sub(b.Center)

	geomScale := 1.0 + d.Len() +
		a.HalfExtents[0] + a.HalfExtents[1] + a.HalfExtents[2] +
		b.HalfExtents[0] + b.HalfExtents[1] + b.HalfExtents[2]
	touchTol := obbTouchSlackFactor * geomScale

	bestOverlap := math.Inf(1)
	var bestAxis Vec3

	consider := func(axis Vec3) {
		dist := math.Abs(d.Dot(axis))

		ra := a.HalfExtents[0]*math.Abs(a.Axes[0].Dot(axis)) +
			a.HalfExtents[1]*math.Abs(a.Axes[1].Dot(axis)) +
			a.HalfExtents[2]*math.Abs(a.Axes[2].Dot(axis))
		rb := b.HalfExtents[0]*math.Abs(b.Axes[0].Dot(axis)) +
			b.HalfExtents[1]*math.Abs(b.Axes[1].Dot(axis)) +
			b.HalfExtents[2]*math.Abs(b.Axes[2].Dot(axis))

		overlap := ra + rb - dist
		if overlap < bestOverlap {
			bestOverlap = overlap
			bestAxis = axis
		}
	}

	// Three face normals of A and three of B.
	for i := 0; i < 3; i++ {
		consider(a.Axes[i])
	}
	for i := 0; i < 3; i++ {
		consider(b.Axes[i])
	}

	// Nine edge cross products. Near-parallel edge pairs produce an
	// unstable/zero cross and are redundant; skip them.
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			cross := a.Axes[i].Cross(b.Axes[j])
			length := cross.Len()
			if length <= obbParallelEps {
				continue
			}
			consider(cross.Mul(1.0 / length))
		}
	}

	if math.IsInf(bestOverlap, 1) {
		// Unreachable for validated boxes (the six face axes always run);
		// kept as a defensive boundary.
		return OBBHit{}, errors.New("mgl64: OBB intersection produced no separating axes")
	}

	// Orient the chosen axis from B towards A for a deterministic sign.
	if d.Dot(bestAxis) < 0 {
		bestAxis = bestAxis.Mul(-1)
	}

	signed := bestOverlap
	if math.Abs(signed) <= touchTol {
		signed = 0
	}

	hit := OBBHit{
		Intersects:  signed >= 0,
		Axis:        bestAxis,
		SignedDepth: signed,
	}
	switch {
	case signed > 0:
		hit.Penetration = signed
		hit.Depth = signed
	case signed < 0:
		hit.Separation = -signed
		hit.Depth = -signed
	default:
		hit.Depth = 0
	}
	return hit, nil
}
