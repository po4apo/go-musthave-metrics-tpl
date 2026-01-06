package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	model "github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

var (
	ErrUnsupportedType = errors.New("the metric type unsupport this action")
	ErrFieldUndefine   = errors.New("required field is not define")
)

type MemStorage struct {
	metrics map[string]model.Metrics // в качестве ключа ID
}

func NewMemStorage() (MemStorage, error) {
	// размер не будем устанавливать через конфиг, так как это временное решение
	// в дальнейшем будет полноценная БД
	return MemStorage{
		metrics: make(map[string]model.Metrics, 128),
	}, nil
}

func (s *MemStorage) IncreaseValue(metric *model.Metrics) error {
	if metric.MType != model.Counter {
		return fmt.Errorf("failed increase %v by %v: %w", metric.Name, metric.Value, ErrUnsupportedType)
	}
	v, exists := s.metrics[metric.Name]
	if !exists {
		s.metrics[metric.Name] = *metric
		return nil
	}

	if metric.Delta == nil {
			return fmt.Errorf("fiels \"Delta\" is not define: %w", ErrFieldUndefine)
		}

	if v.Delta == nil {
		newDelta := *metric.Delta
		v.Delta = &newDelta

	} else {
		newDelta := *v.Delta + *metric.Delta
		v.Delta = &newDelta
	}

	log.Printf("%v успешно увеличена %v", v.Name, *v.Delta)

	s.metrics[metric.Name] = v

	return nil
}

func (s *MemStorage) ReplaceValue(metric *model.Metrics) error {
	if metric.MType != model.Gauge {
		return fmt.Errorf("failed replace %v by %v: %w", metric.Name, metric.Value, ErrUnsupportedType)
	}

	v, exists := s.metrics[metric.Name]
	if !exists {
		s.metrics[metric.Name] = *metric
		return nil
	}

	if metric.Value == nil {
			return fmt.Errorf("fiels \"Value\" is not define: %w", ErrFieldUndefine)
		}

	newValue := *metric.Value
	v.Value = &newValue
	log.Printf("%v успешно заменена на %v", v.Name, *v.Value)

	s.metrics[metric.Name] = v
	return nil
}

func (s *MemStorage) GetAll() (map[string]model.Metrics, error) {
	return s.metrics, nil
}

func (s *MemStorage) LogState() {
    b, _ := json.MarshalIndent(s.metrics, "", "  ")
    log.Printf("Итоговое состояние хранилища:\n%s", string(b))
}
