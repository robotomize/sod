package sod

import (
	"sod/internal/alert"
	"sod/internal/collect"
	"sod/internal/database"
	"sod/internal/outlier"
	"sod/internal/predict"
	"sod/internal/predictor"
	"sod/internal/scrape"
	"sod/internal/setup"
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
	Outlier     outlier.Config
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

func (c Config) OutlierConfig() *outlier.Config {
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
