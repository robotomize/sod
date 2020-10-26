package dispatcher

import (
	"context"
	"fmt"
	"runtime"
	"sod/internal/alert"
	"sod/internal/database"
	"sod/internal/logging"
	metricDb "sod/internal/metric/database"
	"sod/internal/metric/model"
	"sod/internal/predictor"
	"sod/pkg/container/iqueue"
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
	deps               pullDependencies
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

// New return constructor for the Manager structure.
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

	d.opts.deps = pullDependencies{
		fetchMetrics:         d.metricDb.FindAll,
		fetchMetricsByEntity: d.metricDb.FindByEntity,
		deleteMetric:         d.metricDb.Delete,
		deleteMetricsFn:      d.metricDb.DeleteMany,
		appendMetricsFn:      d.metricDb.AppendMany,
		fetchKeys:            d.metricDb.Keys,
		countByEntity:        d.metricDb.CountByEntity,
	}

	// Creating a new instance of newDbScheduler.
	d.dbScheduler = newDbScheduler(dbSchedulerConfig{
		deps:           d.opts.deps,
		maxItemsStored: d.opts.maxItemsStored,
		maxStorageTime: d.opts.maxStorageTime,
		rebuildDbTime:  d.opts.rebuildDbTime,
	})

	// Creates a new instance of dbTxExecutor
	d.dbTxExecutor = newDbTxExecutor(
		db,
		dbTxExecutorOptions{
			deps:        d.opts.deps,
			dbFlushTime: d.opts.dbFlushTime,
			dbFlushSize: d.opts.dbFlushSize,
		},
		shutdownCh,
	)

	return d, nil
}

// Contract for returning the Manager instance
type ProvideFn func(alert.Manager, chan<- error) (Manager, error)

// The interface defines the behavior of the Manager instance with all available methods.
// This interface defines the behavior of the background service.
type Manager interface {
	CollectPredictor
	// Start method of the service
	Run(context.Context) error
	// Method for stopping the service
	Stop()
}

// Collector defines the behavior of the service for data storage and analysis
type Collector interface {
	// The method accepts data from outside and writes it to the queue
	Collect(in ...model.Metric) error
}

// The interface defines the behavior of the service only for predictions
type Predictor interface {
	// The method determines whether the data is an outlier
	Predict(entityID string, in predictor.DataPoint) (*predictor.Conclusion, error)
}

// Aggregation interface for Collector and Predictor interfaces
type CollectPredictor interface {
	Collector
	Predictor
}

// Abstractions for getting dependencies
type (
	fetchMetricsFn         func(context.Context, metricDb.FilterFn) ([]model.Metric, error)
	fetchMetricsByEntityFn func(string, metricDb.FilterFn) ([]model.Metric, error)
	deleteMetricFn         func(context.Context, model.Metric) error
	deleteMetricsFn        func(context.Context, []model.Metric) error
	appendMetricsFn        func(context.Context, []model.Metric) error
	fetchKeysFn            func() ([]string, error)
	countByEntityFn        func(string) (int, error)
)

//  General structure for aggregation of dependency pulling functions
type pullDependencies struct {
	fetchMetrics         fetchMetricsFn
	fetchMetricsByEntity fetchMetricsByEntityFn
	deleteMetric         deleteMetricFn
	deleteMetricsFn      deleteMetricsFn
	appendMetricsFn      appendMetricsFn
	fetchKeys            fetchKeysFn
	countByEntity        countByEntityFn
}

// The main structure of SOD.
// Describes the queue management structure, calls outlier notification functions, and stores data predictors.
type manager struct {
	mtx sync.RWMutex

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

	deps := pullDependencies{
		fetchMetrics:         d.metricDb.FindAll,
		fetchMetricsByEntity: d.metricDb.FindByEntity,
		deleteMetric:         d.metricDb.Delete,
		deleteMetricsFn:      d.metricDb.DeleteMany,
		appendMetricsFn:      d.metricDb.AppendMany,
		fetchKeys:            d.metricDb.Keys,
		countByEntity:        d.metricDb.CountByEntity,
	}

	go d.collector(ctx)
	go d.dbTxExecutor.flusher(ctx)
	go d.dbScheduler.schedule(ctx)

	if err := d.bulkLoad(ctx, deps); err != nil {
		return fmt.Errorf("can not start dispatcher manager: %v", err)
	}

	if err := d.notifier.Run(c); err != nil {
		return fmt.Errorf("alert.Run: %w", err)
	}

	return nil
}

