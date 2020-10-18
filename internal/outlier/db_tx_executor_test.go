package outlier

import (
	"context"
	"sod/internal/geom"
	"sod/internal/metric/model"
	"testing"
	"time"
)

func TestDbTxExecutorShutdown(t *testing.T) {
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
			name: "positive_shutdown",
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

			err := test.txExecutor.shutdown(func(ctx context.Context, metrics []model.Metric) error {
				length = len(metrics)
				return nil
			})
			if err != test.expectedErr {
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
