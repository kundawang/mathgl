// This file is generated from mgl32/ik.go; DO NOT EDIT

// Copyright 2026 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"errors"
	"fmt"
	"math"
)

// Sentinel errors returned by the two-bone inverse kinematics solver. Callers
// can inspect failures with errors.Is; every error returned by this file wraps
// one of these values.
var (
	// ErrIKNonFinite is returned when any input (position, length or hint) is
	// NaN or +/-Inf.
	ErrIKNonFinite = errors.New("mathgl/ik: input contains a non-finite value")

	// ErrIKInvalidLength is returned when either bone length is zero, negative
	// or otherwise unusable.
	ErrIKInvalidLength = errors.New("mathgl/ik: bone lengths must be finite and greater than zero")

	// ErrIKTargetUnreachable is returned (only when reach clamping is disabled)
	// when the root-target distance is outside the interval
	// [abs(Upper-Lower), Upper+Lower].
	ErrIKTargetUnreachable = errors.New("mathgl/ik: target is outside the reachable range")

	// ErrIKBendPlaneDegenerate is returned only in StrictBendPlane mode when
	// the bend hint is zero or parallel to the root-target axis, which makes
	// the bend plane undefined.
	ErrIKBendPlaneDegenerate = errors.New("mathgl/ik: bend plane hint is degenerate")
)

// TwoBoneIKResult is the solved pose of a two-bone chain (for example
// shoulder/elbow/wrist or hip/knee/ankle).
//
// The chain geometry is always:
//
//	Root --UpperLength--> Elbow --LowerLength--> End
//
// so that, up to floating point rounding,
//
//	Elbow.Sub(Root).Len() == UpperLength
//	End.Sub(Elbow).Len()  == LowerLength.
type TwoBoneIKResult struct {
	// Root is the chain origin that was solved against.
	Root Vec3
	// Elbow is the solved middle joint position.
	Elbow Vec3
	// End is the solved end-effector (wrist/hand/palm) position. It equals the
	// requested target while reachable and the clamped boundary point when the
	// target had to be clamped.
	End Vec3

	// Axis is the unit direction from Root to End. It is zero only for the
	// exact zero-reach fold, where an arbitrary stable axis is chosen instead.
	Axis Vec3
	// BendDirection is the unit vector in the bend plane that points from the
	// Root-End axis towards Elbow (the projected bend hint).
	BendDirection Vec3
	// BendNormal is the unit normal of the bend plane, Axis cross
	// BendDirection. It stays stable for a fixed hint.
	BendNormal Vec3

	// Rotation is a stable end-effector orientation built from the solved
	// geometry: its +Y axis points from Elbow to End (the distal bone), its
	// +Z axis is BendNormal and its +X axis completes a right-handed frame.
	// It is useful as a palm/hand orientation; retarget the axes in your
	// content pipeline if your rig expects another convention.
	Rotation Quat
	// Pose is Rotation positioned at End (column-major Mat4). Its translation
	// column is therefore End.
	Pose Mat4

	// UpperLength and LowerLength are the preserved input bone lengths.
	UpperLength float64
	LowerLength float64

	// Distance is the effective Root-End distance used by the solve.
	Distance float64
	// MinReach is abs(UpperLength-LowerLength): the closest fold the chain can
	// achieve.
	MinReach float64
	// MaxReach is UpperLength+LowerLength: the fully extended reach.
	MaxReach float64

	// Reachable reports whether the requested target distance was inside
	// [MinReach, MaxReach] up to numerical tolerance.
	Reachable bool
	// Clamped reports whether the requested target was moved onto a reachable
	// boundary. It is only ever true when clamping was requested.
	Clamped bool
	// BendHintDegenerate reports that the supplied bend hint was zero or
	// parallel to Axis, so a deterministic fallback direction (or a retained
	// previous direction for a TwoBoneIKSolver) was used.
	BendHintDegenerate bool
}

// Hand returns End; it is a readability alias for callers modelling the
// end-effector as a hand or palm.
func (r TwoBoneIKResult) Hand() Vec3 { return r.End }

