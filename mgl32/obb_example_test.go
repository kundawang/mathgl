// Copyright 2024 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl32_test

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
)

func ExampleOBBIntersect() {
	// Side-2 cubes (half extents 1) on the X axis.
	cube := func(x float32) mgl32.OBB {
		return mgl32.NewAxisAlignedOBB(mgl32.Vec3{x, 0, 0}, mgl32.Vec3{1, 1, 1})
	}

	// Centers 2 apart: faces meet exactly, depth 0 counts as a hit.
	hit, axis, depth := mgl32.OBBIntersect(cube(0), cube(2))
	fmt.Printf("distance 2.00: intersects=%v axisX=%.0f depth=%.2f\n", hit, axis[0], depth)

	// Centers 2.01 apart: a 0.01 gap remains.
	hit, axis, depth = mgl32.OBBIntersect(cube(0), cube(2.01))
	fmt.Printf("distance 2.01: intersects=%v axisX=%.0f gap=%.2f\n", hit, axis[0], depth)

	// Output:
	// distance 2.00: intersects=true axisX=-1 depth=0.00
	// distance 2.01: intersects=false axisX=-1 gap=0.01
}

func ExampleOBBIntersection_rotatedLongBox() {
	s := float32(0.70710677) // sqrt(0.5) in float32
	long := mgl32.NewOBB(
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{3, 0.5, 0.5},
		[3]mgl32.Vec3{{s, s, 0}, {-s, s, 0}, {0, 0, 1}},
	)
	trigger := mgl32.NewAxisAlignedOBB(mgl32.Vec3{2.7, 2.7, 0}, mgl32.Vec3{0.5, 0.5, 0.5})

	result := mgl32.OBBIntersection(long, trigger)
	fmt.Printf("OBB trigger: %v, gap=%.3f\n", result.Intersects, result.Separation)

	// Output:
	// OBB trigger: false, gap=0.111
}
