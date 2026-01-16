package main

import "flag"

type startConf struct {
	addr string
	reportInterval int
	pollInterval int	
}

func parseFlags() startConf {
	addr := flag.String("a", "localhost:8080", "addres of the metrics server (host:port)")
	reportInterval := flag.Int("r", 10, "interval for sending metrics server (second)")
	pollInterval := flag.Int("p", 2, "interval for collecting metrics")

	flag.Parse()

	return startConf{
		addr: *addr,
		reportInterval: *reportInterval,
		pollInterval: *pollInterval,
	}

}