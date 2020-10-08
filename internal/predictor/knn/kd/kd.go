package kd

import (
	"context"
	"fmt"
	"rango/internal/predictor"
	"rango/internal/predictor/knn/avlnode"
	"rango/pkg/container/avltree"
	"rango/pkg/container/kdtree"
	"sync"
	"time"
)

func WithMaxItems(n int) Option {
	return func(l *kd) {
		l.opts.maxItemsStored = n
	}
}

func WithStorageTime(t time.Duration) Option {
	return func(l *kd) {
		l.opts.maxStorageTime = t
	}
}

type Option func(*kd)

type Options struct {
	maxItemsStored int
	maxStorageTime time.Duration
}

const (
	bucketSize = 10000
)

const (
	rebuildOutdatedTime = 60 * time.Second
	rebuildSizeTime     = 5 * time.Second
	balanceKDTreeTime   = 1 * time.Minute
)

func NewKDAlg(distFn predictor.PointsDistanceFn, opts ...Option) *kd {
	b := &kd{
		distFn:              distFn,
		timesTree:           avltree.New(),
		rebuildOutdatedTime: time.Now(),
		dataTree:            kdtree.New(func(vec, vec1 []float64) (float64, error) { return distFn(vec, vec1) })}
	for _, opt := range opts {
		opt(b)
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.schedule(ctx)
	return b
}

type kd struct {
	mtx                 sync.RWMutex
	opts                Options
	dataTree            *kdtree.Tree
	timesTree           *avltree.Tree
	distFn              predictor.PointsDistanceFn
	rebuildOutdatedTime time.Time
	kdOpCnt             int
	cancel              func()
}

func (b *kd) Close() {
	b.cancel()
}

func (b *kd) Build(data ...predictor.DataPoint) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if b.timesTree == nil {
		b.timesTree = avltree.New()
	}
	if b.dataTree == nil {
		b.dataTree = kdtree.New(func(vec, vec1 []float64) (float64, error) {
			return b.distFn(vec, vec1)
		})
	}
	items := make([]kdtree.Item, len(data))
	for i := range data {
		items[i] = data[i].Vector()
		b.timesTree.Add(avlnode.TimeNode{
			K: data[i].Time(),
			V: data[i],
		})
	}
	b.dataTree.Build(items...)
}

func (b *kd) Reset() {
	b.mtx.Lock()
	b.dataTree = nil
	b.mtx.Unlock()
}

func (b *kd) KNN(vec predictor.Vector, n int) ([]predictor.Vector, error) {
	b.mtx.RLock()
	kdVectors, err := b.dataTree.KNN(vec, n)
	if err != nil {
		return nil, err
	}
	b.mtx.RUnlock()
	output := make([]predictor.Vector, len(kdVectors))
	for i, vector := range kdVectors {
		output[i] = vector
	}
	return output, nil
}

func (b *kd) Len() int {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return b.dataTree.Len()
}

func (b *kd) Remove(data ...predictor.DataPoint) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range data {
		b.dataTree.Remove(data[i].Vector())
		b.timesTree.Remove(avlnode.TimeNode{
			K: data[i].Time(),
			V: data[i],
		})
		b.kdOpCnt += 1
	}
}

func (b *kd) Append(data ...predictor.DataPoint) {
	b.append(data...)
}

func (b *kd) append(data ...predictor.DataPoint) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range data {
		dat := data[i]
		b.dataTree.Insert(dat.Vector())
		b.timesTree.Add(avlnode.TimeNode{
			K: dat.Time(),
			V: dat,
		})
		b.kdOpCnt += 1
	}
}

func (b *kd) schedule(ctx context.Context) {
	outdatedTicker := time.NewTicker(rebuildOutdatedTime)
	sizeTicker := time.NewTicker(rebuildSizeTime)
	balanceTreeTicker := time.NewTicker(balanceKDTreeTime)
	defer outdatedTicker.Stop()
	defer sizeTicker.Stop()
	defer balanceTreeTicker.Stop()
	for {
		select {
		case <-outdatedTicker.C:
			if b.opts.maxStorageTime > 0 && time.Since(b.rebuildOutdatedTime) > b.opts.maxStorageTime {
				b.rebuildOutdated()
			}
		case <-sizeTicker.C:
			if b.opts.maxItemsStored > 0 && b.timesTree.Len() > b.opts.maxItemsStored {
				b.rebuildSize()
			}
		case <-balanceTreeTicker.C:
			b.rebuildKDTree()
		case <-ctx.Done():
			return
		}
	}
}

func (b *kd) rebuildKDTree() {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.dataTree.Balance()
	b.kdOpCnt = 0
}

func (b *kd) rebuildOutdated() {
	b.mtx.RLock()
	list := b.timesTree.Filter(func(current avltree.Item) bool {
		return time.Since(current.(avlnode.TimeNode).K) < b.opts.maxStorageTime
	})
	b.mtx.RUnlock()
	var delay time.Duration
	for i := range list {
		delay = b.removeNode(delay, list[i].(avlnode.TimeNode))
	}
	b.mtx.Lock()
	b.rebuildOutdatedTime = time.Now()
	b.mtx.Unlock()
}

func (b *kd) rebuildSize() {
	b.mtx.RLock()
	sub := b.timesTree.Len() - b.opts.maxItemsStored
	list := b.timesTree.Walk()
	b.mtx.RUnlock()
	var delay time.Duration
	t := time.Now()
	for i := range list[:sub] {
		delay = b.removeNode(delay, list[i].(avlnode.TimeNode))
	}
	fmt.Println(time.Since(t))
}

func (b *kd) removeNode(currDelay time.Duration, node avlnode.TimeNode) time.Duration {
	time.Sleep(currDelay)
	b.mtx.Lock()
	b.timesTree.Remove(node)
	t := time.Now()
	b.dataTree.Remove(node.V.Vector())
	nextDelay := time.Since(t)
	b.mtx.Unlock()
	return nextDelay
}