// Wrist returns End; it is a readability alias for callers modelling the
// end-effector as a wrist.
func (r TwoBoneIKResult) Wrist() Vec3 { return r.End }

// ElbowPosition returns Elbow.
func (r TwoBoneIKResult) ElbowPosition() Vec3 { return r.Elbow }

// EndPosition returns End.
func (r TwoBoneIKResult) EndPosition() Vec3 { return r.End }

// TwoBoneIKOptions configures a two-bone IK solve.
type TwoBoneIKOptions struct {
	// Root is the chain origin (shoulder/hip).
	Root Vec3
	// UpperLength is the Root-to-Elbow bone length; must be > 0.
	UpperLength float64
	// LowerLength is the Elbow-to-End bone length; must be > 0.
	LowerLength float64
	// Target is the desired end-effector position.
	Target Vec3
	// BendHint (pole vector) indicates the direction the elbow should bend
	// towards. It does not need to be normalized; only its component
	// perpendicular to the Root-Target axis is used. "Bend towards +X" means
	// passing Vec3{1, 0, 0}.
	BendHint Vec3

	// ClampToReach, when true, moves too-close/far targets onto the nearest
	// reachable boundary instead of returning ErrIKTargetUnreachable.
	ClampToReach bool
	// Clamp is a shorthand alias for ClampToReach; either name enables
	// reach clamping.
	Clamp bool
	// StrictBendPlane, when true, makes a degenerate BendHint return
	// ErrIKBendPlaneDegenerate instead of applying a stable fallback.
	StrictBendPlane bool
}

// Solve performs the solve described by o. It is equivalent to calling
// SolveTwoBoneIKWithOptions.
func (o TwoBoneIKOptions) Solve() (TwoBoneIKResult, error) {
	return SolveTwoBoneIKWithOptions(o)
}

// SolveTwoBoneIK solves a two-bone chain from Root with the given upper
// (Root-to-Elbow) and lower (Elbow-to-End) lengths towards Target.
//
// BendHint is the pole/bend direction: the returned elbow lies on the hint
// side of the Root-Target axis. It may be non-unit and may degenerate; a
// deterministic fallback keeps the result stable.
//
// The optional clampToReach argument selects unreachable-target handling:
//
//   - omitted or true: targets farther than Upper+Lower or closer than
//     abs(Upper-Lower) are clamped to the nearest reachable boundary and
//     Result.Clamped is set;
//   - false: such targets produce an error wrapping ErrIKTargetUnreachable.
//
// Non-finite arguments and non-positive bone lengths always produce an error.
//
// The canonical example from the task statement:
//
//	res, err := mgl64.SolveTwoBoneIK(
//	    mgl64.Vec3{0, 0, 0}, 1, 1,
//	    mgl64.Vec3{1, 1, 0}, mgl64.Vec3{1, 0, 0},
//	)
//	// res.Elbow == mgl64.Vec3{1, 0, 0}, res.End ~= mgl64.Vec3{1, 1, 0}
func SolveTwoBoneIK(root Vec3, upperLength, lowerLength float64, target, bendHint Vec3, clampToReach ...bool) (TwoBoneIKResult, error) {
	clamp := true
	if len(clampToReach) > 0 {
		clamp = clampToReach[0]
	}
	if len(clampToReach) > 1 {
		return TwoBoneIKResult{}, fmt.Errorf("mathgl/ik: SolveTwoBoneIK accepts at most one clampToReach argument, got %d", len(clampToReach))
	}
	return SolveTwoBoneIKWithOptions(TwoBoneIKOptions{
		Root:            root,
		UpperLength:     upperLength,
		LowerLength:     lowerLength,
		Target:          target,
		BendHint:        bendHint,
		ClampToReach:    clamp,
		StrictBendPlane: false,
	})
}

// SolveTwoBoneIKStrict is SolveTwoBoneIK with clamping disabled: an
// unreachable target returns an error wrapping ErrIKTargetUnreachable.
func SolveTwoBoneIKStrict(root Vec3, upperLength, lowerLength float64, target, bendHint Vec3) (TwoBoneIKResult, error) {
	return SolveTwoBoneIK(root, upperLength, lowerLength, target, bendHint, false)
}

