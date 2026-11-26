// Copyright 2024 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl32

import (
	"math"
	"math/rand"
	"testing"
)

func obbNear(a, b, eps float32) bool { return Abs(a-b) <= eps }

func obbVecNear(a, b Vec3, eps float32) bool {
	for i := 0; i < 3; i++ {
		if !obbNear(a[i], b[i], eps) {
			return false
		}
	}
	return true
}

func obbAxisMatch(a, b Vec3, eps float32) bool {
	return Abs(Abs(a.Dot(b))-1) <= eps
}

func obbIdentityAxes() [3]Vec3 {
	return [3]Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
}

// The exact acceptance numbers from the task: side-2 cubes (half extent 1),
// centers 2 apart touch with depth 0; 2.01 apart leave a 0.01 gap.
func TestOBBTouchAndSmallGap(t *testing.T) {
	cubeAt := func(x float32) OBB {
		return NewAxisAlignedOBB(Vec3{x, 0, 0}, Vec3{1, 1, 1})
	}

	touch := OBBIntersection(cubeAt(0), cubeAt(2))
	if !touch.Intersects {
		t.Fatalf("cubes 2 apart must exactly touch, got disjoint (sep=%v axis=%v)", touch.Separation, touch.Axis)
	}
	if touch.Depth != 0 || touch.Penetration != 0 || touch.Separation != 0 || touch.SignedDepth != 0 {
		t.Fatalf("touch must report zero depth, got %+v", touch)
	}
	if !obbAxisMatch(touch.Axis, Vec3{1, 0, 0}, 1e-6) {
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
	if !obbNear(gap.Separation, 0.01, 1e-5) {
		t.Fatalf("separation = %.8f, expected 0.01", gap.Separation)
	}
	if !obbNear(gap.Depth, 0.01, 1e-5) || !obbNear(gap.SignedDepth, -0.01, 1e-5) {
		t.Fatalf("depth/signed = %v / %v, expected 0.01 / -0.01", gap.Depth, gap.SignedDepth)
	}
	if !obbAxisMatch(gap.Axis, Vec3{1, 0, 0}, 1e-6) {
		t.Fatalf("gap axis must be X, got %v", gap.Axis)
	}

	// A 1.5 center distance gives a 0.5 penetration.
	pen := OBBIntersection(cubeAt(0), cubeAt(1.5))
	if !pen.Intersects || !obbNear(pen.Penetration, 0.5, 1e-6) {
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
			want := Abs(tc.off.Dot(tc.axis)) - 1.5
			if !obbNear(hit.Separation, want, 1e-6) {
				t.Fatalf("sep=%v want %v", hit.Separation, want)
			}
			if !obbAxisMatch(hit.Axis, tc.axis, 1e-6) {
				t.Fatalf("axis=%v want %v", hit.Axis, tc.axis)
			}
		})
	}
}

// Swapping the boxes must preserve depth and the axis line.
func TestOBBSymmetry(t *testing.T) {
	r := rand.New(rand.NewSource(424242))
	for i := 0; i < 300; i++ {
		a := obbRandom(r)
		b := obbRandom(r)
		h1 := a.Intersection(b)
		h2 := b.Intersection(a)
		if h1.Intersects != h2.Intersects {
			t.Fatalf("bool asymmetry on iter %d", i)
		}
		if !obbNear(h1.SignedDepth, h2.SignedDepth, 1e-4) {
			t.Fatalf("depth asymmetry on iter %d: %v vs %v", i, h1.SignedDepth, h2.SignedDepth)
		}
		if !obbAxisMatch(h1.Axis, h2.Axis, 1e-4) {
			t.Fatalf("axis mismatch on iter %d: %v vs %v", i, h1.Axis, h2.Axis)
		}
		if math.Abs(float64(h1.Axis.Dot(h2.Axis))+1) > 1e-3 {
			t.Fatalf("axis sign should reverse under swap, dot=%v", h1.Axis.Dot(h2.Axis))
		}
	}
}

// Parallel frames, including near-parallel rotations where cross axes shrink.
func TestOBBParallelAndNearParallel(t *testing.T) {
	axes := [3]Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	a := NewOBB(Vec3{0, 0, 0}, Vec3{3, 0.4, 0.4}, axes)

	for _, angle := range []float64{0, 1e-7, 1e-5, 1e-3, 0.05} {
		c, s := float32(math.Cos(angle)), float32(math.Sin(angle))
		bAxes := [3]Vec3{
			{c, s, 0},
			{-s, c, 0},
			{0, 0, 1},
		}
		b := NewOBB(Vec3{4.0, 0, 0}, Vec3{1, 0.4, 0.4}, bAxes)
		hit := OBBIntersection(a, b)
		if math.IsNaN(float64(hit.Depth)) || math.IsInf(float64(hit.Depth), 0) {
			t.Fatalf("angle %v: non-finite depth", angle)
		}
		if hit.Separation > 0.05 || hit.Penetration > 0.05 {
			t.Fatalf("angle %v moved too far from contact: %+v", angle, hit)
		}
	}

	b := NewOBB(Vec3{4.5, 0, 0}, Vec3{1, 0.4, 0.4}, axes)
	hit := OBBIntersection(a, b)
	if hit.Intersects || !obbNear(hit.Separation, 0.5, 1e-6) {
		t.Fatalf("parallel rods: want gap 0.5, got %+v", hit)
	}
}

