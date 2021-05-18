package model

import (
	"time"

	"github.com/go-sod/sod/internal/metric/model"
	"github.com/google/uuid"
)

func NewAlert(entityID string, metrics []model.Metric) Alert {
	return Alert{
		ID:        uuid.New(),
		EntityID:  entityID,
		Metrics:   metrics,
		CreatedAt: time.Now(),
	}
}

type Alert struct {
	ID        uuid.UUID      `json:"id"`
	EntityID  string         `json:"entityId"`
	Metrics   []model.Metric `json:"metrics"`
	CreatedAt time.Time      `json:"createdAt"`
}