func (d *manager) Stop() {
	d.cancel()
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
	result, err := predictorFn.Predict(data.Point())
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

func (d *manager) bulkLoad(ctx context.Context, deps pullDependencies) error {
	var newMetrics []model.Metric

	data, err := deps.fetchMetrics(ctx, nil)
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
		loadPredictor, ok := d.predictors[k]
		if !ok {
			newPredictorFn, err := d.predictorProvideFn()
			if err != nil {
				return fmt.Errorf("can not create predictor instance: %v", err)
			}
			d.predictors[k] = newPredictorFn
			loadPredictor = newPredictorFn
		}
		loadPredictor.Build(list...)
	}

	for i := range newMetrics {
		d.collectCh <- newMetrics[i]
	}

	return nil
}

func (d *manager) process(ctx context.Context, metric model.Metric) error {
	logger := logging.FromContext(ctx)
	d.mtx.RLock()
	entityPredictor, ok := d.predictors[metric.EntityID]
	d.mtx.RUnlock()

	if !ok {
		newPredictor, err := d.predictorProvideFn()
		if err != nil {
			d.mtx.Unlock()
			return fmt.Errorf("can not create predictor instance: %v", err)
		}
		entityPredictor = newPredictor
		d.mtx.Lock()
		d.predictors[metric.EntityID] = newPredictor
		d.mtx.Unlock()
	}

	if entityPredictor.Len() < d.opts.skipItems || entityPredictor.Len() < 3 {
		metric.Status = model.StatusProcessed
		d.dbTxExecutor.append(ctx, metric)
		entityPredictor.Append(&metric)
		return nil
	}

	metric.Status = model.StatusNew

	d.dbTxExecutor.append(ctx, metric)

	result, predictErr := entityPredictor.Predict(metric.Point())
	if predictErr != nil {
		if err := d.opts.deps.deleteMetric(context.Background(), metric); err != nil {
			return fmt.Errorf("unable predict: %v", fmt.Errorf("metric delete error %s: %v", metric.EntityID, err))
		}
		return fmt.Errorf("unable predict: %v", predictErr)
	}

	metric.Outlier = result.Outlier

	if result.Outlier {
		logger.Infof("detect dispatcher, %v", result)
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
		if err := d.opts.deps.deleteMetric(ctx, metric); err != nil {
			return fmt.Errorf("delete transaction error: %v", err)
		}
		return nil
	}

	if (result.Outlier && d.opts.allowAppendOutlier) || !result.Outlier {
		entityPredictor.Append(&metric)
	}

	metric.Status = model.StatusProcessed

	d.dbTxExecutor.append(ctx, metric)

	return nil
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

func (d *manager) shutdown(ctx context.Context, q *iqueue.Queue) error {
	for {
		front := q.Queue().Front()
		if front == nil {
			if !d.recvShutdown() {
				return fmt.Errorf("dispatcher shutdown: closed num receivers not equal created")
			}
			d.cancelNotifier()
			break
		}

		if err := d.process(ctx, front.Value.(model.Metric)); err != nil {
			return fmt.Errorf("dispatcher shutdown: unable processed data: %v", err)
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
		case <-ctx.Done():
			return
		}
	}
}

const workerMul = 2

func (d *manager) worker(ctx context.Context, queue *iqueue.Queue, num int) {
	for i := 0; i < num; i++ {
		go d.receive(ctx, queue)
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
				d.worker(ctx, queue, runtime.NumCPU()*workerMul)
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
