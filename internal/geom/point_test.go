package geom

import "testing"

func TestPoint_Dim(t *testing.T) {
	tests := []struct {
		name     string
		p        Point
		expected float64
		idx      int
	}{
		{name: "positive", p: NewPoint([]float64{1, 2, 3}), idx: 0, expected: 1},
		{name: "positive", p: NewPoint([]float64{1, 2, 3}), idx: 1, expected: 2},
		{name: "positive", p: NewPoint([]float64{1, 2, 3}), idx: 2, expected: 3},
		{name: "negative", p: NewPoint([]float64{1, 2, 3}), idx: 3, expected: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "negative" {
				defer func() {
					if err := recover(); err == nil {
						t.Errorf("expected panic")
					}
				}()
				test.p.Dim(test.idx)
			}
			if test.name == "positive" {
				got := test.p.Dim(test.idx)
				if test.expected != got {
					t.Errorf("dimension specified incorrectly, got: %f, expected: %f", got, test.expected)
				}
			}
		})
	}
}
