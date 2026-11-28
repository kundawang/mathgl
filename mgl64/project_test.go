// This file is generated from mgl32/project_test.go; DO NOT EDIT

// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"math"
	"testing"
)

func TestProject(t *testing.T) {
	t.Parallel()

	obj := Vec3{1002, 960, 0}
	modelview := Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 203, 1, 0, 1}
	projection := Mat4{0.0013020833721384406, 0, 0, 0, -0, -0.0020833334419876337, -0, -0, -0, -0, -1, -0, -1, 1, 0, 1}
	initialX, initialY, width, height := 0, 0, 1536, 960
	win := Project(obj, modelview, projection, initialX, initialY, width, height)
	answer := Vec3{1205.0000359117985, -1.0000501200556755, 0.5} // From glu.Project()

	if !win.ApproxEqualThreshold(answer, 1e-4) {
		t.Errorf("Project does something weird, differs from expected by of %v", win.Sub(answer).Len())
	}

	objr, err := UnProject(win, modelview, projection, initialX, initialY, width, height)
	if err != nil {
		t.Errorf("UnProject returned error: %v", err)
	}
	if !objr.ApproxEqualThreshold(obj, 1e-4) {
		t.Errorf("UnProject(%v) != %v (got %v)", win, obj, objr)
	}
}

// Test from
// http://stackoverflow.com/questions/38471708/opengl-glm-project-method-giving-unexpected-results
func TestProjectNonOneW(t *testing.T) {
	t.Parallel()

	obj := Vec3{5, 0, 0}

	projection := Perspective(
		DegToRad(45), // Field of view (45 degrees).
		800.0/600.0,  // Aspect ratio.
		0.1,          // Near Z at 0.1.
		10)           // Far Z at 10.
	camera := LookAtV(
		Vec3{0, 0.1, 10}, // Camera out on Z and slightly above.
		Vec3{0, 0, 0},    // Looking at the origin.
		Vec3{0, 1, 0})    // Up is positive Y.
	model := Ident4()               // Simple model matrix, to avoid confusion.
	modelView := camera.Mul4(model) // The model-view matrix (== camera, here).

	win := Project(obj, modelView, projection, 0, 0, 800, 600)

	t.Logf("Test:   (%v, %v, %v)", win[0], win[1], win[2])

	answer := Vec3{762.114, 300, 1} // verified with glm

	if !win.ApproxEqualThreshold(answer, 1e-4) {
		t.Errorf("Project does not properly do perspective division, differs from expected by %v", win.Sub(answer).Len())
	}
}

func TestUnprojectSingular(t *testing.T) {
	if _, err := UnProject(Vec3{}, Mat4{}, Mat4{}, 0, 0, 2048, 1152); err == nil {
		t.Errorf("Did not get error from UnProject on singular matrix")
	} else {
		t.Logf("Successfully got error on UnProject: %v", err)
	}
}

func TestLookAtV(t *testing.T) {
	// http://www.euclideanspace.com/maths/algebra/matrix/transforms/examples/index.htm

	tests := []struct {
		Description     string
		Eye, Center, Up Vec3
		Expected        Mat4
	}{
		{
			"forward",
			Vec3{0, 0, 0},
			Vec3{0, 0, -1},
			Vec3{0, 1, 0},
			Ident4(),
		},
		{
			"heading 90 degree",
			Vec3{0, 0, 0},
			Vec3{1, 0, 0},
			Vec3{0, 1, 0},
			Mat4{
				0, 0, -1, 0,
				0, 1, 0, 0,
				1, 0, 0, 0,
				0, 0, 0, 1,
			},
		},
		{
			"heading 180 degree",
			Vec3{0, 0, 0},
			Vec3{0, 0, 1},
			Vec3{0, 1, 0},
			Mat4{
				-1, 0, 0, 0,
				0, 1, 0, 0,
				0, 0, -1, 0,
				0, 0, 0, 1,
			},
		},
		{
			"attitude 90 degree",
			Vec3{0, 0, 0},
			Vec3{0, 0, -1},
			Vec3{1, 0, 0},
			Mat4{
				0, 1, 0, 0,
				-1, 0, 0, 0,
				0, 0, 1, 0,
				0, 0, 0, 1,
			},
		},
		{
			"bank 90 degree",
			Vec3{0, 0, 0},
			Vec3{0, -1, 0},
			Vec3{0, 0, -1},
			Mat4{
				1, 0, 0, 0,
				0, 0, 1, 0,
				0, -1, 0, 0,
				0, 0, 0, 1,
			},
		},
	}

	threshold := float64(math.Pow(10, -2))
	for _, c := range tests {
		if r := LookAtV(c.Eye, c.Center, c.Up); !r.ApproxEqualThreshold(c.Expected, threshold) {
			t.Errorf("%v failed: LookAtV(%v, %v, %v) != %v (got %v)", c.Description, c.Eye, c.Center, c.Up, c.Expected, r)
		}

		if r := LookAt(c.Eye[0], c.Eye[1], c.Eye[2], c.Center[0], c.Center[1], c.Center[2], c.Up[0], c.Up[1], c.Up[2]); !r.ApproxEqualThreshold(c.Expected, threshold) {
			t.Errorf("%v failed: LookAt(%v, %v, %v) != %v (got %v)", c.Description, c.Eye, c.Center, c.Up, c.Expected, r)
		}
	}
}

