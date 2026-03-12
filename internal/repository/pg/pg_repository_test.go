package repository

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	repoerrors "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var logger, _ = zap.NewDevelopment()

const databaseDsn = "user=goclient password=12345678 host=localhost port=5432 database=testgo sslmode=disable"

func createTestTable(ctx context.Context, t *testing.T, r *PostgresRepository) {
	t.Helper()
	tableName := r.tables["metrics"] + strconv.Itoa(rand.Int()) + time.Now().Format("20060102_150405")
	err := r.changeTable("metrics", tableName)
	assert.NoError(t, err)
	err = r.createMetricsTable(ctx)
	assert.NoError(t, err)

	dropQuery := "drop table " + tableName + ";"
	t.Cleanup(
		func() {
			_, err := r.db.ExecContext(ctx, dropQuery)
			assert.NoError(t, err)
		},
	)
}

func createTestR(t *testing.T) *PostgresRepository {
	t.Helper()
	ctx := context.Background()
	r, err := NewPostgresStorage(ctx, logger, databaseDsn)
	if err != nil {
		t.Skip("БД недоступна, пропускаем тест:", err)
	}
	createTestTable(ctx, t, r)

	return r
}

func TestConnectDN(t *testing.T) {
	t.Run(
		"Проверка подключения к БД",
		func(t *testing.T) {
			ctx := context.Background()
			_, err := NewPostgresStorage(ctx, logger, databaseDsn)
			if err != nil {
				t.Skip("БД недоступна, пропускаем тест:", err)
			}
		},
	)

}

func TestSetStateFromSliceAndGetAll(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
	}{
		{name: "Добавление двух типов в БД",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			}},
		{name: "Добавление только counter",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(4))),
			}},
		{name: "Добавление только gauge",
			metrics: []model.Metrics{
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(5.6))),
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name,
			func(t *testing.T) {
				r := createTestR(t)

				err := r.SetStateFromSlice(&tt.metrics)
				assert.NoError(t, err)

				actualMetrics, err := r.GetAll()

				assert.NoError(t, err)
				assert.Equal(t, tt.metrics, actualMetrics)
			},
		)
	}

}

func TestString(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		output  string
	}{
		{name: "Вывод двух типов в БД",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			output: `[
  {
    "type": "counter",
    "delta": 1,
    "name": "test_counter_metric"
  },
  {
    "type": "gauge",
    "value": 2.3,
    "name": "test_gauge_metric"
  }
]`,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				r := createTestR(t)

				err := r.SetStateFromSlice(&tt.metrics)
				assert.NoError(t, err)

				s, err := r.String()
				assert.NoError(t, err)
				assert.Equal(t, tt.output, s)

			},
		)
	}
}

func TestGetMetric(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		target  model.Metrics
	}{
		{name: "Получить counter",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1)))},
		{name: "Получить несуществующую counter метрику ",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewCounterMetrics("unexist", model.Ptr(int64(1)))},
		{name: "Получить несуществующую gauge метрику ",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewGaugeMetric("unexist", model.Ptr(float64(1.1)))},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				r := createTestR(t)

				err := r.SetStateFromSlice(&tt.metrics)
				assert.NoError(t, err)

				m, err := r.GetMetric(tt.target.ID)
				if tt.target.Name == "unexist" {
					assert.ErrorIs(t, err, repoerrors.ErrNotFound)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.target, m)
				}

			},
		)
	}
}

func TestIncreaseValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		target  model.Metrics
		delta   int64
	}{
		{name: "Увелечение существующего значения",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(4))),
			delta:  int64(5),
		},
		{name: "Создание нового значения",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewCounterMetrics("test_new_counter_metric", model.Ptr(int64(4))),
			delta:  int64(4),
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				r := createTestR(t)

				err := r.SetStateFromSlice(&tt.metrics)
				assert.NoError(t, err)

				err = r.IncreaseValue(&tt.target)
				assert.NoError(t, err)

				m, err := r.GetMetric(tt.target.ID)
				assert.NoError(t, err)
				assert.Equal(t, int64(*m.Delta), tt.delta)
			},
		)
	}
}

func TestErrorsIncreaseValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		target  model.Metrics
		err     error
	}{
		{name: "Передача нулевого значения",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewCounterMetrics("test_counter_metric", nil),
			err:    repoerrors.ErrFieldUndefine,
		},
		{name: "Передача типа gauge метрики",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			err:    repoerrors.ErrUnsupportedType,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				r := createTestR(t)

				err := r.SetStateFromSlice(&tt.metrics)
				assert.NoError(t, err)

				err = r.IncreaseValue(&tt.target)
				assert.ErrorIs(t, err, tt.err)

				// значения в бд не должны изменится
				actualMetrics, err := r.GetAll()
				assert.NoError(t, err)
				assert.Equal(t, tt.metrics, actualMetrics)
			},
		)
	}
}

func TestReplaceValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		target  model.Metrics
	}{
		{name: "Изменение сущствующей метрики",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(3.4))),
		},
		{name: "Создание новой метрики",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewGaugeMetric("test_new_gauge_metric", model.Ptr(float64(3.4))),
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				r := createTestR(t)

				err := r.SetStateFromSlice(&tt.metrics)
				assert.NoError(t, err)

				err = r.ReplaceValue(&tt.target)
				assert.NoError(t, err)

				m, err := r.GetMetric(tt.target.ID)
				assert.NoError(t, err)
				assert.Equal(t, m, tt.target)
			},
		)
	}
}

func TestReplaceErrorValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []model.Metrics
		target  model.Metrics
		err     error
	}{
		{name: "Передача нулевого значения",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewGaugeMetric("test_gauge_metric", nil),
			err:    repoerrors.ErrFieldUndefine,
		},
		{name: "Передача counter метрики",
			metrics: []model.Metrics{
				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
			},
			target: model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
			err:    repoerrors.ErrUnsupportedType,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				r := createTestR(t)

				err := r.SetStateFromSlice(&tt.metrics)
				assert.NoError(t, err)

				err = r.ReplaceValue(&tt.target)
				assert.ErrorIs(t, err, tt.err)

				// значения в бд не должны изменится
				actualMetrics, err := r.GetAll()
				assert.NoError(t, err)
				assert.Equal(t, tt.metrics, actualMetrics)

			},
		)
	}
}

// func TestEmpty(t *testing.T) {
// 	tests := []struct{
// 		name string
// 		metrics []model.Metrics
// 	}{
// 			{name: "",
// 			metrics : []model.Metrics{
// 				model.NewCounterMetrics("test_counter_metric", model.Ptr(int64(1))),
// 				model.NewGaugeMetric("test_gauge_metric", model.Ptr(float64(2.3))),
// 			},
// 		},

// 	}

// 	for _, tt := range tests {
// 		t.Run(
// 			tt.name,
// 			func(t *testing.T) {
// 				r := createTestR(t)

// 				err := r.SetStateFromSlice(&tt.metrics)
// 				assert.NoError(t, err)

// 			},
// 		)
// 	}
// }
