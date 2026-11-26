// Copyright 2024 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"math"
	"math/rand"
	"testing"
)

func obbNear(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func obbVecNear(a, b Vec3, eps float64) bool {
	for i := 0; i < 3; i++ {
		if !obbNear(a[i], b[i], eps) {
			return false
		}
	}
	return true
}

func obbAxisMatch(a, b Vec3, eps float64) bool {
	return math.Abs(math.Abs(a.Dot(b))-1) <= eps
}

func obbIdentityAxes() [3]Vec3 {
	return [3]Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
}

// The exact acceptance numbers from the task: side-2 cubes (half extent 1),
// centers 2 apart touch with depth 0; 2.01 apart leave a 0.01 gap.
func TestOBBTouchAndSmallGap(t *testing.T) {
	cubeAt := func(x float64) OBB {
		return NewAxisAlignedOBB(Vec3{x, 0, 0}, Vec3{1, 1, 1})
	}

	touch := OBBIntersection(cubeAt(0), cubeAt(2))
	if !touch.Intersects {
		t.Fatalf("cubes 2 apart must exactly touch, got disjoint (sep=%v axis=%v)", touch.Separation, touch.Axis)
	}
	if touch.Depth != 0 || touch.Penetration != 0 || touch.Separation != 0 || touch.SignedDepth != 0 {
		t.Fatalf("touch must report zero depth, got %+v", touch)
	}
	if !obbAxisMatch(touch.Axis, Vec3{1, 0, 0}, 1e-12) {
		t.Fatalf("touch axis must be X, got %v", touch.Axis)
	}
	// Axis is oriented from B towards A: A is at 0, B at 2, so -X.
	if touch.Axis.Dot(Vec3{-1, 0, 0}) < 0.999 {
		t.Fatalf("touch axis sign should point from B to A (-X), got %v", touch.Axis)
	}

	gap := OBBIntersection(cubeAt(0), cubeAt(2.01))
	if gap.Intersects {
		t.Fatalf("cubes 2.01 apart must be disjoint, got pen=%v", gap.Penetration)
	}
	if !obbNear(gap.Separation, 0.01, 1e-9) {
		t.Fatalf("separation = %.12f, expected 0.01", gap.Separation)
	}
	if !obbNear(gap.Depth, 0.01, 1e-9) || !obbNear(gap.SignedDepth, -0.01, 1e-9) {
		t.Fatalf("depth/signed = %v / %v, expected 0.01 / -0.01", gap.Depth, gap.SignedDepth)
	}
	if !obbAxisMatch(gap.Axis, Vec3{1, 0, 0}, 1e-12) {
		t.Fatalf("gap axis must be X, got %v", gap.Axis)
	}

	// A 1.5 center distance gives a 0.5 penetration.
	pen := OBBIntersection(cubeAt(0), cubeAt(1.5))
	if !pen.Intersects || !obbNear(pen.Penetration, 0.5, 1e-12) {
		t.Fatalf("expected penetration 0.5, got intersects=%v %+v", pen.Intersects, pen)
	}
}

// Face-axis separation on every world axis.
func TestOBBFaceAxes(t *testing.T) {
	for _, tc := range []struct {
		name string
		off  Vec3
		axis Vec3
	}{
		{"x", Vec3{2.5, 0, 0}, Vec3{1, 0, 0}},
		{"y", Vec3{0, -2.25, 0}, Vec3{0, 1, 0}},
		{"z", Vec3{0, 0, 3}, Vec3{0, 0, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{1, 1, 1})
			b := NewAxisAlignedOBB(tc.off, Vec3{0.5, 0.5, 0.5})
			hit := OBBIntersection(a, b)
			if hit.Intersects {
				t.Fatalf("expected separation on %s", tc.name)
			}
			// gap = |off_axis| - (1 + 0.5); off magnitude chosen per axis.
			want := math.Abs(tc.off.Dot(tc.axis)) - 1.5
			if !obbNear(hit.Separation, want, 1e-12) {
				t.Fatalf("sep=%v want %v", hit.Separation, want)
			}
			if !obbAxisMatch(hit.Axis, tc.axis, 1e-12) {
				t.Fatalf("axis=%v want %v", hit.Axis, tc.axis)
			}
		})
	}
}

