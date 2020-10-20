package dispatcher

//func TestProcessOverSizeMetrics(t *testing.T) {
//	tests := []struct {
//		name        string
//		dbScheduler *dbScheduler
//		expectedErr error
//		expectedLen int
//		batch       []model.Metric
//		size        int
//	}{
//		{
//			name:        "positive_process_over_size_metrics",
//			dbScheduler: &dbScheduler{opts: dbSchedulerConfig{maxItemsStored: 3}},
//			batch: []model.Metric{
//				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
//				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
//				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
//				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
//				model.NewMetric("test-data", geom.Point{1, 1, 1, 1}, time.Now(), "test"),
//			},
//			expectedLen: 3,
//			expectedErr: nil,
//		},
//	}
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			err := test.dbScheduler.processOverSizeMetrics("test-metrics", func(s string, fn metricDb.FilterFn) ([]model.Metric, error) {
//
//			}, func(ctx context.Context, metrics []model.Metric) error {
//
//			})
//			if err != test.expectedErr {
//				t.Errorf()
//			}
//		})
//	}
//}
