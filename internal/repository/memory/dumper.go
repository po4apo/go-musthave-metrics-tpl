package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
)

type Dumper interface {
	Save() error
	Load() error
	Close() error
	RunMapDumper() error
}

type MapDumper struct {
	repo          *InMemoryMetricsRepository
	file          *os.File
	storeInterval time.Duration
}

func NewMapDumper(
	repo *InMemoryMetricsRepository,
	storeInterval int,
	fileStoregePath string,
	restore bool,
) (*MapDumper, error) {

	if storeInterval <= 0 {
		return nil, fmt.Errorf("storeInterval must be positive")
	}

	file, err := os.OpenFile(fileStoregePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	dumper := &MapDumper{
		repo:          repo,
		file:          file,
		storeInterval: time.Duration(storeInterval) * time.Second,
	}

	if restore {
		if err := dumper.Load(); err != nil {
			return nil, err
		}
	}

	return dumper, nil
}

func (d *MapDumper) RunMapDumper(ctx context.Context) error {
	go func() {
		defer d.Close()
		ticker := time.NewTicker(d.storeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				d.Save()
			case <-ctx.Done():
				d.Save()
				return
			}
		}
	}()

	return nil
}

func (d *MapDumper) Save() error {
	if _, err := d.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if err := d.file.Truncate(0); err != nil {
		return err
	}

	metrics, err := d.repo.GetAll()
	if err != nil {
		return err
	}
	rawMetrics, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(d.file)
	defer w.Flush()
	if _, err = w.Write(rawMetrics); err != nil {
		return err
	}

	return nil
}

func (d *MapDumper) Load() error {
	var metrics []model.Metrics

	if _, err := d.file.Seek(0, 0); err != nil {
		return err
	}

	b, err := io.ReadAll(d.file)
	if err != nil {
		return err
	}

	if len(b) == 0 {
		return nil
	}

	if err := json.Unmarshal(b, &metrics); err != nil {
		return err
	}

	if err := d.repo.SetStateFromSlice(&metrics); err != nil {
		return err
	}

	return nil
}

func (d *MapDumper) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}
