package dispatcher

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-sod/sod/internal/logging"
	"github.com/go-sod/sod/internal/metric/model"
)

// Scheduler options
type dbSchedulerConfig struct {
	maxItemsStored int
	maxStorageTime time.Duration
	rebuildDBTime  time.Duration
	deps           pullDependencies
}

// return *dbScheduler with dbSchedulerConfig options
func newDBScheduler(config dbSchedulerConfig) *dbScheduler {
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
func (s *dbScheduler) processOutdatedMetrics(entityID string) error {
	metrics, err := s.opts.deps.fetchMetricsByEntity(entityID, func(metric model.Metric) bool {
		// only processed and metrics with a creation date later than specified in the settings
		return metric.Status == model.StatusProcessed && time.Since(metric.CreatedAt) > s.opts.maxStorageTime
	})
	if err != nil {
		return fmt.Errorf("unable find metrics by entity %s: %w", entityID, err)
	}

	if err := s.opts.deps.deleteMetricsFn(context.Background(), metrics); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %w", entityID, err)
	}
	return nil
}

// @TODO not optimal for memory usage
// processOverSizeMetrics retrieves all metrics for the specified entity, sorts by date added,
// and deletes the oldest ones.
func (s *dbScheduler) processOverSizeMetrics(entityID string) error {
	metrics, err := s.opts.deps.fetchMetricsByEntity(entityID, func(metric model.Metric) bool {
		return metric.Status == model.StatusProcessed // only the processed values
	})
	if err != nil {
		return fmt.Errorf("unable find metrics by entity %s: %w", entityID, err)
	}

	// Sort of a metric. This can be a costly operation for large values.
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].CreatedAt.UnixNano() < metrics[j].CreatedAt.UnixNano()
	})

	// Deleting a slice from the first n sorted metrics
	if err := s.opts.deps.deleteMetricsFn(context.Background(), metrics[:len(metrics)-s.opts.maxItemsStored]); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %w", entityID, err)
	}
	return nil
}

// rebuildOutdated gets all keys of an entity and calls the data processing for each entity
// Checks for outdated metrics for each entity
func (s *dbScheduler) rebuildOutdated() error {
	keys, err := s.opts.deps.fetchKeys()
	if err != nil {
		return fmt.Errorf("unable to fetch metric keys: %w", err)
	}
	for i := range keys {
		if err := s.processOutdatedMetrics(keys[i]); err != nil {
			return fmt.Errorf("unable process metrics: %w", err)
		}
	}
	return nil
}

// rebuildSize gets all keys of an entity and calls the data processing for each entity
// calls a check for the number of elements in the DB for each entity
func (s *dbScheduler) rebuildSize() error {
	keys, err := s.opts.deps.fetchKeys()
	if err != nil {
		return fmt.Errorf("unable fetch keys: %w", err)
	}
	for i := range keys {
		// getting the number of metrics for the entity
		length, err := s.opts.deps.countByEntity(keys[i])
		if err != nil {
			return fmt.Errorf("unable count by entity %s: %w", keys[i], err)
		}
		// If the number of elements in the entity is greater than the one specified in the configuration,
		// then run the processOverSizeMetrics
		if length > s.opts.maxItemsStored {
			if err := s.processOverSizeMetrics(keys[i]); err != nil {
				return fmt.Errorf("unable process metrics: %w", err)
			}
		}
	}

	return nil
}

// Scheduler for running data cleanup functions in the DB
func (s *dbScheduler) schedule(ctx context.Context) {
	logger := logging.FromContext(ctx)
	// determining the time of data verification
	if s.opts.rebuildDBTime == 0 {
		s.opts.rebuildDBTime = 5 * time.Second
	}

	ticker := time.NewTicker(s.opts.rebuildDBTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// if the configuration specifies the maximum size of data to store
			// then you need to check the amount of data stored in the storage.
			if s.opts.maxItemsStored > 0 {
				if err := s.rebuildSize(); err != nil {
					logger.Errorf("unable db rebuild size: %v", err)
				}
			}
			// if the configuration specifies the maximum data storage time
			// then you need to check the time when metrics were created in the storage.
			if s.opts.maxStorageTime > 0 {
				if err := s.rebuildOutdated(); err != nil {
					logger.Errorf("unable db rebuild outdated: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
