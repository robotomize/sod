package pqueue

import (
	"sort"
)

func WithOrderAsc() Option {
	return func(q *Queue) {
		q.order = orderAsc
	}
}

func WithOrderDesc() Option {
	return func(q *Queue) {
		q.order = orderDesc
	}
}

func WithCap(size uint) Option {
	return func(q *Queue) {
		q.cap = int(size)
	}
}

type Option func(*Queue)

type order uint8

const (
	orderAsc order = iota
	orderDesc
)

type item struct {
	value interface{}
	prior float64
}

func New(opts ...Option) *Queue {
	p := &Queue{items: &[]*item{}, order: orderAsc, cap: -1}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type Queue struct {
	order order
	cap   int
	items *[]*item
}

func (q *Queue) PopAll() []interface{} {
	pulled := make([]interface{}, len(*q.items))
	for i := range *q.items {
		pulled[i] = (*q.items)[i].value
	}
	*q.items = (*q.items)[:0]
	return pulled
}

func (q *Queue) Head() interface{} {
	if len(*q.items) == 0 {
		return nil
	}
	x := (*q.items)[0]
	*q.items = (*q.items)[1:]
	return x.value
}

func (q *Queue) Tail() interface{} {
	l := len(*q.items) - 1
	if l < 0 {
		return nil
	}
	x := (*q.items)[l]
	*q.items = (*q.items)[:l]
	return x.value
}

func (q *Queue) Push(val interface{}, priority float64) {
	*q.items = append(*q.items, &item{value: val, prior: priority})
	sort.Sort(q)
	if q.cap < 0 {
		return
	}
	if q.cap < len(*q.items) {
		*q.items = (*q.items)[:q.cap]
	}
}

func (q *Queue) Cap() int { return q.cap }

func (q *Queue) Len() int { return len(*q.items) }

func (q *Queue) Swap(i, j int) { (*q.items)[i], (*q.items)[j] = (*q.items)[j], (*q.items)[i] }

func (q *Queue) Less(i, j int) bool {
	if q.order == orderAsc {
		return (*q.items)[i].prior < (*q.items)[j].prior
	}
	return (*q.items)[i].prior > (*q.items)[j].prior
}

func (q *Queue) Seek(idx int) (interface{}, float64) {
	item := (*q.items)[idx]
	return item.value, item.prior
}
