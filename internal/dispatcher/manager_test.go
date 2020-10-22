package dispatcher

import (
	"sod/internal/alert"
	"sod/internal/database"
	"sod/internal/predictor"
	"sod/internal/predictor/mocks"
	"testing"
)

func TestManager_Predict(t *testing.T) {
	tests := []struct {
		name        string
		shutdownCh  chan error
		expectedErr error
		expected    bool
		db          *database.DB
		conclusion  *predictor.Conclusion
	}{
		{
			name:       "positive_predict",
			db:         &database.DB{},
			shutdownCh: make(chan error, 1),
			conclusion: &predictor.Conclusion{Outlier: true},
			expected:   true,
		},
		{
			name:       "positive_predict",
			db:         &database.DB{},
			shutdownCh: make(chan error, 1),
			expected:   false,
			conclusion: &predictor.Conclusion{Outlier: false},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			alrt, _ := alert.New(test.db, test.shutdownCh)

			pred := &mocks.Predictor{}
			dataPoint := &mocks.DataPoint{}
			point := &mocks.Point{}
			point.On("Points").Return([]float64{1, 1})
			point.On("Dimensions").Return(2)
			point.On("Dim").Return(1)
			dataPoint.On("Point").Return(point)
			pred.On("Predict", point).Return(test.conclusion, nil)
			m, _ := New(test.db, func() (predictor.Predictor, error) {
				return pred, nil
			}, alrt, test.shutdownCh)

			conclusion, err := m.Predict("test-entity", dataPoint)
			if err != test.expectedErr {
				t.Errorf("compute Predict, got: %v, expected: %v", err, test.expectedErr)
			}
			if conclusion != nil && conclusion.Outlier != test.expected {
				t.Errorf("compute Predict, got: %v, expected: %v", conclusion.Outlier, test.expected)
			}
		})
	}
}
