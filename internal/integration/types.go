package integration

import "time"

type Request struct {
	EntityID string `json:"entityId"`
	Data     []struct {
		Vec       []float64   `json:"vec"`
		Extra     interface{} `json:"extra"`
		CreatedAt time.Time   `json:"createdAt"`
	} `json:"data"`
}
