package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-sod/sod/internal/database"
	"github.com/go-sod/sod/internal/logging"
	metricDb "github.com/go-sod/sod/internal/metric/database"
	"github.com/go-sod/sod/internal/metric/model"
)

func newDBTxExecutor(db *database.DB, opts dbTxExecutorOptions, shutdownCh chan<- error) *dbTxExecutor {
	return &dbTxExecutor{metricDB: metricDb.New(db), opts: opts, shutdownCh: shutdownCh}
}

// dbTxExecutorOptions Returns the structure with configuration options
type dbTxExecutorOptions struct {
	flushSize int
	flushTime time.Duration
	deps      pullDependencies
}

// A structure that represents the database transaction execution service.
// Accumulates a queue of data and inserts it in bulk into persistent storage.
type dbTxExecutor struct {
	mtx sync.RWMutex

	opts     dbTxExecutorOptions
	metricDB *metricDb.DB
	//  Buffer that accumulates metric data for adding
	buf        []model.Metric
	shutdownCh chan<- error
}

// Urgently inserts all data from the buffer into persistent storage or returns an error
func (tx *dbTxExecutor) shutdown() error {
	tx.mtx.Lock()
	if err := tx.opts.deps.appendMetricsFn(context.Background(), tx.buf); err != nil {
		return fmt.Errorf("txExecutor: write many operation failed: %w", err)
	}
	tx.buf = tx.buf[:0]
	tx.mtx.Unlock()
	return nil
}

// This is the main method for adding data. It adds data to the buffer.
// If the buffer is full, it calls the flush method
func (tx *dbTxExecutor) write(ctx context.Context, data model.Metric) {
	tx.mtx.Lock()
	if len(tx.buf) == 0 {
		tx.buf = make([]model.Metric, 0)
	}

	tx.buf = append(tx.buf, data)
	bufLen := len(tx.buf)
	tx.mtx.Unlock()

	if bufLen >= tx.opts.flushSize {
		go tx.flush(ctx)
	}
}

// Bulk adds data to persistent storage and clears the buffer
func (tx *dbTxExecutor) flush(ctx context.Context) {
	logger := logging.FromContext(ctx)

	tx.mtx.Lock()
	tmpBuf := make([]model.Metric, len(tx.buf))
	copy(tmpBuf, tx.buf)
	tx.buf = tx.buf[:0]
	tx.mtx.Unlock()
	// call appendMetricsFn
	if err := tx.opts.deps.appendMetricsFn(context.Background(), tmpBuf); err != nil {
		logger.Errorf("txExecutor: flush operation failed: %v", err)
	}
}

func (tx *dbTxExecutor) len() int {
	tx.mtx.RLock()
	defer tx.mtx.RUnlock()

	return len(tx.buf)
}

// Every n seconds, data from the buffer must be inserted into the database
func (tx *dbTxExecutor) flusher(ctx context.Context) {
	defer func() {
		tx.shutdownCh <- tx.shutdown()
	}()
	ticker := time.NewTicker(tx.opts.flushTime)
	for {
		select {
		case <-ticker.C:
			tx.flush(ctx)
		case <-ctx.Done():
			return
		}
	}
}
