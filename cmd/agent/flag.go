package main

import (
	"flag"

	"github.com/caarlos0/env/v11"
)

type startConf struct {
	Addr           string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func parseFlags() startConf {
	var conf startConf
	addr := flag.String("a", "localhost:8080", "addres of the metrics server (host:port)")
	reportInterval := flag.Int("r", 10, "interval for sending metrics server (second)")
	pollInterval := flag.Int("p", 2, "interval for collecting metrics")
	key := flag.String("k", "1234", "HMAC key for signing metrics.")
	rateLimit := flag.Int("l", 1, "max number of concurrent outgoing requests to the server")

	flag.Parse()

	conf = startConf{
		Addr:           *addr,
		ReportInterval: *reportInterval,
		PollInterval:   *pollInterval,
		Key:            *key,
		RateLimit:      *rateLimit,
	}

	env.Parse(&conf)

	return conf
}
