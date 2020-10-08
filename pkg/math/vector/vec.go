package vector

import (
	"math"
	"sort"
)

type V []float64

func New(vec []float64) V {
	return vec
}

func (v V) Dimensions() int {
	return len(v)
}

func (v V) Norm() {
	for i := 0; i < len(v); i++ {
		v[i] /= v.Sum()
	}
}

func (v V) Point(idx int) float64 {
	return v[idx]
}

func (v V) Points() []float64 {
	return v
}

func (v V) Copy() V {
	var v1 = make(V, len(v))
	copy(v1, v)
	return v1
}

func (v V) Scale(value float64) {
	length := len(v)
	for i := 0; i < length; i++ {
		v[i] *= value
	}
}

func (v V) Magnitude() float64 {
	result := 0.0
	for i := range v {
		result += math.Pow(v[i], 2)
	}
	return math.Sqrt(result)
}

func (v V) Zero() {
	for i := range v {
		v[i] = 0.0
	}
}

func (v V) Apply(applyFn func(float64) float64) {
	for i := range v {
		v[i] = applyFn(v[i])
	}
}

func (v V) Map(applyFn func(float64) float64) V {
	var v1 = make(V, len(v))
	for i := range v {
		v1[i] = applyFn(v[i])
	}
	return v1
}

func (v V) Len() float64 {
	var s float64
	for i := range v {
		s += math.Pow(v[i], 2)
	}
	return math.Sqrt(s)
}

func (v V) Sum() float64 {
	var s float64
	for i := range v {
		s += v[i]
	}
	return s
}

func (v V) SizeEqual(vec V) bool {
	return len(v) == len(vec)
}

func (v V) Equal(vec V) bool {
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

func (v V) Max() float64 {
	var max float64
	for i := range v {
		if v[i] > max {
			max = v[i]
		}
	}
	return max
}

func (v V) Min() float64 {
	var min = math.MaxFloat64
	for i := range v {
		if v[i] < min {
			min = v[i]
		}
	}
	return min
}

func (v V) Mean() float64 {
	return v.Sum() / float64(len(v))
}

func (v V) GMean() float64 {
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

func (v V) HMean() float64 {
	var p float64
	for i := range v {
		if v[i] <= 0 {
			return math.NaN()
		}
		p += 1 / v[i]
	}
	return float64(len(v)) / p
}

func (v V) Median() float64 {
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

func (v V) Entropy() float64 {
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
