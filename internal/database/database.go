package database

import (
	"context"
	"fmt"

	"github.com/go-sod/sod/internal/logging"
	bolt "go.etcd.io/bbolt"
)

type DB struct {
	DB *bolt.DB
}

func NewFromEnv(ctx context.Context, config *Config) (*DB, error) {
	logger := logging.FromContext(ctx)
	logger.Infof("creating db connection")

	db, err := bolt.Open(config.FileName, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("creating connection Db: %w", err)
	}

	return &DB{DB: db}, nil
}

func (db *DB) Close(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	logger.Infof("closing DB connection")

	if err := db.DB.Close(); err != nil {
		return fmt.Errorf("error close Db connection: %w", err)
	}

	return nil
}
