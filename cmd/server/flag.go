package main

import (
	"flag"

	"github.com/caarlos0/env/v11"
)

type startConig struct {
	Addr            string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoregePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDsn     string `env:"DATABASE_DSN"`
	Restore         bool   `env:"RESTORE"`
}

func parseFlags() startConig {
	addr := flag.String("a", "localhost:8080", "addres of the metrics server")
	storeInterval := flag.Int("i", 300, "interval for saving metrics to disk (seconds)")
	fileStoregePath := flag.String("f", "./dump.json", "path to file for storing metrics")
	restore := flag.Bool("r", false, "restore metrics from file on startup")
	databaseDsn := flag.String(
		"d", "user=goclient password=12345678 host=localhost port=5432 database=testgo sslmode=disable",
		"database connection string (key/value format, e.g. user=postgres password=secret host=localhost port=5432 database=pgx_test sslmode=disable",
	)

	flag.Parse()

	conf := startConig{
		Addr:            *addr,
		StoreInterval:   *storeInterval,
		FileStoregePath: *fileStoregePath,
		DatabaseDsn:     *databaseDsn,
		Restore:         *restore,
	}

	env.Parse(&conf)

	return conf
}
