package alert

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
	"sync"
	"time"

	alertDb "github.com/go-sod/sod/internal/alert/database"
	"github.com/go-sod/sod/internal/alert/model"
	"github.com/go-sod/sod/internal/database"
	"github.com/go-sod/sod/internal/httputil"
	"github.com/go-sod/sod/internal/logging"
	metricModel "github.com/go-sod/sod/internal/metric/model"
	"github.com/go-sod/sod/pkg/rworker"
)

type ProvideFn = func(chan<- error) (Manager, error)

const UserAgent = "SOD/0.1"

type Options struct {
	maxConcurrentRequest int
	requestTimeout       time.Duration
	alertInterval        time.Duration
	targets              Targets
}

type Option func(*manager)

func WithMaxConcurrentRequest(n int) Option {
	return func(o *manager) {
		o.opts.maxConcurrentRequest = n
	}
}

func WithScrapeInterval(t time.Duration) Option {
	return func(o *manager) {
		o.opts.alertInterval = t
	}
}

func WithTargets(m Targets) Option {
	return func(o *manager) {
		o.opts.targets = m
	}
}

type data struct {
	NormalVec  []float64   `json:"norm"`
	OutlierVec []float64   `json:"outlier"`
	CreatedAt  time.Time   `json:"createdAt"`
	Extra      interface{} `json:"extra"`
}

type request struct {
	EntityID string `json:"entityId"`
	Data     []data `json:"data"`
}

func New(db *database.DB, shutdownCh chan<- error, opts ...Option) (*manager, error) {
	m := &manager{
		alertDB:    alertDb.New(db),
		shutdownCh: shutdownCh,
		targets:    Targets{},
		alerts:     map[string][]metricModel.Metric{},
	}
	for _, f := range opts {
		f(m)
	}
	for _, target := range m.targets {
		if _, ok := m.clients[target.EntityID]; !ok {
			client, err := httputil.NewClientFromConfig(target.HTTPConfig, true)
			if err != nil {
				return nil, fmt.Errorf("unable crate client for entity %s: %w", target.EntityID, err)
			}
			m.clients[target.EntityID] = client
		}
	}
	return m, nil
}

type Notifier interface {
	Notify(metrics ...metricModel.Metric)
}

type Manager interface {
	Notifier
	Run(context.Context) error
	Stop()
}

type manager struct {
	mtx        sync.RWMutex
	opts       Options
	alertDB    *alertDb.DB
	shutdownCh chan<- error
	targets    Targets
	clients    map[string]*http.Client
	alerts     map[string][]metricModel.Metric
	cancel     func()
}

func (m *manager) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	go m.notifier(ctx, m.alertDB.Store, m.alertDB.Delete)
	if err := m.bulkLoad(ctx, m.alertDB.Delete, m.alertDB.FindAll); err != nil {
		return fmt.Errorf("can not start alert manager: %w", err)
	}
	return nil
}

func (m *manager) Stop() {
	m.cancel()
}

func (m *manager) Notify(metrics ...metricModel.Metric) {
	m.mtx.Lock()
	for i := range metrics {
		if _, ok := m.alerts[metrics[i].EntityID]; !ok {
			m.alerts[metrics[i].EntityID] = []metricModel.Metric{}
		}
		m.alerts[metrics[i].EntityID] = append(m.alerts[metrics[i].EntityID], metrics[i])
	}
	m.mtx.Unlock()
}

type deleteFn func(context.Context, model.Alert) error

type fetchAllFn func(context.Context, alertDb.FilterFn) ([]model.Alert, error)

func (m *manager) bulkLoad(ctx context.Context, deleteFn deleteFn, fetchFn fetchAllFn) error {
	logger := logging.FromContext(ctx)
	alerts, err := fetchFn(ctx, nil)
	if err != nil {
		logger.Errorf("Error with fetching data from db, %v", err)
	}
	for i := range alerts {
		m.Notify(alerts[i].Metrics...)
		if err := deleteFn(context.Background(), alerts[i]); err != nil {
			return fmt.Errorf("unable delete alert on bulkLoad: %w", err)
		}
	}
	return nil
}

type storeFn func(context.Context, model.Alert) error

func (m *manager) shutdown(fn storeFn) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for entityID, metrics := range m.alerts {
		alert := model.NewAlert(entityID, metrics)
		if err := fn(context.Background(), alert); err != nil {
			return fmt.Errorf("alert shutdown: unable store alert: %w", err)
		}
	}
	return nil
}

type makeRequestFn func() request

func (m *manager) notifier(ctx context.Context, storeFn storeFn, deleteFn deleteFn) {
	logger := logging.FromContext(ctx)
	errCh := make(chan error, 1)
	rateCh := make(chan struct{}, m.opts.maxConcurrentRequest)
	defer close(errCh)
	defer close(rateCh)
	go func() {
		for err := range errCh {
			logger.Errorf("alert error: %v", err)
		}
	}()
	defer func() {
		m.shutdownCh <- m.shutdown(storeFn)
	}()
	wg := sync.WaitGroup{}
	ticker := time.NewTicker(m.opts.alertInterval)
	for {
		select {
		case <-ticker.C:
		OuterLoop:
			for _, target := range m.targets {
				metrics, ok := m.alerts[target.EntityID]
				if !ok || len(metrics) == 0 {
					continue OuterLoop
				}
				rworker.Job(&wg, func() error {
					alertModel := model.NewAlert(metrics[0].EntityID, metrics)
					if err := storeFn(context.Background(), alertModel); err != nil {
						return fmt.Errorf("unable store alert: %w", err)
					}
					if err := m.do(context.Background(), target, func() request {
						outliers := make([]data, len(metrics))
						for i := range metrics {
							outliers[i] = data{
								NormalVec:  metrics[i].NormVec,
								OutlierVec: metrics[i].CheckedVec,
								CreatedAt:  metrics[i].CreatedAt,
								Extra:      metrics[i].Extra,
							}
						}
						return request{
							EntityID: target.EntityID,
							Data:     outliers,
						}
					}); err != nil {
						return fmt.Errorf("alert do request error: %w", err)
					}
					if err := deleteFn(context.Background(), alertModel); err != nil {
						return fmt.Errorf("unable store alert: %w", err)
					}
					m.mtx.Lock()
					m.alerts[target.EntityID] = m.alerts[target.EntityID][:0]
					m.mtx.Unlock()
					return nil
				}, rateCh, errCh)
			}
			wg.Wait()
		case <-ctx.Done():
			return
		}
	}
}

func (m *manager) do(ctx context.Context, target Target, fn makeRequestFn) error {
	ctx, cancel := context.WithTimeout(ctx, m.opts.requestTimeout)
	defer cancel()

	request := fn()
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("unable encode json data: %w", err)
	}

	b := make([]byte, len(body))
	link, err := url.Parse(target.URL)
	if err != nil {
		return fmt.Errorf("url parsing error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", link.String(), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("creating request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Accept-Encoding", "gzip")

	client, ok := m.clients[target.EntityID]
	if !ok {
		return fmt.Errorf("client for entityID %s not defined", target.EntityID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request error: %w", err)
	}

	defer resp.Body.Close()

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("unable create gzip.NewReader: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	_, err = ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response was not 200 OK: %s", body)
	}

	return nil
}
