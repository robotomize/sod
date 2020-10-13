package collect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"rango/internal/geom"
	"rango/internal/httputil"
	"rango/internal/logging"
	"rango/internal/metric/model"
	"rango/internal/outlier"
	"sort"
	"time"
)

const maxBodyBytes = 64 * 1024 * 1024

type request struct {
	EntityID string `json:"entity"`
	Data     []struct {
		Vec       []float64   `json:"vector"`
		Extra     interface{} `json:"extra"`
		CreatedAt time.Time   `json:"createdAt"`
	} `json:"data"`
}

func NewHandler(cfg *Config, outlier outlier.Collector) (http.Handler, error) {
	s := &handler{
		outlier: outlier,
		cfg:     cfg,
	}
	return s, nil
}

type handler struct {
	outlier outlier.Collector
	cfg     *Config
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req request
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.RequestTimeout)
	defer cancel()
	logger := logging.FromContext(ctx)

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		logger.Debug(fmt.Sprintf(`{"error": "method %v is not allowed"}`, r.Method))
		_, _ = fmt.Fprintf(w, `{"error": "method %v is not allowed"}`, r.Method)
		return
	}

	if t := r.Header.Get("content-type"); len(t) < 16 || t[:16] != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		logger.Debug(fmt.Sprintf(`{"error": "%v"}`, "content-type is not application/json"))
		_, _ = fmt.Fprintf(w, `{"error": "%v"}`, "content-type is not application/json")
		return
	}

	defer r.Body.Close()

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	d := json.NewDecoder(r.Body)
	if err := d.Decode(&req); err != nil {
		httputil.DecodeErr(ctx, w, err)
		return
	}

	defer func() {
		logger.Infof("Collected value for bucket %s", req.EntityID)
	}()
	go func() {
		sort.Slice(req.Data, func(i, j int) bool {
			return req.Data[i].CreatedAt.Before(req.Data[j].CreatedAt)
		})
		for _, dat := range req.Data {
			if err := h.outlier.Collect(
				model.NewMetric(req.EntityID, geom.NewPoint(dat.Vec), dat.CreatedAt, dat.Extra),
			); err != nil {
				logger.Errorf("error sending to collect service: %v", err)
			}
		}
	}()
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"status": "ok"}`)
}
