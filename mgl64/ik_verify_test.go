// This file is generated from mgl32/ik_verify_test.go; DO NOT EDIT

// Copyright 2026 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// ikVerifyVecClose reports whether every component of a and b differs by less
// than eps (absolute comparison, deliberately independent of the package
// Epsilon relative rules).
func ikVerifyVecClose(a, b Vec3, eps float64) bool {
	for i := range a {
		d := a[i] - b[i]
		if d < 0 {
			d = -d
		}
		if d > eps {
			return false
		}
	}
	return true
}

func ikVerifyFloatClose(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

// ikVerifyBoneLengths checks both preserved bone lengths with a tolerance
// scaled to each bone.
func ikVerifyBoneLengths(t *testing.T, r TwoBoneIKResult) {
	t.Helper()
	gotUpper := r.Elbow.Sub(r.Root).Len()
	gotLower := r.End.Sub(r.Elbow).Len()
	if tol := 1e-4*r.UpperLength + 1e-6; !ikVerifyFloatClose(gotUpper, r.UpperLength, tol) {
		t.Errorf("upper bone length = %v, want %v (tol %v)", gotUpper, r.UpperLength, tol)
	}
	if tol := 1e-4*r.LowerLength + 1e-6; !ikVerifyFloatClose(gotLower, r.LowerLength, tol) {
		t.Errorf("lower bone length = %v, want %v (tol %v)", gotLower, r.LowerLength, tol)
	}
}

// The task's canonical oracle: root 0, both lengths 1, target (1,1,0), bend
// towards +X must yield elbow (1,0,0) and an end effector on the target.
func TestIKVerifyCanonicalBendPlusX(t *testing.T) {
	const eps = 1e-6
	root := Vec3{0, 0, 0}
	target := Vec3{1, 1, 0}
	hint := Vec3{1, 0, 0}

	r, err := SolveTwoBoneIK(root, 1, 1, target, hint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ikVerifyVecClose(r.Elbow, Vec3{1, 0, 0}, eps) {
		t.Errorf("elbow = %v, want approximately (1,0,0)", r.Elbow)
	}
	if !ikVerifyVecClose(r.End, target, eps) {
		t.Errorf("end = %v, want target %v", r.End, target)
	}
	if r.BendHintDegenerate {
		t.Errorf("+X hint against a 45-degree axis must not be degenerate")
	}
	ikVerifyBoneLengths(t, r)
	e2, end2, err := TwoBoneIK(root, 1, 1, target, hint)
	if err != nil {
		t.Fatalf("TwoBoneIK error: %v", err)
	}
	if !ikVerifyVecClose(e2, r.Elbow, eps) || !ikVerifyVecClose(end2, r.End, eps) {
		t.Errorf("TwoBoneIK = %v,%v disagree with %v,%v", e2, end2, r.Elbow, r.End)
	}

	// The pose wrapper must carry the same elbow and a translation at End.
	e3, pose, err := SolveTwoBoneIKPose(root, 1, 1, target, hint)
	if err != nil {
		t.Fatalf("SolveTwoBoneIKPose error: %v", err)
	}
	if !ikVerifyVecClose(e3, r.Elbow, eps) {
		t.Errorf("pose wrapper elbow = %v, want %v", e3, r.Elbow)
	}
	posePos := Vec3{pose[12], pose[13], pose[14]}
	if !ikVerifyVecClose(posePos, r.End, eps) {
		t.Errorf("pose translation = %v, want %v", posePos, r.End)
	}
}

// Far targets are clamped to MaxReach; strict mode reports the sentinel.
func TestIKVerifyUnreachableFar(t *testing.T) {
	root := Vec3{0, 0, 0}
	far := Vec3{10, 0, 0}

	r, err := SolveTwoBoneIK(root, 1, 1, far, Vec3{0, 1, 0})
	if err != nil {
		t.Fatalf("default clamp solve error: %v", err)
	}
	if !r.Clamped || r.Reachable {
		t.Errorf("flags = Clamped:%v Reachable:%v, want true,false", r.Clamped, r.Reachable)
	}
	if !ikVerifyVecClose(r.End, Vec3{2, 0, 0}, 1e-5) {
		t.Errorf("clamped end = %v, want (2,0,0)", r.End)
	}
	if !ikVerifyVecClose(r.Elbow, Vec3{1, 0, 0}, 1e-5) {
		t.Errorf("fully extended elbow = %v, want (1,0,0)", r.Elbow)
	}
	ikVerifyBoneLengths(t, r)

	// Explicit clamp=true through the variadic form.
	if _, err := SolveTwoBoneIK(root, 1, 1, far, Vec3{0, 1, 0}, true); err != nil {
		t.Errorf("explicit clamp=true returned error: %v", err)
	}

	_, err = SolveTwoBoneIK(root, 1, 1, far, Vec3{0, 1, 0}, false)
	if !errors.Is(err, ErrIKTargetUnreachable) {
		t.Errorf("strict variadic err = %v, want ErrIKTargetUnreachable", err)
	}
	_, err = SolveTwoBoneIKStrict(root, 1, 1, far, Vec3{0, 1, 0})
	if !errors.Is(err, ErrIKTargetUnreachable) {
		t.Errorf("SolveTwoBoneIKStrict err = %v, want ErrIKTargetUnreachable", err)
	}

	// The Clamp field alias must enable clamping too.
	alias, err := SolveTwoBoneIKWithOptions(TwoBoneIKOptions{
		Root: root, UpperLength: 1, LowerLength: 1, Target: far, BendHint: Vec3{0, 1, 0}, Clamp: true,
	})
	if err != nil || !alias.Clamped {
		t.Errorf("Clamp field alias: clamped=%v err=%v", alias.Clamped, err)
	}
}

// Targets closer than |l1-l2| clamp to MinReach; equal bones at the root fold
// with the elbow along the hint.
func TestIKVerifyUnreachableNear(t *testing.T) {
	// Unequal lengths 3 and 1: MinReach 2.
	r, err := SolveTwoBoneIK(Vec3{}, 3, 1, Vec3{0.5, 0, 0}, Vec3{0, 1, 0})
	if err != nil {
		t.Fatalf("near clamp error: %v", err)
	}
	if !r.Clamped {
		t.Errorf("expected Clamped for too-near target")
	}
	if !ikVerifyFloatClose(r.End.Len(), 2, 1e-5) {
		t.Errorf("clamped end distance = %v, want 2", r.End.Len())
	}
	ikVerifyBoneLengths(t, r)

	_, err = SolveTwoBoneIK(Vec3{}, 3, 1, Vec3{0.5, 0, 0}, Vec3{0, 1, 0}, false)
	if !errors.Is(err, ErrIKTargetUnreachable) {
		t.Errorf("near strict err = %v, want ErrIKTargetUnreachable", err)
	}

	// Equal bones, target exactly at root, bend +X: elbow at +X.
	z, err := SolveTwoBoneIK(Vec3{}, 1, 1, Vec3{0, 0, 0}, Vec3{1, 0, 0})
	if err != nil {
		t.Fatalf("zero reach error: %v", err)
	}
	if !ikVerifyVecClose(z.Elbow, Vec3{1, 0, 0}, 1e-5) {
		t.Errorf("folded elbow = %v, want (1,0,0)", z.Elbow)
	}
	if !ikVerifyVecClose(z.End, Vec3{0, 0, 0}, 1e-5) {
		t.Errorf("folded end = %v, want zero", z.End)
	}
	ikVerifyBoneLengths(t, z)
}

// Equal/unequal lengths over a deterministic table plus randomized reachable
// targets must preserve lengths and hit the target.
func TestIKVerifyLengthsAndReachGrid(t *testing.T) {
	lengths := [][2]float64{{1, 1}, {2, 0.5}, {0.5, 2}, {3, 0.25}, {0.25, 3}, {1.7, 1.3}}
	rng := rand.New(rand.NewSource(20261127))

	for _, ll := range lengths {
		l1, l2 := ll[0], ll[1]
		minR := Abs(l1 - l2)
		maxR := l1 + l2
		for i := 0; i < 60; i++ {
			// Uniform direction on the sphere.
			theta := rng.Float64() * 2 * math.Pi
			cosPhi := 2*rng.Float64() - 1
			sinPhi := math.Sqrt(1 - cosPhi*cosPhi)
			dir := Vec3{
				float64(sinPhi * math.Cos(theta)),
				float64(sinPhi * math.Sin(theta)),
				float64(cosPhi),
			}
			// Stay clear of the exact boundaries so target == End tightly.
			span := maxR - minR
			d := minR + float64(0.02)*span + rng.Float64()*0.96*span
			root := Vec3{float64(rng.NormFloat64()) * 50, float64(rng.NormFloat64()) * 50, float64(rng.NormFloat64()) * 50}
			target := root.Add(dir.Mul(d))
			hint := Vec3{float64(rng.NormFloat64()), float64(rng.NormFloat64()), float64(rng.NormFloat64())}

			r, err := SolveTwoBoneIK(root, l1, l2, target, hint)
			if err != nil {
				t.Fatalf("l1=%v l2=%v d=%v unexpected error: %v", l1, l2, d, err)
			}
			if !ikVerifyVecClose(r.End, target, 2e-4*ikMaxFloat(l1, l2)+1e-5) {
				t.Errorf("l1=%v l2=%v d=%v end %v not target %v", l1, l2, d, r.End, target)
			}
			ikVerifyBoneLengths(t, r)
			for _, c := range r.Elbow {
				if x := float64(c); math.IsNaN(x) || math.IsInf(x, 0) {
					t.Fatalf("non-finite elbow: %v", r.Elbow)
				}
			}
		}
	}
}

// Degenerate bend hints must be stable (or strict-error), never NaN/panic.
func TestIKVerifyDegenerateHint(t *testing.T) {
	target := Vec3{1, 1, 0}

	for name, hint := range map[string]Vec3{
		"zero":          {},
		"anti-parallel": {-3, -3, 0},
		"parallel":      {1, 1, 0},
		"huge":          {1e20, 1e20, 0},
	} {
		t.Run(name, func(t *testing.T) {
			r, err := SolveTwoBoneIK(Vec3{}, 1, 1, target, hint)
			if err != nil {
				t.Fatalf("fallback solve error: %v", err)
			}
			if !r.BendHintDegenerate {
				t.Errorf("expected BendHintDegenerate for hint %v", hint)
			}
			ikVerifyBoneLengths(t, r)
			if !ikVerifyVecClose(r.End, target, 1e-5) {
				t.Errorf("end = %v, want target", r.End)
			}
			// Deterministic.
			r2, _ := SolveTwoBoneIK(Vec3{}, 1, 1, target, hint)
			if r.Elbow != r2.Elbow {
				t.Errorf("non-deterministic fallback: %v vs %v", r.Elbow, r2.Elbow)
			}
		})
	}

	_, err := SolveTwoBoneIKWithOptions(TwoBoneIKOptions{
		UpperLength: 1, LowerLength: 1, Target: target, BendHint: Vec3{1, 1, 0},
		StrictBendPlane: true,
	})
	if !errors.Is(err, ErrIKBendPlaneDegenerate) {
		t.Errorf("strict degenerate err = %v, want ErrIKBendPlaneDegenerate", err)
	}
}

// Zero/negative lengths and non-finite inputs produce sentinel errors.
func TestIKVerifyInvalidInputs(t *testing.T) {
	nan := float64(math.NaN())
	inf := float64(math.Inf(1))

	lengthCases := []struct {
		l1, l2 float64
	}{
		{0, 1}, {1, 0}, {0, 0}, {-1, 1}, {1, -0.5}, {nan, 1}, {1, nan}, {inf, 1},
	}
	for _, c := range lengthCases {
		_, err := SolveTwoBoneIK(Vec3{}, c.l1, c.l2, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		if c.l1 <= 0 && ikFinite(c.l1) || c.l2 <= 0 && ikFinite(c.l2) {
			if !errors.Is(err, ErrIKInvalidLength) {
				t.Errorf("lengths %v,%v err=%v, want ErrIKInvalidLength", c.l1, c.l2, err)
			}
		} else {
			if !errors.Is(err, ErrIKNonFinite) {
				t.Errorf("lengths %v,%v err=%v, want ErrIKNonFinite", c.l1, c.l2, err)
			}
		}
	}

	nonFiniteVecs := []Vec3{
		{nan, 0, 0}, {0, inf, 0}, {0, 0, float64(math.Inf(-1))},
	}
	for _, v := range nonFiniteVecs {
		if _, err := SolveTwoBoneIK(v, 1, 1, Vec3{1, 0, 0}, Vec3{0, 1, 0}); !errors.Is(err, ErrIKNonFinite) {
			t.Errorf("root %v err=%v, want ErrIKNonFinite", v, err)
		}
		if _, err := SolveTwoBoneIK(Vec3{}, 1, 1, v, Vec3{0, 1, 0}); !errors.Is(err, ErrIKNonFinite) {
			t.Errorf("target %v err=%v, want ErrIKNonFinite", v, err)
		}
		if _, err := SolveTwoBoneIK(Vec3{}, 1, 1, Vec3{1, 0, 0}, v); !errors.Is(err, ErrIKNonFinite) {
			t.Errorf("hint %v err=%v, want ErrIKNonFinite", v, err)
		}
	}
}

// Approaching full extension must stay finite, preserve lengths and move
// smoothly (no elbow flip).
func TestIKVerifyFullExtensionSweep(t *testing.T) {
	s := NewTwoBoneIKSolver(1, 1)
	var prev Vec3
	maxJump := float64(0)
	for i := 0; i <= 100; i++ {
		x := 0.05 + float64(i)*(1.95/100)
		r, err := s.Solve(Vec3{}, Vec3{x, 1e-3, 0}, Vec3{0, 0, 1})
		if err != nil {
			t.Fatalf("x=%v error: %v", x, err)
		}
		for _, c := range r.Elbow {
			if f := float64(c); math.IsNaN(f) || math.IsInf(f, 0) {
				t.Fatalf("non-finite elbow at x=%v: %v", x, r.Elbow)
			}
		}
		ikVerifyBoneLengths(t, r)
		if i > 0 {
			if j := r.Elbow.Sub(prev).Len(); j > maxJump {
				maxJump = j
			}
		}
		prev = r.Elbow
	}
	if maxJump > 0.25 {
		t.Errorf("elbow jump %.4f too large across extension sweep", maxJump)
	}

	// Exact full extension: elbow on the axis at distance l1.
	r, err := SolveTwoBoneIK(Vec3{}, 1, 1, Vec3{2, 0, 0}, Vec3{0, 1, 0})
	if err != nil {
		t.Fatal(err)
	}
	if !ikVerifyVecClose(r.Elbow, Vec3{1, 0, 0}, 1e-5) {
		t.Errorf("extension elbow = %v, want (1,0,0)", r.Elbow)
	}
}

// Opposite hints give mirror elbows about the root-target axis, and every
// elbow lies on its hinted side.
func TestIKVerifyHintSideAndMirror(t *testing.T) {
	root := Vec3{}
	target := Vec3{1.2, 0.4, 0.2}
	hints := []Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}}

	for _, h := range hints {
		r, err := SolveTwoBoneIK(root, 1.3, 0.9, target, h)
		if err != nil {
			t.Fatal(err)
		}
		proj := h.Sub(r.Axis.Mul(h.Dot(r.Axis)))
		p, ok := ikStableUnit(proj)
		if !ok {
			continue
		}
		if dot := r.Elbow.Sub(root).Dot(p); dot <= 1e-5 {
			t.Errorf("hint %v: elbow not on hint side, dot=%v elbow=%v", h, dot, r.Elbow)
		}

		opp, err := SolveTwoBoneIK(root, 1.3, 0.9, target, h.Mul(-1))
		if err != nil {
			t.Fatal(err)
		}
		mid := r.Elbow.Add(opp.Elbow).Mul(0.5)
		// The midpoint of the two mirrors is the axial projection point.
		along := mid.Sub(root).Dot(r.Axis)
		axialPoint := root.Add(r.Axis.Mul(along))
		if !ikVerifyVecClose(mid, axialPoint, 1e-5) {
			t.Errorf("hint %v: mirror midpoint %v off axis (axial %v)", h, mid, axialPoint)
		}
		ikVerifyBoneLengths(t, opp)
	}
}

// The stateful solver must not jump when a strong hint decays to zero through
// the retained bend plane, and Reset clears the retained frame.
func TestIKVerifyStatefulHintDecayContinuity(t *testing.T) {
	s := NewTwoBoneIKSolver(1, 1)
	target := Vec3{1.2, 0.4, 0}
	scales := []float64{1, 1e-2, 1e-4, 1e-8, 0, 0, 1e-8}

	var prev Vec3
	for i, sc := range scales {
		r, err := s.Solve(Vec3{}, target, Vec3{0, sc, 0})
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			if j := r.Elbow.Sub(prev).Len(); j > 1e-5 {
				t.Errorf("step %d scale %v elbow jumped %v (prev %v now %v)", i, sc, j, prev, r.Elbow)
			}
		}
		prev = r.Elbow
	}
	if !s.HasPreviousFrame() {
		t.Error("expected retained frame")
	}
	s.Reset()
	if s.HasPreviousFrame() {
		t.Error("Reset did not clear frame")
	}

	// Bad lengths must not establish a frame.
	bad := NewTwoBoneIKSolver(0, 1)
	if _, err := bad.Solve(Vec3{}, target, Vec3{0, 1, 0}); err == nil {
		t.Error("expected error for zero length")
	}
	if bad.HasPreviousFrame() {
		t.Error("failed solve must not retain a frame")
	}
}

