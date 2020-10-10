package gbkd

import (
	"context"
	"rango/internal/predictor"
	"rango/internal/predictor/knn/avlnode"
	"rango/pkg/container/avltree"
	"rango/pkg/container/kdtree"
	"sync"
	"sync/atomic"
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
	rebuildOutdatedTime = 60 * time.Second
	rebuildSizeTime     = 5 * time.Second
	balanceKDTreeTime   = 1 * time.Minute
	greenBlueBuildTime  = 10 * time.Second
)

type gbTree struct {
	green *kdtree.Tree
	blue  *kdtree.Tree
	state uint32
}

func (t *gbTree) tree() *kdtree.Tree {
	if atomic.LoadUint32(&t.state) == 0 {
		return t.green
	}
	return t.blue
}

func (t *gbTree) build(items ...kdtree.Item) {
	if atomic.LoadUint32(&t.state) == 0 {
		t.blue.Build(items...)
		atomic.StoreUint32(&t.state, 1)
	} else {
		t.green.Build(items...)
		atomic.StoreUint32(&t.state, 0)
	}
}

func NewBKDAlg(distanceFn func(vec, vec1 []float64) (float64, error), opts ...Option) *kd {
	b := &kd{
		distanceFn:          distanceFn,
		timesTree:           avltree.New(),
		rebuildOutdatedTime: time.Now(),
		gbTree:              &gbTree{state: 0, green: kdtree.New(distanceFn), blue: kdtree.New(distanceFn)},
	}
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
	distanceFn          func(vec, vec1 []float64) (float64, error)
	rebuildOutdatedTime time.Time
	timesTree           *avltree.Tree
	gbTree              *gbTree
	removeOpCnt         int64
	removeOpTime        int64
	appendOpTime        int64
	appendOpCnt         int64
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

	items := make([]kdtree.Item, len(data))

	for i := range data {
		items[i] = data[i].Vector()
		b.timesTree.Add(avlnode.TimeNode{
			K: data[i].Time(),
			V: data[i],
		})
	}

	b.gbTree.build(items...)
}

func (b *kd) Len() int {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return b.gbTree.tree().Len()
}

func (b *kd) Append(data ...predictor.DataPoint) {
	for i := range data {
		b.append(data[i])
	}
}

func (b *kd) KNN(vec predictor.Vector, n int) ([]predictor.Vector, error) {
	var kdVectors []predictor.Vector
	b.mtx.RLock()
	items, err := b.gbTree.tree().KNN(vec, n)
	if err != nil {
		return nil, err
	}
	b.mtx.RUnlock()
	for i := range items {
		kdVectors = append(kdVectors, items[i].(predictor.Vector))
	}
	return kdVectors, nil
}

func (b *kd) Reset() {
	b.mtx.Lock()
	b.gbTree.blue = kdtree.New(b.distanceFn)
	b.gbTree.green = kdtree.New(b.distanceFn)
	b.timesTree = avltree.New()
	b.mtx.Unlock()
}

func (b *kd) append(data predictor.DataPoint) {
	b.mtx.Lock()
	b.gbTree.tree().Insert(data.Vector())
	b.timesTree.Add(avlnode.TimeNode{
		K: data.Time(),
		V: data,
	})
	b.mtx.Unlock()
	atomic.AddInt64(&b.appendOpCnt, 1)
}

func (b *kd) needBalanceKD() bool {
	b.mtx.RLock()
	gbLen := b.gbTree.tree().Len()
	b.mtx.RUnlock()
	timeDiff := time.Now().Unix() - atomic.LoadInt64(&b.appendOpTime)
	valueDiff := float64(atomic.LoadInt64(&b.appendOpCnt)) / float64(gbLen)
	return gbLen > 0 &&
		(valueDiff > 0.001 || (atomic.LoadInt64(&b.appendOpCnt) > 0 && timeDiff > int64(greenBlueBuildTime.Seconds())))
}

func (b *kd) balanceKDTree() {
	if b.needBalanceKD() {
		b.gbTree.build(b.gbTree.tree().Points()...)
		atomic.StoreInt64(&b.appendOpCnt, 0)
		atomic.StoreInt64(&b.appendOpTime, time.Now().Unix())
	}
}

func (b *kd) needGBBuild() bool {
	b.mtx.RLock()
	gbLen := b.gbTree.tree().Len()
	b.mtx.RUnlock()
	timeDiff := time.Now().Unix() - atomic.LoadInt64(&b.removeOpTime)
	valueDiff := float64(atomic.LoadInt64(&b.removeOpCnt)) / float64(gbLen)
	return gbLen > 0 &&
		(valueDiff > 0.01 || (atomic.LoadInt64(&b.removeOpCnt) > 0 && timeDiff > int64(greenBlueBuildTime.Seconds())))
}

func (b *kd) buildGBTree() {
	if b.needGBBuild() {
		items := make([]kdtree.Item, b.timesTree.Len())
		b.mtx.RLock()
		for i, point := range b.timesTree.Points() {
			items[i] = point.(avlnode.TimeNode).V.Vector()
		}
		b.mtx.RUnlock()
		b.gbTree.build(items...)
		atomic.StoreInt64(&b.removeOpCnt, 0)
		atomic.StoreInt64(&b.removeOpTime, time.Now().Unix())
	}
}

func (b *kd) rebuildOutdated() {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	list := b.timesTree.Filter(func(current avltree.Item) bool {
		return time.Since(current.(avlnode.TimeNode).K) > b.opts.maxStorageTime
	})
	for i := range list {
		b.timesTree.Remove(list[i])
		b.removeOpCnt += 1
	}
	b.rebuildOutdatedTime = time.Now()
}

func (b *kd) rebuildSize() {
	b.mtx.RLock()
	sub := b.timesTree.Len() - b.opts.maxItemsStored
	list := b.timesTree.Points()
	b.mtx.RUnlock()
	for _, timeNode := range list[:sub] {
		b.timesTree.Remove(timeNode)
		b.removeOpCnt += 1
	}
}

func (b *kd) schedule(ctx context.Context) {
	outdatedTicker := time.NewTicker(rebuildOutdatedTime)
	sizeTicker := time.NewTicker(rebuildSizeTime)
	kdBalanceTicker := time.NewTicker(balanceKDTreeTime)
	gbBuildTicker := time.NewTicker(5 * time.Second)
	defer outdatedTicker.Stop()
	defer sizeTicker.Stop()
	defer kdBalanceTicker.Stop()
	defer gbBuildTicker.Stop()
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
		case <-kdBalanceTicker.C:
			b.balanceKDTree()
		case <-gbBuildTicker.C:
			b.buildGBTree()
		case <-ctx.Done():
			return
		}
	}
}
