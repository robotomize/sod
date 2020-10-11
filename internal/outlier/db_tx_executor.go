package outlier

import (
	"context"
	"fmt"
	"rango/internal/database"
	"rango/internal/logging"
	metricDb "rango/internal/metric/database"
	"rango/internal/metric/model"
	"sync"
	"time"
)

type StoreFn func(context.Context, model.Metric) error

func newTxExecutor(db *database.DB, opts dbTxExecutorOptions, shutdownCh chan<- error) *dbTxExecutor {
	return &dbTxExecutor{metricDb: metricDb.New(db), opts: opts, shutdownCh: shutdownCh}
}

type dbTxExecutorOptions struct {
	dbFlushSize int
	dbFlushTime time.Duration
}

type dbTxExecutor struct {
	mtx        sync.RWMutex
	opts       dbTxExecutorOptions
	metricDb   *metricDb.DB
	buf        []model.Metric
	shutdownCh chan<- error
}

func (tx *dbTxExecutor) shutdown() error {
	tx.mtx.Lock()
	if err := tx.metricDb.AppendMany(context.Background(), tx.buf); err != nil {
		return fmt.Errorf("txExecutor: append many operation failed: %v", err)
	}
	tx.buf = tx.buf[:0]
	tx.mtx.Unlock()
	return nil
}

func (tx *dbTxExecutor) append(ctx context.Context, data model.Metric) error {
	tx.mtx.Lock()
	if tx.buf == nil {
		tx.buf = []model.Metric{}
	}
	tx.buf = append(tx.buf, data)
	bufLen := len(tx.buf)
	tx.mtx.Unlock()

	if bufLen >= tx.opts.dbFlushSize {
		go tx.appendMany(ctx)
	}
	return nil
}

func (tx *dbTxExecutor) appendMany(ctx context.Context) {
	logger := logging.FromContext(ctx)
	tx.mtx.Lock()
	tmpBuf := make([]model.Metric, len(tx.buf))
	copy(tmpBuf, tx.buf)
	tx.buf = tx.buf[:0]
	tx.mtx.Unlock()
	if err := tx.metricDb.AppendMany(context.Background(), tmpBuf); err != nil {
		logger.Errorf("txExecutor: append many operation failed: %v", err)
	}
}

func (tx *dbTxExecutor) flusher(ctx context.Context) {
	defer func() {
		tx.shutdownCh <- tx.shutdown()
	}()
	ticker := time.NewTicker(tx.opts.dbFlushTime)
	for {
		select {
		case <-ticker.C:
			tx.appendMany(ctx)
		case <-ctx.Done():
			return
		}
	}
}
