package dispatcher

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/go-sod/sod/internal/alert"
	"github.com/go-sod/sod/internal/database"
	"github.com/go-sod/sod/internal/logging"
	metricDb "github.com/go-sod/sod/internal/metric/database"
	"github.com/go-sod/sod/internal/metric/model"
	"github.com/go-sod/sod/internal/predictor"
	"github.com/go-sod/sod/pkg/iqueue"
)

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
	// function for getting all metrics
	fetchMetricsFn func(context.Context, metricDb.FilterFn) ([]model.Metric, error)
	// function for getting metrics based on the loyalty id
	fetchMetricsByEntityFn func(string, metricDb.FilterFn) ([]model.Metric, error)
	// function for deleting a metric
	deleteMetricFn func(context.Context, model.Metric) error
	// function for deleting multiple metrics
	deleteMetricsFn func(context.Context, []model.Metric) error
	// function to add sets of metrics
	appendMetricsFn func(context.Context, []model.Metric) error
	// function for getting all entity IDs
	fetchKeysFn func() ([]string, error)
	// number of metrics by entity id
	countByEntityFn func(string) (int, error)
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

type Options struct {
	skipItems          int
	maxItemsStored     int
	maxStorageTime     time.Duration
	allowAppendData    bool
	allowAppendOutlier bool
	dbFlushTime        time.Duration
	dbFlushSize        int
	rebuildDBTime      time.Duration
	deps               pullDependencies
}

type Option func(*manager)

func WithDBFlushTime(t time.Duration) Option {
	return func(o *manager) {
		o.opts.dbFlushTime = t
	}
}

func WithDBFlushSize(n int) Option {
	return func(o *manager) {
		o.opts.dbFlushSize = n
	}
}

func WithRebuildDBTime(t time.Duration) Option {
	return func(o *manager) {
		o.opts.rebuildDBTime = t
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

// New return manager
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
		metricDB:           metricDb.New(db),
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

	// structure containing functions for getting and adding metrics
	d.opts.deps = pullDependencies{
		fetchMetrics:         d.metricDB.FindAll,
		fetchMetricsByEntity: d.metricDB.FindByEntity,
		deleteMetric:         d.metricDB.Delete,
		deleteMetricsFn:      d.metricDB.DeleteMany,
		appendMetricsFn:      d.metricDB.AppendMany,
		fetchKeys:            d.metricDB.Keys,
		countByEntity:        d.metricDB.CountByEntity,
	}

	// Creating a new instance of newDBScheduler.
	d.dbScheduler = newDBScheduler(dbSchedulerConfig{
		deps:           d.opts.deps,
		maxItemsStored: d.opts.maxItemsStored,
		maxStorageTime: d.opts.maxStorageTime,
		rebuildDBTime:  d.opts.rebuildDBTime,
	})

	// Creates a new instance of dbTxExecutor
	d.dbTxExecutor = newDBTxExecutor(
		db,
		dbTxExecutorOptions{
			deps:      d.opts.deps,
			flushTime: d.opts.dbFlushTime,
			flushSize: d.opts.dbFlushSize,
		},
		shutdownCh,
	)

	return d, nil
}

// The main structure of SOD.
// Describes the queue management structure, calls outlier notification functions, and stores data predictors.
type manager struct {
	mtx sync.RWMutex

	// Manager options
	opts Options
	//  Main metric storage
	metricDB *metricDb.DB
	//  The notification manager
	notifier alert.Manager
	// The transaction manager in the store
	dbTxExecutor *dbTxExecutor
	// Managing data in storage
	dbScheduler *dbScheduler

	// Queue for new data to be processed
	queue map[string]*iqueue.Queue
	// New data channel for processing
	collectCh chan model.Metric
	// Channel to shutdown the application
	shutDownCh chan<- error

	closed bool
	// The factory returns an instance of the predictor
	predictorProvideFn predictor.ProvideFn
	// Created predictors
	predictors map[string]predictor.Predictor
	// The last vector is not outlier
	normVectors map[string][]float64

	// cancellation
	cancelNotifier func()
	cancel         func()
}

// The Run method starts the main data collection and analysis functions
func (d *manager) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	c, cancel := context.WithCancel(context.Background())
	d.cancelNotifier = cancel

	go d.collector(ctx)
	go d.dbTxExecutor.flusher(ctx)
	go d.dbScheduler.schedule(ctx)

	// Loading data from storage to memory
	if err := d.bulkLoad(ctx); err != nil {
		return fmt.Errorf("can not start dispatcher manager: %w", err)
	}
	// Launching the notification service
	if err := d.notifier.Run(c); err != nil {
		return fmt.Errorf("alert.Run: %w", err)
	}

	return nil
}

