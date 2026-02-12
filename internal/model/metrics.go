package model

import "strings"

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
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
	Name  string   `json:"name"`
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
