// This file is generated from mgl32/shapes.go; DO NOT EDIT

// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64

import (
	"fmt"
	"math"
	"sort"
)

// Circle generates a circle centered at (0,0) with a given radius.
// The radii are assumed to be in GL's coordinate sizing.
//
// Technically this draws an ellipse with two axes that match with the X and Y axes, the reason it has a radiusX and radiusY is because GL's coordinate system
// is proportional to screen width and screen height. So if you have a non-square viewport, a single radius will appear
// to "squash" the circle in one direction (usually the Y direction), so the X and Y radius allow for a circle to be made.
// A good way to get the correct radii is with mathgl.ScreenToGLCoords(radius, radius, screenWidth, screenHeight) which will get you the correct
// proportional GL coords.
//
// The numSlices argument specifies how many triangles you want your circle divided into, setting this
// number to too low a value may cause problem (and too high will cause it to take a lot of memory and time to compute
// without much gain in resolution).
//
// This uses discrete triangles, not a triangle fan
func Circle(radiusX, radiusY float64, numSlices int) []Vec2 {
	twoPi := float64(2.0 * math.Pi)

	circlePoints := make([]Vec2, 0, numSlices*3)
	center := Vec2{0.0, 0.0}
	previous := Vec2{radiusX, 0.0}

	for theta := twoPi / float64(numSlices); !FloatEqual(theta, twoPi); theta = Clamp(theta+twoPi/float64(numSlices), 0.0, twoPi) {
		sin, cos := math.Sincos(float64(theta))
		curr := Vec2{float64(cos) * radiusX, float64(sin) * radiusY}

		circlePoints = append(circlePoints, center, previous, curr)
		previous = curr
	}

	// Now add the final point at theta=2pi
	circlePoints = append(circlePoints, center, previous, Vec2{radiusX, 0.0})
	return circlePoints
}

// Rect generates a 2-triangle rectangle for use with GL_TRIANGLES. The width and height should use GL's proportions (that is, where a width of 1.0
// is equivalent to half of the width of the render target); however, the y-coordinates grow downwards, not upwards. That is, it
// assumes you want the origin of the rectangle with the top-left corner at (0.0,0.0).
//
// Keep in mind that GL's coordinate system is proportional, so width=height will not result in a square unless your viewport is square.
// If you want to maintain proportionality regardless of screen size, use the results of w,h := ScreenToGLCoordsf(absoluteWidth, absoluteHeight, screenWidth, screenHeight);
// w,h=w+1,h-1 in the call to this function. (The w+1,h-1 step maps the coordinates to start at 0.0 rather than -1.0)
func Rect(width, height float64) []Vec2 {
	return []Vec2{
		{0.0, 0.0},
		{0.0, -height},
		{width, -height},

		{0.0, 0.0},
		{width, -height},
		{width, 0.0},
	}
}

// QuadraticBezierCurve2D interpolates the point t on the given bezier curve.
func QuadraticBezierCurve2D(t float64, cPoint1, cPoint2, cPoint3 Vec2) Vec2 {
	if t < 0.0 || t > 1.0 {
		panic("Can't interpolate on bezier curve with t out of range [0.0,1.0]")
	}

	return cPoint1.Mul((1.0 - t) * (1.0 - t)).Add(cPoint2.Mul(2 * (1 - t) * t)).Add(cPoint3.Mul(t * t))
}

// QuadraticBezierCurve3D interpolates the point t on the given bezier curve.
func QuadraticBezierCurve3D(t float64, cPoint1, cPoint2, cPoint3 Vec3) Vec3 {
	if t < 0.0 || t > 1.0 {
		panic("Can't interpolate on bezier curve with t out of range [0.0,1.0]")
	}

	return cPoint1.Mul((1.0 - t) * (1.0 - t)).Add(cPoint2.Mul(2 * (1 - t) * t)).Add(cPoint3.Mul(t * t))
}