// Stop the manager
func (d *manager) Stop() {
	d.cancel()
}

// Predict returns a structure with the result of checking the transmitted data for deviations
func (d *manager) Predict(entityID string, data predictor.DataPoint) (*predictor.Conclusion, error) {
	d.mtx.Lock()
	if d.closed {
		d.mtx.Unlock()
		return nil, fmt.Errorf("error to predict, shutting down")
	}
	//  If the predictor instance does not exist we return a new one from the factory
	predictorFn, ok := d.predictors[entityID]
	if !ok {
		newPredictor, err := d.predictorProvideFn()
		if err != nil {
			d.mtx.Unlock()
			return nil, fmt.Errorf("can not create predictor instance: %w", err)
		}
		predictorFn = newPredictor
		d.predictors[entityID] = newPredictor
	}

	d.mtx.Unlock()
	// Calling predict
	result, err := predictorFn.Predict(data.Point())
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Collect adds data to the feed for saving to the queue
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

// bulkLoad loading data from storage to memory
func (d *manager) bulkLoad(ctx context.Context) error {
	var newMetrics []model.Metric

	// getting all metrics that are in the storage
	data, err := d.opts.deps.fetchMetrics(ctx, nil)
	if err != nil {
		return fmt.Errorf("error fetching all metrics: %w", err)
	}

	processedMetrics := map[string][]predictor.DataPoint{}
	for _, dat := range data {
		if _, ok := processedMetrics[dat.EntityID]; !ok {
			processedMetrics[dat.EntityID] = []predictor.DataPoint{}
		}
		// divide metrics by the statuses "processed" and " new"
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
				return fmt.Errorf("can not create predictor instance: %w", err)
			}
			d.predictors[k] = newPredictorFn
			loadPredictor = newPredictorFn
		}
		// bulk load data to the predictor
		loadPredictor.Build(list...)
	}
	// metrics with the "new" status are sent to the queue for processing
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
			return fmt.Errorf("can not create predictor instance: %w", err)
		}
		entityPredictor = newPredictor
		d.mtx.Lock()
		d.predictors[metric.EntityID] = newPredictor
		d.mtx.Unlock()
	}

	if entityPredictor.Len() < d.opts.skipItems || entityPredictor.Len() < 3 {
		metric.Status = model.StatusProcessed
		d.dbTxExecutor.write(ctx, metric)
		entityPredictor.Append(&metric)
		return nil
	}

	metric.Status = model.StatusNew

	d.dbTxExecutor.write(ctx, metric)

	result, predictErr := entityPredictor.Predict(metric.Point())
	if predictErr != nil {
		if err := d.opts.deps.deleteMetric(context.Background(), metric); err != nil {
			return fmt.Errorf("unable predict: %w", err)
		}
		return fmt.Errorf("unable predict: %w", predictErr)
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
			return fmt.Errorf("delete transaction error: %w", err)
		}
		return nil
	}

	if (result.Outlier && d.opts.allowAppendOutlier) || !result.Outlier {
		entityPredictor.Append(&metric)
	}

	metric.Status = model.StatusProcessed

	d.dbTxExecutor.write(ctx, metric)

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
			return fmt.Errorf("dispatcher shutdown: unable processed data: %w", err)
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
