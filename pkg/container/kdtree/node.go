package kdtree

type node struct {
	Key   Item
	Left  *node
	Right *node
}

func (n *node) Points() []Item {
	var points []Item
	if n.Left != nil {
		points = n.Left.Points()
	}
	points = append(points, n.Key)
	if n.Right != nil {
		points = append(points, n.Right.Points()...)
	}
	return points
}

func (n *node) insertLeft(p Item, dim int) {
	if n.Left == nil {
		n.Left = &node{Key: p}
	} else {
		n.Left.Insert(p, (dim+1)%n.Key.Dimensions())
	}
}

func (n *node) insertRight(p Item, dim int) {
	if n.Right == nil {
		n.Right = &node{Key: p}
	} else {
		n.Right.Insert(p, (dim+1)%n.Key.Dimensions())
	}
}

func (n *node) Insert(p Item, dim int) {
	if p.Point(dim) < n.Key.Point(dim) {
		n.insertLeft(p, dim)
	} else {
		n.insertRight(p, dim)
	}
}

type Range struct {
	Min, Max float64
}

func (n *node) RangeSearch(r []Range, axis int) []Item {
	var points []Item

	for dim, limit := range r {
		if limit.Min > n.Key.Point(dim) || limit.Max < n.Key.Point(dim) {
			goto checkChildren
		}
	}
	points = append(points, n.Key)

checkChildren:
	if n.Left != nil && n.Key.Point(axis) >= r[axis].Min {
		points = append(points, n.Left.RangeSearch(r, (axis+1)%n.Key.Dimensions())...)
	}
	if n.Right != nil && n.Key.Point(axis) <= r[axis].Max {
		points = append(points, n.Right.RangeSearch(r, (axis+1)%n.Key.Dimensions())...)
	}

	return points
}
