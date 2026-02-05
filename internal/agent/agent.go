package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

func GetMetrics() map[string]float64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return map[string]float64{
		"BuckHashSys":   float64(ms.BuckHashSys),
		"Frees":         float64(ms.Frees),
		"GCCPUFraction": float64(ms.GCCPUFraction),
		"GCSys":         float64(ms.GCSys),
		"HeapAlloc":     float64(ms.HeapAlloc),
		"HeapIdle":      float64(ms.HeapIdle),
		"HeapInuse":     float64(ms.HeapInuse),
		"HeapObjects":   float64(ms.HeapObjects),
		"HeapReleased":  float64(ms.HeapReleased),
		"HeapSys":       float64(ms.HeapSys),
		"LastGC":        float64(ms.LastGC),
		"Lookups":       float64(ms.Lookups),
		"MCacheInuse":   float64(ms.MCacheInuse),
		"MCacheSys":     float64(ms.MCacheSys),
		"MSpanInuse":    float64(ms.MSpanInuse),
		"MSpanSys":      float64(ms.MSpanSys),
		"Mallocs":       float64(ms.Mallocs),
		"NextGC":        float64(ms.NextGC),
		"NumForcedGC":   float64(ms.NumForcedGC),
		"NumGC":         float64(ms.NumGC),
		"OtherSys":      float64(ms.OtherSys),
		"PauseTotalNs":  float64(ms.PauseTotalNs),
		"StackInuse":    float64(ms.StackInuse),
		"StackSys":      float64(ms.StackSys),
		"Sys":           float64(ms.Sys),
		"TotalAlloc":    float64(ms.TotalAlloc),
	}
}

func SendMetric(serverAddr string, m model.Metrics) {
	url := fmt.Sprintf("http://%s/update", serverAddr)

	request_body, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	res, err := http.Post(url, "application/json", bytes.NewReader(request_body))
	if err != nil {
		log.Printf("Failed to send metric %s: %v", m.Name, err)
		return
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Got unexpected status code: %v\n Body: %v", res.StatusCode, body)
	}
}
