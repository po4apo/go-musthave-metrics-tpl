package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/handler"
	internalMiddleware "github.com/po4apo/go-musthave-metrics-tpl/internal/middleware"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

func main() {
	startConf := parseFlags()

	if err := run(startConf); err != nil {
		panic(fmt.Sprintf("Run server faliled: %v", err))
	}
}

func run(config startConig) error {
	//TODO: заменить на slog
	logger, err := zap.NewDevelopment()

	if err != nil {
		return fmt.Errorf("failed to inittialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Server starting", zap.String("addr", config.Addr))

	repo, err := repository.NewMemStorage(logger)
	if err != nil {
		return fmt.Errorf("failed to inittialize storage: %w", err)
	}

	dumper, err := repository.NewMapDumper(
		&repo,
		config.StoreInterval,
		config.FileStoregePath,
		config.Restore,
	)
	if err != nil {
		return fmt.Errorf("failed to inittialize dumper %w", err)
	}

	_, err = dumper.RunMapDumper()
	if err != nil {
		return fmt.Errorf("failed to run dumper %w", err)
	}

	r := chi.NewRouter()
	r.Use(internalMiddleware.CustomLogger(logger))
	r.Use(internalMiddleware.CompressGzip())
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/", handler.ViewMetrics(&repo))
	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetricsHandler(&repo))
	r.Post("/update", handler.UpdateMetricsWithBodyHandler(&repo))
	r.Post("/update/", handler.UpdateMetricsWithBodyHandler(&repo))
	r.Get("/value/{type}/{name}", handler.GetMetricHandler(&repo))
	r.Post("/value", handler.GetMetricWithBodyHandler(&repo))
	r.Post("/value/", handler.GetMetricWithBodyHandler(&repo))

	if err = http.ListenAndServe(config.Addr, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
