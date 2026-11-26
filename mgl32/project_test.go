// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl32

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

	threshold := float32(math.Pow(10, -2))
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
		Near, Far float32
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
		Bottom, Top float32
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
		Near, Far float32
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
		Near, Far float32
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

// clipDepthZ applies m to the on-axis eye-space point (0, 0, eyeZ) and
// returns its normalized device depth after perspective division.
func clipDepthZ(t *testing.T, m Mat4, eyeZ float32) float32 {
	t.Helper()
	clip := m.Mul4x1(Vec4{0, 0, eyeZ, 1})
	return clip[2] / clip[3]
}

func TestPerspectiveReverseZ(t *testing.T) {
	t.Parallel()

	fovy, aspect, near, far := DegToRad(90.0), float32(16.0/9.0), float32(1.0), float32(3.0)

	expected := Mat4{0.56249994, 0, 0, 0, 0, 0.99999994, 0, 0, 0, 0, 2, -1, 0, 0, 3, 0}
	m, err := PerspectiveReverseZ(fovy, aspect, near, far)
	if err != nil {
		t.Fatalf("PerspectiveReverseZ returned unexpected error: %v", err)
	}
	if !m.ApproxEqualThreshold(expected, 1e-4) {
		t.Errorf("PerspectiveReverseZ(%v, %v, %v, %v) != %v (got %v)", fovy, aspect, near, far, expected, m)
	}

	// Near plane (eye z = -near) maps to clip depth +1, far plane to -1.
	if z := clipDepthZ(t, m, -near); !floatEqual(z, 1, 1e-4) {
		t.Errorf("near plane maps to clip depth %v, expected +1", z)
	}
	if z := clipDepthZ(t, m, -far); !floatEqual(z, -1, 1e-4) {
		t.Errorf("far plane maps to clip depth %v, expected -1", z)
	}

	// The x/y extent must be identical to the ordinary Perspective matrix.
	standard := Perspective(fovy, aspect, near, far)
	point := Vec4{0.5, -0.25, -2, 1}
	got := m.Mul4x1(point)
	want := standard.Mul4x1(point)
	if !floatEqual(got[0]/got[3], want[0]/want[3], 1e-4) ||
		!floatEqual(got[1]/got[3], want[1]/want[3], 1e-4) {
		t.Errorf("reverse-Z perspective changed x/y mapping: got (%v, %v), want (%v, %v)",
			got[0]/got[3], got[1]/got[3], want[0]/want[3], want[1]/want[3])
	}
}

func TestPerspectiveReverseZInf(t *testing.T) {
	t.Parallel()

	fovy, aspect, near := DegToRad(90.0), float32(16.0/9.0), float32(1.0)

	expected := Mat4{0.56249994, 0, 0, 0, 0, 0.99999994, 0, 0, 0, 0, 1, -1, 0, 0, 2, 0}
	m, err := PerspectiveReverseZInf(fovy, aspect, near)
	if err != nil {
		t.Fatalf("PerspectiveReverseZInf returned unexpected error: %v", err)
	}
	if !m.ApproxEqualThreshold(expected, 1e-4) {
		t.Errorf("PerspectiveReverseZInf(%v, %v, %v) != %v (got %v)", fovy, aspect, near, expected, m)
	}

	// Near plane maps to clip depth +1; infinite depth approaches -1.
	if z := clipDepthZ(t, m, -near); !floatEqual(z, 1, 1e-4) {
		t.Errorf("near plane maps to clip depth %v, expected +1", z)
	}
	if z := clipDepthZ(t, m, float32(-1e18)); !floatEqual(z, -1, 1e-4) {
		t.Errorf("infinite depth maps to clip depth %v, expected -1", z)
	}

	// The x/y extent must match the ordinary Perspective matrix (the
	// perspective scale depends only on fovy and aspect).
	standard := Perspective(fovy, aspect, near, 100)
	point := Vec4{0.5, -0.25, -2, 1}
	got := m.Mul4x1(point)
	want := standard.Mul4x1(point)
	if !floatEqual(got[0]/got[3], want[0]/want[3], 1e-4) ||
		!floatEqual(got[1]/got[3], want[1]/want[3], 1e-4) {
		t.Errorf("reverse-Z infinite perspective changed x/y mapping: got (%v, %v), want (%v, %v)",
			got[0]/got[3], got[1]/got[3], want[0]/want[3], want[1]/want[3])
	}
}

