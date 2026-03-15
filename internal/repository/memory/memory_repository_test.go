package repository

import (
	"testing"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	repoerrors "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestIncreaseValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		want    map[string]model.Metrics
	}{
		{
			name: "Добавление одной метрики",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test1", model.Ptr(int64(1))),
			},
			want: map[string]model.Metrics{
				"counter_test1": model.NewCounterMetrics("test1", model.Ptr(int64(1))),
			},
		},
		{
			name: "Добавление двух метрик",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test1", model.Ptr(int64(1))),
				model.NewCounterMetrics("test2", model.Ptr(int64(2))),
			},
			want: map[string]model.Metrics{
				"counter_test1": model.NewCounterMetrics("test1", model.Ptr(int64(1))),
				"counter_test2": model.NewCounterMetrics("test2", model.Ptr(int64(2))),
			},
		},
		{
			name: "Добавление и увеличение метрики",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test1", model.Ptr(int64(1))),
				model.NewCounterMetrics("test1", model.Ptr(int64(2))),
			},
			want: map[string]model.Metrics{
				"counter_test1": model.NewCounterMetrics("test1", model.Ptr(int64(3))),
			},
		},
		{
			name: "Инициализация с nil и добовление",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test1", nil),
				model.NewCounterMetrics("test1", model.Ptr(int64(2))),
			},
			want: map[string]model.Metrics{
				"counter_test1": model.NewCounterMetrics("test1", model.Ptr(int64(2))),
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Helper()
				logger, _ := zap.NewDevelopment()
				ms, _ := NewMemStorage(logger)
				for _, metric := range tt.metrics {
					err := ms.IncreaseValue(&metric)
					assert.NoError(t, err)
				}
				assert.Equal(t, tt.want, ms.metrics)
			},
		)
	}

}

func TestErrFieldUndefineIncreaseValue(t *testing.T) {
	tests := []struct {
		name           string
		initValue      *int64
		secondaryValue *int64
	}{
		{
			name:           "Добавление nil к nil",
			initValue:      nil,
			secondaryValue: nil,
		},
		{
			name:           "Добавление nil к числу",
			initValue:      model.Ptr(int64(1)),
			secondaryValue: nil,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Helper()
				logger, _ := zap.NewDevelopment()
				ms, _ := NewMemStorage(logger)

				m1 := model.NewCounterMetrics("test1", tt.initValue)
				err := ms.IncreaseValue(&m1)
				assert.NoError(t, err)

				m2 := model.NewCounterMetrics("test1", tt.secondaryValue)
				err = ms.IncreaseValue(&m2)
				assert.Error(t, err)
				assert.ErrorAs(t, err, &repoerrors.ErrFieldUndefine)
			},
		)
	}
}

func TestReplaceValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		want    map[string]model.Metrics
	}{
		{
			name: "Добавление одной метрики",
			metrics: []model.Metrics{
				model.NewGaugeMetric("test1", model.Ptr(1.0)),
			},
			want: map[string]model.Metrics{
				"gauge_test1": model.NewGaugeMetric("test1", model.Ptr(1.0)),
			},
		},
		{
			name: "Добавление двух разных метрик",
			metrics: []model.Metrics{
				model.NewGaugeMetric("test1", model.Ptr(1.0)),
				model.NewGaugeMetric("test2", model.Ptr(2.0)),
			},
			want: map[string]model.Metrics{
				"gauge_test1": model.NewGaugeMetric("test1", model.Ptr(1.0)),
				"gauge_test2": model.NewGaugeMetric("test2", model.Ptr(2.0)),
			},
		},
		{
			name: "Добавление и замена метрики",
			metrics: []model.Metrics{
				model.NewGaugeMetric("test1", model.Ptr(1.0)),
				model.NewGaugeMetric("test1", model.Ptr(2.0)),
			},
			want: map[string]model.Metrics{
				"gauge_test1": model.NewGaugeMetric("test1", model.Ptr(2.0)),
			},
		},
		{
			name: "Инициализация с nil и добовление",
			metrics: []model.Metrics{
				model.NewGaugeMetric("test1", nil),
				model.NewGaugeMetric("test1", model.Ptr(2.0)),
			},
			want: map[string]model.Metrics{
				"gauge_test1": model.NewGaugeMetric("test1", model.Ptr(2.0)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Helper()
				logger, _ := zap.NewDevelopment()
				ms, _ := NewMemStorage(logger)
				for _, metric := range tt.metrics {
					err := ms.ReplaceValue(&metric)
					assert.NoError(t, err)
				}
				assert.Equal(t, tt.want, ms.metrics)
			},
		)
	}
}

