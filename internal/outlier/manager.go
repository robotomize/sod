package outlier

import (
	"context"
	"fmt"
	"rango/internal/alert"
	"rango/internal/database"
	"rango/internal/logging"
	metricDb "rango/internal/metric/database"
	"rango/internal/metric/model"
	"rango/internal/predictor"
	"rango/pkg/container/iqueue"
	"runtime"
	"sync"
	"time"
)

type Options struct {
	skipItems          int
	maxItemsStored     int
	maxStorageTime     time.Duration
	allowAppendData    bool
	allowAppendOutlier bool
	dbFlushTime        time.Duration
	dbFlushSize        int
	rebuildDbTime      time.Duration
}

type Option func(*manager)

func WithDbFlushTime(t time.Duration) Option {
	return func(o *manager) {
		o.opts.dbFlushTime = t
	}
}

func WithDbFlushSize(n int) Option {
	return func(o *manager) {
		o.opts.dbFlushSize = n
	}
}

func WithRebuildDbTime(t time.Duration) Option {
	return func(o *manager) {
		o.opts.rebuildDbTime = t
	}
}

func WithSkipItems(n int) Option {
	return func(o *manager) {
		o.opts.skipItems = n
	}
}

func WithMaxItemsStored(n int) Option {
	return func(o *manager) {
		o.opts.maxItemsStored = n
	}
}

func WithMaxStorageTime(t time.Duration) Option {
	return func(o *manager) {
		o.opts.maxStorageTime = t
	}
}

func WithAllowAppendData(t bool) Option {
	return func(o *manager) {
		o.opts.allowAppendData = t
	}
}

func WithAllowAppendOutlier(t bool) Option {
	return func(o *manager) {
		o.opts.allowAppendOutlier = t
	}
}

func New(
	db *database.DB,
	providePredictorFn predictor.ProvideFn,
	notifier alert.Manager,
	shutdownCh chan<- error,
	opts ...Option,
) (*manager, error) {
	if notifier == nil {
		return nil, fmt.Errorf("notifier instance is not created")
	}
	if providePredictorFn == nil {
		return nil, fmt.Errorf("predictor instance is not created")
	}
	d := &manager{
		metricDb:           metricDb.New(db),
		collectCh:          make(chan model.Metric, 1),
		shutDownCh:         shutdownCh,
		predictorProvideFn: providePredictorFn,
		predictors:         map[string]predictor.Predictor{},
		queue:              map[string]*iqueue.Queue{},
		normVectors:        map[string][]float64{},
		notifier:           notifier,
	}
	for _, f := range opts {
		f(d)
	}
	d.dbScheduler = newDBScheduler(db, dbSchedulerConfig{
		maxItemsStored: d.opts.maxItemsStored,
		maxStorageTime: d.opts.maxStorageTime,
		rebuildDbTime:  d.opts.rebuildDbTime,
	})
	d.dbTxExecutor = newTxExecutor(
		db, dbTxExecutorOptions{dbFlushTime: d.opts.dbFlushTime, dbFlushSize: d.opts.dbFlushSize}, shutdownCh)
	return d, nil
}

type ProvideFn func(alert.Manager, chan<- error) (Manager, error)

type Manager interface {
	CollectPredictor
	Run(context.Context) error
	Stop()
}

type Collector interface {
	Collect(in ...model.Metric) error
}

type Predictor interface {
	Predict(entityID string, in predictor.DataPoint) (*predictor.Conclusion, error)
}

type CollectPredictor interface {
	Collector
	Predictor
}

type manager struct {
	mtx  sync.RWMutex
	opts Options

	metricDb     *metricDb.DB
	notifier     alert.Manager
	dbTxExecutor *dbTxExecutor
	dbScheduler  *dbScheduler

	queue map[string]*iqueue.Queue

	collectCh  chan model.Metric
	shutDownCh chan<- error

	closed             bool
	predictorProvideFn predictor.ProvideFn
	predictors         map[string]predictor.Predictor

	normVectors map[string][]float64

	cancelNotifier func()
	cancel         func()
}

func (d *manager) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	c, cancel := context.WithCancel(context.Background())
	d.cancelNotifier = cancel
	go d.collector(ctx)
	go d.dbTxExecutor.flusher(ctx)
	go d.dbScheduler.schedule(ctx)
	if err := d.initialize(ctx); err != nil {
		return fmt.Errorf("can not start outlier manager: %v", err)
	}
	if err := d.notifier.Run(c); err != nil {
		return fmt.Errorf("alert.Run: %w", err)
	}
	return nil
}

func (d *manager) Stop() {
	d.cancel()
}

func (d *manager) alert(in ...model.Metric) {
	d.mtx.RLock()
	if !d.closed {
		d.mtx.RUnlock()
		d.notifier.Notify(in...)
		return
	}
	d.mtx.RUnlock()
}

func (d *manager) Predict(entityID string, data predictor.DataPoint) (*predictor.Conclusion, error) {
	d.mtx.Lock()
	if d.closed {
		d.mtx.Unlock()
		return nil, fmt.Errorf("error to predict, shutting down")
	}

	predictorFn, ok := d.predictors[entityID]
	if !ok {
		newPredictor, err := d.predictorProvideFn()
		if err != nil {
			d.mtx.Unlock()
			return nil, fmt.Errorf("can not create predictor instance: %v", err)
		}
		predictorFn = newPredictor
		d.predictors[entityID] = newPredictor
	}
	d.mtx.Unlock()
	result, err := predictorFn.Predict(data.Vector())
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *manager) Collect(data ...model.Metric) error {
	d.mtx.RLock()
	if d.closed {
		d.mtx.RUnlock()
		return fmt.Errorf("error send to collect, shutting down")
	}
	for i := range data {
		d.collectCh <- data[i]
	}
	d.mtx.RUnlock()
	return nil
}

