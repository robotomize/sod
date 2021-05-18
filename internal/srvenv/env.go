package srvenv

import (
	"context"

	"github.com/go-sod/sod/internal/alert"
	"github.com/go-sod/sod/internal/database"
	"github.com/go-sod/sod/internal/dispatcher"
	"github.com/go-sod/sod/internal/predictor"
	"github.com/go-sod/sod/internal/scrape"
)

type Option func(*SrvEnv) *SrvEnv

func New(opts ...Option) *SrvEnv {
	env := &SrvEnv{}
	for _, f := range opts {
		env = f(env)
	}

	return env
}

type SrvEnv struct {
	database  *database.DB
	predictor predictor.ProvideFn
	outlier   dispatcher.ProvideFn
	notifier  alert.ProvideFn
	scrapper  scrape.ProvideFn
}

func (s *SrvEnv) ProvideScrapper() scrape.ProvideFn {
	return s.scrapper
}

func (s *SrvEnv) ProvideNotifier() alert.ProvideFn {
	return s.notifier
}

func (s *SrvEnv) ProvideOutlier() dispatcher.ProvideFn {
	return s.outlier
}

func (s *SrvEnv) ProvidePredictor() predictor.ProvideFn {
	return s.predictor
}

func (s *SrvEnv) Database() *database.DB {
	return s.database
}

func WithScrapper(fn scrape.ProvideFn) Option {
	return func(s *SrvEnv) *SrvEnv {
		s.scrapper = fn
		return s
	}
}

func WithNotifier(fn alert.ProvideFn) Option {
	return func(s *SrvEnv) *SrvEnv {
		s.notifier = fn
		return s
	}
}

func WithOutlier(fn dispatcher.ProvideFn) Option {
	return func(s *SrvEnv) *SrvEnv {
		s.outlier = fn
		return s
	}
}

func WithPredictor(fn predictor.ProvideFn) Option {
	return func(s *SrvEnv) *SrvEnv {
		s.predictor = fn
		return s
	}
}

func WithDatabase(db *database.DB) Option {
	return func(s *SrvEnv) *SrvEnv {
		s.database = db
		return s
	}
}

func (s *SrvEnv) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}

	if s.database != nil {
		return s.database.Close(ctx)
	}
	return nil
}
