package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/handler"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
)

func main() {
	startConf := parseFlags()

	if err := run(startConf.Addr); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}

func run(addr string) error {
	log.Printf("Запуск сервера по адрессу: %v", addr)

	repo, err := repository.NewMemStorage()
	if err != nil {
		return fmt.Errorf("не удалось инициализировать хранилище: %w", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", handler.ViewMetrics(&repo))
	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetricsHandler(&repo))
	r.Get("/value/{type}/{name}", handler.GetMetricHandler(&repo))

	if err = http.ListenAndServe(addr, r); err != nil {
		return fmt.Errorf("не удалось запустить сервер: %w", err)
	}

	return nil
}
