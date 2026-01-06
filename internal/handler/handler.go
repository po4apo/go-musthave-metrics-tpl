package handler

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	model "github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	rep "github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
)

var (
	ErrBadRequest = errors.New("badrequest")
)

func validateUpdateMetrics(url *url.URL, result *model.Metrics) (error) {
	COUNT_PARAMS := 4

	values := strings.Split(strings.Trim(url.Path, "/"), "/")
	log.Printf("Данные запроса: %v, len: %d", values, len(values))

	if len(values) != COUNT_PARAMS {
		return ErrBadRequest
	}
	op := values[0]
	
	if values[1] != model.Counter && values[1] != model.Gauge {
		return ErrBadRequest
	}

	result.MType = values[1]	
	result.Name = values[2]

	if result.MType == model.Counter {
		if delta, err := strconv.ParseInt(values[3], 10, 64); err == nil {
			result.Delta = &delta
		} else {
			return ErrBadRequest
		}
	} else {
		if value, err := strconv.ParseFloat(values[3], 64); err == nil {
			result.Value = &value
		} else {
			return ErrBadRequest
		}
	}
	log.Printf("Итоговая модель: %v, Операция: %s", result, op)
	return nil

}

// обрабатывает запросы типа
// http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
func UpdateMetricsHandler(repo *rep.MemStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var metric model.Metrics
		if err := validateUpdateMetrics(r.URL, &metric); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		}
		
		if metric.MType == model.Counter {
			repo.IncreaseValue(&metric)
			repo.LogState()
			rw.Header().Set("test", "test")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("Counter increased!"))
			return
		}

		if metric.MType == model.Gauge {
			repo.ReplaceValue(&metric)
			repo.LogState()
			rw.Header().Set("test", "test")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("Gauge repalced!"))
			return
		}

		rw.Header().Set("test", "test")
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("Unexpected error! Contact support"))
	}
}