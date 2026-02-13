package model

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"-"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
	Name  string   `json:"name"`
}

// rawMetrics для разбора JSON: клиент может передать "id" или "name" как имя метрики.
// Внутренний ID всегда генерируется на сервере (GenerateID), не принимается от клиента.
type rawMetrics struct {
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
	Name  string   `json:"name"`
	ID    string   `json:"id"`
}

func (m *Metrics) UnmarshalJSON(data []byte) error {
	var raw rawMetrics
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		name = strings.TrimSpace(raw.ID)
	}
	if name == "" {
		return errors.New("field \"name\" or \"id\" is required")
	}
	m.MType = raw.MType
	m.Delta = raw.Delta
	m.Value = raw.Value
	m.Hash = raw.Hash
	m.Name = name
	m.ID = GenerateID(raw.MType, name)
	return nil
}

func GenerateID(mType string, name string) string {
	var builder strings.Builder
	builder.WriteString(mType)
	builder.WriteByte('_')
	builder.WriteString(name)

	return builder.String()
}

func NewCounterMetrics(name string, delta *int64) Metrics {
	mType := Counter
	return Metrics{
		ID:    GenerateID(mType, name),
		Name:  name,
		MType: mType,
		Delta: delta,
	}
}

func NewGaugeMetric(name string, value *float64) Metrics {
	mType := Gauge
	return Metrics{
		ID:    GenerateID(mType, name),
		Name:  name,
		MType: mType,
		Value: value,
	}
}

func Ptr[T any](v T) *T { return &v }
