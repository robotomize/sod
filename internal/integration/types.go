package integration

import "time"

type Request struct {
	EntityID string `json:"entity"`
	Data     []struct {
		Vec       []float64   `json:"vector"`
		Extra     interface{} `json:"extra"`
		CreatedAt time.Time   `json:"createdAt"`
	} `json:"data"`
}