func (d *manager) shutdown(ctx context.Context, q *iqueue.Queue) error {
	for {
		front := q.Queue().Front()
		if front == nil {
			if !d.recvShutdown() {
				return fmt.Errorf("outlier shutdown: closed num receivers not equal created")
			}
			d.cancelNotifier()
			break
		}
		if err := d.process(ctx, front.Value.(model.Metric)); err != nil {
			return fmt.Errorf("outlier shutdown: unable processed data: %v", err)
		}
		q.Queue().Remove(front)
	}
	return nil
}

func (d *manager) recvShutdown() bool {
	finishedNum, predictorsNum := 0, len(d.queue)
	for _, q := range d.queue {
		if q.Queue().Len() == 0 {
			finishedNum += 1
		}
	}
	return finishedNum == predictorsNum
}

func (d *manager) initialize(ctx context.Context) error {
	var newMetrics []model.Metric

	data, err := d.metricDb.FindAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("error fetching all metrics: %v", err)
	}
	processedMetrics := map[string][]predictor.DataPoint{}
	for _, dat := range data {
		if _, ok := processedMetrics[dat.EntityID]; !ok {
			processedMetrics[dat.EntityID] = []predictor.DataPoint{}
		}
		if dat.IsProcessed() {
			processedMetrics[dat.EntityID] = append(processedMetrics[dat.EntityID], dat)
		}
		if dat.IsNew() {
			newMetrics = append(newMetrics, dat)
		}
	}

	for k, list := range processedMetrics {
		predictor, ok := d.predictors[k]
		if !ok {
			newPredictorFn, err := d.predictorProvideFn()
			if err != nil {
				return fmt.Errorf("can not create predictor instance: %v", err)
			}
			d.predictors[k] = newPredictorFn
			predictor = newPredictorFn
		}
		predictor.Build(list...)
	}

	for i := range newMetrics {
		d.collectCh <- newMetrics[i]
	}
	return nil
}

func (d *manager) process(ctx context.Context, metric model.Metric) error {
	logger := logging.FromContext(ctx)
	d.mtx.Lock()
	entityPredictor, ok := d.predictors[metric.EntityID]
	if !ok {
		newPredictor, err := d.predictorProvideFn()
		if err != nil {
			d.mtx.Unlock()
			return fmt.Errorf("can not create predictor instance: %v", err)
		}
		entityPredictor = newPredictor
		d.predictors[metric.EntityID] = newPredictor
	}
	d.mtx.Unlock()

	if entityPredictor.Len() < d.opts.skipItems || entityPredictor.Len() < 3 {
		metric.Status = model.StatusProcessed
		if err := d.dbTxExecutor.append(metric); err != nil {
			return fmt.Errorf("error commit to tx executor: %v", err)
		}
		entityPredictor.Append(&metric)
		return nil
	}

	metric.Status = model.StatusNew
	if err := d.dbTxExecutor.append(metric); err != nil {
		return fmt.Errorf("error commit to tx executor: %v", err)
	}

	result, predictErr := entityPredictor.Predict(metric.Vector())
	if predictErr != nil {
		if err := d.metricDb.Delete(context.Background(), metric); err != nil {
			return fmt.Errorf("unable predict: %v", fmt.Errorf("metric delete error %s: %v", metric.EntityID, err))
		}
		return fmt.Errorf("unable predict: %v", predictErr)
	}

	metric.Outlier = result.Outlier

	if result.Outlier {
		logger.Infof("detect outlier, %v", result)
		d.mtx.RLock()
		if vec, ok := d.normVectors[metric.EntityID]; ok {
			metric.NormVec = vec
		}
		d.mtx.RUnlock()
		d.alert(metric)
	} else {
		d.mtx.Lock()
		d.normVectors[metric.EntityID] = metric.NormVec
		d.mtx.Unlock()
	}

	if !d.opts.allowAppendData {
		if err := d.metricDb.Delete(ctx, metric); err != nil {
			return fmt.Errorf("delete transaction error: %v", err)
		}
		return nil
	}

	if (result.Outlier && d.opts.allowAppendOutlier) || !result.Outlier {
		entityPredictor.Append(&metric)
	}

	metric.Status = model.StatusProcessed
	if err := d.dbTxExecutor.append(metric); err != nil {
		return fmt.Errorf("error commit to tx executor: %v", err)
	}
	return nil
}

func (d *manager) receive(ctx context.Context, q *iqueue.Queue) {
	logger := logging.FromContext(ctx)
	defer func() {
		d.shutDownCh <- d.shutdown(ctx, q)
	}()
	for {
		select {
		case recv := <-q.Receive():
			if err := d.process(ctx, recv.(model.Metric)); err != nil {
				logger.Errorf("unable processed data: %v", err)
			}
			fmt.Println("queue: ", q.Len())
		case <-ctx.Done():
			return
		}
	}
}

func (d *manager) collector(ctx context.Context) {
	defer close(d.collectCh)
	for {
		select {
		case in := <-d.collectCh:
			q, ok := d.queue[in.EntityID]
			if !ok {
				queue := iqueue.New()
				go queue.Loop()
				for i := 0; i < runtime.NumCPU(); i++ {
					go d.receive(ctx, queue)
				}

				d.queue[in.EntityID] = queue
				q = queue
			}
			q.Send(in)
		case <-ctx.Done():
			d.mtx.Lock()
			d.closed = true
			d.mtx.Unlock()
			return
		}
	}
}