// CubicBezierCurve2D interpolates the point t on the given bezier curve.
func CubicBezierCurve2D(t float64, cPoint1, cPoint2, cPoint3, cPoint4 Vec2) Vec2 {
	if t < 0.0 || t > 1.0 {
		panic("Can't interpolate on bezier curve with t out of range [0.0,1.0]")
	}

	sum14 := cPoint1.Mul((1 - t) * (1 - t) * (1 - t)).Add(cPoint4.Mul(t * t * t))
	sum23 := cPoint2.Mul(3 * (1 - t) * (1 - t) * t).Add(cPoint3.Mul(3 * (1 - t) * t * t))
	return sum14.Add(sum23)
}

// CubicBezierCurve3D interpolates the point t on the given bezier curve.
func CubicBezierCurve3D(t float64, cPoint1, cPoint2, cPoint3, cPoint4 Vec3) Vec3 {
	if t < 0.0 || t > 1.0 {
		panic("Can't interpolate on bezier curve with t out of range [0.0,1.0]")
	}

	sum14 := cPoint1.Mul((1 - t) * (1 - t) * (1 - t)).Add(cPoint4.Mul(t * t * t))
	sum23 := cPoint2.Mul(3 * (1 - t) * (1 - t) * t).Add(cPoint3.Mul(3 * (1 - t) * t * t))
	return sum14.Add(sum23)
}

// BezierCurve2D returns the point at point t along an n-control point Bezier curve
//
// t must be in the range 0.0 and 1.0 or this function will panic. Consider [0.0,1.0] to be similar to a percentage,
// 0.0 is first control point, and the point at 1.0 is the last control point. Any point in between is how far along the path you are between 0 and 1.
//
// This function is not sensitive to the coordinate system of the control points. It will correctly interpolate regardless of whether they're in screen coords,
// gl coords, or something else entirely
func BezierCurve2D(t float64, cPoints []Vec2) Vec2 {
	if t < 0.0 || t > 1.0 {
		panic("Input to bezier has t not in range [0,1]. If you think this is a precision error, use mathgl.Clamp[f|d] before calling this function")
	}

	n := len(cPoints) - 1
	point := cPoints[0].Mul(float64(math.Pow(float64(1.0-t), float64(n))))

	for i := 1; i <= n; i++ {
		point = point.Add(cPoints[i].Mul(float64(float64(choose(n, i)) * math.Pow(float64(1-t), float64(n-i)) * math.Pow(float64(t), float64(i))))) // P += P_i * nCi * (1-t)^(n-i) * t^i
	}

	return point
}

// BezierCurve3D same as the 2D version, except the line is in 3D space
func BezierCurve3D(t float64, cPoints []Vec3) Vec3 {
	if t < 0.0 || t > 1.0 {
		panic("Input to bezier has t not in range [0,1]. If you think this is a precision error, use mathgl.Clamp[f|d] before calling this function")
	}

	n := len(cPoints) - 1
	point := cPoints[0].Mul(float64(math.Pow(float64(1.0-t), float64(n))))

	for i := 1; i <= n; i++ {
		point = point.Add(cPoints[i].Mul(float64(float64(choose(n, i)) * math.Pow(float64(1-t), float64(n-i)) * math.Pow(float64(t), float64(i))))) // P += P_i * nCi * (1-t)^(n-i) * t^i
	}

	return point
}

// MakeBezierCurve2D generates a bezier curve with controlPoints cPoints. The
// numPoints argument determines how many "samples" it makes along the line. For
// instance, a call to this with numPoints 2 will have exactly two points: the
// start and end points For any points above 2 it will divide it into
// numPoints-1 chunks (which means it will generate numPoints-2 vertices other
// than the beginning and end). So for 3 points it will divide it in half, 4
// points into thirds, and so on.
//
// This is likely to get rather expensive for anything over perhaps a cubic
// curve.
func MakeBezierCurve2D(numPoints int, cPoints []Vec2) (line []Vec2) {
	line = make([]Vec2, numPoints)
	if numPoints == 0 {
		return
	} else if numPoints == 1 {
		line[0] = cPoints[0]
		return
	} else if numPoints == 2 {
		line[0] = cPoints[0]
		line[1] = cPoints[len(cPoints)-1]
		return
	}

	line[0] = cPoints[0]
	for i := 1; i < numPoints-1; i++ {
		line[i] = BezierCurve2D(Clamp(float64(i)/float64(numPoints-1), 0.0, 1.0), cPoints)
	}
	line[numPoints-1] = cPoints[len(cPoints)-1]

	return
}

