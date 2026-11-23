package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"go-poker-sim/internal/sim"
)

func main() {
	players := flag.Int("players", 2, "number of players at the table (2-10)")
	flag.IntVar(players, "p", 2, "shorthand for -players")
	runs := flag.Int("runs", 0, "number of hands to simulate (required, must be > 0)")
	flag.IntVar(runs, "r", 0, "shorthand for -runs")
	workers := flag.Int("workers", runtime.NumCPU(), "number of concurrent worker goroutines")
	chunk := flag.Int("chunk", sim.DefaultChunkSize, "number of runs per output chunk file")
	out := flag.String("out", "results", "output directory for results")
	format := flag.String("format", sim.FormatCSV, "output format: csv or jsonl")
	seed := flag.Uint64("seed", 0, "RNG seed for reproducible runs (0 = random)")
	flag.Parse()

	if *runs <= 0 {
		fmt.Fprintln(os.Stderr, "error: -runs is required and must be > 0")
		flag.Usage()
		os.Exit(2)
	}

	cfg := sim.Config{
		Players:   *players,
		Runs:      *runs,
		Workers:   *workers,
		ChunkSize: *chunk,
		OutDir:    *out,
		Format:    *format,
		Seed:      *seed,
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	log.Printf("starting pre-flop sim: %d runs, %d players, %d workers, format=%s, seed=%d",
		cfg.Runs, cfg.Players, cfg.Workers, cfg.Format, cfg.Seed)
	stats, err := sim.Run(cfg)
	if err != nil {
		log.Fatalf("simulation failed: %v", err)
	}
	log.Printf("done in %s: %d runs (%d player-hands), results written to %s",
		stats.Duration.Round(time.Millisecond), stats.Runs, stats.Hands, stats.OutDir)
}
