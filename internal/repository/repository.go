package repository

import (
	"errors"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

var ErrNotEmplemented error = errors.New("this method is not implemented")

type MetricsRepository interface {
	// вспомогательные
	SetStateFromSlice(*[]model.Metrics) error
	String() (string, error)

	// сеттеры
	IncreaseValue(*model.Metrics) error
	ReplaceValue(*model.Metrics) error

	//геттеры
	GetMetric(string) (model.Metrics, error) // по id
	GetAll() ([]model.Metrics, error)
	Ping() error
}

var _ MetricsRepository = (*InMemoryMetricsRepository)(nil)
var _ MetricsRepository = (*PostgresRepository)(nil)