// MakeBezierCurve3D same as the 2D version, except with the line in 3D space.
func MakeBezierCurve3D(numPoints int, cPoints []Vec3) (line []Vec3) {
	line = make([]Vec3, numPoints)
	if numPoints == 0 {
		return
	} else if numPoints == 1 {
		line[0] = cPoints[0]
		return
	} else if numPoints == 2 {
		line[0] = cPoints[0]
		line[1] = cPoints[len(cPoints)-1]
		return
	}

	line[0] = cPoints[0]
	for i := 1; i < numPoints-1; i++ {
		line[i] = BezierCurve3D(Clamp(float64(i)/float64(numPoints-1), 0.0, 1.0), cPoints)
	}
	line[numPoints-1] = cPoints[len(cPoints)-1]

	return
}

// BezierSurface creates a 2-dimensional Bezier surface of arbitrary degree in
// 3D Space Like the curve functions, if u or v are not in the range [0.0,1.0]
// the function will panic, use Clamp[f|d] to ensure it is correct.
//
// The control point matrix must not be jagged, or this will end up panicking
// from an index out of bounds exception
func BezierSurface(u, v float64, cPoints [][]Vec3) Vec3 {
	if u < 0.0 || u > 1.0 || v < 0.0 || v > 1.0 {
		panic("u or v not in range [0.0,1.0] in BezierSurface")
	}

	n := len(cPoints) - 1
	m := len(cPoints[0]) - 1

	point := cPoints[0][0].Mul(float64(math.Pow(float64(1.0-u), float64(n)) * math.Pow(float64(1.0-v), float64(m))))

	for i := 0; i <= n; i++ {
		for j := 0; j <= m; j++ {
			if i == 0 && j == 0 {
				continue
			}

			point = point.Add(cPoints[i][j].Mul(float64(float64(choose(n, i)) * math.Pow(float64(u), float64(i)) * math.Pow(float64(1.0-u), float64(n-i)) * float64(choose(m, j)) * math.Pow(float64(v), float64(j)) * math.Pow(float64(1.0-v), float64(m-j)))))
		}
	}

	return point
}

// MaxBezierSurfaceTessellationDepth is the hard limit on the number of times a
// parameter-domain cell is split in half along either axis when tessellating a
// Bezier surface. It bounds the work and memory used even when the requested
// maxError is smaller than the surface can represent, so the smallest possible
// cell spans 1/(1<<MaxBezierSurfaceTessellationDepth) of the parameter domain.
const MaxBezierSurfaceTessellationDepth = 10

// BezierSurfaceMesh is an error-controlled triangle tessellation of a
// BezierSurface. It is a vertex pool with parallel slices: Vertices[i] carries
// UVs[i] and Normals[i].
//
// Indices holds independent triangles (three indices per triangle). Vertex
// indices are stable: identical parameter coordinates always resolve to one
// shared vertex, so two tessellations of the same surface (or of surfaces
// stitched together with matching boundary control points) share their
// boundary vertices exactly and cannot crack. Winding is consistent over the
// whole mesh: with (u, v) parameterized as usual, triangles face the same
// direction as the stored normals (+Z for the flat unit XY square).
//
// A normal that cannot be defined (a zero-length cross product of the partial
// derivatives, e.g. on a collapsed or folded patch) is the zero Vec3 rather
// than a NaN; callers can detect this with Normals[i].LenSqr() == 0.
type BezierSurfaceMesh struct {
	Vertices []Vec3
	UVs      []Vec2
	Normals  []Vec3
	Indices  []uint32
}