// Extremely thin boxes must not produce NaN or false results.
func TestOBBThinBoxes(t *testing.T) {
	thin := float32(1e-6) // float32 cannot resolve 1e-12 relative to unit boxes
	a := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{2, 2, thin})
	b := NewAxisAlignedOBB(Vec3{0, 0, 2*thin + 1e-4}, Vec3{1, 1, thin})
	hit := OBBIntersection(a, b)
	if hit.Intersects {
		t.Fatalf("thin boxes should be separated by 1e-4, got %+v", hit)
	}
	if !obbNear(hit.Separation, 1e-4, 1e-7) {
		t.Fatalf("thin gap = %v, want 1e-4", hit.Separation)
	}

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
	if _, err := TryOBBIntersection(flat, goodCube32()); err == nil {
		t.Fatal("strict intersection must error on zero half extents")
	}
	if !mustDegenerate32(t, flat, goodCube32()).Intersects {
		t.Fatal("flat box through cube center should intersect via degenerate path")
	}
	above := NewAxisAlignedOBB(Vec3{0, 0, 0.6}, Vec3{0.5, 0.5, 0.5})
	if mustDegenerate32(t, flat, above).Intersects {
		t.Fatal("flat box below cube should be disjoint (z gap 0.1)")
	}
}

func goodCube32() OBB { return NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{0.5, 0.5, 0.5}) }

func mustDegenerate32(t *testing.T, a, b OBB) OBBHit {
	t.Helper()
	hit, err := TryOBBIntersectionDegenerate(a, b)
	if err != nil {
		t.Fatalf("degenerate intersection: %v", err)
	}
	return hit
}

// Wildly different scales must stay numerically sane.
func TestOBBDifferentScales(t *testing.T) {
	giant := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{1e3, 1e3, 1e3})
	tiny := NewAxisAlignedOBB(Vec3{1e3 + 0.5, 0, 0}, Vec3{1, 1, 1})
	hit := OBBIntersection(giant, tiny)
	if !hit.Intersects || !obbNear(hit.Penetration, 0.5, 1e-3) {
		t.Fatalf("scale 1e3: want pen 0.5, got %+v", hit)
	}

	micro := NewAxisAlignedOBB(Vec3{0, 0, 0}, Vec3{1e-3, 1e-3, 1e-3})
	micro2 := NewAxisAlignedOBB(Vec3{2.01e-3, 0, 0}, Vec3{1e-3, 1e-3, 1e-3})
	g := OBBIntersection(micro, micro2)
	if g.Intersects || !obbNear(g.Separation, 0.01e-3, 1e-9) {
		t.Fatalf("micro gap want 1e-5, got %+v", g)
	}
}

// The original bug story: a rotated long box whose AABB overlaps a trigger
// while the OBBs are actually disjoint.
func TestOBBRotatedLongBoxFalseAABBTrigger(t *testing.T) {
	s := float32(math.Sqrt(0.5))
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
	wantGap := float32(2*2.7*math.Sqrt(0.5)) - (3 + 2*0.5*s)
	if !obbNear(hit.Separation, wantGap, 1e-5) {
		t.Fatalf("gap=%v want analytic %v (full result %+v)", hit.Separation, wantGap, hit)
	}
	if !obbAxisMatch(hit.Axis, Vec3{s, s, 0}, 1e-5) {
		t.Fatalf("separation axis should be the long axis, got %v", hit.Axis)
	}

	pContact := (3 + s) / (2 * s)
	contact := NewAxisAlignedOBB(Vec3{pContact, pContact, 0}, Vec3{0.5, 0.5, 0.5})
	chit := OBBIntersection(long, contact)
	if !chit.Intersects || !obbNear(chit.Depth, 0, 2e-5) {
		t.Fatalf("boundary placement should approximately touch, got %+v", chit)
	}
}

