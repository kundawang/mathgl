// This file is generated from mgl32/dualquat.go; DO NOT EDIT

// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"math"
)

// DualQuat represents a rigid transformation (rotation + translation) as a
// dual quaternion of the form Real + ε·Dual, where ε is the dual unit with
// ε² = 0. Dual quaternions are commonly used for skeletal animation: blending
// them (see DualQuatBlend) avoids the joint-collapse ("candy wrapper")
// artifacts of linear matrix blending.
//
// A rigid transformation with unit rotation quaternion r and translation
// vector t is represented as:
//
//	Real = r
//	Dual = 0.5 * (0, t) * r
//
// A DualQuat built with NewDualQuat or DualQuatFromMat4, or returned by
// DualQuatBlend, is always "unit" (|Real| == 1, Real ⟂ Dual) and therefore
// always describes a rigid transformation. Functions in this package never
// return an error; degenerate inputs follow the documented conventions of
// each function (generally: fall back to the identity transformation).
type DualQuat struct {
	// Real is the rotational part of the transformation.
	Real Quat
	// Dual encodes the translation (combined with Real).
	Dual Quat
}

// DualQuatIdent returns the dual quaternion identity, representing no
// rotation and no translation.
func DualQuatIdent() DualQuat {
	return DualQuat{QuatIdent(), Quat{0, Vec3{0, 0, 0}}}
}

// NewDualQuat builds a rigid transformation dual quaternion from a rotation
// quaternion and a translation vector.
//
// The rotation does not need to be unit length; it is normalized first, so a
// non-unit (but non-zero) rotation still yields a rigid transformation. A
// zero-length rotation is treated as the identity rotation, matching the
// convention of Quat.Normalize.
func NewDualQuat(rotation Quat, translation Vec3) DualQuat {
	r := rotation.Normalize()
	// Dual = 0.5 * (0, t) * r
	t := Quat{0, translation}
	return DualQuat{r, t.Mul(r).Scale(0.5)}
}

// DualQuatFromMat4 extracts a rigid transformation from a homogeneous 4x4
// matrix.
//
// The matrix is assumed to consist of a rotation and a translation, such as
// the result of Translate3D(tx,ty,tz).Mul4(HomogRotate3D(...)). If the matrix
// also contains scaling or shear, those components are discarded by
// convention: the 3x3 part is orthonormalized (Gram-Schmidt on its columns)
// before the rotation is extracted, and the translation is read from the last
// column. If the 3x3 part is singular (e.g. zero scale on an axis) the
// resulting rotation is unspecified.
func DualQuatFromMat4(m Mat4) DualQuat {
	c0 := Vec3{m[0], m[1], m[2]}.Normalize()
	c1 := Vec3{m[4], m[5], m[6]}
	c1 = c1.Sub(c0.Mul(c1.Dot(c0))).Normalize()
	c2 := c0.Cross(c1)

	rot := Mat4ToQuat(Mat4{
		c0[0], c0[1], c0[2], 0,
		c1[0], c1[1], c1[2], 0,
		c2[0], c2[1], c2[2], 0,
		0, 0, 0, 1,
	})
	return NewDualQuat(rot, Vec3{m[12], m[13], m[14]})
}

// Rotation returns the (unit) rotation quaternion of the transformation.
func (dq DualQuat) Rotation() Quat {
	return dq.Real.Normalize()
}

// Translation returns the translation vector of the transformation.
//
// The result is only meaningful for a rigid (unit) dual quaternion, e.g. one
// built with NewDualQuat or normalized with Normalize.
func (dq DualQuat) Translation() Vec3 {
	// t = 2 * Dual * Real*
	return dq.Dual.Mul(dq.Real.Conjugate()).Scale(2).V
}

// Add adds two dual quaternions component-wise. It is mostly useful as a
// building block for blending; the sum of two rigid dual quaternions is
// generally not rigid until normalized.
func (dq1 DualQuat) Add(dq2 DualQuat) DualQuat {
	return DualQuat{dq1.Real.Add(dq2.Real), dq1.Dual.Add(dq2.Dual)}
}

// Scale multiplies every element of both parts by a constant factor. Like
// Add, it is a building block for blending; scaling a rigid dual quaternion
// generally makes it non-rigid until normalized.
func (dq DualQuat) Scale(c float64) DualQuat {
	return DualQuat{dq.Real.Scale(c), dq.Dual.Scale(c)}
}

