// Copyright 2018 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.package mgl64_test

package mgl32

import (
	"math"
	"reflect"
	"testing"
)

func TestScreenToGLCoords(t *testing.T) {
	// use a small screen size in order to minimize errors due to fp rounding.
	const (
		sw = 100
		sh = 100
	)
	x, y := ScreenToGLCoords(0, sh-1, sw, sh)
	if x != -1 {
		t.Errorf("x = %f, expected -1.0", x)
	}
	if y != -1 {
		t.Errorf("y = %f, expected -1.0", y)
	}

	x, y = ScreenToGLCoords(sw-1, 0, sw, sh)
	if x != 1 {
		t.Errorf("x = %f, expected 1.0", x)
	}
	if y != 1 {
		t.Errorf("y = %f, expected 1.0", y)
	}
}

func TestGLToScreenCoords(t *testing.T) {
	const (
		sw = 100
		sh = 100
	)
	x, y := GLToScreenCoords(-1, -1, sw, sh)
	if x != 0 {
		t.Errorf("x = %d, expected 0", x)
	}
	if y != sh-1 {
		t.Errorf("y = %d, expected %d", y, sh-1)
	}

	x, y = GLToScreenCoords(1, 1, sw, sh)
	if x != sw-1 {
		t.Errorf("x = %d, expected %d", x, sw-1)
	}
	if y != 0 {
		t.Errorf("y = %d, expected 0", y)
	}
}

func Test_choose(t *testing.T) {

	tests := []struct {
		name string
		n    int
		want []int
	}{
		// test cases.
		{
			name: "C(2,k)",
			n:    2,
			want: []int{1, 2, 1},
		},
		{
			name: "C(3,k)",
			n:    3,
			want: []int{1, 3, 3, 1},
		},
		{
			name: "C(4,k)",
			n:    4,
			want: []int{1, 4, 6, 4, 1},
		},
		{
			name: "C(5,k)",
			n:    5,
			want: []int{1, 5, 10, 10, 5, 1},
		},
		{
			name: "C(6,k)",
			n:    6,
			want: []int{1, 6, 15, 20, 25, 6, 1},
		},
	}
	for _, tt := range tests {
		// t.Run(tt.name, func(t *testing.T) {
		got := []int{}
		fail := false
		for k := 0; k < tt.n; k++ {
			got = append(got, choose(tt.n, k))
			fail = got[k] != tt.want[k]
		}
		if fail {
			t.Errorf("choose() = %v, want %v", got, tt.want)
		}
		// })
	}
}

// flatUnitSquareGrid is the bilinear control grid whose surface is the flat
// unit XY square with normal +Z.
func flatUnitSquareGrid() [][]Vec3 {
	return [][]Vec3{
		{{0, 0, 0}, {1, 0, 0}},
		{{0, 1, 0}, {1, 1, 0}},
	}
}

// raisedCenterGrid is a quadratic Bezier grid identical to the flat unit
// square except for a lifted interior control point.
func raisedCenterGrid(height float32) [][]Vec3 {
	return [][]Vec3{
		{{0, 0, 0}, {0.5, 0, 0}, {1, 0, 0}},
		{{0, 0.5, 0}, {0.5, 0.5, height}, {1, 0.5, 0}},
		{{0, 1, 0}, {0.5, 1, 0}, {1, 1, 0}},
	}
}

// meshEdges reconstructs the set of oriented edges of every triangle. Each
// entry maps an edge (smaller endpoint index first) to its two oriented uses;
// a watertight manifold uses every interior edge exactly twice in opposite
// directions and each boundary edge exactly once.
func meshEdges(mesh BezierSurfaceMesh) map[[2]uint32][][2]uint32 {
	edges := make(map[[2]uint32][][2]uint32)
	for k := 0; k+2 < len(mesh.Indices); k += 3 {
		tri := [3]uint32{mesh.Indices[k], mesh.Indices[k+1], mesh.Indices[k+2]}
		for e := 0; e < 3; e++ {
			a, b := tri[e], tri[(e+1)%3]
			key := [2]uint32{a, b}
			if a > b {
				key = [2]uint32{b, a}
			}
			edges[key] = append(edges[key], [2]uint32{a, b})
		}
	}
	return edges
}

