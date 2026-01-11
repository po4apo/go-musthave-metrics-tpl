package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	model "github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
	"github.com/stretchr/testify/assert"
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
func makeRequest(pattern string, method string, target string, handler http.HandlerFunc) *http.Response {
	r := chi.NewRouter()
	r.Method(method, pattern, handler)

	request := httptest.NewRequest(method, target, nil)
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
				repo, _ := repository.NewMemStorage()
				res := makeRequest(
					"/update/{type}/{name}/{value}",
					http.MethodPost,
					tt.request,
					UpdateMetricsHandler(&repo),
				)

				body, err := io.ReadAll(res.Body)
				res.Body.Close()
				assert.NoError(t, err)

				assert.Equal(t, tt.want.code, res.StatusCode)
				assert.Equal(t,
					"text/plain; charset=utf-8",
					res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.body, string(body))

				t.Logf("Header: %v", res.Header)
				t.Logf("Status: %v", res.StatusCode)
				t.Logf("Body: %s", body)
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
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				repoState := []model.Metrics{
					model.NewGaugeMetric("test_gauge", model.Ptr(2.1)),
					model.NewCounterMetrics("test_counter", model.Ptr(int64(1))),
				}

				repo, _ := repository.NewMemStorage()
				repo.SetStateFromSlice(&repoState)

				res := makeRequest(
					"/value/{type}/{name}",
					http.MethodGet,
					tt.request,
					GetMetricHandler(&repo),
				)

				body, err := io.ReadAll(res.Body)
				res.Body.Close()
				assert.NoError(t, err)

				assert.Equal(t, tt.want.code, res.StatusCode)
				assert.Equal(t,
					"text/plain; charset=utf-8",
					res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.body, string(body))

				t.Logf("Header: %v", res.Header)
				t.Logf("Status: %v", res.StatusCode)
				t.Logf("Body: %s", body)
			},
		)
	}
}
