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
		shutdownCh     chan error
		expectedErr    error
		expectedLen    int
		expectedBufLen int
		waitingTime    time.Duration
		batch          []model.Metric
	}{
		{
			name:        "positive_flusher",
			waitingTime: 1 * time.Second,
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
			txExecutor := &dbTxExecutor{
				opts: dbTxExecutorOptions{
					dbFlushTime: 1 * time.Second,
					deps: pullDependencies{
						appendMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
							if atomic.LoadInt64(&bit) == 0 {
								length = len(metrics)
								atomic.StoreInt64(&bit, 1)
							}

							return nil
						},
						fetchKeys:     nil,
						countByEntity: nil,
					},
				},
			}

			ctx, cancel := context.WithCancel(context.TODO())
			txExecutor.buf = test.batch
			go txExecutor.flusher(ctx)

			time.Sleep(test.waitingTime * 2)
			cancel()

			if length != test.expectedLen {
				t.Errorf(
					"calling the flusher method, the length of the inserted data got: %v, expected: %v",
					length,
					test.expectedLen,
				)
			}

			if len(txExecutor.buf) != test.expectedBufLen {
				t.Errorf(
					"calling the shutdown method, the length of buffer got: %v, expected: %v",
					len(txExecutor.buf),
					test.expectedBufLen,
				)
			}
		})
	}
}

func TestDbTxExecutorAppend(t *testing.T) {
	tests := []struct {
		name           string
		items          []model.Metric
		shutdownCh     chan error
		expectedErr    error
		expectedLen    int
		expectedBufLen int
	}{
		{
			name: "positive_append",
			items: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 1,
		},
		{
			name: "positive_append",
			items: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 2,
		},
		{
			name: "positive_append",
			items: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			txExecutor := &dbTxExecutor{opts: dbTxExecutorOptions{deps: pullDependencies{
				appendMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
					return nil
				},
			}}}

			for _, item := range test.items {
				txExecutor.append(context.Background(), item)
			}

			if len(txExecutor.buf) != test.expectedLen {
				t.Errorf(
					"calling the append method, the length of the inserted data got: %v, expected: %v",
					len(txExecutor.buf),
					test.expectedLen,
				)
			}
		})
	}
}

func TestDbTxExecutorBulkAppend(t *testing.T) {
	tests := []struct {
		name           string
		shutdownCh     chan error
		expectedErr    error
		expectedLen    int
		expectedBufLen int
		buf            []model.Metric
	}{
		{
			name: "positive_bulk_append",
			buf: []model.Metric{
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
		{
			name:           "positive_bulk_append",
			buf:            []model.Metric{},
			expectedLen:    0,
			expectedBufLen: 0,
			expectedErr:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			length := 0
			txExecutor := &dbTxExecutor{
				buf: test.buf,
				opts: dbTxExecutorOptions{deps: pullDependencies{
					appendMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
						length = len(metrics)
						return nil
					},
				}}}

			txExecutor.bulkAppend(context.Background())

			if length != test.expectedLen {
				t.Errorf(
					"calling the bulkAppend method, the length of the inserted data got: %v, expected: %v",
					length,
					test.expectedLen,
				)
			}

			if len(txExecutor.buf) != test.expectedBufLen {
				t.Errorf(
					"calling the bulkAppend method, the length of buffer got: %v, expected: %v",
					len(txExecutor.buf),
					test.expectedBufLen,
				)
			}
		})
	}
}

func TestDbTxExecutorShutdown(t *testing.T) {
	tests := []struct {
		name           string
		shutdownCh     chan error
		expectedLen    int
		expectedBufLen int
		expectedErr    error
		buf            []model.Metric
	}{
		{
			name: "positive_shutdown",
			buf: []model.Metric{
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
		{
			name:           "negative_shutdown",
			buf:            []model.Metric{},
			expectedLen:    0,
			expectedBufLen: 0,
			expectedErr:    errors.New("test"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			length := 0
			txExecutor := &dbTxExecutor{opts: dbTxExecutorOptions{deps: pullDependencies{
				appendMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
					length = len(metrics)

					return test.expectedErr
				},
			}}}

			txExecutor.buf = test.buf
			err := txExecutor.shutdown()

			if test.expectedErr == nil && err != nil {
				t.Errorf("calling the shutdown method, err got: %v, expected: %v", err, test.expectedErr)
			}

			if length != test.expectedLen {
				t.Errorf(
					"calling the shutdown method, the length of the inserted data got: %v, expected: %v",
					length,
					test.expectedLen,
				)
			}

			if err == nil && len(txExecutor.buf) != test.expectedBufLen {
				t.Errorf(
					"calling the shutdown method, the length of buffer got: %v, expected: %v",
					len(txExecutor.buf),
					test.expectedBufLen,
				)
			}
		})
	}
}
