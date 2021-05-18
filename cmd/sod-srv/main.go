package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/go-sod/sod/internal/collect"
	sod "github.com/go-sod/sod/internal/config"
	"github.com/go-sod/sod/internal/logging"
	"github.com/go-sod/sod/internal/predict"
	"github.com/go-sod/sod/internal/server"
	"github.com/go-sod/sod/internal/setup"
	"github.com/go-sod/sod/internal/shutdown"
)

var (
	version     string
	buildTime   = time.Now().String()
	projectName = "SOD server"
	graffiti    = " _____  ___________ \n/  ___||  _  |  _  \\\n\\ `--. | | | | | | |\n `--. \\| | | | | | |\n/\\__/ /\\ \\_/ / |/ / \n\\____/  \\___/|___/  \n\n"
)

func main() {
	_, _ = fmt.Fprint(os.Stdout, graffiti)
	_, _ = fmt.Fprintf(os.Stdout, "%s: %s, %s\n", projectName, buildTime, version)

	ctx, done := shutdown.New()
	logger := logging.FromContext(ctx)
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
		return fmt.Errorf("dispatcher provider function error: %w", err)
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
		return fmt.Errorf("dispatcher.Run: %w", err)
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

	go func() {
		if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
			cancel()
		}
	}()

	return <-shutdownCh
}
