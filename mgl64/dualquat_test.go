// This file is generated from mgl32/dualquat_test.go; DO NOT EDIT

// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"math"
	"testing"
)

const dqEpsilon = 1e-5

// mat4Approx and vec3Approx compare with a plain absolute tolerance, unlike
// FloatEqualThreshold which demands epsilon^2 agreement near zero.
func mat4Approx(a, b Mat4, tol float64) bool {
	for i := range a {
		if d := a[i] - b[i]; d > tol || d < -tol {
			return false
		}
	}
	return true
}

func vec3Approx(a, b Vec3, tol float64) bool {
	for i := range a {
		if d := a[i] - b[i]; d > tol || d < -tol {
			return false
		}
	}
	return true
}

func rigidMats() []Mat4 {
	return []Mat4{
		Ident4(),
		Translate3D(1, 2, 3),
		HomogRotate3DZ(math.Pi / 3),
		Translate3D(-4, 0.5, 7).Mul4(HomogRotate3D(1.1, Vec3{1, 2, 3}.Normalize())),
		HomogRotate3DX(-2.4).Mul4(Translate3D(0, -3, 0.25)),
	}
}

func TestDualQuatMat4RoundTrip(t *testing.T) {
	for _, m := range rigidMats() {
		got := DualQuatFromMat4(m).Mat4()
		if !mat4Approx(m, got, dqEpsilon) {
			t.Errorf("round trip mismatch:\nwant %v\ngot  %v", m, got)
		}
	}
}

func TestDualQuatTransformPointMatchesMatrix(t *testing.T) {
	points := []Vec3{{0, 0, 0}, {1, 0, 0}, {1, 2, 3}, {-0.5, 4, -2}}
	for _, m := range rigidMats() {
		dq := DualQuatFromMat4(m)
		for _, p := range points {
			want := m.Mul4x1(p.Vec4(1)).Vec3()
			got := dq.TransformPoint(p)
			if !vec3Approx(want, got, dqEpsilon) {
				t.Errorf("TransformPoint(%v) = %v, want %v", p, got, want)
			}
		}
	}
}

func TestDualQuatMulMatchesMatrixComposition(t *testing.T) {
	m1 := Translate3D(1, 2, 3).Mul4(HomogRotate3DY(0.7))
	m2 := Translate3D(-2, 0, 5).Mul4(HomogRotate3DZ(-1.2))

	dq := DualQuatFromMat4(m1).Mul(DualQuatFromMat4(m2))
	want := m1.Mul4(m2)
	if got := dq.Mat4(); !mat4Approx(want, got, dqEpsilon) {
		t.Errorf("composition mismatch:\nwant %v\ngot  %v", want, got)
	}
}

func TestDualQuatInverse(t *testing.T) {
	for _, m := range rigidMats() {
		dq := DualQuatFromMat4(m)
		id := dq.Mul(dq.Inverse())
		if got := id.Mat4(); !mat4Approx(Ident4(), got, dqEpsilon) {
			t.Errorf("dq * dq.Inverse() != identity:\n%v", got)
		}

		p := Vec3{3, -1, 2}
		if got := dq.Inverse().TransformPoint(dq.TransformPoint(p)); !vec3Approx(p, got, dqEpsilon) {
			t.Errorf("inverse transform did not undo %v: got %v", p, got)
		}
	}
}

// Two-bone twist: one bone holds the identity, the other twists 120° about
// the arm's X axis. Linear matrix blending collapses vertices at the joint
// toward the axis; the dual quaternion blend must keep a vertex on the axis
// exactly in place and rotate an off-axis vertex by half the twist.
func TestDualQuatBlendTwoBoneTwist(t *testing.T) {
	rest := DualQuatIdent()
	twist := NewDualQuat(QuatRotate(2*math.Pi/3, Vec3{1, 0, 0}), Vec3{0, 0, 0})

	blended := DualQuatBlend([]DualQuat{rest, twist}, []float64{0.5, 0.5})

	// A vertex on the twist axis must not collapse toward the joint.
	if got := blended.TransformPoint(Vec3{2, 0, 0}); !vec3Approx(Vec3{2, 0, 0}, got, dqEpsilon) {
		t.Errorf("on-axis vertex moved to %v, want (2,0,0)", got)
	}

	// An off-axis vertex sees half the twist: 60° about X.
	want := QuatRotate(math.Pi/3, Vec3{1, 0, 0}).Rotate(Vec3{1, 1, 0})
	if got := blended.TransformPoint(Vec3{1, 1, 0}); !vec3Approx(want, got, dqEpsilon) {
		t.Errorf("off-axis vertex = %v, want %v", got, want)
	}
}

