package repository

import (
	"encoding/json"
	"fmt"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	repoerrors "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/errors"
	"go.uber.org/zap"
)

type InMemoryMetricsRepository struct {
	metrics map[string]model.Metrics // в качестве ключа ID
	logger  *zap.Logger
}

func NewMemStorage(logger *zap.Logger) (*InMemoryMetricsRepository, error) {
	// размер не будем устанавливать через конфиг, так как это временное решение
	// в дальнейшем будет полноценная БД
	l := logger.With(zap.String("component", "MemStorage"))
	l.Info(
		"Create mem storage",
	)
	return &InMemoryMetricsRepository{
		metrics: make(map[string]model.Metrics, 128),
		logger:  l,
	}, nil
}

func (s *InMemoryMetricsRepository) SetStateFromSlice(repoState *[]model.Metrics) error {
	for _, m := range *repoState {
		s.metrics[m.ID] = m
	}
	return nil
}

func (s *InMemoryMetricsRepository) IncreaseValue(metric *model.Metrics) error {
	if metric.MType != model.Counter {
		return fmt.Errorf("failed increase %v by %v: %w", metric.Name, metric.Value, repoerrors.ErrUnsupportedType)
	}
	v, exists := s.metrics[metric.ID]
	if !exists {
		s.metrics[metric.ID] = *metric
		return nil
	}

	if metric.Delta == nil {
		return fmt.Errorf("field \"Delta\" is not define: %w", repoerrors.ErrFieldUndefine)
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
	state, err := s.String()
	s.logger.Debug("State", zap.String("state", state), zap.Error(err))

	return nil
}

func (s *InMemoryMetricsRepository) ReplaceValue(metric *model.Metrics) error {
	if metric.MType != model.Gauge {
		return fmt.Errorf("failed replace %v by %v: %w", metric.Name, metric.Value, repoerrors.ErrUnsupportedType)
	}

	v, exists := s.metrics[metric.ID]
	if !exists {
		s.metrics[metric.ID] = *metric
		return nil
	}

	if metric.Value == nil {
		return fmt.Errorf("field \"Value\" is not define: %w", repoerrors.ErrFieldUndefine)
	}

	newValue := *metric.Value
	v.Value = &newValue

	s.metrics[metric.ID] = v

	s.logger.Info(
		"Value replaced",
		zap.String("name", v.Name),
		zap.Float64("newValue", *v.Value),
	)
	state, err := s.String()
	s.logger.Debug("State", zap.String("state", state), zap.Error(err))

	return nil
}

func (s *InMemoryMetricsRepository) GetMetric(id string) (model.Metrics, error) {
	metric, ok := s.metrics[id]
	if !ok {
		return model.Metrics{}, repoerrors.ErrNotFound
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

func (s *InMemoryMetricsRepository) Ping() error {
	return nil
}

func (s *InMemoryMetricsRepository) String() (string, error) {
	b, _ := json.MarshalIndent(s.metrics, "", "  ")
	return string(b), nil
}
