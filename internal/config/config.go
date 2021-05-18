package sod

import (
	"github.com/go-sod/sod/internal/alert"
	"github.com/go-sod/sod/internal/collect"
	"github.com/go-sod/sod/internal/database"
	"github.com/go-sod/sod/internal/dispatcher"
	"github.com/go-sod/sod/internal/predict"
	"github.com/go-sod/sod/internal/predictor"
	"github.com/go-sod/sod/internal/scrape"
	"github.com/go-sod/sod/internal/setup"
)

var (
	_ setup.PredictorConfigProvider = (*Config)(nil)
	_ setup.DatabaseConfigProvider  = (*Config)(nil)
	_ setup.NotifierConfigProvider  = (*Config)(nil)
	_ setup.ScrapeConfigProvider    = (*Config)(nil)
	_ setup.PredictorConfigProvider = (*Config)(nil)
)

const (
	SvcModeTypeCollect = "COLLECT"
	SvcModeTypeScrape  = "SCRAPE"
)

type Config struct {
	SvcModeType string `envconfig:"SOD_SVC_MODE" default:"COLLECT"`
	SrvAddr     string `envconfig:"SOD_ADDR" default:":8787"`
	Outlier     dispatcher.Config
	Collect     collect.Config
	Predict     predict.Config
	Database    database.Config
	Scrape      scrape.Config
	Predictor   predictor.Config
	Alert       alert.Config
}

func (c Config) SvcMode() string {
	return c.SvcModeType
}

func (c Config) OutlierConfig() *dispatcher.Config {
	return &c.Outlier
}

func (c Config) NotifyConfig() *alert.Config {
	return &c.Alert
}

func (c Config) ScrapeConfig() *scrape.Config {
	return &c.Scrape
}

func (c Config) DatabaseConfig() *database.Config {
	return &c.Database
}

func (c Config) PredictType() predictor.AlgType {
	return c.Predictor.Type
}

func (c Config) PredictConfig() *predictor.Config {
	return &c.Predictor
}
