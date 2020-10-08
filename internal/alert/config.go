package alert

import (
	"encoding/json"
	"rango/internal/httputil"
	"time"
)

type Config struct {
	AllowAlerts          bool          `envconfig:"SOD_ALLOW_ALERTS" default:"true"`
	Targets              Targets       `envconfig:"SOD_ALERT_TARGETS"`
	Interval             time.Duration `envconfig:"SOD_ALERT_INTERVAL" default:"5s"`
	MaxConcurrentRequest int           `envconfig:"SOD_ALERT_MAX_CONCURRENT_REQUEST" default:"64"`
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
	Url        string                    `json:"url"`
	EntityID   string                    `json:"entityId"`
	HTTPConfig httputil.HTTPClientConfig `json:"httpConfig"`
}
