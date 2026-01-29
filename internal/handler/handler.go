package handler

import (
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

var (
	ErrBadRequest = errors.New("badrequest")
	ErrNotFound   = errors.New("notfound")
)

type Handler struct {
	repo   *repository.MetricsRepository
	logger *zap.Logger
}

func NewHandler(repo *repository.MetricsRepository, logger *zap.Logger) *Handler {
	return &Handler{
		repo:   repo,
		logger: logger,
	}
}
func validateUpdateMetrics(
	mType string,
	name string,
	value string,
	result *model.Metrics) error {

	if name == "" {
		return ErrNotFound
	}

	switch mType {
	case model.Counter:
		if delta, err := strconv.ParseInt(value, 10, 64); err == nil {
			*result = model.NewCounterMetrics(name, &delta)
		} else {
			return fmt.Errorf("%w: %w", err, ErrBadRequest)
		}
	case model.Gauge:
		if value, err := strconv.ParseFloat(value, 64); err == nil {
			*result = model.NewGaugeMetric(name, &value)
		} else {
			return fmt.Errorf("%w: %w", err, ErrBadRequest)
		}
	default:
		return fmt.Errorf("\"%v\" is unknown metric type: %w", mType, ErrBadRequest)
	}

	return nil

}

// обрабатывает запросы типа
// http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
func UpdateMetricsHandler(repo repository.MetricsRepository) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var metric model.Metrics

		mType := chi.URLParam(r, "type")
		name := chi.URLParam(r, "name")
		value := chi.URLParam(r, "value")

		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if err := validateUpdateMetrics(mType, name, value, &metric); err != nil {
			log.Printf("%v", err)
			if errors.Is(err, ErrNotFound) {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(err.Error()))
				return
			}
			if errors.Is(err, ErrBadRequest) {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte(err.Error()))
				return
			}
		}

		if metric.MType == model.Counter {
			repo.IncreaseValue(&metric)
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("Counter increased!"))
			return
		}

		if metric.MType == model.Gauge {
			repo.ReplaceValue(&metric)
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("Gauge repalced!"))
			return
		}

		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("Unexpected error! Contact support"))
	}
}

func ViewMetrics(repo repository.MetricsRepository) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metrics, err := repo.GetAll()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		var builder strings.Builder

		// Начало HTML документа
		builder.WriteString("<!DOCTYPE html>\n<html><head><meta charset='utf-8'><title>Metrics</title></head><body>\n")
		builder.WriteString("<h1>Metrics</h1>\n<table border='1' cellpadding='5' cellspacing='0'>\n")
		builder.WriteString("<tr><th>Name</th><th>Type</th><th>Value</th></tr>\n")

		// Вывод метрик
		for _, metric := range metrics {
			builder.WriteString("<tr><td>")
			builder.WriteString(html.EscapeString(metric.Name))
			builder.WriteString("</td><td>")
			builder.WriteString(html.EscapeString(metric.MType))
			builder.WriteString("</td><td>")

			if metric.MType == model.Counter && metric.Delta != nil {
				builder.WriteString(strconv.FormatInt(*metric.Delta, 10))
			} else if metric.MType == model.Gauge && metric.Value != nil {
				builder.WriteString(strconv.FormatFloat(*metric.Value, 'f', -1, 64))
			} else {
				builder.WriteString("N/A")
			}

			builder.WriteString("</td></tr>\n")
		}

		builder.WriteString("</table>\n</body></html>")

		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(builder.String()))
	}
}

func GetMetricHandler(repo repository.MetricsRepository) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "type")
		name := chi.URLParam(r, "name")
		id := model.GenerateID(mType, name)
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")

		metric, err := repo.GetMetric(id)
		log.Print(metric)
		if errors.Is(err, repository.ErrNotFound) {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		var value string
		if metric.MType == model.Counter {
			value = strconv.FormatInt(*metric.Delta, 10)
		} else {
			value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(value))

	}
}
