package avltree

import (
	"math"
)

const needBalanceHeight = 2

type Item interface {
	Subtraction(current Item) int
	Key() interface{}
	Value() interface{}
}

type node struct {
	item   Item
	left   *node
	right  *node
	height int
}

func (n *node) invert() *node {
	var tmp *node
	if n.left != nil {
		tmp = n.right
		n.right = n.left.invert()
	}
	if n.right != nil {
		if tmp == nil {
			n.left = n.right.invert()
		} else {
			n.left = tmp.invert()
		}

	}
	return n
}

func (n *node) insertLeft(item Item) *node {
	root := n
	n.left = n.addToSubTree(n.left, item)
	if n.heightDiff() == needBalanceHeight {
		if item.Subtraction(n.item) <= 0 {
			root = n.rotateRight()
		} else {
			root = n.rotateLeftThenRight()
		}
	}
	return root
}

func (n *node) insertRight(item Item) *node {
	root := n
	n.right = n.addToSubTree(n.right, item)
	if n.heightDiff() == -needBalanceHeight {
		if item.Subtraction(n.item) > 0 {
			root = n.rotateLeft()
		} else {
			n.rotateRightThenLeft()
		}
	}
	return root
}

func (n *node) add(item Item) *node {
	var root *node
	if item.Subtraction(n.item) <= 0 {
		root = n.insertLeft(item)
	} else {
		root = n.insertRight(item)
	}
	root.computeHeight()
	return root
}

func (n *node) removeFromParent(parent *node, item Item) *node {
	if parent != nil {
		return parent.remove(item)
	}
	return nil
}

func (n *node) remove(item Item) *node {
	root := n
	switch {
	case item.Subtraction(n.item) == 0:
		if n.left == nil {
			return n.right
		}

		child := n.left
		for child.right != nil {
			child = child.right
		}
		childKey := child.item
		n.left = n.removeFromParent(n.left, childKey)
		n.item = childKey
	case item.Subtraction(n.item) < 0:
		n.left = n.removeFromParent(n.left, item)
		if n.heightDiff() == needBalanceHeight {
			if item.Subtraction(n.item) <= 0 {
				root = n.rotateRight()
			} else {
				root = n.rotateLeftThenRight()
			}
		}
	default:
		n.right = n.removeFromParent(n.right, item)
		if n.heightDiff() == -needBalanceHeight {
			if item.Subtraction(n.item) > 0 {
				root = n.rotateLeft()
			} else {
				n.rotateRightThenLeft()
			}
		}
	}
	root.computeHeight()
	return root
}

func (n *node) rotateRight() *node {
	root := n.left
	grandson := root.right
	n.left = grandson
	root.right = n
	n.computeHeight()
	return root
}

func (n *node) rotateLeft() *node {
	root := n.right
	grandson := root.left
	n.right = grandson
	root.left = n
	n.computeHeight()
	return root
}

func (n *node) rotateRightThenLeft() *node {
	child := n.right
	root := child.left
	if root != nil {
		grandFirst := root.left
		grandSecond := root.right
		child.left = grandSecond
		child.right = grandFirst
		root.left = n
		root.right = child
	}
	child.computeHeight()
	n.computeHeight()
	return root
}

func (n *node) rotateLeftThenRight() *node {
	child := n.left
	root := child.right
	grandFirst := root.left
	grandSecond := root.right
	child.right = grandFirst
	n.left = grandSecond
	root.left = child
	root.right = n
	child.computeHeight()
	n.computeHeight()
	return root
}

func (n *node) addToSubTree(parent *node, item Item) *node {
	if parent == nil {
		return &node{item: item}
	}

	return parent.add(item)
}

func (n *node) computeHeight() {
	height := -1
	if n.left != nil {
		height = int(math.Max(float64(height), float64(n.left.height)))
	}
	if n.right != nil {
		height = int(math.Max(float64(height), float64(n.right.height)))
	}
	n.height = height + 1
}

func (n *node) heightDiff() int {
	leftTarget, rightTarget := 0, 0
	if n.left != nil {
		leftTarget = 1 + n.left.height
	}
	if n.right != nil {
		rightTarget = 1 + n.right.height
	}
	return leftTarget - rightTarget
}
