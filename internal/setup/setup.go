package setup

import (
	"context"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"sod/internal/alert"
	"sod/internal/database"
	"sod/internal/logging"
	"sod/internal/outlier"
	"sod/internal/predictor"
	"sod/internal/predictor/lof"
	"sod/internal/scrape"
	"sod/internal/srvenv"
)

const (
	SvcModeScrape  string = "SCRAPE"
	SvcModeCollect string = "COLLECT"
)

type SvcModeConfigProvider interface {
	SvcMode() string
}

type OutlierConfigProvider interface {
	OutlierConfig() *outlier.Config
}

type NotifierConfigProvider interface {
	NotifyConfig() *alert.Config
}

type ScrapeConfigProvider interface {
	ScrapeConfig() *scrape.Config
}

type PredictorConfigProvider interface {
	PredictConfig() *predictor.Config
	PredictType() predictor.AlgType
}

type DatabaseConfigProvider interface {
	DatabaseConfig() *database.Config
}

func Setup(ctx context.Context, config interface{}) (*srvenv.SrvEnv, error) {
	logger := logging.FromContext(ctx)
	var serverEnvOpts []srvenv.Option
	if err := envconfig.Process("", config); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %w", err)
	}

	var (
		db                 *database.DB
		predictorProvideFn predictor.ProvideFn
		notifierProvideFn  alert.ProvideFn
		outlierProvideFn   outlier.ProvideFn
		scrapperProvideFn  scrape.ProvideFn
	)
	if dbConfigProvider, ok := config.(DatabaseConfigProvider); ok {
		logger.Info("Configuring db")
		if err := envconfig.Process("", dbConfigProvider); err != nil {
			return nil, fmt.Errorf("dont process db env: %w", err)
		}
		dbFromEnv, err := database.NewFromEnv(ctx, dbConfigProvider.DatabaseConfig())
		if err != nil {
			return nil, fmt.Errorf("unable to connect to database: %v", err)
		}
		db = dbFromEnv
		serverEnvOpts = append(serverEnvOpts, srvenv.WithDatabase(db))
	}

	if notifyConfigProvider, ok := config.(NotifierConfigProvider); ok {
		logger.Info("Configuring db")

		provideFn, err := ProvideNotifierFor(notifyConfigProvider, db)
		if err != nil {
			return nil, fmt.Errorf("unable create predictor provide function: %v", err)
		}
		notifierProvideFn = provideFn
		serverEnvOpts = append(serverEnvOpts, srvenv.WithNotifier(notifierProvideFn))
	}

	if predictConfigProvider, ok := config.(PredictorConfigProvider); ok {
		logger.Info("Configuring db")
		cfg := predictConfigProvider.PredictConfig()

		if err := envconfig.Process("", cfg); err != nil {
			return nil, fmt.Errorf("dont process db env: %w", err)
		}
		outlierConfigProvider, ok := config.(OutlierConfigProvider)
		if !ok {
			return nil, fmt.Errorf("unable read outlier config")
		}
		provideFn, err := ProvidePredictorFor(cfg, outlierConfigProvider.OutlierConfig())
		if err != nil {
			return nil, fmt.Errorf("unable create predictor provide function: %v", err)
		}
		predictorProvideFn = provideFn
		serverEnvOpts = append(serverEnvOpts, srvenv.WithPredictor(predictorProvideFn))
	}

	if outlierConfigProvider, ok := config.(OutlierConfigProvider); ok {
		logger.Info("Configuring db")
		provideFn, err := ProvideOutlierFor(outlierConfigProvider, predictorProvideFn, db)
		if err != nil {
			return nil, fmt.Errorf("unable create predictor provide function: %v", err)
		}
		outlierProvideFn = provideFn
		serverEnvOpts = append(serverEnvOpts, srvenv.WithOutlier(outlierProvideFn))
	}

	if svcModeConfigProvider, ok := config.(SvcModeConfigProvider); ok && svcModeConfigProvider.SvcMode() == SvcModeScrape {
		if scrapeConfigProvider, ok := config.(ScrapeConfigProvider); ok {
			logger.Info("Configuring db")
			provideFn, err := ProvideScrapperFor(scrapeConfigProvider)
			if err != nil {
				return nil, fmt.Errorf("unable create predictor provide function: %v", err)
			}
			scrapperProvideFn = provideFn
			serverEnvOpts = append(serverEnvOpts, srvenv.WithScrapper(scrapperProvideFn))
		}
	}
	return srvenv.New(serverEnvOpts...), nil
}

