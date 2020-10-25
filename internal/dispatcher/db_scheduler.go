package dispatcher

import (
	"context"
	"fmt"
	"sod/internal/logging"
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

// return *dbScheduler with dbSchedulerConfig options
func newDbScheduler(config dbSchedulerConfig) *dbScheduler {
	return &dbScheduler{opts: config}
}

// The scheduler is responsible for deleting old data from the DB
// It can maintain the required amount of data in the DB or delete old data depending on the configuration.
type dbScheduler struct {
	opts dbSchedulerConfig
}

// @TODO not optimal for memory usage
// processOutdatedMetrics retrieves all metrics for the specified entity, filters, leaving the oldest metrics,
// and performs bulk deletion.
func (s *dbScheduler) processOutdatedMetrics(entityID string, deps pullDependencies) error {
	metrics, err := deps.fetchMetricsByEntity(entityID, func(metric model.Metric) bool {
		// only processed and metrics with a creation date later than specified in the settings
		return metric.Status == model.StatusProcessed && time.Since(metric.CreatedAt) > s.opts.maxStorageTime
	})

	if err != nil {
		return fmt.Errorf("unable find metrics by entity %s: %v", entityID, err)
	}

	if err := deps.deleteMetricsFn(context.Background(), metrics); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %v", entityID, err)
	}
	return nil
}

// @TODO not optimal for memory usage
// processOverSizeMetrics retrieves all metrics for the specified entity, sorts by date added,
// and deletes the oldest ones.
func (s *dbScheduler) processOverSizeMetrics(entityID string, deps pullDependencies) error {
	metrics, err := deps.fetchMetricsByEntity(entityID, func(metric model.Metric) bool {
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
	if err := deps.deleteMetricsFn(context.Background(), metrics[:len(metrics)-s.opts.maxItemsStored]); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %v", entityID, err)
	}
	return nil
}

// rebuildOutdated gets all keys of an entity and calls the data processing for each entity
// Checks for outdated metrics for each entity
func (s *dbScheduler) rebuildOutdated(deps pullDependencies) error {
	keys, err := deps.fetchKeys()
	if err != nil {
		return fmt.Errorf("unable to fetch metric keys: %v", err)
	}
	for i := range keys {
		if err := s.processOutdatedMetrics(keys[i], deps); err != nil {
			return fmt.Errorf("unable process metrics: %v", err)
		}
	}
	return nil
}

// rebuildSize gets all keys of an entity and calls the data processing for each entity
// calls a check for the number of elements in the DB for each entity
func (s *dbScheduler) rebuildSize(deps pullDependencies) error {
	keys, err := deps.fetchKeys()
	if err != nil {
		return fmt.Errorf("unable fetch keys: %v", err)
	}
	for i := range keys {
		// getting the number of metrics for the entity
		length, err := deps.countByEntity(keys[i])
		if err != nil {
			return fmt.Errorf("unable count by entity %s: %v", keys[i], err)
		}
		// If the number of elements in the entity is greater than the one specified in the configuration,
		//t hen run the processOverSizeMetrics
		if length > s.opts.maxItemsStored {
			if err := s.processOverSizeMetrics(keys[i], deps); err != nil {
				return fmt.Errorf("unable process metrics: %v", err)
			}
		}
	}

	return nil
}

// Scheduler for running data cleanup functions in the DB
func (s *dbScheduler) schedule(ctx context.Context, deps pullDependencies) {
	logger := logging.FromContext(ctx)
	// determining the time of data verification
	if s.opts.rebuildDbTime == 0 {
		s.opts.rebuildDbTime = 5 * time.Second
	}

	ticker := time.NewTicker(s.opts.rebuildDbTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// if the configuration specifies the maximum size of data to store
			// then you need to check the amount of data stored in the storage.
			if s.opts.maxItemsStored > 0 {
				if err := s.rebuildSize(deps); err != nil {
					logger.Errorf("unable db rebuild size: %v", err)
				}
			}
			// if the configuration specifies the maximum data storage time
			// then you need to check the time when metrics were created in the storage.
			if s.opts.maxStorageTime > 0 {
				if err := s.rebuildOutdated(deps); err != nil {
					logger.Errorf("unable db rebuild outdated: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
