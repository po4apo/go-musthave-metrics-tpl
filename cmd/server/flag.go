package main

import (
	"flag"

	"github.com/caarlos0/env"
)

type startConig struct {
	Addr string `env:"ADDRESS"`
}

func parseFlags() startConig {
	addr := flag.String("a", "localhost:8080", "addres of the metrics server")

	flag.Parse()

	conf := startConig{
		Addr: *addr,
	}

	env.Parse(&conf)

	return conf
}
