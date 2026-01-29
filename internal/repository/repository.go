package repository

import (
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

type MetricsRepository interface {
	// вспомогательные
	SetStateFromSlice(*[]model.Metrics)
	String() string

	// сеттеры
	IncreaseValue(*model.Metrics) error
	ReplaceValue(*model.Metrics) error

	//геттеры
	GetMetric(string) (model.Metrics, error) // по id
	GetAll() ([]model.Metrics, error)
}

var _ MetricsRepository = (*InMemoryMetricsRepository)(nil)
