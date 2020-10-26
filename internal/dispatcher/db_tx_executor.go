package dispatcher

import (
	"context"
	"fmt"
	"sod/internal/database"
	"sod/internal/logging"
	metricDb "sod/internal/metric/database"
	"sod/internal/metric/model"
	"sync"
	"time"
)

func newDbTxExecutor(db *database.DB, opts dbTxExecutorOptions, shutdownCh chan<- error) *dbTxExecutor {
	return &dbTxExecutor{metricDb: metricDb.New(db), opts: opts, shutdownCh: shutdownCh}
}

// dbTxExecutorOptions Returns the structure with configuration options
type dbTxExecutorOptions struct {
	dbFlushSize int
	dbFlushTime time.Duration
	deps        pullDependencies
}

// A structure that represents the database transaction execution service.
// Accumulates a queue of data and inserts it in bulk into persistent storage.
type dbTxExecutor struct {
	mtx sync.RWMutex

	opts     dbTxExecutorOptions
	metricDb *metricDb.DB
	//  Buffer that accumulates metric data for adding
	buf        []model.Metric
	shutdownCh chan<- error
}

// Urgently inserts all data from the buffer into persistent storage or returns an error
func (tx *dbTxExecutor) shutdown() error {
	tx.mtx.Lock()
	if err := tx.opts.deps.appendMetricsFn(context.Background(), tx.buf); err != nil {
		return fmt.Errorf("txExecutor: append many operation failed: %v", err)
	}
	tx.buf = tx.buf[:0]
	tx.mtx.Unlock()
	return nil
}

// This is the main method for adding data. It adds data to the buffer.
// If the buffer is full, it calls the bulkAppend method
func (tx *dbTxExecutor) append(ctx context.Context, data model.Metric) {
	tx.mtx.Lock()
	if tx.buf == nil {
		tx.buf = []model.Metric{}
	}

	tx.buf = append(tx.buf, data)
	bufLen := len(tx.buf)
	tx.mtx.Unlock()

	if bufLen >= tx.opts.dbFlushSize {
		go tx.bulkAppend(ctx)
	}
}

// Bulk adds data to persistent storage and clears the buffer
func (tx *dbTxExecutor) bulkAppend(ctx context.Context) {
	logger := logging.FromContext(ctx)

	tx.mtx.Lock()
	tmpBuf := make([]model.Metric, len(tx.buf))
	copy(tmpBuf, tx.buf)
	tx.buf = tx.buf[:0]
	tx.mtx.Unlock()
	// call appendMetricsFn
	if err := tx.opts.deps.appendMetricsFn(context.Background(), tmpBuf); err != nil {
		logger.Errorf("txExecutor: append many operation failed: %v", err)
	}
}

// Every n seconds, data from the buffer must be inserted into the database
func (tx *dbTxExecutor) flusher(ctx context.Context) {
	defer func() {
		tx.shutdownCh <- tx.shutdown()
	}()
	ticker := time.NewTicker(tx.opts.dbFlushTime)
	for {
		select {
		case <-ticker.C:
			tx.bulkAppend(ctx)
		case <-ctx.Done():
			return
		}
	}
}
