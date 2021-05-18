package scrape

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/go-sod/sod/internal/dispatcher"
	"github.com/go-sod/sod/internal/geom"
	"github.com/go-sod/sod/internal/logging"
	"github.com/go-sod/sod/internal/metric/model"
	"github.com/go-sod/sod/pkg/container/rworker"
)

type response struct {
	EntityID string `json:"entity"`
	Data     []struct {
		Vec       []float64   `json:"vector"`
		Extra     interface{} `json:"extra"`
		CreatedAt time.Time   `json:"createdAt"`
	} `json:"data"`
}

type Manager interface {
	Run(context.Context) error
	Stop()
}

type ProvideFn = func(dispatcher.Manager, chan<- error) (Manager, error)

const UserAgent = "SOD/0.1"

type Options struct {
	maxConcurrentRequest  int
	requestTimeout        time.Duration
	tlsHandshakeTimeout   time.Duration
	responseHeaderTimeout time.Duration
	scrapeInterval        time.Duration
}

type Option func(*manager)

func WithMaxConcurrentRequest(n int) Option {
	return func(o *manager) {
		o.opts.maxConcurrentRequest = n
	}
}

func WithInterval(t time.Duration) Option {
	return func(o *manager) {
		o.opts.scrapeInterval = t
	}
}

func WithTargetUrls(m Targets) Option {
	return func(o *manager) {
		o.targets = m
	}
}

func New(outlier dispatcher.Manager, shutdownCh chan<- error, opts ...Option) (*manager, error) {
	if outlier == nil {
		return nil, fmt.Errorf("dispatcher instance is not defined")
	}
	m := &manager{
		targets:    Targets{},
		shutdownCh: shutdownCh,
		outlier:    outlier,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.client = &http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout:   m.opts.tlsHandshakeTimeout,
			ResponseHeaderTimeout: m.opts.responseHeaderTimeout,
		},
	}
	return m, nil
}

type manager struct {
	opts          Options
	targets       Targets
	outlier       dispatcher.Manager
	client        *http.Client
	shutdownCh    chan<- error
	cancelOutlier func()
	cancel        func()
}

func (s *manager) Stop() {
	s.cancel()
}

func (s *manager) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	c, cancel := context.WithCancel(context.Background())
	s.cancelOutlier = cancel
	if err := s.outlier.Run(c); err != nil {
		return fmt.Errorf("dispatcher.Run: %w", err)
	}
	go func() {
		defer func() {
			s.shutdownCh <- nil
			s.cancelOutlier()
		}()
		ticker := time.NewTicker(s.opts.scrapeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.scrapping(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (s *manager) scrape(url string) (response, error) {
	var response response
	ctx, cancel := context.WithTimeout(context.Background(), s.opts.requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return response, fmt.Errorf("creating request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Accept-Encoding", "gzip")
	resp, err := s.client.Do(req)
	if err != nil {
		return response, fmt.Errorf("sending request error: %w", err)
	}

	defer resp.Body.Close()

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return response, fmt.Errorf("unable create gzip.NewReader: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("response was not 200 OK: %s", body)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil {
		return response, fmt.Errorf("decoding response error: %w", err)
	}

	return response, nil
}

func (s *manager) scrapping(ctx context.Context) {
	wg := sync.WaitGroup{}
	logger := logging.FromContext(ctx)
	errCh := make(chan error, 1)
	defer close(errCh)
	rateCh := make(chan struct{}, s.opts.maxConcurrentRequest)
	defer close(rateCh)
	for err := range errCh {
		logger.Errorf("scrape manager error: %v", err)
	}
OuterLoop:
	for _, link := range s.targets {
		urlData, err := url.Parse(link.URL)
		if err != nil {
			errCh <- fmt.Errorf("url parsing error: %w", err)
			continue OuterLoop
		}
		rworker.Job(&wg, func() error {
			resp, err := s.scrape(urlData.String())
			if err != nil {
				return fmt.Errorf("scrape error: %w", err)
			}
			sort.Slice(resp.Data, func(i, j int) bool {
				return resp.Data[i].CreatedAt.Before(resp.Data[j].CreatedAt)
			})
			for _, dat := range resp.Data {
				if err := s.outlier.Collect(model.NewMetric(resp.EntityID, geom.NewPoint(dat.Vec), dat.CreatedAt, dat.Extra)); err != nil {
					return fmt.Errorf("send to collect error: %w", err)
				}
			}
			return nil
		}, rateCh, errCh)
	}
	wg.Wait()
}
