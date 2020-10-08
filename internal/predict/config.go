package predict

import "time"

type Config struct {
	RequestTimeout  time.Duration `envconfig:"SOD_PREDICT_REQUEST_TIMEOUT" default:"30s"`
	MaxDataItemsLen int           `envconfig:"SOD_PREDICT_MAX_DATA_ITEMS_LEN" default:"10"`
}