// Coincident boxes: center delta is zero, so axis sign is arbitrary but the
// result must stay finite, intersecting and symmetric.
func TestOBBCoincident(t *testing.T) {
	o := NewOBB(Vec3{1, 2, 3}, Vec3{2, 0.5, 1}, [3]Vec3{
		{0, 1, 0}, {-1, 0, 0}, {0, 0, 1},
	})
	h := OBBIntersection(o, o)
	if !h.Intersects || math.IsNaN(float64(h.Depth)) || h.Penetration <= 0 {
		t.Fatalf("identical boxes must intersect with positive depth: %+v", h)
	}
	if !obbNear(h.Penetration, 1.0, 1e-6) {
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
			Center:      Vec3{float32(math.NaN()), 0, 0},
			HalfExtents: Vec3{1, 1, 1},
			Axes:        obbIdentityAxes(),
		},
		"inf half": {
			Center:      Vec3{0, 0, 0},
			HalfExtents: Vec3{1, float32(math.Inf(1)), 1},
			Axes:        obbIdentityAxes(),
		},
		"nan axis": {
			Center:      Vec3{},
			HalfExtents: Vec3{1, 1, 1},
			Axes: [3]Vec3{
				{float32(math.NaN()), 0, 0}, {0, 1, 0}, {0, 0, 1},
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
	if !obbVecNear(min, Vec3{0, 1, 2}, 1e-6) || !obbVecNear(max, Vec3{2, 3, 4}, 1e-6) {
		t.Fatalf("AABB = %v..%v", min, max)
	}
	if corners := o.Corners(); len(corners) != 8 {
		t.Fatal("want 8 corners")
	}

	// Quaternion orientation must produce a validated frame matching the
	// equivalent Z rotation.
	q := QuatRotate(math.Pi/4, Vec3{0, 0, 1})
	qo := OBBFromQuat(Vec3{}, Vec3{1, 1, 1}, q)
	s := float32(math.Sqrt(0.5))
	if !obbVecNear(qo.Axes[0], Vec3{s, s, 0}, 1e-6) ||
		!obbVecNear(qo.Axes[1], Vec3{-s, s, 0}, 1e-6) {
		t.Fatalf("quat axes wrong: %v", qo.Axes)
	}
}

// ---------------------------------------------------------------------------
// Independent corner oracle, computed in float64 for stability.
// ---------------------------------------------------------------------------

func f64v(v Vec3) [3]float64 { return [3]float64{float64(v[0]), float64(v[1]), float64(v[2])} }

func obbOracleAxesF64(a, b OBB) [][3]float64 {
	axes := make([][3]float64, 0, 15)
	for i := 0; i < 3; i++ {
		axes = append(axes, f64v(a.Axes[i]), f64v(b.Axes[i]))
	}
	af := [3][3]float64{f64v(a.Axes[0]), f64v(a.Axes[1]), f64v(a.Axes[2])}
	bf := [3][3]float64{f64v(b.Axes[0]), f64v(b.Axes[1]), f64v(b.Axes[2])}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			x := af[i][1]*bf[j][2] - af[i][2]*bf[j][1]
			y := af[i][2]*bf[j][0] - af[i][0]*bf[j][2]
			z := af[i][0]*bf[j][1] - af[i][1]*bf[j][0]
			l := math.Sqrt(x*x + y*y + z*z)
			if l > 1e-6 {
				axes = append(axes, [3]float64{x / l, y / l, z / l})
			}
		}
	}
	return axes
}

func obbProjectedIntervalF64(o OBB, axis [3]float64) (center, radius float64) {
	var lo, hi = math.Inf(1), math.Inf(-1)
	for _, corner := range o.Corners() {
		cf := f64v(corner)
		p := cf[0]*axis[0] + cf[1]*axis[1] + cf[2]*axis[2]
		lo = math.Min(lo, p)
		hi = math.Max(hi, p)
	}
	return (lo + hi) * 0.5, (hi - lo) * 0.5
}

func obbOracleF64(a, b OBB) (minOverlap float64, best [3]float64, separating int) {
	minOverlap = math.Inf(1)
	for _, ax := range obbOracleAxesF64(a, b) {
		ca, ra := obbProjectedIntervalF64(a, ax)
		cb, rb := obbProjectedIntervalF64(b, ax)
		ov := ra + rb - math.Abs(ca-cb)
		if ov < minOverlap {
			minOverlap = ov
			best = ax
		}
		if ov < -1e-6 {
			separating++
		}
	}
	return
}

func obbRandomUnitF64(r *rand.Rand) [3]float64 {
	for {
		v := [3]float64{r.NormFloat64(), r.NormFloat64(), r.NormFloat64()}
		l := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
		if l > 1e-3 {
			return [3]float64{v[0] / l, v[1] / l, v[2] / l}
		}
	}
}

