package main

import (
	"math/rand"
	"time"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/agent"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

func main() {
	startConf := parseFlags()
	pollInterval := startConf.PollInterval
	reportInterval := startConf.ReportInterval

	iterationCount := 0

	metricsReport := make([][]model.Metrics, 0, 128)
	for {
		metricsCollector := make([]model.Metrics, 0, 16)

		// получение метрик машины
		for k, v := range agent.GetMetrics() {
			value := v
			metricsCollector = append(metricsCollector, model.Metrics{
				MType: model.Gauge,
				Name:  k,
				Value: &value,
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
			for _, mc := range metricsReport {
				for _, m := range mc {
					agent.SendMetric(startConf.Addr, m)
				}
			}
			metricsReport = make([][]model.Metrics, 0)
			iterationCount = 0
		}
	}
}

