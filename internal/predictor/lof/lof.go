package lof

import (
	"fmt"
	"math"
	"time"

	"github.com/go-sod/sod/internal/predictor"
)

var _ predictor.Predictor = (*lof)(nil)

const (
	// local predict factor delimiter
	LOF = 1
)

type Option func(*lof)

func WithSkipItems(n int) Option {
	return func(l *lof) {
		l.opts.skipItems = n
	}
}

func WithMaxItems(n int) Option {
	return func(l *lof) {
		l.opts.maxItemsStored = n
	}
}

func WithStorageTime(t time.Duration) Option {
	return func(l *lof) {
		l.opts.maxStorageTime = t
	}
}

func WithKNum(k int) Option {
	return func(l *lof) {
		l.kNum = k
	}
}

func WithDistance(f func(vec, vec1 []float64) (float64, error)) Option {
	return func(l *lof) {
		l.distFunc = f
	}
}

func WithAlg(alg AlgType) Option {
	return func(l *lof) {
		l.opts.algType = alg
	}
}

var defaultOptions = Options{algType: AlgTypeBrute, distanceFuncType: DistanceFuncTypeEuclidean}

type Options struct {
	algType          AlgType
	distanceFuncType DistanceFuncType
	skipItems        int
	maxItemsStored   int
	maxStorageTime   time.Duration
}

func New(opts ...Option) (*lof, error) {
	lof := &lof{
		kNum: MinKNum,
		opts: defaultOptions,
	}
	for _, f := range opts {
		f(lof)
	}
	distFunc, err := DistanceFuncFor(lof.opts.distanceFuncType)
	if err != nil {
		return nil, fmt.Errorf("unable creating lof instance, %w", err)
	}
	lof.distFunc = distFunc
	alg, err := NNFor(lof.opts.algType, lof.opts.maxItemsStored, lof.opts.maxStorageTime, distFunc)
	if err != nil {
		return nil, fmt.Errorf("unable creating lof instance, %w", err)
	}
	lof.alg = alg
	return lof, nil
}

type lof struct {
	opts     Options
	kNum     int
	alg      predictor.KNNAlg
	distFunc func(vec, vec1 []float64) (float64, error)
}

func (l *lof) Len() int {
	return l.alg.Len()
}

func (l *lof) Build(data ...predictor.DataPoint) {
	l.alg.Build(data...)
}

func (l *lof) Append(data ...predictor.DataPoint) {
	l.alg.Append(data...)
}

func (l *lof) Predict(vec predictor.Point) (*predictor.Conclusion, error) {
	if l.Len() == 0 {
		return nil, fmt.Errorf("unable to predict, test vec size 0")
	}
	if l.Len() < l.opts.skipItems {
		return nil, fmt.Errorf("unable to predict, test vec less skip items param")
	}
	result, err := l.predict(vec)
	if err != nil {
		return nil, fmt.Errorf("unable to predict %v, %w", vec, err)
	}
	return result, nil
}

func (l *lof) Reset() {
	l.alg.Reset()
}

func (l *lof) Lof(vec predictor.Point) (float64, error) {
	var lrdSum, avgLrd float64
	nn, err := l.alg.KNN(vec, l.kNum)
	if err != nil {
		return 0.0, fmt.Errorf("unable compute KNN: %w", err)
	}
	for _, y := range nn {
		lrd, err := l.lrd(y)
		if err != nil {
			return 0.0, fmt.Errorf("unable compute lrd: %w", err)
		}
		lrdSum += lrd
	}
	avgLrd = lrdSum / float64(l.kNum)
	lrd, err := l.lrd(vec)
	if err != nil {
		return 0.0, fmt.Errorf("unable compute lrd: %w", err)
	}
	return avgLrd / lrd, nil
}

func (l *lof) DistanceFunc() func(vec, vec1 []float64) (float64, error) {
	return l.distFunc
}

func (l *lof) KNum() int {
	return l.kNum
}

func (l *lof) predict(data predictor.Point) (*predictor.Conclusion, error) {
	if err := l.validateKNum(); err != nil {
		return nil, err
	}
	lof, err := l.Lof(data)
	if err != nil {
		return nil, fmt.Errorf("unable compute lof: %w", err)
	}
	conclusion := &predictor.Conclusion{Outlier: false}
	if lof > LOF {
		conclusion.Outlier = true
	}
	return conclusion, nil
}

func (l *lof) validateKNum() error {
	if l.kNum < MinKNum {
		return fmt.Errorf("the k selected in the config is too small")
	}
	return nil
}

func (l *lof) kDistance(in predictor.Point) (float64, error) {
	vectors, err := l.alg.KNN(in, 3)
	if err != nil {
		return 0.0, fmt.Errorf("unable compute KNN: %w", err)
	}
	return l.distFunc(in.Points(), vectors[0].Points())
}

func (l *lof) reachabilityDist(vec, vec1 predictor.Point) (float64, error) {
	kDistance, err := l.kDistance(vec)
	if err != nil {
		return 0.0, fmt.Errorf("unable compute kDistance: %w", err)
	}
	distance, err := l.distFunc(vec.Points(), vec1.Points())
	if err != nil {
		return 0.0, fmt.Errorf("unable compute distance: %w", err)
	}
	return math.Max(kDistance, distance), nil
}

func (l *lof) lrd(vec predictor.Point) (float64, error) {
	var rSum float64
	nn, err := l.alg.KNN(vec, l.kNum)
	if err != nil {
		return 0.0, fmt.Errorf("unable to compute KNN: %w", err)
	}
	for _, vec1 := range nn {
		rDistance, err := l.reachabilityDist(vec, vec1)
		if err != nil {
			return 0.0, fmt.Errorf("unable to compute reachabilityDist: %w", err)
		}
		rSum += rDistance
	}
	lrd := 1 / (rSum / float64(l.kNum))
	return lrd, nil
}
