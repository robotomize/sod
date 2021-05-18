package geom

import "testing"

func TestChebyshevDistance(t *testing.T) {
	t.Parallel()
	positive := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "positive", p: []float64{1.2, 2.0}, p1: []float64{2.0, 3.0}, expected: 1},
		{name: "positive", p: []float64{10, 2.0}, p1: []float64{5, 3.0}, expected: 5},
	}

	negative := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "negative", p: []float64{5, 2.0}, p1: []float64{3}, expected: 0},
		{name: "negative", p: []float64{2.0}, p1: []float64{3, 4.0}, expected: 0},
	}

	for _, test := range positive {
		test := test
		t.Run("positive", func(t *testing.T) {
			t.Parallel()
			got, _ := ChebyshevDistance(test.p, test.p1)
			if test.expected != got {
				t.Errorf(
					"the distance obtained does not correspond to the slice distance, got %f, slice %f",
					got, test.expected)
			}
		})
	}

	for _, test := range negative {
		test := test
		t.Run("negative", func(t *testing.T) {
			t.Parallel()
			_, err := ChebyshevDistance(test.p, test.p1)
			if err == nil {
				t.Errorf("the dimension of the vectors is different, an error must be output %v", ErrDimNotEqual)
			}
		})
	}
}

func TestEuclideanDistance(t *testing.T) {
	t.Parallel()
	positive := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "positive", p: []float64{1.2, 2.0}, p1: []float64{2.0, 3.0}, expected: 1.2806248474865698},
		{name: "positive", p: []float64{10, 2.0}, p1: []float64{5, 3.0}, expected: 5.0990195135927845},
	}

	negative := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "negative", p: []float64{5, 2.0}, p1: []float64{3}, expected: 0},
		{name: "negative", p: []float64{2.0}, p1: []float64{3, 4.0}, expected: 0},
	}

	for _, test := range positive {
		test := test
		t.Run("positive", func(t *testing.T) {
			t.Parallel()
			got, _ := EuclideanDistance(test.p, test.p1)
			if test.expected != got {
				t.Errorf(
					"the distance obtained does not correspond to the slice distance, got %f, slice %f",
					got, test.expected)
			}
		})
	}

	for _, test := range negative {
		test := test
		t.Run("negative", func(t *testing.T) {
			t.Parallel()
			_, err := EuclideanDistance(test.p, test.p1)
			if err == nil {
				t.Errorf("the dimension of the vectors is different, an error must be output %v", ErrDimNotEqual)
			}
		})
	}
}

func TestManhattanDistance(t *testing.T) {
	t.Parallel()
	positive := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "positive", p: []float64{1.2, 2.0}, p1: []float64{2.0, 3.0}, expected: 1.800000},
		{name: "positive", p: []float64{10, 2.0}, p1: []float64{5, 3.0}, expected: 6.000000},
	}

	negative := []struct {
		name     string
		p        []float64
		p1       []float64
		expected float64
	}{
		{name: "negative", p: []float64{5, 2.0}, p1: []float64{3}, expected: 0},
		{name: "negative", p: []float64{2.0}, p1: []float64{3, 4.0}, expected: 0},
	}
	for _, test := range positive {
		test := test
		t.Run("positive", func(t *testing.T) {
			t.Parallel()
			got, _ := ManhattanDistance(test.p, test.p1)
			if test.expected != got {
				t.Errorf(
					"the distance obtained does not correspond to the slice distance, got %f, slice %f",
					got, test.expected)
			}
		})
	}

	for _, test := range negative {
		test := test
		t.Run("negative", func(t *testing.T) {
			t.Parallel()
			_, err := ManhattanDistance(test.p, test.p1)
			if err == nil {
				t.Errorf("the dimension of the vectors is different, an error must be output %v", ErrDimNotEqual)
			}
		})
	}
}
