package main

import (
	"flag"

	"github.com/caarlos0/env/v11"
)

type startConig struct {
	Addr            string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoregePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func parseFlags() startConig {
	addr := flag.String("a", "localhost:8080", "addres of the metrics server")
	storeInterval := flag.Int("i", 300, "interval for saving metrics to disk (seconds)")
	fileStoregePath := flag.String("f", "./dump.json", "path to file for storing metrics")
	restore := flag.Bool("r", false, "restore metrics from file on startup")

	flag.Parse()

	conf := startConig{
		Addr:            *addr,
		StoreInterval:   *storeInterval,
		FileStoregePath: *fileStoregePath,
		Restore:         *restore,
	}

	env.Parse(&conf)

	return conf
}