// TwoBoneIK solves the chain with reach clamping and returns only the joint
// positions. It is a compact convenience wrapper around SolveTwoBoneIK for
// callers that do not need the orientation or diagnostics.
func TwoBoneIK(root Vec3, upperLength, lowerLength float64, target, bendHint Vec3) (elbow, end Vec3, err error) {
	res, err := SolveTwoBoneIK(root, upperLength, lowerLength, target, bendHint, true)
	if err != nil {
		return Vec3{}, Vec3{}, err
	}
	return res.Elbow, res.End, nil
}

// SolveTwoBoneIKPoints is an alias of TwoBoneIK that names the returned
// points explicitly.
func SolveTwoBoneIKPoints(root Vec3, upperLength, lowerLength float64, target, bendHint Vec3) (elbow, end Vec3, err error) {
	return TwoBoneIK(root, upperLength, lowerLength, target, bendHint)
}

// SolveTwoBoneIKPose solves the chain and returns the elbow together with the
// end-effector pose matrix (rotation plus translation at the solved end
// point). The optional clampToReach argument behaves as in SolveTwoBoneIK.
func SolveTwoBoneIKPose(root Vec3, upperLength, lowerLength float64, target, bendHint Vec3, clampToReach ...bool) (elbow Vec3, pose Mat4, err error) {
	res, err := SolveTwoBoneIK(root, upperLength, lowerLength, target, bendHint, clampToReach...)
	if err != nil {
		return Vec3{}, Mat4{}, err
	}
	return res.Elbow, res.Pose, nil
}

// SolveTwoBoneIKOrientation solves the chain and returns the elbow together
// with the stable end-effector orientation quaternion. The optional
// clampToReach argument behaves as in SolveTwoBoneIK.
func SolveTwoBoneIKOrientation(root Vec3, upperLength, lowerLength float64, target, bendHint Vec3, clampToReach ...bool) (elbow Vec3, orientation Quat, err error) {
	res, err := SolveTwoBoneIK(root, upperLength, lowerLength, target, bendHint, clampToReach...)
	if err != nil {
		return Vec3{}, Quat{}, err
	}
	return res.Elbow, res.Rotation, nil
}

// TwoBoneIKConfig is a readability alias for TwoBoneIKOptions.
type TwoBoneIKConfig = TwoBoneIKOptions

// SolveTwoBoneIKWithOptions is the full, option-driven entry point.
func SolveTwoBoneIKWithOptions(o TwoBoneIKOptions) (TwoBoneIKResult, error) {
	res, _, _, err := solveTwoBoneIKCore(o, nil, nil)
	return res, err
}

// TwoBoneIKSolver is a reusable, frame-to-frame continuous two-bone IK
// solver. In addition to the stateless behaviour of SolveTwoBoneIK it retains
// the bend plane of the previous successful solve, so a bend hint that passes
// through the Root-Target direction (the pole-vector singularity) does not
// cause a sudden elbow flip: the previous plane is reused while the hint is
// degenerate.
//
// The zero value is not usable; construct one with NewTwoBoneIKSolver or set
// UpperLength and LowerLength before the first Solve.
type TwoBoneIKSolver struct {
	// UpperLength and LowerLength are the preserved bone lengths. They may be
	// adjusted between frames.
	UpperLength float64
	LowerLength float64
	// ClampToReach mirrors TwoBoneIKOptions.ClampToReach for every Solve.
	ClampToReach bool

	axis     Vec3
	pole     Vec3
	hasFrame bool
}

// TwoBoneIKChain is a readability alias for TwoBoneIKSolver.
type TwoBoneIKChain = TwoBoneIKSolver

// NewTwoBoneIKSolver builds a continuous solver for the given bone lengths.
// Reach clamping is enabled by default; toggle ClampToReach to change it.
func NewTwoBoneIKSolver(upperLength, lowerLength float64) *TwoBoneIKSolver {
	return &TwoBoneIKSolver{
		UpperLength:  upperLength,
		LowerLength:  lowerLength,
		ClampToReach: true,
	}
}

