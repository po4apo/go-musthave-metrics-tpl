package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/handler"
	internalMiddleware "github.com/po4apo/go-musthave-metrics-tpl/internal/middleware"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
	memrepo "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/memory"
	pgrepo "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/pg"
	"go.uber.org/zap"
)

func main() {
	startConf := parseFlags()

	if err := run(startConf); err != nil {
		log.Fatalf("Run server faliled: %v", err)
	}
}

func run(config startConig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var repo repository.MetricsRepository
	logger, err := zap.NewDevelopment()

	if err != nil {
		return fmt.Errorf("failed to inittialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Server starting", zap.String("addr", config.Addr))
	if config.DatabaseDsn != "" {
		repo, err = pgrepo.NewPostgresStorage(ctx, logger, config.DatabaseDsn)
		if err != nil {
			logger.Warn("failed to inittialize pg storage", zap.Error(err))
		}
	} else {
		repo, err = memrepo.NewMemStorage(logger)
		if err != nil {
			return fmt.Errorf("failed to inittialize storage: %w", err)
		}
	}

	v, ok := repo.(*memrepo.InMemoryMetricsRepository)
	if ok {
		dumper, err := memrepo.NewMapDumper(
			v,
			config.StoreInterval,
			config.FileStoregePath,
			config.Restore,
		)
		if err != nil {
			return fmt.Errorf("failed to inittialize dumper %w", err)
		}

		if err := dumper.RunMapDumper(ctx); err != nil {
			return fmt.Errorf("failed to run dumper %w", err)
		}
	}

	r := chi.NewRouter()
	r.Use(internalMiddleware.CustomLogger(logger))
	r.Use(internalMiddleware.CompressGzip())
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/", handler.ViewMetrics(repo))
	r.Get("/ping", handler.PingDBHandler(repo))
	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetricsHandler(repo))
	r.Post("/update", handler.UpdateMetricsWithBodyHandler(repo))
	r.Post("/update/", handler.UpdateMetricsWithBodyHandler(repo))
	r.Get("/value/{type}/{name}", handler.GetMetricHandler(repo))
	r.Post("/value", handler.GetMetricWithBodyHandler(repo))
	r.Post("/value/", handler.GetMetricWithBodyHandler(repo))

	if err = http.ListenAndServe(config.Addr, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
