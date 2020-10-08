package outlier

import (
	"context"
	"fmt"
	"rango/internal/database"
	"rango/internal/logging"
	metricDb "rango/internal/metric/database"
	"rango/internal/metric/model"
	"sort"
	"time"
)

type dbSchedulerConfig struct {
	maxItemsStored int
	maxStorageTime time.Duration
	rebuildDbTime  time.Duration
}

func newDBScheduler(db *database.DB, config dbSchedulerConfig) *dbScheduler {
	return &dbScheduler{metricDb: metricDb.New(db), opts: config}
}

type dbScheduler struct {
	opts     dbSchedulerConfig
	metricDb *metricDb.DB
}

// @TODO not optimal for memory usage
func (s *dbScheduler) processOutdatedMetrics(entityID string) error {
	metrics, err := s.metricDb.FindByEntity(entityID, func(metric model.Metric) bool {
		return metric.Status == model.StatusProcessed && time.Since(metric.CreatedAt) > s.opts.maxStorageTime
	})
	if err != nil {
		return fmt.Errorf("unable find metrics by entity %s: %v", entityID, err)
	}
	if err := s.metricDb.DeleteMany(context.Background(), metrics); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %v", entityID, err)
	}
	return nil
}

// @TODO not optimal for memory usage
func (s *dbScheduler) processOverSizeMetrics(entityID string) error {
	metrics, err := s.metricDb.FindByEntity(entityID, func(metric model.Metric) bool {
		return metric.Status == model.StatusProcessed
	})
	if err != nil {
		return fmt.Errorf("unable find metrics by entity %s: %v", entityID, err)
	}
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].CreatedAt.UnixNano() < metrics[j].CreatedAt.UnixNano()
	})
	if err := s.metricDb.DeleteMany(context.Background(), metrics[:len(metrics)-s.opts.maxItemsStored]); err != nil {
		return fmt.Errorf("unable delete resizable metrics entity %s: %v", entityID, err)
	}
	return nil
}

func (s *dbScheduler) rebuildOutdated() error {
	keys, err := s.metricDb.Keys()
	if err != nil {
		return fmt.Errorf("unable to fetch metric keys: %v", err)
	}
	for i := range keys {
		if err := s.processOutdatedMetrics(keys[i]); err != nil {
			return fmt.Errorf("unable process metrics: %v", err)
		}
	}
	return nil
}

func (s *dbScheduler) rebuildSize() error {
	keys, err := s.metricDb.Keys()
	if err != nil {
		return fmt.Errorf("unable fetch keys: %v", err)
	}
	for i := range keys {
		length, err := s.metricDb.CountByEntity(keys[i])
		if err != nil {
			return fmt.Errorf("unable count by entity %s: %v", keys[i], err)
		}
		if length > s.opts.maxItemsStored {
			if err := s.processOverSizeMetrics(keys[i]); err != nil {
				return fmt.Errorf("unable process metrics: %v", err)
			}
		}
	}

	return nil
}

func (s *dbScheduler) schedule(ctx context.Context) {
	logger := logging.FromContext(ctx)
	ticker := time.NewTicker(s.opts.rebuildDbTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if s.opts.maxItemsStored > 0 {
				if err := s.rebuildSize(); err != nil {
					logger.Errorf("unable db rebuild size: %v", err)
				}
			}
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
