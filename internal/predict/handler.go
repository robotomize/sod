package predict

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"rango/internal/httputil"
	"rango/internal/logging"
	"rango/internal/outlier"
	"rango/internal/predictor"
	"rango/pkg/math/vector"
	"sync"
	"time"
)

const maxBodyBytes = 64 * 1024 * 1024

type DataPoint struct {
	Vec       predictor.Vector `json:"vec"`
	CreatedAt time.Time        `json:"createdAt"`
}

func (d DataPoint) Vector() predictor.Vector {
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

func NewHandler(cfg *Config, outlier outlier.Predictor) (http.Handler, error) {
	return &handler{
		cfg:     cfg,
		outlier: outlier,
	}, nil
}

type handler struct {
	outlier outlier.Predictor
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

	if len(req.Data) > h.cfg.MaxDataItemsLen {
		httputil.RespBadRequest(ctx, w, `{"error": "data items is too large, max allowed len is %d"}`, h.cfg.MaxDataItemsLen)
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
				Vec:       vector.New(dat.Vec),
				CreatedAt: dat.CreatedAt,
			}
			result, err := h.outlier.Predict(req.EntityID, point)
			if err != nil {
				return fmt.Errorf("predict error: %v", err)
			}
			mtx.Lock()
			respData = append(respData, struct {
				Outlier   bool        `json:"outlier"`
				Vec       []float64   `json:"vector"`
				Extra     interface{} `json:"extra"`
				CreatedAt time.Time   `json:"createdAt"`
			}{Outlier: result.Outlier, Vec: point.Vector().Points(), Extra: dat.Extra, CreatedAt: dat.CreatedAt})
			mtx.Unlock()
			return nil
		})
	}
	if err := errGrp.Wait(); err != nil {
		httputil.RespInternalError(ctx, w, `{"error": "predict processing error, %v"}`, err)
	}
	resp := response{
		EntityID: req.EntityID,
	}
	resp.Data = respData
	bytes, err := json.Marshal(resp)
	if err != nil {
		httputil.RespInternalError(ctx, w, `{"error": "failed to encode output json %v"}`, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "%s", bytes)
}
