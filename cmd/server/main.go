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

	if err := run(startConf.Addr); err != nil {
		panic(fmt.Sprintf("Run server faliled: %v", err))
	}
}

func run(addr string) error {
	//TODO: заменить на slog
	logger, err := zap.NewDevelopment()

	if err != nil {
		return fmt.Errorf("failed to inittialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Server starting", zap.String("addr", addr))

	repo, err := repository.NewMemStorage(logger)
	if err != nil {
		return fmt.Errorf("failed to inittialize storage: %w", err)
	}

	r := chi.NewRouter()
	r.Use(internalMiddleware.CustomLogger(logger))
	r.Use(internalMiddleware.CompressHTML())
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/", handler.ViewMetrics(&repo))
	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetricsHandler(&repo))
	r.Post("/update", handler.UpdateMetricsWithBodyHandler(&repo))
	r.Post("/update/", handler.UpdateMetricsWithBodyHandler(&repo))
	r.Get("/value/{type}/{name}", handler.GetMetricHandler(&repo))
	r.Post("/value", handler.GetMetricWithBodyHandler(&repo))
	r.Post("/value/", handler.GetMetricWithBodyHandler(&repo))

	if err = http.ListenAndServe(addr, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
