package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// makeRequest создает HTTP запрос через chi роутер для тестирования хендлера.
//
// Параметры:
//   - pattern: паттерн маршрута для регистрации в роутере (например, "/update/{type}/{name}/{value}")
//   - repo: указатель на хранилище метрик
//   - method: HTTP метод запроса (например, "POST", "GET")
//   - target: URL запроса для выполнения
//
// Возвращает:
//   - *http.Response: результат выполнения запроса

func newLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger

}

func makeRequest(pattern string, method string, target string, handler http.HandlerFunc, body io.Reader) *http.Response {
	r := chi.NewRouter()
	r.Method(method, pattern, handler)

	request := httptest.NewRequest(method, target, body)
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)
	res := recorder.Result()

	return res
}

// тесты проверяющие тербования первого инкремента
func TestUpdateMetricsHandler(t *testing.T) {
	type Want struct {
		code int
		body string
	}

	tests := []struct {
		name    string
		request string
		want    Want
	}{{
		name:    "Отправка counter",
		request: "/update/counter/test/1",
		want: Want{
			code: http.StatusOK,
			body: "Counter increased!",
		},
	},
		{
			name:    "Отправка gauge",
			request: "/update/gauge/test/1",
			want: Want{
				code: http.StatusOK,
				body: "Gauge repalced!",
			},
		},
		{
			name:    "Отпрвка запроса без имени",
			request: "/update/gauge//1",
			want: Want{
				code: http.StatusNotFound,
				body: "notfound",
			},
		},
		{
			name:    "Отправка с некоретным типом метрики",
			request: "/update/gauge_failed/test/1",
			want: Want{
				code: http.StatusBadRequest,
				body: "\"gauge_failed\" is unknown metric type: badrequest",
			},
		},
		{
			name:    "Отправка counter с float значением",
			request: "/update/counter/test/1.2",
			want: Want{
				code: http.StatusBadRequest,
				body: "strconv.ParseInt: parsing \"1.2\": invalid syntax: badrequest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				logger := newLogger()
				repo, _ := repository.NewMemStorage(logger)
				res := makeRequest(
					"/update/{type}/{name}/{value}",
					http.MethodPost,
					tt.request,
					UpdateMetricsHandler(&repo),
					nil,
				)

				body, err := io.ReadAll(res.Body)
				res.Body.Close()
				assert.NoError(t, err)

				assert.Equal(t, tt.want.code, res.StatusCode)
				assert.Equal(t,
					"text/plain; charset=utf-8",
					res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.body, string(body))
			},
		)
	}
}

// тесты проверяющие тербования первого инкремента
func TestUpdateMetricsWithBodyHandler(t *testing.T) {
	const endpoint = "/update"

	type Want struct {
		code int
		body string
	}

	tests := []struct {
		name string
		body model.Metrics
		want Want
	}{{
		name: "Отправка counter",
		body: model.NewCounterMetrics("test", model.Ptr(int64(1))),
		want: Want{
			code: http.StatusOK,
			body: "Counter increased!",
		},
	},
		{
			name: "Отправка gauge",
			body: model.NewGaugeMetric("test", model.Ptr(float64(1))),
			want: Want{
				code: http.StatusOK,
				body: "Gauge repalced!",
			},
		},
		{
			name: "Отпрвка запроса без имени",
			body: model.NewGaugeMetric("", model.Ptr(float64(1))),
			want: Want{
				code: http.StatusBadRequest,
				body: "validation error: field \"name\" is required. badrequest",
			},
		},
		{
			name: "Отправка с некоретным типом метрики",
			body: model.Metrics{Name: "test", MType: "gauge_failed", Value: model.Ptr(float64(1.2))},
			want: Want{
				code: http.StatusBadRequest,
				body: "\"gauge_failed\" is unknown metric type: badrequest",
			},
		},
		{
			name: "Отправка counter с пустым Delta",
			body: model.Metrics{Name: "test", MType: model.Counter, Value: model.Ptr(float64(1.2))},
			want: Want{
				code: http.StatusBadRequest,
				body: "field \"delte\" is required for counter; badrequest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				logger := newLogger()
				repo, _ := repository.NewMemStorage(logger)

				bBody, err := json.Marshal(tt.body)
				assert.NoError(t, err)

				res := makeRequest(
					endpoint,
					http.MethodPost,
					endpoint,
					UpdateMetricsWithBodyHandler(&repo),
					bytes.NewReader(bBody),
				)

				body, err := io.ReadAll(res.Body)
				res.Body.Close()
				assert.NoError(t, err)

				assert.Equal(t, tt.want.code, res.StatusCode)
				assert.Equal(t,
					"text/plain; charset=utf-8",
					res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.body, string(body))
			},
		)
	}
}

func TestGetMetricHandler(t *testing.T) {
	type Want struct {
		code int
		body string
	}

	tests := []struct {
		name    string
		request string
		want    Want
	}{{
		name:    "Получение counter",
		request: "/value/counter/test_counter",
		want: Want{
			code: http.StatusOK,
			body: "1",
		},
	},
		{
			name:    "Получение gauge",
			request: "/value/gauge/test_gauge",
			want: Want{
				code: http.StatusOK,
				body: "2.1",
			},
		},
		{
			name:    "Получение не существующий метрики gauge",
			request: "/value/gauge/undefined_gauge",
			want: Want{
				code: http.StatusNotFound,
				body: "",
			},
		},
		{
			name:    "Получение не существующий метрики counter",
			request: "/value/counter/undefined_counter",
			want: Want{
				code: http.StatusNotFound,
				body: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				repoState := []model.Metrics{
					model.NewGaugeMetric("test_gauge", model.Ptr(2.1)),
					model.NewCounterMetrics("test_counter", model.Ptr(int64(1))),
				}
				logger := newLogger()
				repo, _ := repository.NewMemStorage(logger)
				repo.SetStateFromSlice(&repoState)

				res := makeRequest(
					"/value/{type}/{name}",
					http.MethodGet,
					tt.request,
					GetMetricHandler(&repo),
					nil,
				)

				body, err := io.ReadAll(res.Body)
				res.Body.Close()
				assert.NoError(t, err)

				assert.Equal(t, tt.want.code, res.StatusCode)
				assert.Equal(t,
					"text/plain; charset=utf-8",
					res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.body, string(body))
			},
		)
	}
}

func TestViewMetrics(t *testing.T) {

	t.Run(
		"Проверка отображения метрик",
		func(t *testing.T) {
			repoState := []model.Metrics{
				model.NewGaugeMetric("test_gauge2", model.Ptr(2.1)),
				model.NewGaugeMetric("test_gauge", model.Ptr(1.1)),
				model.NewCounterMetrics("test_counter", model.Ptr(int64(1))),
			}
			logger := newLogger()
			repo, _ := repository.NewMemStorage(logger)
			repo.SetStateFromSlice(&repoState)

			res := makeRequest(
				"/",
				http.MethodGet,
				"/",
				ViewMetrics(&repo),
				nil,
			)

			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			assert.NoError(t, err)

			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t,
				"text/html; charset=utf-8",
				res.Header.Get("Content-Type"))

			assert.Contains(t, string(body), `<tr><td>test_gauge</td><td>gauge</td><td>1.1</td></tr>`)
			assert.Contains(t, string(body), `<tr><td>test_gauge2</td><td>gauge</td><td>2.1</td></tr>`)
			assert.Contains(t, string(body), `<tr><td>test_counter</td><td>counter</td><td>1</td></tr>`)

			t.Logf("Body: %s", body)
		},
	)
}