// Swapping the boxes must preserve depth and flip the axis sign.
func TestOBBSymmetry(t *testing.T) {
	r := rand.New(rand.NewSource(424242))
	for i := 0; i < 200; i++ {
		a := obbRandom(r, 1.0)
		b := obbRandom(r, 1.0)
		h1 := a.Intersection(b)
		h2 := b.Intersection(a)
		if h1.Intersects != h2.Intersects {
			t.Fatalf("bool asymmetry on iter %d", i)
		}
		if !obbNear(h1.SignedDepth, h2.SignedDepth, 1e-9) {
			t.Fatalf("depth asymmetry on iter %d: %v vs %v", i, h1.SignedDepth, h2.SignedDepth)
		}
		if !obbAxisMatch(h1.Axis, h2.Axis, 1e-6) {
			// Near-ties can make the two call orders pick different minimizers;
			// the swapped witness must still attain the same minimum overlap.
			ca, ra := obbProjectedInterval(a, h2.Axis)
			cb, rb := obbProjectedInterval(b, h2.Axis)
			if ov := ra + rb - math.Abs(ca-cb); ov-h1.SignedDepth > 1e-7 {
				t.Fatalf("axis mismatch on iter %d: %v vs %v", i, h1.Axis, h2.Axis)
			}
			continue
		}
		// Opposite center ordering must flip the oriented sign.
		if math.Abs(h1.Axis.Dot(h2.Axis)+1) > 1e-6 {
			t.Fatalf("axis sign should reverse under swap, %v vs %v (dot=%v)", h1.Axis, h2.Axis, h1.Axis.Dot(h2.Axis))
		}
	}
}

// Parallel frames, including near-parallel rotations where cross axes shrink.
func TestOBBParallelAndNearParallel(t *testing.T) {
	axes := [3]Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	a := NewOBB(Vec3{0, 0, 0}, Vec3{3, 0.4, 0.4}, axes)

	for _, angle := range []float64{0, 1e-13, 1e-9, 1e-6, 1e-3, 0.05} {
		c, s := math.Cos(angle), math.Sin(angle)
		bAxes := [3]Vec3{
			{c, s, 0},
			{-s, c, 0},
			{0, 0, 1},
		}
		// Place B just overlapping A along X-ish direction.
		b := NewOBB(Vec3{4.0, 0, 0}, Vec3{1, 0.4, 0.4}, bAxes)
		hit := OBBIntersection(a, b)
		// At angle 0: x gap = 4 - (3+1) = 0 exact touch. Small rotations add
		// tiny projections; result must stay finite and near contact.
		if math.IsNaN(hit.Depth) || math.IsInf(hit.Depth, 0) {
			t.Fatalf("angle %v: non-finite depth", angle)
		}
		if hit.Separation > 0.05 || hit.Penetration > 0.05 {
			t.Fatalf("angle %v moved too far from contact: %+v", angle, hit)
		}
	}

	// Exactly parallel translated rods have a known X gap.
	b := NewOBB(Vec3{4.5, 0, 0}, Vec3{1, 0.4, 0.4}, axes)
	hit := OBBIntersection(a, b)
	if hit.Intersects || !obbNear(hit.Separation, 0.5, 1e-12) {
		t.Fatalf("parallel rods: want gap 0.5, got %+v", hit)
	}
}

// Extremely thin boxes must not produce NaN or false results.
func TestOBBThinBoxes(t *testing.T) {
	thin := 1e-12
	a := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{2, 2, thin})
	b := NewAxisAlignedOBB(Vec3{0, 0, 2*thin + 1e-6}, Vec3{1, 1, thin})
	hit := OBBIntersection(a, b)
	if hit.Intersects {
		t.Fatalf("thin boxes should be separated by 1e-6, got %+v", hit)
	}
	if !obbNear(hit.Separation, 1e-6, 1e-10) {
		t.Fatalf("thin gap = %v, want 1e-6", hit.Separation)
	}

	// Coplanar thin sheets at the same Z intersect over their XY area.
	c := NewAxisAlignedOBB(Vec3{0.5, 0, 0}, Vec3{0.5, 0.5, thin})
	if !OBBIntersects(a, c) {
		t.Fatal("coplanar thin sheets should intersect")
	}

	// Zero half extent is rejected by the strict API but supported as an
	// explicit degenerate box.
	flat := OBB{Center: Vec3{0, 0, 0}, HalfExtents: Vec3{1, 1, 0}, Axes: obbIdentityAxes()}
	if err := flat.Validate(); err == nil {
		t.Fatal("strict Validate must reject zero half extents")
	}
	if err := flat.ValidatePositive(); err == nil {
		t.Fatal("ValidatePositive must reject zero half extents")
	}
	if err := flat.ValidateDegenerate(); err != nil {
		t.Fatalf("ValidateDegenerate should accept zero half extents: %v", err)
	}
	if _, err := TryOBBIntersection(flat, goodCube()); err == nil {
		t.Fatal("strict intersection must error on zero half extents")
	}
	if !mustDegenerate(t, flat, goodCube()).Intersects {
		t.Fatal("flat box through cube center should intersect via degenerate path")
	}
	above := NewAxisAlignedOBB(Vec3{0, 0, 0.6}, Vec3{0.5, 0.5, 0.5})
	if mustDegenerate(t, flat, above).Intersects {
		t.Fatal("flat box below cube should be disjoint (z gap 0.1)")
	}
}

