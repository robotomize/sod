package lof

import (
	"fmt"
	"rango/internal/predictor"
	"rango/internal/predictor/knn/bkd"
	"rango/internal/predictor/knn/brute"
	"rango/pkg/math/vector"
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

func NNFor(a AlgType, maxItems int, maxTime time.Duration, distFn predictor.PointsDistanceFn) (predictor.KNNAlg, error) {
	switch a {
	case AlgTypeBrute:
		return brute.NewBruteAlg(distFn, brute.WithMaxItems(maxItems), brute.WithStorageTime(maxTime)), nil
	case AlgTypeKDTree:
		//return kd.NewKDAlg(distFn, kd.WithStorageTime(maxTime), kd.WithMaxItems(maxItems)), nil
		return bkd.NewBKDAlg(distFn, bkd.WithStorageTime(maxTime), bkd.WithMaxItems(maxItems)), nil
	default:
		return nil, fmt.Errorf("unable to create alg with alg type %s", a)
	}
}

func DistanceFuncFor(d DistanceFuncType) (predictor.PointsDistanceFn, error) {
	switch d {
	case DistanceFuncTypeChebyshev:
		return vector.ChebyshevDistance, nil
	case DistanceFuncTypeEuclidean:
		return vector.EuclideanDistance, nil
	case DistanceFuncTypeManhattan:
		return vector.ManhattanDistance, nil
	default:
		return nil, fmt.Errorf("unknown distance function: %s", d)
	}
}
