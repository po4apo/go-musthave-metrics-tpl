package main

import (
	"flag"

	"github.com/caarlos0/env"
)

type startConf struct {
	Addr           string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func parseFlags() startConf {
	var conf startConf
	addr := flag.String("a", "localhost:8080", "addres of the metrics server (host:port)")
	reportInterval := flag.Int("r", 2, "interval for sending metrics server (second)")
	pollInterval := flag.Int("p", 1, "interval for collecting metrics")

	flag.Parse()

	conf = startConf{
		Addr:           *addr,
		ReportInterval: *reportInterval,
		PollInterval:   *pollInterval,
	}

	env.Parse(&conf)

	return conf

}