func obbRandom(r *rand.Rand) OBB {
	u := obbRandomUnitF64(r)
	v := obbRandomUnitF64(r)
	d := v[0]*u[0] + v[1]*u[1] + v[2]*u[2]
	v = [3]float64{v[0] - u[0]*d, v[1] - u[1]*d, v[2] - u[2]*d}
	for math.Sqrt(v[0]*v[0]+v[1]*v[1]+v[2]*v[2]) < 0.2 {
		v = obbRandomUnitF64(r)
		d = v[0]*u[0] + v[1]*u[1] + v[2]*u[2]
		v = [3]float64{v[0] - u[0]*d, v[1] - u[1]*d, v[2] - u[2]*d}
	}
	vl := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	v = [3]float64{v[0] / vl, v[1] / vl, v[2] / vl}
	w := [3]float64{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	}
	he := Vec3{
		float32(math.Pow(10, r.Float64()*4-2)),
		float32(math.Pow(10, r.Float64()*4-2)),
		float32(math.Pow(10, r.Float64()*4-2)),
	}
	radius := float64(he[0] + he[1] + he[2])
	cu := obbRandomUnitF64(r)
	cr := r.Float64() * radius * 1.5
	return OBB{
		Center:      Vec3{float32(cu[0] * cr), float32(cu[1] * cr), float32(cu[2] * cr)},
		HalfExtents: he,
		Axes: [3]Vec3{
			{float32(u[0]), float32(u[1]), float32(u[2])},
			{float32(v[0]), float32(v[1]), float32(v[2])},
			{float32(w[0]), float32(w[1]), float32(w[2])},
		},
	}
}

// Randomized differential test against the corner oracle plus the MTV
// move-to-contact invariant.
func TestOBBRandomOracleAndMTV32(t *testing.T) {
	r := rand.New(rand.NewSource(13579))
	for i := 0; i < 800; i++ {
		a := obbRandom(r)
		b := obbRandom(r)

		want, _, nSep := obbOracleF64(a, b)
		hit := OBBIntersection(a, b)

		scale := 1 + float64(a.Center.Len()+b.Center.Len()+
			a.HalfExtents[0]+a.HalfExtents[1]+a.HalfExtents[2]+
			b.HalfExtents[0]+b.HalfExtents[1]+b.HalfExtents[2])
		eps := 1e-4 * scale

		if !obbNear(float32(hit.SignedDepth), float32(want), float32(eps)) {
			// Float32 snapping at contact is allowed to differ more strongly;
			// compare via the slack-aware condition instead.
			if math.Abs(float64(hit.SignedDepth)-want) > eps &&
				math.Abs(want) > float64(obbTouchSlackFactor)*scale {
				t.Fatalf("iter %d: signed=%v oracle=%v eps=%v", i, hit.SignedDepth, want, eps)
			}
		}

		if hit.Intersects {
			moved := a
			moved.Center = a.Center.Add(hit.Axis.Mul(hit.Penetration))
			h2 := moved.Intersection(b)
			if math.Abs(float64(h2.SignedDepth)) > 1e-3*scale {
				t.Fatalf("iter %d: MTV move did not reach contact, signed=%v scale=%v", i, h2.SignedDepth, scale)
			}
		} else if nSep == 1 {
			moved := a
			moved.Center = a.Center.Add(hit.Axis.Mul(-hit.Separation))
			h2 := moved.Intersection(b)
			if math.Abs(float64(h2.SignedDepth)) > 1e-3*scale {
				t.Fatalf("iter %d: gap close did not reach contact, signed=%v", i, h2.SignedDepth)
			}
		}
	}
}

// Cross-axis winners must be reachable in float32 as well.
func TestOBBCrossAxisWinner32(t *testing.T) {
	r := rand.New(rand.NewSource(97531))
	found := false
	for i := 0; i < 20000 && !found; i++ {
		a := obbRandom(r)
		a.HalfExtents[r.Intn(3)] *= 8
		b := obbRandom(r)
		b.HalfExtents[r.Intn(3)] *= 8

		want, oracleAxis, _ := obbOracleF64(a, b)
		face := false
		for k := 0; k < 3; k++ {
			af := f64v(a.Axes[k])
			bf := f64v(b.Axes[k])
			if math.Abs(oracleAxis[0]*af[0]+oracleAxis[1]*af[1]+oracleAxis[2]*af[2]) > 1-1e-4 ||
				math.Abs(oracleAxis[0]*bf[0]+oracleAxis[1]*bf[1]+oracleAxis[2]*bf[2]) > 1-1e-4 {
				face = true
			}
		}
		if face || want >= -1e-4 {
			continue
		}
		hit := OBBIntersection(a, b)
		dot := float64(hit.Axis[0])*oracleAxis[0] + float64(hit.Axis[1])*oracleAxis[1] + float64(hit.Axis[2])*oracleAxis[2]
		if math.Abs(math.Abs(dot)-1) > 1e-3 {
			t.Fatalf("cross-axis case %d: axis %v vs oracle %v", i, hit.Axis, oracleAxis)
		}
		found = true
	}
	if !found {
		t.Fatal("never found a disjoint cross-axis winner")
	}
}
