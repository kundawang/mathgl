// Copyright 2024 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mgl64_test

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl64"
)

func ExampleOBBIntersect() {
	// Side-2 cubes (half extents 1) on the X axis.
	cube := func(x float64) mgl64.OBB {
		return mgl64.NewAxisAlignedOBB(mgl64.Vec3{x, 0, 0}, mgl64.Vec3{1, 1, 1})
	}

	// Centers 2 apart: faces meet exactly, depth 0 counts as a hit.
	hit, axis, depth := mgl64.OBBIntersect(cube(0), cube(2))
	fmt.Printf("distance 2.00: intersects=%v axisX=%.0f depth=%.2f\n", hit, axis[0], depth)

	// Centers 2.01 apart: a 0.01 gap remains.
	hit, axis, depth = mgl64.OBBIntersect(cube(0), cube(2.01))
	fmt.Printf("distance 2.01: intersects=%v axisX=%.0f gap=%.2f\n", hit, axis[0], depth)

	// Output:
	// distance 2.00: intersects=true axisX=-1 depth=0.00
	// distance 2.01: intersects=false axisX=-1 gap=0.01
}

func ExampleOBBIntersection_rotatedLongBox() {
	s := 0.7071067811865476 // sqrt(0.5)
	long := mgl64.NewOBB(
		mgl64.Vec3{0, 0, 0},
		mgl64.Vec3{3, 0.5, 0.5},
		[3]mgl64.Vec3{{s, s, 0}, {-s, s, 0}, {0, 0, 1}},
	)
	trigger := mgl64.NewAxisAlignedOBB(mgl64.Vec3{2.7, 2.7, 0}, mgl64.Vec3{0.5, 0.5, 0.5})

	// Their axis-aligned bounds overlap (a naive AABB trigger fires)...
	min0, max0 := long.AABB()
	min1, max1 := trigger.AABB()
	aabbHit := min0[0] <= max1[0] && min1[0] <= max0[0] &&
		min0[1] <= max1[1] && min1[1] <= max0[1]
	fmt.Println("AABB trigger:", aabbHit)

	// ...but the oriented boxes are separated along the long axis.
	result := mgl64.OBBIntersection(long, trigger)
	fmt.Printf("OBB trigger: %v, gap=%.3f\n", result.Intersects, result.Separation)

	// Output:
	// AABB trigger: true
	// OBB trigger: false, gap=0.111
}
