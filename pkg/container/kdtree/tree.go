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

import (
	"fmt"
	"math"
	"sort"

	"github.com/go-sod/sod/pkg/container/pqueue"
)

type Point interface {
	Dim(idx int) float64
	Dimensions() int
	Points() []float64
}

func New(distFn func(vec, vec1 []float64) (float64, error)) *Tree {
	return &Tree{
		root:   nil,
		len:    0,
		distFn: distFn,
	}
}

type Tree struct {
	root   *node
	len    int
	distFn func(vec, vec1 []float64) (float64, error)
}

func (t *Tree) RangeSearch(r []Range) []Point {
	return t.root.RangeSearch(r, 0)
}

func (t *Tree) Build(points ...Point) {
	t.len = len(points)
	t.root = buildTreeRecursive(points, 0)
}

func (t *Tree) Len() int {
	return t.len
}

func (t *Tree) Insert(p Point) {
	if t.root == nil {
		t.root = &node{Key: p}
	} else {
		t.root.Insert(p, 0)
	}
	t.len += 1
}

func (t *Tree) Balance() {
	t.root = buildTreeRecursive(t.Points(), 0)
}

func (t *Tree) Points() []Point {
	if t.root == nil {
		return []Point{}
	}
	return t.root.Points()
}

func (t *Tree) KNN(p Point, k int) ([]Point, error) {
	if t.root == nil || k == 0 {
		return []Point{}, fmt.Errorf("root is nil or K is 0")
	}

	queue := pqueue.New(pqueue.WithCap(uint(k)))

	if err := t.knn(p, k, t.root, 0, queue); err != nil {
		return []Point{}, err
	}

	points := make([]Point, queue.Len())
	for i := 0; i < k && 0 < queue.Len(); i++ {
		o := queue.Head().(*node).Key
		points[i] = o
	}

	return points, nil
}

func (t *Tree) knn(p Point, k int, first *node, dim int, queue *pqueue.Queue) error {
	if k == 0 || first == nil {
		return nil
	}

	var path []*node
	currentNode := first

	for currentNode != nil {
		path = append(path, currentNode)
		if p.Dim(dim) < currentNode.Key.Dim(dim) {
			currentNode = currentNode.Left
		} else {
			currentNode = currentNode.Right
		}
		dim = (dim + 1) % p.Dimensions()
	}

	dim = (dim - 1 + p.Dimensions()) % p.Dimensions()
	for path, currentNode = popLast(path); currentNode != nil; path, currentNode = popLast(path) {
		currentDistance, err := t.distFn(p.Points(), currentNode.Key.Points())
		if err != nil {
			return fmt.Errorf("compute knn error: %w", err)
		}
		checkedDistance := getKthOrLastDistance(queue, k-1)
		if currentDistance < checkedDistance {
			queue.Push(currentNode, currentDistance)
			checkedDistance = getKthOrLastDistance(queue, k-1)
		}

		if distanceForDimension(p, currentNode.Key, dim) < checkedDistance {
			var next *node
			if p.Dim(dim) < currentNode.Key.Dim(dim) {
				next = currentNode.Right
			} else {
				next = currentNode.Left
			}
			if err := t.knn(p, k, next, (dim+1)%p.Dimensions(), queue); err != nil {
				return err
			}
		}
		dim = (dim - 1 + p.Dimensions()) % p.Dimensions()
	}
	return nil
}

type sortPoints struct {
	dim    int
	points []Point
}

func (b *sortPoints) Len() int {
	return len(b.points)
}

func (b *sortPoints) Less(i, j int) bool {
	return b.points[i].Dim(b.dim) < b.points[j].Dim(b.dim)
}

func (b *sortPoints) Swap(i, j int) {
	b.points[i], b.points[j] = b.points[j], b.points[i]
}

func buildTreeRecursive(points []Point, dim int) *node {
	if len(points) == 0 {
		return nil
	}
	if len(points) == 1 {
		return &node{Key: points[0]}
	}

	sort.Sort(&sortPoints{dim: dim, points: points})
	mid := len(points) / 2
	root := points[mid]
	nextDim := (dim + 1) % root.Dimensions()
	return &node{
		Key:   root,
		Left:  buildTreeRecursive(points[:mid], nextDim),
		Right: buildTreeRecursive(points[mid+1:], nextDim),
	}
}

func distanceForDimension(vec, vec1 Point, dim int) float64 {
	return math.Abs(vec1.Dim(dim) - vec.Dim(dim))
}

func popLast(arr []*node) ([]*node, *node) {
	l := len(arr) - 1
	if l < 0 {
		return arr, nil
	}
	return arr[:l], arr[l]
}

func getKthOrLastDistance(queue *pqueue.Queue, i int) float64 {
	if queue.Len() <= i {
		return math.MaxFloat64
	}
	_, distance := queue.Seek(i)
	return distance
}