func TestErrFieldUndefineReplaceValue(t *testing.T) {
	tests := []struct {
		name           string
		initValue      *float64
		secondaryValue *float64
	}{
		{
			name:           "Добавление nil к nil",
			initValue:      nil,
			secondaryValue: nil,
		},
		{
			name:           "Добавление nil к числу",
			initValue:      model.Ptr(1.0),
			secondaryValue: nil,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Helper()
				logger, _ := zap.NewDevelopment()
				ms, _ := NewMemStorage(logger)

				m1 := model.NewGaugeMetric("test1", tt.initValue)
				err := ms.ReplaceValue(&m1)
				assert.NoError(t, err)

				m2 := model.NewGaugeMetric("test1", tt.secondaryValue)
				err = ms.ReplaceValue(&m2)
				assert.Error(t, err)
				assert.ErrorAs(t, err, &repoerrors.ErrFieldUndefine)
			},
		)
	}

}

func TestErrUnsupportedTypeReplaceValue(t *testing.T) {
	t.Run(
		"ReplaceValue для MType = Counter",
		func(t *testing.T) {
			t.Helper()
			logger, _ := zap.NewDevelopment()
			ms, _ := NewMemStorage(logger)

			m1 := model.NewCounterMetrics("test1", model.Ptr(int64(1)))
			err := ms.ReplaceValue(&m1)
			assert.Error(t, err)
			assert.ErrorAs(t, err, &repoerrors.ErrUnsupportedType)
		},
	)
}

func TestBatchUpdate(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		want    map[string]model.Metrics
	}{
		{
			name: "Батч с gauge и counter",
			metrics: []model.Metrics{
				model.NewGaugeMetric("g1", model.Ptr(1.5)),
				model.NewCounterMetrics("c1", model.Ptr(int64(10))),
			},
			want: map[string]model.Metrics{
				"gauge_g1":   model.NewGaugeMetric("g1", model.Ptr(1.5)),
				"counter_c1": model.NewCounterMetrics("c1", model.Ptr(int64(10))),
			},
		},
		{
			name: "Counter суммируется внутри батча",
			metrics: []model.Metrics{
				model.NewCounterMetrics("c1", model.Ptr(int64(3))),
				model.NewCounterMetrics("c1", model.Ptr(int64(7))),
			},
			want: map[string]model.Metrics{
				"counter_c1": model.NewCounterMetrics("c1", model.Ptr(int64(10))),
			},
		},
		{
			name: "Gauge перезаписывается последним значением",
			metrics: []model.Metrics{
				model.NewGaugeMetric("g1", model.Ptr(1.0)),
				model.NewGaugeMetric("g1", model.Ptr(9.9)),
			},
			want: map[string]model.Metrics{
				"gauge_g1": model.NewGaugeMetric("g1", model.Ptr(9.9)),
			},
		},
		{
			name:    "Пустой батч",
			metrics: []model.Metrics{},
			want:    map[string]model.Metrics{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			logger, _ := zap.NewDevelopment()
			ms, _ := NewMemStorage(logger)

			err := ms.BatchUpdate(tt.metrics)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, ms.metrics)
		})
	}
}

func TestBatchUpdateUnknownType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ms, _ := NewMemStorage(logger)

	err := ms.BatchUpdate([]model.Metrics{
		{Name: "bad", MType: "unknown"},
	})
	assert.Error(t, err)
}

func TestErrUnsupportedTypeIncreaseValue(t *testing.T) {
	t.Run(
		"IncreaseValue для MType = Gauge",
		func(t *testing.T) {
			t.Helper()
			logger, _ := zap.NewDevelopment()
			ms, _ := NewMemStorage(logger)

			m1 := model.NewGaugeMetric("test1", model.Ptr(1.0))
			err := ms.IncreaseValue(&m1)
			assert.Error(t, err)
			assert.ErrorAs(t, err, &repoerrors.ErrUnsupportedType)
		},
	)
}
