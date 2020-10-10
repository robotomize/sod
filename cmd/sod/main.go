package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"rango/internal/collect"
	"rango/internal/logging"
	"rango/internal/predict"
	"rango/internal/server"
	"rango/internal/setup"
	"rango/internal/shutdown"
	"rango/internal/sod"
)

func main() {
	ctx, done := shutdown.New()
	logger := logging.FromContext(ctx)
	go http.ListenAndServe("0.0.0.0:8080", nil)
	if err := run(ctx, done); err != nil {
		logger.Fatal(err)
	}
	defer done()
}

func run(ctx context.Context, cancel func()) error {
	var (
		shutdownCh    chan error
		shutdownCount = 2
	)
	config := sod.Config{}
	env, err := setup.Setup(ctx, &config)
	if err != nil {
		return fmt.Errorf("setup.Setup: %w", err)
	}

	if config.SvcModeType == sod.SvcModeTypeScrape {
		shutdownCount++
	}

	shutdownCh = make(chan error, shutdownCount)
	notifier, err := env.ProvideNotifier()(shutdownCh)
	if err != nil {
		return fmt.Errorf("notifier provider function error: %w", err)
	}
	outlier, err := env.ProvideOutlier()(notifier, shutdownCh)
	if err != nil {
		return fmt.Errorf("outlier provider function error: %w", err)
	}

	if config.SvcModeType == sod.SvcModeTypeScrape {
		scrapper, err := env.ProvideScrapper()(outlier, shutdownCh)
		if err != nil {
			return fmt.Errorf("scrapperCaller: %w", err)
		}
		if err := scrapper.Run(ctx); err != nil {
			return fmt.Errorf("scrapperRun: %w", err)
		}
	} else if err := outlier.Run(ctx); err != nil {
		return fmt.Errorf("outlier.Run: %w", err)
	}

	srv, err := server.New(config.SrvAddr)
	if err != nil {
		return fmt.Errorf("sever.New: %w", err)
	}

	mux := http.NewServeMux()

	predictHandler, err := predict.NewHandler(&config.Predict, outlier)
	if err != nil {
		return fmt.Errorf("collect.NewHandler: %w", err)
	}

	mux.Handle("/predict", predictHandler)
	mux.Handle("/health", server.HandleHealth(ctx))

	if config.SvcModeType == sod.SvcModeTypeCollect {
		collectHandler, err := collect.NewHandler(&config.Collect, outlier)
		if err != nil {
			return fmt.Errorf("collect.NewHandler: %w", err)
		}
		mux.Handle("/collect", collectHandler)
	}

	go func() {
		if err := srv.ServeHTTPHandler(ctx, mux); err != nil {
			cancel()
		}
	}()

	return <-shutdownCh
}
