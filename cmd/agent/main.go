package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/agent"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/utils/hasher"
	"go.uber.org/zap"
)

// metricsStore — потокобезопасное хранилище текущих значений метрик.
type metricsStore struct {
	mu        sync.RWMutex
	gauges    map[string]float64
	pollCount int64
}

func newMetricsStore() *metricsStore {
	return &metricsStore{
		gauges: make(map[string]float64),
	}
}

func (s *metricsStore) updateGauges(m map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range m {
		s.gauges[k] = v
	}
}

func (s *metricsStore) incrementPoll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollCount++
}

// snapshot возвращает копию текущего состояния и сбрасывает pollCount.
func (s *metricsStore) snapshot() []model.Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch := make([]model.Metrics, 0, len(s.gauges)+1)
	for name, val := range s.gauges {
		v := val
		batch = append(batch, model.Metrics{
			MType: model.Gauge,
			Name:  name,
			Value: &v,
		})
	}
	delta := s.pollCount
	batch = append(batch, model.Metrics{
		MType: model.Counter,
		Name:  "PollCount",
		Delta: &delta,
	})
	s.pollCount = 0
	return batch
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to inittialize logger: %v", err))
	}

	conf := parseFlags()

	_hasher, err := hasher.NewHasher(*logger.With(zap.String("component", "Hasher")), conf.Key)
	if err != nil {
		panic(fmt.Sprintf("failed to inittialize hasher: %v", err))
	}
	a, err := agent.NewAgent(*logger.With(zap.String("component", "Agent")), _hasher)
	if err != nil {
		panic(fmt.Sprintf("failed to inittialize agent: %v", err))
	}

	store := newMetricsStore()
	sysCollector := agent.NewSystemMetricsCollector()

	pollTicker := time.NewTicker(time.Duration(conf.PollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(conf.ReportInterval) * time.Second)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	rateLimit := conf.RateLimit
	if rateLimit <= 0 {
		rateLimit = 1
	}

	// jobs — канал с батчами для отправки; буфер равен rateLimit, чтобы
	// репортер не блокировался при полном пуле воркеров.
	jobs := make(chan []model.Metrics, rateLimit)

	// Горутина сбора runtime-метрик.
	go func() {
		for range pollTicker.C {
			m := agent.GetMetrics()
			m["RandomValue"] = rand.Float64()
			store.updateGauges(m)
			store.incrementPoll()
		}
	}()

	// Горутина сбора системных метрик (память + CPU).
	go func() {
		for range pollTicker.C {
			store.updateGauges(sysCollector.Collect())
		}
	}()

	// Горутина-репортер: по reportInterval снимает состояние и кладёт в jobs.
	go func() {
		for range reportTicker.C {
			batch := store.snapshot()
			if len(batch) > 0 {
				jobs <- batch
			}
		}
	}()

	// Worker pool: rateLimit воркеров одновременно отправляют батчи.
	var wg sync.WaitGroup
	for i := 0; i < rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range jobs {
				if err := a.SendMetricsBatch(conf.Addr, batch); err != nil {
					logger.Warn("batch send failed", zap.Error(err))
				}
			}
		}()
	}

	wg.Wait()
}
