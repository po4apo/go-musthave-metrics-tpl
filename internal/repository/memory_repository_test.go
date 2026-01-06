package repository

import (
	"testing"

	model "github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"github.com/stretchr/testify/assert"
)

func i64(v int64) *int64 { return &v }

func f64(v float64) *float64 { return &v }

func getCounterMetric(name string, delta *int64) model.Metrics {
	return model.Metrics{
		Name:  name,
		MType: model.Counter,
		Delta: delta,
	}
}

func getGaugeMetric(name string, value *float64) model.Metrics {
	return model.Metrics{
		Name:  name,
		MType: model.Gauge,
		Value: value,
	}
}

func TestIncreaseValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		want    map[string]model.Metrics
	}{
		{
			name: "Добавление одной метрики",
			metrics: []model.Metrics{
				getCounterMetric("test1", i64(1)),
			},
			want: map[string]model.Metrics{
				"test1": getCounterMetric("test1", i64(1)),
			},
		},
		{
			name: "Добавление двух метрик",
			metrics: []model.Metrics{
				getCounterMetric("test1", i64(1)),
				getCounterMetric("test2", i64(2)),
			},
			want: map[string]model.Metrics{
				"test1": getCounterMetric("test1", i64(1)),
				"test2": getCounterMetric("test2", i64(2)),
			},
		},
		{
			name: "Добавление и увеличение метрики",
			metrics: []model.Metrics{
				getCounterMetric("test1", i64(1)),
				getCounterMetric("test1", i64(2)),
			},
			want: map[string]model.Metrics{
				"test1": getCounterMetric("test1", i64(3)),
			},
		},
		{
			name: "Инициализация с nil и добовление",
			metrics: []model.Metrics{
				getCounterMetric("test1", nil),
				getCounterMetric("test1", i64(2)),
			},
			want: map[string]model.Metrics{
				"test1": getCounterMetric("test1", i64(2)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				ms, _ := NewMemStorage()
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
			initValue:      i64(1),
			secondaryValue: nil,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				ms, _ := NewMemStorage()

				m1 := getCounterMetric("test1", tt.initValue)
				err := ms.IncreaseValue(&m1)
				assert.NoError(t, err)

				m2 := getCounterMetric("test1", tt.secondaryValue)
				err = ms.IncreaseValue(&m2)
				assert.Error(t, err)
				assert.ErrorAs(t, err, &ErrFieldUndefine)
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
				getGaugeMetric("test1", f64(1)),
			},
			want: map[string]model.Metrics{
				"test1": getGaugeMetric("test1", f64(1)),
			},
		},
		{
			name: "Добавление двух разных метрик",
			metrics: []model.Metrics{
				getGaugeMetric("test1", f64(1)),
				getGaugeMetric("test2", f64(2)),
			},
			want: map[string]model.Metrics{
				"test1": getGaugeMetric("test1", f64(1)),
				"test2": getGaugeMetric("test2", f64(2)),
			},
		},
		{
			name: "Добавление и замена метрики",
			metrics: []model.Metrics{
				getGaugeMetric("test1", f64(1)),
				getGaugeMetric("test1", f64(2)),
			},
			want: map[string]model.Metrics{
				"test1": getGaugeMetric("test1", f64(2)),
			},
		},
		{
			name: "Инициализация с nil и добовление",
			metrics: []model.Metrics{
				getGaugeMetric("test1", nil),
				getGaugeMetric("test1", f64(2)),
			},
			want: map[string]model.Metrics{
				"test1": getGaugeMetric("test1", f64(2)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				ms, _ := NewMemStorage()
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
			initValue:      f64(1),
			secondaryValue: nil,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				ms, _ := NewMemStorage()

				m1 := getGaugeMetric("test1", tt.initValue)
				err := ms.ReplaceValue(&m1)
				assert.NoError(t, err)

				m2 := getGaugeMetric("test1", tt.secondaryValue)
				err = ms.ReplaceValue(&m2)
				assert.Error(t, err)
				assert.ErrorAs(t, err, &ErrFieldUndefine)
			},
		)
	}

}

func TestErrUnsupportedTypeReplaceValue(t *testing.T) {
	t.Run(
		"ReplaceValue для MType = Counter",
		func(t *testing.T) {
			ms, _ := NewMemStorage()

			m1 := getCounterMetric("test1", i64(1))
			err := ms.ReplaceValue(&m1)
			assert.Error(t, err)
			assert.ErrorAs(t, err, &ErrUnsupportedType)
		},
	)
}

func TestErrUnsupportedTypeIncreaseValue(t *testing.T) {
	t.Run(
		"IncreaseValue для MType = Gauge",
		func(t *testing.T) {
			ms, _ := NewMemStorage()

			m1 := getGaugeMetric("test1", f64(1))
			err := ms.IncreaseValue(&m1)
			assert.Error(t, err)
			assert.ErrorAs(t, err, &ErrUnsupportedType)
		},
	)
}
