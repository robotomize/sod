package geom

import (
	"math"
	"sort"
)

type Point []float64

func New(vec []float64) Point {
	return vec
}

func (v Point) Dimensions() int {
	return len(v)
}

func (v Point) Norm() {
	for i := 0; i < len(v); i++ {
		v[i] /= v.Sum()
	}
}

func (v Point) Dim(idx int) float64 {
	return v[idx]
}

func (v Point) Points() []float64 {
	return v
}

func (v Point) Copy() Point {
	var v1 = make(Point, len(v))
	copy(v1, v)
	return v1
}

func (v Point) Scale(value float64) {
	length := len(v)
	for i := 0; i < length; i++ {
		v[i] *= value
	}
}

func (v Point) Magnitude() float64 {
	result := 0.0
	for i := range v {
		result += math.Pow(v[i], 2)
	}
	return math.Sqrt(result)
}

func (v Point) Zero() {
	for i := range v {
		v[i] = 0.0
	}
}

func (v Point) Apply(applyFn func(float64) float64) {
	for i := range v {
		v[i] = applyFn(v[i])
	}
}

func (v Point) Map(applyFn func(float64) float64) Point {
	var v1 = make(Point, len(v))
	for i := range v {
		v1[i] = applyFn(v[i])
	}
	return v1
}

func (v Point) Len() float64 {
	var s float64
	for i := range v {
		s += math.Pow(v[i], 2)
	}
	return math.Sqrt(s)
}

func (v Point) Sum() float64 {
	var s float64
	for i := range v {
		s += v[i]
	}
	return s
}

func (v Point) SizeEqual(vec Point) bool {
	return len(v) == len(vec)
}

func (v Point) Equal(vec Point) bool {
	if len(v) != len(vec) {
		return false
	}
	for i, value := range v {
		if vec[i] != value {
			return false
		}
	}
	return true
}

func (v Point) Max() float64 {
	var max float64
	for i := range v {
		if v[i] > max {
			max = v[i]
		}
	}
	return max
}

func (v Point) Min() float64 {
	var min = math.MaxFloat64
	for i := range v {
		if v[i] < min {
			min = v[i]
		}
	}
	return min
}

func (v Point) Mean() float64 {
	return v.Sum() / float64(len(v))
}

func (v Point) GMean() float64 {
	var p float64
	for i := range v {
		if p == 0 {
			p = v[i]
		} else {
			p *= v[i]
		}
	}
	return math.Pow(p, 1/float64(len(v)))
}

func (v Point) HMean() float64 {
	var p float64
	for i := range v {
		if v[i] <= 0 {
			return math.NaN()
		}
		p += 1 / v[i]
	}
	return float64(len(v)) / p
}

func (v Point) Median() float64 {
	var p float64
	v1 := v.Copy()
	sort.Slice(v1, func(i, j int) bool {
		return v1[i] < v1[j]
	})
	if len(v1)%2 == 0 {
		vc := v1[len(v1)/2-1 : len(v1)/2+1]
		p = vc.Sum() / float64(len(vc))
	} else {
		p = v1[len(v1)/2]
	}

	return p
}

func (v Point) Entropy() float64 {
	var result float64
	v1 := v.Copy()
	v1.Norm()
	for i := range v1 {
		if v[i] != 0 {
			result += v[i] * math.Log(v[i])
		}
	}
	return -result
}