// assertWatertightAndWound checks the manifold, seam and winding invariants
// shared by every valid tessellation.
func assertWatertightAndWound(t *testing.T, mesh BezierSurfaceMesh, name string) {
	t.Helper()
	if len(mesh.Indices)%3 != 0 {
		t.Fatalf("%s: index count %d is not a multiple of 3", name, len(mesh.Indices))
	}
	nv := len(mesh.Vertices)
	if len(mesh.UVs) != nv || len(mesh.Normals) != nv {
		t.Fatalf("%s: parallel slices disagree: %d vertices, %d uvs, %d normals",
			name, nv, len(mesh.UVs), len(mesh.Normals))
	}
	for _, idx := range mesh.Indices {
		if int(idx) >= nv {
			t.Fatalf("%s: index %d out of range for %d vertices", name, idx, nv)
		}
	}

	edges := meshEdges(mesh)
	for key, uses := range edges {
		if len(uses) > 2 {
			t.Errorf("%s: edge %v used by %d triangles, expected 1 or 2",
				name, key, len(uses))
			continue
		}
		if len(uses) == 2 && uses[0] == uses[1] {
			t.Errorf("%s: edge %v wound the same way by both adjacent triangles", name, key)
		}
	}

	// Every triangle must face d/du x d/v: the geometric normal of its
	// vertices has a non-negative dot product with the stored unit normals.
	const tol = float32(1e-3)
	for k := 0; k+2 < len(mesh.Indices); k += 3 {
		a := mesh.Vertices[mesh.Indices[k]]
		b := mesh.Vertices[mesh.Indices[k+1]]
		c := mesh.Vertices[mesh.Indices[k+2]]
		geoNormal := b.Sub(a).Cross(c.Sub(a))
		if geoNormal.LenSqr() == 0 {
			t.Errorf("%s: triangle %d is degenerate", name, k/3)
			continue
		}
		geoNormal = geoNormal.Normalize()
		for _, idx := range mesh.Indices[k : k+3] {
			n := mesh.Normals[idx]
			if n.LenSqr() == 0 {
				continue // undefined normal on a collapsed patch
			}
			if n.Dot(geoNormal) < -tol {
				t.Errorf("%s: triangle %d wound against its stored normals (dot=%g)",
					name, k/3, n.Dot(geoNormal))
			}
		}
	}
}

// assertVerticesOnSurface checks every emitted vertex lies on the exact
// Bezier surface at its recorded (u, v).
func assertVerticesOnSurface(t *testing.T, cPoints [][]Vec3, mesh BezierSurfaceMesh, name string) {
	t.Helper()
	const tol = float32(1e-5)
	for i, uv := range mesh.UVs {
		want := BezierSurface(uv[0], uv[1], cPoints)
		if got := mesh.Vertices[i]; got.Sub(want).Len() > tol {
			t.Errorf("%s: vertex %d at uv %v = %v, surface says %v",
				name, i, uv, got, want)
		}
	}
}

