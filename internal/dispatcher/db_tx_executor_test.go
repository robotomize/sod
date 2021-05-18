package dispatcher

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-sod/sod/internal/geom"
	"github.com/go-sod/sod/internal/metric/model"
)

func TestDbxExecutorFlusher(t *testing.T) {
	t.Parallel()
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
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			appendedCnt := 0
			mtx := sync.RWMutex{}
			txExecutor := &dbTxExecutor{
				opts: dbTxExecutorOptions{
					flushTime: 500 * time.Millisecond,
					deps: pullDependencies{
						appendMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
							mtx.Lock()
							defer mtx.Unlock()

							if len(metrics) > 0 {
								appendedCnt = len(metrics)
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

			time.Sleep(test.waitingTime)

			cancel()
			mtx.RLock()
			if appendedCnt != test.expectedLen {
				t.Errorf(
					"calling the flusher method, the length of the inserted data got: %v, expected: %v",
					appendedCnt,
					test.expectedLen,
				)
			}
			mtx.RUnlock()

			if txExecutor.len() != test.expectedBufLen {
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
	t.Parallel()
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
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			txExecutor := &dbTxExecutor{
				opts: dbTxExecutorOptions{
					flushSize: 10,
					deps: pullDependencies{
						appendMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
							return nil
						},
					},
				},
			}

			for _, item := range test.items {
				txExecutor.write(context.Background(), item)
			}

			if txExecutor.len() != test.expectedLen {
				t.Errorf(
					"calling the write method, the length of the inserted data got: %v, expected: %v",
					len(txExecutor.buf),
					test.expectedLen,
				)
			}
		})
	}
}

func TestDbTxExecutorFlush(t *testing.T) {
	t.Parallel()
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
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			length := 0
			txExecutor := &dbTxExecutor{
				buf: test.buf[:],
				opts: dbTxExecutorOptions{
					deps: pullDependencies{
						appendMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
							length = len(metrics)
							return nil
						},
					},
				},
			}

			txExecutor.flush(context.Background())

			if length != test.expectedLen {
				t.Errorf(
					"calling the flush method, the length of the inserted data got: %v, expected: %v",
					length,
					test.expectedLen,
				)
			}
			if len(txExecutor.buf) != test.expectedBufLen {
				t.Errorf(
					"calling the flush method, the length of buffer got: %v, expected: %v",
					len(txExecutor.buf),
					test.expectedBufLen,
				)
			}
		})
	}
}

func TestDbTxExecutorShutdown(t *testing.T) {
	t.Parallel()
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
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
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
