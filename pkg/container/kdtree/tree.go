package kdtree

import (
	"fmt"
	"math"
	"rango/pkg/container/pqueue"
	"sort"
)

type Item interface {
	Point(idx int) float64
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

func (t *Tree) RangeSearch(r []Range) []Item {
	return t.root.RangeSearch(r, 0)
}

func (t *Tree) Build(points ...Item) {
	t.len = len(points)
	t.root = buildTreeRecursive(points, 0)
}

func (t *Tree) Len() int {
	return t.len
}

func (t *Tree) Insert(p Item) {
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

func (t *Tree) Points() []Item {
	if t.root == nil {
		return []Item{}
	}
	return t.root.Points()
}

func (t *Tree) KNN(p Item, k int) ([]Item, error) {
	if t.root == nil || k == 0 {
		return []Item{}, fmt.Errorf("root is nil or K is 0")
	}

	queue := pqueue.New(pqueue.WithCap(uint(k)))

	if err := t.knn(p, k, t.root, 0, queue); err != nil {
		return []Item{}, err
	}

	points := make([]Item, queue.Len())
	for i := 0; i < k && 0 < queue.Len(); i++ {
		o := queue.Head().(*node).Key
		points[i] = o
	}

	return points, nil
}

func (t *Tree) knn(p Item, k int, first *node, dim int, queue *pqueue.Queue) error {
	if k == 0 || first == nil {
		return nil
	}

	var path []*node
	currentNode := first

	for currentNode != nil {
		path = append(path, currentNode)
		if p.Point(dim) < currentNode.Key.Point(dim) {
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
			if p.Point(dim) < currentNode.Key.Point(dim) {
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
	points []Item
}

func (b *sortPoints) Len() int {
	return len(b.points)
}

func (b *sortPoints) Less(i, j int) bool {
	return b.points[i].Point(b.dim) < b.points[j].Point(b.dim)
}

func (b *sortPoints) Swap(i, j int) {
	b.points[i], b.points[j] = b.points[j], b.points[i]
}

func buildTreeRecursive(points []Item, dim int) *node {
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

func distanceForDimension(vec, vec1 Item, dim int) float64 {
	return math.Abs(vec1.Point(dim) - vec.Point(dim))
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