func TestOrtho(t *testing.T) {
	tests := []struct {
		Left, Right,
		Bottom, Top,
		Near, Far float64
		Expected Mat4
	}{
		{
			-1.0, 1.0, -1.0, 1.0, 1.0, -1.0,
			Ident4(),
		}, {
			-10.0, 10.0, -10.0, 10.0, 0.0, 100.0,
			Mat4{0.1, 0.0, 0.0, 0.0, 0.0, 0.1, 0.0, 0.0, 0.0, 0.0, -0.02, 0.0, 0.0, 0.0, -1.0, 1.0},
		}, {
			0.0, 10.0, 0.0, 10.0, 0.0, 100.0,
			Mat4{0.2, 0.0, 0.0, 0.0, 0.0, 0.2, 0.0, 0.0, 0.0, 0.0, -0.02, 0.0, -1.0, -1.0, -1.0, 1.0},
		},
	}

	for _, c := range tests {
		if r := Ortho(c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far); !r.ApproxEqualThreshold(c.Expected, 1e-4) {
			t.Errorf("Ortho(%v, %v, %v, %v, %v, %v) != %v (got %v)", c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far, c.Expected, r)
		}
	}
}

func TestOrtho2D(t *testing.T) {
	tests := []struct {
		Left, Right,
		Bottom, Top float64
		Expected Mat4
	}{
		{
			-1.0, 1.0, -1.0, 1.0,
			Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, -1, 0, 0, 0, 0, 1},
		}, {
			-10.0, 10.0, -10.0, 10.0,
			Mat4{0.1, 0.0, 0.0, 0.0, 0.0, 0.1, 0.0, 0.0, 0.0, 0.0, -1.0, 0.0, 0.0, 0.0, 0.0, 1.0},
		}, {
			0.0, 10.0, 0.0, 10.0,
			Mat4{0.2, 0.0, 0.0, 0.0, 0.0, 0.2, 0.0, 0.0, 0.0, 0.0, -1.0, 0.0, -1.0, -1.0, 0.0, 1.0},
		},
	}

	for _, c := range tests {
		if r := Ortho2D(c.Left, c.Right, c.Bottom, c.Top); !r.ApproxEqualThreshold(c.Expected, 1e-4) {
			t.Errorf("Ortho2D(%v, %v, %v, %v) != %v (got %v)", c.Left, c.Right, c.Bottom, c.Top, c.Expected, r)
		}
	}
}

func TestPerspective(t *testing.T) {
	tests := []struct {
		Fovy, Aspect,
		Near, Far float64
		Expected Mat4
	}{
		{
			DegToRad(450.0), 1.0, -1.0, 1.0,
			Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, -1, 0, 0, 1, 0},
		}, {
			DegToRad(45.0), 4.0 / 3.0, 0.1, 100.0,
			Mat4{1.810660, 0.0, 0.0, 0.0, 0.0, 2.4142134, 0.0, 0.0, 0.0, 0.0, -1.002002, -1.0, 0.0, 0.0, -0.2002002, 0.0},
		}, {
			DegToRad(90.0), 16.0 / 9.0, -1.0, 1.0,
			Mat4{0.562500, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, -0.0, -1.0, 0.0, 0.0, 1.0, 0.0},
		},
	}

	for _, c := range tests {
		if r := Perspective(c.Fovy, c.Aspect, c.Near, c.Far); !r.ApproxEqualThreshold(c.Expected, 1e-4) {
			t.Errorf("Perspective(%v, %v, %v, %v) != %v (got %v)", c.Fovy, c.Aspect, c.Near, c.Far, c.Expected, r)
		}
	}
}