func TestTessellateBezierSurfaceFlatSquare(t *testing.T) {
	grid := flatUnitSquareGrid()
	mesh := TessellateBezierSurface(grid, 1e-3)

	if len(mesh.Indices) != 6 {
		t.Fatalf("flat surface produced %d indices, expected 6 (two triangles)",
			len(mesh.Indices))
	}
	if len(mesh.Vertices) != 4 {
		t.Fatalf("flat surface produced %d vertices, expected 4 corners",
			len(mesh.Vertices))
	}

	// The two triangles split the unit square along one diagonal; both
	// triangle centroids and the overall centroid sit at (0.5, 0.5, 0).
	var centroid Vec3
	for _, p := range mesh.Vertices {
		centroid = centroid.Add(p)
	}
	centroid = centroid.Mul(1.0 / float32(len(mesh.Vertices)))
	if want := (Vec3{0.5, 0.5, 0}); centroid.Sub(want).Len() > 1e-6 {
		t.Errorf("mesh centroid = %v, want %v", centroid, want)
	}
	var triCentroidSum Vec3
	for k := 0; k+2 < len(mesh.Indices); k += 3 {
		var tc Vec3
		for _, idx := range mesh.Indices[k : k+3] {
			tc = tc.Add(mesh.Vertices[idx])
		}
		tc = tc.Mul(1.0 / 3.0)
		triCentroidSum = triCentroidSum.Add(tc)
	}
	triCentroid := triCentroidSum.Mul(0.5)
	if want := (Vec3{0.5, 0.5, 0}); triCentroid.Sub(want).Len() > 1e-6 {
		t.Errorf("triangle-centroid average = %v, want %v", triCentroid, want)
	}

	// Every corner normal is the unit +Z normal.
	for i, n := range mesh.Normals {
		if n.Sub(Vec3{0, 0, 1}).Len() > 1e-6 {
			t.Errorf("normal %d = %v, want (0,0,1)", i, n)
		}
	}
	// UVs cover the four corners of the parameter square exactly once.
	wantUVs := map[Vec2]bool{
		{0, 0}: true, {1, 0}: true, {1, 1}: true, {0, 1}: true,
	}
	gotUVs := map[Vec2]bool{}
	for _, uv := range mesh.UVs {
		gotUVs[uv] = true
	}
	if !reflect.DeepEqual(gotUVs, wantUVs) {
		t.Errorf("uvs = %v, want %v", gotUVs, wantUVs)
	}
	assertWatertightAndWound(t, mesh, "flat square")
	assertVerticesOnSurface(t, grid, mesh, "flat square")
}

func TestTessellateBezierSurfaceRaisedCenter(t *testing.T) {
	grid := raisedCenterGrid(0.5)

	loose := TessellateBezierSurface(grid, 0.1)
	tight := TessellateBezierSurface(grid, 0.01)

	if len(tight.Indices) <= len(loose.Indices) {
		t.Errorf("tighter tolerance did not refine: loose=%d indices, tight=%d",
			len(loose.Indices), len(tight.Indices))
	}
	if len(loose.Indices) <= 6 {
		t.Errorf("raised center was not tessellated beyond the flat baseline: %d indices",
			len(loose.Indices))
	}

	for _, tc := range []struct {
		name string
		mesh BezierSurfaceMesh
	}{
		{"loose", loose}, {"tight", tight},
	} {
		assertWatertightAndWound(t, tc.mesh, tc.name)
		assertVerticesOnSurface(t, grid, tc.mesh, tc.name)

		// Basic output hygiene: finite positions, unit normals, in-range UVs.
		for _, p := range tc.mesh.Vertices {
			if math.IsNaN(float64(p[0])) || math.IsNaN(float64(p[2])) {
				t.Errorf("%s: non-finite vertex %v", tc.name, p)
			}
		}
		for i, n := range tc.mesh.Normals {
			if l := n.Len(); math.Abs(float64(l-1.0)) > 1e-5 {
				t.Errorf("%s: normal %d has length %g, want 1", tc.name, i, l)
			}
		}
		for i, uv := range tc.mesh.UVs {
			if uv[0] < 0 || uv[0] > 1 || uv[1] < 0 || uv[1] > 1 {
				t.Errorf("%s: uv %d out of [0,1]^2: %v", tc.name, i, uv)
			}
		}
	}

	// The surface center rises: BezierSurface(0.5,0.5) is the weighted sum
	// of the nine control points, only the center of which is lifted.
	center := BezierSurface(0.5, 0.5, grid)
	if center[2] <= 0.0 {
		t.Errorf("raised center surface center has z=%g, want positive", center[2])
	}
	// At least one emitted vertex must capture the raised interior.
	foundInterior := false
	for _, p := range tight.Vertices {
		if p[2] > 0.01 {
			foundInterior = true
		}
	}
	if !foundInterior {
		t.Error("no raised interior vertex was emitted")
	}

	// Stable indices: identical inputs produce byte-identical meshes.
	again := TessellateBezierSurface(grid, 0.01)
	if !reflect.DeepEqual(again, tight) {
		t.Error("tessellation is not deterministic across calls")
	}
}

