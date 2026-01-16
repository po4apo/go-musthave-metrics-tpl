package main

import "flag"

type startConig struct {
	addr string	
}

func parseFlags() startConig {
	addr := flag.String("addr", "localhost:8080", "addres of the metrics server")

	flag.Parse()

	return startConig{
		addr: *addr,
	}
}