// bezierSurfaceTessellator carries the shared state of one tessellation run.
type bezierSurfaceTessellator struct {
	cPoints [][]Vec3
	maxErr  float64

	// leaves maps a quadtree cell key to the cells that survived
	// subdivision (flat cells and cells at the depth limit).
	leaves map[uint64]quadCell

	// vertices maps a parameter-domain grid point to its vertex pool index,
	// guaranteeing one vertex per unique (u, v).
	vertices map[uint64]int

	// points caches surface evaluations keyed by their parameter pair;
	// without it the same point would be re-evaluated (with math.Pow at
	// every control point) each time two cells or stencils touch it.
	points map[paramPoint]Vec3
	mesh   BezierSurfaceMesh
}

// paramPoint is a (u, v) parameter pair used as a cache key.
type paramPoint struct{ u, v float64 }

// quadCell identifies one square cell of the adaptive quadtree by integer
// grid coordinates. A cell at depth d covers the parameter square
// [i/2^d, (i+1)/2^d] x [j/2^d, (j+1)/2^d].
type quadCell struct {
	i, j, depth int
}

// cellKey packs a quadtree cell into a unique uint64. Depth is at most
// MaxBezierSurfaceTessellationDepth (10), leaving ample room per axis.
func cellKey(i, j, depth int) uint64 {
	return uint64(uint32(i))<<32 | uint64(uint32(j))<<8 | uint64(depth)
}

// TessellateBezierSurface builds an adaptive triangle mesh of the Bezier
// surface defined by cPoints so that no mesh vertex deviates from the true
// surface by more than maxError (measured in the units of the control points,
// as the 3D distance between the sampled midpoint and the bilinear interpolant
// of the cell corners).
//
// Flat regions stay coarse (a perfectly flat surface yields exactly two
// triangles) while strongly curved regions are recursively split in half
// along u and v until the error bound is met or
// MaxBezierSurfaceTessellationDepth is reached. The quadtree is then balanced
// so neighboring cells differ by at most one depth, and shared edge vertices
// make the mesh crack-free with uniform winding.
//
// The returned BezierSurfaceMesh contains positions, (u, v) parameter
// coordinates, unit-length surface normals and triangle indices.
//
// It panics when:
//   - cPoints is empty, has fewer than two rows or columns, or is jagged
//     (rows of different lengths);
//   - any control point component is NaN or infinite;
//   - maxError is zero, negative, NaN or infinite.
//
// A surface that collapses (e.g. every control point identical, or a folded
// patch with zero partial derivatives somewhere) tessellates without panicking;
// only the undefined normals are emitted as the zero Vec3.
func TessellateBezierSurface(cPoints [][]Vec3, maxError float64) BezierSurfaceMesh {
	if len(cPoints) < 2 || len(cPoints[0]) < 2 {
		panic("TessellateBezierSurface: control grid must have at least 2x2 control points")
	}
	cols := len(cPoints[0])
	for row := range cPoints {
		if len(cPoints[row]) != cols {
			panic("TessellateBezierSurface: control grid must be rectangular, not jagged")
		}
		for _, p := range cPoints[row] {
			if !isFiniteVec3(p) {
				panic("TessellateBezierSurface: control points must all be finite")
			}
		}
	}
	if !(maxError > 0.0) || math.IsInf(float64(maxError), 1) {
		panic("TessellateBezierSurface: maxError must be a finite, positive number")
	}

	t := &bezierSurfaceTessellator{
		cPoints:  cPoints,
		maxErr:   maxError,
		leaves:   make(map[uint64]quadCell),
		vertices: make(map[uint64]int),
		points:   make(map[paramPoint]Vec3),
	}

	// Phase 1: adaptive refinement driven by the geometric error.
	t.refine(quadCell{0, 0, 0})
	// Phase 2: enforce the 2:1 balance constraint, so no leaf edge borders
	// cells more than one level finer.
	t.balance()
	// Phase 3: emit shared vertices and consistently wound triangles.
	t.emit()

	return t.mesh
}

