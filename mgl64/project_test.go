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

func TestCascadeSplitDepths(t *testing.T) {
	t.Parallel()

	depths, err := CascadeSplitDepths(1, 16, 2, 0.5)
	if err != nil {
		t.Fatalf("CascadeSplitDepths returned error: %v", err)
	}
	expected := []float64{1, 6.25, 16}
	if len(depths) != len(expected) {
		t.Fatalf("CascadeSplitDepths returned %d depths, want %d", len(depths), len(expected))
	}
	for i := range expected {
		if !FloatEqualThreshold(depths[i], expected[i], 1e-5) {
			t.Errorf("depth %d = %v, want %v", i, depths[i], expected[i])
		}
	}

	// lambda = 0 is purely linear, lambda = 1 purely logarithmic.
	linear, err := CascadeSplitDepths(1, 16, 4, 0)
	if err != nil {
		t.Fatalf("linear CascadeSplitDepths returned error: %v", err)
	}
	wantLinear := []float64{1, 4.75, 8.5, 12.25, 16}
	for i := range wantLinear {
		if !FloatEqualThreshold(linear[i], wantLinear[i], 1e-5) {
			t.Errorf("linear depth %d = %v, want %v", i, linear[i], wantLinear[i])
		}
	}

	logarithmic, err := CascadeSplitDepths(1, 16, 4, 1)
	if err != nil {
		t.Fatalf("logarithmic CascadeSplitDepths returned error: %v", err)
	}
	wantLog := []float64{1, 2, 4, 8, 16}
	for i := range wantLog {
		if !FloatEqualThreshold(logarithmic[i], wantLog[i], 1e-5) {
			t.Errorf("logarithmic depth %d = %v, want %v", i, logarithmic[i], wantLog[i])
		}
	}

	// Boundaries must be strictly increasing for any legal blend weight.
	for _, lambda := range []float64{0, 0.25, 0.5, 0.75, 1} {
		d, err := CascadeSplitDepths(1, 16, 4, lambda)
		if err != nil {
			t.Fatalf("CascadeSplitDepths(1, 16, 4, %v) returned error: %v", lambda, err)
		}
		for i := 1; i < len(d); i++ {
			if !(d[i] > d[i-1]) {
				t.Errorf("lambda %v: depths are not increasing at %d: %v", lambda, i, d)
			}
		}
	}
}

