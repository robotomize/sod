package brute

import (
	"context"
	"fmt"
	"rango/internal/predictor"
	"rango/internal/predictor/knn/avlnode"
	"rango/pkg/container/avltree"
	"rango/pkg/container/pqueue"
	"sync"
	"time"
)

func WithMaxItems(n int) Option {
	return func(l *brute) {
		l.opts.maxItemsStored = n
	}
}

func WithStorageTime(t time.Duration) Option {
	return func(l *brute) {
		l.opts.maxStorageTime = t
	}
}

type Option func(*brute)

type Options struct {
	maxItemsStored int
	maxStorageTime time.Duration
}

const (
	rebuildOutdatedTime = 60 * time.Second
	rebuildSizeTime     = 5 * time.Second
)

func NewBruteAlg(distFn func(vec, vec1 []float64) (float64, error), opts ...Option) *brute {
	b := &brute{distFunc: distFn, data: avltree.New()}
	for _, opt := range opts {
		opt(b)
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.schedule(ctx)
	return b
}

type brute struct {
	opts      Options
	mtx       sync.RWMutex
	data      *avltree.Tree
	createdAt time.Time
	distFunc  func(vec, vec1 []float64) (float64, error)
	cancel    func()
}

func (b *brute) Reset() {
	b.mtx.Lock()
	b.data = avltree.New()
	b.mtx.Unlock()
}

func (b *brute) KNN(vec predictor.Vector, k int) ([]predictor.Vector, error) {
	return b.knn(vec, k)
}

func (b *brute) Len() int {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return b.data.Len()
}

func (b *brute) Build(data ...predictor.DataPoint) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if b.data == nil {
		b.data = avltree.New()
	}
	for i := range data {
		b.data.Add(avlnode.TimeNode{
			K: data[i].Time(),
			V: data[i],
		})
	}
}

func (b *brute) Append(data ...predictor.DataPoint) {
	b.append(data...)
}

func (b *brute) knn(vec predictor.Vector, n int) ([]predictor.Vector, error) {
	b.mtx.RLock()
	list := b.data.Points()
	b.mtx.RUnlock()
	pq := pqueue.New(pqueue.WithCap(uint(n)))
	for _, item := range list {
		distance, err := b.distFunc(vec.Points(), item.Value().(predictor.DataPoint).Vector().Points())
		if err != nil {
			return nil, fmt.Errorf(
				"unable to compute distance between %v and %v: %w",
				vec.Points(), item.Value().(predictor.Vector).Points(),
				err,
			)
		}
		pq.Push(item.Value().(predictor.DataPoint).Vector(), distance)
	}
	knn := make([]predictor.Vector, pq.Len())
	for i, pData := range pq.PopAll() {
		knn[i] = pData.(predictor.Vector)
	}

	if len(knn) < n {
		return nil, fmt.Errorf("knn less minimal value")
	}
	return knn, nil
}

func (b *brute) append(data ...predictor.DataPoint) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for _, dat := range data {
		b.data.Add(avlnode.TimeNode{
			K: dat.Time(),
			V: dat,
		})
	}
}

func (b *brute) schedule(ctx context.Context) {
	outdatedTicker := time.NewTicker(rebuildOutdatedTime)
	sizeTicker := time.NewTicker(rebuildSizeTime)
	defer outdatedTicker.Stop()
	defer sizeTicker.Stop()
	for {
		select {
		case <-outdatedTicker.C:
			if b.opts.maxStorageTime > 0 {
				b.rebuildOutdated()
			}
		case <-sizeTicker.C:
			if b.opts.maxItemsStored > 0 {
				b.rebuildSize()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *brute) rebuildOutdated() {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if time.Since(b.createdAt) > b.opts.maxStorageTime {
		list := b.data.Filter(func(current avltree.Item) bool {
			return time.Since(current.(avlnode.TimeNode).K) < b.opts.maxStorageTime
		})

		for i := range list {
			b.data.Remove(list[i])
		}
		b.createdAt = time.Now()
	}
}

func (b *brute) rebuildSize() {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if b.data.Len() > b.opts.maxItemsStored {
		list := b.data.Points()
		sub := b.data.Len() - b.opts.maxItemsStored

		for i := range list[:sub] {
			b.data.Remove(list[i].(avlnode.TimeNode))
		}
	}
}
