package predictor

type AlgType string

const (
	AlgTypeLof         AlgType = "LOF"
	AlgIsolationForest AlgType = "ISOLATION_FOREST"
)

type Config struct {
	Type AlgType `envconfig:"SOD_PREDICTOR_TYPE" default:"LOF"`
}

func (c Config) PredictorType() AlgType {
	return c.Type
}

func (c Config) PredictorConfig() Config {
	return c
}
