/*
 * Copyright 2020 Dennis Kuhnert
 * Copyright 2020 Ivanov Nikita
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */
package kdtree

type node struct {
	Key   Point
	Left  *node
	Right *node
}

func (n *node) Points() []Point {
	var points []Point
	if n.Left != nil {
		points = n.Left.Points()
	}
	points = append(points, n.Key)
	if n.Right != nil {
		points = append(points, n.Right.Points()...)
	}
	return points
}

func (n *node) insertLeft(p Point, dim int) {
	if n.Left == nil {
		n.Left = &node{Key: p}
	} else {
		n.Left.Insert(p, (dim+1)%n.Key.Dimensions())
	}
}

func (n *node) insertRight(p Point, dim int) {
	if n.Right == nil {
		n.Right = &node{Key: p}
	} else {
		n.Right.Insert(p, (dim+1)%n.Key.Dimensions())
	}
}

func (n *node) Insert(p Point, dim int) {
	if p.Dim(dim) < n.Key.Dim(dim) {
		n.insertLeft(p, dim)
	} else {
		n.insertRight(p, dim)
	}
}

type Range struct {
	Min, Max float64
}

func (n *node) RangeSearch(r []Range, axis int) []Point {
	var points []Point

	for dim, limit := range r {
		if limit.Min > n.Key.Dim(dim) || limit.Max < n.Key.Dim(dim) {
			goto checkChildren
		}
	}
	points = append(points, n.Key)

checkChildren:
	if n.Left != nil && n.Key.Dim(axis) >= r[axis].Min {
		points = append(points, n.Left.RangeSearch(r, (axis+1)%n.Key.Dimensions())...)
	}
	if n.Right != nil && n.Key.Dim(axis) <= r[axis].Max {
		points = append(points, n.Right.RangeSearch(r, (axis+1)%n.Key.Dimensions())...)
	}

	return points
}
