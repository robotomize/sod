package database

import (
	"context"
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"sod/internal/database"
	"sod/internal/metric/model"
	"strings"
)

const (
	entityKeys = "entity:keys:"
	prefix     = "metric:"
)

type FilterFn func(metric model.Metric) bool

func New(db *database.DB) *DB {
	return &DB{sDB: db}
}

type DB struct {
	sDB *database.DB
}

func (db *DB) extractKey(key string) string {
	prefixPos := strings.Index(key, prefix)

	return key[prefixPos+len(prefix):]
}

func (db *DB) Keys() ([]string, error) {
	var bucketKeys []string
	err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(entityKeys))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			bucketKeys = append(bucketKeys, db.extractKey(string(k)))
		}
		return nil
	})

	return bucketKeys, err
}

func (db *DB) Store(_ context.Context, metric model.Metric) error {
	var b *bolt.Bucket
	bytes, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	if err := db.sDB.DB.Update(func(tx *bolt.Tx) error {
		b = tx.Bucket([]byte(prefix + metric.EntityID))
		if b == nil {
			b, err = tx.CreateBucket([]byte(prefix + metric.EntityID))
			if err != nil {
				return fmt.Errorf("create bucket: %w", err)
			}
		}
		if err := b.Put([]byte(metric.ID.String()), bytes); err != nil {
			return fmt.Errorf("put to bucket error: %w", err)
		}
		b = tx.Bucket([]byte(entityKeys))
		if b == nil {
			b, err = tx.CreateBucket([]byte(entityKeys))
			if err != nil {
				return fmt.Errorf("unable create entityies bucket: %w", err)
			}
		}
		if err := b.Put([]byte(prefix+metric.EntityID), []byte{0x0}); err != nil {
			return fmt.Errorf("unable put to entityies bucket: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update transaction error: %v", err)
	}

	return nil
}

func (db *DB) AppendMany(_ context.Context, metrics []model.Metric) error {
	var b *bolt.Bucket
	if err := db.sDB.DB.Batch(func(tx *bolt.Tx) error {
		for _, metric := range metrics {
			b = tx.Bucket([]byte(prefix + metric.EntityID))
			if b == nil {
				entityBucket, err := tx.CreateBucket([]byte(prefix + metric.EntityID))
				if err != nil {
					return fmt.Errorf("create bucket: %w", err)
				}
				b = entityBucket
			}
			bytes, err := json.Marshal(metric)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(metric.ID.String()), bytes); err != nil {
				return fmt.Errorf("put to bucket error: %w", err)
			}
			b = tx.Bucket([]byte(entityKeys))
			if b == nil {
				keysBucket, err := tx.CreateBucket([]byte(entityKeys))
				if err != nil {
					return fmt.Errorf("unable create entityies bucket: %w", err)
				}
				if err := keysBucket.Put([]byte(prefix+metric.EntityID), []byte{0x0}); err != nil {
					return fmt.Errorf("unable put to entityies bucket: %w", err)
				}
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update transaction error: %v", err)
	}

	return nil
}

func (db *DB) DeleteMany(_ context.Context, metrics []model.Metric) error {
	var b *bolt.Bucket
	if err := db.sDB.DB.Batch(func(tx *bolt.Tx) error {
		for _, metric := range metrics {
			b = tx.Bucket([]byte(prefix + metric.EntityID))
			if b == nil {
				continue
			}
			if err := b.Delete([]byte(metric.ID.String())); err != nil {
				return fmt.Errorf("unable delete: %w", err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update transaction error: %v", err)
	}

	return nil
}

func (db *DB) Delete(_ context.Context, metric model.Metric) error {
	var b *bolt.Bucket
	if err := db.sDB.DB.Update(func(tx *bolt.Tx) error {
		b = tx.Bucket([]byte(prefix + metric.EntityID))
		if b == nil {
			return nil
		}

		return b.Delete([]byte(metric.ID.String()))
	}); err != nil {
		return fmt.Errorf("update transaction error: %v", err)
	}

	return nil
}

func (db *DB) FindAll(_ context.Context, filter FilterFn) ([]model.Metric, error) {
	var (
		keys    []string
		metrics []model.Metric
	)
	tx, err := db.sDB.DB.Begin(true)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %v", err)
	}

	defer tx.Rollback()

	b := tx.Bucket([]byte(entityKeys))
	if b == nil {
		b, err = tx.CreateBucket([]byte(entityKeys))
		if err != nil {
			return nil, fmt.Errorf("can not create bucket %s: %w", entityKeys, err)
		}
	}

	c := b.Cursor()

	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		keys = append(keys, string(k))
	}

	for _, key := range keys {
		b := tx.Bucket([]byte(key))
		if b == nil {
			b, err = tx.CreateBucket([]byte(key))
			if err != nil {
				return nil, fmt.Errorf("can not create bucket %s: %w", entityKeys, err)
			}
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var m model.Metric
			if err := json.Unmarshal(v, &m); err != nil {
				return nil, fmt.Errorf("metricCollector unmarshal error, %q", err)
			}
			metrics = append(metrics, m)
		}
	}

	if filter == nil {
		return metrics, nil
	}

	filtered := metrics[:0]
	for _, x := range metrics {
		if filter(x) {
			filtered = append(filtered, x)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %v", err)
	}

	return filtered, nil
}

func (db *DB) CountByEntity(entityID string) (int, error) {
	var length int
	if err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(prefix + entityID))
		if b == nil {
			length = 0
			return nil
		}
		stats := b.Stats()
		length = stats.KeyN
		return nil
	}); err != nil {
		return 0, fmt.Errorf("view transaction error: %v", err)
	}

	return length, nil
}

func (db *DB) FindByEntity(entityID string, filter FilterFn) ([]model.Metric, error) {
	var list []model.Metric
	if err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(prefix + entityID))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var metric model.Metric
			if err := json.Unmarshal(v, &metric); err != nil {
				return fmt.Errorf("json unmarshal error, %q", err)
			}
			if filter == nil || filter(metric) {
				list = append(list, metric)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("view transaction error: %v", err)
	}

	return list, nil
}
