package database

import (
	"context"
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"sod/internal/alert/model"
	"sod/internal/database"
)

const (
	alertKeys = "alert:keys:"
	prefix    = "alert:"
)

type FilterFn func(alert model.Alert) bool

func New(db *database.DB) *DB {
	return &DB{sDB: db}
}

type DB struct {
	sDB *database.DB
}

func (db *DB) Keys() ([]string, error) {
	var bucketKeys []string
	err := db.sDB.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertKeys))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			bucketKeys = append(bucketKeys, string(k))
		}
		return nil
	})
	return bucketKeys, err
}

func (db *DB) Store(_ context.Context, alert model.Alert) error {
	var b *bolt.Bucket
	bytes, err := json.Marshal(alert)
	if err != nil {
		return err
	}
	if err := db.sDB.DB.Update(func(tx *bolt.Tx) error {
		b = tx.Bucket([]byte(prefix + alert.EntityID))
		if b == nil {
			b, err = tx.CreateBucket([]byte("alert:" + alert.EntityID))
			if err != nil {
				return fmt.Errorf("create bucket: %w", err)
			}
		}
		if err := b.Put([]byte(alert.ID.String()), bytes); err != nil {
			return fmt.Errorf("put to bucket error: %w", err)
		}
		b = tx.Bucket([]byte(alertKeys))
		if b == nil {
			b, err = tx.CreateBucket([]byte(alertKeys))
			if err != nil {
				return fmt.Errorf("unable create entityies bucket: %w", err)
			}
		}
		if err := b.Put([]byte(prefix+alert.EntityID), []byte{0x0}); err != nil {
			return fmt.Errorf("unable put to entityies bucket: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update transaction error: %v", err)
	}
	return nil
}

func (db *DB) Delete(_ context.Context, alert model.Alert) error {
	var b *bolt.Bucket
	if err := db.sDB.DB.Update(func(tx *bolt.Tx) error {
		b = tx.Bucket([]byte(prefix + alert.EntityID))
		if b == nil {
			return nil
		}

		return b.Delete([]byte(alert.ID.String()))
	}); err != nil {
		return fmt.Errorf("update transaction error: %v", err)
	}
	return nil
}

func (db *DB) FindAll(_ context.Context, filter FilterFn) ([]model.Alert, error) {
	var (
		keys    []string
		metrics []model.Alert
	)
	tx, err := db.sDB.DB.Begin(true)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %v", err)
	}

	defer tx.Rollback() //nolint:errcheck

	b := tx.Bucket([]byte(alertKeys))
	if b == nil {
		b, err = tx.CreateBucket([]byte(alertKeys))
		if err != nil {
			return nil, fmt.Errorf("can not create bucket %s: %w", alertKeys, err)
		}
	}

	c := b.Cursor()

	for k, v := c.First(); k != nil; k, v = c.Next() {
		keys = append(keys, string(v))
	}

	for _, key := range keys {
		b := tx.Bucket([]byte(key))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var m model.Alert
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