// isFiniteVec3 reports whether every component is neither NaN nor infinite.
func isFiniteVec3(v Vec3) bool {
	for _, x := range v {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return false
		}
	}
	return true
}

// surfaceAt evaluates the Bezier surface, clamping parameters into [0,1]
// (BezierSurface panics out of range), checking the result stays finite and
// memoizing it so shared cell corners and normal stencils are evaluated once.
func (t *bezierSurfaceTessellator) surfaceAt(u, v float64) Vec3 {
	u = Clamp(u, 0.0, 1.0)
	v = Clamp(v, 0.0, 1.0)
	key := paramPoint{u, v}
	if p, ok := t.points[key]; ok {
		return p
	}
	p := BezierSurface(u, v, t.cPoints)
	if !isFiniteVec3(p) {
		panic("TessellateBezierSurface: surface evaluated to a non-finite point")
	}
	t.points[key] = p
	return p
}

// refine recursively splits cells whose edge- and center-midpoint samples
// deviate from the bilinear patch spanned by the cell corners by more than
// the allowed error. Cells meeting the error bound, and cells at the depth
// limit, become leaves.
func (t *bezierSurfaceTessellator) refine(c quadCell) {
	scale := float64(1.0) / float64(uint32(1)<<uint(c.depth))
	u0 := float64(c.i) * scale
	v0 := float64(c.j) * scale
	u1 := float64(c.i+1) * scale
	v1 := float64(c.j+1) * scale

	p00 := t.surfaceAt(u0, v0)
	p10 := t.surfaceAt(u1, v0)
	p11 := t.surfaceAt(u1, v1)
	p01 := t.surfaceAt(u0, v1)

	um := (u0 + u1) * 0.5
	vm := (v0 + v1) * 0.5

	// Bilinear interpolant of the four corners at (su, sv) in [0,1]^2.
	linear := func(su, sv float64) Vec3 {
		bottom := p00.Add(p10.Sub(p00).Mul(su))
		top := p01.Add(p11.Sub(p01).Mul(su))
		return bottom.Add(top.Sub(bottom).Mul(sv))
	}

	// Five interior samples: four edge midpoints and the cell center.
	samples := [5]struct {
		u, v float64
	}{
		{um, v0}, {u1, vm}, {um, v1}, {u0, vm}, {um, vm},
	}
	flat := true
	for _, s := range samples {
		su := (s.u - u0) / (u1 - u0)
		sv := (s.v - v0) / (v1 - v0)
		if t.surfaceAt(s.u, s.v).Sub(linear(su, sv)).Len() > t.maxErr {
			flat = false
			break
		}
	}

	if flat || c.depth >= MaxBezierSurfaceTessellationDepth {
		t.leaves[cellKey(c.i, c.j, c.depth)] = c
		return
	}

	next := c.depth + 1
	t.refine(quadCell{2 * c.i, 2 * c.j, next})
	t.refine(quadCell{2*c.i + 1, 2 * c.j, next})
	t.refine(quadCell{2 * c.i, 2*c.j + 1, next})
	t.refine(quadCell{2*c.i + 1, 2*c.j + 1, next})
}

// hasFinerNeighbor reports whether any cell adjacent to c across one of its
// four edges is a leaf more than one level below c. It walks the quadtree
// down from each half-size neighbor slot: reaching a leaf at any depth below
// c's means a finer neighbor (reaching c's depth+1 with a leaf is exactly the
// 2:1 violation).
func (t *bezierSurfaceTessellator) hasFinerNeighbor(c quadCell) bool {
	// Half-size (depth+1) cell coordinates touching each edge of c. The two
	// slots alongside an edge share one fixed axis coordinate.
	pi, pj := 2*c.i, 2*c.j
	type slot struct{ i, j int }
	slots := [8]slot{
		{pi - 2, pj}, {pi - 2, pj + 1}, // left
		{pi + 2, pj}, {pi + 2, pj + 1}, // right
		{pi, pj - 2}, {pi + 1, pj - 2}, // bottom
		{pi, pj + 2}, {pi + 1, pj + 2}, // top
	}
	for _, s := range slots {
		if s.i < 0 || s.j < 0 {
			continue // parameter-domain boundary: no neighbor there
		}
		i, j := s.i, s.j
		for depth := c.depth + 1; depth <= MaxBezierSurfaceTessellationDepth; depth++ {
			if _, ok := t.leaves[cellKey(i, j, depth)]; ok {
				return true // leaf strictly finer than c
			}
			// Descend into the sub-cell that stays under this slot.
			i *= 2
			j *= 2
		}
	}
	return false
}

