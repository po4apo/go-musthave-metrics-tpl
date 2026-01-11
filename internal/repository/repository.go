package repository

import (
	model "github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

type Repository interface {
	// вспомогательные
	SetStateFromSlice(*[]model.Metrics)
	LogState()

	// сеттеры
	IncreaseValue(*model.Metrics) error
	ReplaceValue(*model.Metrics) error

	//геттеры
	GetMetric(string) (model.Metrics, error) // по id
	GetAll() ([]model.Metrics, error)
}

var _ Repository = (*MemStorage)(nil)