func TestCascadeSplits(t *testing.T) {
	t.Parallel()

	// An off-axis perspective frustum, with the identity view matrix so
	// the corners are directly readable as world coordinates.
	projection := Frustum(-1, 3, -2, 1, 1, 16)
	view := Ident4()

	splits, err := CascadeSplits(view, projection, 1, 16, 2, 0.5)
	if err != nil {
		t.Fatalf("CascadeSplits returned error: %v", err)
	}
	if len(splits) != 2 {
		t.Fatalf("CascadeSplits returned %d splits, want 2", len(splits))
	}

	if !FloatEqualThreshold(splits[0].Near, 1, 1e-5) || !FloatEqualThreshold(splits[0].Far, 6.25, 1e-5) {
		t.Errorf("first split depth interval = [%v, %v], want [1, 6.25]", splits[0].Near, splits[0].Far)
	}
	if !FloatEqualThreshold(splits[1].Near, 6.25, 1e-5) || !FloatEqualThreshold(splits[1].Far, 16, 1e-5) {
		t.Errorf("second split depth interval = [%v, %v], want [6.25, 16]", splits[1].Near, splits[1].Far)
	}

	// At depth d the off-axis frustum spans x in [d*left/near, d*right/near]
	// and y in [d*bottom/near, d*top/near]; the view axis maps to z = -d.
	atDepth := func(d float64) [8]Vec3 {
		x0, x1 := d*(-1)/1, d*3/1
		y0, y1 := d*(-2)/1, d*1/1
		return [8]Vec3{
			{x0, y0, -d}, {x1, y0, -d}, {x1, y1, -d}, {x0, y1, -d},
			{x0, y0, -d}, {x1, y0, -d}, {x1, y1, -d}, {x0, y1, -d},
		}
	}
	nearCorners := atDepth(1)
	midCorners := atDepth(6.25)
	farCorners := atDepth(16)
	expected := [2][8]Vec3{}
	expected[0] = [8]Vec3{
		nearCorners[0], nearCorners[1], nearCorners[2], nearCorners[3],
		midCorners[0], midCorners[1], midCorners[2], midCorners[3],
	}
	expected[1] = [8]Vec3{
		midCorners[0], midCorners[1], midCorners[2], midCorners[3],
		farCorners[0], farCorners[1], farCorners[2], farCorners[3],
	}

	for s := range splits {
		for i := range splits[s].Corners {
			if !splits[s].Corners[i].ApproxEqualThreshold(expected[s][i], 1e-4) {
				t.Errorf("split %d corner %d = %v, want %v", s, i, splits[s].Corners[i], expected[s][i])
			}
		}
	}

	// Neighbouring splits must share the boundary cross-section exactly.
	for i := 0; i < 4; i++ {
		if splits[0].Corners[i+4] != splits[1].Corners[i] {
			t.Errorf("shared boundary corner %d differs: %v vs %v", i, splits[0].Corners[i+4], splits[1].Corners[i])
		}
	}

	// NearCorners / FarCorners mirror the flat layout.
	for i := 0; i < 4; i++ {
		if splits[0].NearCorners()[i] != splits[0].Corners[i] {
			t.Errorf("NearCorners()[%d] = %v, want %v", i, splits[0].NearCorners()[i], splits[0].Corners[i])
		}
		if splits[0].FarCorners()[i] != splits[0].Corners[i+4] {
			t.Errorf("FarCorners()[%d] = %v, want %v", i, splits[0].FarCorners()[i], splits[0].Corners[i+4])
		}
	}

	// The concrete acceptance case: near=1, far=16, two splits, blend 0.5,
	// with a symmetric 90 degree perspective camera (aspect 1) so the
	// corners have exact values.
	persp := Perspective(DegToRad(90), 1, 1, 16)
	psplits, err := CascadeSplits(Ident4(), persp, 1, 16, 2, 0.5)
	if err != nil {
		t.Fatalf("CascadeSplits(Perspective) returned error: %v", err)
	}
	section := func(d float64) [4]Vec3 {
		return [4]Vec3{{-d, -d, -d}, {d, -d, -d}, {d, d, -d}, {-d, d, -d}}
	}
	expectedNear := section(1)
	expectedMid := section(6.25)
	expectedFar := section(16)
	want := [2][8]Vec3{
		{expectedNear[0], expectedNear[1], expectedNear[2], expectedNear[3], expectedMid[0], expectedMid[1], expectedMid[2], expectedMid[3]},
		{expectedMid[0], expectedMid[1], expectedMid[2], expectedMid[3], expectedFar[0], expectedFar[1], expectedFar[2], expectedFar[3]},
	}
	if !FloatEqualThreshold(psplits[0].Near, 1, 1e-5) || !FloatEqualThreshold(psplits[0].Far, 6.25, 1e-5) ||
		!FloatEqualThreshold(psplits[1].Near, 6.25, 1e-5) || !FloatEqualThreshold(psplits[1].Far, 16, 1e-5) {
		t.Errorf("Perspective splits = [%v, %v] [%v, %v], want [1, 6.25] [6.25, 16]",
			psplits[0].Near, psplits[0].Far, psplits[1].Near, psplits[1].Far)
	}
	for s := range psplits {
		for i := range psplits[s].Corners {
			if !psplits[s].Corners[i].ApproxEqualThreshold(want[s][i], 1e-4) {
				t.Errorf("Perspective split %d corner %d = %v, want %v", s, i, psplits[s].Corners[i], want[s][i])
			}
		}
		t.Logf("split %d depth [%v, %v] corners %v", s, psplits[s].Near, psplits[s].Far, psplits[s].Corners)
	}
}

func TestFrustumCorners(t *testing.T) {
	t.Parallel()

	projection := Frustum(-1, 3, -2, 1, 1, 16)

	corners, err := FrustumCorners(Ident4(), projection, 1, 16)
	if err != nil {
		t.Fatalf("FrustumCorners returned error: %v", err)
	}

	expected := [8]Vec3{
		{-1, -2, -1}, {3, -2, -1}, {3, 1, -1}, {-1, 1, -1},
		{-16, -32, -16}, {48, -32, -16}, {48, 16, -16}, {-16, 16, -16},
	}
	for i := range expected {
		if !corners[i].ApproxEqualThreshold(expected[i], 1e-4) {
			t.Errorf("frustum corner %d = %v, want %v", i, corners[i], expected[i])
		}
	}

	// A bright sanity check: the far corners of a symmetric Perspective
	// camera must lie on the view axis' side planes.
	proj := Perspective(DegToRad(90), 1, 1, 16)
	c, err := FrustumCorners(Ident4(), proj, 1, 16)
	if err != nil {
		t.Fatalf("FrustumCorners(Perspective) returned error: %v", err)
	}
	// With a 90 degree vertical field of view and aspect 1 the far plane
	// spans [-16, 16] on both axes.
	if !c[4].ApproxEqualThreshold(Vec3{-16, -16, -16}, 1e-4) {
		t.Errorf("Perspective far bottom left = %v, want %v", c[4], Vec3{-16, -16, -16})
	}
	if !c[7].ApproxEqualThreshold(Vec3{-16, 16, -16}, 1e-4) {
		t.Errorf("Perspective far top left = %v, want %v", c[7], Vec3{-16, 16, -16})
	}
}

