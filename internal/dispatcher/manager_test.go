package dispatcher

import (
	"errors"
	"sod/internal/alert"
	"sod/internal/database"
	"sod/internal/geom"
	"sod/internal/metric/model"
	"sod/internal/predictor"
	"sod/internal/predictor/mocks"
	"testing"
	"time"
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
			notifier, _ := alert.New(test.db, test.shutdownCh)

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
			}, notifier, test.shutdownCh)

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

func TestManager_Collect(t *testing.T) {
	tests := []struct {
		name        string
		shutdownCh  chan error
		expectedErr error
		expectedLen int
		closed      bool
		db          *database.DB
		conclusion  *predictor.Conclusion
		metric      model.Metric
	}{
		{
			name:        "positive_collect",
			db:          &database.DB{},
			shutdownCh:  make(chan error, 1),
			conclusion:  &predictor.Conclusion{Outlier: true},
			closed:      false,
			expectedLen: 1,
			metric:      model.NewMetric("test-entity", geom.NewPoint([]float64{1, 1, 1}), time.Now(), "test"),
		},
		{
			name:        "positive_collect",
			db:          &database.DB{},
			shutdownCh:  make(chan error, 1),
			conclusion:  &predictor.Conclusion{Outlier: true},
			expectedLen: 0,
			closed:      true,
			metric:      model.NewMetric("test-entity", geom.NewPoint([]float64{1, 1, 1}), time.Now(), "test"),
			expectedErr: errors.New("test-err"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			notifier, _ := alert.New(test.db, test.shutdownCh)

			pred := &mocks.Predictor{}
			dataPoint := &mocks.DataPoint{}
			point := &mocks.Point{}

			point.On("Points").Return([]float64{1, 1})
			point.On("Dimensions").Return(2)
			point.On("Dim").Return(1)
			dataPoint.On("Point").Return(point)
			pred.On("Collect", point).Return(test.conclusion, nil)

			m, _ := New(test.db, func() (predictor.Predictor, error) {
				return pred, nil
			}, notifier, test.shutdownCh)
			m.closed = test.closed
			err := m.Collect(test.metric)
			if test.expectedErr != nil && err == nil {
				t.Errorf("compute Collect, got: %v, expected: %v", err, test.expectedErr)
			}

			if err == nil && len(m.collectCh) != 1 {
				t.Errorf("compute Collect, got: %v, expected: %v", len(m.collectCh), test.expectedLen)
			}
		})
	}
}
