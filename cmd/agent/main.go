package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/agent"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/utils/hasher"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()

	if err != nil {
		panic(fmt.Sprintf("failed to inittialize logger: %v", err))
	}

	startConf := parseFlags()
	pollInterval := startConf.PollInterval
	reportInterval := startConf.ReportInterval

	iterationCount := 0
	pollCount := int64(0)
	latestGauges := make(map[string]float64)
	_hasher, err := hasher.NewHasher(*logger.With(zap.String("component", "Hasher")), startConf.Key)
	if err != nil {
		panic(fmt.Sprintf("failed to inittialize hasher: %v", err))
	}
	a, err := agent.NewAgent(*logger.With(zap.String("component", "Agent")), _hasher)
	if err != nil {
		panic(fmt.Sprintf("failed to inittialize agent: %v", err))
	}

	for {
		// получение метрик машины
		for k, v := range agent.GetMetrics() {
			latestGauges[k] = v
		}
		latestGauges["RandomValue"] = rand.Float64()
		pollCount++

		time.Sleep(time.Duration(pollInterval) * time.Second)

		iterationCount++
		iterationPerRequest := reportInterval / pollInterval

		// отправка батча на сервер
		if iterationCount >= iterationPerRequest {
			batch := make([]model.Metrics, 0, len(latestGauges)+1)

			for name, val := range latestGauges {
				v := val
				batch = append(batch, model.Metrics{
					MType: model.Gauge,
					Name:  name,
					Value: &v,
				})
			}

			delta := pollCount
			batch = append(batch, model.Metrics{
				MType: model.Counter,
				Name:  "PollCount",
				Delta: &delta,
			})

			if err := a.SendMetricsBatch(startConf.Addr, batch); err != nil {
				logger.Warn("batch send failed", zap.Error(err))
			}

			pollCount = 0
			iterationCount = 0
		}
	}
}
