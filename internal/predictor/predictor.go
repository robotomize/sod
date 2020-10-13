package predictor

import (
	"time"
)

type ProvideFn func() (Predictor, error)

type Point interface {
	Dim(idx int) float64
	Dimensions() int
	Points() []float64
}

type DataPoint interface {
	Point() Point
	Time() time.Time
}

type Predictor interface {
	Reset()
	Len() int
	Build(data ...DataPoint)
	Append(data ...DataPoint)
	Predict(vec Point) (*Conclusion, error)
}

type KNNAlg interface {
	Reset()
	Len() int
	Build(data ...DataPoint)
	Append(data ...DataPoint)
	KNN(vec Point, k int) ([]Point, error)
}

type Conclusion struct {
	Outlier bool
}
