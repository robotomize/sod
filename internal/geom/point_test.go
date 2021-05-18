package geom

import "testing"

func TestPoint_Dimensions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		p        Point
		expected int
	}{
		{
			name:     "positive",
			p:        NewPoint([]float64{1, 2, 3, 4, 5}),
			expected: 5,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cmp := test.p.Dimensions()
			if cmp != test.expected {
				t.Errorf("the comparison is incorrect got: %v, expected: %v", cmp, test.expected)
			}
		})
	}
}

func TestPoint_Points(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		p        Point
		slice    []float64
		expected bool
	}{
		{name: "positive", p: NewPoint([]float64{1, 2, 3}), slice: []float64{1, 2, 3}, expected: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			slice := test.p.Points()
			for i := range slice {
				if slice[i] != test.slice[i] {
					t.Errorf(
						"conversion to []float64 got: slice[%d] != test.slice[%d], "+
							"expected: slice[%d] == test.slice[%d]", i, i, i, i)
				}
			}
		})
	}
}

func TestPoint_Equal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		p        Point
		p1       Point
		expected bool
	}{
		{
			name:     "positive",
			p:        Point{10, 10},
			p1:       Point{10, 10},
			expected: true,
		},
		{
			name:     "negative",
			p:        Point{10, 10},
			p1:       Point{11, 10},
			expected: false,
		},
	}
	for _, test := range tests {
		if test.p.Equal(test.p1) != test.expected {
			t.Errorf("the comparison of points, got: %v, expected: %v", test.p.Equal(test.p1), test.expected)
		}
	}
}

func TestPoint_SizeEqual(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		p        Point
		p1       Point
		expected bool
	}{
		{
			name:     "positive",
			p:        Point{10, 10},
			p1:       Point{10, 10},
			expected: true,
		},
		{
			name:     "negative",
			p:        Point{10, 10},
			p1:       Point{11},
			expected: false,
		},
	}
	for _, test := range tests {
		if test.p.SizeEqual(test.p1) != test.expected {
			t.Errorf("comparison of the size of the points, got: %v, expected: %v", test.p.SizeEqual(test.p1), test.expected)
		}
	}
}

func TestPoint_Dim(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		p        Point
		expected float64
		idx      int
	}{
		{name: "positive", p: NewPoint([]float64{1, 2, 3}), idx: 0, expected: 1},
		{name: "positive", p: NewPoint([]float64{1, 2, 3}), idx: 1, expected: 2},
		{name: "positive", p: NewPoint([]float64{1, 2, 3}), idx: 2, expected: 3},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.name == "positive" {
				got := test.p.Dim(test.idx)
				if test.expected != got {
					t.Errorf("dimension specified incorrectly, got: %f, slice: %f", got, test.expected)
				}
			}
		})
	}
}