func TestTessellateBezierSurfaceDepthBound(t *testing.T) {
	// A tall spike with an impossibly tight tolerance forces refinement all
	// the way to the depth cap near the parameter center.
	grid := raisedCenterGrid(50.0)
	mesh := TessellateBezierSurface(grid, 1e-12)

	// Reconstruct the deepest grid level present among the vertices: for
	// each uv find the shallowest grid it lies on, then take the maximum.
	maxDepth := 0
	for _, uv := range mesh.UVs {
		vertexDepth := -1
		for d := 0; d <= MaxBezierSurfaceTessellationDepth; d++ {
			denom := float64(uint32(1) << uint(d))
			i := float64(uv[0]) * denom
			j := float64(uv[1]) * denom
			if math.Abs(i-math.Round(i)) < 1e-3 &&
				math.Abs(j-math.Round(j)) < 1e-3 {
				vertexDepth = d
				break
			}
		}
		if vertexDepth < 0 {
			t.Errorf("uv %v lies on no grid up to depth %d",
				uv, MaxBezierSurfaceTessellationDepth)
			continue
		}
		if vertexDepth > maxDepth {
			maxDepth = vertexDepth
		}
	}
	if maxDepth > MaxBezierSurfaceTessellationDepth {
		t.Errorf("refinement reached depth %d, cap is %d",
			maxDepth, MaxBezierSurfaceTessellationDepth)
	}
	if maxDepth < MaxBezierSurfaceTessellationDepth {
		t.Errorf("expected the spike to hit the depth cap %d, got %d",
			MaxBezierSurfaceTessellationDepth, maxDepth)
	}

	// Bounded output size: depth-10 quadtree leaves alone cannot exceed
	// 4^10 cells, and each leaf emits at most 8 triangles.
	if len(mesh.Indices)/3 > 8*(1<<(2*MaxBezierSurfaceTessellationDepth)) {
		t.Errorf("mesh has %d triangles, beyond the advertised bound",
			len(mesh.Indices)/3)
	}
	assertWatertightAndWound(t, mesh, "spike")
	assertVerticesOnSurface(t, grid, mesh, "spike")
}

func TestTessellateBezierSurfaceErrors(t *testing.T) {
	cases := []struct {
		name string
		grid [][]Vec3
		err  float32
	}{
		{"nil grid", nil, 1e-3},
		{"empty grid", [][]Vec3{}, 1e-3},
		{"single row", [][]Vec3{{{0, 0, 0}, {1, 0, 0}}}, 1e-3},
		{"single column", [][]Vec3{{{0, 0, 0}}, {{0, 1, 0}}}, 1e-3},
		{"jagged grid",
			[][]Vec3{{{0, 0, 0}, {1, 0, 0}}, {{0, 1, 0}}}, 1e-3},
		{"NaN control point",
			[][]Vec3{{{0, 0, 0}, {float32(math.NaN()), 0, 0}},
				{{0, 1, 0}, {1, 1, 0}}}, 1e-3},
		{"Inf control point",
			[][]Vec3{{{0, 0, 0}, {float32(math.Inf(1)), 0, 0}},
				{{0, 1, 0}, {1, 1, 0}}}, 1e-3},
		{"zero tolerance", flatUnitSquareGrid(), 0},
		{"negative tolerance", flatUnitSquareGrid(), -1e-3},
		{"NaN tolerance", flatUnitSquareGrid(), float32(math.NaN())},
		{"infinite tolerance", flatUnitSquareGrid(), float32(math.Inf(1))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("%s: expected a panic, got none", tc.name)
				}
			}()
			TessellateBezierSurface(tc.grid, tc.err)
		})
	}
}