func goodCube() OBB { return NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{0.5, 0.5, 0.5}) }

func mustDegenerate(t *testing.T, a, b OBB) OBBHit {
	t.Helper()
	hit, err := TryOBBIntersectionDegenerate(a, b)
	if err != nil {
		t.Fatalf("degenerate intersection: %v", err)
	}
	return hit
}

// Wildly different scales must stay numerically sane.
func TestOBBDifferentScales(t *testing.T) {
	giant := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{1e6, 1e6, 1e6})
	tiny := NewAxisAlignedOBB(Vec3{1e6 + 0.5, 0, 0}, Vec3{1, 1, 1})
	hit := OBBIntersection(giant, tiny)
	// X: center 1000000.5, radii 1000000 + 1 -> overlap 0.5.
	if !hit.Intersects || !obbNear(hit.Penetration, 0.5, 1e-6) {
		t.Fatalf("scale 1e6: want pen 0.5, got %+v", hit)
	}

	micro := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{1e-6, 1e-6, 1e-6})
	micro2 := NewAxisAlignedOBB(Vec3{2.01e-6, 0, 0}, Vec3{1e-6, 1e-6, 1e-6})
	g := OBBIntersection(micro, micro2)
	if g.Intersects || !obbNear(g.Separation, 0.01e-6, 1e-14) {
		t.Fatalf("micro gap want 1e-8, got %+v", g)
	}
}

// The original bug story: a rotated long box whose AABB overlaps a trigger
// while the OBBs are actually disjoint.
func TestOBBRotatedLongBoxFalseAABBTrigger(t *testing.T) {
	s := math.Sqrt(0.5)
	long := NewOBB(Vec3{0, 0, 0}, Vec3{3, 0.5, 0.5}, [3]Vec3{
		{s, s, 0},
		{-s, s, 0},
		{0, 0, 1},
	})
	trigger := NewAxisAlignedOBB(Vec3{2.7, 2.7, 0}, Vec3{0.5, 0.5, 0.5})

	lmin, lmax := long.AABB()
	tmin, tmax := trigger.AABB()
	aabbOverlap := lmin[0] <= tmax[0] && tmin[0] <= lmax[0] &&
		lmin[1] <= tmax[1] && tmin[1] <= lmax[1] &&
		lmin[2] <= tmax[2] && tmin[2] <= lmax[2]
	if !aabbOverlap {
		t.Fatal("test setup: expected AABBs to overlap so AABB test would false-trigger")
	}

	hit := OBBIntersection(long, trigger)
	if hit.Intersects {
		t.Fatalf("rotated long box should not touch trigger, pen=%v axis=%v", hit.Penetration, hit.Axis)
	}
	// Along the long axis: center projection 2*2.7*s, radii 3 + s*0.5.
	wantGap := 2*2.7*s - (3 + s*0.5 + s*0.5)
	if !obbNear(hit.Separation, wantGap, 1e-9) {
		t.Fatalf("gap=%v want analytic %v (full result %+v)", hit.Separation, wantGap, hit)
	}
	if !obbAxisMatch(hit.Axis, Vec3{s, s, 0}, 1e-9) {
		t.Fatalf("separation axis should be the long axis, got %v", hit.Axis)
	}

	// Sliding the trigger in must deepen penetration; at the analytic
	// boundary p = (3 + sqrt(0.5)) / sqrt(2) it must exactly touch.
	inside := NewAxisAlignedOBB(Vec3{2.6, 2.6, 0}, Vec3{0.5, 0.5, 0.5})
	ihit := OBBIntersection(long, inside)
	if !ihit.Intersects || !obbNear(ihit.Penetration, 0.0301515, 1e-6) {
		t.Fatalf("2.6 placement should penetrate by ~0.0302, got %+v", ihit)
	}

	pContact := (3 + s) / (2 * s)
	contact := NewAxisAlignedOBB(Vec3{pContact, pContact, 0}, Vec3{0.5, 0.5, 0.5})
	chit := OBBIntersection(long, contact)
	if !chit.Intersects || chit.Depth != 0 {
		t.Fatalf("boundary placement should exactly touch, got %+v", chit)
	}
}

