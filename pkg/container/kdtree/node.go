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

func (n *node) Insert(p Item, dim int) {
	if p.Point(dim) < n.Key.Point(dim) {
		if n.Left == nil {
			n.Left = &node{Key: p}
		} else {
			n.Left.Insert(p, (dim+1)%n.Key.Dimensions())
		}
	} else {
		if n.Right == nil {
			n.Right = &node{Key: p}
		} else {
			n.Right.Insert(p, (dim+1)%n.Key.Dimensions())
		}
	}
}

func (n *node) Remove(p Item, dim int) (*node, *node) {
	for i := 0; i < n.Key.Dimensions(); i++ {
		if n.Key.Point(i) != p.Point(i) {
			if n.Left != nil {
				returnedNode, substitutedNode := n.Left.Remove(p, (dim+1)%n.Key.Dimensions())
				if returnedNode != nil {
					if returnedNode == n.Left {
						n.Left = substitutedNode
					}
					return returnedNode, nil
				}
			}
			if n.Right != nil {
				returnedNode, substitutedNode := n.Right.Remove(p, (dim+1)%n.Key.Dimensions())
				if returnedNode != nil {
					if returnedNode == n.Right {
						n.Right = substitutedNode
					}
					return returnedNode, nil
				}
			}
			return nil, nil
		}
	}

	if n.Left != nil {
		largest := n.Left.FindLargest(dim, nil)
		removed, sub := n.Left.Remove(largest.Key, (dim+1)%n.Key.Dimensions())

		removed.Left = n.Left
		removed.Right = n.Right
		if n.Left == removed {
			removed.Left = sub
		}
		return n, removed
	}

	if n.Right != nil {
		smallest := n.Right.FindSmallest(dim, nil)
		removed, sub := n.Right.Remove(smallest.Key, (dim+1)%n.Key.Dimensions())

		removed.Left = n.Left
		removed.Right = n.Right
		if n.Right == removed {
			removed.Right = sub
		}
		return n, removed
	}

	return n, nil
}

func (n *node) FindSmallest(dim int, smallest *node) *node {
	if smallest == nil || n.Key.Point(dim) < smallest.Key.Point(dim) {
		smallest = n
	}
	if n.Left != nil {
		smallest = n.Left.FindSmallest(dim, smallest)
	}
	if n.Right != nil {
		smallest = n.Right.FindSmallest(dim, smallest)
	}
	return smallest
}

func (n *node) FindLargest(dim int, largest *node) *node {
	if largest == nil || n.Key.Point(dim) > largest.Key.Point(dim) {
		largest = n
	}
	if n.Left != nil {
		largest = n.Left.FindLargest(dim, largest)
	}
	if n.Right != nil {
		largest = n.Right.FindLargest(dim, largest)
	}
	return largest
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
