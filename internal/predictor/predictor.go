package predictor

import (
	"time"
)

type ProvideFn func() (Predictor, error)

type PointsDistanceFn func(vec, vec1 []float64) (float64, error)

type Vector interface {
	Point(idx int) float64
	Dimensions() int
	Points() []float64
}

type DataPoint interface {
	Vector() Vector
	Time() time.Time
}

type Predictor interface {
	Reset()
	Len() int
	Build(data ...DataPoint)
	Append(data ...DataPoint)
	Predict(vec Vector) (*Conclusion, error)
}

type KNNAlg interface {
	Reset()
	Len() int
	Build(data ...DataPoint)
	Remove(data ...DataPoint)
	Append(data ...DataPoint)
	KNN(vec Vector, k int) ([]Vector, error)
}

type Conclusion struct {
	Outlier bool
}