func TestDualQuatBlendTranslation(t *testing.T) {
	stay := DualQuatIdent()
	move := NewDualQuat(QuatIdent(), Vec3{2, 0, 0})

	blended := DualQuatBlend([]DualQuat{stay, move}, []float64{0.5, 0.5})
	if got := blended.TransformPoint(Vec3{1, 1, 1}); !vec3Approx(Vec3{2, 1, 1}, got, dqEpsilon) {
		t.Errorf("blended translation moved point to %v, want (2,1,1)", got)
	}
}

// A bone and its antipodal twin (-q, same transformation) must blend to the
// same transformation as the bone alone.
func TestDualQuatBlendAntipodal(t *testing.T) {
	bone := DualQuatFromMat4(Translate3D(1, -2, 3).Mul4(HomogRotate3D(0.9, Vec3{0, 1, 1}.Normalize())))
	antipodal := bone.Scale(-1)

	p := Vec3{2, 3, -4}
	want := bone.TransformPoint(p)

	blended := DualQuatBlend([]DualQuat{bone, antipodal}, []float64{0.5, 0.5})
	if got := blended.TransformPoint(p); !vec3Approx(want, got, dqEpsilon) {
		t.Errorf("antipodal blend = %v, want %v", got, want)
	}

	// Same with the antipodal twin listed first.
	blended = DualQuatBlend([]DualQuat{antipodal, bone}, []float64{0.5, 0.5})
	if got := blended.TransformPoint(p); !vec3Approx(want, got, dqEpsilon) {
		t.Errorf("reordered antipodal blend = %v, want %v", got, want)
	}
}

// A single bone with weight 1 (or any non-zero weight) must reproduce that
// bone's transformation exactly, and a zero-weight bone must not contribute.
func TestDualQuatBlendSingleBoneAndZeroWeight(t *testing.T) {
	bone := DualQuatFromMat4(Translate3D(3, 1, -2).Mul4(HomogRotate3DX(0.4)))
	other := DualQuatFromMat4(Translate3D(-5, 0, 1).Mul4(HomogRotate3DY(2.0)))
	p := Vec3{1, -2, 0.5}
	want := bone.TransformPoint(p)

	for _, blend := range []DualQuat{
		DualQuatBlend([]DualQuat{bone}, []float64{1}),
		DualQuatBlend([]DualQuat{bone}, []float64{3.7}),
		DualQuatBlend([]DualQuat{bone, other}, []float64{1, 0}),
		DualQuatBlend([]DualQuat{other, bone}, []float64{0, 1}),
	} {
		if got := blend.TransformPoint(p); !vec3Approx(want, got, dqEpsilon) {
			t.Errorf("single-bone blend = %v, want %v", got, want)
		}
	}
}

// Degenerate blends (all-zero weights, NaN, Inf) fall back to the identity
// transformation by convention.
func TestDualQuatBlendDegenerateWeights(t *testing.T) {
	bone := DualQuatFromMat4(Translate3D(1, 2, 3))
	p := Vec3{4, 5, 6}

	for name, blend := range map[string]DualQuat{
		"zero weights": DualQuatBlend([]DualQuat{bone, bone}, []float64{0, 0}),
		"NaN weight":   DualQuatBlend([]DualQuat{bone}, []float64{float64(math.NaN())}),
		"Inf weight":   DualQuatBlend([]DualQuat{bone}, []float64{float64(math.Inf(1))}),
	} {
		if got := blend.TransformPoint(p); !vec3Approx(p, got, dqEpsilon) {
			t.Errorf("%s: got %v, want identity transform of %v", name, got, p)
		}
	}
}

