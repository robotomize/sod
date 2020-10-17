package lof

import (
	"fmt"
	"sod/internal/geom"
	"sod/internal/predictor"
	"sod/internal/predictor/knn/brute"
	"sod/internal/predictor/knn/gbkd"
	"time"
)

const MinKNum = 3

type DistanceFuncType string

const (
	DistanceFuncTypeEuclidean = "EUCLIDEAN"
	DistanceFuncTypeChebyshev = "CHEBYSHEV"
	DistanceFuncTypeManhattan = "MANHATTAN"
)

type AlgType string

const (
	AlgTypeAuto     AlgType = "AUTO"
	AlgTypeBallTree AlgType = "BALL_TREE"
	AlgTypeKDTree   AlgType = "KD_TREE"
	AlgTypeBrute    AlgType = "BRUTE"
)

type Config struct {
	SkipItems      int              `envconfig:"SKIP_ITEMS"`
	KNum           int              `envconfig:"LOF_K_NUM" default:"3"`
	MetricFuncType DistanceFuncType `envconfig:"LOF_DISTANCE_FUNC" default:"EUCLIDEAN"`
	AlgType        AlgType          `envconfig:"LOF_ALG_TYPE" default:"KD_TREE"`
}

func NNFor(a AlgType, maxItems int, maxTime time.Duration, distFn func(vec, vec1 []float64) (float64, error)) (predictor.KNNAlg, error) {
	switch a {
	case AlgTypeBrute:
		return brute.NewBruteAlg(distFn, brute.WithMaxItems(maxItems), brute.WithStorageTime(maxTime)), nil
	case AlgTypeKDTree:
		//return kd.NewKDAlg(distFn, kd.WithStorageTime(maxTime), kd.WithMaxItems(maxItems)), nil
		return gbkd.NewGBkdAlg(distFn, gbkd.WithStorageTime(maxTime), gbkd.WithMaxItems(maxItems)), nil
	default:
		return nil, fmt.Errorf("unable to create alg with alg type %s", a)
	}
}

func DistanceFuncFor(d DistanceFuncType) (func(vec, vec1 []float64) (float64, error), error) {
	switch d {
	case DistanceFuncTypeChebyshev:
		return geom.ChebyshevDistance, nil
	case DistanceFuncTypeEuclidean:
		return geom.EuclideanDistance, nil
	case DistanceFuncTypeManhattan:
		return geom.ManhattanDistance, nil
	default:
		return nil, fmt.Errorf("unknown distance function: %s", d)
	}
}