// Reset forgets the retained bend plane. The next Solve behaves like the
// stateless solver until a new plane is established.
func (s *TwoBoneIKSolver) Reset() {
	s.hasFrame = false
	s.axis = Vec3{}
	s.pole = Vec3{}
}

// HasPreviousFrame reports whether a bend plane is retained.
func (s *TwoBoneIKSolver) HasPreviousFrame() bool { return s.hasFrame }

// Solve advances the continuous solver for one target/hint pair.
func (s *TwoBoneIKSolver) Solve(root, target, bendHint Vec3) (TwoBoneIKResult, error) {
	o := TwoBoneIKOptions{
		Root:         root,
		UpperLength:  s.UpperLength,
		LowerLength:  s.LowerLength,
		Target:       target,
		BendHint:     bendHint,
		ClampToReach: s.ClampToReach,
	}

	var prevPole, prevAxis *Vec3
	if s.hasFrame {
		prevPole = &s.pole
		prevAxis = &s.axis
	}

	res, pole, axis, err := solveTwoBoneIKCore(o, prevPole, prevAxis)
	if err != nil {
		return res, err
	}
	s.pole = pole
	s.axis = axis
	s.hasFrame = true
	return res, nil
}

// ikEps is the relative tolerance used for reach boundaries, zero-distance
// folds and degeneracy. It is deliberately a little above float64 rounding
// noise at unit scale while staying negligible for gameplay.
const ikEps = 1e-6

// ikMachEps is the machine epsilon of the Vec3 component type. It is computed
// rather than hard-coded so the mgl64-to-mgl64 generator naturally produces
// the correct value for each precision.
var ikMachEps = func() float64 {
	e := float64(1.0)
	for float64(1.0)+e/2 != float64(1.0) {
		e /= 2
	}
	return e
}()

// ikDenoise zeroes components that are pure floating-point rounding residue
// relative to the given local scale. It keeps axis-aligned canonical
// solutions exactly clean (for example local offset (1,~3e-8,0) becomes
// (1,0,0)) without touching values that carry real geometric meaning at the
// type's precision.
func ikDenoise(v Vec3, scale float64) Vec3 {
	tol := 8 * ikMachEps * scale
	for i := range v {
		if Abs(v[i]) <= tol {
			v[i] = 0
		}
	}
	return v
}

// ikSnapLocalZero denoises the anchor-relative offset of p (at scale, usually
// the relevant bone length) and maps it back to world space. Cleaning local
// offsets rather than world coordinates keeps the operation correct under
// large root translations and prevents a long other bone from loosening the
// tolerance of a short bone.
func ikSnapLocalZero(anchor, p Vec3, scale float64) Vec3 {
	return anchor.Add(ikDenoise(p.Sub(anchor), scale))
}

func ikMaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func ikFinite(f float64) bool {
	x := float64(f)
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func ikFiniteVec3(v Vec3) bool {
	return ikFinite(v[0]) && ikFinite(v[1]) && ikFinite(v[2])
}

// ikFallbackPerp returns a stable unit vector perpendicular to unit u by
// crossing u with the world axis it is least aligned with. Because the chosen
// reference is never near-parallel to u, the result is always well defined.
func ikFallbackPerp(u Vec3) Vec3 {
	ax := Abs(u[0])
	ay := Abs(u[1])
	az := Abs(u[2])

	var ref Vec3
	switch {
	case ax <= ay && ax <= az:
		ref = Vec3{1, 0, 0}
	case ay <= az:
		ref = Vec3{0, 1, 0}
	default:
		ref = Vec3{0, 0, 1}
	}

	p := u.Cross(ref)
	if p.LenSqr() == 0 {
		// Unreachable for a unit u with the least-aligned reference, but keep a
		// hard guarantee against NaNs.
		return Vec3{0, 1, 0}
	}
	return p.Normalize()
}

// ikStableUnit normalizes v, returning (fallback, false) when v is too small
// to normalize reliably.
func ikStableUnit(v Vec3) (Vec3, bool) {
	l2 := v.LenSqr()
	// lenSqr <= eps^2 is equivalent to len <= eps while avoiding a sqrt.
	if !(l2 > ikEps*ikEps) { // also catches NaN
		return Vec3{}, false
	}
	l := float64(math.Sqrt(float64(l2)))
	u := v.Mul(1.0 / l)
	if !ikFiniteVec3(u) {
		return Vec3{}, false
	}
	return u, true
}

// solveTwoBoneIKCore contains the whole algorithm. When prevPole/prevAxis are
// non-nil they provide frame-to-frame continuity. It additionally returns the
// effective pole and axis so a stateful caller can retain them.
func solveTwoBoneIKCore(o TwoBoneIKOptions, prevPole, prevAxis *Vec3) (TwoBoneIKResult, Vec3, Vec3, error) {
	res := TwoBoneIKResult{}
	// ---- Input validation -------------------------------------------------
	if !ikFiniteVec3(o.Root) || !ikFiniteVec3(o.Target) || !ikFiniteVec3(o.BendHint) ||
		!ikFinite(o.UpperLength) || !ikFinite(o.LowerLength) {
		return TwoBoneIKResult{}, Vec3{}, Vec3{}, ErrIKNonFinite
	}
	if o.UpperLength <= 0 || o.LowerLength <= 0 {
		return TwoBoneIKResult{}, Vec3{}, Vec3{}, fmt.Errorf("%w: upper=%v lower=%v",
			ErrIKInvalidLength, o.UpperLength, o.LowerLength)
	}

	l1 := o.UpperLength
	l2 := o.LowerLength
	if o.Clamp {
		o.ClampToReach = true
	}
	boneScale := ikMaxFloat(l1, l2)
	reachEps := ikEps * boneScale

	minReach := Abs(l1 - l2)
	maxReach := l1 + l2

	res.Root = o.Root
	res.UpperLength = l1
	res.LowerLength = l2
	res.MinReach = minReach
	res.MaxReach = maxReach

	// ---- Target distance and reachability --------------------------------
	delta := o.Target.Sub(o.Root)
	d := delta.Len()
	if !ikFinite(d) {
		return TwoBoneIKResult{}, Vec3{}, Vec3{}, ErrIKNonFinite
	}

	reachable := d >= minReach-reachEps && d <= maxReach+reachEps
	res.Reachable = reachable

	beyondMax := d > maxReach
	beyondMin := d < minReach
	outsideTolerance := d > maxReach+reachEps || d < minReach-reachEps
	if outsideTolerance && !o.ClampToReach {
		if d > maxReach {
			return TwoBoneIKResult{}, Vec3{}, Vec3{}, fmt.Errorf("%w: distance %v > max reach %v",
				ErrIKTargetUnreachable, d, maxReach)
		}
		return TwoBoneIKResult{}, Vec3{}, Vec3{}, fmt.Errorf("%w: distance %v < min reach %v",
			ErrIKTargetUnreachable, d, minReach)
	}

	r := d
	if beyondMax {
		// Also snaps a sub-tolerance float overshoot onto the exact boundary.
		r = maxReach
	}
	if beyondMin {
		r = minReach
	}
	res.Clamped = o.ClampToReach && (beyondMax || beyondMin)
	res.Distance = r

	// A zero effective reach only exists for (nearly) equal bones folded back
	// onto the root. Handle it before any division by r.
	zeroReachEps := reachEps
	if r <= zeroReachEps {
		return solveZeroReach(o, &res, prevPole)
	}

	// ---- Root -> End axis -------------------------------------------------
	var axis Vec3
	if d > reachEps {
		axis = delta.Mul(1.0 / d)
	} else if h, ok := ikStableUnit(o.BendHint); ok {
		// Target is on the root: let the hint choose the approach axis.
		axis = h
	} else if prevAxis != nil {
		if a, ok := ikStableUnit(*prevAxis); ok {
			axis = a
		} else {
			axis = ikFallbackAxis()
		}
	} else {
		axis = ikFallbackAxis()
	}

	// ---- Bend pole: project hint onto plane perpendicular to axis ---------
	pole, hintDegenerate, perr := ikSelectPole(o, axis, prevPole)
	if perr != nil {
		return TwoBoneIKResult{}, Vec3{}, Vec3{}, perr
	}
	res.BendHintDegenerate = hintDegenerate

	// ---- Law of cosines ---------------------------------------------------
	// a: axial distance Root -> elbow projection.
	// h: perpendicular elbow height.
	rd := float64(r)
	l1d := float64(l1)
	l2d := float64(l2)
	a64 := (l1d*l1d - l2d*l2d + rd*rd) / (2.0 * rd)
	h2 := l1d*l1d - a64*a64
	if h2 < 0 {
		// Floating point overshoot at full extension/fold: clamp to the line.
		h2 = 0
	}
	h64 := math.Sqrt(h2)
	a := float64(a64)
	h := float64(h64)

	axial := axis.Mul(a)
	bulge := pole.Mul(h)
	elbow0 := o.Root.Add(axial).Add(bulge)

	// End sits on the axis at distance r. Building it this way reproduces the
	// requested target exactly for reachable inputs (axis came from delta).
	end0 := o.Root.Add(axis.Mul(r))

	// Enforce the upper bone length by construction. The law-of-cosines point
	// can drift by a few rounding ULPs; re-projecting the elbow along its bone
	// direction makes Elbow-Root match the requested length as closely as
	// float precision allows. The direction stays well defined because the
	// length is strictly positive.
	elbow := elbow0
	if d, ok := ikStableUnit(elbow0.Sub(o.Root)); ok {
		elbow = o.Root.Add(d.Mul(l1))
	}
	elbow = ikSnapLocalZero(o.Root, elbow, l1)

	// Finalize the distal bone. For a reachable target, aim it directly at the
	// requested target so the hand lands on the target to within the triangle
	// rounding; for a clamped target, aim along the analytic boundary
	// direction instead, which keeps the end on the reach sphere. In both
	// cases stepping exactly LowerLength along the unit direction preserves
	// the second bone length.
	var distal Vec3
	if res.Clamped {
		distal = end0.Sub(elbow)
	} else {
		distal = o.Target.Sub(elbow)
	}
	end := end0
	if d, ok := ikStableUnit(distal); ok {
		end = elbow.Add(d.Mul(l2))
	}

	end = ikSnapLocalZero(elbow, end, l2)

	res.Elbow = elbow
	res.End = end
	res.Axis = axis
	res.BendDirection = pole
	normal := axis.Cross(pole)
	if n, ok := ikStableUnit(normal); ok {
		res.BendNormal = n
	} else {
		res.BendNormal = Vec3{}
	}

	res.fillOrientation()

	return res, pole, axis, nil
}

// ikFallbackAxis is the stable arbitrary root->end axis used when both the
// target direction and the hint are unavailable.
func ikFallbackAxis() Vec3 { return Vec3{1, 0, 0} }

// ikSelectPole determines the unit bulge direction perpendicular to axis,
// implementing hint projection, strict errors, previous-frame retention and
// the deterministic fallback in that priority order.
func ikSelectPole(o TwoBoneIKOptions, axis Vec3, prevPole *Vec3) (pole Vec3, degenerate bool, err error) {
	hintL2 := o.BendHint.LenSqr()
	if hintL2 > 0 && ikFinite(hintL2) {
		dot := o.BendHint.Dot(axis)
		proj := o.BendHint.Sub(axis.Mul(dot))
		projL2 := proj.LenSqr()
		// Relative degeneracy test: the perpendicular component must carry at
		// least ikEps of the hint magnitude. This is scale independent.
		if projL2 >= ikEps*ikEps*hintL2 {
			if p, ok := ikStableUnit(proj); ok {
				return p, false, nil
			}
		}
	}

	// Hint is zero or parallel to the axis: bend plane is undefined.
	if o.StrictBendPlane {
		return Vec3{}, true, ErrIKBendPlaneDegenerate
	}

	if prevPole != nil {
		dot := prevPole.Dot(axis)
		proj := prevPole.Sub(axis.Mul(dot))
		if p, ok := ikStableUnit(proj); ok {
			return p, true, nil
		}
	}

	return ikFallbackPerp(axis), true, nil
}

// solveZeroReach handles the fully folded configuration where Root and End
// coincide (only possible for equal-length bones). The elbow is placed one
// upper-bone length along the chosen bulge direction and the end one lower
// bone back, which preserves both lengths exactly.
func solveZeroReach(o TwoBoneIKOptions, res *TwoBoneIKResult, prevPole *Vec3) (TwoBoneIKResult, Vec3, Vec3, error) {
	// Prefer the raw hint as the bulge itself: with no Root-Target axis the
	// intuitive meaning of "bend towards +X" is "elbow at +X".
	var bulgeDir Vec3
	degenerate := false

	if b, ok := ikStableUnit(o.BendHint); ok {
		bulgeDir = b
	} else {
		degenerate = true
		if prevPole != nil {
			if b, ok := ikStableUnit(*prevPole); ok {
				bulgeDir = b
			}
		}
		if bulgeDir == (Vec3{}) {
			bulgeDir = ikFallbackAxis()
		}
	}

	if degenerate && o.StrictBendPlane {
		return TwoBoneIKResult{}, Vec3{}, Vec3{}, ErrIKBendPlaneDegenerate
	}

	elbow := o.Root.Add(bulgeDir.Mul(o.UpperLength))
	end := elbow.Sub(bulgeDir.Mul(o.LowerLength))

	// Build a consistent orthonormal frame: pole = bulge, axis = stable perp,
	// normal = axis x pole.
	pole := bulgeDir
	axis := ikFallbackPerp(pole)
	normal := axis.Cross(pole).Normalize()

	elbow = ikSnapLocalZero(o.Root, elbow, o.UpperLength)
	end = ikSnapLocalZero(elbow, end, o.LowerLength)
	res.Elbow = elbow
	res.End = end
	res.Axis = axis
	res.BendDirection = pole
	res.BendNormal = normal
	res.Distance = end.Sub(o.Root).Len()
	res.fillOrientation()

	return *res, pole, axis, nil
}

// fillOrientation builds the stable end-effector frame and Pose matrix.
// Convention: +Y = Elbow->End (distal bone), +Z = BendNormal, +X = Y x Z.
func (r *TwoBoneIKResult) fillOrientation() {
	y, ok := ikStableUnit(r.End.Sub(r.Elbow))
	if !ok {
		y = Vec3{0, 1, 0}
	}
	z := r.BendNormal
	if z.LenSqr() == 0 || !ikFiniteVec3(z) {
		z = Vec3{0, 0, 1}
	}
	// Re-orthogonalize z against y to remove accumulated rounding.
	z = z.Sub(y.Mul(z.Dot(y)))
	if zn, ok := ikStableUnit(z); ok {
		z = zn
	} else {
		z = ikFallbackPerp(y)
	}
	x := y.Cross(z)
	if xn, ok := ikStableUnit(x); ok {
		x = xn
	} else {
		x = Vec3{1, 0, 0}
	}
	// Final Gram-Schmidt pass guarantees an orthonormal right-handed frame.
	z = x.Cross(y).Normalize()

	r.Rotation = Mat4ToQuat(Mat4FromCols(
		Vec4{x[0], x[1], x[2], 0},
		Vec4{y[0], y[1], y[2], 0},
		Vec4{z[0], z[1], z[2], 0},
		Vec4{r.End[0], r.End[1], r.End[2], 1},
	)).Normalize()

	r.Pose = Mat4FromCols(
		Vec4{x[0], x[1], x[2], 0},
		Vec4{y[0], y[1], y[2], 0},
		Vec4{z[0], z[1], z[2], 0},
		Vec4{r.End[0], r.End[1], r.End[2], 1},
	)
}
