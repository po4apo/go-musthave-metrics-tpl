package main

import (
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"log"

	models "github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

func main() {
	const pollInterval = 2
	const reportInterval = 10

	iterationCount := 0

	metricsReport := make([][]models.Metrics, 0, 128)
	for {
		metricsCollector := make([]models.Metrics, 0, 16)
		for k, v := range getMetrics() {
			value := v
			metricsCollector = append(metricsCollector, models.Metrics{
				MType: "gauge",
				Name: k,
				Value: &value,
			})
		}

		var PollCountValue int64 = 1
		metricsCollector = append(metricsCollector, models.Metrics{
				MType: "counter",
				Name: "PollCount",
				Delta: &PollCountValue,
			})


		randValue := rand.Float64()
		metricsCollector = append(metricsCollector, models.Metrics{
				MType: "gauge",
				Name: "RandomValue",
				Value: &randValue,
			})
		metricsReport = append(metricsReport, metricsCollector)
		time.Sleep(pollInterval * time.Second)
		
		iterationCount++
		iterationPerRequest := reportInterval / pollInterval
		
		if iterationCount >= iterationPerRequest {
			for _, mc := range metricsReport {
				for _, m := range mc {
					sendMetric(m)
				}
			}
			metricsReport = make([][]models.Metrics, 0)
			iterationCount = 0
		}
	}	
}

func getMetrics() map[string]float64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return map[string]float64{
		"BuckHashSys": float64(ms.BuckHashSys),
		"Frees": float64(ms.Frees),
		"GCCPUFraction": float64(ms.GCCPUFraction),
		"GCSys": float64(ms.GCSys),
		"HeapAlloc": float64(ms.HeapAlloc),
		"HeapIdle": float64(ms.HeapIdle),
		"HeapInuse": float64(ms.HeapInuse),
		"HeapObjects": float64(ms.HeapObjects),
		"HeapReleased": float64(ms.HeapReleased),
		"HeapSys": float64(ms.HeapSys),
		"LastGC": float64(ms.LastGC),
		"Lookups": float64(ms.Lookups),
		"MCacheInuse": float64(ms.MCacheInuse),
		"MCacheSys": float64(ms.MCacheSys),
		"MSpanInuse": float64(ms.MSpanInuse),
		"MSpanSys": float64(ms.MSpanSys),
		"Mallocs": float64(ms.Mallocs),
		"NextGC": float64(ms.NextGC),
		"NumForcedGC": float64(ms.NumForcedGC),
		"NumGC": float64(ms.NumGC),
		"OtherSys": float64(ms.OtherSys),
		"PauseTotalNs": float64(ms.PauseTotalNs),
		"StackInuse": float64(ms.StackInuse),
		"StackSys": float64(ms.StackSys),
		"Sys": float64(ms.Sys),
		"TotalAlloc": float64(ms.TotalAlloc),
	}
}

func sendMetric(m models.Metrics) {
	url := "http://localhost:8080/update"
	var metricValue string

	if m.MType == models.Counter {
		metricValue = strconv.FormatInt(*m.Delta, 10)
	} else {
		metricValue = strconv.FormatFloat(*m.Value, 'f', -1, 64)
	}

	request := url + "/" + m.MType + "/" + m.Name + "/" + metricValue
	res, _ := http.Post(request, "application/json", nil)

	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Got unexpected status code: %v\n Body: %v", res.StatusCode, body)
	}
}
