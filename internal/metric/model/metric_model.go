package model

import (
	"github.com/google/uuid"
	"rango/internal/predictor"
	"rango/pkg/math/vector"
	"time"
)

type Status uint8

const (
	StatusNew Status = iota
	StatusProcessed
)

func NewMetric(entityID string, vec vector.V, createdAt time.Time, extra interface{}) Metric {
	return Metric{
		ID:         uuid.New(),
		EntityID:   entityID,
		Outlier:    false,
		Status:     StatusNew,
		CheckedVec: vec,
		CreatedAt:  createdAt,
		Extra:      extra,
	}
}

var _ predictor.DataPoint = (*Metric)(nil)

type Metric struct {
	ID         uuid.UUID   `json:"id"`
	EntityID   string      `json:"entityId"`
	NormVec    vector.V    `json:"normVec"`
	CheckedVec vector.V    `json:"checkedVec"`
	Outlier    bool        `json:"outlier"`
	Status     Status      `json:"status"`
	CreatedAt  time.Time   `json:"createdAt"`
	Extra      interface{} `json:"extra"`
}

func (m Metric) IsProcessed() bool {
	return m.Status == StatusProcessed
}

func (m Metric) IsNew() bool {
	return m.Status == StatusNew
}

func (m Metric) Vector() predictor.Vector {
	return m.CheckedVec
}

func (m Metric) Time() time.Time {
	return m.CreatedAt
}
