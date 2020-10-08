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

func (t *Tree) Walk() []Item {
	var list []Item
	t.walk(&list, t.root)
	return list
}

func (t *Tree) walkOrderAsc(list *[]Item, node *node) {
	if node.left != nil {
		t.walkOrderAsc(list, node.left)
	}

	*list = append(*list, node.item)

	if node.right != nil {
		t.walkOrderAsc(list, node.right)
	}
}

func (t *Tree) walkOrderDesc(list *[]Item, node *node) {
	if node.right != nil {
		t.walkOrderDesc(list, node.right)
	}

	*list = append(*list, node.item)

	if node.left != nil {
		t.walkOrderDesc(list, node.left)
	}
}

func (t *Tree) walk(list *[]Item, node *node) {
	if node != nil {
		if t.order == orderAsc {
			t.walkOrderAsc(list, node)
		} else {
			t.walkOrderDesc(list, node)
		}
	}
}

func (t *Tree) filterAsc(list *[]Item, current *node, fn FilterFn) {
	if current.left != nil {
		t.filterAsc(list, current.left, fn)
	}

	if fn == nil {
		*list = append(*list, current.item)
		return
	}

	if fn(current.item) {
		*list = append(*list, current.item)
	}

	if current.right != nil {
		t.filterAsc(list, current.right, fn)
	}
}

func (t *Tree) filterDesc(list *[]Item, current *node, fn FilterFn) {
	if current.right != nil {
		t.filterDesc(list, current.right, fn)
	}

	if fn == nil {
		*list = append(*list, current.item)
		return
	}

	if fn(current.item) {
		*list = append(*list, current.item)
	}

	if current.left != nil {
		t.filterDesc(list, current.left, fn)
	}
}

func (t *Tree) filter(list *[]Item, node *node, fn FilterFn) {
	if node != nil {
		if t.order == orderAsc {
			t.filterAsc(list, node, fn)
		} else {
			t.filterDesc(list, node, fn)
		}
	}
}

func (t *Tree) Filter(fn FilterFn) []Item {
	var list []Item
	t.filter(&list, t.root, fn)
	return list
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