func TestFrustum(t *testing.T) {
	tests := []struct {
		Left, Right,
		Bottom, Top,
		Near, Far float64
		Expected Mat4
	}{
		{
			-1.0, 1.0, -1.0, 1.0, 1.0, 2.0,
			Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, -3, -1, 0, 0, -4, 0},
		},
		// TODO: more tests
	}

	for _, c := range tests {
		if r := Frustum(c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far); !r.ApproxEqualThreshold(c.Expected, 1e-4) {
			t.Errorf("Frustum(%v, %v, %v, %v, %v, %v) != %v (got %v)", c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far, c.Expected, r)
		}
	}
}

// ndcX returns the normalized device coordinate x of a world point under the
// given view and projection matrices.
func stereoNDCX(p Vec3, view, proj Mat4) float64 {
	clip := proj.Mul4(view).Mul4x1(p.Vec4(1))
	return clip[0] / clip[3]
}

// Fixed reference scene: eye separation 0.06, convergence distance 2.
func stereoTestScene() (eye, center, up Vec3, eyeSep, convDist, fovy, aspect, near, far float64) {
	return Vec3{0, 0, 0}, Vec3{0, 0, -1}, Vec3{0, 1, 0},
		0.06, 2, DegToRad(60), 4.0 / 3.0, 0.1, 100
}

func TestStereoPerspectiveConvergence(t *testing.T) {
	t.Parallel()

	eye, center, up, eyeSep, convDist, fovy, aspect, near, far := stereoTestScene()
	leftView, rightView, leftProj, rightProj, err := StereoPerspective(eye, center, up, eyeSep, convDist, fovy, aspect, near, far)
	if err != nil {
		t.Fatalf("StereoPerspective returned error: %v", err)
	}

	// A target on the convergence plane (deliberately off the view axis in y)
	// must project to the same horizontal NDC position in both eyes, and to
	// the same position as under the central monocular camera.
	target := Vec3{0, 0.3, -2}
	xl := stereoNDCX(target, leftView, leftProj)
	xr := stereoNDCX(target, rightView, rightProj)
	xc := stereoNDCX(target, LookAtV(eye, center, up), Perspective(fovy, aspect, near, far))

	if math.Abs(float64(xl-xr)) > 1e-5 {
		t.Errorf("convergence plane target: left x %v != right x %v", xl, xr)
	}
	if math.Abs(float64(xl-xc)) > 1e-5 {
		t.Errorf("convergence plane target: stereo x %v != central x %v", xl, xc)
	}

	// Parallax direction: a point nearer than the convergence plane has
	// crossed disparity (left image x > right image x); a farther point has
	// uncrossed disparity (left image x < right image x).
	nearPt := Vec3{0, 0.3, -1}
	if xl, xr := stereoNDCX(nearPt, leftView, leftProj), stereoNDCX(nearPt, rightView, rightProj); !(xl > xr) {
		t.Errorf("near point: expected crossed disparity (left x > right x), got left %v right %v", xl, xr)
	}
	farPt := Vec3{0, 0.3, -10}
	if xl, xr := stereoNDCX(farPt, leftView, leftProj), stereoNDCX(farPt, rightView, rightProj); !(xl < xr) {
		t.Errorf("far point: expected uncrossed disparity (left x < right x), got left %v right %v", xl, xr)
	}
}