// Coincident boxes: center delta is zero, so axis sign is arbitrary but the
// result must stay finite, intersecting and symmetric.
func TestOBBCoincident(t *testing.T) {
	o := NewOBB(Vec3{1, 2, 3}, Vec3{2, 0.5, 1}, [3]Vec3{
		{0, 1, 0}, {-1, 0, 0}, {0, 0, 1},
	})
	h := OBBIntersection(o, o)
	if !h.Intersects || math.IsNaN(h.Depth) || h.Penetration <= 0 {
		t.Fatalf("identical boxes must intersect with positive depth: %+v", h)
	}
	// Canonical SAT depth for identical boxes is twice the smallest half
	// extent, on the corresponding face axis.
	if !obbNear(h.Penetration, 1.0, 1e-12) {
		t.Fatalf("pen=%v want 1.0 (2*0.5)", h.Penetration)
	}
}

// Explicit malformed-input handling.
func TestOBBInvalidInputs(t *testing.T) {
	good := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{1, 1, 1})

	badAxesNonOrtho := [3]Vec3{{1, 0, 0}, Vec3{1, 1, 0}.Normalize(), {0, 0, 1}}
	badCases := map[string]OBB{
		"negative half": {
			Center:      Vec3{0, 0, 0},
			HalfExtents: Vec3{1, -1, 1},
			Axes:        obbIdentityAxes(),
		},
		"zero half": {
			Center:      Vec3{0, 0, 0},
			HalfExtents: Vec3{1, 0, 1},
			Axes:        obbIdentityAxes(),
		},
		"nan center": {
			Center:      Vec3{math.NaN(), 0, 0},
			HalfExtents: Vec3{1, 1, 1},
			Axes:        obbIdentityAxes(),
		},
		"inf half": {
			Center:      Vec3{0, 0, 0},
			HalfExtents: Vec3{1, math.Inf(1), 1},
			Axes:        obbIdentityAxes(),
		},
		"nan axis": {
			Center:      Vec3{},
			HalfExtents: Vec3{1, 1, 1},
			Axes: [3]Vec3{
				{math.NaN(), 0, 0}, {0, 1, 0}, {0, 0, 1},
			},
		},
		"non unit axis": {
			Center:      Vec3{},
			HalfExtents: Vec3{1, 1, 1},
			Axes: [3]Vec3{
				{2, 0, 0}, {0, 1, 0}, {0, 0, 1},
			},
		},
		"non orthogonal axes": {
			Center:      Vec3{},
			HalfExtents: Vec3{1, 1, 1},
			Axes:        badAxesNonOrtho,
		},
	}

	for name, bad := range badCases {
		t.Run(name, func(t *testing.T) {
			if err := bad.Validate(); err == nil {
				t.Fatal("Validate must reject malformed OBB")
			}
			if bad.IsValid() {
				t.Fatal("IsValid must be false")
			}
			if _, err := TryOBBIntersection(bad, good); err == nil {
				t.Fatal("TryOBBIntersection must return an error")
			}
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("OBBIntersection must panic on malformed OBB")
					}
				}()
				_ = bad.Intersection(good)
			}()
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("NewOBB must panic on malformed axes")
					}
				}()
				_ = NewOBB(bad.Center, bad.HalfExtents, bad.Axes)
			}()
		})
	}

	// The second box is validated too.
	if _, err := TryOBBIntersection(good, badCases["negative half"]); err == nil {
		t.Fatal("second OBB must be validated")
	}
}