func TestOrthoReverseZ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Left, Right,
		Bottom, Top,
		Near, Far float32
		Expected Mat4
	}{
		{
			-1.0, 1.0, -1.0, 1.0, 1.0, 3.0,
			Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 2, 1},
		}, {
			0.0, 10.0, 0.0, 10.0, 1.0, 100.0,
			Mat4{0.2, 0, 0, 0, 0, 0.2, 0, 0, 0, 0, 0.02020202, 0, -1, -1, 1.020202, 1},
		},
	}

	for _, c := range tests {
		m, err := OrthoReverseZ(c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far)
		if err != nil {
			t.Errorf("OrthoReverseZ(%v, %v, %v, %v, %v, %v) returned unexpected error: %v",
				c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far, err)
			continue
		}
		if !m.ApproxEqualThreshold(c.Expected, 1e-4) {
			t.Errorf("OrthoReverseZ(%v, %v, %v, %v, %v, %v) != %v (got %v)",
				c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far, c.Expected, m)
		}

		// Near plane maps to clip depth +1, far plane to -1.
		if z := clipDepthZ(t, m, -c.Near); !floatEqual(z, 1, 1e-4) {
			t.Errorf("near plane maps to clip depth %v, expected +1", z)
		}
		if z := clipDepthZ(t, m, -c.Far); !floatEqual(z, -1, 1e-4) {
			t.Errorf("far plane maps to clip depth %v, expected -1", z)
		}

		// The x/y bounds must match the ordinary Ortho matrix.
		standard := Ortho(c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far)
		point := Vec4{(c.Left + c.Right) / 2, (c.Bottom + c.Top) / 2, -(c.Near + c.Far) / 2, 1}
		got := m.Mul4x1(point)
		want := standard.Mul4x1(point)
		if !floatEqual(got[0], want[0], 1e-4) || !floatEqual(got[1], want[1], 1e-4) {
			t.Errorf("reverse-Z ortho changed x/y mapping: got (%v, %v), want (%v, %v)",
				got[0], got[1], want[0], want[1])
		}
	}
}

func TestReverseZInvalidParameters(t *testing.T) {
	t.Parallel()

	pi := float32(math.Pi)

	perspectiveCases := []struct {
		Name                    string
		Fovy, Aspect, Near, Far float32
	}{
		{"zero near", DegToRad(90), 1, 0, 3},
		{"negative near", DegToRad(90), 1, -1, 3},
		{"far equal near", DegToRad(90), 1, 1, 1},
		{"far below near", DegToRad(90), 1, 3, 1},
		{"zero fovy", 0, 1, 1, 3},
		{"negative fovy", DegToRad(-45), 1, 1, 3},
		{"fovy at pi", pi, 1, 1, 3},
		{"fovy beyond pi", pi * 1.5, 1, 1, 3},
		{"zero aspect", DegToRad(90), 0, 1, 3},
		{"negative aspect", DegToRad(90), -1, 1, 3},
		{"NaN fovy", NaN, 1, 1, 3},
		{"infinite far", DegToRad(90), 1, 1, InfPos},
		{"NaN near", DegToRad(90), 1, NaN, 3},
	}
	for _, c := range perspectiveCases {
		if m, err := PerspectiveReverseZ(c.Fovy, c.Aspect, c.Near, c.Far); err == nil {
			t.Errorf("PerspectiveReverseZ(%v) should fail, got matrix %v", c.Name, m)
		} else if m != (Mat4{}) {
			t.Errorf("PerspectiveReverseZ(%v) should return a zero matrix on error, got %v", c.Name, m)
		}
	}

	infCases := []struct {
		Name               string
		Fovy, Aspect, Near float32
	}{
		{"zero near", DegToRad(90), 1, 0},
		{"negative near", DegToRad(90), 1, -1},
		{"zero fovy", 0, 1, 1},
		{"fovy at pi", pi, 1, 1},
		{"zero aspect", DegToRad(90), 0, 1},
		{"negative aspect", DegToRad(90), -1, 1},
		{"NaN aspect", DegToRad(90), NaN, 1},
		{"infinite near", DegToRad(90), 1, InfPos},
	}
	for _, c := range infCases {
		if m, err := PerspectiveReverseZInf(c.Fovy, c.Aspect, c.Near); err == nil {
			t.Errorf("PerspectiveReverseZInf(%v) should fail, got matrix %v", c.Name, m)
		} else if m != (Mat4{}) {
			t.Errorf("PerspectiveReverseZInf(%v) should return a zero matrix on error, got %v", c.Name, m)
		}
	}

	orthoCases := []struct {
		Name                                string
		Left, Right, Bottom, Top, Near, Far float32
	}{
		{"zero near", -1, 1, -1, 1, 0, 3},
		{"negative near", -1, 1, -1, 1, -1, 3},
		{"far equal near", -1, 1, -1, 1, 1, 1},
		{"far below near", -1, 1, -1, 1, 3, 1},
		{"left equal right", 1, 1, -1, 1, 1, 3},
		{"bottom equal top", -1, 1, 2, 2, 1, 3},
		{"NaN left", NaN, 1, -1, 1, 1, 3},
		{"infinite far", -1, 1, -1, 1, 1, InfPos},
	}
	for _, c := range orthoCases {
		if m, err := OrthoReverseZ(c.Left, c.Right, c.Bottom, c.Top, c.Near, c.Far); err == nil {
			t.Errorf("OrthoReverseZ(%v) should fail, got matrix %v", c.Name, m)
		} else if m != (Mat4{}) {
			t.Errorf("OrthoReverseZ(%v) should return a zero matrix on error, got %v", c.Name, m)
		}
	}
}

// floatEqual compares two float32 values against an absolute threshold.
func floatEqual(a, b, threshold float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= threshold
}
