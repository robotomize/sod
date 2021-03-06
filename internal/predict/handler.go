package predict

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-sod/sod/internal/dispatcher"
	"github.com/go-sod/sod/internal/geom"
	"github.com/go-sod/sod/internal/httputil"
	"github.com/go-sod/sod/internal/logging"
	"github.com/go-sod/sod/internal/predictor"
	"golang.org/x/sync/errgroup"
)

const maxBodyBytes = 64 * 1024 * 1024

type DataPoint struct {
	Vec       predictor.Point `json:"vector"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (d DataPoint) Point() predictor.Point {
	return d.Vec
}

func (d DataPoint) Time() time.Time {
	return d.CreatedAt
}

type request struct {
	EntityID string `json:"entity"`
	Data     []struct {
		Vec       []float64   `json:"vector"`
		Extra     interface{} `json:"extra"`
		CreatedAt time.Time   `json:"createdAt"`
	} `json:"data"`
}

type response struct {
	EntityID string `json:"entity"`
	Data     []struct {
		Outlier   bool        `json:"outlier"`
		Vec       []float64   `json:"vector"`
		Extra     interface{} `json:"extra"`
		CreatedAt time.Time   `json:"createdAt"`
	} `json:"data"`
}

func NewHandler(cfg *Config, outlier dispatcher.Predictor) (http.Handler, error) {
	return &handler{
		cfg:     cfg,
		outlier: outlier,
	}, nil
}

type handler struct {
	outlier dispatcher.Predictor
	cfg     *Config
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req request
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.RequestTimeout)
	defer cancel()
	logger := logging.FromContext(ctx)

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		logger.Debugf(`{"error": "method %v is not allowed"}`, r.Method)
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

	if len(req.Data) > h.cfg.MaxDataItemsLen {
		httputil.RespBadRequestErrorf(
			ctx,
			w,
			`{"error": "data items is too large, max allowed len is %d"}`,
			h.cfg.MaxDataItemsLen,
		)
		return
	}
	var respData []struct {
		Outlier   bool        `json:"outlier"`
		Vec       []float64   `json:"vector"`
		Extra     interface{} `json:"extra"`
		CreatedAt time.Time   `json:"createdAt"`
	}
	errGrp := errgroup.Group{}
	mtx := sync.Mutex{}
	for _, dat := range req.Data {
		dat := dat
		errGrp.Go(func() error {
			point := DataPoint{
				Vec:       geom.NewPoint(dat.Vec),
				CreatedAt: dat.CreatedAt,
			}
			result, err := h.outlier.Predict(req.EntityID, point)
			if err != nil {
				return fmt.Errorf("predict error: %w", err)
			}
			mtx.Lock()
			respData = append(respData, struct {
				Outlier   bool        `json:"outlier"`
				Vec       []float64   `json:"vector"`
				Extra     interface{} `json:"extra"`
				CreatedAt time.Time   `json:"createdAt"`
			}{Outlier: result.Outlier, Vec: point.Point().Points(), Extra: dat.Extra, CreatedAt: dat.CreatedAt})
			mtx.Unlock()
			return nil
		})
	}
	if err := errGrp.Wait(); err != nil {
		httputil.RespInternalErrorf(ctx, w, "predict processing error: %v", err)
		return
	}
	resp := response{
		EntityID: req.EntityID,
	}
	resp.Data = respData
	bytes, err := json.Marshal(resp)
	if err != nil {
		httputil.RespInternalErrorf(ctx, w, "failed to encode output json %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "%s", bytes)
}