func ProvideScrapperFor(provider ScrapeConfigProvider) (scrape.ProvideFn, error) {
	cfg := provider.ScrapeConfig()
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("dont process scrapper env: %w", err)
	}
	return func(outlier outlier.Manager, shutdownCh chan<- error) (scrape.Manager, error) {
		return scrape.New(
			outlier,
			shutdownCh,
			scrape.WithInterval(cfg.Interval),
			scrape.WithMaxConcurrentRequest(cfg.MaxConcurrentRequest),
			scrape.WithTargetUrls(cfg.Targets),
		)
	}, nil
}

func ProvideNotifierFor(provider NotifierConfigProvider, db *database.DB) (alert.ProvideFn, error) {
	cfg := provider.NotifyConfig()
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("dont process notifier env: %w", err)
	}
	return func(shutdownCh chan<- error) (alert.Manager, error) {
		return alert.New(
			db,
			shutdownCh,
			alert.WithMaxConcurrentRequest(cfg.MaxConcurrentRequest),
			alert.WithScrapeInterval(cfg.Interval),
			alert.WithTargets(cfg.Targets),
		)
	}, nil
}

func ProvideOutlierFor(provider OutlierConfigProvider, providePredictFn predictor.ProvideFn, db *database.DB) (outlier.ProvideFn, error) {
	cfg := provider.OutlierConfig()
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("dont process outlier env: %w", err)
	}
	return func(notifier alert.Manager, shutdownCh chan<- error) (outlier.Manager, error) {
		return outlier.New(
			db,
			providePredictFn,
			notifier,
			shutdownCh,
			outlier.WithRebuildDbTime(cfg.RebuildDBTime),
			outlier.WithAllowAppendData(cfg.AllowAppendData),
			outlier.WithAllowAppendOutlier(cfg.AllowAppendOutlier),
			outlier.WithMaxItemsStored(cfg.MaxItemsStored),
			outlier.WithMaxStorageTime(cfg.MaxStorageTime),
			outlier.WithSkipItems(cfg.SkipItems),
			outlier.WithDbFlushSize(cfg.DbFlushSize),
			outlier.WithDbFlushTime(cfg.DbFlushTime),
		)
	}, nil
}

func ProvidePredictorFor(cfg *predictor.Config, outlierCfg *outlier.Config) (predictor.ProvideFn, error) {
	switch cfg.PredictorType() {
	case predictor.AlgTypeLof:
		cfgLof := lof.Config{}
		if err := envconfig.Process("", &cfgLof); err != nil {
			return nil, fmt.Errorf("error loading environment variables: %w", err)
		}
		distFunc, err := lof.DistanceFuncFor(cfgLof.MetricFuncType)
		if err != nil {
			return nil, fmt.Errorf("unable provide distance function: %v", err)
		}
		return func() (predictor.Predictor, error) {
			l, err := lof.New(
				lof.WithSkipItems(cfgLof.SkipItems),
				lof.WithKNum(cfgLof.KNum),
				lof.WithDistance(distFunc),
				lof.WithStorageTime(outlierCfg.MaxStorageTime),
				lof.WithMaxItems(outlierCfg.MaxItemsStored),
				lof.WithAlg(cfgLof.AlgType),
			)
			if err != nil {
				return nil, fmt.Errorf("unable create lof instance: %v", err)
			}
			return l, nil
		}, nil
	default:
		return nil, fmt.Errorf("unknown predictor type: %s", cfg.PredictorType())
	}
}