// Mat3 constructor and Corners/AABB helpers.
func TestOBBMat3AndCorners(t *testing.T) {
	o := OBBFromMat3(Vec3{1, 2, 3}, Vec3{1, 1, 1}, Mat3FromCols(
		Vec3{1, 0, 0}, Vec3{0, 1, 0}, Vec3{0, 0, 1},
	))
	if o.Center != (Vec3{1, 2, 3}) {
		t.Fatalf("center not preserved: %v", o.Center)
	}
	min, max := o.AABB()
	if !obbVecNear(min, Vec3{0, 1, 2}, 1e-12) || !obbVecNear(max, Vec3{2, 3, 4}, 1e-12) {
		t.Fatalf("AABB = %v..%v", min, max)
	}
	corners := o.Corners()
	if len(corners) != 8 {
		t.Fatal("want 8 corners")
	}

	// Quaternion orientation must produce a validated frame matching the
	// equivalent Z rotation.
	q := QuatRotate(math.Pi/4, Vec3{0, 0, 1})
	qo := OBBFromQuat(Vec3{}, Vec3{1, 1, 1}, q)
	s := math.Sqrt(0.5)
	if !obbVecNear(qo.Axes[0], Vec3{s, s, 0}, 1e-12) ||
		!obbVecNear(qo.Axes[1], Vec3{-s, s, 0}, 1e-12) {
		t.Fatalf("quat axes wrong: %v", qo.Axes)
	}
}

// ---------------------------------------------------------------------------
// Independent oracle: project the eight generated corners onto each of the
// fifteen SAT axes and derive radii/centers from the projected intervals.
// This shares no radius formula with the implementation.
// ---------------------------------------------------------------------------

func obbOracleAxes(a, b OBB) []Vec3 {
	axes := make([]Vec3, 0, 15)
	for i := 0; i < 3; i++ {
		axes = append(axes, a.Axes[i], b.Axes[i])
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			c := a.Axes[i].Cross(b.Axes[j])
			if l := c.Len(); l > obbParallelEps {
				axes = append(axes, c.Mul(1/l))
			}
		}
	}
	return axes
}

func obbProjectedInterval(o OBB, axis Vec3) (center, radius float64) {
	corners := o.Corners()
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, c := range corners {
		p := c.Dot(axis)
		lo = math.Min(lo, p)
		hi = math.Max(hi, p)
	}
	return (lo + hi) * 0.5, (hi - lo) * 0.5
}

func obbOracle(a, b OBB) (minOverlap float64, bestAxis Vec3, separating int) {
	minOverlap = math.Inf(1)
	for _, ax := range obbOracleAxes(a, b) {
		ca, ra := obbProjectedInterval(a, ax)
		cb, rb := obbProjectedInterval(b, ax)
		ov := ra + rb - math.Abs(ca-cb)
		if ov < minOverlap {
			minOverlap = ov
			bestAxis = ax
		}
		if ov < -1e-9 {
			separating++
		}
	}
	return
}

func obbRandomUnit(r *rand.Rand) Vec3 {
	v := Vec3{r.NormFloat64(), r.NormFloat64(), r.NormFloat64()}
	for v.LenSqr() < 1e-6 {
		v = Vec3{r.NormFloat64(), r.NormFloat64(), r.NormFloat64()}
	}
	return v.Normalize()
}

func obbRandomFrame(r *rand.Rand) [3]Vec3 {
	u := obbRandomUnit(r)
	v := obbRandomUnit(r)
	v = v.Sub(u.Mul(v.Dot(u)))
	for v.LenSqr() < 0.05 {
		v = obbRandomUnit(r)
		v = v.Sub(u.Mul(v.Dot(u)))
	}
	v = v.Normalize()
	w := u.Cross(v).Normalize()
	return [3]Vec3{u, v, w}
}

func obbRandom(r *rand.Rand, centerScale float64) OBB {
	he := Vec3{
		math.Pow(10, r.Float64()*6-3),
		math.Pow(10, r.Float64()*6-3),
		math.Pow(10, r.Float64()*6-3),
	}
	if r.Float64() < 0.15 {
		he[r.Intn(3)] = math.Pow(10, -12+r.Float64()*3) // extremely thin
	}
	axes := obbRandomFrame(r)
	radius := he[0] + he[1] + he[2]
	c := obbRandomUnit(r).Mul(r.Float64() * radius * 1.5 * centerScale)
	return OBB{Center: c, HalfExtents: he, Axes: axes}
}

