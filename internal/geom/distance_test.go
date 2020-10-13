package geom

import "testing"

func TestChebyshevDistance(t *testing.T) {
	tests := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "positive", p: []float64{1.2, 2.0}, p1: []float64{2.0, 3.0}, expected: 1},
		{name: "positive", p: []float64{10, 2.0}, p1: []float64{5, 3.0}, expected: 5},
		{name: "err", p: []float64{5, 2.0}, p1: []float64{3}, expected: 0},
		{name: "err", p: []float64{2.0}, p1: []float64{3, 4.0}, expected: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ChebyshevDistance(test.p, test.p1)
			if test.name == "positive" {
				if err != nil {
					t.Errorf("the error should not be returned")
				}
				if got != test.expected {
					t.Errorf(
						"the distance obtained does not correspond to the expected distance, got %f, expected %f",
						got, test.expected)
				}
			}
			if test.name == "err" {
				if err == nil {
					t.Errorf("the dimension of the vectors is different, an error must be output %v", ErrDimNotEqual)
				}
			}
		})
	}
}

func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "positive", p: []float64{1.2, 2.0}, p1: []float64{2.0, 3.0}, expected: 1.2806248474865698},
		{name: "positive", p: []float64{10, 2.0}, p1: []float64{5, 3.0}, expected: 5.0990195135927845},
		{name: "err", p: []float64{5, 2.0}, p1: []float64{3}, expected: 0},
		{name: "err", p: []float64{2.0}, p1: []float64{3, 4.0}, expected: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := EuclideanDistance(test.p, test.p1)
			if test.name == "positive" {
				if err != nil {
					t.Errorf("the error should not be returned")
				}
				if got != test.expected {
					t.Errorf(
						"the distance obtained does not correspond to the expected distance, got %f, expected %f",
						got, test.expected)
				}
			}
			if test.name == "err" {
				if err == nil {
					t.Errorf("the dimension of the vectors is different, an error must be output %v", ErrDimNotEqual)
				}
			}
		})
	}
}

func TestManhattanDistance(t *testing.T) {
	tests := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "positive", p: []float64{1.2, 2.0}, p1: []float64{2.0, 3.0}, expected: 1.8},
		{name: "positive", p: []float64{10, 2.0}, p1: []float64{5, 3.0}, expected: 6},
		{name: "err", p: []float64{5, 2.0}, p1: []float64{3}, expected: 0},
		{name: "err", p: []float64{2.0}, p1: []float64{3, 4.0}, expected: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ManhattanDistance(test.p, test.p1)
			if test.name == "positive" {
				if err != nil {
					t.Errorf("the error should not be returned")
				}
				if got != test.expected {
					t.Errorf(
						"the distance obtained does not correspond to the expected distance, got %f, expected %f",
						got, test.expected)
				}
			}
			if test.name == "err" {
				if err == nil {
					t.Errorf("the dimension of the vectors is different, an error must be output %v", ErrDimNotEqual)
				}
			}
		})
	}
}
