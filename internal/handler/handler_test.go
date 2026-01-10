package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
	"github.com/stretchr/testify/assert"
)

// тесты проверяющие тербования первого инкремента
func TestUpdateMetricsHandler(t *testing.T) {
	type Want struct {
		code int
		body string
	}

	tests := []struct{
		name string
		request string
		want Want
	}{{
			name: "Отправка counter",
			request: "/update/counter/test/1",
			want: Want{
				code: http.StatusOK,
				body: "Counter increased!",
			},
		},
		{
			name: "Отправка gauge",
			request: "/update/gauge/test/1",
			want: Want{
				code: http.StatusOK,
				body: "Gauge repalced!",
			},

		},
		{
			name: "Отпрвка запроса без имени",
			request: "/update/gauge//1",
			want: Want{
				code: http.StatusNotFound,
				body: "notfound",
			},

		},
		{
			name: "Отправка с некоретным типом метрики",
			request: "/update/gauge_failed/test/1",
			want: Want{
				code: http.StatusBadRequest,
				body: "\"gauge_failed\" is unknown metric type: badrequest",
			},

		},
		{
			name: "Отправка counter с float значением",
			request: "/update/counter/test/1.0",
			want: Want{
				code: http.StatusBadRequest,
				body: "strconv.ParseInt: parsing \"1.0\": invalid syntax: badrequest",
			},

		},

	}

	for _, tt := range tests {
		t.Run(
			tt.name, 
			func(t *testing.T) {
				repo, _ := repository.NewMemStorage()

				request := httptest.NewRequest(http.MethodPost, tt.request, nil)
				recorder := httptest.NewRecorder()

				handler := UpdateMetricsHandler(&repo)
				handler(recorder, request)

				res := recorder.Result()
				body, err := io.ReadAll(res.Body)
				res.Body.Close()
				assert.NoError(t, err)

				assert.Equal(t, tt.want.code, res.StatusCode)
				assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
				assert.Equal(t, tt.want.body, string(body))
				
				t.Logf("Header: %v", res.Header)
				t.Logf("Status: %v", res.StatusCode)
				t.Logf("Body: %s", body)
			},
		)
	}
}