func TestCascadeSplitsRoundTrip(t *testing.T) {
	t.Parallel()

	// Use a real camera transform to make sure the corners are world-space
	// points and project back onto the frustum boundaries.
	projection := Frustum(-0.5, 1.5, -2, 1, 0.5, 100)
	view := LookAtV(Vec3{3, 4, 5}, Vec3{0, 0, 0}, Vec3{0, 1, 0})

	splits, err := CascadeSplits(view, projection, 0.5, 100, 3, 0.5)
	if err != nil {
		t.Fatalf("CascadeSplits returned error: %v", err)
	}

	ndcX := [4]float64{-1, 1, 1, -1}
	ndcY := [4]float64{-1, -1, 1, 1}
	for s := range splits {
		for i := 0; i < 8; i++ {
			corner := splits[s].Corners[i]
			clip := projection.Mul4(view).Mul4x1(corner.Vec4(1))
			if clip.W() == 0 {
				t.Fatalf("split %d corner %d has zero clip w", s, i)
			}
			ndc := Vec3{clip.X() / clip.W(), clip.Y() / clip.W(), clip.Z() / clip.W()}
			if !ndc.ApproxEqualThreshold(Vec3{ndcX[i%4], ndcY[i%4], ndc[2]}, 1e-4) {
				t.Errorf("split %d corner %d projects to x/y (%v, %v), want (%v, %v)", s, i, ndc[0], ndc[1], ndcX[i%4], ndcY[i%4])
			}

			// The recovered point must sit at the split boundary depth.
			depth := splits[s].Near
			if i >= 4 {
				depth = splits[s].Far
			}
			viewPos := view.Mul4x1(corner.Vec4(1))
			if viewPos.W() == 0 || !FloatEqualThreshold(viewPos.Z()/viewPos.W(), -depth, 1e-4) {
				t.Errorf("split %d corner %d has view depth %v, want %v", s, i, viewPos.Z()/viewPos.W(), -depth)
			}
			if ndc[2] < -1-1e-4 || ndc[2] > 1+1e-4 {
				t.Errorf("split %d corner %d has clip depth %v outside [-1, 1]", s, i, ndc[2])
			}
		}
	}
}

func TestCascadeSplitsErrors(t *testing.T) {
	t.Parallel()

	projection := Frustum(-1, 1, -1, 1, 1, 16)

	badDepth := []struct{ near, far float64 }{
		{0, 16}, {-1, 16}, {1, 1}, {2, 1}, {1, 0}, {float64(math.Inf(1)), 16}, {1, float64(math.Inf(1))},
	}
	for _, c := range badDepth {
		if _, err := CascadeSplits(Ident4(), projection, c.near, c.far, 2, 0.5); err == nil {
			t.Errorf("CascadeSplits(near=%v, far=%v) did not return an error", c.near, c.far)
		}
	}

	badLambda := []float64{-0.1, 1.1, float64(math.NaN()), float64(math.Inf(1))}
	for _, lambda := range badLambda {
		if _, err := CascadeSplits(Ident4(), projection, 1, 16, 2, lambda); err == nil {
			t.Errorf("CascadeSplits(lambda=%v) did not return an error", lambda)
		}
	}

	if _, err := CascadeSplits(Ident4(), projection, 1, 16, 0, 0.5); err == nil {
		t.Error("CascadeSplits with zero cascades did not return an error")
	}

	if _, err := CascadeSplitDepths(1, 16, -1, 0.5); err == nil {
		t.Error("CascadeSplitDepths with a negative cascade count did not return an error")
	}

	if _, err := CascadeSplits(Mat4{}, Mat4{}, 1, 16, 2, 0.5); err == nil {
		t.Error("CascadeSplits with singular view/projection did not return an error")
	}

	if _, err := FrustumCorners(Mat4{}, Mat4{}, 1, 16); err == nil {
		t.Error("FrustumCorners with singular view/projection did not return an error")
	}

	infProjection := projection
	infProjection[0] = float64(math.Inf(1))
	if _, err := CascadeSplits(Ident4(), infProjection, 1, 16, 2, 0.5); err == nil {
		t.Error("CascadeSplits with a non-finite projection did not return an error")
	}

	if _, err := CascadeSplits(Ident4(), projection, 0, 16, 2, 0.5); err == nil {
		t.Error("CascadeSplits with a zero near plane did not return an error")
	}

	if _, err := CascadeSplitDepths(1, 16, 2, 1.5); err == nil {
		t.Error("CascadeSplitDepths with an out-of-range lambda did not return an error")
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
