package dispatcher

import (
	"context"
	"errors"
	"sod/internal/geom"
	metricDb "sod/internal/metric/database"
	"sod/internal/metric/model"
	"testing"
	"time"
)

func TestProcessOverSizeMetrics(t *testing.T) {
	tests := []struct {
		name           string
		maxItemsStored int
		expectedErr    error
		expectedLen    int
		batch          []model.Metric
		size           int
	}{
		{
			name:           "positive_process_over_size_metrics",
			maxItemsStored: 3,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 3,
			expectedErr: nil,
		},
		{
			name:           "negative_process_over_size_metrics",
			maxItemsStored: 3,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 3,
			expectedErr: errors.New("test error"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := &dbScheduler{opts: dbSchedulerConfig{maxItemsStored: test.maxItemsStored}}
			err := scheduler.processOverSizeMetrics(
				"test-metrics",
				func(s string, fn metricDb.FilterFn) ([]model.Metric, error) {
					return test.batch, test.expectedErr
				},
				func(ctx context.Context, metrics []model.Metric) error {
					test.batch = test.batch[0:test.maxItemsStored]
					return test.expectedErr
				},
			)
			if test.expectedErr != nil && err == nil {
				t.Errorf(
					"calling the processOverSizeMetrics method, the length of data got: %v, expected: %v",
					err,
					test.expectedErr,
				)
			}
			if err == nil && len(test.batch) != test.expectedLen {
				t.Errorf(
					"calling the processOverSizeMetrics method, the length of data got: %v, expected: %v",
					len(test.batch),
					test.expectedLen,
				)
			}
		})
	}
}
