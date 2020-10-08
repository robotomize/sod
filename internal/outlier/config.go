package outlier

import (
	"time"
)

type Config struct {
	RebuildDBTime  time.Duration `envconfig:"SOD_OUTLIER_REBUILD_DB_TIME" default:"15s"`
	SkipItems      int           `envconfig:"SOD_OUTLIER_SKIP_ITEMS"`
	MaxItemsStored int           `envconfig:"SOD_OUTLIER_MAX_ITEMS_STORED" default:"1000000"`
	MaxStorageTime time.Duration `envconfig:"SOD_OUTLIER_MAX_STORAGE_TIME" default:"0s"`

	DbFlushSize        int           `envconfig:"SOD_DB_FLUSH_SIZE" default:"10"`
	DbFlushTime        time.Duration `envconfig:"SOD_DB_FLUSH_TIME" default:"5s"`
	AllowAppendData    bool          `envconfig:"SOD_OUTLIER_ALLOW_APPEND_DATA" default:"true"`
	AllowAppendOutlier bool          `envconfig:"SOD_OUTLIER_ALLOW_APPEND_OUTLIER" default:"true"`
}
