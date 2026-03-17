package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/agent"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()

	if err != nil {
		panic(fmt.Sprintf("failed to inittialize logger: %w", err))
	}

	startConf := parseFlags()
	pollInterval := startConf.PollInterval
	reportInterval := startConf.ReportInterval

	iterationCount := 0

	metricsReport := make([][]model.Metrics, 0, 128)
	for {
		metricsCollector := make([]model.Metrics, 0, 16)

		// получение метрик машины
		for k, v := range agent.GetMetrics() {
			metricsCollector = append(metricsCollector, model.Metrics{
				MType: model.Gauge,
				Name:  k,
				Value: &v,
			})
		}

		// добавление счётчика коллектов
		pollCountValue := int64(1)
		metricsCollector = append(metricsCollector, model.Metrics{
			MType: "counter",
			Name:  "PollCount",
			Delta: &pollCountValue,
		})

		// добавление рандомной переменной
		randValue := rand.Float64()
		metricsCollector = append(metricsCollector, model.Metrics{
			MType: model.Gauge,
			Name:  "RandomValue",
			Value: &randValue,
		})

		// добавление данных в репорт
		metricsReport = append(metricsReport, metricsCollector)
		time.Sleep(time.Duration(pollInterval) * time.Second)

		iterationCount++
		iterationPerRequest := reportInterval / pollInterval

		// отправка репорта на сервер
		if iterationCount >= iterationPerRequest {
			batch := make([]model.Metrics, 0, len(metricsReport)*len(metricsCollector))
			for _, mc := range metricsReport {
				batch = append(batch, mc...)
			}
			if len(batch) > 0 {
				if err := agent.SendMetricsBatch(startConf.Addr, batch); err != nil {
					logger.Warn("batch send failed: %v", zap.Error(err))
				}
			}
			metricsReport = make([][]model.Metrics, 0)
			iterationCount = 0
		}
	}
}