// balance repeatedly splits leaves violating the 2:1 rule until no leaf has a
// neighbor two or more levels finer. Splitting one leaf only creates cells at
// the next depth, and the depth cap bounds the process, so the fixed point is
// reached after finitely many splits.
func (t *bezierSurfaceTessellator) balance() {
	for changed := true; changed; {
		changed = false
		for key, c := range t.leaves {
			if c.depth >= MaxBezierSurfaceTessellationDepth || !t.hasFinerNeighbor(c) {
				continue
			}
			delete(t.leaves, key)
			next := c.depth + 1
			children := [4]quadCell{
				{2 * c.i, 2 * c.j, next},
				{2*c.i + 1, 2 * c.j, next},
				{2 * c.i, 2*c.j + 1, next},
				{2*c.i + 1, 2*c.j + 1, next},
			}
			for _, child := range children {
				t.leaves[cellKey(child.i, child.j, child.depth)] = child
			}
			changed = true
		}
	}
}

// emit walks the balanced leaves in deterministic order (depth, then row i,
// then column j), allocating one vertex per unique parameter point and
// triangulating each leaf with a uniform winding.
func (t *bezierSurfaceTessellator) emit() {
	leaves := make([]quadCell, 0, len(t.leaves))
	for _, c := range t.leaves {
		leaves = append(leaves, c)
	}
	sort.Slice(leaves, func(a, b int) bool {
		x, y := leaves[a], leaves[b]
		return x.depth < y.depth || (x.depth == y.depth &&
			(x.i < y.i || (x.i == y.i && x.j < y.j)))
	})
	for _, c := range leaves {
		t.emitCell(c)
	}
}

// vertex returns the pool index of the vertex at parameter-grid point
// (i, j, depth), evaluating and caching it on first use.
func (t *bezierSurfaceTessellator) vertex(i, j, depth int) int {
	key := cellKey(i, j, depth)
	if idx, ok := t.vertices[key]; ok {
		return idx
	}
	scale := float64(1.0) / float64(uint32(1)<<uint(depth))
	u := float64(i) * scale
	v := float64(j) * scale
	idx := len(t.mesh.Vertices)
	t.mesh.Vertices = append(t.mesh.Vertices, t.surfaceAt(u, v))
	t.mesh.UVs = append(t.mesh.UVs, Vec2{u, v})
	t.mesh.Normals = append(t.mesh.Normals, t.normalAt(u, v))
	t.vertices[key] = idx
	return idx
}

