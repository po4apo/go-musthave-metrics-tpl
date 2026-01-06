package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/handler"
	rep "github.com/po4apo/go-musthave-metrics-tpl/internal/repository"
)

func main() {
	if err := run(":8080"); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %w", err)
	}
}


 func run(addr string) error {
	log.Printf("Запуск сервера по адрессу: %v", addr)

	repo, err := rep.NewMemStorage()
	if err != nil {
		return fmt.Errorf("не удалось инициализировать хранилище: %w", err)
	}

	http.Handle("/update/", handler.UpdateMetricsHandler(&repo))

	
	if err = http.ListenAndServe(addr, nil); err != nil {
		return fmt.Errorf("не удалось запустить сервер: %w", err)
	}

	return nil
 }