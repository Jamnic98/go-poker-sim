// Command api serves the go-poker-sim results directory and simulator over
// HTTP for the web dashboard. It binds to loopback only by default.
//
//	go run ./cmd/api -addr 127.0.0.1:8080 -results results
package main

import (
	"flag"
	"log"
	"net/http"

	"go-poker-sim/internal/api"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address (loopback only by default)")
	results := flag.String("results", "results", "results root directory (same as the simulator -out flag)")
	flag.Parse()

	handler, err := api.NewHandler(*results)
	if err != nil {
		log.Fatalf("initialising API: %v", err)
	}
	log.Printf("go-poker-sim API listening on http://%s (results dir: %s)", *addr, *results)
	log.Fatal(http.ListenAndServe(*addr, handler))
}
