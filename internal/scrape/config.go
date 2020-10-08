package scrape

import (
	"encoding/json"
	"time"
)

type Config struct {
	Targets              Targets       `envconfig:"SOD_SCRAPE_TARGET_URLS"`
	MaxConcurrentRequest int           `envconfig:"SOD_SCRAPE_MAX_CONCURRENT_REQUEST" default:"64"`
	Interval             time.Duration `envconfig:"SOD_SCRAPE_INTERVAL" default:"1s"`
}

type Targets []Target

func (ts *Targets) Decode(value string) error {
	targets := []Target{}
	if err := json.Unmarshal([]byte(value), &targets); err != nil {
		return err
	}
	*ts = targets
	return nil
}

type Target struct {
	Url      string `json:"url"`
	EntityID string `json:"entityId"`
}
