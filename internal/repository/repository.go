package repository

import (
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	memrepo "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/memory"
	pgrepo "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/pg"
)

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

var _ MetricsRepository = (*memrepo.InMemoryMetricsRepository)(nil)
var _ MetricsRepository = (*pgrepo.PostgresRepository)(nil)