func TestTessellateBezierSurfaceDegenerateNormal(t *testing.T) {
	// Every control point coincides: the surface collapses to one point and
	// every partial derivative is zero, so no normal is defined.
	point := Vec3{1, 2, 3}
	grid := [][]Vec3{
		{point, point},
		{point, point},
	}
	mesh := TessellateBezierSurface(grid, 1e-3)

	if len(mesh.Indices) != 6 {
		t.Fatalf("collapsed surface produced %d indices, expected 6",
			len(mesh.Indices))
	}
	for i, p := range mesh.Vertices {
		if p.Sub(point).Len() != 0 {
			t.Errorf("vertex %d = %v, want the collapsed point %v", i, p, point)
		}
	}
	for i, n := range mesh.Normals {
		if n.LenSqr() != 0 {
			t.Errorf("normal %d = %v on a collapsed patch, want zero Vec3", i, n)
		}
	}
}

func TestTessellateBezierSurfaceSeamsAndStability(t *testing.T) {
	grid := raisedCenterGrid(1.0)
	mesh := TessellateBezierSurface(grid, 0.02)

	// One vertex per unique (u, v): duplicate UVs would mean two pool
	// entries for the same parameter point, i.e. a potential crack.
	seen := make(map[Vec2]int, len(mesh.UVs))
	for i, uv := range mesh.UVs {
		if prev, ok := seen[uv]; ok {
			t.Errorf("uv %v emitted twice (vertices %d and %d)", uv, prev, i)
		}
		seen[uv] = i
	}

	// Every edge used by only one triangle must lie on the parameter-domain
	// boundary; an interior boundary edge would be a hole or a crack.
	onDomainBoundary := func(uv Vec2) bool {
		const eps = 1e-6
		return uv[0] <= eps || uv[0] >= 1.0-eps ||
			uv[1] <= eps || uv[1] >= 1.0-eps
	}
	for edge, uses := range meshEdges(mesh) {
		if len(uses) != 1 {
			continue
		}
		for _, idx := range edge {
			if !onDomainBoundary(mesh.UVs[idx]) {
				t.Errorf("boundary edge %v ends at interior uv %v", edge, mesh.UVs[idx])
			}
		}
	}

	// Refining the same surface must reuse every coarser vertex: the coarse
	// mesh's parameter points must all appear in the refined mesh.
	fine := TessellateBezierSurface(grid, 0.005)
	if len(fine.Indices) <= len(mesh.Indices) {
		t.Fatalf("refinement did not grow the mesh: coarse=%d fine=%d indices",
			len(mesh.Indices), len(fine.Indices))
	}
	fineUVs := make(map[Vec2]bool, len(fine.UVs))
	for _, uv := range fine.UVs {
		fineUVs[uv] = true
	}
	for _, uv := range mesh.UVs {
		if !fineUVs[uv] {
			t.Errorf("refined mesh dropped coarse parameter point %v", uv)
		}
	}
}

func TestTessellateBezierSurfaceCubicGrid(t *testing.T) {
	// A non-symmetric 3x4 cubic control grid exercises arbitrary degrees and
	// both u and v refinement independently.
	grid := [][]Vec3{
		{{0, 0, 0}, {0.3, 0, 0.2}, {0.7, 0, -0.1}, {1, 0, 0}},
		{{0, 0.4, 0.1}, {0.3, 0.4, 0.8}, {0.7, 0.4, 0.2}, {1, 0.4, -0.3}},
		{{0, 1, 0}, {0.3, 1, -0.2}, {0.7, 1, 0.5}, {1, 1, 0}},
	}
	mesh := TessellateBezierSurface(grid, 0.01)
	if len(mesh.Indices) <= 6 {
		t.Fatalf("curved cubic grid stayed at the flat two-triangle baseline")
	}
	assertWatertightAndWound(t, mesh, "cubic")
	assertVerticesOnSurface(t, grid, mesh, "cubic")

	// A very loose tolerance must still produce a valid mesh, while a tight
	// one produces strictly more triangles.
	loose := TessellateBezierSurface(grid, 10.0)
	assertWatertightAndWound(t, loose, "loose cubic")
	if len(loose.Indices) > len(mesh.Indices) {
		t.Errorf("loose tolerance produced more triangles than the tight one")
	}
}