// emitCell triangulates one balanced leaf. Every triangle is wound so its
// geometric face normal agrees with the dv x du surface normal used by
// normalAt (+Z on the flat unit square). Edge midpoint vertices are shared
// with the finer neighbors across the edge, guaranteeing watertight seams;
// the optional cell-center vertex is unique to this leaf.
func (t *bezierSurfaceTessellator) emitCell(c quadCell) {
	i, j, d := c.i, c.j, c.depth

	// Four corners, CCW in the (u, v) plane starting at (u0, v0).
	v00 := t.vertex(i, j, d)
	v10 := t.vertex(i+1, j, d)
	v11 := t.vertex(i+1, j+1, d)
	v01 := t.vertex(i, j+1, d)

	// A midpoint exists exactly where a half-size neighbor leaf touches the
	// edge. In a balanced quadtree there are at most four such edges.
	next := d + 1
	var midpoints [4]int // [bottom, right, top, left], -1 when absent
	for k := range midpoints {
		midpoints[k] = -1
	}
	if d < MaxBezierSurfaceTessellationDepth {
		pi, pj := 2*i, 2*j
		if _, ok := t.leaves[cellKey(pi, pj-2, next)]; ok {
			midpoints[0] = t.vertex(pi+1, pj, next) // bottom edge
		}
		if _, ok := t.leaves[cellKey(pi+2, pj, next)]; ok {
			midpoints[1] = t.vertex(pi+2, pj+1, next) // right edge
		}
		if _, ok := t.leaves[cellKey(pi, pj+2, next)]; ok {
			midpoints[2] = t.vertex(pi+1, pj+2, next) // top edge
		}
		if _, ok := t.leaves[cellKey(pi-2, pj, next)]; ok {
			midpoints[3] = t.vertex(pi, pj+1, next) // left edge
		}
	}

	addTri := func(a, b, c int) {
		t.mesh.Indices = append(t.mesh.Indices, uint32(a), uint32(b), uint32(c))
	}

	splitEdges := 0
	for _, m := range midpoints {
		if m >= 0 {
			splitEdges++
		}
	}

	if splitEdges == 0 {
		// No finer neighbors: the flat case stays at two triangles.
		addTri(v00, v11, v10)
		addTri(v00, v01, v11)
		return
	}

	// At least one split edge: insert a cell-center vertex and fan out.
	// Boundary ring in CCW (u, v) order starting at v00: corner, midpoint,
	// corner, midpoint, ... Midpoint slots are skipped when that edge is not
	// split. Triangles use the reversed ring so the geometric face normal
	// agrees with the dv x du surface normal (+Z on the flat unit square).
	center := t.vertex(2*i+1, 2*j+1, next)
	ring := []int{
		v00, midpoints[0], v10, midpoints[1],
		v11, midpoints[2], v01, midpoints[3],
	}
	var boundary []int
	for _, idx := range ring {
		if idx >= 0 {
			boundary = append(boundary, idx)
		}
	}
	for k := 0; k < len(boundary); k++ {
		addTri(center, boundary[(k+1)%len(boundary)], boundary[k])
	}
}

// normalAt estimates the unit surface normal at (u, v) from central finite
// differences of the partial derivatives. At the parameter boundaries a
// one-sided difference replaces the out-of-range sample. A zero cross product
// (a collapsed or folded surface) yields the zero Vec3, which callers can
// detect, instead of propagating NaNs.
func (t *bezierSurfaceTessellator) normalAt(u, v float64) Vec3 {
	// Fixed stencil: wide enough to dominate float32 rounding yet far below
	// the coarsest cell size, so the normal follows the local tangent.
	const h = float64(1.0 / 1024.0)

	du := func() Vec3 {
		if u+h <= 1.0 {
			return t.surfaceAt(u+h, v).Sub(t.surfaceAt(Clamp(u-h, 0.0, 1.0), v))
		}
		// At the u=1 boundary use a forward difference from u-h to u.
		return t.surfaceAt(u, v).Sub(t.surfaceAt(Clamp(u-h, 0.0, 1.0), v))
	}()
	dv := func() Vec3 {
		if v+h <= 1.0 {
			return t.surfaceAt(u, v+h).Sub(t.surfaceAt(u, Clamp(v-h, 0.0, 1.0)))
		}
		// At the v=1 boundary use a forward difference from v-h to v.
		return t.surfaceAt(u, v).Sub(t.surfaceAt(u, Clamp(v-h, 0.0, 1.0)))
	}()

	// Cross dv x du so the canonical flat unit square (corners at the XY
	// origin and (1,1,0)) has the +Z normal, matching the outward face used
	// by the triangle winding below.
	n := dv.Cross(du)
	if n.LenSqr() == 0.0 {
		return Vec3{}
	}
	normal := n.Normalize()
	if !isFiniteVec3(normal) {
		return Vec3{}
	}
	return normal
}

