package repository

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"go.uber.org/zap"
)

var (
	ErrUnsupportedType = errors.New("the metric type unsupport this action")
	ErrFieldUndefine   = errors.New("required field is not define")
	ErrNotFound        = errors.New("notfound")
)

type InMemoryMetricsRepository struct {
	metrics map[string]model.Metrics // в качестве ключа ID
	logger  *zap.Logger
}

func NewMemStorage(logger *zap.Logger) (InMemoryMetricsRepository, error) {
	// размер не будем устанавливать через конфиг, так как это временное решение
	// в дальнейшем будет полноценная БД
	l := logger.With(zap.String("component", "MemStorage"))
	l.Info(
		"Create mem storage",
	)
	return InMemoryMetricsRepository{
		metrics: make(map[string]model.Metrics, 128),
		logger:  l,
	}, nil
}

func (s *InMemoryMetricsRepository) SetStateFromSlice(repoState *[]model.Metrics) {
	for _, m := range *repoState {
		s.metrics[m.ID] = m
	}
}

func (s *InMemoryMetricsRepository) IncreaseValue(metric *model.Metrics) error {
	if metric.MType != model.Counter {
		return fmt.Errorf("failed increase %v by %v: %w", metric.Name, metric.Value, ErrUnsupportedType)
	}
	v, exists := s.metrics[metric.ID]
	if !exists {
		s.metrics[metric.ID] = *metric
		return nil
	}

	if metric.Delta == nil {
		return fmt.Errorf("field \"Delta\" is not define: %w", ErrFieldUndefine)
	}

	if v.Delta == nil {
		newDelta := *metric.Delta
		v.Delta = &newDelta

	} else {
		newDelta := *v.Delta + *metric.Delta
		v.Delta = &newDelta
	}

	s.metrics[metric.ID] = v
	s.logger.Info(
		"Value increased",
		zap.String("name", v.Name),
		zap.Int64("delta", *v.Delta),
	)
	s.logger.Debug("State", zap.String("state", s.String()))

	return nil
}

func (s *InMemoryMetricsRepository) ReplaceValue(metric *model.Metrics) error {
	if metric.MType != model.Gauge {
		return fmt.Errorf("failed replace %v by %v: %w", metric.Name, metric.Value, ErrUnsupportedType)
	}

	v, exists := s.metrics[metric.ID]
	if !exists {
		s.metrics[metric.ID] = *metric
		return nil
	}

	if metric.Value == nil {
		return fmt.Errorf("field \"Value\" is not define: %w", ErrFieldUndefine)
	}

	newValue := *metric.Value
	v.Value = &newValue

	s.metrics[metric.ID] = v

	s.logger.Info(
		"Value replaced",
		zap.String("name", v.Name),
		zap.Float64("newValue", *v.Value),
	)
	s.logger.Debug("State", zap.String("state", s.String()))

	return nil
}

func (s *InMemoryMetricsRepository) GetMetric(id string) (model.Metrics, error) {
	metric, ok := s.metrics[id]
	if !ok {
		return model.Metrics{}, ErrNotFound
	}

	s.logger.Info(
		"Metric got",
		zap.String("id", id),
	)
	return metric, nil
}

func (s *InMemoryMetricsRepository) GetAll() ([]model.Metrics, error) {
	r := make([]model.Metrics, 0, len(s.metrics))
	for _, v := range s.metrics {
		r = append(r, v)
	}
	return r, nil
}

func (s *InMemoryMetricsRepository) String() string {
	b, _ := json.MarshalIndent(s.metrics, "", "  ")
	return string(b)
}
