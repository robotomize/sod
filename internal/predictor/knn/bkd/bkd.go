package bkd

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"rango/internal/predictor"
	"rango/internal/predictor/knn/avlnode"
	"rango/internal/util"
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

const maxBucketSize = 16000

const (
	rebuildOutdatedTime = 60 * time.Second
	rebuildSizeTime     = 5 * time.Second
	balanceKDTreeTime   = 1 * time.Minute
	greenBlueBuildTime  = 10 * time.Second
)

func NewBKDAlg(vecDistFn predictor.PointsDistanceFn, opts ...Option) *kd {
	b := &kd{
		distanceFn:          vecDistFn,
		timesTree:           avltree.New(),
		rebuildOutdatedTime: time.Now(),
		hashBucketId:        map[[32]byte][]uuid.UUID{},
		hashingFn:           util.СomputeVectorHash,
		buckets:             map[uuid.UUID]*kdContainer{},
		greenBlueTree: &greenBlueTree{state: 0, green: kdtree.New(func(vec, vec1 []float64) (float64, error) {
			return vecDistFn(vec, vec1)
		}), blue: kdtree.New(func(vec, vec1 []float64) (float64, error) {
			return vecDistFn(vec, vec1)
		})},
	}
	for _, opt := range opts {
		opt(b)
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.schedule(ctx)
	return b
}

type kdContainer struct {
	mtx  sync.RWMutex
	tree *kdtree.Tree
}

type kd struct {
	mtx                 sync.RWMutex
	opts                Options
	timesTree           *avltree.Tree
	distanceFn          predictor.PointsDistanceFn
	rebuildOutdatedTime time.Time
	buckets             map[uuid.UUID]*kdContainer
	hashBucketId        map[[32]byte][]uuid.UUID
	hashingFn           func([]float64) [32]byte
	greenBlueTree       *greenBlueTree
	gbOpCnt             int64
	gbOpTime            int64
	kdOpTime            int64
	kdOpCnt             int64
	cancel              func()
}

func (b *kd) Close() {
	b.cancel()
}

func (b *kd) balanceKD() {
	b.mtx.RLock()
	gbLen := b.greenBlueTree.tree().Len()
	b.mtx.RUnlock()
	timeDiff := time.Now().Unix() - atomic.LoadInt64(&b.kdOpTime)
	valueDiff := float64(atomic.LoadInt64(&b.kdOpCnt)) / float64(gbLen)
	if gbLen > 0 && (valueDiff > 0.001 || (atomic.LoadInt64(&b.kdOpCnt) > 0 && timeDiff > int64(greenBlueBuildTime.Seconds()))) {
		for _, bucket := range b.buckets {
			bucket.mtx.Lock()
			bucket.tree.Balance()
			bucket.mtx.Unlock()
		}
		atomic.StoreInt64(&b.kdOpCnt, 0)
		atomic.StoreInt64(&b.kdOpTime, time.Now().Unix())
	}
}

func (b *kd) buildGB() {
	b.mtx.RLock()
	gbLen := b.greenBlueTree.tree().Len()
	b.mtx.RUnlock()
	timeDiff := time.Now().Unix() - atomic.LoadInt64(&b.gbOpTime)
	valueDiff := float64(atomic.LoadInt64(&b.gbOpCnt)) / float64(gbLen)
	if gbLen > 0 && (valueDiff > 0.01 || (atomic.LoadInt64(&b.gbOpCnt) > 0 && timeDiff > int64(greenBlueBuildTime.Seconds()))) {
		var items []kdtree.Item
		for _, bucket := range b.buckets {
			bucket.mtx.RLock()
			items = append(items, bucket.tree.Points()...)
			bucket.mtx.RUnlock()
		}
		b.greenBlueTree.build(items...)
		atomic.StoreInt64(&b.gbOpCnt, 0)
		atomic.StoreInt64(&b.gbOpTime, time.Now().Unix())
	}
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
	bucketNum := len(data) / maxBucketSize
	bucketMod := len(data) % maxBucketSize
	var i int
	for i = 0; i < bucketNum; i++ {
		tree := kdtree.New(func(vec, vec1 []float64) (float64, error) {
			return b.distanceFn(vec, vec1)
		})
		tree.Build(items[i*maxBucketSize : (i+1)*maxBucketSize]...)
		if b.buckets == nil {
			b.buckets = map[uuid.UUID]*kdContainer{}
		}
		bucketId := uuid.New()
		b.buckets[bucketId] = &kdContainer{tree: tree}
		for _, bucket := range items[i*maxBucketSize : (i+1)*maxBucketSize] {
			hash := util.СomputeVectorHash(bucket.Points())
			if _, ok := b.hashBucketId[hash]; !ok {
				b.hashBucketId[hash] = []uuid.UUID{}
			}
			b.hashBucketId[hash] = append(b.hashBucketId[hash], bucketId)
		}
	}
	if bucketMod > 0 {
		tree := kdtree.New(func(vec, vec1 []float64) (float64, error) {
			return b.distanceFn(vec, vec1)
		})
		bucketId := uuid.New()
		b.buckets[bucketId] = &kdContainer{tree: tree}
		for _, bucket := range items[i*maxBucketSize : i*maxBucketSize+bucketMod] {
			hash := util.СomputeVectorHash(bucket.Points())
			if _, ok := b.hashBucketId[hash]; !ok {
				b.hashBucketId[hash] = []uuid.UUID{}
			}
			b.hashBucketId[hash] = append(b.hashBucketId[hash], bucketId)
		}
		tree.Build(items[i*maxBucketSize : maxBucketSize*i+bucketMod]...)
	}
	b.greenBlueTree.build(items...)
}

func (b *kd) Reset() {
	b.mtx.Lock()
	b.buckets = map[uuid.UUID]*kdContainer{}
	b.timesTree = avltree.New()
	b.mtx.Unlock()
}

func (b *kd) KNN(vec predictor.Vector, n int) ([]predictor.Vector, error) {
	var kdVectors []predictor.Vector
	items, err := b.greenBlueTree.tree().KNN(vec, n)
	if err != nil {
		return nil, err
	}
	for i := range items {
		kdVectors = append(kdVectors, items[i].(predictor.Vector))
	}
	return kdVectors, nil
}

func (b *kd) Len() int {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return b.greenBlueTree.tree().Len()
}

func (b *kd) Remove(data ...predictor.DataPoint) {
	for i := range data {
		b.remove(data[i])
	}
}

func (b *kd) Append(data ...predictor.DataPoint) {
	for i := range data {
		b.append(data[i])
	}
}

func (b *kd) bucket() uuid.UUID {
	for idx, bucket := range b.buckets {
		if bucket.tree.Len() < maxBucketSize {
			return idx
		}
	}
	bucketId := uuid.New()
	b.buckets[bucketId] = &kdContainer{tree: kdtree.New(func(vec, vec1 []float64) (float64, error) {
		return b.distanceFn(vec, vec1)
	})}
	return bucketId
}

func (b *kd) append(data predictor.DataPoint) {
	hash := util.СomputeVectorHash(data.Vector().Points())
	b.mtx.Lock()
	id := b.bucket()
	if _, ok := b.hashBucketId[hash]; !ok {
		b.hashBucketId[hash] = []uuid.UUID{}
	}
	b.hashBucketId[hash] = append(b.hashBucketId[hash], id)
	b.mtx.Unlock()
	b.buckets[id].mtx.Lock()
	b.buckets[id].tree.Insert(data.Vector())
	b.buckets[id].mtx.Unlock()
	b.mtx.Lock()
	b.timesTree.Add(avlnode.TimeNode{
		K: data.Time(),
		V: data,
	})
	b.mtx.Unlock()
	atomic.AddInt64(&b.gbOpCnt, 1)
	atomic.AddInt64(&b.kdOpCnt, 1)
}

func (b *kd) remove(data predictor.DataPoint) time.Duration {
	t := time.Now()
	hash := util.СomputeVectorHash(data.Vector().Points())
	b.mtx.RLock()
	bucketId, ok := b.hashBucketId[hash]
	if !ok || len(bucketId) == 0 {
		b.mtx.RUnlock()
		return 0
	}

	id := bucketId[0]

	kdContainer, ok := b.buckets[id]
	if !ok {
		b.mtx.RUnlock()
		return 0
	}
	b.mtx.RUnlock()
	kdContainer.mtx.Lock()
	kdContainer.tree.Remove(data.Vector())
	kdContainer.mtx.Unlock()
	b.timesTree.Remove(avlnode.TimeNode{
		K: data.Time(),
		V: data,
	})
	b.mtx.Lock()
	if len(b.hashBucketId[hash]) == 0 {
		delete(b.hashBucketId, hash)
	}
	b.hashBucketId[hash] = b.hashBucketId[hash][1:]
	b.mtx.Unlock()
	kdContainer.mtx.RLock()
	kdContainerLen := kdContainer.tree.Len()
	kdContainer.mtx.RUnlock()
	if kdContainerLen == 0 {
		b.mtx.Lock()
		delete(b.buckets, id)
		b.mtx.Unlock()
	}
	atomic.AddInt64(&b.gbOpCnt, 1)
	atomic.AddInt64(&b.kdOpCnt, 1)
	return time.Since(t)
}

func (b *kd) schedule(ctx context.Context) {
	outdatedTicker := time.NewTicker(rebuildOutdatedTime)
	sizeTicker := time.NewTicker(rebuildSizeTime)
	balanceTreeTicker := time.NewTicker(balanceKDTreeTime)
	greenBlueBuildTicker := time.NewTicker(1 * time.Second)
	defer outdatedTicker.Stop()
	defer sizeTicker.Stop()
	defer balanceTreeTicker.Stop()
	defer greenBlueBuildTicker.Stop()
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
			b.balanceKD()
		case <-greenBlueBuildTicker.C:
			b.buildGB()
		case <-ctx.Done():
			return
		}
	}
}

type greenBlueTree struct {
	green *kdtree.Tree
	blue  *kdtree.Tree
	state uint32
}

func (t *greenBlueTree) tree() *kdtree.Tree {
	if atomic.LoadUint32(&t.state) == 0 {
		return t.green
	}
	return t.blue
}

func (t *greenBlueTree) build(items ...kdtree.Item) {
	if atomic.LoadUint32(&t.state) == 0 {
		t.blue.Build(items...)
		atomic.StoreUint32(&t.state, 1)
	} else {
		t.green.Build(items...)
		atomic.StoreUint32(&t.state, 0)
	}
}

func (b *kd) rebuildOutdated() {
	b.mtx.RLock()
	list := b.timesTree.Filter(func(current avltree.Item) bool {
		return time.Since(current.(avlnode.TimeNode).K) < b.opts.maxStorageTime
	})
	b.mtx.RUnlock()
	for i := range list {
		delay := b.remove(list[i].(avlnode.TimeNode).V)
		time.Sleep(delay)
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
	t := time.Now()
	for i := range list[:sub] {
		b.remove(list[i].(avlnode.TimeNode).V)
	}
	fmt.Println(time.Since(t))
}