func TestDualQuatBlendPanicsOnBadInput(t *testing.T) {
	bone := DualQuatIdent()
	for name, f := range map[string]func(){
		"no bones":        func() { DualQuatBlend(nil, nil) },
		"mismatched lens": func() { DualQuatBlend([]DualQuat{bone}, []float64{1, 0}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected panic", name)
				}
			}()
			f()
		}()
	}
}

// Non-unit and zero rotations are normalized on construction; a zero
// rotation is treated as the identity rotation.
func TestNewDualQuatNonUnitRotation(t *testing.T) {
	unit := NewDualQuat(QuatRotate(0.8, Vec3{0, 0, 1}), Vec3{1, 2, 3})
	scaled := NewDualQuat(QuatRotate(0.8, Vec3{0, 0, 1}).Scale(5), Vec3{1, 2, 3})
	if got, want := scaled.Mat4(), unit.Mat4(); !mat4Approx(want, got, dqEpsilon) {
		t.Errorf("scaled rotation gave different transform:\nwant %v\ngot  %v", want, got)
	}

	zero := NewDualQuat(Quat{0, Vec3{0, 0, 0}}, Vec3{1, 2, 3})
	if got, want := zero.Mat4(), Translate3D(1, 2, 3); !mat4Approx(want, got, dqEpsilon) {
		t.Errorf("zero rotation:\nwant %v\ngot  %v", want, got)
	}
}

// Scale and shear in a matrix are discarded by convention; only the rigid
// part (orthonormalized rotation + translation) is kept.
func TestDualQuatFromMat4DropsScaleAndShear(t *testing.T) {
	m := Translate3D(1, 2, 3).Mul4(HomogRotate3DZ(0.5)).Mul4(Scale3D(2, 2, 2))
	want := Translate3D(1, 2, 3).Mul4(HomogRotate3DZ(0.5))
	if got := DualQuatFromMat4(m).Mat4(); !mat4Approx(want, got, dqEpsilon) {
		t.Errorf("scaled matrix:\nwant %v\ngot  %v", want, got)
	}

	shear := Translate3D(-1, 0, 2).Mul4(Mat4{
		1, 0, 0, 0,
		0.5, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	})
	dq := DualQuatFromMat4(shear)
	if got := dq.Translation(); !vec3Approx(Vec3{-1, 0, 2}, got, dqEpsilon) {
		t.Errorf("shear matrix translation = %v, want (-1,0,2)", got)
	}
	// The extracted rotation must be a valid orthonormal rotation.
	if l := dq.Rotation().Len(); Abs(l-1) > dqEpsilon {
		t.Errorf("shear matrix rotation not unit length: %v", l)
	}
}

// A degenerate dual part (or non-finite input) normalizes to the identity
// transformation instead of propagating NaN.
func TestDualQuatNormalizeDegenerate(t *testing.T) {
	nan := float64(math.NaN())
	inf := float64(math.Inf(1))
	zero := DualQuat{}

	for name, dq := range map[string]DualQuat{
		"zero":       zero,
		"NaN real":   {Quat{nan, Vec3{0, 0, 0}}, Quat{}},
		"Inf real":   {Quat{inf, Vec3{0, 0, 0}}, Quat{}},
		"NaN dual":   {QuatIdent(), Quat{nan, Vec3{nan, nan, nan}}},
		"zero dual":  {QuatIdent(), Quat{}},
		"deg. blend": zero.Add(zero),
	} {
		n := dq.Normalize()
		if name == "zero dual" {
			if !n.ApproxEqualThreshold(DualQuatIdent(), dqEpsilon) {
				t.Errorf("%s: got %v, want identity", name, n)
			}
			continue
		}
		p := Vec3{1, 2, 3}
		if got := n.TransformPoint(p); !vec3Approx(p, got, dqEpsilon) {
			t.Errorf("%s: TransformPoint = %v, want identity", name, got)
		}
	}
}