// Translating the root translates the whole solution.
func TestIKVerifyRootOffset(t *testing.T) {
	root := Vec3{10, -20, 30}
	r, err := SolveTwoBoneIK(root, 1, 1, root.Add(Vec3{1, 1, 0}), Vec3{1, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if !ikVerifyVecClose(r.Elbow, root.Add(Vec3{1, 0, 0}), 1e-6) {
		t.Errorf("offset elbow = %v, want %v", r.Elbow, root.Add(Vec3{1, 0, 0}))
	}
	ikVerifyBoneLengths(t, r)
}

// Rotation/Pose form a stable orthonormal end-effector frame at End.
func TestIKVerifyOrientationAndPose(t *testing.T) {
	r, err := SolveTwoBoneIK(Vec3{}, 1, 1, Vec3{1, 1, 0}, Vec3{1, 0, 0})
	if err != nil {
		t.Fatal(err)
	}

	distal, ok := ikStableUnit(r.End.Sub(r.Elbow))
	if !ok {
		t.Fatal("zero distal direction")
	}
	gotY := r.Rotation.Rotate(Vec3{0, 1, 0})
	if !ikVerifyVecClose(gotY, distal, 1e-5) {
		t.Errorf("rotation +Y = %v, want distal %v", gotY, distal)
	}
	if ql := math.Sqrt(float64(r.Rotation.Dot(r.Rotation))); !ikVerifyFloatClose(float64(ql), 1, 1e-5) {
		t.Errorf("quaternion norm = %v, want 1", ql)
	}

	cols := [3]Vec3{
		{r.Pose[0], r.Pose[1], r.Pose[2]},
		{r.Pose[4], r.Pose[5], r.Pose[6]},
		{r.Pose[8], r.Pose[9], r.Pose[10]},
	}
	for i, c := range cols {
		if !ikVerifyFloatClose(c.Len(), 1, 1e-5) {
			t.Errorf("pose col %d len = %v, want 1", i, c.Len())
		}
	}
	if d := cols[0].Dot(cols[1]); !ikVerifyFloatClose(d, 0, 1e-5) {
		t.Errorf("col0.col1 = %v, want 0", d)
	}
	if d := cols[1].Dot(cols[2]); !ikVerifyFloatClose(d, 0, 1e-5) {
		t.Errorf("col1.col2 = %v, want 0", d)
	}
	translation := Vec3{r.Pose[12], r.Pose[13], r.Pose[14]}
	if !ikVerifyVecClose(translation, r.End, 1e-6) {
		t.Errorf("pose translation %v != End %v", translation, r.End)
	}
}

// ExampleSolveTwoBoneIK demonstrates the canonical oracle and reach clamping.
func ExampleSolveTwoBoneIK() {
	clean := func(x float64) float64 {
		if Abs(x) < 1e-6 {
			return 0
		}
		return x
	}

	r, _ := SolveTwoBoneIK(Vec3{0, 0, 0}, 1, 1, Vec3{1, 1, 0}, Vec3{1, 0, 0})
	fmt.Printf("elbow=(%.1f, %.1f, %.1f)\n", clean(r.Elbow[0]), clean(r.Elbow[1]), clean(r.Elbow[2]))

	far, _ := SolveTwoBoneIK(Vec3{0, 0, 0}, 1, 1, Vec3{10, 0, 0}, Vec3{0, 1, 0})
	fmt.Printf("far-end=(%.1f, %.1f, %.1f) clamped=%v\n",
		clean(far.End[0]), clean(far.End[1]), clean(far.End[2]), far.Clamped)

	// Output:
	// elbow=(1.0, 0.0, 0.0)
	// far-end=(2.0, 0.0, 0.0) clamped=true
}
