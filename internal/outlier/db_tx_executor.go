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
	closed     bool
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

func (tx *dbTxExecutor) append(data model.Metric) error {
	tx.mtx.Lock()
	if tx.buf == nil {
		tx.buf = []model.Metric{}
	}
	tx.buf = append(tx.buf, data)
	if len(tx.buf) >= tx.opts.dbFlushSize {
		if err := tx.metricDb.AppendMany(context.Background(), tx.buf); err != nil {
			return fmt.Errorf("txExecutor: append many operation failed: %v", err)
		}
		tx.buf = tx.buf[:0]
	}
	tx.mtx.Unlock()
	return nil
}

func (tx *dbTxExecutor) flusher(ctx context.Context) {
	logger := logging.FromContext(ctx)
	defer func() {
		tx.shutdownCh <- tx.shutdown()
	}()
	ticker := time.NewTicker(tx.opts.dbFlushTime)
	for {
		select {
		case <-ticker.C:
			tx.mtx.Lock()
			if err := tx.metricDb.AppendMany(context.Background(), tx.buf); err != nil {
				logger.Errorf("txExecutor: append many operation failed: %v", err)
			}
			tx.buf = tx.buf[:0]
			tx.mtx.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