// Randomized differential test against the corner oracle plus the MTV
// move-to-contact invariant.
func TestOBBRandomOracleAndMTV(t *testing.T) {
	r := rand.New(rand.NewSource(13579))
	for i := 0; i < 3000; i++ {
		a := obbRandom(r, 1)
		b := obbRandom(r, 1)

		want, oracleAxis, nSep := obbOracle(a, b)
		hit := OBBIntersection(a, b)

		scale := 1 + a.Center.Len() + b.Center.Len() +
			a.HalfExtents[0] + a.HalfExtents[1] + a.HalfExtents[2] +
			b.HalfExtents[0] + b.HalfExtents[1] + b.HalfExtents[2]
		eps := 1e-8 * scale

		wantSigned := want
		if math.Abs(want) <= obbTouchSlackFactor*scale {
			wantSigned = 0
		}
		if !obbNear(float64(hit.SignedDepth), wantSigned, eps) {
			t.Fatalf("iter %d: signed=%v oracle=%v eps=%v", i, hit.SignedDepth, wantSigned, eps)
		}
		wantIntersects := wantSigned >= 0
		if hit.Intersects != wantIntersects {
			t.Fatalf("iter %d: intersects=%v want %v (oracle=%v)", i, hit.Intersects, wantIntersects, want)
		}
		if !obbAxisMatch(hit.Axis, oracleAxis, 1e-6) {
			// Tolerate near-ties: the chosen axis must itself attain the
			// oracle minimum within numerical slack.
			ca, ra := obbProjectedInterval(a, hit.Axis)
			cb, rb := obbProjectedInterval(b, hit.Axis)
			if ov := ra + rb - math.Abs(ca-cb); ov-want > eps {
				t.Fatalf("iter %d: axis %v does not match oracle %v (ov=%v want=%v)", i, hit.Axis, oracleAxis, ov, want)
			}
		}

		if hit.Intersects {
			// Translating A by the MTV must bring the axis to exact contact.
			moved := a
			moved.Center = a.Center.Add(hit.Axis.Mul(hit.Penetration))
			h2 := moved.Intersection(b)
			if math.Abs(h2.SignedDepth) > 1e-6*scale {
				t.Fatalf("iter %d: MTV move did not reach contact, signed=%v scale=%v", i, h2.SignedDepth, scale)
			}
		} else if nSep == 1 {
			// With one separating axis, closing the reported gap contacts.
			moved := a
			moved.Center = a.Center.Add(hit.Axis.Mul(-hit.Separation))
			h2 := moved.Intersection(b)
			if math.Abs(h2.SignedDepth) > 1e-6*scale {
				t.Fatalf("iter %d: gap close did not reach contact, signed=%v", i, h2.SignedDepth)
			}
		}
	}
}

// Search configurations where the winning axis is an edge cross product, so
// the nine cross axes are demonstrably exercised end to end.
func TestOBBCrossAxisWinner(t *testing.T) {
	r := rand.New(rand.NewSource(97531))
	foundDisjoint, foundPenetrating := false, false
	for i := 0; i < 20000 && !(foundDisjoint && foundPenetrating); i++ {
		a := obbRandom(r, 1)
		// Bias towards elongated boxes so edge-edge axes matter.
		a.HalfExtents[r.Intn(3)] *= 8
		b := obbRandom(r, 1)
		b.HalfExtents[r.Intn(3)] *= 8

		want, oracleAxis, _ := obbOracle(a, b)
		faceAxis := false
		faceMin := math.Inf(1)
		for k := 0; k < 3; k++ {
			for _, f := range [2]Vec3{a.Axes[k], b.Axes[k]} {
				if math.Abs(math.Abs(oracleAxis.Dot(f))-1) < 1e-6 {
					faceAxis = true
				}
				ca, ra := obbProjectedInterval(a, f)
				cb, rb := obbProjectedInterval(b, f)
				if ov := ra + rb - math.Abs(ca-cb); ov < faceMin {
					faceMin = ov
				}
			}
		}
		if faceAxis {
			continue
		}
		// Require a strict margin over every face axis so the cross-axis win
		// is unique and not a floating-point tie.
		if faceMin-want < 1e-7 {
			continue
		}

		hit := OBBIntersection(a, b)
		if !obbAxisMatch(hit.Axis, oracleAxis, 1e-7) {
			t.Fatalf("cross-axis case %d: axis %v vs oracle %v", i, hit.Axis, oracleAxis)
		}
		if hit.Intersects && want > 1e-9 {
			foundPenetrating = true
			moved := a
			moved.Center = a.Center.Add(hit.Axis.Mul(hit.Penetration))
			h2 := moved.Intersection(b)
			if math.Abs(h2.SignedDepth) > 1e-6 {
				t.Fatalf("cross-axis MTV did not contact: %v", h2.SignedDepth)
			}
		} else if !hit.Intersects && want < -1e-9 {
			foundDisjoint = true
		}
	}
	if !foundDisjoint {
		t.Fatal("never found a disjoint cross-axis winner")
	}
	if !foundPenetrating {
		t.Fatal("never found a penetrating cross-axis winner")
	}
}
