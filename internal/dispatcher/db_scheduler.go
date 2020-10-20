package dispatcher

import (
	"context"
	"fmt"
	"sod/internal/database"
	"sod/internal/logging"
	metricDb "sod/internal/metric/database"
	"sod/internal/metric/model"
	"sort"
	"time"
)

// Scheduler options
type dbSchedulerConfig struct {
	maxItemsStored int
	maxStorageTime time.Duration
	rebuildDbTime  time.Duration
}

func newDBScheduler(db *database.DB, config dbSchedulerConfig) *dbScheduler {
	return &dbScheduler{metricDb: metricDb.New(db), opts: config}
}

// The scheduler is responsible for deleting old data from the DB
// It can maintain the required amount of data in the DB or delete old data depending on the configuration.
type dbScheduler struct {
	opts     dbSchedulerConfig
	metricDb *metricDb.DB
}

//  abstraction layer for defining a group of metrics
type deleteMetricsFn func(context.Context, []model.Metric) error

// abstraction level for fetching metrics by entity id
type fetchMetricsByEntityFn func(string, metricDb.FilterFn) ([]model.Metric, error)

// @TODO not optimal for memory usage
// processOutdatedMetrics retrieves all metrics for the specified entity, filters, leaving the oldest metrics,
// and performs bulk deletion.
func (s *dbScheduler) processOutdatedMetrics(
	entityID string,
	fetchFn fetchMetricsByEntityFn,
	deleteFn deleteMetricsFn,
) error {
	metrics, err := fetchFn(entityID, func(metric model.Metric) bool {
		// only processed and metrics with a creation date later than specified in the settings
		return metric.Status == model.StatusProcessed && time.Since(metric.CreatedAt) > s.opts.maxStorageTime
	})

	if err != nil {
		return fmt.Errorf("unable find metrics by entity %s: %v", entityID, err)
	}

	if err := deleteFn(context.Background(), metrics); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %v", entityID, err)
	}
	return nil
}

// @TODO not optimal for memory usage
// processOverSizeMetrics retrieves all metrics for the specified entity, sorts by date added,
// and deletes the oldest ones.
func (s *dbScheduler) processOverSizeMetrics(
	entityID string,
	fetchFn fetchMetricsByEntityFn,
	deleteFn deleteMetricsFn,
) error {
	metrics, err := fetchFn(entityID, func(metric model.Metric) bool {
		return metric.Status == model.StatusProcessed // only the processed values
	})

	if err != nil {
		return fmt.Errorf("unable find metrics by entity %s: %v", entityID, err)
	}

	// Sort of a metric. This can be a costly operation for large values.
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].CreatedAt.UnixNano() < metrics[j].CreatedAt.UnixNano()
	})

	// Deleting a slice from the first n sorted metrics
	if err := deleteFn(context.Background(), metrics[:len(metrics)-s.opts.maxItemsStored]); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %v", entityID, err)
	}
	return nil
}

// rebuildOutdated gets all keys of an entity and calls the data processing for each entity
// Checks for outdated metrics for each entity
func (s *dbScheduler) rebuildOutdated(
	keysFn fetchKeysFn,
	fetchFn fetchMetricsByEntityFn,
	deleteFn deleteMetricsFn,
) error {
	keys, err := keysFn()
	if err != nil {
		return fmt.Errorf("unable to fetch metric keys: %v", err)
	}
	for i := range keys {
		if err := s.processOutdatedMetrics(keys[i], fetchFn, deleteFn); err != nil {
			return fmt.Errorf("unable process metrics: %v", err)
		}
	}
	return nil
}

type fetchKeysFn func() ([]string, error)

type countByEntityFn func(string) (int, error)

// rebuildSize gets all keys of an entity and calls the data processing for each entity
// calls a check for the number of elements in the DB for each entity
func (s *dbScheduler) rebuildSize(keysFn fetchKeysFn, countEntityFn countByEntityFn) error {
	keys, err := keysFn()
	if err != nil {
		return fmt.Errorf("unable fetch keys: %v", err)
	}
	for i := range keys {
		// getting the number of metrics for the entity
		length, err := countEntityFn(keys[i])
		if err != nil {
			return fmt.Errorf("unable count by entity %s: %v", keys[i], err)
		}
		// If the number of elements in the entity is greater than the one specified in the configuration,
		//t hen run the processOverSizeMetrics
		if length > s.opts.maxItemsStored {
			if err := s.processOverSizeMetrics(keys[i], s.metricDb.FindByEntity, s.metricDb.DeleteMany); err != nil {
				return fmt.Errorf("unable process metrics: %v", err)
			}
		}
	}

	return nil
}

// Scheduler for running data cleanup functions in the DB
func (s *dbScheduler) schedule(
	ctx context.Context,
	keysFn fetchKeysFn,
	countEntityFn countByEntityFn,
	fetchFn fetchMetricsByEntityFn,
	deleteFn deleteMetricsFn,
) {
	logger := logging.FromContext(ctx)
	ticker := time.NewTicker(s.opts.rebuildDbTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if s.opts.maxItemsStored > 0 {
				if err := s.rebuildSize(keysFn, countEntityFn); err != nil {
					logger.Errorf("unable db rebuild size: %v", err)
				}
			}
			if s.opts.maxStorageTime > 0 {
				if err := s.rebuildOutdated(keysFn, fetchFn, deleteFn); err != nil {
					logger.Errorf("unable db rebuild outdated: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