func TestStereoPerspectiveLeftRightSwap(t *testing.T) {
	t.Parallel()

	eye, center, up, eyeSep, convDist, fovy, aspect, near, far := stereoTestScene()
	leftView, rightView, leftProj, rightProj, err := StereoPerspective(eye, center, up, eyeSep, convDist, fovy, aspect, near, far)
	if err != nil {
		t.Fatalf("StereoPerspective returned error: %v", err)
	}

	// The rig is symmetric about the view plane x=0: mirroring a point and
	// swapping eyes must mirror the projected x coordinate.
	for _, p := range []Vec3{{0.5, 0.2, -1}, {-0.7, -0.4, -2}, {0.1, 0.9, -10}} {
		mirror := Vec3{-p[0], p[1], p[2]}
		xl := stereoNDCX(p, leftView, leftProj)
		xr := stereoNDCX(mirror, rightView, rightProj)
		if math.Abs(float64(xl+xr)) > 1e-5 {
			t.Errorf("swap symmetry failed for %v: left x %v, mirrored right x %v", p, xl, xr)
		}
	}

	// The eyes sit at +-eyeSep/2 along the camera's right (+X) axis: each
	// eye position maps to the origin of its own view space.
	for i, tc := range []struct {
		eyePos Vec3
		view   Mat4
	}{
		{Vec3{-0.03, 0, 0}, leftView},
		{Vec3{0.03, 0, 0}, rightView},
	} {
		if o := tc.view.Mul4x1(tc.eyePos.Vec4(1)); !o.Vec3().ApproxEqualThreshold(Vec3{}, 1e-5) {
			t.Errorf("eye %d: view does not place eye at origin (got %v)", i, o.Vec3())
		}
	}
}

func TestStereoPerspectiveZeroEyeSep(t *testing.T) {
	t.Parallel()

	eye, center, up, _, _, fovy, aspect, near, far := stereoTestScene()
	leftView, rightView, leftProj, rightProj, err := StereoPerspective(eye, center, up, 0, 2, fovy, aspect, near, far)
	if err != nil {
		t.Fatalf("StereoPerspective returned error: %v", err)
	}

	monoView := LookAtV(eye, center, up)
	monoProj := Perspective(fovy, aspect, near, far)

	if !leftView.ApproxEqualThreshold(monoView, 1e-6) || !rightView.ApproxEqualThreshold(monoView, 1e-6) {
		t.Errorf("zero eye separation: views differ from monocular LookAtV")
	}
	if !leftProj.ApproxEqualThreshold(monoProj, 1e-6) || !rightProj.ApproxEqualThreshold(monoProj, 1e-6) {
		t.Errorf("zero eye separation: projections differ from monocular Perspective")
	}
}

func TestStereoPerspectiveErrors(t *testing.T) {
	t.Parallel()

	eye, center, up, eyeSep, convDist, fovy, aspect, near, far := stereoTestScene()
	nan := float64(math.NaN())
	inf := float64(math.Inf(1))

	tests := []struct {
		Description                          string
		Eye, Center, Up                      Vec3
		EyeSep, ConvDist, Fovy, Aspect, N, F float64
	}{
		{"NaN eye separation", eye, center, up, nan, convDist, fovy, aspect, near, far},
		{"infinite convergence", eye, center, up, eyeSep, inf, fovy, aspect, near, far},
		{"NaN eye", Vec3{nan, 0, 0}, center, up, eyeSep, convDist, fovy, aspect, near, far},
		{"negative eye separation", eye, center, up, -0.06, convDist, fovy, aspect, near, far},
		{"zero convergence", eye, center, up, eyeSep, 0, fovy, aspect, near, far},
		{"negative convergence", eye, center, up, eyeSep, -2, fovy, aspect, near, far},
		{"zero near", eye, center, up, eyeSep, convDist, fovy, aspect, 0, far},
		{"far before near", eye, center, up, eyeSep, convDist, fovy, aspect, 10, 1},
		{"far equals near", eye, center, up, eyeSep, convDist, fovy, aspect, 1, 1},
		{"zero fovy", eye, center, up, eyeSep, convDist, 0, aspect, near, far},
		{"fovy of pi", eye, center, up, eyeSep, convDist, float64(math.Pi), aspect, near, far},
		{"negative aspect", eye, center, up, eyeSep, convDist, fovy, -1, near, far},
		{"eye equals center", eye, eye, up, eyeSep, convDist, fovy, aspect, near, far},
		{"zero up", eye, center, Vec3{}, eyeSep, convDist, fovy, aspect, near, far},
		{"up parallel to view", eye, center, Vec3{0, 0, 1}, eyeSep, convDist, fovy, aspect, near, far},
	}

	for _, c := range tests {
		if _, _, _, _, err := StereoPerspective(c.Eye, c.Center, c.Up, c.EyeSep, c.ConvDist, c.Fovy, c.Aspect, c.N, c.F); err == nil {
			t.Errorf("%v: expected error, got nil", c.Description)
		}
	}
}
