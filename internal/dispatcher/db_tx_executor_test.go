package dispatcher

import (
	"context"
	"errors"
	"sod/internal/geom"
	"sod/internal/metric/model"
	"sync/atomic"
	"testing"
	"time"
)

func TestDbxExecutorFlusher(t *testing.T) {
	tests := []struct {
		name           string
		txExecutor     *dbTxExecutor
		shutdownCh     chan error
		expectedErr    error
		expectedLen    int
		expectedBufLen int
		waitingTime    time.Duration
		batch          []model.Metric
	}{
		{
			name:        "positive_shutdown",
			waitingTime: 1 * time.Second,
			txExecutor:  &dbTxExecutor{opts: dbTxExecutorOptions{dbFlushTime: 1 * time.Second}},
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:    5,
			expectedBufLen: 0,
			expectedErr:    nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			length := 0
			bit := int64(0)
			ctx, cancel := context.WithCancel(context.TODO())
			test.txExecutor.buf = test.batch
			go test.txExecutor.flusher(ctx, func(ctx context.Context, metrics []model.Metric) error {
				if atomic.LoadInt64(&bit) == 0 {
					length = len(metrics)
				} else {
					atomic.StoreInt64(&bit, 1)
				}

				return nil
			})

			time.Sleep(test.waitingTime)
			cancel()
			if length != test.expectedLen {
				t.Errorf(
					"calling the shutdown method, the length of the inserted data got: %v, expected: %v",
					length, test.expectedLen)
			}
			if len(test.txExecutor.buf) != test.expectedBufLen {
				t.Errorf(
					"calling the shutdown method, the length of buffer got: %v, expected: %v",
					len(test.txExecutor.buf), test.expectedBufLen)
			}
		})
	}
}

func TestDbTxExecutorAppend(t *testing.T) {
	tests := []struct {
		name           string
		txExecutor     *dbTxExecutor
		model          model.Metric
		shutdownCh     chan error
		expectedErr    error
		expectedLen    int
		expectedBufLen int
	}{
		{
			name:        "positive_shutdown",
			txExecutor:  &dbTxExecutor{},
			model:       model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			expectedLen: 1,
			expectedErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.txExecutor.append(context.Background(), test.model, func(ctx context.Context, metrics []model.Metric) error {
				return nil
			})
			if len(test.txExecutor.buf) != test.expectedLen {
				t.Errorf(
					"calling the shutdown method, the length of the inserted data got: %v, expected: %v",
					len(test.txExecutor.buf), test.expectedLen)
			}
		})
	}
}

func TestDbTxExecutorBulkAppend(t *testing.T) {
	tests := []struct {
		name           string
		txExecutor     *dbTxExecutor
		shutdownCh     chan error
		expectedErr    error
		expectedLen    int
		expectedBufLen int
	}{
		{
			name: "positive_shutdown",
			txExecutor: &dbTxExecutor{
				buf: []model.Metric{
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				},
			},
			expectedLen:    5,
			expectedBufLen: 0,
			expectedErr:    nil,
		},
		{
			name: "negative_shutdown",
			txExecutor: &dbTxExecutor{
				buf: []model.Metric{},
			},
			expectedLen:    0,
			expectedBufLen: 0,
			expectedErr:    nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			length := 0

			test.txExecutor.bulkAppend(context.Background(), func(ctx context.Context, metrics []model.Metric) error {
				length = len(metrics)
				return nil
			})
			if length != test.expectedLen {
				t.Errorf(
					"calling the shutdown method, the length of the inserted data got: %v, expected: %v",
					length, test.expectedLen)
			}
			if len(test.txExecutor.buf) != test.expectedBufLen {
				t.Errorf(
					"calling the shutdown method, the length of buffer got: %v, expected: %v",
					len(test.txExecutor.buf), test.expectedBufLen)
			}
		})
	}
}

func TestDbTxExecutorShutdown(t *testing.T) {
	tests := []struct {
		name           string
		txExecutor     *dbTxExecutor
		shutdownCh     chan error
		expectedLen    int
		expectedBufLen int
		expectedErr    error
	}{
		{
			name: "positive_shutdown",
			txExecutor: &dbTxExecutor{
				buf: []model.Metric{
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
					model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				},
			},
			expectedLen:    5,
			expectedBufLen: 0,
			expectedErr:    nil,
		},
		{
			name: "negative_shutdown",
			txExecutor: &dbTxExecutor{
				buf: []model.Metric{},
			},
			expectedLen:    0,
			expectedBufLen: 0,
			expectedErr:    errors.New("test"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			length := 0

			err := test.txExecutor.shutdown(func(ctx context.Context, metrics []model.Metric) error {
				length = len(metrics)
				if test.expectedErr != nil {
					return test.expectedErr
				}
				return nil
			})
			if test.expectedErr == nil && err != nil {
				t.Errorf("calling the shutdown method, err got: %v, expected: %v", err, test.expectedErr)
			}
			if length != test.expectedLen {
				t.Errorf(
					"calling the shutdown method, the length of the inserted data got: %v, expected: %v",
					length, test.expectedLen)
			}
			if len(test.txExecutor.buf) != test.expectedBufLen {
				t.Errorf(
					"calling the shutdown method, the length of buffer got: %v, expected: %v",
					len(test.txExecutor.buf), test.expectedBufLen)
			}
		})
	}
}
