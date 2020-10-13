package geom

import (
	"fmt"
	"math"
)

var ErrDimNotEqual = fmt.Errorf("vectors dimension is not equal")

func EuclideanDistance(vec, vec1 []float64) (float64, error) {
	var d float64
	if len(vec) != len(vec1) {
		return 0.0, ErrDimNotEqual
	}

	for i := 0; i < len(vec); i++ {
		d += math.Pow(vec[i]-vec1[i], 2)
	}
	return math.Sqrt(d), nil
}

func ChebyshevDistance(vec, vec1 []float64) (float64, error) {
	var absDistance, distance float64
	if len(vec) != len(vec1) {
		return 0.0, ErrDimNotEqual
	}
	for i := 0; i < len(vec1); i++ {
		absDistance = math.Abs(vec[i] - vec1[i])
		if distance < absDistance {
			distance = absDistance
		}
	}
	return distance, nil
}

func ManhattanDistance(vec, vec1 []float64) (float64, error) {
	var distance float64
	if len(vec) != len(vec1) {
		return 0.0, ErrDimNotEqual
	}
	distance = 0
	for i := 0; i < len(vec); i++ {
		distance += math.Abs(vec[i] - vec1[i])
	}
	return distance, nil
}