// Mul composes two rigid transformations: dq1.Mul(dq2) is the transformation
// that applies dq2 first and dq1 second, mirroring the order of Mat4.Mul4.
func (dq1 DualQuat) Mul(dq2 DualQuat) DualQuat {
	return DualQuat{
		dq1.Real.Mul(dq2.Real),
		dq1.Real.Mul(dq2.Dual).Add(dq1.Dual.Mul(dq2.Real)),
	}
}

// Conjugate returns the quaternion conjugate of both parts. For a rigid
// (unit) dual quaternion this equals its inverse.
func (dq DualQuat) Conjugate() DualQuat {
	return DualQuat{dq.Real.Conjugate(), dq.Dual.Conjugate()}
}

// Inverse returns the inverse rigid transformation, i.e.
// dq.Mul(dq.Inverse()) == DualQuatIdent().
//
// The dual quaternion is normalized first, so the result is correct even if
// dq has drifted slightly away from unit length (e.g. after blending).
func (dq DualQuat) Inverse() DualQuat {
	return dq.Normalize().Conjugate()
}

// Normalize scales the dual quaternion so that |Real| == 1, making it a
// rigid transformation again after operations such as Add and Scale.
//
// Conventions for degenerate input: if |Real| is zero, NaN, or infinite
// (e.g. blending weights that cancel out), or if the dual part contains
// non-finite values (e.g. a NaN translation), the identity transformation is
// returned.
func (dq DualQuat) Normalize() DualQuat {
	length := dq.Real.Len()
	if length == 0 || length == InfPos || math.IsNaN(float64(length)) {
		return DualQuatIdent()
	}
	d := dq.Dual
	for _, e := range [4]float64{d.W, d.V[0], d.V[1], d.V[2]} {
		if math.IsNaN(float64(e)) || math.IsInf(float64(e), 0) {
			return DualQuatIdent()
		}
	}
	return dq.Scale(1 / length)
}

// Mat4 converts the rigid transformation back to a homogeneous 4x4 matrix,
// the inverse of DualQuatFromMat4. The dual quaternion is normalized first,
// so the result is always a pure rotation + translation matrix.
func (dq DualQuat) Mat4() Mat4 {
	dq = dq.Normalize()
	m := dq.Real.Mat4()
	m[12], m[13], m[14] = dq.Translation().Elem()
	return m
}

// TransformPoint applies the rigid transformation to a point, equivalent to
// converting to a matrix with Mat4 and multiplying. The dual quaternion is
// normalized first, so blending results can be used directly.
func (dq DualQuat) TransformPoint(p Vec3) Vec3 {
	dq = dq.Normalize()
	return dq.Real.Rotate(p).Add(dq.Translation())
}

// ApproxEqualThreshold returns whether both parts of the dual quaternions are
// approximately equal with the given tolerance, element-wise.
//
// Note that, like quaternions, a dual quaternion and its negation describe
// the same transformation; use TransformPoint or Mat4 to compare
// transformations instead of representations.
func (dq1 DualQuat) ApproxEqualThreshold(dq2 DualQuat, epsilon float64) bool {
	return dq1.Real.ApproxEqualThreshold(dq2.Real, epsilon) &&
		dq1.Dual.ApproxEqualThreshold(dq2.Dual, epsilon)
}

// DualQuatBlend blends several bone transformations with the given weights
// (dual quaternion linear blending, DLB) and returns the blended rigid
// transformation. This is the dual-quaternion replacement for linear blend
// skinning with matrices and does not collapse joints under rotation.
//
// Antipodal ambiguity (q and -q describe the same rotation) is resolved by
// flipping the sign of every dual quaternion whose real part points opposite
// to the first one, so blending a transformation with its negated twin is a
// no-op. The weighted sum is normalized, so the weights do not need to sum to
// one; in particular a single bone with weight 1 (or any non-zero weight)
// yields exactly that bone's transformation.
//
// Conventions for degenerate input: it panics if no bones are given or if
// len(weights) != len(dqs). Zero-weight bones contribute nothing; if all
// weights are zero (or the blend otherwise degenerates to a zero or
// non-finite result, e.g. through NaN inputs), the identity transformation is
// returned.
func DualQuatBlend(dqs []DualQuat, weights []float64) DualQuat {
	if len(dqs) == 0 {
		panic("DualQuatBlend: no dual quaternions given")
	}
	if len(dqs) != len(weights) {
		panic("DualQuatBlend: mismatched dual quaternion and weight counts")
	}

	reference := dqs[0].Real
	sum := DualQuat{}
	for i, dq := range dqs {
		if dq.Real.Dot(reference) < 0 {
			dq = dq.Scale(-1)
		}
		sum = sum.Add(dq.Scale(weights[i]))
	}
	return sum.Normalize()
}
