package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/utils/hasher"
	"go.uber.org/zap"
)

var client = NewRetryClient()

func GetMetrics() map[string]float64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return map[string]float64{
		"Alloc":         float64(ms.Alloc),
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

// SystemMetricsCollector собирает системные метрики через gopsutil.
type SystemMetricsCollector struct{}

// NewSystemMetricsCollector создаёт коллектор системных метрик.
func NewSystemMetricsCollector() *SystemMetricsCollector {
	// Первый вызов cpu.Percent инициализирует внутренний baseline gopsutil.
	cpu.Percent(0, true) //nolint:errcheck
	return &SystemMetricsCollector{}
}

// Collect возвращает актуальные системные метрики.
//
// :returns: карта имя→значение для TotalMemory, FreeMemory и CPUutilizationN.
func (c *SystemMetricsCollector) Collect() map[string]float64 {
	result := make(map[string]float64)

	if vmStat, err := mem.VirtualMemory(); err == nil {
		result["TotalMemory"] = float64(vmStat.Total)
		result["FreeMemory"] = float64(vmStat.Free)
	}

	if percents, err := cpu.Percent(0, true); err == nil {
		for i, p := range percents {
			result[fmt.Sprintf("CPUutilization%d", i+1)] = p
		}
	}

	return result
}

type Agent struct {
	Hasher hasher.Hasher
	Logger zap.Logger
}

func NewAgent(logger zap.Logger, h hasher.Hasher) (Agent, error) {
	return Agent{
		Hasher: h,
		Logger: logger,
	}, nil
}

func (a Agent) SendMetric(serverAddr string, m model.Metrics) error {
	url := fmt.Sprintf("http://%s/update", serverAddr)

	requestBody, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if a.Hasher.Enabled() {
		sign, err := a.Hasher.SignData(requestBody)
		if err != nil {
			return nil
		}
		req.Header.Set("HashSHA256", sign)
	}

	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(requestBody)), nil
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metric %s: %w", m.Name, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got unexpected status code: %v\n Body: %v", res.StatusCode, body)
	}
	return nil
}

// SendMetricsBatch отправляет батч метрик на /updates/ с gzip-сжатием.
func (a Agent) SendMetricsBatch(serverAddr string, metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	url := fmt.Sprintf("http://%s/updates/", serverAddr)

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	compressedData, err := compressGzip(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if a.Hasher.Enabled() {
		sign, err := a.Hasher.SignData(jsonData)
		if err != nil {
			return nil
		}
		req.Header.Set("HashSHA256", sign)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(compressedData)), nil
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got unexpected status code: %v\n Body: %s", res.StatusCode, body)
	}
	return nil
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(data); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
