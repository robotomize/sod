package avltree

type FilterFn func(current Item) bool

type Option func(*Tree)

func WalkOrderAsc() Option {
	return func(o *Tree) {
		o.order = orderAsc
	}
}

func WalkOrderDesc() Option {
	return func(o *Tree) {
		o.order = orderDesc
	}
}

type order uint8

const (
	orderAsc order = iota
	orderDesc
)

func New(opts ...Option) *Tree {
	t := &Tree{order: orderAsc}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

type Tree struct {
	root  *node
	order order
	len   int
}

func (t *Tree) Build(items ...Item) {
	for i := range items {
		t.Add(items[i])
	}
}

func (t *Tree) Len() int {
	return t.len
}

func (t *Tree) Invert() {
	if t.root != nil {
		t.root.invert()
	}
}

func (t *Tree) Points() []Item {
	if t.root == nil {
		return []Item{}
	}
	return t.root.points(t.order)
}

func (t *Tree) Filter(fn FilterFn) []Item {
	if t.root == nil {
		return []Item{}
	}
	return t.root.filter(t.root, fn)
}

func (t *Tree) Add(item Item) {
	if t.root == nil {
		t.root = &node{item: item}
	} else {
		t.root = t.root.add(item)
	}
	t.len += 1
}

func (t *Tree) Remove(item Item) {
	if t.root != nil {
		t.root = t.root.remove(item)
		t.len -= 1
	}
}

func (t *Tree) Contains(item Item) bool {
	node := t.root
	for node != nil {
		if item.Subtraction(node.item) == 0 {
			return true
		}
		if item.Subtraction(node.item) < 0 {
			node = node.left
		} else {
			node = node.right
		}
	}
	return false
}