// BezierSplineInterpolate2D does interpolation over a spline of several bezier
// curves. Each bezier curve must have a finite range, though the spline may be
// disjoint. The bezier curves are not required to be in any particular order.
//
// If t is out of the range of all given curves, this function will panic
func BezierSplineInterpolate2D(t float64, ranges [][2]float64, cPoints [][]Vec2) Vec2 {
	if len(ranges) != len(cPoints) {
		panic("Each bezier curve needs a range")
	}

	for i, curveRange := range ranges {
		if t >= curveRange[0] && t <= curveRange[1] {
			return BezierCurve2D((t-curveRange[0])/(curveRange[1]-curveRange[0]), cPoints[i])
		}
	}

	panic("t is out of the range of all bezier curves in this spline")
}

// BezierSplineInterpolate3D does interpolation over a spline of several bezier
// curves. Each bezier curve must have a finite range, though the spline may be
// disjoint. The bezier curves are not required to be in any particular order.
//
// If t is out of the range of all given curves, this function will panic
func BezierSplineInterpolate3D(t float64, ranges [][2]float64, cPoints [][]Vec3) Vec3 {
	if len(ranges) != len(cPoints) {
		panic("Each bezier curve needs a range")
	}

	for i, curveRange := range ranges {
		if t >= curveRange[0] && t <= curveRange[1] {
			return BezierCurve3D((t-curveRange[0])/(curveRange[1]-curveRange[0]), cPoints[i])
		}
	}

	panic("t is out of the range of all bezier curves in this spline")
}

// ReticulateSplines reticulates ALL the Splines.
//
// For the overly serious: the function is just for fun. It does nothing except prints a Maxis reference. Technically you could "reticulate splines"
// by joining a bunch of splines together, but that ruins the joke.
func ReticulateSplines(ranges [][][2]float64, cPoints [][][]Vec2, withLlamas bool) {
	if !withLlamas {
		fmt.Println("You can't reticulate splines without llamas, silly.")
	} else {
		fmt.Println("Actually, you can't even reticulate splines WITH llamas")
	}
}

// ScreenToGLCoords transforms from pixel coordinates to GL coordinates.
//
// This assumes that your pixel coordinate system considers its origin to be in the top left corner (GL's is in the bottom left).
// The coordinates x and y may be out of the range [0,screenWidth-1] and [0,screeneHeight-1].
//
// GL's coordinate system maps [screenWidth-1,0] to [1.0,1.0] and [0,screenHeight-1] to [-1.0,-1.0]. If x and y are out of the range, they'll still
// be mapped correctly, just off the screen. (e.g. if y = 2*(screenHeight-1) you'll get -3.0 for yOut)
//
// This is similar to Unproject, except for 2D cases and much simpler (especially since an inverse may always be found)
func ScreenToGLCoords(x, y int, screenWidth, screenHeight int) (xOut, yOut float64) {
	xOut = 2.0*float64(x)/float64(screenWidth-1) - 1.0
	yOut = -2.0*float64(y)/float64(screenHeight-1) + 1.0

	return
}

// GLToScreenCoords transforms from GL's proportional system to pixel coordinates.
//
// Assumes the pixel coordinate system has its origin in the top left corner. (GL's is in the bottom left)
//
// GL's coordinate system maps [screenWidth-1,0] to [1.0,1.0] and [0,screenHeight-1] to [-1.0,-1.0]. If x and y are out of the range, they'll still
// be mapped correctly, just off the screen. (e.g. if y=-3.0, you'll get 2*(screenHeight-1) for yOut)
//
// This is similar to Project, except for 2D cases and much simpler
func GLToScreenCoords(x, y float64, screenWidth, screenHeight int) (xOut, yOut int) {
	xOut = int((x + 1.0) * float64(screenWidth-1) / 2.0)
	yOut = int((1.0 - y) * float64(screenHeight-1) / 2.0)

	return
}

// choose calculates the binomial coefficient C(n,k) aka nCk
func choose(n, k int) int {
	if k == 0 {
		return 1
	} else if n == 0 {
		return 0
	}
	result := float64(n - (k - 1))
	for i := 2; i <= k; i++ {
		result *= float64(n-(k-i)) / float64(i)
	}

	return int(result)
}
