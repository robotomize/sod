package outlier

import (
	"time"
)

type Config struct {
	// Timer for performing data cleaning operations in the DB
	RebuildDBTime time.Duration `envconfig:"SOD_OUTLIER_REBUILD_DB_TIME" default:"15s"`
	// Skipping the first n metrics that are not passed through predictor, accumulating the dataset
	SkipItems int `envconfig:"SOD_OUTLIER_SKIP_ITEMS"`
	// maximum number of elements in the DB for each entity
	MaxItemsStored int `envconfig:"SOD_OUTLIER_MAX_ITEMS_STORED" default:"1000000"`
	// maximum retention period for elements in the DB for each entity
	MaxStorageTime time.Duration `envconfig:"SOD_OUTLIER_MAX_STORAGE_TIME" default:"0s"`
	//Critical buffer size in dbTxExecutor DP where data is flushed to disk
	DbFlushSize int `envconfig:"SOD_DB_FLUSH_SIZE" default:"10"`
	// Critical time of life in dbTxExecutor buffer in which data to be flushed to disk
	DbFlushTime time.Duration `envconfig:"SOD_DB_FLUSH_TIME" default:"5s"`
	//  Allow adding data to the dataset
	AllowAppendData bool `envconfig:"SOD_OUTLIER_ALLOW_APPEND_DATA" default:"true"`
	// Allow adding outliers to the dataset
	AllowAppendOutlier bool `envconfig:"SOD_OUTLIER_ALLOW_APPEND_OUTLIER" default:"true"`
}
