package collect

import (
	"time"
)

type Config struct {
	RequestTimeout time.Duration `envconfig:"SOD_COLLECT_REQUEST_TIMEOUT" default:"60s"`
}
