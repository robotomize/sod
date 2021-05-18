package dispatcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-sod/sod/internal/geom"
	metricDb "github.com/go-sod/sod/internal/metric/database"
	"github.com/go-sod/sod/internal/metric/model"
)

func TestRebuildSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		maxItemsStored    int
		expectedKeysErr   error
		expectedCountErr  error
		expectedFetchErr  error
		expectedDeleteErr error
		expectedLen       int
		batch             []model.Metric
	}{
		{
			name:           "positive_rebuild_size",
			maxItemsStored: 3,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 3,
		},
		{
			name:           "positive_rebuild_size",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 1,
		},
		{
			name:           "negative_rebuild_size",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:     1,
			expectedKeysErr: errors.New("test error"),
		},
		{
			name:           "negative_rebuild_size",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:      1,
			expectedCountErr: errors.New("test error"),
		},
		{
			name:           "negative_rebuild_size",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:      1,
			expectedFetchErr: errors.New("test error"),
		},
		{
			name:           "negative_rebuild_size",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:       1,
			expectedDeleteErr: errors.New("test error"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := &dbScheduler{
				opts: dbSchedulerConfig{
					maxItemsStored: test.maxItemsStored,
					deps: pullDependencies{
						fetchKeys: func() ([]string, error) {
							return []string{"test-entity"}, test.expectedKeysErr
						},
						countByEntity: func(s string) (int, error) {
							return len(test.batch), test.expectedCountErr
						},
						fetchMetricsByEntity: func(s string, fn metricDb.FilterFn) ([]model.Metric, error) {
							return test.batch, test.expectedFetchErr
						},
						deleteMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
							test.batch = test.batch[0:test.maxItemsStored]
							return test.expectedDeleteErr
						},
					},
				},
			}

			err := scheduler.rebuildSize()
			if test.expectedKeysErr != nil && err == nil {
				t.Errorf(
					"calling the TestRebuildSize method, the length of data got: %v, expected: %v",
					err,
					test.expectedKeysErr,
				)
			}

			if err == nil && len(test.batch) != test.expectedLen {
				t.Errorf(
					"calling the TestRebuildSize method, the length of data got: %v, expected: %v",
					len(test.batch),
					test.expectedLen,
				)
			}
		})
	}
}

func TestRebuildOutdated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		maxItemsStored    int
		expectedKeysErr   error
		expectedFetchErr  error
		expectedDeleteErr error
		expectedLen       int
		batch             []model.Metric
	}{
		{
			name:           "positive_rebuild_outdated",
			maxItemsStored: 3,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 3,
		},
		{
			name:           "positive_rebuild_outdated",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 1,
		},
		{
			name:           "positive_rebuild_outdated",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:     1,
			expectedKeysErr: errors.New("test error"),
		},
		{
			name:           "positive_rebuild_outdated",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:       1,
			expectedDeleteErr: errors.New("test error"),
		},
		{
			name:           "positive_rebuild_outdated",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:      1,
			expectedFetchErr: errors.New("test error"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := &dbScheduler{
				opts: dbSchedulerConfig{
					maxItemsStored: test.maxItemsStored,
					deps: pullDependencies{
						fetchKeys: func() ([]string, error) {
							return []string{"test-entity"}, test.expectedKeysErr
						},
						fetchMetricsByEntity: func(s string, fn metricDb.FilterFn) ([]model.Metric, error) {
							return test.batch, test.expectedFetchErr
						},
						deleteMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
							test.batch = test.batch[0:test.maxItemsStored]
							return test.expectedDeleteErr
						},
					},
				},
			}
			err := scheduler.rebuildOutdated()
			if test.expectedKeysErr != nil && err == nil {
				t.Errorf(
					"calling the rebuildOutdated method, the length of data got: %v, expected: %v",
					err,
					test.expectedKeysErr,
				)
			}
			if err == nil && len(test.batch) != test.expectedLen {
				t.Errorf(
					"calling the rebuildOutdated method, the length of data got: %v, expected: %v",
					len(test.batch),
					test.expectedLen,
				)
			}
		})
	}
}

func TestProcessOverSizeMetrics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		maxItemsStored    int
		expectedFetchErr  error
		expectedDeleteErr error
		expectedLen       int
		batch             []model.Metric
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
		},
		{
			name:           "positive_process_over_size_metrics",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen: 1,
		},
		{
			name:           "negative_process_over_size_metrics",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:      1,
			expectedFetchErr: errors.New("test error"),
		},
		{
			name:           "negative_process_over_size_metrics",
			maxItemsStored: 1,
			batch: []model.Metric{
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
			},
			expectedLen:       1,
			expectedDeleteErr: errors.New("test error"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := &dbScheduler{
				opts: dbSchedulerConfig{
					maxItemsStored: test.maxItemsStored,
					deps: pullDependencies{
						fetchMetricsByEntity: func(s string, fn metricDb.FilterFn) ([]model.Metric, error) {
							return test.batch, test.expectedFetchErr
						},
						deleteMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
							test.batch = test.batch[0:test.maxItemsStored]
							return test.expectedDeleteErr
						},
					},
				},
			}
			err := scheduler.processOverSizeMetrics("test-metrics")
			if test.expectedFetchErr != nil && err == nil {
				t.Errorf(
					"calling the processOverSizeMetrics method, the length of data got: %v, expected: %v",
					err,
					test.expectedFetchErr,
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

// @TODO add logger test
func TestSchedule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		optsMaxItemsStored int
		optsMaxStorageTime time.Duration
	}{
		{
			name:               "positive_schedule_max_items",
			optsMaxItemsStored: 1,
		},
		{
			name:               "negative_schedule_max_items",
			optsMaxItemsStored: 0,
		},
		{
			name:               "positive_schedule_max_storage_time",
			optsMaxStorageTime: 1 * time.Second,
		},
		{
			name:               "negative_schedule_max_storage_time",
			optsMaxStorageTime: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := &dbScheduler{opts: dbSchedulerConfig{
				maxItemsStored: test.optsMaxItemsStored,
				maxStorageTime: test.optsMaxStorageTime,
				rebuildDBTime:  100 * time.Millisecond,
				deps: pullDependencies{
					fetchKeys: func() ([]string, error) {
						return []string{"test-entity"}, nil
					},
					countByEntity: func(s string) (int, error) {
						return 1, nil
					},
					fetchMetricsByEntity: func(s string, fn metricDb.FilterFn) ([]model.Metric, error) {
						return []model.Metric{}, nil
					},
					deleteMetricsFn: func(ctx context.Context, metrics []model.Metric) error {
						return nil
					},
				},
			}}
			ctx, cancel := context.WithTimeout(context.Background(), scheduler.opts.rebuildDBTime*2)
			defer cancel()

			scheduler.schedule(ctx)
		})
	}
}
