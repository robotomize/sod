package geom

import (
	"fmt"
	"math"
)

var ErrDimNotEqual = fmt.Errorf("points dimension is not equal")

func EuclideanDistance(p, p1 []float64) (float64, error) {
	var d float64
	if len(p) != len(p1) {
		return 0.0, ErrDimNotEqual
	}

	for i := 0; i < len(p); i++ {
		d += math.Pow(p[i]-p1[i], 2)
	}
	return math.Sqrt(d), nil
}

func ChebyshevDistance(p, p1 []float64) (float64, error) {
	var absDistance, distance float64
	if len(p) != len(p1) {
		return 0.0, ErrDimNotEqual
	}
	for i := 0; i < len(p1); i++ {
		absDistance = math.Abs(p[i] - p1[i])
		if distance < absDistance {
			distance = absDistance
		}
	}
	return distance, nil
}

func ManhattanDistance(p, p1 []float64) (float64, error) {
	var distance float64
	if len(p) != len(p1) {
		return 0.0, ErrDimNotEqual
	}
	distance = 0
	for i := 0; i < len(p); i++ {
		distance += math.Abs(p[i] - p1[i])
	}
	return distance, nil
}
